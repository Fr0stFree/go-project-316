// Package crawler provides functionality for web crawling and analysis.
package crawler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"code/internal/common/logger"
	"code/internal/crawler"
	"code/internal/report"
)

// Options defines the configuration for the web crawling operation.
type Options struct {
	URL         string
	Depth       int
	Retries     int
	Delay       time.Duration
	RPS         int
	Timeout     time.Duration
	UserAgent   string
	Concurrency int
	IndentJSON  bool
	Verbose     bool
	HTTPClient  *http.Client
}

// Report contains the pages and metadata collected during an analysis.
type Report = report.Report

// Analyze crawls the configured URL and returns the resulting report.
func Analyze(ctx context.Context, opts Options) (*Report, error) {
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

	return report, nil
}
