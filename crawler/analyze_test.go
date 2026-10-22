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

	args := m.Called(request.Method, path)

	return args.Get(0).(*http.Response), args.Error(1)
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

func newHTTPResponse(statusCode int, body string, size int64) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: size,
		Header:        make(http.Header),
	}
}

type fixtureResponse struct {
	method     string
	path       string
	statusCode int
	bodyFile   string
	size       int64
	errorText  string
}

type analyzeFixture struct {
	name      string
	depth     int
	responses []fixtureResponse
}

func TestAnalyze(t *testing.T) {
	const rootURL = "http://example.test"

	fixedTime := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	fixtures := []analyzeFixture{
		{
			name:  "simple",
			depth: 1,
			responses: []fixtureResponse{
				{method: http.MethodGet, path: "/", statusCode: http.StatusOK, bodyFile: "index.html"},
			},
		},
		{
			name:  "linked",
			depth: 2,
			responses: []fixtureResponse{
				{method: http.MethodGet, path: "/", statusCode: http.StatusOK, bodyFile: "index.html"},
				{method: http.MethodGet, path: "/about", statusCode: http.StatusOK, bodyFile: "about.html"},
				{method: http.MethodGet, path: "/missing", statusCode: http.StatusNotFound},
				{method: http.MethodHead, path: "/logo.svg", statusCode: http.StatusOK, size: 3},
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
						On("RoundTrip", response.method, response.path).
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
					On("RoundTrip", response.method, response.path).
					Return(newHTTPResponse(response.statusCode, body, size), nil).
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
}
