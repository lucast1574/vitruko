package extractapi

import (
	"context"
	"encoding/json"
	"github.com/lucast1574/vitruko/internal/linkpreview"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidationNeverFetchesAndCachePreservesVariant(t *testing.T) {
	a := New()
	calls := 0
	a.fetch = func(ctx context.Context, u string) (linkpreview.Metadata, error) {
		calls++
		return linkpreview.Metadata{Title: "Taza", URL: u, ImageURL: "https://cdn.example/cup.jpg"}, nil
	}
	request := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/extract", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	for _, body := range []string{`{"url":"https://127.0.0.1/"}`, `{"url":"http://shop.example/p"}`, `{"url":"https://shop.example/","cookie":"private"}`, `{"url":"https://shop.example/"} {}`} {
		if w := request(body); w.Code != 400 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if calls != 0 {
		t.Fatal("invalid requests fetched")
	}
	one := request(`{"url":"https://shop.example/products/cup?variant=1&utm_source=x"}`)
	two := request(`{"url":"https://shop.example/products/cup?variant=1"}`)
	three := request(`{"url":"https://shop.example/products/cup?variant=2"}`)
	if one.Code != 200 || two.Header().Get("X-Cache") != "HIT" || three.Header().Get("X-Cache") != "MISS" || calls != 2 {
		t.Fatal(one, two, three, calls)
	}
}

func TestRichResponsePreservesLegacyFieldsAndCacheObservation(t *testing.T) {
	a := New()
	a.fetch = func(_ context.Context, u string) (linkpreview.Metadata, error) {
		return linkpreview.Metadata{Title: "Cup", URL: u, ImageURL: "https://cdn.example/cup.jpg", Description: "Ceramic", Price: &linkpreview.Price{Amount: "19.90", Currency: "PEN"}}, nil
	}
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/extract", strings.NewReader(`{"url":"https://shop.example/cup"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	first, second := request(), request()
	if first.Code != 200 || first.Body.String() != second.Body.String() || second.Header().Get("X-Cache") != "HIT" {
		t.Fatal(first, second)
	}
	var out map[string]any
	if json.Unmarshal(first.Body.Bytes(), &out) != nil {
		t.Fatal(first.Body)
	}
	for _, field := range []string{"title", "description", "imageUrl", "url", "images", "brand", "price", "availability", "quality"} {
		if _, ok := out[field]; !ok {
			t.Fatal("missing field", field)
		}
	}
	if out["price"].(map[string]any)["amount"] != "19.90" || out["availability"] != "unknown" || out["quality"].(map[string]any)["extractedAt"] == "" {
		t.Fatal(out)
	}
}
