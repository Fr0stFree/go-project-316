// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"net/http"
	"os"

	"log/slog"

	"code/internal/common/fmttools"
	"code/internal/crawler"
	"code/internal/report"
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

// Analyze crawls the configured URL and returns the resulting report as JSON.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	configureLogger()

	crawler, err := crawler.New(
		opts.HTTPClient,
		crawler.WithUserAgent(opts.UserAgent),
		crawler.WithPoolSize(opts.Concurrency),
		crawler.WithMaxDepth(opts.Depth),
	)
	if err != nil {
		return nil, err
	}

	crawledResults, err := crawler.Run(ctx, opts.URL)
	if err != nil {
		return nil, err
	}

	report := report.Build(opts.URL, opts.Depth, crawledResults)

	return fmttools.ToJSON(report, opts.IndentJSON)
}

func configureLogger() {
	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	slog.SetDefault(slog.New(handler))
}
