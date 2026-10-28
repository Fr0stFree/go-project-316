package httptools_test

import (
	"net/url"
	"testing"

	"code/internal/common/httptools"

	"github.com/stretchr/testify/require"
)

func TestIsSameDomain(t *testing.T) {
	type subTest struct {
		name     string
		first    string
		second   string
		expected bool
	}

	tests := []subTest{
		{
			name:     "same domain",
			first:    "https://example.com/path",
			second:   "https://example.com/other",
			expected: true,
		},
		{
			name:     "case insensitive domain",
			first:    "https://EXAMPLE.COM",
			second:   "https://example.com",
			expected: true,
		},
		{
			name:     "different domains",
			first:    "https://example.com",
			second:   "https://another.com",
			expected: false,
		},
		{
			name:     "different subdomains",
			first:    "https://api.example.com",
			second:   "https://www.example.com",
			expected: false,
		},
		{
			name:     "different schemes",
			first:    "http://example.com",
			second:   "https://example.com",
			expected: true,
		},
		{
			name:     "different ports",
			first:    "https://example.com:8080",
			second:   "https://example.com:9090",
			expected: true,
		},
		{
			name:     "different paths and queries",
			first:    "https://example.com/a?x=1",
			second:   "https://example.com/b?y=2",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first, err := url.Parse(test.first)
			require.NoError(t, err)

			second, err := url.Parse(test.second)
			require.NoError(t, err)

			actual := httptools.IsSameDomain(first, second)

			require.Equal(t, test.expected, actual)
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	type subTest struct {
		name     string
		input    string
		expected string
	}

	tests := []subTest{
		{
			name:     "lowercase hostname",
			input:    "https://EXAMPLE.COM/path",
			expected: "https://example.com/path",
		},
		{
			name:     "remove fragment",
			input:    "https://example.com/path#section",
			expected: "https://example.com/path",
		},
		{
			name:     "lowercase hostname and remove fragment",
			input:    "https://EXAMPLE.COM/path#section",
			expected: "https://example.com/path",
		},
		{
			name:     "preserve query parameters",
			input:    "https://EXAMPLE.COM/path?a=1&b=2",
			expected: "https://example.com/path?a=1&b=2",
		},
		{
			name:     "preserve port",
			input:    "https://EXAMPLE.COM:8080/path",
			expected: "https://example.com:8080/path",
		},
		{
			name:     "preserve path casing",
			input:    "https://EXAMPLE.COM/Some/Path",
			expected: "https://example.com/Some/Path",
		},
		{
			name:     "already normalized",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original, err := url.Parse(test.input)
			require.NoError(t, err)

			normalized := httptools.NormalizeURL(original)

			require.Equal(t, test.expected, normalized.String())

			// Normalization must not modify the original URL.
			require.Equal(t, test.input, original.String())

			// The function must return a new URL instance.
			require.NotSame(t, original, normalized)
		})
	}
}
