// Package httptools provides utility functions for working with HTTP requests and responses.
package httptools

import (
	"net/url"
	"strings"
)

const (
	// HTTP is the URL scheme for unencrypted HTTP connections.
	HTTP = "http"

	// HTTPS is the URL scheme for HTTP connections secured with TLS.
	HTTPS = "https"
)

// IsSameDomain checks if two URLs have the same domain.
func IsSameDomain(url1, url2 *url.URL) bool {
	return strings.EqualFold(url1.Hostname(), url2.Hostname())
}

// NormalizeURL returns a normalized copy without changing the original.
func NormalizeURL(u *url.URL) *url.URL {
	normalized := *u
	normalized.Host = strings.ToLower(normalized.Host)
	normalized.Fragment = ""
	normalized.RawFragment = ""

	return &normalized
}

// URLKey returns the identity used to deduplicate HTTP URLs. An empty path and "/" identify the same resource.
func URLKey(u *url.URL) string {
	normalized := NormalizeURL(u)
	if normalized.Path == "" {
		normalized.Path = "/"
	}

	return normalized.String()
}
