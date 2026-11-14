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
		r := httptest.NewRequest("GET", "https://vitruko.lat"+path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		body := w.Body.String()
		if w.Code != 200 || strings.Count(body, "<h1") != 1 || !strings.Contains(body, `data-prerendered="true"`) || !strings.Contains(body, `rel="canonical" href="https://vitruko.lat`+path+`"`) {
			t.Fatalf("invalid public HTML %s %d", path, w.Code)
		}
		schema := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(body)
		if len(schema) != 2 || !json.Valid([]byte(schema[1])) {
			t.Fatal("invalid structured data", path)
		}
		var structured struct {
			Context string           `json:"@context"`
			Graph   []map[string]any `json:"@graph"`
		}
		if err := json.Unmarshal([]byte(schema[1]), &structured); err != nil {
			t.Fatal(err)
		}
		if structured.Context != "https://schema.org" {
			t.Fatal("invalid schema vocabulary", path)
		}
		for _, node := range structured.Graph {
			switch node["@type"] {
			case "Organization":
				logo := node["logo"].(map[string]any)
				if logo["url"] != "https://vitruko.lat/icon-512.png?v=tag-2" || logo["contentUrl"] != logo["url"] || logo["width"] != float64(512) || logo["height"] != float64(512) {
					t.Fatal("organization must reference the current crawlable logo", path)
				}
			case "WebSite":
				if node["name"] != "Vitruko" || node["url"] != "https://vitruko.lat/" {
					t.Fatal("inconsistent site identity", path)
				}
			case "BreadcrumbList":
				items := node["itemListElement"].([]any)
				for i, item := range items {
					entry := item.(map[string]any)
					if entry["position"] != float64(i+1) || entry["name"] == "" || !strings.HasPrefix(entry["item"].(string), "https://vitruko.lat/") {
						t.Fatal("invalid breadcrumb", path)
					}
				}
				if items[len(items)-1].(map[string]any)["item"] != "https://vitruko.lat"+path {
					t.Fatal("breadcrumb must end at the canonical page", path)
				}
			}
		}
		if strings.Contains(body, "OpenStreetMap") || strings.Contains(body, "Mapora") {
			t.Fatal("unrelated product content", path)
		}
	}
	for _, path := range []string{"/login", "/signup", "/app", "/app/inspector"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "https://app.vitruko.lat"+path, nil))
		if w.Code != 200 || !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") || strings.Contains(w.Body.String(), `rel="canonical"`) {
			t.Fatal("private route indexable", path)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "https://vitruko.lat/unknown", nil))
	if w.Code != 404 {
		t.Fatal("soft 404")
	}
}
