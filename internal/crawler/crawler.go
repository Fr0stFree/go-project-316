// Package crawler provides functionality for crawling web pages and generating reports on their structure and links.
package crawler

import (
	"context"
	"net/http"
	"time"

	"code/internal/common/timeutils"
	"code/internal/htmlparser"
)

// CrawlReport represents the result of a web crawling operation, including the root URL, depth, generation time, and a list of page reports.
type CrawlReport struct {
	URL         string       `json:"root_url"`
	Depth       int          `json:"depth"`
	GeneratedAt time.Time    `json:"generated_at"`
	Pages       []PageReport `json:"pages"`
}

// PageReport represents a report for a single page, including its URL, depth, HTTP status, and any broken links found during crawling.
type PageReport struct {
	URL          string       `json:"url"`
	Depth        int          `json:"depth"`
	HTTPStatus   int          `json:"http_status"`
	Status       string       `json:"status"`
	BrokenLinks  []BrokenLink `json:"broken_links"`
	DiscoveredAt time.Time    `json:"discovered_at"`
}

// BrokenLink represents a broken link found during the crawling process, including the URL, error message, and HTTP status code.
type BrokenLink struct {
	URL        string `json:"url"`
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
}

// Crawler is a web crawler with an HTTP client for making requests.
type Crawler struct {
	client     *http.Client
	userAgent  string
	maxRetries int
	maxDepth   int
	timeout    time.Duration
}

// New creates a new instance of Crawler with the provided HTTP client.
func New(client *http.Client) *Crawler {
	return &Crawler{
		client:     client,
		maxRetries: 1,
		maxDepth:   1,
		userAgent:  "hexlet-go-crawler",
		timeout:    30 * time.Second,
	}
}

// WithUserAgent sets the user agent for the crawler's HTTP requests.
func (c *Crawler) WithUserAgent(userAgent string) *Crawler {
	c.userAgent = userAgent

	return c
}

// WithMaxRetries sets the maximum number of retries for failed requests.
func (c *Crawler) WithMaxRetries(maxRetries int) *Crawler {
	c.maxRetries = maxRetries

	return c
}

// WithMaxDepth sets the maximum depth for the crawling operation.
func (c *Crawler) WithMaxDepth(maxDepth int) *Crawler {
	c.maxDepth = maxDepth

	return c
}

// WithTimeout sets the timeout duration for HTTP requests made by the crawler.
func (c *Crawler) WithTimeout(timeout time.Duration) *Crawler {
	c.timeout = timeout

	return c
}

// Crawl performs a web crawling operation starting from the specified URL up to the given maximum depth.
func (c *Crawler) Crawl(ctx context.Context, url string) (CrawlReport, error) {
	report := CrawlReport{
		URL:         url,
		Depth:       c.maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]PageReport, 0),
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.makeRequest(ctx, url)
	if err != nil {
		return CrawlReport{}, err
	}
	defer func() { _ = response.Body.Close() }()

	_, err = htmlparser.ParsePage(response.Body)
	if err != nil {
		return CrawlReport{}, err
	}

	pageReport := PageReport{
		URL:          url,
		Depth:        0,
		HTTPStatus:   response.StatusCode,
		Status:       response.Status,
		BrokenLinks:  make([]BrokenLink, 0),
		DiscoveredAt: timeutils.UTCNowPretty(),
	}
	report.Pages = append(report.Pages, pageReport)

	return report, nil
}

func (c *Crawler) makeRequest(ctx context.Context, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", c.userAgent)

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
