package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicRootAndConsoleFallback(t *testing.T) {
	for _, path := range []string{"/", "/app", "/app/keys", "/login", "/legal"} {
		w := httptest.NewRecorder()
		Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "<div id=\"root\"") {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body)
		}
	}
}
func TestMissingAssetsDoNotReturnHTML(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest("GET", "/missing-asset.jpg", nil))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
