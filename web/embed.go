package web

import (
	"embed"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:dist
var assets embed.FS

func Handler() http.Handler {
	root, _ := fs.Sub(assets, "dist")
	files := http.FileServer(http.FS(root))
	var publicPaths []string
	manifest, err := fs.ReadFile(root, "routes.json")
	if err != nil || json.Unmarshal(manifest, &publicPaths) != nil {
		panic("missing prerendered web routes: run the frontend production build")
	}
	pages := map[string]string{}
	for _, route := range publicPaths {
		file := strings.TrimPrefix(route, "/") + "/index.html"
		if route == "/" {
			file = "index.html"
		}
		pages[route] = file
	}
	verification := os.Getenv("GOOGLE_SITE_VERIFICATION")
	serveHTML := func(w http.ResponseWriter, r *http.Request, file string, status int) {
		data, err := fs.ReadFile(root, file)
		if err != nil {
			http.Error(w, "Web content unavailable", 503)
			return
		}
		if verification != "" && status == 200 && file != "shell.html" {
			data = []byte(strings.Replace(string(data), "</head>", `<meta name="google-site-verification" content="`+html.EscapeString(verification)+`"></head>`, 1))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(status)
		if r.Method != "HEAD" {
			_, _ = w.Write(data)
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", 405)
			return
		}
		host := strings.ToLower(strings.Split(r.Host, ":")[0])
		urlPath := path.Clean(r.URL.Path)
		if urlPath == "/index.html" {
			urlPath = "/"
		}
		if strings.HasSuffix(urlPath, "/index.html") {
			urlPath = strings.TrimSuffix(urlPath, "/index.html")
		}
		if r.URL.Path == "/" && host == "app.vitruko.lat" {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			http.Redirect(w, r, "/app", http.StatusTemporaryRedirect)
			return
		}
		if file, ok := pages[urlPath]; ok {
			if host == "api.vitruko.lat" || host == "app.vitruko.lat" {
				http.Redirect(w, r, "https://vitruko.lat"+urlPath, http.StatusPermanentRedirect)
				return
			}
			if r.URL.Path != urlPath {
				target := urlPath
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}
			serveHTML(w, r, file, 200)
			return
		}
		if urlPath == "/sitemap.xml" && (host == "api.vitruko.lat" || host == "app.vitruko.lat") {
			http.Redirect(w, r, "https://vitruko.lat/sitemap.xml", http.StatusPermanentRedirect)
			return
		}
		clean := strings.TrimPrefix(urlPath, "/")
		if clean != "shell.html" && clean != "404.html" && clean != "routes.json" && !strings.HasSuffix(clean, ".html") {
			if info, err := fs.Stat(root, clean); err == nil && !info.IsDir() {
				if strings.HasPrefix(clean, "assets/") {
					w.Header().Set("Cache-Control", "public,max-age=31536000,immutable")
				} else if strings.HasPrefix(clean, "fonts/") || strings.HasSuffix(clean, ".png") || strings.HasSuffix(clean, ".svg") {
					w.Header().Set("Cache-Control", "public,max-age=86400")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		if urlPath == "/app" || strings.HasPrefix(urlPath, "/app/") || strings.HasPrefix(urlPath, "/map/") || urlPath == "/login" || urlPath == "/signup" || urlPath == "/forgot" || urlPath == "/reset" {
			serveHTML(w, r, "shell.html", 200)
			return
		}
		serveHTML(w, r, "404.html", 404)
	})
}
