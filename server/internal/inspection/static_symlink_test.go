package inspection

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticBuildSymlinksCannotEscapeWebRoot(t *testing.T) {
	store, _, _ := testServer(t)
	root := t.TempDir()
	webDir := filepath.Join(root, "web")
	outsideDir := filepath.Join(root, "private")
	if err := os.MkdirAll(webDir, 0700); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(outsideDir, 0700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("SAFE SPA"), 0600); err != nil { t.Fatal(err) }
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("DO_NOT_EXPOSE"), 0600); err != nil { t.Fatal(err) }
	if err := os.Symlink(secret, filepath.Join(webDir, "leak.txt")); err != nil { t.Skipf("symlinks unavailable: %v", err) }
	if err := os.Symlink(outsideDir, filepath.Join(webDir, "escape")); err != nil { t.Skipf("symlinks unavailable: %v", err) }

	h := NewHandler(store, Options{WebDir: webDir})
	for _, route := range []string{"/leak.txt", "/escape/secret.txt"} {
		t.Run(route, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, route, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("SAFE SPA")) ||
				bytes.Contains(w.Body.Bytes(), []byte("DO_NOT_EXPOSE")) {
				t.Fatalf("unsafe static response: %d %q", w.Code, w.Body.String())
			}
		})
	}
}
