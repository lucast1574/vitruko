package extractapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lucast1574/vitruko/internal/linkpreview"
)

type cached struct {
	data    linkpreview.Metadata
	expires time.Time
}
type API struct {
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
	select {
	case a.slots <- struct{}{}:
		defer func() { <-a.slots }()
	default:
		w.Header().Set("Retry-After", "2")
		fail(w, 503, "busy", "La extracción está ocupada. Reintenta en unos segundos.")
		return
	}
	data, err := a.fetch(r.Context(), key)
	if err != nil {
		fail(w, 422, "extraction_unavailable", err.Error())
		return
	}
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
	w.Header().Set("X-Cache", "MISS")
	respond(w, 200, data)
}
