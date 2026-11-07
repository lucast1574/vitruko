package linkpreview

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoreLimitStopsFallbackAndFurtherRequestsToSameHost(t *testing.T) {
	var calls atomic.Int32
	scheduler := &sourceScheduler{hosts: map[string]*hostGate{}}
	client := &http.Client{Transport: &politeTransport{scheduler: scheduler, base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.Path != "/products/cup.js" {
			t.Error("HTML requested after API throttle", r.URL.Path)
		}
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": {"60"}}, Body: io.NopCloser(strings.NewReader("limited"))}, nil
	})}}
	for _, path := range []string{"cup", "other"} {
		_, err := fetchWithClient(context.Background(), "https://shop.example/products/"+path, client)
		var limited *SourceRateLimit
		if !errors.As(err, &limited) || limited.RetryAfter() < 59 {
			t.Fatal("lost upstream deadline", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("throttled source requested again", calls.Load())
	}
}

func TestAjaxProductAPIIsPreferredAndKeepsLocaleAndVariantURL(t *testing.T) {
	var calls int
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/es/products/cup.js" || r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"title":"Cup","featured_image":"/cup.jpg","description":"Ceramic cup"}`))}, nil
	})}
	raw := "https://shop.example/es/products/cup?variant=42"
	data, err := fetchWithClient(context.Background(), raw, client)
	if err != nil || calls != 1 || data.Title != "Cup" || data.URL != raw || data.ImageURL != "https://shop.example/cup.jpg" {
		t.Fatal(data, err, calls)
	}
}

func TestStoreRequestsAreSerializedAndWaitingCanBeCancelled(t *testing.T) {
	var active, peak atomic.Int32
	scheduler := &sourceScheduler{hosts: map[string]*hostGate{}}
	client := &http.Client{Transport: &politeTransport{scheduler: scheduler, base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		n := active.Add(1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := client.Get("https://shop.example/item")
			if err != nil {
				t.Error(err)
				return
			}
			res.Body.Close()
		}()
	}
	wg.Wait()
	if peak.Load() != 1 {
		t.Fatal("parallel requests to the same store", peak.Load())
	}
	release, err := scheduler.acquire(context.Background(), "other.example")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = scheduler.acquire(ctx, "other.example"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRetryAfterDateAndDefaultAreRetained(t *testing.T) {
	future := time.Now().Add(2 * time.Minute).UTC().Truncate(time.Second)
	if got := sourceLimit(future.Format(http.TimeFormat)); !got.Until.Equal(future) {
		t.Fatal(got)
	}
	if got := sourceLimit("invalid").RetryAfter(); got < 59 || got > 61 {
		t.Fatal(got)
	}
}

func TestPartialAjaxProductIsCompletedFromHTML(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, mime := `{"title":"Cup"}`, "application/json"
		if !strings.HasSuffix(r.URL.Path, ".js") {
			body, mime = `<title>Store</title><meta property="og:image" content="/cup.jpg"><meta name="description" content="Ceramic cup">`, "text/html"
		}
		return &http.Response{StatusCode: 200, Request: r, Header: http.Header{"Content-Type": {mime}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	data, err := fetchWithClient(context.Background(), "https://shop.example/products/cup", client)
	if err != nil || data.Title != "Cup" || data.ImageURL != "https://shop.example/cup.jpg" || data.Description != "Ceramic cup" {
		t.Fatal(data, err)
	}
}

func TestAjaxProductUsesSelectedVariantAndPlainDescription(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"title":"Cup","description":"<p>Blue &amp; ceramic</p><script>ignore()</script>","featured_image":"/default.jpg","variants":[{"id":42,"featured_image":{"src":"/blue.jpg"}}]}`))}, nil
	})}
	data, err := fetchWithClient(context.Background(), "https://shop.example/products/cup?variant=42", client)
	if err != nil || data.ImageURL != "https://shop.example/blue.jpg" || data.Description != "Blue & ceramic" {
		t.Fatal(data, err)
	}
}

type idleCloser struct{ closed bool }

func (t *idleCloser) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unused")
}
func (t *idleCloser) CloseIdleConnections() { t.closed = true }
func TestPoliteClientClosesUnderlyingIdleConnections(t *testing.T) {
	base := &idleCloser{}
	client := &http.Client{Transport: &politeTransport{base: base, scheduler: &sourceScheduler{hosts: map[string]*hostGate{}}}}
	client.CloseIdleConnections()
	if !base.closed {
		t.Fatal("underlying transport connections leaked")
	}
}
