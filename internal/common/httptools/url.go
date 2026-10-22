// Package httptools provides utility functions for working with HTTP requests and responses.
package httptools

import (
	"net/url"
	"strings"
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
