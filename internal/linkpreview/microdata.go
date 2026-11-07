package linkpreview

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

func microdata(body []byte) Metadata {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Metadata{}
	}
	budget := 50000
	var find func(*html.Node, int) Metadata
	find = func(n *html.Node, depth int) Metadata {
		budget--
		if depth > 128 || budget < 0 {
			return Metadata{}
		}
		for _, kind := range strings.Fields(attribute(n, "itemtype")) {
			if isProduct(kind) {
				return readMicroProduct(n)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if found := find(child, depth+1); found.Title != "" {
				return found
			}
		}
		return Metadata{}
	}
	return find(root, 0)
}
func attribute(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func readMicroProduct(root *html.Node) Metadata {
	out := sourced(Metadata{}, "microdata")
	var offers []*html.Node
	budget := 10000
	var visit func(*html.Node, int)
	visit = func(n *html.Node, depth int) {
		budget--
		if depth > 64 || budget < 0 {
			return
		}
		if n != root && attribute(n, "itemtype") != "" && attribute(n, "itemprop") != "image" {
			prop := attribute(n, "itemprop")
			if prop == "offers" {
				offers = append(offers, n)
			}
			if prop == "brand" {
				out.Brand = microValueFor(n, "name")
			}
			return
		}
		for _, prop := range strings.Fields(attribute(n, "itemprop")) {
			value := attribute(n, "content")
			if value == "" {
				value = attribute(n, "src")
			}
			if value == "" {
				value = attribute(n, "href")
			}
			if value == "" {
				value = nodeText(n, 0)
			}
			switch prop {
			case "name":
				if out.Title == "" {
					out.Title = value
				}
			case "brand":
				out.Brand = value
			case "image":
				out.Images = append(out.Images, value)
				if out.ImageURL == "" {
					out.ImageURL = value
				}
			case "description":
				if out.Description == "" {
					out.Description = value
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child, depth+1)
		}
	}
	visit(root, 0)
	if len(offers) == 1 {
		out.Price = price(microValueFor(offers[0], "price"), microValueFor(offers[0], "priceCurrency"))
		out.Availability = availability(microValueFor(offers[0], "availability"))
	} else if len(offers) > 1 {
		out.warnings = append(out.warnings, "multiple_offers")
	}
	return out
}
func nodeText(n *html.Node, depth int) string {
	if depth > 20 {
		return ""
	}
	if n.Type == html.TextNode {
		return truncate(n.Data, 1000)
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil && b.Len() < 1000; c = c.NextSibling {
		b.WriteString(nodeText(c, depth+1))
		b.WriteByte(' ')
	}
	return truncate(b.String(), 1000)
}

func microValueFor(root *html.Node, prop string) string {
	budget := 1000
	var find func(*html.Node, int) string
	find = func(n *html.Node, depth int) string {
		budget--
		if depth > 20 || budget < 0 {
			return ""
		}
		if n != root && attribute(n, "itemtype") != "" {
			return ""
		}
		for _, p := range strings.Fields(attribute(n, "itemprop")) {
			if p == prop {
				for _, attr := range []string{"content", "href", "src"} {
					if v := attribute(n, attr); v != "" {
						return v
					}
				}
				return nodeText(n, 0)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if v := find(c, depth+1); v != "" {
				return v
			}
		}
		return ""
	}
	return find(root, 0)
}
