// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"code/internal/common/httptools"
	"code/internal/crawler/httpfetcher"
	"code/internal/crawler/ratelimiter"
	"code/internal/crawler/workerpool"

	"github.com/samber/lo"
)

// Crawler manages concurrent web crawling and tracks discovered URLs and results.
type Crawler struct {
	fetcher  *httpfetcher.Fetcher
	now      func() time.Time
	poolSize int
	maxDepth int
}

const (
	defaultMaxDepth   = 3
	defaultPoolSize   = 3
	defaultBufferSize = 100
)

// Option configures a Crawler and returns an error if the configuration is invalid.
type Option func(*Crawler) error

// WithClock sets the time source used for page discovery timestamps.
func WithClock(now func() time.Time) Option {
	return func(c *Crawler) error {
		if now != nil {
			c.now = now
		} else {
			c.now = time.Now
		}

		return nil
	}
}

// WithPoolSize sets the number of concurrent workers.
func WithPoolSize(size int) Option {
	return func(c *Crawler) error {
		if size == 0 {
			return nil
		}

		if size < 0 {
			return errors.New("size cannot be negative")
		}

		c.poolSize = size

		return nil
	}
}

// WithMaxDepth sets the maximum crawling depth.
func WithMaxDepth(depth int) Option {
	return func(c *Crawler) error {
		if depth < 0 {
			return errors.New("max depth cannot be negative")
		}

		c.maxDepth = depth

		return nil
	}
}

// WithUserAgent sets the User-Agent header used for HTTP requests.
func WithUserAgent(userAgent string) Option {
	return func(c *Crawler) error {
		if userAgent == "" {
			return nil
		}

		err := c.fetcher.SetUserAgent(userAgent)
		if err != nil {
			return err
		}

		return nil
	}
}

// WithTimeout sets the timeout for every HTTP request.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Crawler) error {
		if timeout == 0 {
			return nil
		}

		return c.fetcher.SetTimeout(timeout)
	}
}

// WithRPS configures the crawler to limit HTTP requests to the specified number of requests per second
func WithRPS(ctx context.Context, rps int) Option {
	return func(c *Crawler) error {
		if rps == 0 {
			return nil
		}

		limiter, err := ratelimiter.NewRPSRateLimiter(ctx, rps)
		if err != nil {
			return err
		}

		err = c.fetcher.SetRateLimiter(limiter)

		return err
	}
}

// WithDelay configures the crawler to enforce a minimum delay between HTTP requests
func WithDelay(delay time.Duration) Option {
	return func(c *Crawler) error {
		if delay == 0 {
			return nil
		}

		limiter, err := ratelimiter.NewDelayRateLimiter(delay)
		if err != nil {
			return err
		}

		err = c.fetcher.SetRateLimiter(limiter)

		return err
	}
}

// WithRetries sets the number of retries after the first HTTP request.
func WithRetries(retries int) Option {
	return func(c *Crawler) error {
		return c.fetcher.SetRetries(retries)
	}
}

// New creates a Crawler with the specified HTTP client, worker count, and maximum crawl depth.
func New(client *http.Client, opts ...Option) (*Crawler, error) {
	c := &Crawler{
		maxDepth: defaultMaxDepth,
		fetcher:  httpfetcher.New(client),
		now:      time.Now,
		poolSize: defaultPoolSize,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Run crawls pages starting from rootURL up to the configured maximum depth.
func (c *Crawler) Run(ctx context.Context, rootURL string) (map[string]Result, error) {
	root, err := c.cleanURL(rootURL)
	if err != nil {
		return nil, err
	}

	pool := workerpool.New[task, taskResult](defaultBufferSize)
	state := newCrawlerState(root)

	state.recordURL(root, 0)
	state.markPageScheduled(root)
	firstTask := task{url: root, kind: pageTask}
	state.pending.PushBack(firstTask)

	stop := pool.Start(ctx, c.process, c.poolSize)
	defer stop()

	for state.hasWork() {
		if err := c.runIteration(ctx, pool, state); err != nil {
			return c.mergeResults(state), err
		}
	}

	return c.mergeResults(state), nil
}

func (c *Crawler) runIteration(
	ctx context.Context,
	pool *workerpool.WorkerPool[task, taskResult],
	state *crawlerState,
) error {
	var (
		tasks    chan<- task
		nextTask task
	)

	if next := state.pending.Front(); next != nil {
		tasks = pool.Tasks()
		nextTask = next.Value.(task)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case tasks <- nextTask:
		state.pending.Remove(state.pending.Front())
		state.running++

	case result, ok := <-pool.Results():
		if !ok {
			return errors.New("worker pool stopped unexpectedly")
		}

		state.running--

		switch {
		case result.page != nil:
			c.handlePageResult(state, *result.page)

		case result.link != nil:
			c.handleLinkResult(state, *result.link)

		case result.asset != nil:
			c.handleAssetResult(state, *result.asset)

		default:
			return errors.New("worker returned an empty result")
		}
	}

	return nil
}

func (c *Crawler) handleAssetResult(state *crawlerState, result assetTaskResult) {
	state.assets[httptools.URLKey(result.url)] = result
}

func (c *Crawler) handleLinkResult(state *crawlerState, result linkTaskResult) {
	state.links[httptools.URLKey(result.url)] = result
}

func (c *Crawler) handlePageResult(state *crawlerState, result pageTaskResult) {
	result = state.savePage(result)

	if result.html == nil || result.depth >= max(1, c.maxDepth) {
		return
	}

	for _, asset := range result.html.Assets {
		if !state.markAssetSeen(asset.URL) {
			continue
		}

		state.pending.PushBack(task{url: asset.URL, kind: assetTask})
	}

	for _, link := range result.html.Links {
		depth := result.depth + 1
		key := httptools.URLKey(link)

		first, improved := state.recordURL(link, depth)
		if !improved {
			continue
		}

		if depth < max(1, c.maxDepth) && httptools.IsSameDomain(state.root, link) {
			if page, done := state.pages[key]; done {
				page.depth = depth
				c.handlePageResult(state, page)
			} else if state.markPageScheduled(link) {
				state.pending.PushBack(task{url: link, depth: depth, kind: pageTask})
			}

			continue
		}

		if first {
			state.pending.PushBack(task{url: link, depth: depth, kind: linkCheckTask})
		}
	}
}

func (c *Crawler) mergeResults(state *crawlerState) map[string]Result {
	results := make(map[string]Result, len(state.pages)+len(state.links))

	for key, page := range state.pages {
		result := Result{
			URL:          page.url,
			Depth:        page.depth,
			HTTPStatus:   page.httpStatus,
			Status:       page.status,
			DiscoveredAt: page.discoveredAt,
			HTML:         page.html,
			Assets:       make([]AssetResult, 0),
			Err:          page.err,
		}

		if page.html != nil {
			for _, asset := range page.html.Assets {
				item := AssetResult{
					URL:  asset.URL,
					Type: asset.Type,
				}

				if fetched, ok := state.assets[httptools.URLKey(asset.URL)]; ok {
					item.StatusCode = fetched.httpStatus
					item.SizeBytes = fetched.sizeBytes

					if fetched.err != nil {
						item.Error = fetched.err
					}
				} else {
					item.Error = errors.New("asset was not fetched")
				}

				result.Assets = append(result.Assets, item)
			}
		}

		results[key] = result
	}

	for key, link := range state.links {
		if _, exists := state.pages[key]; exists {
			continue
		}

		status := lo.Ternary(
			link.err != nil || link.httpStatus >= http.StatusBadRequest,
			statusError,
			statusOK,
		)

		results[key] = Result{
			URL:         link.url,
			Depth:       state.minDepth(link.url, link.depth),
			CheckedOnly: true,
			HTTPStatus:  link.httpStatus,
			Status:      status,
			Err:         link.err,
		}
	}

	return results
}

func (c *Crawler) cleanURL(rawURL string) (*url.URL, error) {
	url, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if url.Scheme != httptools.HTTP && url.Scheme != httptools.HTTPS {
		return nil, fmt.Errorf("unsupported root URL scheme %q: only http and https are allowed", url.Scheme)
	}

	if url.Hostname() == "" {
		return nil, fmt.Errorf("invalid URL: hostname is required")
	}

	return httptools.NormalizeURL(url), nil
}
