package extractapi

import (
	"context"
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
