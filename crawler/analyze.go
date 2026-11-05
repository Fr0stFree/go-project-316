// Package crawler provides functionality for web crawling and analysis.
package crawler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"code/internal/common/fmttools"
	"code/internal/common/timeutils"
	"code/internal/crawler"
	"code/internal/report"

	"github.com/samber/lo"
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
	HTTPClient  *http.Client
	Now         func() time.Time
}

// Report contains the pages and metadata collected during an analysis.
type Report = report.Report

// Analyze crawls the configured URL and returns the resulting report.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	now := lo.Ternary(opts.Now != nil, opts.Now, time.Now)

	crawl, err := crawler.New(
		opts.HTTPClient,
		crawler.WithClock(now),
		crawler.WithUserAgent(opts.UserAgent),
		crawler.WithRetries(opts.Retries),
		crawler.WithMaxDepth(opts.Depth),
		crawler.WithPoolSize(opts.Concurrency),
		crawler.WithTimeout(opts.Timeout),
		crawler.WithDelay(opts.Delay),
		crawler.WithRPS(ctx, opts.RPS),
	)
	if err != nil {
		return nil, err
	}

	crawlResults, crawlErr := crawl.Run(ctx, opts.URL)
	if crawlErr != nil && !errors.Is(crawlErr, context.Canceled) {
		return nil, crawlErr
	}

	result := report.Build(opts.URL, opts.Depth, crawlResults, timeutils.UTCTruncatedToSeconds(now))

	data, err := fmttools.ToJSON(result, opts.IndentJSON)
	if err != nil {
		return nil, err
	}

	return data, crawlErr
}
