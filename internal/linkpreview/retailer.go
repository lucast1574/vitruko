package linkpreview

import (
	"net/url"
	"strings"
)

// Keep variant and seller identifiers; remove only known share/analytics fields.
// In particular Mercado Libre /up links need pdp_filters/item_id to identify the item.
func NormalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			q.Del(key)
		}
	}
	for _, key := range []string{"matt_tool", "matt_word", "matt_source", "matt_campaign_id", "gclid", "fbclid"} {
		q.Del(key)
	}
	if mercadoHost(u.Hostname()) {
		for _, key := range []string{"ua", "origin", "sid", "action"} {
			q.Del(key)
		}
		// Some copied links carry the item solely in the fragment.
		fragment, _ := url.ParseQuery(u.Fragment)
		if q.Get("pdp_filters") == "" && q.Get("wid") == "" && fragment.Get("wid") != "" {
			q.Set("pdp_filters", "item_id:"+fragment.Get("wid"))
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}

func mercadoHost(host string) bool {
	host = strings.ToLower(host)
	for _, domain := range []string{"mercadolibre.com.pe", "mercadolibre.com.ar", "mercadolibre.com.mx", "mercadolibre.com.co", "mercadolibre.cl", "mercadolibre.com.uy", "mercadolibre.com.ec", "mercadolibre.com.bo", "mercadolibre.com.ve", "mercadolivre.com.br"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func checkedImageURL(page *url.URL, raw string) string {
	image, err := page.Parse(raw)
	if err != nil || raw == "" || !ValidURL(image.String()) {
		return ""
	}
	// Adafruit publishes some product images with a duplicated path separator.
	// Its CDN returns 404 for that form and 200 for the canonical image path.
	// Do not rewrite arbitrary CDN paths, escaped paths, or signed query strings.
	if strings.EqualFold(image.Hostname(), "cdn-shop.adafruit.com") && image.RawQuery == "" && image.RawPath == "" {
		for strings.Contains(image.Path, "//") {
			image.Path = strings.ReplaceAll(image.Path, "//", "/")
		}
	}
	return image.String()
}

// A login, search or storefront title is not the name of the requested gift.
func usefulTitle(title, raw string) bool {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return false
	}
	for _, generic := range []string{"mercado libre", "mercado livre", "coolbox", "ripley", "falabella", "access denied", "just a moment...", "robot or human?", "security check", "iniciar sesión", "login", "página no encontrada", "page not found"} {
		if title == generic {
			return false
		}
	}
	u, _ := url.Parse(raw)
	if u != nil && mercadoHost(u.Hostname()) && (strings.Contains(title, "verifica") || strings.Contains(title, "captcha")) {
		return false
	}
	return true
}
