package crawler

import (
	"net/url"
	"time"

	"code/internal/crawler/htmlparser"
)

type taskKind uint8

const (
	pageTask taskKind = iota
	linkCheckTask
	assetTask
)

type task struct {
	url   *url.URL
	depth int
	kind  taskKind
}

type taskResult struct {
	page  *pageTaskResult
	link  *linkTaskResult
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

type linkTaskResult struct {
	url        *url.URL
	depth      int
	httpStatus int
	err        error
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
	CheckedOnly  bool
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	HTML         *htmlparser.ParsedHTML
	Assets       []AssetResult
	Err          error
}
