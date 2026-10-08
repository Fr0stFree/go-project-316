// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"code/internal/common/timeutils"
	"code/internal/workerpool"
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// CrawlJob represents a URL scheduled for crawling.
type taskPayload struct {
	URL          string
	Depth        int
	DiscoveredAt time.Time
}

func newTaskPayload(url string, depth int) *taskPayload {
	return &taskPayload{
		URL:          url,
		Depth:        depth,
		DiscoveredAt: timeutils.UTCNowPretty(),
	}
}

// TaskResult represents the result of processing a single crawl job.
type TaskResult struct {
	URL          string
	Depth        int
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	FoundURLs    []string
	Err          error
}

type Crawler struct {
	seenUrls    map[string]struct{}
	results     map[string]TaskResult
	client      *http.Client
	poolSize    int
	runningJobs atomic.Int64
	maxDepth    int
	userAgent   string
}

func New(userAgent string, client *http.Client, poolSize int, maxDepth int) *Crawler {
	return &Crawler{
		seenUrls:  make(map[string]struct{}),
		results:   make(map[string]TaskResult),
		maxDepth:  maxDepth,
		client:    client,
		userAgent: userAgent,
		poolSize:  poolSize,
	}
}

func (c *Crawler) Run(ctx context.Context, rootURL string) (map[string]TaskResult, error) {
	pool := workerpool.New(c.poolSize, c.process)
	pool.Start(ctx)

	c.seenUrls[rootURL] = struct{}{}
	job := *newTaskPayload(rootURL, 0)

	pool.Jobs() <- job

	c.runningJobs.Add(1)

	for c.runningJobs.Load() > 0 {
		select {
		case <-ctx.Done():
			pool.Stop()
		case result := <-pool.Results():
			c.runningJobs.Add(-1)

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
					pool.Stop()
				case pool.Jobs() <- job:
					c.runningJobs.Add(1)
				}
			}
		}
	}

	pool.Stop()

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
		job.URL,
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
