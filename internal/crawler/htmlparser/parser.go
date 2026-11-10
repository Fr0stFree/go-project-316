// Package htmlparser provides utilities for extracting data from html pages.
package htmlparser

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"slices"
	"strings"

	"code/internal/common/httptools"
	"code/internal/common/stringtools"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/xmlquery"
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
func Parse(r io.Reader, pageURL *url.URL, parseRSS bool) (*ParsedHTML, error) {
	var data []byte

	if parseRSS {
		var err error

		data, err = io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		r = bytes.NewReader(data)
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	baseURL := retrieveDocumentBaseURL(doc, pageURL)
	links := findLinks(doc, pageURL, baseURL)

	if parseRSS {
		links = append(links, findRSSLinks(data, pageURL)...)
	}

	return &ParsedHTML{
		Links:       links,
		Assets:      slices.Concat(findImages(doc, baseURL), findScripts(doc, baseURL), findStyles(doc, baseURL)),
		Title:       findTitle(doc),
		Description: findDescription(doc),
		H1:          findHeader(doc),
	}, nil
}

func retrieveDocumentBaseURL(doc *goquery.Document, pageURL *url.URL) *url.URL {
	rawBase, exists := doc.Find("base[href]").First().Attr("href")
	if !exists {
		return pageURL
	}

	baseURL, err := resolveURL(strings.TrimSpace(rawBase), pageURL)
	if err != nil {
		return pageURL
	}

	return baseURL
}

func findLinks(doc *goquery.Document, pageURL, baseURL *url.URL) []*url.URL {
	links := make([]*url.URL, 0)

	normalizedPageURL := httptools.NormalizeURL(pageURL)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		rawURL := strings.TrimSpace(s.AttrOr("href", ""))
		if rawURL == "" {
			return
		}

		absoluteURL, err := resolveURL(rawURL, baseURL)
		if err != nil {
			return
		}

		if strings.HasPrefix(rawURL, "#") && absoluteURL.String() == normalizedPageURL.String() {
			return
		}

		links = append(links, absoluteURL)
	})

	return links
}

func findRSSLinks(data []byte, pageURL *url.URL) []*url.URL {
	links := make([]*url.URL, 0)

	doc, err := xmlquery.Parse(bytes.NewReader(data))
	if err != nil {
		return links
	}

	for _, node := range xmlquery.Find(doc, "//rss/channel/link | //rss/channel/item/link") {
		absoluteURL, err := resolveURL(strings.TrimSpace(node.InnerText()), pageURL)
		if err != nil {
			continue
		}

		links = append(links, absoluteURL)
	}

	return links
}

func findHeader(doc *goquery.Document) *string {
	h1 := doc.Find("h1").First()
	if h1.Length() == 0 {
		return nil
	}

	return lo.ToPtr(stringtools.CleanText(h1.Text()))
}

func findDescription(doc *goquery.Document) *string {
	description := doc.Find("meta[name]").FilterFunction(
		func(_ int, s *goquery.Selection) bool {
			return strings.EqualFold(s.AttrOr("name", ""), "description")
		},
	).First()

	if description.Length() == 0 {
		return nil
	}

	return lo.ToPtr(stringtools.CleanText(description.AttrOr("content", "")))
}

func findTitle(doc *goquery.Document) *string {
	title := doc.Find("title").First()
	if title.Length() == 0 {
		return nil
	}

	return lo.ToPtr(stringtools.CleanText(title.Text()))
}

func findImages(doc *goquery.Document, pageURL *url.URL) []Asset {
	var assets []Asset

	addImage := func(rawURL string) {
		absoluteURL, err := resolveURL(strings.TrimSpace(rawURL), pageURL)
		if err != nil {
			return
		}

		assets = append(assets, Asset{Type: AssetImage, URL: absoluteURL})
	}

	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		addImage(s.AttrOr("src", ""))
	})

	doc.Find("img[srcset], picture source[srcset]").Each(func(_ int, s *goquery.Selection) {
		for _, rawURL := range parseSrcset(s.AttrOr("srcset", "")) {
			addImage(rawURL)
		}
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

	doc.Find("link[rel][href]").Each(func(_ int, s *goquery.Selection) {
		isStylesheet := false

		for _, rel := range strings.Fields(s.AttrOr("rel", "")) {
			if strings.EqualFold(rel, "stylesheet") {
				isStylesheet = true

				break
			}
		}

		if !isStylesheet {
			return
		}

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

	if absoluteURL.Scheme != httptools.HTTP && absoluteURL.Scheme != httptools.HTTPS {
		return nil, errors.New("unsupported URL scheme")
	}

	if absoluteURL.Host == "" {
		return nil, errors.New("URL host is empty")
	}

	return httptools.NormalizeURL(absoluteURL), nil
}

func parseSrcset(srcset string) []string {
	var urls []string

	for candidate := range strings.SplitSeq(srcset, ",") {
		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}

		urls = append(urls, fields[0])
	}

	return urls
}
