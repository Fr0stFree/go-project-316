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
	client     *http.Client
	userAgent  string
	maxRetries int
	limiter    rateLimiter
	retryDelay time.Duration
}

const (
	defaultRequestTimeout = 5 * time.Second
	defaultUserAgent      = "GoCrawler/1.0"
	defaultBufferSize     = 100
	defaultRetryDelay     = 1 * time.Second
	defaultMaxRetries     = 0
)

// New creates a Fetcher with the specified HTTP client and default settings.
func New(client *http.Client) *Fetcher {
	customClient := &http.Client{
		Timeout: defaultRequestTimeout,
	}

	if client != nil {
		buf := *client
		customClient = &buf
	}

	return &Fetcher{
		client:     customClient,
		userAgent:  defaultUserAgent,
		maxRetries: defaultMaxRetries,
		limiter:    &noopLimiter{},
		retryDelay: defaultRetryDelay,
	}
}

// SetRetries sets the number of retries after the first HTTP request.
func (h *Fetcher) SetRetries(retries int) error {
	if retries < 0 {
		return errors.New("retries cannot be negative")
	}

	h.maxRetries = retries

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

		if !h.shouldRetry(ctx, response, err, attempt) {
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

func (h *Fetcher) shouldRetry(ctx context.Context, response *http.Response, err error, attempt int) bool {
	if attempt >= h.maxRetries || ctx.Err() != nil {
		return false
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return false
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}

		return httptools.IsRetryableError(err)
	}

	return response != nil && httptools.IsRetryableStatus(response.StatusCode)
}
