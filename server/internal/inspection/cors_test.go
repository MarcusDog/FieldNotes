package inspection

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostedDashboardCanUpdateFromActualSameOrigin(t *testing.T) {
	_, h, _ := testServer(t)
	issue := issueResponse(t, request(h, "POST", "/api/issues", validBody, ""), 201)
	for _, base := range []string{"http://192.168.1.20:8080", "https://fieldnotes.local:8443"} {
		t.Run(base, func(t *testing.T) {
			r := httptest.NewRequest("PATCH", base+"/api/issues/"+issue.ID+"/status", strings.NewReader(`{"status":"resolved"}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", base)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			updated := issueResponse(t, w, 200)
			if updated.Status != "resolved" {
				t.Fatal("status not updated")
			}
			if w.Header().Get("Access-Control-Allow-Origin") != base {
				t.Fatalf("unexpected allowed origin: %s", w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
func TestCORSRejectsOriginSchemePortAndForwardedMismatches(t *testing.T) {
	_, h, _ := testServer(t)
	for _, tc := range []struct{ requestURL, origin string }{
		{"http://fieldnotes.local:8080/api/health", "https://fieldnotes.local:8080"},
		{"http://fieldnotes.local:8080/api/health", "http://fieldnotes.local:8081"},
		{"http://fieldnotes.local:8080/api/health", "http://fieldnotes.local"},
		{"https://fieldnotes.local:8443/api/health", "http://fieldnotes.local:8443"},
		{"https://fieldnotes.local:8443/api/health", "https://fieldnotes.local:8444"},
		{"http://fieldnotes.local:8080/api/health", "https://proxy.example"},
	} {
		t.Run(tc.origin+" to "+tc.requestURL, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.requestURL, nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Forwarded-Proto", "https")
			r.Header.Set("X-Forwarded-Host", "proxy.example")
			r.Header.Set("Forwarded", "host=proxy.example;proto=https")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			assertError(t, w, 403)
			if w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("rejected origin was allowed")
			}
		})
	}
}
func TestCORSRejectsMalformedOriginsEvenWithWildcard(t *testing.T) {
	store, _, _ := testServer(t)
	h := NewHandler(store, Options{CORSOrigins: []string{"*"}})
	for _, origin := range []string{"null", "not-an-origin", "ftp://fieldnotes.local", "http://user:pass@fieldnotes.local", "http://fieldnotes.local/", "http://fieldnotes.local/path", "http://fieldnotes.local?", "http://fieldnotes.local#", "http://fieldnotes.local:", "http://fieldnotes.local:bad", "http://fieldnotes.local:65536", "http://::1", "http://[invalid]", "http://fieldnotes.local http://other.local"} {
		t.Run(origin, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://fieldnotes.local/api/health", nil)
			r.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			assertError(t, w, 403)
		})
	}
}
