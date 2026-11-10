package report_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"code/internal/crawler"
	"code/internal/crawler/htmlparser"
	"code/internal/report"

	"github.com/stretchr/testify/require"
)

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	return parsed
}

func TestBuild(t *testing.T) {
	root := mustParseURL(t, "https://example.com/")
	broken := mustParseURL(t, "https://example.com/missing")
	checked := mustParseURL(t, "https://other.com/")
	logo := mustParseURL(t, "https://example.com/logo.png")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	title := "Example"
	results := map[string]crawler.Result{
		root.String(): {
			URL:          root,
			HTTPStatus:   http.StatusOK,
			Status:       "ok",
			DiscoveredAt: now,
			HTML: &htmlparser.ParsedHTML{
				Title: &title,
				Links: []*url.URL{broken},
			},
			Assets: []crawler.AssetResult{{
				URL:        logo,
				Type:       htmlparser.AssetImage,
				StatusCode: http.StatusOK,
				SizeBytes:  128,
			}},
		},
		broken.String(): {
			URL:          broken,
			Depth:        1,
			HTTPStatus:   http.StatusNotFound,
			Status:       "error",
			DiscoveredAt: now,
			Err:          errors.New("page not found"),
		},
		checked.String(): {
			URL:         checked,
			CheckedOnly: true,
		},
	}

	actual := report.Build(root.String(), 2, results, now)

	require.Equal(t, &report.Report{
		RootURL:     root.String(),
		Depth:       2,
		GeneratedAt: now,
		Pages: []report.Page{
			{
				URL:          root.String(),
				HTTPStatus:   http.StatusOK,
				Status:       "ok",
				DiscoveredAt: now,
				SEO:          report.SEO{HasTitle: true, Title: title},
				BrokenLinks: []report.NodeLink{{
					URL:        broken.String(),
					StatusCode: http.StatusNotFound,
					Error:      "page not found",
				}},
				Assets: []report.Asset{{
					URL:        "https://example.com/logo.png",
					Type:       "image",
					StatusCode: http.StatusOK,
					SizeBytes:  128,
				}},
			},
			{
				URL:          broken.String(),
				Depth:        1,
				HTTPStatus:   http.StatusNotFound,
				Status:       "error",
				Error:        "page not found",
				DiscoveredAt: now,
			},
		},
	}, actual)
}
