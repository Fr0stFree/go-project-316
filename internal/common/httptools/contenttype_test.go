package httptools_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"code/internal/common/httptools"

	"github.com/stretchr/testify/require"
)

func TestIsMarkupResponse(t *testing.T) {
	type subTest struct {
		name     string
		response *http.Response
		expected bool
	}

	tests := []subTest{
		{
			name:     "nil response",
			response: nil,
			expected: false,
		},
		{
			name: "HTML content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/html"},
				},
			},
			expected: true,
		},
		{
			name: "HTML with charset",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/html; charset=utf-8"},
				},
			},
			expected: true,
		},
		{
			name: "HTML with uppercase media type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"TEXT/HTML"},
				},
			},
			expected: true,
		},
		{
			name: "XML content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"application/xml"},
				},
			},
			expected: true,
		},
		{
			name: "text XML content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/xml"},
				},
			},
			expected: true,
		},
		{
			name: "JSON content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
			},
			expected: false,
		},
		{
			name: "plain text content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/plain"},
				},
			},
			expected: false,
		},
		{
			name: "missing content type",
			response: &http.Response{
				Header: http.Header{},
			},
			expected: false,
		},
		{
			name: "invalid content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/html; charset="},
				},
			},
			expected: false,
		},
		{
			name: "empty content type",
			response: &http.Response{
				Header: http.Header{
					"Content-Type": {""},
				},
			},
			expected: false,
		},
		{
			name:     "nil headers",
			response: &http.Response{},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := httptools.IsMarkupResponse(test.response)

			require.Equal(t, test.expected, actual)
		})
	}
}

func TestDecodeMarkupResponse(t *testing.T) {
	t.Run("should decode ISO-8859-1 HTML", func(t *testing.T) {
		response := &http.Response{
			Header: http.Header{
				"Content-Type": {"text/html; charset=iso-8859-1"},
			},
			Body: io.NopCloser(strings.NewReader("Caf\xe9")),
		}
		defer func() { _ = response.Body.Close() }()

		reader, err := httptools.DecodeMarkupResponse(response)
		require.NoError(t, err)

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "Café", string(data))
	})

	t.Run("should leave XML unchanged", func(t *testing.T) {
		body := "<rss><title>Example</title></rss>"

		response := &http.Response{
			Header: http.Header{
				"Content-Type": {"application/xml"},
			},
			Body: io.NopCloser(strings.NewReader(body)),
		}
		defer func() { _ = response.Body.Close() }()

		reader, err := httptools.DecodeMarkupResponse(response)
		require.NoError(t, err)

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, body, string(data))
	})
}
