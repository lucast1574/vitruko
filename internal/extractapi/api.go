package extractapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lucast1574/vitruko/internal/linkpreview"
	"golang.org/x/sync/singleflight"
)

type cached struct {
	data    linkpreview.Metadata
	expires time.Time
}
type API struct {
	work  singleflight.Group
	mu    sync.Mutex
	cache map[string]cached
	slots chan struct{}
	fetch func(context.Context, string) (linkpreview.Metadata, error)
}

func New() *API {
	return &API{cache: make(map[string]cached), slots: make(chan struct{}, 8), fetch: linkpreview.Fetch}
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]string{"code": code, "error": message})
}
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"status": "ok", "service": "vitruko"})
	})
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openAPI)
	})
	mux.HandleFunc("POST /v1/extract", a.extract)
	return mux
}
func (a *API) extract(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, 415, "content_type", "Envía un cuerpo application/json.")
		return
	}
	if decoder.Decode(&in) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, "invalid_body", "Envía un único objeto JSON con el campo url.")
		return
	}
	in.URL = strings.TrimSpace(in.URL)
	if !linkpreview.ValidURL(in.URL) {
		fail(w, 400, "invalid_url", "Usa un enlace HTTPS público, sin credenciales ni puertos personalizados.")
		return
	}
	key := linkpreview.NormalizeURL(in.URL)
	a.mu.Lock()
	entry, ok := a.cache[key]
	a.mu.Unlock()
	if ok && time.Now().Before(entry.expires) {
		w.Header().Set("X-Cache", "HIT")
		respond(w, 200, entry.data)
		return
	}
	shared := a.work.DoChan(key, func() (any, error) {
		return a.fetchShared(key)
	})
	var outcome singleflight.Result
	select {
	case outcome = <-shared:
	case <-r.Context().Done():
		fail(w, 408, "request_cancelled", "La petición fue cancelada. Reintenta cuando estés listo.")
		return
	}
	if outcome.Err != nil {
		if errors.Is(outcome.Err, errBusy) {
			w.Header().Set("Retry-After", "2")
			fail(w, 503, "busy", "La extracción está ocupada. Reintenta en unos segundos.")
			return
		}
		var limited *linkpreview.SourceRateLimit
		if errors.As(outcome.Err, &limited) {
			w.Header().Set("Retry-After", strconv.Itoa(limited.RetryAfter()))
			fail(w, 429, "source_rate_limited", limited.Error())
			return
		}
		fail(w, 422, "extraction_unavailable", outcome.Err.Error())
		return
	}
	result := outcome.Val.(fetchResult)
	if result.hit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	respond(w, 200, result.data)
}

var errBusy = errors.New("extractor occupied")

type fetchResult struct {
	data linkpreview.Metadata
	hit  bool
}

func (a *API) fetchShared(key string) (fetchResult, error) {
	a.mu.Lock()
	entry, ok := a.cache[key]
	a.mu.Unlock()
	if ok && time.Now().Before(entry.expires) {
		return fetchResult{data: entry.data, hit: true}, nil
	}
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	default:
		return fetchResult{}, errBusy
	}
	// The work is shared and bounded; one disconnect must not cancel other callers.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	data, err := a.fetch(ctx, key)
	if err != nil {
		return fetchResult{}, err
	}
	data = linkpreview.Complete(data)
	a.mu.Lock()
	if len(a.cache) >= 256 {
		for k, v := range a.cache {
			if time.Now().After(v.expires) {
				delete(a.cache, k)
			}
		}
		if len(a.cache) >= 256 {
			for k := range a.cache {
				delete(a.cache, k)
				break
			}
		}
	}
	a.cache[key] = cached{data: data, expires: time.Now().Add(5 * time.Minute)}
	a.mu.Unlock()
	return fetchResult{data: data}, nil
}
