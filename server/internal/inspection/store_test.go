package inspection

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

func TestCorruptTimestampOrderingFailsStartup(t *testing.T) {
	_, h, path := testServer(t)
	issue := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	issue.CreatedAt = "2026-10-29T12:00:00.12Z"
	issue.UpdatedAt = "2026-10-29T12:00:00.1Z"
	encoded, err := json.Marshal(diskState{Version: 1, Issues: []Issue{issue}, Idempotency: map[string]retryRecord{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path); err == nil {
		t.Fatal("accepted updatedAt before createdAt with different fractional precision")
	}
}
func TestStatusUpdateSurvivesClockMovingBackwards(t *testing.T) {
	_, h, path := testServer(t)
	issue := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339Nano)
	issue.CreatedAt = future
	issue.UpdatedAt = future
	encoded, err := json.Marshal(diskState{Version: 1, Issues: []Issue{issue}, Idempotency: map[string]retryRecord{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateStatus(issue.ID, "resolved")
	if err != nil {
		t.Fatal(err)
	}
	created, _ := time.Parse(time.RFC3339Nano, updated.CreatedAt)
	last, _ := time.Parse(time.RFC3339Nano, updated.UpdatedAt)
	if last.Before(created) {
		t.Fatal("clock adjustment made update older than creation")
	}
	if _, err := NewStore(path); err != nil {
		t.Fatalf("clock adjustment corrupted persisted state: %v", err)
	}
}
func TestStoreRejectsNonFiniteCoordinates(t *testing.T) {
	store, _, _ := testServer(t)
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := store.Create(CreateIssue{Title: "Hazard", Priority: "low", Position: Position{X: value}}, ""); err == nil {
			t.Fatal("accepted nonfinite coordinate")
		}
	}
	if len(store.List()) != 0 {
		t.Fatal("invalid coordinates changed store")
	}
}
