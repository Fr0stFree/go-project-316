package crawler

import (
	"net/url"
	"time"
)

type taskPayload struct {
	URL          *url.URL
	Depth        int
	DiscoveredAt time.Time
}

// TaskResult represents the result of processing a single crawl task.
type TaskResult struct {
	URL          *url.URL
	Depth        int
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	FoundURLs    []*url.URL
	Err          error
}
