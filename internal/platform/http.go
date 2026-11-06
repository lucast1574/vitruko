package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type httpHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

func defaultHTTP() *http.Client { return &http.Client{Timeout: 12 * time.Second} }
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, msg string) {
	reply(w, status, map[string]string{"code": code, "error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (p *Platform) session(r *http.Request) (User, error) {
	c, err := r.Cookie("__Host-vitruko_session")
	if err != nil {
		return User{}, err
	}
	return scanUser(p.db.QueryRowContext(r.Context(), "SELECT u."+strings.ReplaceAll(userColumns, ",", ",u.")+" FROM users u JOIN sessions s ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND NOT u.disabled", digest(c.Value)))
}

type userContext struct{}

func current(r *http.Request) User { return r.Context().Value(userContext{}).(User) }
func (p *Platform) private(fn http.HandlerFunc, admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := p.session(r)
		if err != nil {
			problem(w, 401, "unauthorized", "Inicia sesión para continuar.")
			return
		}
		if admin && u.Role != "admin" {
			problem(w, 403, "forbidden", "Esta acción requiere una cuenta administradora.")
			return
		}
		fn(w, r.WithContext(context.WithValue(r.Context(), userContext{}, u)))
	}
}
func (p *Platform) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /console/signup", p.signup)
	mux.HandleFunc("POST /console/login", p.login)
	mux.HandleFunc("GET /console/auth/providers", p.providers)
	mux.HandleFunc("POST /console/google", p.googleLogin)
	mux.HandleFunc("POST /console/forgot", p.forgot)
	mux.HandleFunc("POST /console/reset", p.reset)
	mux.HandleFunc("GET /console/plans", p.plans)
	routes := map[string]http.HandlerFunc{"GET /console/session": p.me, "POST /console/logout": p.logout, "PATCH /console/me": p.profile, "POST /console/password": p.password, "GET /console/keys": p.listKeys, "POST /console/keys": p.createKey, "DELETE /console/keys/{id}": p.revokeKey, "GET /console/usage": p.usage, "POST /console/billing/checkout": p.checkout, "POST /console/billing/portal": p.portal, "POST /console/billing/sync": p.syncBilling}
	for route, fn := range routes {
		mux.HandleFunc(route, p.private(fn, false))
	}
	for route, fn := range map[string]http.HandlerFunc{"GET /console/admin/users": p.adminUsers, "PATCH /console/admin/users/{id}": p.adminUser, "GET /console/admin/overview": p.adminOverview, "PATCH /console/admin/plans/{id}": p.adminPlan} {
		mux.HandleFunc(route, p.private(fn, true))
	}
	mux.HandleFunc("POST /stripe/webhook", p.webhook)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/console/") || strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/map/") || r.URL.Path == "/stripe/webhook" || r.URL.Path == "/health" || r.URL.Path == "/openapi.json" || apiEndpoint(r.URL.Path) != "" {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://*.googleusercontent.com; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self' https://checkout.stripe.com https://billing.stripe.com")
		if r.URL.Path == "/documentacion" || r.URL.Path == "/app/inspector" || r.URL.Path == "/app/docs" {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		}
		if r.URL.Path == "/login" || r.URL.Path == "/signup" {
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://apis.google.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://*.googleusercontent.com; font-src 'self'; connect-src 'self' https://identitytoolkit.googleapis.com https://securetoken.googleapis.com https://www.googleapis.com; frame-src https://"+p.config.FirebaseProjectID+".firebaseapp.com https://accounts.google.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		}
		if strings.HasPrefix(r.URL.Path, "/console/") {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host {
					problem(w, 403, "origin", "Origen no permitido.")
					return
				}
			}
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			mux.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if r.URL.Path == "/stripe/webhook" {
			mux.ServeHTTP(w, r)
			return
		}
		if apiEndpoint(r.URL.Path) != "" {
			p.protectAPI(w, r)
			return
		}
		p.geo.ServeHTTP(w, r)
	})
}
func newSession(w http.ResponseWriter, raw string) {
	http.SetCookie(w, &http.Cookie{Name: "__Host-vitruko_session", Value: raw, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 86400})
}
func validPassword(password string) bool { return len(password) >= 12 && len(password) <= 72 }

var errInvalid = errors.New("invalid input")
