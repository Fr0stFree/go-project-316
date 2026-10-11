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
)

// Fetcher performs HTTP requests with configurable retries, User-Agent headers, and rate limiting.
type Fetcher struct {
	client           *http.Client
	userAgent        string
	maxRetryAttempts int
	limiter          rateLimiter
}

// New creates a Fetcher with the specified HTTP client and default settings.
func New(client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return &Fetcher{
		client:           client,
		userAgent:        "GoCrawler/1.0",
		maxRetryAttempts: 1,
		limiter:          &noopLimiter{},
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

// Fetch performs an HTTP GET request to the specified URL.
func (h *Fetcher) Fetch(ctx context.Context, url *url.URL) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		slog.Debug(
			"Making HTTP request",
			"url", url.String(),
			"attempt", attempt+1,
		)

		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
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

		response, err := h.client.Do(request)

		shouldRetry := err != nil
		if err == nil {
			shouldRetry = isRetryableStatus(response.StatusCode)
		}

		if !shouldRetry || attempt >= h.maxRetryAttempts-1 {
			return response, err
		}

		if response != nil {
			_ = response.Body.Close()
		}
	}
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
