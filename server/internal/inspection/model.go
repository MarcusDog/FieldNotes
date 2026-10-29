package inspection

import (
	"errors"
	"math"
	"strings"
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
	return strings.TrimSpace(c.Title) != "" && validPriority(c.Priority) && finite(c.Position)
}
