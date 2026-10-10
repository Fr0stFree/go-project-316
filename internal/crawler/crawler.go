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
	seenUrls map[*url.URL]struct{}
	results  map[*url.URL]TaskResult
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
		seenUrls: make(map[*url.URL]struct{}),
		results:  make(map[*url.URL]TaskResult),
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

type taskPayload struct {
	URL          *url.URL
	Depth        int
	DiscoveredAt time.Time
}

func newTaskPayload(url *url.URL, depth int) *taskPayload {
	return &taskPayload{
		URL:          url,
		Depth:        depth,
		DiscoveredAt: timeutils.UTCNowPretty(),
	}
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

// Run crawls pages starting from rootURL up to the configured maximum depth.
// It returns the collected results or an error if the context is canceled.
func (c *Crawler) Run(ctx context.Context, rootURL string) (map[*url.URL]TaskResult, error) {
	stopPool := c.pool.Start(ctx, c.process, c.poolSize)
	defer stopPool()

	parsedRootURL, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}

	c.seenUrls[parsedRootURL] = struct{}{}
	task := *newTaskPayload(parsedRootURL, 0)

	c.pool.Jobs() <- task

	runningTasks := 1
	for runningTasks > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case result := <-c.pool.Results():
			runningTasks--

			c.results[result.URL] = result
			if result.Depth >= c.maxDepth {
				continue
			}

			for _, foundURL := range result.FoundURLs {
				if _, hasSeen := c.seenUrls[foundURL]; hasSeen {
					continue
				}

				c.seenUrls[foundURL] = struct{}{}

				job := *newTaskPayload(foundURL, result.Depth+1)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()

				case c.pool.Jobs() <- job:
					runningTasks++
				}
			}
		}
	}

	return c.results, nil
}

func (c *Crawler) process(ctx context.Context, job taskPayload) TaskResult {
	slog.Debug(
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

		return result
	}

	result.HTTPStatus = response.StatusCode
	result.Status = response.Status

	page, parseErr := htmlparser.ParsePage(response.Body, job.URL)
	closeErr := response.Body.Close()

	if parseErr != nil {
		result.Err = parseErr

		return result
	}

	if closeErr != nil {
		result.Err = closeErr

		return result
	}

	result.FoundURLs = page.Links

	return result
}
