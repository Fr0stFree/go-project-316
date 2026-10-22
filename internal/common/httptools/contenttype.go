package httptools

import (
	"mime"
	"net/http"
)

// IsHTMLResponse checks if the given HTTP response has a Content-Type indicating HTML.
func IsHTMLResponse(response *http.Response) bool {
	if response == nil {
		return false
	}

	contentType := response.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "text/html"
}
