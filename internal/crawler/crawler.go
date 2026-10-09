// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"code/internal/common/timeutils"
	"code/internal/common/types"
	"code/internal/workerpool"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type taskPayload struct {
	URL          types.URL
	Depth        int
	DiscoveredAt time.Time
}

func newTaskPayload(url types.URL, depth int) *taskPayload {
	return &taskPayload{
		URL:          url,
		Depth:        depth,
		DiscoveredAt: timeutils.UTCNowPretty(),
	}
}

// TaskResult represents the result of processing a single crawl task.
type TaskResult struct {
	URL          types.URL
	Depth        int
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	FoundURLs    []types.URL
	Err          error
}

// Crawler manages concurrent web crawling and tracks discovered URLs and results.
type Crawler struct {
	seenUrls  map[types.URL]struct{}
	results   map[types.URL]TaskResult
	client    *http.Client
	poolSize  int
	maxDepth  int
	userAgent string
}

// New creates a Crawler with the specified HTTP client, worker count, and maximum crawl depth.
func New(userAgent string, client *http.Client, poolSize int, maxDepth int) (*Crawler, error) {
	if poolSize <= 0 {
		return nil, errors.New("invalid amount of workers")
	}

	return &Crawler{
		seenUrls:  make(map[types.URL]struct{}),
		results:   make(map[types.URL]TaskResult),
		maxDepth:  maxDepth,
		client:    client,
		userAgent: userAgent,
		poolSize:  poolSize,
	}, nil
}

// Run crawls pages starting from rootURL up to the configured maximum depth.
// It returns the collected results or an error if the context is canceled.
func (c *Crawler) Run(ctx context.Context, rootURL types.URL) (map[types.URL]TaskResult, error) {
	pool := workerpool.New(c.poolSize, c.process)

	stopPool := pool.Start(ctx)
	defer stopPool()

	c.seenUrls[rootURL] = struct{}{}
	task := *newTaskPayload(rootURL, 0)

	pool.Jobs() <- task

	// Track pending tasks to determine when crawling is complete.
	runningTasks := 1

	for runningTasks > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case result := <-pool.Results():
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

				case pool.Jobs() <- job:
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

	request, reqErr := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		string(job.URL),
		nil,
	)
	if reqErr != nil {
		result.Err = reqErr

		return result
	}

	request.Header.Set("User-Agent", c.userAgent)

	response, err := c.client.Do(request)
	if err != nil {
		result.Err = err

		return result
	}

	result.HTTPStatus = response.StatusCode
	result.Status = response.Status

	page, parseErr := parseHTMLPage(response.Body)
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
