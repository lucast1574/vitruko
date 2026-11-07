package linkpreview

import (
	"context"
	"encoding/json"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Public storefront APIs complement HTML metadata; no scripts are executed,
// no browser cookies are forwarded and every destination uses the checked client.
func storefront(ctx context.Context, client *http.Client, page *url.URL, body []byte) Metadata {
	if product := shopifyStorefront(ctx, client, page); product.Title != "" {
		return product
	}
	return vtexStorefront(ctx, client, page, body)
}
func shopifyStorefront(ctx context.Context, client *http.Client, page *url.URL) Metadata {
	parts := strings.Split(strings.Trim(page.Path, "/"), "/")
	endpoint := *page
	endpoint.RawQuery, endpoint.Fragment = "", ""
	if len(parts) >= 2 && parts[len(parts)-2] == "products" && !strings.Contains(parts[len(parts)-1], ".") {
		endpoint.Path = strings.TrimRight(page.Path, "/") + ".js"
		var product struct {
			Title         string `json:"title"`
			Description   string `json:"description"`
			FeaturedImage string `json:"featured_image"`
			Variants      []struct {
				ID            json.Number `json:"id"`
				FeaturedImage *struct {
					Src string `json:"src"`
				} `json:"featured_image"`
			} `json:"variants"`
		}
		if readStoreJSON(ctx, client, endpoint.String(), &product) {
			if id := page.Query().Get("variant"); id != "" && len(product.Variants) > 0 {
				found := false
				for _, variant := range product.Variants {
					if variant.ID.String() == id {
						found = true
						if variant.FeaturedImage != nil {
							product.FeaturedImage = variant.FeaturedImage.Src
						}
						break
					}
				}
				if !found {
					return Metadata{}
				}
			}
			return checkedStoreMetadata(Metadata{Title: product.Title, ImageURL: product.FeaturedImage, Description: product.Description}, page)
		}
	}
	return Metadata{}
}
func vtexStorefront(ctx context.Context, client *http.Client, page *url.URL, body []byte) Metadata {
	parts := strings.Split(strings.Trim(page.Path, "/"), "/")
	endpoint := *page
	endpoint.RawQuery, endpoint.Fragment = "", ""
	if len(parts) == 2 && parts[1] == "p" && strings.Contains(strings.ToLower(string(body)), "vtex") {
		endpoint.Path = "/api/catalog_system/pub/products/search/" + parts[0] + "/p"
		var products []struct {
			Name        string `json:"productName"`
			LinkText    string `json:"linkText"`
			Description string `json:"description"`
			Items       []struct {
				Images []struct {
					URL string `json:"imageUrl"`
				} `json:"images"`
			} `json:"items"`
		}
		if readStoreJSON(ctx, client, endpoint.String(), &products) {
			for _, product := range products {
				if product.LinkText != parts[0] {
					continue
				}
				image := ""
				if len(product.Items) > 0 && len(product.Items[0].Images) > 0 {
					image = product.Items[0].Images[0].URL
				}
				return checkedStoreMetadata(Metadata{Title: product.Name, ImageURL: image, Description: product.Description}, page)
			}
		}
	}
	return Metadata{}
}
func readStoreJSON(ctx context.Context, client *http.Client, raw string, dst any) bool {
	if !ValidURL(raw) {
		return false
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", "Vitruko/1.0 (+https://vitruko.nexode.app; product preview)")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	mime := strings.ToLower(strings.Split(res.Header.Get("Content-Type"), ";")[0])
	if (res.StatusCode != 200 && res.StatusCode != 206) || (!strings.Contains(mime, "json") && mime != "text/javascript" && mime != "application/javascript") {
		return false
	}
	return json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(dst) == nil
}
func checkedStoreMetadata(data Metadata, page *url.URL) Metadata {
	if !usefulTitle(data.Title, page.String()) {
		return Metadata{}
	}
	data.Title = truncate(data.Title, 240)
	data.Description = plainStoreDescription(data.Description)
	data.ImageURL = checkedImageURL(page, data.ImageURL)
	data.URL = page.String()
	return data
}

func plainStoreDescription(raw string) string {
	z := html.NewTokenizer(strings.NewReader(raw))
	var text strings.Builder
	skip := false
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			break
		}
		token := z.Token()
		if kind == html.StartTagToken && (token.Data == "script" || token.Data == "style") {
			skip = true
		}
		if kind == html.EndTagToken && (token.Data == "script" || token.Data == "style") {
			skip = false
			continue
		}
		if kind == html.TextToken && !skip {
			text.WriteString(token.Data)
			text.WriteByte(' ')
		}
	}
	return truncate(text.String(), 1000)
}
