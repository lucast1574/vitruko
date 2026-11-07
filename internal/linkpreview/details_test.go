package linkpreview

import (
	"strings"
	"testing"
	"time"
)

func TestRichProductAndHonestQuality(t *testing.T) {
	html := `<title>Generic shop</title><meta property="og:image" content="/logo.jpg"><script type="application/ld+json">{"@type":"Product","name":"Cafetera","description":"<b>Café</b> en casa","brand":{"name":"Casa"},"image":["/a.jpg","/a.jpg",{"contentUrl":"//cdn.example/b.jpg"},"http://shop.example/unsafe.jpg","https://127.0.0.1/private"],"offers":{"@type":"Offer","price":129.90,"priceCurrency":"PEN","availability":"https://schema.org/InStock"}}</script>`
	m := Complete(Parse(strings.NewReader(html), "https://shop.example/cafetera"))
	if m.Title != "Cafetera" || m.ImageURL != "https://shop.example/a.jpg" || len(m.Images) != 2 || m.Brand != "Casa" || m.Description != "Café en casa" || m.Price == nil || m.Price.Amount != "129.90" || m.Price.Currency != "PEN" || m.Availability != "in_stock" {
		t.Fatalf("unexpected product: %+v", m)
	}
	if !m.Quality.ProductDetected || len(m.Quality.MissingFields) != 0 || m.Quality.FieldSources["price"] != "json_ld" || len(m.Quality.Warnings) != 0 {
		t.Fatal(m.Quality)
	}
	if _, err := time.Parse(time.RFC3339, m.Quality.ExtractedAt); err != nil {
		t.Fatal(err)
	}
}
func TestPricesNeverInferCurrencyOrChooseArbitraryOffers(t *testing.T) {
	for _, offer := range []string{
		`{"price":"19.90"}`,
		`{"price":"$19,90","priceCurrency":"PEN"}`,
		`{"price":-1,"priceCurrency":"PEN"}`,
		`{"@type":"AggregateOffer","lowPrice":"10","highPrice":"20","priceCurrency":"USD"}`,
		`[{"price":"10","priceCurrency":"USD"},{"price":"20","priceCurrency":"USD"}]`,
	} {
		m := Complete(Parse(strings.NewReader(`<script type="application/ld+json">{"@type":"Product","name":"Cup","offers":`+offer+`}</script>`), "https://shop.example/cup"))
		if m.Price != nil {
			t.Fatal("invented price", m.Price)
		}
	}
	m := Complete(Parse(strings.NewReader(`<meta property="og:title" content="Store page">`), "https://shop.example/page"))
	if m.Quality.ProductDetected || m.Price != nil || m.Availability != "unknown" || len(m.Images) != 0 || len(m.Quality.MissingFields) == 0 || m.Quality.Warnings[0] != "product_not_confirmed" {
		t.Fatal(m)
	}
}
func TestSelectExactProductAcrossScriptsAndVariantOffer(t *testing.T) {
	h := `<script type="application/ld+json">{"@type":"Product","name":"Unrelated","url":"/other","offers":{"price":"999","priceCurrency":"USD"}}</script><script type="application/ld+json">{"@type":"Product","name":"Cup","url":"/cup","offers":[{"url":"/cup?variant=1","price":"10.00","priceCurrency":"PEN"},{"url":"/cup?variant=2","price":"20.00","priceCurrency":"PEN","availability":"https://schema.org/OutOfStock"}]}</script>`
	// Two products without an exact requested variant identity are ambiguous.
	m := Complete(Parse(strings.NewReader(h), "https://shop.example/cup?variant=2"))
	if m.Price != nil {
		t.Fatal("cross-product price", m)
	}
	h = strings.ReplaceAll(h, `"url":"/cup","offers"`, `"url":"/cup?variant=2","offers"`)
	m = Complete(Parse(strings.NewReader(h), "https://shop.example/cup?variant=2"))
	if m.Title != "Cup" || m.Price == nil || m.Price.Amount != "20.00" || m.Availability != "out_of_stock" {
		t.Fatal(m)
	}
	generic := `<script type="application/ld+json">{"@type":"Product","name":"Cup","offers":{"price":"10","priceCurrency":"USD","availability":"InStock"}}</script>`
	m = Complete(Parse(strings.NewReader(generic), "https://shop.example/cup?variant=2"))
	if m.Price != nil || m.Availability != "unknown" {
		t.Fatal("generic price used for variant", m)
	}
}
func TestRichMicrodataKeepsBrandAndOfferScopeSeparate(t *testing.T) {
	h := `<div itemscope itemtype="https://schema.org/Product"><span itemprop="name">Cup</span><div itemprop="brand" itemscope itemtype="https://schema.org/Brand"><span itemprop="name">Maker</span></div><img itemprop="image" src="/a.jpg"><img itemprop="image" src="/b.jpg"><div itemprop="offers" itemscope itemtype="https://schema.org/Offer"><span itemprop="name">Seller</span><meta itemprop="price" content="0"><meta itemprop="priceCurrency" content="USD"><link itemprop="availability" href="https://schema.org/PreOrder"></div></div>`
	m := Complete(Parse(strings.NewReader(h), "https://shop.example/cup"))
	if m.Title != "Cup" || m.Brand != "Maker" || m.Price == nil || m.Price.Amount != "0" || m.Availability != "preorder" || len(m.Images) != 2 || m.Quality.FieldSources["price"] != "microdata" {
		t.Fatal(m)
	}
}

func TestUnrelatedExplicitProductIdentityNeverSuppliesAPrice(t *testing.T) {
	h := `<meta property="og:title" content="Current page"><script type="application/ld+json">{"@type":"Product","name":"Wrong product","url":"/other","offers":{"price":"100","priceCurrency":"USD"}}</script>`
	m := Complete(Parse(strings.NewReader(h), "https://shop.example/cup"))
	if m.Title != "Current page" || m.Price != nil || m.Quality.ProductDetected {
		t.Fatal(m)
	}
}
