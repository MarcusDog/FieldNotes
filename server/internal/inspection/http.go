package inspection

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxBodyBytes = 64 << 10

type Options struct {
	CORSOrigins []string
	WebDir      string
}
type handler struct {
	store   *Store
	options Options
}

func NewHandler(store *Store, options Options) http.Handler {
	if options.CORSOrigins == nil {
		options.CORSOrigins = []string{"http://localhost:5173"}
	}
	return &handler{store, options}
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	writeJSON(w, status, struct {
		Error apiError `json:"error"`
	}{apiError{code, message, fields}})
}
func data(w http.ResponseWriter, status int, value any) {
	writeJSON(w, status, struct {
		Data any `json:"data"`
	}{value})
}
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
		if !h.cors(w, r) {
			return
		}
		h.api(w, r)
		return
	}
	if h.options.WebDir != "" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		// Only serve regular files within the configured build directory.
		clean := filepath.Clean("/" + r.URL.Path)
		name := filepath.Join(h.options.WebDir, clean)
		info, err := os.Stat(name)
		if err != nil || !info.Mode().IsRegular() {
			name = filepath.Join(h.options.WebDir, "index.html")
		}
		http.ServeFile(w, r, name)
		return
	}
	writeError(w, 404, "not_found", "Resource not found", nil)
}
func (h *handler) cors(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	w.Header().Add("Vary", "Origin")
	allowed := ""
	for _, candidate := range h.options.CORSOrigins {
		if candidate == "*" {
			allowed = "*"
			break
		}
		if candidate == origin {
			allowed = origin
			break
		}
	}
	if allowed == "" {
		writeError(w, 403, "origin_not_allowed", "Origin is not allowed", nil)
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", allowed)
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		w.Header().Add("Vary", "Access-Control-Request-Headers")
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	return true
}
func methods(w http.ResponseWriter, r *http.Request, allowed string) bool {
	for _, m := range strings.Split(allowed, ", ") {
		if m == r.Method {
			return true
		}
	}
	w.Header().Set("Allow", allowed)
	writeError(w, 405, "method_not_allowed", "Method not allowed", nil)
	return false
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		writeError(w, 415, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err = dec.Decode(value)
	if err == nil {
		err = dec.Decode(new(any))
		if err == io.EOF {
			return true
		}
		if err == nil {
			err = errors.New("multiple JSON values")
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, 413, "request_too_large", "Request body exceeds 64 KiB", nil)
	} else {
		writeError(w, 400, "invalid_json", "Body must contain one valid JSON object with supported fields", nil)
	}
	return false
}
func (h *handler) api(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/health":
		if methods(w, r, "GET") {
			writeJSON(w, 200, map[string]string{"status": "ok"})
		}
	case r.URL.Path == "/api/issues":
		if !methods(w, r, "GET, POST") {
			return
		}
		if r.Method == http.MethodGet {
			data(w, 200, h.store.List())
			return
		}
		h.create(w, r)
	default:
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/issues/"), "/")
		if strings.HasPrefix(r.URL.Path, "/api/issues/") && len(parts) == 2 && parts[0] != "" && parts[1] == "status" {
			if methods(w, r, "PATCH") {
				h.update(w, r, parts[0])
			}
			return
		}
		writeError(w, 404, "not_found", "Resource not found", nil)
	}
}
func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		Position    *struct {
			X *float64 `json:"x"`
			Y *float64 `json:"y"`
			Z *float64 `json:"z"`
		} `json:"position"`
	}
	if !decode(w, r, &input) {
		return
	}
	fields := map[string]string{}
	if strings.TrimSpace(input.Title) == "" {
		fields["title"] = "Title is required"
	}
	if !validPriority(input.Priority) {
		fields["priority"] = "Must be low, medium, or high"
	}
	var position Position
	if input.Position == nil || input.Position.X == nil || input.Position.Y == nil || input.Position.Z == nil {
		fields["position"] = "Finite x, y, and z coordinates are required"
	} else {
		position = Position{*input.Position.X, *input.Position.Y, *input.Position.Z}
		if !finite(position) {
			fields["position"] = "Coordinates must be finite"
		}
	}
	key := r.Header.Get("Idempotency-Key")
	if key != "" && !validKey(key) {
		fields["Idempotency-Key"] = "Must be a UUID"
	}
	if len(fields) > 0 {
		writeError(w, 400, "validation_error", "Please correct the invalid fields", fields)
		return
	}
	issue, err := h.store.Create(CreateIssue{input.Title, input.Description, input.Priority, position}, key)
	if errors.Is(err, ErrConflict) {
		writeError(w, 409, "idempotency_conflict", ErrConflict.Error(), nil)
		return
	}
	if err != nil {
		writeError(w, 500, "persistence_error", "Could not save issue; please retry", nil)
		return
	}
	data(w, 201, issue)
}
func (h *handler) update(w http.ResponseWriter, r *http.Request, id string) {
	var input struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !validStatus(input.Status) {
		writeError(w, 400, "validation_error", "Please correct the invalid fields", map[string]string{"status": "Must be open, in_progress, or resolved"})
		return
	}
	issue, err := h.store.UpdateStatus(id, input.Status)
	if errors.Is(err, ErrNotFound) {
		writeError(w, 404, "not_found", "Issue not found", nil)
		return
	}
	if err != nil {
		writeError(w, 500, "persistence_error", "Could not save status; please retry", nil)
		return
	}
	data(w, 200, issue)
}
