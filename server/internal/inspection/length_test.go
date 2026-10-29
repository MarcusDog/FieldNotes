package inspection

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCreateUnicodeLengthLimits(t *testing.T) {
	_, h, _ := testServer(t)
	for _, tc := range []struct{ name, title, description, field string }{
		{"title", strings.Repeat("检", 121), "", "title"},
		{"description", "Issue", strings.Repeat("😀", 2001), "description"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(CreateIssue{Title: tc.title, Description: tc.description, Priority: "low", Position: Position{}})
			w := request(h, "POST", "/api/issues", string(body), "")
			assertError(t, w, 400)
			var response struct {
				Error struct {
					Code   string            `json:"code"`
					Fields map[string]string `json:"fields"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != "validation_error" || response.Error.Fields[tc.field] == "" {
				t.Fatalf("missing field validation: %s", w.Body.String())
			}
		})
	}
	title := strings.Repeat("😀", 120)
	description := strings.Repeat("检", 2000)
	body, _ := json.Marshal(CreateIssue{Title: " \n" + title + "\t ", Description: " \t" + description + "\n ", Priority: "medium", Position: Position{}})
	issue := issueResponse(t, request(h, "POST", "/api/issues", string(body), ""), 201)
	if issue.Title != title || issue.Description != description {
		t.Fatal("boundary values not trimmed and preserved")
	}
}
func TestStoreLengthValidationAndTrimming(t *testing.T) {
	store, _, _ := testServer(t)
	for _, input := range []CreateIssue{
		{Title: strings.Repeat("😀", 121), Priority: "low"},
		{Title: "Issue", Description: strings.Repeat("检", 2001), Priority: "low"},
	} {
		if _, err := store.Create(input, ""); err == nil {
			t.Fatal("store accepted overlong issue")
		}
	}
	issue, err := store.Create(CreateIssue{Title: " \t Issue \n ", Description: " \n Description \t ", Priority: "low"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if issue.Title != "Issue" || issue.Description != "Description" {
		t.Fatalf("store failed to trim: %+v", issue)
	}
}
func TestStartupRejectsOverlongStoredText(t *testing.T) {
	_, h, path := testServer(t)
	original := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	for _, field := range []string{"title", "description"} {
		t.Run(field, func(t *testing.T) {
			issue := original
			if field == "title" {
				issue.Title = strings.Repeat("检", 121)
			} else {
				issue.Description = strings.Repeat("😀", 2001)
			}
			encoded, err := json.Marshal(diskState{Version: 1, Issues: []Issue{issue}, Idempotency: map[string]retryRecord{}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewStore(path); err == nil {
				t.Fatalf("startup accepted overlong %s", field)
			}
		})
	}
}
