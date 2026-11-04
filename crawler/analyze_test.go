package crawler_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"code/crawler"
)

type mockHTTPTransport struct {
	mock.Mock
}

func (m *mockHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	path := request.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	args := m.Called(request.Method, request.URL.Host, path)

	response := args.Get(0).(*http.Response)
	if response != nil {
		response.Request = request
	}

	return response, args.Error(1)
}

type analyzeMocks struct {
	client    *http.Client
	transport *mockHTTPTransport
}

func newAnalyzeMocks(t *testing.T) analyzeMocks {
	t.Helper()

	transport := new(mockHTTPTransport)

	t.Cleanup(func() {
		transport.AssertExpectations(t)
	})

	return analyzeMocks{
		client:    &http.Client{Transport: transport},
		transport: transport,
	}
}

func newHTTPResponse(statusCode int, body string, size int64, contentType string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	return &http.Response{
		StatusCode:    statusCode,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: size,
		Header:        header,
	}
}

type fixtureResponse struct {
	method      string
	path        string
	statusCode  int
	bodyFile    string
	size        int64
	contentType string
	errorText   string
}

type analyzeFixture struct {
	name      string
	depth     int
	responses []fixtureResponse
}

func TestAnalyze(t *testing.T) {
	t.Run("should match report fixtures", func(t *testing.T) {
		const rootURL = "http://example.test"

		fixedTime := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		fixtures := []analyzeFixture{
			{
				name:  "simple",
				depth: 1,
				responses: []fixtureResponse{
					{method: http.MethodGet, path: "/", statusCode: http.StatusOK, bodyFile: "index.html", contentType: "text/html"},
				},
			},
			{
				name:  "linked",
				depth: 2,
				responses: []fixtureResponse{
					{method: http.MethodGet, path: "/", statusCode: http.StatusOK, bodyFile: "index.html", contentType: "text/html"},
					{method: http.MethodGet, path: "/about", statusCode: http.StatusOK, bodyFile: "about.html", contentType: "text/html"},
					{method: http.MethodGet, path: "/missing", statusCode: http.StatusNotFound},
					{method: http.MethodHead, path: "/logo.svg", statusCode: http.StatusOK, size: 345, contentType: "image/svg+xml"},
				},
			},
			{
				name:  "network-error",
				depth: 1,
				responses: []fixtureResponse{
					{method: http.MethodGet, path: "/", errorText: "network unavailable"},
				},
			},
		}

		for _, fixture := range fixtures {
			t.Run("should match "+fixture.name+" fixture", func(t *testing.T) {
				mocks := newAnalyzeMocks(t)

				for _, response := range fixture.responses {
					if response.errorText != "" {
						mocks.transport.
							On("RoundTrip", response.method, "example.test", response.path).
							Return((*http.Response)(nil), errors.New(response.errorText)).
							Once()

						continue
					}

					body := ""

					if response.bodyFile != "" {
						content, err := os.ReadFile(filepath.Join("testdata", fixture.name, response.bodyFile))
						require.NoError(t, err)

						body = string(content)
					}

					size := int64(len(body))
					if response.size > 0 {
						size = response.size
					}

					mocks.transport.
						On("RoundTrip", response.method, "example.test", response.path).
						Return(newHTTPResponse(response.statusCode, body, size, response.contentType), nil).
						Once()
				}

				data, err := crawler.Analyze(t.Context(), crawler.Options{
					URL:        rootURL,
					Depth:      fixture.depth,
					Timeout:    time.Second,
					HTTPClient: mocks.client,
					Now:        func() time.Time { return fixedTime },
				})
				require.NoError(t, err)

				expectedData, err := os.ReadFile(filepath.Join("testdata", fixture.name, "expected.json"))
				require.NoError(t, err)

				var expected map[string]any
				require.NoError(t, json.Unmarshal(expectedData, &expected))

				var actual map[string]any
				require.NoError(t, json.Unmarshal(data, &actual))

				assert.Equal(t, expected, actual)
			})
		}
	})

	t.Run("should resolve links and assets against the final document URL", func(t *testing.T) {
		mocks := newAnalyzeMocks(t)
		redirect := newHTTPResponse(http.StatusFound, "", 0, "")
		redirect.Header.Set("Location", "/new/page")
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/old/page").Return(redirect, nil).Once()
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/new/page").
			Return(newHTTPResponse(http.StatusOK, `<a href="missing">Missing</a><img src="logo.png">`, 0, "text/html"), nil).
			Once()
		mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/new/missing").
			Return(newHTTPResponse(http.StatusNotFound, "", 0, ""), nil).
			Once()
		mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/new/logo.png").
			Return(newHTTPResponse(http.StatusOK, "", 42, "image/png"), nil).
			Once()

		data, err := crawler.Analyze(t.Context(), crawler.Options{
			URL:        "http://example.test/old/page",
			Depth:      1,
			HTTPClient: mocks.client,
		})
		require.NoError(t, err)

		var report crawler.Report
		require.NoError(t, json.Unmarshal(data, &report))
		require.Len(t, report.Pages, 1)
		require.Len(t, report.Pages[0].BrokenLinks, 1)
		require.Len(t, report.Pages[0].Assets, 1)
		assert.Equal(t, "http://example.test/old/page", report.Pages[0].URL)
		assert.Equal(t, "http://example.test/new/missing", report.Pages[0].BrokenLinks[0].URL)
		assert.Equal(t, http.StatusNotFound, report.Pages[0].BrokenLinks[0].StatusCode)
		assert.Equal(t, "http://example.test/new/logo.png", report.Pages[0].Assets[0].URL)
		assert.Equal(t, 42, report.Pages[0].Assets[0].SizeBytes)
	})

	t.Run("should check one resource and report both link spellings", func(t *testing.T) {
		mocks := newAnalyzeMocks(t)
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
			Return(newHTTPResponse(http.StatusOK,
				`<a href="http://other.test">First</a><a href="http://other.test/">Second</a>`,
				0, "text/html"), nil).
			Once()
		mocks.transport.On("RoundTrip", http.MethodHead, "other.test", "/").
			Return(newHTTPResponse(http.StatusNotFound, "", 0, ""), nil).
			Once()

		data, err := crawler.Analyze(t.Context(), crawler.Options{
			URL:        "http://example.test",
			Depth:      1,
			HTTPClient: mocks.client,
		})
		require.NoError(t, err)

		var report crawler.Report
		require.NoError(t, json.Unmarshal(data, &report))
		require.Len(t, report.Pages, 1)
		assert.Equal(t, "http://example.test", report.Pages[0].URL)
		require.Len(t, report.Pages[0].BrokenLinks, 2)
		assert.Equal(t, "http://other.test", report.Pages[0].BrokenLinks[0].URL)
		assert.Equal(t, "http://other.test/", report.Pages[0].BrokenLinks[1].URL)
		assert.Equal(t, http.StatusNotFound, report.Pages[0].BrokenLinks[0].StatusCode)
		assert.Equal(t, http.StatusNotFound, report.Pages[0].BrokenLinks[1].StatusCode)
	})

	t.Run("should report SEO fields", func(t *testing.T) {
		tests := []struct {
			name           string
			html           string
			hasTitle       bool
			title          string
			hasDescription bool
			description    string
			hasH1          bool
		}{
			{
				name: "should report missing SEO tags",
				html: `<html><body>Plain page</body></html>`,
			},
			{
				name:           "should decode HTML entities in SEO text",
				html:           `<html><head><title>Fish &amp; Chips</title><meta name="description" content="Bread &amp; butter"></head><body><h1>Menu</h1></body></html>`,
				hasTitle:       true,
				title:          "Fish & Chips",
				hasDescription: true,
				description:    "Bread & butter",
				hasH1:          true,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				mocks := newAnalyzeMocks(t)
				mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
					Return(newHTTPResponse(http.StatusOK, test.html, int64(len(test.html)), "text/html"), nil).
					Once()

				data, err := crawler.Analyze(t.Context(), crawler.Options{
					URL:        "http://example.test",
					Depth:      1,
					HTTPClient: mocks.client,
				})
				require.NoError(t, err)

				var report crawler.Report
				require.NoError(t, json.Unmarshal(data, &report))
				require.Len(t, report.Pages, 1)

				seo := report.Pages[0].SEO
				assert.Equal(t, test.hasTitle, seo.HasTitle)
				assert.Equal(t, test.title, seo.Title)
				assert.Equal(t, test.hasDescription, seo.HasDescription)
				assert.Equal(t, test.description, seo.Description)
				assert.Equal(t, test.hasH1, seo.HasH1)
			})
		}
	})
}
