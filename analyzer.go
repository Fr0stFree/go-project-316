// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"log/slog"

	"code/internal/common/fmttools"
	"code/internal/common/timeutils"
	"code/internal/htmlparser"
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

// CrawlJob represents a URL scheduled for crawling.
type CrawlJob struct {
	URL          string
	Depth        int
	DiscoveredAt time.Time
}

// CrawlResult represents the result of processing a single crawl job.
type CrawlResult struct {
	URL          string
	Depth        int
	HTTPStatus   int
	Status       string
	DiscoveredAt time.Time
	FoundURLs    []string
	Err          error
}

// IsBroken reports whether crawling the URL resulted in a network,
// parsing, or HTTP error.
func (r CrawlResult) IsBroken() bool {
	return r.Err != nil || r.HTTPStatus >= http.StatusBadRequest
}

// Analyze crawls the configured URL and returns the resulting report as JSON.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	configureLogger()

	state := NewCrawlState()
	pool := NewWorkerPool(
		ctx,
		opts.UserAgent,
		opts.HTTPClient,
	)
	pool.Start(opts.Concurrency)

	rootJob := CrawlJob{
		URL:          opts.URL,
		Depth:        0,
		DiscoveredAt: timeutils.UTCNowPretty(),
	}

	state.MarkSeen(rootJob.URL)

	if !pool.Submit(rootJob) {
		pool.Stop()

		return nil, ctx.Err()
	}

	pendingJobs := 1

	for pendingJobs > 0 {
		select {
		case <-ctx.Done():
			pool.Stop()

			return nil, ctx.Err()

		case result := <-pool.Results():
			pendingJobs--

			if err := state.AddResult(result); err != nil {
				pool.Stop()

				return nil, err
			}

			if result.Depth >= opts.Depth {
				continue
			}

			for _, foundURL := range result.FoundURLs {
				if !state.TryMarkSeen(foundURL) {
					continue
				}

				job := CrawlJob{
					URL:          foundURL,
					Depth:        result.Depth + 1,
					DiscoveredAt: timeutils.UTCNowPretty(),
				}

				if !pool.Submit(job) {
					pool.Stop()

					return nil, ctx.Err()
				}

				pendingJobs++
			}
		}
	}

	pool.Stop()

	report := state.BuildReport(opts.URL, opts.Depth)

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

// CrawlState contains the state accumulated during a crawl.
type CrawlState struct {
	seen    map[string]struct{}
	results map[string]CrawlResult
}

// NewCrawlState creates an empty crawl state.
func NewCrawlState() *CrawlState {
	return &CrawlState{
		seen:    make(map[string]struct{}),
		results: make(map[string]CrawlResult),
	}
}

// MarkSeen marks a URL as discovered.
func (s *CrawlState) MarkSeen(url string) {
	s.seen[url] = struct{}{}
}

// TryMarkSeen marks a URL as discovered if it has not been seen before.
//
// It returns true when the URL was newly discovered.
func (s *CrawlState) TryMarkSeen(url string) bool {
	if _, exists := s.seen[url]; exists {
		return false
	}

	s.MarkSeen(url)

	return true
}

// AddResult stores the result of a completed crawl job.
func (s *CrawlState) AddResult(result CrawlResult) error {
	if _, exists := s.results[result.URL]; exists {
		return errors.New("tried to add a duplicate crawl result")
	}

	s.results[result.URL] = result

	return nil
}

// BuildReport creates a report from the accumulated crawl state.
func (s *CrawlState) BuildReport(rootURL string, maxDepth int) *Report {
	report := &Report{
		URL:         rootURL,
		Depth:       maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]ReportPage, 0, len(s.results)),
	}

	for _, result := range s.results {
		report.Pages = append(report.Pages, ReportPage{
			URL:          result.URL,
			Depth:        result.Depth,
			HTTPStatus:   result.HTTPStatus,
			Status:       result.Status,
			BrokenLinks:  s.brokenLinks(result),
			DiscoveredAt: result.DiscoveredAt,
		})
	}

	return report
}

func (s *CrawlState) brokenLinks(result CrawlResult) []NodeLink {
	brokenLinks := make([]NodeLink, 0)

	for _, url := range result.FoundURLs {
		target, exists := s.results[url]
		if !exists || !target.IsBroken() {
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

// WorkerPool concurrently processes crawl jobs.
type WorkerPool struct {
	ctx       context.Context
	wg        sync.WaitGroup
	jobs      chan CrawlJob
	results   chan CrawlResult
	userAgent string
	client    *http.Client
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(
	ctx context.Context,
	userAgent string,
	client *http.Client,
) *WorkerPool {
	return &WorkerPool{
		ctx:       ctx,
		jobs:      make(chan CrawlJob, 100),
		results:   make(chan CrawlResult, 100),
		userAgent: userAgent,
		client:    client,
	}
}

// Start starts the specified number of workers.
func (p *WorkerPool) Start(workerCount int) {
	for range workerCount {
		p.wg.Add(1)
		go p.runWorker()
	}
}

// Submit schedules a crawl job.
//
// It returns false if the pool context has been cancelled.
func (p *WorkerPool) Submit(job CrawlJob) bool {
	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	}
}

// Results returns completed crawl results.
func (p *WorkerPool) Results() <-chan CrawlResult {
	return p.results
}

// Stop stops the worker pool and waits for all workers to finish.
func (p *WorkerPool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

func (p *WorkerPool) runWorker() {
	defer p.wg.Done()

	for job := range p.jobs {
		result := p.process(job)

		select {
		case p.results <- result:
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *WorkerPool) process(job CrawlJob) CrawlResult {
	slog.Debug(
		"Processing URL",
		"url", job.URL,
		"depth", job.Depth,
	)

	result := CrawlResult{
		URL:          job.URL,
		Depth:        job.Depth,
		DiscoveredAt: job.DiscoveredAt,
	}

	response, err := p.doRequest(job.URL)
	if err != nil {
		result.Err = err

		return result
	}

	result.HTTPStatus = response.StatusCode
	result.Status = response.Status

	page, parseErr := htmlparser.ParsePage(response.Body)
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

func (p *WorkerPool) doRequest(url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		p.ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", p.userAgent)

	return p.client.Do(request)
}
