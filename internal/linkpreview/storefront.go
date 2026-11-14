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
			Title         string   `json:"title"`
			Description   string   `json:"description"`
			FeaturedImage string   `json:"featured_image"`
			Vendor        string   `json:"vendor"`
			Images        []string `json:"images"`
			Available     *bool    `json:"available"`
			Variants      []struct {
				ID            json.Number `json:"id"`
				Available     *bool       `json:"available"`
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
						product.Available = variant.Available
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
			if page.Query().Get("variant") != "" && len(product.Variants) == 0 {
				product.Available = nil
			}
			stock := "unknown"
			if product.Available != nil {
				stock = "out_of_stock"
				if *product.Available {
					stock = "in_stock"
				}
			}
			return checkedStoreMetadata(sourced(Metadata{Title: product.Title, ImageURL: product.FeaturedImage, Images: product.Images, Description: product.Description, Brand: product.Vendor, Availability: stock}, "shopify_json"), page)
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
			Brand       string `json:"brand"`
			Items       []struct {
				ID      string `json:"itemId"`
				Sellers []struct {
					Offer struct {
						Price    json.Number `json:"Price"`
						Currency string      `json:"CurrencyCode"`
						Quantity *int        `json:"AvailableQuantity"`
					} `json:"commertialOffer"`
				} `json:"sellers"`
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
				m := sourced(Metadata{Title: product.Name, Description: product.Description, Brand: product.Brand}, "vtex_json")
				items := product.Items
				if sku := page.Query().Get("skuId"); sku != "" {
					items = nil
					for _, item := range product.Items {
						if item.ID == sku {
							items = append(items, item)
							break
						}
					}
					if len(items) == 0 {
						return Metadata{}
					}
				}
				for _, item := range items {
					for _, image := range item.Images {
						m.Images = append(m.Images, image.URL)
					}
				}
				if len(items) == 1 && len(items[0].Sellers) == 1 {
					offer := items[0].Sellers[0].Offer
					m.Price = price(offer.Price.String(), offer.Currency)
					if offer.Quantity != nil {
						m.Availability = "out_of_stock"
						if *offer.Quantity > 0 {
							m.Availability = "in_stock"
						}
					}
				} else if len(items) > 0 {
					m.warnings = append(m.warnings, "multiple_offers")
				}
				return checkedStoreMetadata(m, page)
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
	req.Header.Set("User-Agent", "Vitruko/1.0 (+https://vitruko.lat; product preview)")
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
	return normalizeMetadata(data)
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
