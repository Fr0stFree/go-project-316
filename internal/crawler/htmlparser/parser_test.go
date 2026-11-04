package htmlparser_test

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"code/internal/crawler/htmlparser"

	"github.com/stretchr/testify/require"
)

func prepareHTMLDoc(t *testing.T, name string) *os.File {
	t.Helper()

	file, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	return file
}

func prepareURL(t *testing.T) *url.URL {
	t.Helper()

	pageURL, err := url.Parse("https://example.com/pages/home")
	require.NoError(t, err)

	return pageURL
}

func TestParse(t *testing.T) {
	t.Run("should parse links assets and SEO", func(t *testing.T) {
		doc := prepareHTMLDoc(t, "example.html")
		pageURL := prepareURL(t)

		result, err := htmlparser.Parse(doc, pageURL)
		require.NoError(t, err)
		require.Equal(t, "Example Website", *result.Title)
		require.Equal(t, "A sample website for testing", *result.Description)
		require.Equal(t, "Welcome to Example", *result.H1)

		links := make([]string, 0, len(result.Links))
		for _, link := range result.Links {
			links = append(links, link.String())
		}

		require.Equal(t, []string{
			"https://example.com/about",
			"https://other.com/contact",
			"https://example.com/blog",
		}, links)

		assets := make([]htmlparser.Asset, 0, len(result.Assets))
		assets = append(assets, result.Assets...)

		require.Len(t, assets, 6)

		require.Equal(t, htmlparser.AssetImage, assets[0].Type)
		require.Equal(t, "https://example.com/images/logo.png", assets[0].URL.String())

		require.Equal(t, htmlparser.AssetImage, assets[1].Type)
		require.Equal(t, "https://cdn.example.com/banner.jpg", assets[1].URL.String())

		require.Equal(t, htmlparser.AssetScript, assets[2].Type)
		require.Equal(t, "https://example.com/js/app.js", assets[2].URL.String())

		require.Equal(t, htmlparser.AssetScript, assets[3].Type)
		require.Equal(t, "https://cdn.example.com/lib.js", assets[3].URL.String())

		require.Equal(t, htmlparser.AssetStyle, assets[4].Type)
		require.Equal(t, "https://example.com/css/main.css", assets[4].URL.String())

		require.Equal(t, htmlparser.AssetStyle, assets[5].Type)
		require.Equal(t, "https://cdn.example.com/theme.css", assets[5].URL.String())
	})

	t.Run("should resolve relative URLs against the first base and recognize stylesheet case", func(t *testing.T) {
		pageURL := prepareURL(t)
		doc := prepareHTMLDoc(t, "base-and-styles.html")

		result, err := htmlparser.Parse(doc, pageURL)
		require.NoError(t, err)
		require.Len(t, result.Links, 2)
		require.Equal(t, "https://example.com/assets/next", result.Links[0].String())
		require.Equal(t, "https://example.com/fixed", result.Links[1].String())
		require.Len(t, result.Assets, 4)
		require.Equal(t, "https://example.com/assets/logo.png", result.Assets[0].URL.String())
		require.Equal(t, "https://example.com/assets/app.js", result.Assets[1].URL.String())
		require.Equal(t, htmlparser.AssetStyle, result.Assets[2].Type)
		require.Equal(t, "https://example.com/assets/main.css", result.Assets[2].URL.String())
		require.Equal(t, htmlparser.AssetStyle, result.Assets[3].Type)
		require.Equal(t, "https://example.com/assets/other.css", result.Assets[3].URL.String())
	})
}
