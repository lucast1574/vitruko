package extractapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lucast1574/vitruko/internal/linkpreview"
)

func TestConcurrentRequestsShareFetchEvenIfFirstCallerDisconnects(t *testing.T) {
	a := New()
	started, finish := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	a.fetch = func(ctx context.Context, raw string) (linkpreview.Metadata, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-finish:
			return linkpreview.Metadata{Title: "Cup", URL: raw}, nil
		case <-ctx.Done():
			return linkpreview.Metadata{}, ctx.Err()
		}
	}
	request := func(ctx context.Context) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/extract", strings.NewReader(`{"url":"https://shop.example/products/cup"}`)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- request(ctx) }()
	<-started
	second := make(chan *httptest.ResponseRecorder, 1)
	go func() { second <- request(context.Background()) }()
	cancel()
	if w := <-first; w.Code != 408 {
		t.Fatal(w.Code)
	}
	close(finish)
	if w := <-second; w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate upstream fetch", calls.Load())
	}
	if w := request(context.Background()); w.Code != 200 || w.Header().Get("X-Cache") != "HIT" {
		t.Fatal(w.Code, w.Body)
	}
}

func TestStoreThrottleIsDistinctFromAccountQuota(t *testing.T) {
	a := New()
	a.fetch = func(context.Context, string) (linkpreview.Metadata, error) {
		return linkpreview.Metadata{}, &linkpreview.SourceRateLimit{Until: time.Now().Add(time.Minute)}
	}
	r := httptest.NewRequest("POST", "/v1/extract", strings.NewReader(`{"url":"https://shop.example/product"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 429 || body["code"] != "source_rate_limited" || w.Header().Get("Retry-After") == "" {
		t.Fatal(w.Code, w.Body, w.Header())
	}
}
