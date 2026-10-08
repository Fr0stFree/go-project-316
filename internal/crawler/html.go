// Package crawler provides functions for parsing HTML content.
package crawler

import (
	"errors"
	"io"

	"golang.org/x/net/html"
)

// ParsedPage represents a parsed HTML page.
type parsedPage struct {
	Links []string
}

// ParsePage parses the HTML content from the provided reader and extracts all the links from anchor tags.
func parseHTMLPage(r io.Reader) (parsedPage, error) {
	page := parsedPage{Links: make([]string, 0)}

	tokenizer := html.NewTokenizer(r)
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return page, nil
			}

			return page, tokenizer.Err()

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()

			if token.Data != "a" {
				continue
			}

			for _, attr := range token.Attr {
				if attr.Key == "href" {
					page.Links = append(page.Links, attr.Val)

					break
				}
			}
		}
	}
}
