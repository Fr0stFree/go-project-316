// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"context"
	"net/http"
)

// MakeRequest makes an HTTP request to the specified URL with the given user agent.
func MakeRequest(ctx context.Context, client *http.Client, url string, userAgent string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", userAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
