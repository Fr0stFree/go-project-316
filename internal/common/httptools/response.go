package httptools

import (
	"io"
	"net/http"
)

// MeasureRespBodySize returns the size of an HTTP response body in bytes.
func MeasureRespBodySize(resp *http.Response) (int, error) {
	if resp.ContentLength >= 0 {
		return int(resp.ContentLength), nil
	}

	size, err := io.Copy(io.Discard, resp.Body)

	return int(size), err
}
