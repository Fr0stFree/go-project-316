package htmlparser

import (
	"code/internal/common/types"
	"errors"
	"io"
	"net/url"
	"slices"

	"golang.org/x/net/html"
)

type ParsedPage struct {
	Links []types.URL
}

func ParsePage(r io.Reader, pageURL types.URL) (ParsedPage, error) {
	page := ParsedPage{Links: make([]types.URL, 0)}

	baseURL, err := url.Parse(string(pageURL))
	if err != nil {
		return page, err
	}

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
				if attr.Key != "href" || attr.Val == "" {
					break
				}

				ref, err := url.Parse(attr.Val)
				if err != nil {
					break
				}

				absoluteURL := baseURL.ResolveReference(ref)
				if !slices.Contains([]string{"http", "https"}, absoluteURL.Scheme) {
					break
				}

				absoluteURL.Fragment = ""
				page.Links = append(page.Links, types.URL(absoluteURL.String()))
			}
		}
	}
}
