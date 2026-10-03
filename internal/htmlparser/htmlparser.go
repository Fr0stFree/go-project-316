// Package htmlparser provides functions for parsing HTML content.
package htmlparser

import (
	"errors"
	"io"

	"golang.org/x/net/html"
)

// ParsedPage represents a parsed HTML page.
type ParsedPage struct {
	Links []string
}

// ParsePage parses the HTML content from the provided reader and extracts all the links from anchor tags.
func ParsePage(r io.Reader) (ParsedPage, error) {
	page := ParsedPage{Links: make([]string, 0)}

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
