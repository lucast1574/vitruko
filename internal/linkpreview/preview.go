package linkpreview

import (
	"bytes"
	"golang.org/x/net/html"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"
)

type Metadata struct {
	Title        string      `json:"title"`
	ImageURL     string      `json:"imageUrl"`
	Description  string      `json:"description"`
	URL          string      `json:"url"`
	Price        *Price      `json:"price"`
	Availability string      `json:"availability"`
	Brand        string      `json:"brand"`
	Images       []string    `json:"images"`
	Quality      QualityInfo `json:"quality"`
	sources      map[string]string
	detected     bool
	warnings     []string
}

func truncate(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	for len(value) > max {
		_, n := utf8.DecodeLastRuneInString(value)
		value = value[:len(value)-n]
	}
	return value
}
func Parse(body io.Reader, base string) Metadata {
	out := Metadata{URL: base}
	content, _ := io.ReadAll(io.LimitReader(body, 4<<20))
	z := html.NewTokenizer(bytes.NewReader(content))
	title := false
	structured := Metadata{}
	var products []map[string]any
	var amount, currency string
	inBody := false
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			break
		}
		token := z.Token()
		if kind == html.StartTagToken && token.Data == "body" {
			inBody = true
		}
		if kind == html.StartTagToken && token.Data == "script" {
			for _, attr := range token.Attr {
				if attr.Key == "type" && strings.EqualFold(strings.TrimSpace(attr.Val), "application/ld+json") {
					if z.Next() == html.TextToken {
						products = append(products, decodeProducts(string(z.Text()))...)
					}
					break
				}
			}
		}
		if inBody {
			continue
		}
		if kind == html.StartTagToken && token.Data == "title" {
			title = true
			continue
		}
		if kind == html.EndTagToken && token.Data == "title" {
			title = false
		}
		if title && kind == html.TextToken && out.Title == "" {
			out.Title = token.Data
			setSource(&out, "title", "html_title")
		}
		if (kind != html.StartTagToken && kind != html.SelfClosingTagToken) || token.Data != "meta" {
			continue
		}
		key, value := "", ""
		for _, attr := range token.Attr {
			if attr.Key == "property" || attr.Key == "name" {
				key = strings.ToLower(attr.Val)
			}
			if attr.Key == "content" {
				value = attr.Val
			}
		}
		switch key {
		case "og:type":
			out.detected = value == "product" || value == "og:product"
		case "product:price:amount", "og:price:amount":
			out.detected = true
			amount = value
		case "product:price:currency", "og:price:currency":
			currency = value
		case "product:availability":
			out.Availability = availability(value)
			setSource(&out, "availability", "product_meta")
		case "product:brand":
			out.Brand = value
			setSource(&out, "brand", "product_meta")
		case "og:title", "twitter:title":
			out.Title = value
			setSource(&out, "title", "open_graph")
		case "og:description":
			out.Description = value
			setSource(&out, "description", "html_meta")
		case "description":
			if out.Description == "" {
				out.Description = value
				setSource(&out, "description", "html_meta")
			}
		case "og:image:secure_url":
			out.ImageURL = value
			setSource(&out, "images", "open_graph")
		case "og:image", "twitter:image":
			if out.ImageURL == "" {
				out.ImageURL = value
				setSource(&out, "images", "open_graph")
			}
		}
	}
	structured = selectProduct(products, base)
	out.warnings = append(out.warnings, structured.warnings...)
	if amount != "" && currency != "" {
		out.Price = price(amount, currency)
		if out.Price != nil {
			setSource(&out, "price", "product_meta")
		}
	}
	if structured.Title == "" && len(products) <= 1 {
		structured = microdata(content)
	}
	if structured.Title != "" {
		out = mergeMetadata(out, structured)
	}

	if u, err := url.Parse(base); err == nil {
		out.ImageURL = checkedImageURL(u, out.ImageURL)
	}
	out.Title = truncate(out.Title, 240)
	out.Description = plainStoreDescription(out.Description)
	return normalizeMetadata(out)
}
