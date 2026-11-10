package httpfetcher_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"code/internal/crawler/httpfetcher"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestFetcher(t *testing.T, transport roundTripperFunc) *httpfetcher.Fetcher {
	t.Helper()

	fetcher := httpfetcher.New(&http.Client{
		Transport: transport,
	})

	require.NoError(t, fetcher.SetRetryDelay(0))

	return fetcher
}

func testResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("test response")),
	}
}

func prepareURL(t *testing.T) *url.URL {
	t.Helper()

	parsed, err := url.Parse("https://example.com")
	require.NoError(t, err)

	return parsed
}
func TestFetch(t *testing.T) {
	t.Run("simple fetch", func(t *testing.T) {
		url := prepareURL(t)
		fetcher := newTestFetcher(t, func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "GoCrawler/1.0", req.Header.Get("User-Agent"))
			require.Equal(t, "example.com", req.URL.Host)

			return testResponse(http.StatusOK), nil
		})

		response, err := fetcher.Fetch(context.Background(), url, http.MethodGet)

		require.NoError(t, err)
		require.NotNil(t, response)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("fetch with retries", func(t *testing.T) {
		type subTest struct {
			name                  string
			statuses              []int
			retries               int
			expectedStatus        int
			expectedRequestsCount int
		}

		tests := []subTest{
			{
				name:                  "successful response",
				statuses:              []int{200},
				retries:               2,
				expectedStatus:        200,
				expectedRequestsCount: 1,
			},
			{
				name:                  "retry server error",
				statuses:              []int{500, 200},
				retries:               1,
				expectedStatus:        200,
				expectedRequestsCount: 2,
			},
			{
				name:                  "retry rate limiting",
				statuses:              []int{429, 200},
				retries:               1,
				expectedStatus:        200,
				expectedRequestsCount: 2,
			},
			{
				name:                  "do not retry client error",
				statuses:              []int{404},
				retries:               2,
				expectedStatus:        404,
				expectedRequestsCount: 1,
			},
			{
				name:                  "retry limit reached",
				statuses:              []int{503, 503, 200},
				retries:               1,
				expectedStatus:        503,
				expectedRequestsCount: 2,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				requestsMade := 0
				url := prepareURL(t)
				fetcher := newTestFetcher(t, func(_ *http.Request) (*http.Response, error) {
					status := test.statuses[requestsMade]
					requestsMade++

					return testResponse(status), nil
				})
				require.NoError(t, fetcher.SetRetries(test.retries))

				response, err := fetcher.Fetch(context.Background(), url, http.MethodGet)

				require.NoError(t, err)
				require.NotNil(t, response)
				t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
				require.Equal(t, test.expectedStatus, response.StatusCode)
				require.Equal(t, test.expectedRequestsCount, requestsMade)
			})
		}
	})

	t.Run("fetch with error", func(t *testing.T) {
		requestsMade := 0
		url := prepareURL(t)
		fetcher := newTestFetcher(t, func(_ *http.Request) (*http.Response, error) {
			requestsMade++
			if requestsMade == 1 {
				return nil, syscall.ECONNRESET
			}

			return testResponse(http.StatusOK), nil
		})
		require.NoError(t, fetcher.SetRetries(1))

		response, err := fetcher.Fetch(context.Background(), url, http.MethodGet)

		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, 2, requestsMade)
	})
}

type interruptedBody struct{}

func (interruptedBody) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (interruptedBody) Close() error {
	return nil
}

func TestFetchSize(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		requestsMade := 0

		fetcher := newTestFetcher(t, func(req *http.Request) (*http.Response, error) {
			requestsMade++

			require.Equal(t, http.MethodGet, req.Method)

			return testResponse(http.StatusOK), nil
		})

		status, size, err := fetcher.FetchSize(context.Background(), prepareURL(t))

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, int64(len("test response")), size)
		require.Equal(t, 1, requestsMade)
	})

	t.Run("retry interrupted body", func(t *testing.T) {
		requestsMade := 0

		fetcher := newTestFetcher(t, func(_ *http.Request) (*http.Response, error) {
			requestsMade++

			if requestsMade == 1 {
				response := testResponse(http.StatusOK)
				response.Body = interruptedBody{}

				return response, nil
			}

			return testResponse(http.StatusOK), nil
		})
		require.NoError(t, fetcher.SetRetries(1))

		status, size, err := fetcher.FetchSize(context.Background(), prepareURL(t))

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, int64(len("test response")), size)
		require.Equal(t, 2, requestsMade)
	})

	t.Run("retry budget exhausted", func(t *testing.T) {
		requestsMade := 0

		fetcher := newTestFetcher(t, func(_ *http.Request) (*http.Response, error) {
			requestsMade++

			response := testResponse(http.StatusOK)
			response.Body = interruptedBody{}

			return response, nil
		})
		require.NoError(t, fetcher.SetRetries(1))

		status, size, err := fetcher.FetchSize(context.Background(), prepareURL(t))

		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		require.Equal(t, http.StatusOK, status)
		require.Zero(t, size)
		require.Equal(t, 2, requestsMade)
	})

	t.Run("retry server error", func(t *testing.T) {
		requestsMade := 0

		fetcher := newTestFetcher(t, func(_ *http.Request) (*http.Response, error) {
			requestsMade++

			if requestsMade == 1 {
				return testResponse(http.StatusInternalServerError), nil
			}

			return testResponse(http.StatusOK), nil
		})
		require.NoError(t, fetcher.SetRetries(1))

		status, size, err := fetcher.FetchSize(context.Background(), prepareURL(t))

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, int64(len("test response")), size)
		require.Equal(t, 2, requestsMade)
	})
}
