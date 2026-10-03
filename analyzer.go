// Package code provides functionality for web crawling and analysis.
package code

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"code/internal/common/fmttools"
	"code/internal/crawler"
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

type workerJob struct {
	URL   string
	Depth int
}

type workerResult struct {
	PageReport PageReport
	Err        error
}

// Analyze performs a web crawling operation based on the provided options and returns a JSON report.
func Analyze(ctx context.Context, opts Options) ([]byte, error) {
	var wg sync.WaitGroup

	jobCh := make(chan workerJob, opts.Concurrency*4)
	resultCh := make(chan workerResult, opts.Concurrency)
	report := CrawlReport{
		URL:         opts.URL,
		Depth:       opts.Depth,
		GeneratedAt: time.Now().UTC(),
		Pages:       make([]PageReport, 0),
	}

	go func() {
		for range opts.Concurrency {
			go worker(ctx, &wg, jobCh, resultCh, opts)
		}
	}()

	wg.Add(1)

	go func() {
		jobCh <- workerJob{
			URL:   opts.URL,
			Depth: 0,
		}
	}()

	go func() {
		wg.Wait()
		close(resultCh)
		close(jobCh)
	}()

	for result := range resultCh {
		if result.Err != nil {
			// Handle error (e.g., log it, add to report, etc.)
			continue
		}

		report.Pages = append(report.Pages, result.PageReport)
	}

	return fmttools.ToJSON(report, opts.IndentJSON)
}

func worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs chan workerJob,
	results chan<- workerResult,
	opts Options,
) {
	for job := range jobs {
		func() {
			defer wg.Done()

			fmt.Println("Crawling URL:", job.URL, "at depth:", job.Depth)

			report := PageReport{
				URL:          job.URL,
				Depth:        job.Depth,
				BrokenLinks:  []BrokenLink{},
				DiscoveredAt: time.Now().UTC(),
			}

			response, err := crawler.MakeRequest(
				ctx,
				opts.HTTPClient,
				job.URL,
				opts.UserAgent,
			)
			if err != nil {
				results <- workerResult{
					PageReport: report,
					Err:        err,
				}

				return
			}

			err = response.Body.Close()
			if err != nil {
				results <- workerResult{
					PageReport: report,
					Err:        err,
				}

				return
			}

			page, err := htmlparser.ParsePage(response.Body)
			if err != nil {
				results <- workerResult{
					PageReport: report,
					Err:        err,
				}

				return
			}

			report.HTTPStatus = response.StatusCode
			report.Status = response.Status

			results <- workerResult{
				PageReport: report,
			}

			if job.Depth >= opts.Depth {
				return
			}

			for _, url := range page.Links {
				wg.Add(1)

				go func() {
					jobs <- workerJob{
						URL:   url,
						Depth: job.Depth + 1,
					}
				}()
			}
		}()
	}
}
