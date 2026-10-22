package httptools

import (
	"net/http"
)

// IsRetryableStatus checks if the given HTTP status code is considered retryable.
func IsRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError && status < 600
}

// IsMethodSupported reports whether the response does not explicitly reject the requested method.
func IsMethodSupported(status int) bool {
	return status != http.StatusMethodNotAllowed && status != http.StatusNotImplemented
}
