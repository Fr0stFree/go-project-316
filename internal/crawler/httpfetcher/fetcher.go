// Package httpfetcher provides HTTP fetching with configurable parameters
package httpfetcher

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code/internal/common/httptools"
)

// Fetcher performs HTTP requests with configurable retries, User-Agent headers, and rate limiting.
type Fetcher struct {
	client           *http.Client
	userAgent        string
	maxRetryAttempts int
	limiter          rateLimiter
	retryDelay       time.Duration
}

const (
	defaultRequestTimeout = 5 * time.Second
	defaultUserAgent      = "GoCrawler/1.0"
	defaultMaxAttempts    = 1
	defaultBufferSize     = 100
	defaultRetryDelay     = 1 * time.Second
)

// New creates a Fetcher with the specified HTTP client and default settings.
func New(client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{
			Timeout: defaultRequestTimeout,
		}
	}

	return &Fetcher{
		client:           client,
		userAgent:        defaultUserAgent,
		maxRetryAttempts: defaultMaxAttempts,
		limiter:          &noopLimiter{},
		retryDelay:       defaultRetryDelay,
	}
}

// SetMaxAttempts sets the maximum number of HTTP request attempts.
func (h *Fetcher) SetMaxAttempts(attempts int) error {
	if attempts <= 0 {
		return errors.New("max attempts must be positive")
	}

	h.maxRetryAttempts = attempts

	return nil
}

// SetUserAgent sets the User-Agent header used for HTTP requests.
func (h *Fetcher) SetUserAgent(userAgent string) error {
	if strings.TrimSpace(userAgent) == "" {
		return errors.New("user agent cannot be empty")
	}

	h.userAgent = userAgent

	return nil
}

// SetRateLimiter configures the rate limiter used between retry attempts.
func (h *Fetcher) SetRateLimiter(limiter rateLimiter) error {
	if limiter == nil {
		return errors.New("expected a limiter, got nil")
	}

	h.limiter = limiter

	return nil
}

// SetTimeout configures the timeout for every HTTP request.
func (h *Fetcher) SetTimeout(timeout time.Duration) error {
	if timeout <= time.Duration(0) {
		return errors.New("timeout cannot be negative")
	}

	h.client.Timeout = timeout

	return nil
}

// Fetch performs an HTTP request to the specified URL.
func (h *Fetcher) Fetch(ctx context.Context, url *url.URL, method string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := h.waitBeforeRetry(ctx); err != nil {
				return nil, err
			}
		}

		response, err := h.doRequest(ctx, url, method)

		if !h.shouldRetry(response, err, attempt) {
			return response, err
		}

		if response != nil {
			_ = response.Body.Close()
		}
	}
}

func (h *Fetcher) waitBeforeRetry(ctx context.Context) error {
	timer := time.NewTimer(h.retryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (h *Fetcher) doRequest(ctx context.Context, url *url.URL, method string) (*http.Response, error) {
	slog.Debug("Making HTTP request", "url", url.String())

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		url.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", h.userAgent)

	if err := h.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	return h.client.Do(request)
}

func (h *Fetcher) shouldRetry(response *http.Response, err error, attempt int) bool {
	if attempt >= h.maxRetryAttempts-1 {
		return false
	}

	if err != nil || response == nil {
		return true
	}

	return httptools.IsRetryableStatus(response.StatusCode)
}
