package inspection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const validBody = `{"title":"Loose railing","description":"North stairs","priority":"high","position":{"x":1.2,"y":0,"z":-3}}`
const requestKey = "550e8400-e29b-41d4-a716-446655440000"

func testServer(t *testing.T) (*Store, http.Handler, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "issues.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	return s, NewHandler(s, Options{}), path
}
func request(h http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func issueResponse(t *testing.T, w *httptest.ResponseRecorder, status int) Issue {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body.String())
	}
	var data struct {
		Data Issue `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data.Data
}
func assertError(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body.String())
	}
	var data struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || data.Error.Code == "" || data.Error.Message == "" {
		t.Fatalf("invalid error: %s (%v)", w.Body.String(), err)
	}
}
func TestLifecyclePersistsAcrossRestart(t *testing.T) {
	_, h, path := testServer(t)
	w := request(h, "GET", "/api/issues", "", "")
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"data":[]}` {
		t.Fatalf("empty list: %s", w.Body.String())
	}
	first := issueResponse(t, request(h, "POST", "/api/issues", validBody, requestKey), 201)
	if first.ID == "" || first.Status != "open" || first.Title != "Loose railing" || first.Position.X != 1.2 {
		t.Fatalf("bad issue: %+v", first)
	}
	for _, value := range []string{first.CreatedAt, first.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, value); err != nil || !strings.HasSuffix(value, "Z") {
			t.Fatalf("bad UTC time %q", value)
		}
	}
	second := issueResponse(t, request(h, "POST", "/api/issues", strings.Replace(validBody, "Loose railing", "Water leak", 1), ""), 201)
	if second.ID == first.ID {
		t.Fatal("duplicate IDs")
	}
	updated := issueResponse(t, request(h, "PATCH", "/api/issues/"+first.ID+"/status", `{"status":"in_progress"}`, ""), 200)
	if updated.Status != "in_progress" || updated.CreatedAt != first.CreatedAt {
		t.Fatalf("bad update: %+v", updated)
	}
	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	h = NewHandler(s, Options{})
	var list struct {
		Data []Issue `json:"data"`
	}
	w = request(h, "GET", "/api/issues", "", "")
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 2 || list.Data[0].ID != second.ID || list.Data[1].Status != "in_progress" {
		t.Fatalf("list: %+v", list)
	}
	original := issueResponse(t, request(h, "POST", "/api/issues", validBody, requestKey), 201)
	if original != first {
		t.Fatalf("retry changed result: %+v vs %+v", original, first)
	}
}
func TestStrictValidation(t *testing.T) {
	_, h, _ := testServer(t)
	cases := []string{`{}`, `null`, `{"title":"  ","priority":"low","position":{"x":0,"y":0,"z":0}}`, strings.Replace(validBody, `"high"`, `"urgent"`, 1), strings.Replace(validBody, `"x":1.2,`, "", 1), strings.Replace(validBody, `"x":1.2`, `"x":null`, 1), strings.Replace(validBody, `"x":1.2`, `"x":1e999`, 1), strings.Replace(validBody, `"x":1.2`, `"x":"1"`, 1), strings.Replace(validBody, `"title":`, `"unknown":true,"title":`, 1), validBody + ` {}`, `[]`}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) { assertError(t, request(h, "POST", "/api/issues", body, ""), 400) })
	}
	assertError(t, request(h, "POST", "/api/issues", validBody, "not-a-uuid"), 400)
	issue := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	for _, body := range []string{`{}`, `{"status":"done"}`, `{"status":"open","id":"x"}`, `{"status":"open"} null`} {
		assertError(t, request(h, "PATCH", "/api/issues/"+issue.ID+"/status", body, ""), 400)
	}
	assertError(t, request(h, "PATCH", "/api/issues/missing/status", `{"status":"resolved"}`, ""), 404)
}
func TestHTTPProtocolErrorsAndHealth(t *testing.T) {
	_, h, _ := testServer(t)
	w := request(h, "GET", "/api/health", "", "")
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"status":"ok"}` {
		t.Fatal(w.Body.String())
	}
	for _, path := range []string{"/api/health", "/api/issues", "/api/issues/x/status"} {
		w = request(h, "DELETE", path, "", "")
		assertError(t, w, 405)
		if w.Header().Get("Allow") == "" {
			t.Fatal("missing Allow")
		}
	}
	for _, path := range []string{"/api/missing", "/api/issues/x", "/api/issues//status", "/api/issues/x/status/extra"} {
		assertError(t, request(h, "GET", path, "", ""), 404)
	}
	r := httptest.NewRequest("POST", "/api/issues", strings.NewReader(validBody))
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertError(t, w, 415)
	assertError(t, request(h, "POST", "/api/issues", `{"title":"`+strings.Repeat("x", 65536)+`"}`, ""), 413)
}
func TestCORS(t *testing.T) {
	s, h, _ := testServer(t)
	for _, origin := range []string{"http://localhost:5173", "http://evil.example"} {
		r := httptest.NewRequest("OPTIONS", "/api/issues", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Access-Control-Request-Method", "POST")
		r.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if origin == "http://localhost:5173" {
			if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != origin || !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Idempotency-Key") {
				t.Fatalf("bad preflight: %d %v", w.Code, w.Header())
			}
		} else {
			assertError(t, w, 403)
		}
	}
	h = NewHandler(s, Options{CORSOrigins: []string{"*"}})
	r := httptest.NewRequest("GET", "/api/health", nil)
	r.Header.Set("Origin", "https://any.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal(w.Header())
	}
}
func TestAtomicFailuresAndCorruptStartup(t *testing.T) {
	s, h, path := testServer(t)
	first := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	assertError(t, request(h, "POST", "/api/issues", validBody, requestKey), 500)
	assertError(t, request(h, "PATCH", "/api/issues/"+first.ID+"/status", `{"status":"resolved"}`, ""), 500)
	if list := s.List(); len(list) != 1 || list[0].Status != "open" {
		t.Fatalf("failed write changed memory: %+v", list)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	issueResponse(t, request(h, "POST", "/api/issues", validBody, requestKey), 201)
	for _, data := range []string{"", `{broken`, `{"version":1,"issues":[{}],"idempotency":{}}`, `{"version":99,"issues":[],"idempotency":{}}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewStore(path); err == nil {
			t.Fatalf("accepted corrupt disk %q", data)
		}
	}
}
func TestIdempotencyConcurrentRequests(t *testing.T) {
	_, h, _ := testServer(t)
	const count = 24
	ids := make(chan string, count)
	errs := make(chan string, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(h, "POST", "/api/issues", validBody, requestKey)
			if w.Code != 201 {
				errs <- w.Body.String()
				return
			}
			var data struct {
				Data Issue `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				errs <- err.Error()
				return
			}
			ids <- data.Data.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatal("duplicate retry created another issue")
		}
	}
	assertError(t, request(h, "POST", "/api/issues", strings.Replace(validBody, "railing", "door", 1), requestKey), 409)
	w := request(h, "GET", "/api/issues", "", "")
	var data struct {
		Data []Issue `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &data)
	if len(data.Data) != 1 {
		t.Fatalf("created %d issues", len(data.Data))
	}
}
func TestConcurrentDistinctCreates(t *testing.T) {
	s, h, path := testServer(t)
	var wg sync.WaitGroup
	fail := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := strings.Replace(validBody, "Loose railing", fmt.Sprintf("Issue %d", i), 1)
			w := request(h, "POST", "/api/issues", body, "")
			if w.Code != 201 {
				fail <- w.Body.String()
			}
			request(h, "GET", "/api/issues", "", "")
		}(i)
	}
	wg.Wait()
	close(fail)
	for msg := range fail {
		t.Error(msg)
	}
	if len(s.List()) != 20 {
		t.Fatal("lost in-memory writes")
	}
	reopened, err := NewStore(path)
	if err != nil || len(reopened.List()) != 20 {
		t.Fatalf("lost persisted writes: %v", err)
	}
}
func TestStaticSPAFallback(t *testing.T) {
	s, _, _ := testServer(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>FieldNotes</main>"), 0600)
	os.WriteFile(filepath.Join(dir, "asset.js"), []byte("console.log('ok')"), 0600)
	h := NewHandler(s, Options{WebDir: dir})
	w := request(h, "GET", "/issues/active", "", "")
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("FieldNotes")) {
		t.Fatalf("SPA: %d %s", w.Code, w.Body.String())
	}
	w = request(h, "GET", "/asset.js", "", "")
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("console.log")) {
		t.Fatal("asset not served")
	}
	assertError(t, request(h, "GET", "/api/missing", "", ""), 404)
}
