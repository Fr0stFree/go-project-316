// Package httptools provides utility functions for working with HTTP requests and responses.
package httptools

import (
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

const (
	// HTTP is the URL scheme for unencrypted HTTP connections.
	HTTP = "http"

	// HTTPS is the URL scheme for HTTP connections secured with TLS.
	HTTPS = "https"
)

// IsSameDomain checks if two URLs have the same domain.
func IsSameDomain(url1, url2 *url.URL) bool {
	if url1 == nil || url2 == nil {
		return false
	}

	host1, err1 := idna.Lookup.ToASCII(url1.Hostname())

	host2, err2 := idna.Lookup.ToASCII(url2.Hostname())
	if err1 != nil || err2 != nil {
		return false
	}

	return strings.EqualFold(host1, host2)
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

// SanitizeURL removes user credentials without modifying the original URL.
func SanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	sanitized := *u
	sanitized.User = nil

	return sanitized.String()
}

// SanitizeURLStr removes user credentials from a URL string.
func SanitizeURLStr(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	return SanitizeURL(u)
}
