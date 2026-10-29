package inspection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartupRejectsIncompleteDocumentsWithoutChangingDisk(t *testing.T) {
	cases := []string{
		`null`, `{}`, `{"version":1}`, `{"issues":[],"idempotency":{}}`,
		`{"version":1,"issues":[]}`, `{"version":1,"idempotency":{}}`,
		`{"version":1,"issues":null,"idempotency":{}}`,
		`{"version":1,"issues":[],"idempotency":null}`,
	}
	for _, document := range cases {
		t.Run(document, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "issues.json")
			if err := os.WriteFile(path, []byte(document), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewStore(path); err == nil {
				t.Fatalf("accepted incomplete stored document: %s", document)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, []byte(document)) {
				t.Fatal("failed startup changed data file")
			}
		})
	}
	// An explicitly complete empty document remains a valid initialized database.
	path := filepath.Join(t.TempDir(), "issues.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"issues":[],"idempotency":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(path)
	if err != nil || len(store.List()) != 0 {
		t.Fatalf("complete empty state rejected: %v", err)
	}
}
func TestStartupRejectsCorruptIdempotencyMetadata(t *testing.T) {
	_, h, path := testServer(t)
	issue := issueResponse(t, request(h, "POST", "/api/issues", validBody, requestKey), 201)
	var original diskState
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &original); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*retryRecord)
	}{
		{"missing fingerprint", func(r *retryRecord) { r.Fingerprint = "" }},
		{"invalid fingerprint", func(r *retryRecord) { r.Fingerprint = strings.Repeat("z", 64) }},
		{"unknown issue", func(r *retryRecord) { r.Issue.ID = strings.Repeat("f", 32) }},
		{"title drift", func(r *retryRecord) { r.Issue.Title = "Different title" }},
		{"description drift", func(r *retryRecord) { r.Issue.Description = "Different description" }},
		{"priority drift", func(r *retryRecord) { r.Issue.Priority = "low" }},
		{"position drift", func(r *retryRecord) { r.Issue.Position.X = 999 }},
		{"creation time drift", func(r *retryRecord) {
			parsed, _ := time.Parse(time.RFC3339Nano, r.Issue.CreatedAt)
			r.Issue.CreatedAt = parsed.Add(-time.Hour).Format(time.RFC3339Nano)
			r.Issue.UpdatedAt = r.Issue.CreatedAt
		}},
		{"noninitial status", func(r *retryRecord) { r.Issue.Status = "resolved" }},
		{"noninitial update time", func(r *retryRecord) {
			parsed, _ := time.Parse(time.RFC3339Nano, r.Issue.CreatedAt)
			r.Issue.UpdatedAt = parsed.Add(time.Hour).Format(time.RFC3339Nano)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := original.Idempotency[requestKey]
			tc.mutate(&record)
			state := diskState{Version: 1, Issues: []Issue{issue}, Idempotency: map[string]retryRecord{requestKey: record}}
			data, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewStore(path); err == nil {
				t.Fatal("accepted corrupt idempotency metadata")
			}
		})
	}
}

func TestStartupRejectsMissingStoredPositionCoordinates(t *testing.T) {
	_, h, path := testServer(t)
	issueResponse(t, request(h, "POST", "/api/issues", validBody, requestKey), 201)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, position := range []string{"missing", `null`, `{}`, `{"x":null,"y":0,"z":0}`, `{"y":0,"z":0}`, `{"x":0,"z":0}`, `{"x":0,"y":0}`} {
		for _, target := range []string{"current", "snapshot"} {
			t.Run(target+"/"+position, func(t *testing.T) {
				var state map[string]any
				if err := json.Unmarshal(original, &state); err != nil {
					t.Fatal(err)
				}
				record := state["issues"].([]any)[0].(map[string]any)
				if target == "snapshot" {
					record = state["idempotency"].(map[string]any)[requestKey].(map[string]any)["issue"].(map[string]any)
				}
				if position == "missing" {
					delete(record, "position")
				} else {
					var value any
					if err := json.Unmarshal([]byte(position), &value); err != nil {
						t.Fatal(err)
					}
					record["position"] = value
				}
				encoded, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, encoded, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := NewStore(path); err == nil {
					t.Fatal("accepted missing stored position coordinates")
				}
			})
		}
	}
}
