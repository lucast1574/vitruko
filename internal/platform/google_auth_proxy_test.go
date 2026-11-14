package platform

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type authHelperTransport func(*http.Request) (*http.Response, error)

func (f authHelperTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleCustomDomainAndHelperProxy(t *testing.T) {
	p := &Platform{config: Config{SiteURL: "https://product.example", FirebaseProjectID: "test-project", FirebaseAPIKey: "public-web-key", FirebaseAppID: "web-app"}}
	h := p.Handler()
	req := httptest.NewRequest("GET", "https://app.product.example/console/auth/providers", nil)
	req.Host = "attacker.invalid"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"authDomain":"product.example"`) {
		t.Fatal(w.Body.String())
	}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	calls := 0
	http.DefaultTransport = authHelperTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Scheme != "https" || r.URL.Host != "test-project.firebaseapp.com" || r.Host != "test-project.firebaseapp.com" {
			t.Fatalf("unexpected upstream: %v %s", r.URL, r.Host)
		}
		if r.URL.Path != "/__/auth/handler" || r.URL.Query().Get("state") != "test-state" {
			t.Fatal("lost callback parameters")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("local credentials leaked")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/html"}, "Content-Security-Policy": {"script-src 'unsafe-inline'"}}, Body: io.NopCloser(strings.NewReader("firebase helper"))}, nil
	})
	req = httptest.NewRequest("GET", "https://product.example/__/auth/handler?state=test-state", nil)
	req.Header.Set("Cookie", "local-session=private")
	req.Header.Set("Authorization", "Bearer private")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "firebase helper" || w.Header().Get("Content-Security-Policy") != "script-src 'unsafe-inline'" {
		t.Fatalf("helper headers or body changed: %d %v", w.Code, w.Header())
	}
	req = httptest.NewRequest("DELETE", "https://product.example/__/auth/handler", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 405 || calls != 1 {
		t.Fatal("unexpected method forwarded")
	}
	w = httptest.NewRecorder()
	newFirebaseAuthProxy("invalid/host").ServeHTTP(w, req)
	if w.Code != 404 || calls != 1 {
		t.Fatal("invalid project forwarded")
	}
}

func TestGoogleAuthDomainFallback(t *testing.T) {
	for _, site := range []string{"", "http://product.example", "https://user:pass@product.example", "https://product.example:8443"} {
		p := &Platform{config: Config{SiteURL: site, FirebaseProjectID: "test-project"}}
		if p.googleAuthDomain() != "test-project.firebaseapp.com" {
			t.Fatal("invalid site became auth domain")
		}
	}
}
