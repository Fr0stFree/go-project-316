package crawler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newTestCrawler(t *testing.T, transport roundTripperFunc) *Crawler {
	t.Helper()

	crawler, err := New(&http.Client{Transport: transport})
	require.NoError(t, err)

	return crawler
}

func testResponse(status int, length int64, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		ContentLength: length,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
	}
}

func TestCrawler_ProcessLinkCheck(t *testing.T) {
	methods := make([]string, 0, 2)
	crawler := newTestCrawler(t, func(request *http.Request) (*http.Response, error) {
		methods = append(methods, request.Method)

		if request.Method == http.MethodHead {
			return testResponse(http.StatusMethodNotAllowed, 0, ""), nil
		}

		return testResponse(http.StatusOK, 0, ""), nil
	})
	link := mustParseURL(t, "https://example.com/about")

	result := crawler.processLinkCheck(context.Background(), link, 1)

	require.NoError(t, result.err)
	require.Equal(t, link, result.url)
	require.Equal(t, 1, result.depth)
	require.Equal(t, http.StatusOK, result.httpStatus)
	require.Equal(t, []string{http.MethodHead, http.MethodGet}, methods)
}

func TestCrawler_ProcessWebPage(t *testing.T) {
	crawler := newTestCrawler(t, func(_ *http.Request) (*http.Response, error) {
		response := testResponse(http.StatusOK, -1, "<rss><channel><title>Crawler Blog</title></channel></rss>")
		response.Header.Set("Content-Type", "application/xml")

		return response, nil
	})
	page := mustParseURL(t, "https://example.com/feed.xml")

	result := crawler.processWebPage(context.Background(), page, 1)

	require.NoError(t, result.err)
	require.NotNil(t, result.html)
	require.Equal(t, "Crawler Blog", *result.html.Title)
}

func TestCrawler_ProcessAsset(t *testing.T) {
	t.Run("size from HEAD", func(t *testing.T) {
		methods := make([]string, 0, 1)
		crawler := newTestCrawler(t, func(request *http.Request) (*http.Response, error) {
			methods = append(methods, request.Method)

			return testResponse(http.StatusOK, 5, ""), nil
		})
		asset := mustParseURL(t, "https://example.com/logo.png")

		result := crawler.processAsset(context.Background(), asset)

		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.httpStatus)
		require.Equal(t, 5, result.sizeBytes)
		require.Equal(t, []string{http.MethodHead}, methods)
	})

	t.Run("size from GET when HEAD has no length", func(t *testing.T) {
		methods := make([]string, 0, 2)
		crawler := newTestCrawler(t, func(request *http.Request) (*http.Response, error) {
			methods = append(methods, request.Method)

			if request.Method == http.MethodHead {
				return testResponse(http.StatusOK, -1, ""), nil
			}

			return testResponse(http.StatusOK, 5, "hello"), nil
		})
		asset := mustParseURL(t, "https://example.com/logo.png")

		result := crawler.processAsset(context.Background(), asset)

		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.httpStatus)
		require.Equal(t, 5, result.sizeBytes)
		require.Equal(t, []string{http.MethodHead, http.MethodGet}, methods)
	})
}
