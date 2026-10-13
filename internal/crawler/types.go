package crawler

import (
	"code/internal/crawler/htmlparser"
	"net/url"
	"time"
)

type taskKind uint8

const (
	pageTask taskKind = iota
	assetTask
)

type task struct {
	url   *url.URL
	depth int
	kind  taskKind
}

type taskResult struct {
	page  *pageTaskResult
	asset *assetTaskResult
}

type pageTaskResult struct {
	url          *url.URL
	depth        int
	httpStatus   int
	status       string
	discoveredAt time.Time
	html         *htmlparser.ParsedHTML
	err          error
}

type assetTaskResult struct {
	url        *url.URL
	httpStatus int
	sizeBytes  int
	err        error
}

// AssetResult represents the result of fetching a page asset.
type AssetResult struct {
	URL        string
	Type       htmlparser.AssetType
	StatusCode int
	SizeBytes  int
	Error      error
}

// Result represents a crawled web page and its associated assets.
type Result struct {
	URL          *url.URL
	Depth        int
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	HTML         *htmlparser.ParsedHTML
	Assets       []AssetResult
	Err          error
}
