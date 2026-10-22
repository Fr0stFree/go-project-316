// Package htmlparser provides utilities for extracting data from html pages.
package htmlparser

import (
	"errors"
	"io"
	"net/url"
	"slices"
	"strings"

	"code/internal/common/httptools"

	"github.com/PuerkitoBio/goquery"
	"github.com/samber/lo"
)

// AssetType represents the type of a resource referenced by an HTML page.
type AssetType string

const (
	// AssetImage represents an image resource.
	AssetImage AssetType = "image"
	// AssetScript represents a JavaScript resource.
	AssetScript AssetType = "script"
	// AssetStyle represents a CSS stylesheet.
	AssetStyle AssetType = "style"
)

// Asset represents a resource referenced by an HTML page.
type Asset struct {
	Type AssetType
	URL  *url.URL
}

// ParsedHTML represents links and SEO metadata extracted from an HTML document.
type ParsedHTML struct {
	Links       []*url.URL
	Assets      []Asset
	Title       *string
	Description *string
	H1          *string
}

// Parse reads an HTML document and extracts links and SEO metadata.
func Parse(r io.Reader, pageURL *url.URL) (*ParsedHTML, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	page := &ParsedHTML{
		Links:       findLinks(doc, pageURL),
		Assets:      slices.Concat(findImages(doc, pageURL), findScripts(doc, pageURL), findStyles(doc, pageURL)),
		Title:       findTitle(doc),
		Description: findDescription(doc),
		H1:          findHeader(doc),
	}

	return page, nil
}

func findLinks(doc *goquery.Document, pageURL *url.URL) []*url.URL {
	links := make([]*url.URL, 0)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		rawURL, _ := s.Attr("href")

		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" || strings.HasPrefix(rawURL, "#") {
			return
		}

		absoluteURL, err := resolveURL(rawURL, pageURL)
		if err != nil {
			return
		}

		links = append(links, absoluteURL)
	})

	return links
}

func findHeader(doc *goquery.Document) *string {
	h1 := doc.Find("h1").First()
	if h1.Length() > 0 {
		return lo.ToPtr(h1.Text())
	}

	return nil
}

func findDescription(doc *goquery.Document) *string {
	description := doc.Find(`meta[name="description"]`).First()
	if description.Length() > 0 {
		return lo.ToPtr(description.AttrOr("content", ""))
	}

	return nil
}

func findTitle(doc *goquery.Document) *string {
	title := doc.Find("title").First()
	if title.Length() > 0 {
		return lo.ToPtr(title.Text())
	}

	return nil
}

func findImages(doc *goquery.Document, pageURL *url.URL) []Asset {
	var assets []Asset

	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		rawURL, _ := s.Attr("src")

		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}

		absoluteURL, err := resolveURL(rawURL, pageURL)
		if err != nil {
			return
		}

		asset := Asset{Type: AssetImage, URL: absoluteURL}
		assets = append(assets, asset)
	})

	return assets
}

func findScripts(doc *goquery.Document, pageURL *url.URL) []Asset {
	var assets []Asset

	doc.Find("script[src]").Each(func(_ int, s *goquery.Selection) {
		rawURL, _ := s.Attr("src")

		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}

		absoluteURL, err := resolveURL(rawURL, pageURL)
		if err != nil {
			return
		}

		asset := Asset{Type: AssetScript, URL: absoluteURL}
		assets = append(assets, asset)
	})

	return assets
}

func findStyles(doc *goquery.Document, pageURL *url.URL) []Asset {
	var assets []Asset

	doc.Find(`link[rel~="stylesheet"][href]`).Each(func(_ int, s *goquery.Selection) {
		rawURL, _ := s.Attr("href")

		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}

		absoluteURL, err := resolveURL(rawURL, pageURL)
		if err != nil {
			return
		}

		asset := Asset{Type: AssetStyle, URL: absoluteURL}
		assets = append(assets, asset)
	})

	return assets
}

func resolveURL(rawURL string, pageURL *url.URL) (*url.URL, error) {
	ref, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	absoluteURL := pageURL.ResolveReference(ref)

	if absoluteURL.Scheme != "http" && absoluteURL.Scheme != "https" {
		return nil, errors.New("unsupported URL scheme")
	}

	if absoluteURL.Host == "" {
		return nil, errors.New("URL host is empty")
	}

	return httptools.NormalizeURL(absoluteURL), nil
}
