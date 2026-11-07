package linkpreview

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Amount is a decimal string in major currency units, never a binary float.
type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
type QualityInfo struct {
	ProductDetected bool              `json:"productDetected"`
	MissingFields   []string          `json:"missingFields"`
	FieldSources    map[string]string `json:"fieldSources"`
	Warnings        []string          `json:"warnings"`
	ExtractedAt     string            `json:"extractedAt"`
}

var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,17})(\.[0-9]{1,6})?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func price(amount, currency string) *Price {
	amount, currency = strings.TrimSpace(amount), strings.ToUpper(strings.TrimSpace(currency))
	if !decimalPattern.MatchString(amount) || !currencyPattern.MatchString(currency) {
		return nil
	}
	return &Price{Amount: amount, Currency: currency}
}
func availability(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		raw = raw[i+1:]
	}
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "_", ""), " ", "")
	switch raw {
	case "instock", "available", "limitedavailability":
		return "in_stock"
	case "outofstock", "soldout":
		return "out_of_stock"
	case "preorder", "presale":
		return "preorder"
	case "backorder":
		return "backorder"
	case "discontinued":
		return "discontinued"
	default:
		return "unknown"
	}
}
func setSource(m *Metadata, field, source string) {
	if m.sources == nil {
		m.sources = map[string]string{}
	}
	m.sources[field] = source
}
func sourced(m Metadata, source string) Metadata {
	for _, f := range []string{"title", "description", "images", "brand", "price", "availability"} {
		setSource(&m, f, source)
	}
	m.detected = true
	return m
}

// Merge only fields belonging to the same product; preferred fields win as a unit.
func mergeMetadata(base, preferred Metadata) Metadata {
	if preferred.Title != "" {
		base.Title = preferred.Title
		setSource(&base, "title", preferred.sources["title"])
	}
	if preferred.Description != "" {
		base.Description = preferred.Description
		setSource(&base, "description", preferred.sources["description"])
	}
	if preferred.ImageURL != "" || len(preferred.Images) > 0 {
		base.ImageURL = preferred.ImageURL
		base.Images = preferred.Images
		setSource(&base, "images", preferred.sources["images"])
	}
	if preferred.Brand != "" {
		base.Brand = preferred.Brand
		setSource(&base, "brand", preferred.sources["brand"])
	}
	if preferred.Price != nil {
		base.Price = preferred.Price
		setSource(&base, "price", preferred.sources["price"])
	}
	if preferred.Availability != "" && preferred.Availability != "unknown" {
		base.Availability = preferred.Availability
		setSource(&base, "availability", preferred.sources["availability"])
	}
	base.detected = base.detected || preferred.detected
	base.warnings = append(base.warnings, preferred.warnings...)
	return base
}
func normalizeMetadata(m Metadata) Metadata {
	m.Title = truncate(m.Title, 240)
	m.Description = plainStoreDescription(m.Description)
	m.Brand = truncate(plainStoreDescription(m.Brand), 120)
	m.Availability = availability(m.Availability)
	if m.Price != nil {
		m.Price = price(m.Price.Amount, m.Price.Currency)
	}
	images := []string{}
	if page, err := url.Parse(m.URL); err == nil {
		for _, raw := range append([]string{m.ImageURL}, m.Images...) {
			image := checkedImageURL(page, raw)
			if image == "" || len(image) > 4096 {
				continue
			}
			found := false
			for _, prev := range images {
				if image == prev {
					found = true
					break
				}
			}
			if !found {
				images = append(images, image)
			}
			if len(images) == 20 {
				break
			}
		}
	}
	m.Images = images
	m.ImageURL = ""
	if len(images) > 0 {
		m.ImageURL = images[0]
	}
	return m
}

// Complete describes the observation at extraction time. Cached responses retain this timestamp.
func Complete(m Metadata) Metadata {
	m = normalizeMetadata(m)
	fields := map[string]bool{"title": m.Title != "", "description": m.Description != "", "images": len(m.Images) > 0, "brand": m.Brand != "", "price": m.Price != nil, "availability": m.Availability != "unknown"}
	q := QualityInfo{ProductDetected: m.detected, MissingFields: []string{}, Warnings: []string{}, FieldSources: map[string]string{}, ExtractedAt: time.Now().UTC().Format(time.RFC3339)}
	for _, field := range []string{"title", "description", "images", "brand", "price", "availability"} {
		if !fields[field] {
			q.MissingFields = append(q.MissingFields, field)
		} else if m.sources[field] != "" {
			q.FieldSources[field] = m.sources[field]
		}
	}
	if !m.detected {
		q.Warnings = append(q.Warnings, "product_not_confirmed")
	}
	for _, w := range m.warnings {
		seen := false
		for _, p := range q.Warnings {
			if w == p {
				seen = true
			}
		}
		if !seen {
			q.Warnings = append(q.Warnings, w)
		}
	}
	m.Quality = q
	return m
}
