package crawler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
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

	t.Run("should report asset request outcomes", func(t *testing.T) {
		tests := []struct {
			name       string
			headStatus int
			headError  string
			getBody    string
			wantStatus int
			wantSize   int
			wantError  string
		}{
			{
				name:       "should use GET when HEAD is not supported",
				headStatus: http.StatusMethodNotAllowed,
				getBody:    "hello",
				wantStatus: http.StatusOK,
				wantSize:   5,
			},
			{
				name:       "should report HTTP error from HEAD",
				headStatus: http.StatusNotFound,
				wantStatus: http.StatusNotFound,
				wantError:  "unexpected HTTP status: 404",
			},
			{
				name:      "should report network error from HEAD",
				headError: "network unavailable",
				wantError: "network unavailable",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				mocks := newAnalyzeMocks(t)
				mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
					Return(newHTTPResponse(http.StatusOK, `<img src="/logo.png">`, 0, "text/html"), nil).
					Once()

				if test.headError != "" {
					mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/logo.png").
						Return((*http.Response)(nil), errors.New(test.headError)).
						Once()
				} else {
					mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/logo.png").
						Return(newHTTPResponse(test.headStatus, "", 0, ""), nil).
						Once()
				}

				if test.getBody != "" {
					mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/logo.png").
						Return(newHTTPResponse(http.StatusOK, test.getBody, int64(len(test.getBody)), "image/png"), nil).
						Once()
				}

				data, err := crawler.Analyze(t.Context(), crawler.Options{
					URL:        "http://example.test",
					Depth:      1,
					HTTPClient: mocks.client,
				})
				require.NoError(t, err)

				var report crawler.Report
				require.NoError(t, json.Unmarshal(data, &report))
				require.Len(t, report.Pages, 1)
				require.Len(t, report.Pages[0].Assets, 1)

				asset := report.Pages[0].Assets[0]
				assert.Equal(t, "http://example.test/logo.png", asset.URL)
				assert.Equal(t, test.wantStatus, asset.StatusCode)
				assert.Equal(t, test.wantSize, asset.SizeBytes)

				if test.wantError == "" {
					assert.Empty(t, asset.Error)
				} else {
					assert.Contains(t, asset.Error, test.wantError)
				}
			})
		}
	})

	t.Run("should reuse one asset result across pages", func(t *testing.T) {
		mocks := newAnalyzeMocks(t)
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
			Return(newHTTPResponse(http.StatusOK, `<a href="/about">About</a><img src="/logo.png">`, 0, "text/html"), nil).
			Once()
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/about").
			Return(newHTTPResponse(http.StatusOK, `<img src="/logo.png">`, 0, "text/html"), nil).
			Once()
		mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/logo.png").
			Return(newHTTPResponse(http.StatusOK, "", 345, "image/png"), nil).
			Once()

		data, err := crawler.Analyze(t.Context(), crawler.Options{
			URL:        "http://example.test",
			Depth:      2,
			HTTPClient: mocks.client,
		})
		require.NoError(t, err)

		var report crawler.Report
		require.NoError(t, json.Unmarshal(data, &report))
		require.Len(t, report.Pages, 2)
		require.Len(t, report.Pages[0].Assets, 1)
		require.Len(t, report.Pages[1].Assets, 1)
		assert.Equal(t, report.Pages[0].Assets[0], report.Pages[1].Assets[0])
		assert.Equal(t, "http://example.test/logo.png", report.Pages[0].Assets[0].URL)
		assert.Equal(t, http.StatusOK, report.Pages[0].Assets[0].StatusCode)
		assert.Equal(t, 345, report.Pages[0].Assets[0].SizeBytes)
		assert.Empty(t, report.Pages[0].Assets[0].Error)
	})

	t.Run("should report HTTP 500 in JSON", func(t *testing.T) {
		mocks := newAnalyzeMocks(t)
		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
			Return(newHTTPResponse(http.StatusInternalServerError, "", 0, ""), nil).
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
		assert.Equal(t, http.StatusInternalServerError, report.Pages[0].HTTPStatus)
		assert.Equal(t, "error", report.Pages[0].Status)
		assert.Empty(t, report.Pages[0].Error)
		assert.Empty(t, report.Pages[0].BrokenLinks)
		assert.Empty(t, report.Pages[0].Assets)
	})

	t.Run("should report request timeout in JSON", func(t *testing.T) {
		client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			<-request.Context().Done()

			return nil, request.Context().Err()
		})}

		data, err := crawler.Analyze(t.Context(), crawler.Options{
			URL:        "http://example.test",
			Depth:      1,
			Timeout:    20 * time.Millisecond,
			HTTPClient: client,
		})
		require.NoError(t, err)

		var report crawler.Report
		require.NoError(t, json.Unmarshal(data, &report))
		require.Len(t, report.Pages, 1)
		assert.Zero(t, report.Pages[0].HTTPStatus)
		assert.Equal(t, "error", report.Pages[0].Status)
		assert.Contains(t, report.Pages[0].Error, context.DeadlineExceeded.Error())
	})

	t.Run("should return completed pages after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		started := make(chan struct{})

		release := make(chan struct{})
		defer close(release)

		client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "":
				response := newHTTPResponse(http.StatusOK, `<title>Ready</title><a href="/slow">Slow</a>`, 0, "text/html")
				response.Request = request

				return response, nil
			case "/slow":
				close(started)
				<-request.Context().Done()
				<-release

				return nil, request.Context().Err()
			default:
				return nil, errors.New("unexpected URL")
			}
		})}

		type analyzeResult struct {
			data []byte
			err  error
		}

		result := make(chan analyzeResult, 1)

		go func() {
			data, err := crawler.Analyze(ctx, crawler.Options{
				URL:         "http://example.test",
				Depth:       2,
				Concurrency: 1,
				HTTPClient:  client,
			})
			result <- analyzeResult{data: data, err: err}
		}()

		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("the second request did not start")
		}

		cancel()

		select {
		case outcome := <-result:
			require.ErrorIs(t, context.Canceled, outcome.err)

			var report crawler.Report
			require.NoError(t, json.Unmarshal(outcome.data, &report))
			require.Len(t, report.Pages, 1)
			assert.Equal(t, "http://example.test", report.Pages[0].URL)
			assert.Equal(t, http.StatusOK, report.Pages[0].HTTPStatus)
			assert.Equal(t, "Ready", report.Pages[0].SEO.Title)
		case <-time.After(5 * time.Second):
			t.Fatal("Analyze did not return after cancellation")
		}
	})

	t.Run("should enforce RPS limit across workers", func(t *testing.T) {
		var (
			mu           sync.Mutex
			requestTimes []time.Time
		)

		client := &http.Client{
			Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				mu.Lock()

				requestTimes = append(requestTimes, time.Now())
				mu.Unlock()

				if request.URL.Path == "" {
					return newHTTPResponse(
						http.StatusOK,
						`<img src="/a.png"><img src="/b.png">`,
						0,
						"text/html",
					), nil
				}

				return newHTTPResponse(http.StatusOK, "", 100, "image/png"), nil
			}),
		}

		_, err := crawler.Analyze(t.Context(), crawler.Options{
			URL:         "http://example.test",
			Depth:       1,
			RPS:         2,
			Concurrency: 3,
			HTTPClient:  client,
		})
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()

		require.Len(t, requestTimes, 3)
		assert.GreaterOrEqual(t, requestTimes[2].Sub(requestTimes[0]), 900*time.Millisecond)
	})

	t.Run("should check links at depth boundary without crawling them", func(t *testing.T) {
		mocks := newAnalyzeMocks(t)

		mocks.transport.On("RoundTrip", http.MethodGet, "example.test", "/").
			Return(newHTTPResponse(
				http.StatusOK,
				`<a href="/missing">Missing</a>`,
				0,
				"text/html",
			), nil).
			Once()

		mocks.transport.On("RoundTrip", http.MethodHead, "example.test", "/missing").
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

		require.Len(t, report.Pages[0].BrokenLinks, 1)
		assert.Equal(t, "http://example.test/missing", report.Pages[0].BrokenLinks[0].URL)
		assert.Equal(t, http.StatusNotFound, report.Pages[0].BrokenLinks[0].StatusCode)
	})
}
