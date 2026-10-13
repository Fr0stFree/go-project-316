package report

import (
	"time"
)

// Report represents the result of a web crawling operation.
type Report struct {
	URL         string    `json:"root_url"`
	Depth       int       `json:"depth"`
	GeneratedAt time.Time `json:"generated_at"`
	Pages       []Page    `json:"pages"`
}

// Page represents a single crawled page.
type Page struct {
	URL          string     `json:"url"`
	Depth        int        `json:"depth"`
	HTTPStatus   int        `json:"http_status"`
	Status       string     `json:"status"`
	SEO          SEO        `json:"seo"`
	BrokenLinks  []NodeLink `json:"broken_links"`
	Assets       []Asset    `json:"assets"`
	DiscoveredAt time.Time  `json:"discovered_at"`
}

// Asset represents a web resource and its HTTP fetch results.
type Asset struct {
	URL        string `json:"url"`
	Type       string `json:"type"`
	StatusCode int    `json:"status_code"`
	SizeBytes  int    `json:"size_bytes"`
	Error      string `json:"error"`
}

// NodeLink represents a broken link found on a crawled page.
type NodeLink struct {
	URL        string `json:"url"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// SEO represents basic SEO metadata extracted from an HTML page.
type SEO struct {
	HasTitle       bool   `json:"has_title"`
	Title          string `json:"title"`
	HasDescription bool   `json:"has_description"`
	Description    string `json:"description"`
	HasH1          bool   `json:"has_h1"`
}
