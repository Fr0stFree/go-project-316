package httptools_test

import (
	"net/http"
	"testing"

	"code/internal/common/httptools"

	"github.com/stretchr/testify/require"
)

func TestIsRetryableStatus(t *testing.T) {
	type subTest struct {
		name     string
		status   int
		expected bool
	}

	tests := []subTest{
		{
			name:     "retry 429",
			status:   http.StatusTooManyRequests,
			expected: true,
		},
		{
			name:     "retry 500",
			status:   http.StatusInternalServerError,
			expected: true,
		},
		{
			name:     "skip unexpected big number",
			status:   600,
			expected: false,
		},
		{
			name:     "skip ok",
			status:   200,
			expected: false,
		},
		{
			name:     "skip unexpected small number",
			status:   0,
			expected: false,
		},
		{
			name:     "skip 499",
			status:   499,
			expected: false,
		},
		{
			name:     "retry 501",
			status:   http.StatusNotImplemented,
			expected: true,
		},
		{
			name:     "retry 599",
			status:   599,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := httptools.IsRetryableStatus(test.status)

			require.Equal(t, test.expected, actual)
		})
	}
}

func TestIsMethodSupported(t *testing.T) {
	type subTest struct {
		name     string
		status   int
		expected bool
	}

	tests := []subTest{
		{
			name:     "method not allowed",
			status:   http.StatusMethodNotAllowed,
			expected: false,
		},
		{
			name:     "not implemented",
			status:   http.StatusNotImplemented,
			expected: false,
		},
		{
			name:     "ok",
			status:   http.StatusOK,
			expected: true,
		},
		{
			name:     "internal server error",
			status:   http.StatusInternalServerError,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := httptools.IsMethodSupported(test.status)
			require.Equal(t, test.expected, actual)
		})
	}
}
