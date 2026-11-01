package crawler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"code/internal/crawler/htmlparser"

	"github.com/stretchr/testify/require"
)

func newNilCrawler(t *testing.T, opts ...Option) *Crawler {
	t.Helper()

	crawler, err := New(nil, opts...)
	require.NoError(t, err)

	return crawler
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	return parsed
}

func TestNew(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		crawler, err := New(nil)

		require.NoError(t, err)
		require.NotNil(t, crawler)
		require.NotNil(t, crawler.fetcher)
		require.NotNil(t, crawler.now)
		require.Equal(t, defaultMaxDepth, crawler.maxDepth)
		require.Equal(t, defaultPoolSize, crawler.poolSize)
	})

	t.Run("custom configuration", func(t *testing.T) {
		now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

		crawler, err := New(
			nil,
			WithClock(func() time.Time { return now }),
			WithPoolSize(5),
			WithMaxDepth(2),
			WithUserAgent("TestCrawler/1.0"),
			WithTimeout(2*time.Second),
			WithRetries(3),
		)

		require.NoError(t, err)
		require.Equal(t, 5, crawler.poolSize)
		require.Equal(t, 2, crawler.maxDepth)
		require.Equal(t, now, crawler.now())
	})

	t.Run("invalid configuration", func(t *testing.T) {
		type subTest struct {
			name string
			opts []Option
		}

		tests := []subTest{
			{
				name: "negative pool size",
				opts: []Option{WithPoolSize(-1)},
			},
			{
				name: "negative max depth",
				opts: []Option{WithMaxDepth(-1)},
			},
			{
				name: "invalid user agent",
				opts: []Option{WithUserAgent("   ")},
			},
			{
				name: "negative timeout",
				opts: []Option{WithTimeout(-time.Second)},
			},
			{
				name: "negative retries",
				opts: []Option{WithRetries(-1)},
			},
			{
				name: "negative RPS",
				opts: []Option{WithRPS(context.Background(), -1)},
			},
			{
				name: "negative delay",
				opts: []Option{WithDelay(-time.Second)},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				crawler, err := New(nil, test.opts...)

				require.Error(t, err)
				require.Nil(t, crawler)
			})
		}
	})
}

func TestCrawler_Run(t *testing.T) {
	type subTest struct {
		name string
		url  string
	}

	tests := []subTest{
		{name: "unsupported scheme", url: "ftp://example.com"},
		{name: "missing hostname", url: "https:///example.com"},
		{name: "invalid URL", url: "https://%"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			crawler := newNilCrawler(t)

			results, err := crawler.Run(context.Background(), test.url)

			require.Error(t, err)
			require.Nil(t, results)
		})
	}
}

func TestCrawler_HandleAssetResult(t *testing.T) {
	crawler := newNilCrawler(t)
	root := mustParseURL(t, "https://example.com")
	state := newCrawlerState(root)
	assetURL := mustParseURL(t, "https://example.com/image.png")
	result := assetTaskResult{
		url:        assetURL,
		httpStatus: http.StatusOK,
		sizeBytes:  1024,
	}

	crawler.handleAssetResult(state, result)

	require.Equal(t, result, state.assets[assetURL.String()])
	require.Len(t, state.assets, 1)
}

func TestCrawler_HandleLinkResult(t *testing.T) {
	crawler := newNilCrawler(t)
	root := mustParseURL(t, "https://example.com")
	state := newCrawlerState(root)
	linkURL := mustParseURL(t, "https://example.com/about")
	result := linkTaskResult{url: linkURL, httpStatus: http.StatusOK}

	crawler.handleLinkResult(state, result)

	require.Equal(t, result, state.links[linkURL.String()])
	require.Len(t, state.links, 1)
}

func TestCrawler_HandlePageResult(t *testing.T) {
	root := mustParseURL(t, "https://example.com")
	internal := mustParseURL(t, "https://example.com/about")
	external := mustParseURL(t, "https://other.com")
	asset := mustParseURL(t, "https://example.com/image.png")

	t.Run("no HTML", func(t *testing.T) {
		crawler := newNilCrawler(t)
		state := newCrawlerState(root)
		result := pageTaskResult{
			url:        root,
			httpStatus: http.StatusInternalServerError,
			status:     statusError,
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, result, state.pages[root.String()])
		require.Zero(t, state.pending.Len())
	})

	t.Run("discover assets and internal links", func(t *testing.T) {
		crawler := newNilCrawler(t)
		state := newCrawlerState(root)
		result := pageTaskResult{
			url: root,
			html: &htmlparser.ParsedHTML{
				Assets: []htmlparser.Asset{
					{Type: htmlparser.AssetImage, URL: asset},
				},
				Links: []*url.URL{internal},
			},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, result, state.pages[root.String()])
		require.Equal(t, 2, state.pending.Len())

		first := state.pending.Front().Value.(task)
		second := state.pending.Back().Value.(task)

		require.Equal(t, assetTask, first.kind)
		require.Equal(t, asset, first.url)
		require.Equal(t, pageTask, second.kind)
		require.Equal(t, internal, second.url)
		require.Equal(t, 1, second.depth)
	})

	t.Run("external link", func(t *testing.T) {
		crawler := newNilCrawler(t)
		state := newCrawlerState(root)
		result := pageTaskResult{
			url:  root,
			html: &htmlparser.ParsedHTML{Links: []*url.URL{external}},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, 1, state.pending.Len())

		scheduled := state.pending.Front().Value.(task)

		require.Equal(t, linkCheckTask, scheduled.kind)
		require.Equal(t, external, scheduled.url)
		require.Equal(t, 1, scheduled.depth)
	})

	t.Run("link at depth boundary", func(t *testing.T) {
		crawler := newNilCrawler(t, WithMaxDepth(2))
		state := newCrawlerState(root)
		result := pageTaskResult{
			url:   root,
			depth: 1,
			html: &htmlparser.ParsedHTML{
				Links: []*url.URL{internal},
			},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, 1, state.pending.Len())

		scheduled := state.pending.Front().Value.(task)

		require.Equal(t, linkCheckTask, scheduled.kind)
		require.Equal(t, 2, scheduled.depth)
	})

	t.Run("maximum depth reached", func(t *testing.T) {
		crawler := newNilCrawler(t, WithMaxDepth(2))
		state := newCrawlerState(root)
		result := pageTaskResult{
			url:   root,
			depth: 2,
			html: &htmlparser.ParsedHTML{
				Links: []*url.URL{internal},
				Assets: []htmlparser.Asset{
					{Type: htmlparser.AssetImage, URL: asset},
				},
			},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, result, state.pages[root.String()])
		require.Zero(t, state.pending.Len())
	})

	t.Run("duplicate links and assets", func(t *testing.T) {
		crawler := newNilCrawler(t)
		state := newCrawlerState(root)
		result := pageTaskResult{
			url: root,
			html: &htmlparser.ParsedHTML{
				Links: []*url.URL{internal, internal},
				Assets: []htmlparser.Asset{
					{Type: htmlparser.AssetImage, URL: asset},
					{Type: htmlparser.AssetImage, URL: asset},
				},
			},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, 2, state.pending.Len())
	})

	t.Run("zero max depth", func(t *testing.T) {
		crawler := newNilCrawler(t, WithMaxDepth(0))
		state := newCrawlerState(root)
		result := pageTaskResult{
			url: root,
			html: &htmlparser.ParsedHTML{
				Links: []*url.URL{internal},
			},
		}

		crawler.handlePageResult(state, result)

		require.Equal(t, 1, state.pending.Len())

		scheduled := state.pending.Front().Value.(task)

		require.Equal(t, linkCheckTask, scheduled.kind)
		require.Equal(t, 1, scheduled.depth)
	})
}

func TestCrawler_MergeResults(t *testing.T) {
	crawler := newNilCrawler(t)
	root := mustParseURL(t, "https://example.com/")
	link := mustParseURL(t, "https://other.com/missing")
	asset := mustParseURL(t, "https://example.com/logo.png")
	state := newCrawlerState(root)
	state.pages[root.String()] = pageTaskResult{
		url:        root,
		httpStatus: http.StatusOK,
		status:     statusOK,
		html: &htmlparser.ParsedHTML{
			Assets: []htmlparser.Asset{{Type: htmlparser.AssetImage, URL: asset}},
		},
	}
	state.assets[asset.String()] = assetTaskResult{
		url:        asset,
		httpStatus: http.StatusOK,
		sizeBytes:  128,
	}
	state.links[link.String()] = linkTaskResult{
		url:        link,
		depth:      1,
		httpStatus: http.StatusNotFound,
		err:        errors.New("not found"),
	}

	results := crawler.mergeResults(state)

	require.Len(t, results, 2)
	require.Equal(t, []AssetResult{{
		URL:        asset.String(),
		Type:       htmlparser.AssetImage,
		StatusCode: http.StatusOK,
		SizeBytes:  128,
	}}, results[root.String()].Assets)
	require.True(t, results[link.String()].CheckedOnly)
	require.Equal(t, statusError, results[link.String()].Status)
	require.Equal(t, http.StatusNotFound, results[link.String()].HTTPStatus)
}
