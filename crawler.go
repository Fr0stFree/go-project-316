// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"net/http"

	"code/internal/common/fmttools"
	"code/internal/crawler"
)

// Options defines the configuration for the web crawling operation.
type Options struct {
	URL         string
	Depth       int
	Retries     int
	Delay       string
	Timeout     string
	UserAgent   string
	Concurrency int
	IndentJSON  bool
	HTTPClient  *http.Client
}

// Analyze performs a web crawling operation based on the provided options and returns a JSON report.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	parser := crawler.New(opts.HTTPClient).
		WithUserAgent(opts.UserAgent).
		WithMaxRetries(opts.Retries).
		WithMaxDepth(opts.Depth)

	page, err := parser.Crawl(ctx, opts.URL)
	if err != nil {
		return nil, err
	}

	report, err := fmttools.ToJSON(page, opts.IndentJSON)
	if err != nil {
		return nil, err
	}

	return report, nil
}
