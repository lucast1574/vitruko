package platform

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// The public site domain is the stable name shown by Google's account chooser.
// Never derive it from an incoming Host header.
func (p *Platform) googleAuthDomain() string {
	u, err := url.Parse(p.config.SiteURL)
	if err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Port() == "" {
		return u.Hostname()
	}
	return p.config.FirebaseProjectID + ".firebaseapp.com"
}

func newFirebaseAuthProxy(project string) http.Handler {
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`).MatchString(project) {
		return http.NotFoundHandler()
	}
	target := &url.URL{Scheme: "https", Host: project + ".firebaseapp.com"}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = target.Host
			// Local app sessions and API credentials never belong to the auth helper host.
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("Authorization")
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Set("X-Robots-Tag", "noindex, nofollow, noarchive")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			http.Error(w, "Authentication helper unavailable", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/__/auth/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, POST, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		proxy.ServeHTTP(w, r.WithContext(ctx))
	})
}
