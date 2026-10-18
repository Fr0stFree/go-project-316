// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"errors"
	"net/http"

	"code/internal/common/fmttools"
	"code/internal/common/logger"
	"code/internal/crawler"
	"code/internal/report"
)

// Options defines the configuration for the web crawling operation.
type Options struct {
	URL         string
	Depth       int
	Retries     int
	Delay       string
	RPS         int
	Timeout     string
	UserAgent   string
	Concurrency int
	IndentJSON  bool
	Verbose     bool
	HTTPClient  *http.Client
}

// Analyze crawls the configured URL and returns the resulting report as JSON.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	logger.Configure(opts.Verbose)

	crawl, err := crawler.New(
		opts.HTTPClient,
		crawler.WithUserAgent(opts.UserAgent),
		crawler.WithMaxAttempts(opts.Retries),
		crawler.WithMaxDepth(opts.Depth),
		crawler.WithPoolSize(opts.Concurrency),
		crawler.WithTimeout(opts.Timeout),
		crawler.WithDelay(opts.Delay),
		crawler.WithRPS(ctx, opts.RPS),
	)
	if err != nil {
		return nil, err
	}

	crawlResults, err := crawl.Run(ctx, opts.URL)
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	report := report.Build(opts.URL, opts.Depth, crawlResults)

	data, err := fmttools.ToJSON(report, opts.IndentJSON)
	if err != nil {
		return nil, err
	}

	return data, nil
}
