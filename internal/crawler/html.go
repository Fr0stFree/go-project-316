package crawler

import (
	"code/internal/common/types"
	"errors"
	"io"

	"golang.org/x/net/html"
)

type parsedPage struct {
	Links []types.URL
}

func parseHTMLPage(r io.Reader) (parsedPage, error) {
	page := parsedPage{Links: make([]types.URL, 0)}

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
					page.Links = append(page.Links, types.URL(attr.Val))

					break
				}
			}
		}
	}
}
