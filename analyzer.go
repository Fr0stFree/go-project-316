// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"net/http"
	"os"
	"time"

	"log/slog"

	"code/internal/common/fmttools"
	"code/internal/common/timeutils"
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

// Analyze crawls the configured URL and returns the resulting report as JSON.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	configureLogger()

	crawler := crawler.New(opts.UserAgent, opts.HTTPClient, opts.Concurrency, opts.Depth)

	crawledResults, err := crawler.Run(ctx, opts.URL)
	if err != nil {
		return nil, err
	}

	report := buildReport(opts.URL, opts.Depth, crawledResults)

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

func buildReport(rootURL string, maxDepth int, results map[string]crawler.TaskResult) *Report {
	report := &Report{
		URL:         rootURL,
		Depth:       maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]ReportPage, 0, len(results)),
	}

	for _, result := range results {
		report.Pages = append(report.Pages, ReportPage{
			URL:          result.URL,
			Depth:        result.Depth,
			HTTPStatus:   result.HTTPStatus,
			Status:       result.Status,
			BrokenLinks:  brokenLinks(result, results),
			DiscoveredAt: result.DiscoveredAt,
		})
	}

	return report
}

func brokenLinks(current crawler.TaskResult, all map[string]crawler.TaskResult) []NodeLink {
	brokenLinks := make([]NodeLink, 0)

	for _, url := range current.FoundURLs {
		target, exists := all[url]
		if !exists || target.Err == nil {
			continue
		}

		link := NodeLink{
			URL:        target.URL,
			StatusCode: target.HTTPStatus,
		}

		if target.Err != nil {
			link.Error = target.Err.Error()
		}

		brokenLinks = append(brokenLinks, link)
	}

	return brokenLinks
}
