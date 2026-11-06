package platform

import (
	"net/url"
	"strings"
)

// Pictures come from verified Firebase claims, never the login request body.
func googlePicture(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".googleusercontent.com") {
		return ""
	}
	return u.String()
}
