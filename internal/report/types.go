package report

import "time"

// Report represents the result of a web crawling operation.
type Report struct {
	URL         string       `json:"root_url"`
	Depth       int          `json:"depth"`
	GeneratedAt time.Time    `json:"generated_at"`
	Pages       []ReportPage `json:"pages"`
}

// ReportPage represents a single crawled page.
type ReportPage struct {
	URL          string     `json:"url"`
	Depth        int        `json:"depth"`
	HTTPStatus   int        `json:"http_status"`
	Status       string     `json:"status"`
	BrokenLinks  []NodeLink `json:"broken_links"`
	DiscoveredAt time.Time  `json:"discovered_at"`
}

// NodeLink represents a broken link found on a crawled page.
type NodeLink struct {
	URL        string `json:"url"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}
