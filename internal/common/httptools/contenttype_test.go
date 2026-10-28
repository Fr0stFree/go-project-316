package httptools_test

import (
	"net/http"
	"testing"

	"code/internal/common/httptools"

	"github.com/stretchr/testify/require"
)

func TestIsHTMLResponse(t *testing.T) {
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
			actual := httptools.IsHTMLResponse(test.response)

			require.Equal(t, test.expected, actual)
		})
	}
}
