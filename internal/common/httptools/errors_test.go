package httptools_test

import (
	"errors"
	"io"
	"net"
	"syscall"
	"testing"

	"code/internal/common/httptools"

	"github.com/stretchr/testify/require"
)

func TestIsRetryableError(t *testing.T) {
	type subTest struct {
		name     string
		err      error
		expected bool
	}

	tests := []subTest{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "EOF",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "unexpected EOF",
			err:      io.ErrUnexpectedEOF,
			expected: true,
		},
		{
			name:     "connection reset",
			err:      syscall.ECONNRESET,
			expected: true,
		},
		{
			name:     "connection refused",
			err:      syscall.ECONNREFUSED,
			expected: true,
		},
		{
			name:     "connection aborted",
			err:      syscall.ECONNABORTED,
			expected: true,
		},
		{
			name:     "broken pipe",
			err:      syscall.EPIPE,
			expected: true,
		},
		{
			name:     "network timeout",
			err:      &net.DNSError{Err: "i/o timeout", IsTimeout: true},
			expected: true,
		},
		{
			name:     "temporary DNS error",
			err:      &net.DNSError{Err: "temporary failure", IsTemporary: true},
			expected: true,
		},
		{
			name:     "permanent DNS error",
			err:      &net.DNSError{Err: "no such host", IsTemporary: false},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "permission denied",
			err:      syscall.EACCES,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := httptools.IsRetryableError(test.err)

			require.Equal(t, test.expected, actual)
		})
	}
}
