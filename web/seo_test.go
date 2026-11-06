package web

import (
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPublicPagesArePrerenderedAndPrivatePagesNoindex(t *testing.T) {
	h := Handler()
	for _, path := range []string{"/", "/documentacion", "/api-extraer-productos", "/vista-previa-enlaces", "/shopify-vtex", "/legal"} {
		r := httptest.NewRequest("GET", "https://vitruko.nexode.app"+path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		body := w.Body.String()
		if w.Code != 200 || strings.Count(body, "<h1") != 1 || !strings.Contains(body, `data-prerendered="true"`) || !strings.Contains(body, `rel="canonical" href="https://vitruko.nexode.app`+path+`"`) {
			t.Fatalf("invalid public HTML %s %d", path, w.Code)
		}
		schema := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(body)
		if len(schema) != 2 || !json.Valid([]byte(schema[1])) {
			t.Fatal("invalid structured data", path)
		}
		if strings.Contains(body, "OpenStreetMap") || strings.Contains(body, "Mapora") {
			t.Fatal("unrelated product content", path)
		}
	}
	for _, path := range []string{"/login", "/signup", "/app", "/app/inspector"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "https://app-vitruko.nexode.app"+path, nil))
		if w.Code != 200 || !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") || strings.Contains(w.Body.String(), `rel="canonical"`) {
			t.Fatal("private route indexable", path)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "https://vitruko.nexode.app/unknown", nil))
	if w.Code != 404 {
		t.Fatal("soft 404")
	}
}
