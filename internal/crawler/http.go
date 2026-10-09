package crawler

import (
	"code/internal/common/types"
	"context"
	"log/slog"
	"net/http"
	"time"
)

type retryConfig struct {
	maxAttempts int
	retryDelay  time.Duration
}

type httpFetcher struct {
	client    *http.Client
	userAgent string
	retry     retryConfig
}

func newHTTPFetcher(client *http.Client) *httpFetcher {
	return &httpFetcher{
		client:    client,
		userAgent: "GoCrawler/1.0",
		retry: retryConfig{
			retryDelay:  time.Second * 1,
			maxAttempts: 1,
		},
	}
}

func (h *httpFetcher) fetch(ctx context.Context, url types.URL) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			string(url),
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

		if !shouldRetry || attempt >= h.retry.maxAttempts-1 {
			return response, err
		}

		if response != nil {
			_ = response.Body.Close()
		}

		slog.Debug(
			"Retrying HTTP request",
			"url", url,
			"attempt", attempt+1,
			"delay", h.retry.retryDelay,
			"error", err,
		)

		timer := time.NewTimer(h.retry.retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()

			return nil, ctx.Err()

		case <-timer.C:
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
