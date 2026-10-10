package httpfetcher

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
)

type Fetcher struct {
	client           *http.Client
	userAgent        string
	maxRetryAttempts int
	limiter          rateLimiter
}

func New(client *http.Client) *Fetcher {
	return &Fetcher{
		client:           client,
		userAgent:        "GoCrawler/1.0",
		maxRetryAttempts: 1,
		limiter:          &noopLimiter{},
	}
}

func (h *Fetcher) SetMaxAttempts(attempts int) error {
	h.maxRetryAttempts = attempts
	// todo: add validation
	return nil
}

func (h *Fetcher) SetUserAgent(userAgent string) error {
	h.userAgent = userAgent
	// todo: add validation
	return nil
}

func (h *Fetcher) SetRateLimiter(limiter rateLimiter) {
	h.limiter = limiter
}

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

		if err = h.limiter.Wait(ctx); err != nil {
			return nil, err
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
