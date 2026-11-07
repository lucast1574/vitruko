package linkpreview

import (
	"encoding/json"
	"net/url"
	"strings"
)

func productMetadata(raw string) Metadata { return productMetadataForPage(raw, "") }
func productMetadataForPage(raw, base string) Metadata {
	return selectProduct(decodeProducts(raw), base)
}
func decodeProducts(raw string) []map[string]any {
	var v any
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return nil
	}
	products := []map[string]any{}
	var walk func(any, int)
	walk = func(v any, depth int) {
		if depth > 12 || len(products) >= 128 {
			return
		}
		switch node := v.(type) {
		case []any:
			for _, c := range node {
				walk(c, depth+1)
			}
		case map[string]any:
			if isProduct(node["@type"]) {
				products = append(products, node)
				return
			}
			for _, key := range []string{"@graph", "mainEntity", "hasVariant"} {
				walk(node[key], depth+1)
			}
		}
	}
	walk(v, 0)
	return products
}
func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	}
	return ""
}
func sameProductURL(raw, base string) bool {
	if raw == "" || base == "" {
		return false
	}
	b, err := url.Parse(base)
	if err != nil {
		return false
	}
	u, err := b.Parse(raw)
	if err != nil {
		return false
	}
	u.Fragment = ""
	b.Fragment = ""
	return strings.TrimRight(NormalizeURL(u.String()), "/") == strings.TrimRight(NormalizeURL(b.String()), "/")
}
func nodeMatches(n map[string]any, base string) bool {
	for _, key := range []string{"url", "@id"} {
		if sameProductURL(stringValue(n[key]), base) {
			return true
		}
	}
	return false
}
func selectProduct(nodes []map[string]any, base string) Metadata {
	var chosen map[string]any
	for _, n := range nodes {
		if nodeMatches(n, base) {
			if chosen != nil {
				return Metadata{warnings: []string{"multiple_products"}}
			}
			chosen = n
		}
	}
	if chosen == nil {
		if len(nodes) != 1 {
			if len(nodes) > 1 {
				return Metadata{warnings: []string{"multiple_products"}}
			}
			return Metadata{}
		}
		chosen = nodes[0]
		if raw := stringValue(chosen["url"]); base != "" && raw != "" && !nodeMatches(chosen, base) {
			// A canonical product URL may omit the requested variant, but not change products.
			b, err := url.Parse(base)
			if err != nil {
				return Metadata{warnings: []string{"product_not_confirmed"}}
			}
			q := b.Query()
			q.Del("variant")
			q.Del("skuId")
			b.RawQuery = q.Encode()
			if !sameProductURL(raw, b.String()) {
				return Metadata{warnings: []string{"product_not_confirmed"}}
			}
		}
	}
	name := stringValue(chosen["name"])
	m := Metadata{Title: name, Description: stringValue(chosen["description"]), Images: productImages(chosen["image"], 0), Brand: brandName(chosen["brand"])}
	if len(m.Images) > 0 {
		m.ImageURL = m.Images[0]
	}
	m = sourced(m, "json_ld")
	offer, warning := selectOffer(chosen["offers"], base)
	if warning != "" {
		m.warnings = append(m.warnings, warning)
	}
	// A variant URL must not silently inherit a generic offer's price or stock.
	page, _ := url.Parse(base)
	selectedVariant := page != nil && (page.Query().Get("variant") != "" || page.Query().Get("skuId") != "")
	if offer != nil && selectedVariant && !nodeMatches(offer, base) && !nodeMatches(chosen, base) {
		m.warnings = append(m.warnings, "variant_not_confirmed")
	} else if offer != nil {
		m.Price = price(stringValue(offer["price"]), stringValue(offer["priceCurrency"]))
		m.Availability = availability(stringValue(offer["availability"]))
	}
	return m
}
func brandName(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		return stringValue(x["name"])
	}
	return ""
}
func selectOffer(v any, base string) (map[string]any, string) {
	switch n := v.(type) {
	case map[string]any:
		if stringValue(n["@type"]) == "AggregateOffer" || n["lowPrice"] != nil || n["highPrice"] != nil {
			return nil, "price_range"
		}
		return n, ""
	case []any:
		if len(n) == 1 {
			return selectOffer(n[0], base)
		}
		var found map[string]any
		for _, v := range n {
			if x, ok := v.(map[string]any); ok && nodeMatches(x, base) {
				if found != nil {
					return nil, "multiple_offers"
				}
				found = x
			}
		}
		if found != nil {
			return selectOffer(found, base)
		}
		if len(n) > 0 {
			return nil, "multiple_offers"
		}
	}
	return nil, ""
}
func isProduct(value any) bool {
	switch kind := value.(type) {
	case string:
		return kind == "Product" || kind == "ProductGroup" || strings.TrimSuffix(kind, "/") == "https://schema.org/Product" || strings.TrimSuffix(kind, "/") == "http://schema.org/Product"
	case []any:
		for _, entry := range kind {
			if isProduct(entry) {
				return true
			}
		}
	}
	return false
}
func productImages(v any, depth int) []string {
	if depth > 6 {
		return nil
	}
	out := []string{}
	switch x := v.(type) {
	case string:
		out = append(out, x)
	case []any:
		for _, v := range x {
			out = append(out, productImages(v, depth+1)...)
			if len(out) >= 20 {
				return out[:20]
			}
		}
	case map[string]any:
		if u := stringValue(x["url"]); u != "" {
			out = append(out, u)
		} else if u = stringValue(x["contentUrl"]); u != "" {
			out = append(out, u)
		}
	}
	return out
}
func productImage(v any, depth int) string {
	images := productImages(v, depth)
	if len(images) > 0 {
		return images[0]
	}
	return ""
}
