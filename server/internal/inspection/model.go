package inspection

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

// Position is a Unity world-space point, measured in meters for one AR session.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Position    Position `json:"position"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
type CreateIssue struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Position    Position `json:"position"`
}

var ErrNotFound = errors.New("issue not found")
var ErrConflict = errors.New("idempotency key already used with a different request")

func validPriority(s string) bool { return s == "low" || s == "medium" || s == "high" }
func validStatus(s string) bool   { return s == "open" || s == "in_progress" || s == "resolved" }
func finite(p Position) bool {
	for _, v := range []float64{p.X, p.Y, p.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func validCreate(c CreateIssue) bool {
	return len(validateText(c.Title, c.Description)) == 0 && validPriority(c.Priority) && finite(c.Position)
}

// Text limits count Unicode code points after trimming surrounding whitespace.
func validateText(title, description string) map[string]string {
	fields := map[string]string{}
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		fields["title"] = "Title is required"
	} else if utf8.RuneCountInString(title) > 120 {
		fields["title"] = "Title must be at most 120 characters"
	}
	if utf8.RuneCountInString(description) > 2000 {
		fields["description"] = "Description must be at most 2000 characters"
	}
	return fields
}
