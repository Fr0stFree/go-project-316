package httptools

import (
	"io"
	"mime"
	"net/http"

	"golang.org/x/net/html/charset"
)

func identifyResponseMediaType(response *http.Response) string {
	if response == nil {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return ""
	}

	return mediaType
}

// IsHTMLResponse reports whether the response contains HTML.
func IsHTMLResponse(response *http.Response) bool {
	return identifyResponseMediaType(response) == "text/html"
}

// IsXMLResponse reports whether the response contains a supported XML media type.
func IsXMLResponse(response *http.Response) bool {
	switch identifyResponseMediaType(response) {
	case "application/xml", "text/xml", "application/rss+xml":
		return true
	default:
		return false
	}
}

// IsMarkupResponse reports whether the response contains HTML or XML.
func IsMarkupResponse(response *http.Response) bool {
	return IsHTMLResponse(response) || IsXMLResponse(response)
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
