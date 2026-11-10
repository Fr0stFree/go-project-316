package httptools

import (
	"io"
	"mime"
	"net/http"

	"golang.org/x/net/html/charset"
)

// IsMarkupResponse checks if the response has an HTML or XML content type.
func IsMarkupResponse(response *http.Response) bool {
	if response == nil {
		return false
	}

	contentType := response.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "text/html" || mediaType == "application/xml" || mediaType == "text/xml"
}

// DecodeMarkupResponse converts HTML to UTF-8 and leaves XML unchanged.
func DecodeMarkupResponse(response *http.Response) (io.Reader, error) {
	contentType := response.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	if mediaType == "text/html" {
		return charset.NewReader(response.Body, contentType)
	}

	return response.Body, nil
}
