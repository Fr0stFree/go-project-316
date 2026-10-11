// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"code/internal/common/timeutils"
	"code/internal/crawler/htmlparser"
	"code/internal/crawler/httpfetcher"
	"code/internal/crawler/ratelimiter"
	"code/internal/crawler/workerpool"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Crawler manages concurrent web crawling and tracks discovered URLs and results.
type Crawler struct {
	fetcher  *httpfetcher.Fetcher
	pool     *workerpool.WorkerPool[taskPayload, TaskResult]
	poolSize int
	maxDepth int
}

// Option configures a Crawler and returns an error if the configuration is invalid.
type Option func(*Crawler) error

// WithPoolSize sets the number of concurrent workers.
func WithPoolSize(size int) Option {
	return func(c *Crawler) error {
		if size <= 0 {
			return errors.New("size cannot be non-positive")
		}

		c.poolSize = size

		return nil
	}
}

// WithMaxDepth sets the maximum crawling depth.
func WithMaxDepth(depth int) Option {
	return func(c *Crawler) error {
		if depth < 0 {
			return errors.New("max depth cannot negative")
		}

		c.maxDepth = depth

		return nil
	}
}

// WithUserAgent sets the User-Agent header used for HTTP requests.
func WithUserAgent(userAgent string) Option {
	return func(c *Crawler) error {
		err := c.fetcher.SetUserAgent(userAgent)
		if err != nil {
			return err
		}

		return nil
	}
}

// WithRPS configures the crawler to limit HTTP requests to the specified number of requests per second
func WithRPS(ctx context.Context, rps int) Option {
	return func(c *Crawler) error {
		limiter, err := ratelimiter.NewRPSRateLimiter(ctx, rps)
		if err != nil {
			return err
		}

		c.fetcher.SetRateLimiter(limiter)

		return nil
	}
}

// WithDelay configures the crawler to enforce a minimum delay between HTTP requests
func WithDelay(delay time.Duration) Option {
	return func(c *Crawler) error {
		limiter, err := ratelimiter.NewDelayRateLimiter(delay)
		if err != nil {
			return err
		}

		c.fetcher.SetRateLimiter(limiter)

		return nil
	}
}

// WithMaxAttempts sets the retry configuration for HTTP requests.
func WithMaxAttempts(maxAttempts int) Option {
	return func(c *Crawler) error {
		err := c.fetcher.SetMaxAttempts(maxAttempts)
		if err != nil {
			return err
		}

		return nil
	}
}

// New creates a Crawler with the specified HTTP client, worker count, and maximum crawl depth.
func New(client *http.Client, opts ...Option) (*Crawler, error) {
	c := &Crawler{
		maxDepth: 3,
		fetcher:  httpfetcher.New(client),
		pool:     workerpool.New[taskPayload, TaskResult](),
		poolSize: 3,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Run crawls pages starting from rootURL up to the configured maximum depth.
func (c *Crawler) Run(ctx context.Context, rootURL string) (map[string]TaskResult, error) {
	state := newCrawlerState()
	startedAt := time.Now()

	slog.DebugContext(
		ctx,
		"Running crawler...",
		"maxDepth", c.maxDepth,
		"poolSize", c.poolSize,
		"rootURL", rootURL,
	)

	stopPool := c.pool.Start(ctx, c.process, c.poolSize)
	defer stopPool()

	defer func() {
		slog.DebugContext(
			ctx,
			"Finished processing urls",
			"duration", time.Since(startedAt),
			"urlsAmount", len(state.seenURLs),
		)
	}()

	parsedRootURL, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}

	state.tryMarkSeen(parsedRootURL)
	task := c.newTaskPayload(parsedRootURL, 0)

	select {
	case c.pool.Jobs() <- task:
		state.increaseTasks()
	case <-ctx.Done():
		return state.results, ctx.Err()
	}

	for state.hasTasks() {
		select {
		case <-ctx.Done():
			return state.results, ctx.Err()

		case result, ok := <-c.pool.Results():
			if !ok {
				return state.results, errors.New("worker pool stopped unexpectedly")
			}

			state.completeTask(result)

			if result.Depth >= c.maxDepth || result.HTML == nil {
				continue
			}

			for _, url := range result.HTML.Links {
				if hasSeen := !state.tryMarkSeen(url); hasSeen {
					continue
				}

				task := c.newTaskPayload(url, result.Depth+1)
				select {
				case <-ctx.Done():
					return state.results, ctx.Err()

				case c.pool.Jobs() <- task:
					state.increaseTasks()
				}
			}
		}
	}

	return state.results, nil
}

func (c *Crawler) process(ctx context.Context, job taskPayload) TaskResult {
	slog.DebugContext(
		ctx,
		"Processing URL",
		"url", job.URL,
		"depth", job.Depth,
	)

	result := TaskResult{
		URL:          job.URL,
		Depth:        job.Depth,
		DiscoveredAt: job.DiscoveredAt,
	}

	response, respErr := c.fetcher.Fetch(ctx, job.URL)
	if respErr != nil {
		result.Err = respErr
		slog.DebugContext(
			ctx,
			"Failed to fetch url",
			"url", job.URL.String(),
			"err", respErr.Error(),
		)

		return result
	}

	result.HTTPStatus = response.StatusCode
	result.Status = response.Status

	html, parseErr := htmlparser.Parse(response.Body, job.URL)
	closeErr := response.Body.Close()

	if parseErr != nil {
		result.Err = parseErr

		return result
	}

	if closeErr != nil {
		result.Err = closeErr

		return result
	}

	result.HTML = html

	return result
}

func (c *Crawler) newTaskPayload(url *url.URL, depth int) taskPayload {
	return taskPayload{
		URL:          url,
		Depth:        depth,
		DiscoveredAt: timeutils.UTCNowPretty(),
	}
}
