package httptools

import (
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
)

// IsRetryableStatus checks if the given HTTP status code is considered retryable.
func IsRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError && status < 600
}

// IsRetryableError checks if the given error is considered retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return true
	}

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) && dnsError.IsTemporary {
		return true
	}

	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE)
}
