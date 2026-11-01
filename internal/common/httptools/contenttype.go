package httptools

import (
	"mime"
	"net/http"
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
