// Package htmlparser provides utilities for extracting data from html pages.
package htmlparser

import (
	"io"
	"net/url"
	"slices"

	"github.com/PuerkitoBio/goquery"
	"github.com/samber/lo"
)

// ParsedHTML represents links and SEO metadata extracted from an HTML document.
type ParsedHTML struct {
	Links       []*url.URL
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
		Links:       collectLinks(doc, pageURL),
		Title:       findTitle(doc),
		Description: findDescription(doc),
		H1:          findHeader(doc),
	}

	return page, nil
}

func collectLinks(doc *goquery.Document, pageURL *url.URL) []*url.URL {
	links := make([]*url.URL, 0)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href == "" {
			return
		}

		ref, err := url.Parse(href)
		if err != nil {
			return
		}

		absoluteURL := pageURL.ResolveReference(ref)
		if !slices.Contains([]string{"http", "https"}, absoluteURL.Scheme) {
			return
		}

		absoluteURL.Fragment = ""

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
