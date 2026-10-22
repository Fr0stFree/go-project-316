package crawler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"code/internal/common/httptools"
	"code/internal/common/timeutils"
	"code/internal/crawler/htmlparser"

	"github.com/samber/lo"
)

const (
	statusOK    = "ok"
	statusError = "error"
)

func (c *Crawler) process(ctx context.Context, t task) taskResult {
	switch t.kind {
	case pageTask:
		result := c.processWebPage(ctx, t.url, t.depth)

		return taskResult{page: &result}

	case assetTask:
		result := c.processAsset(ctx, t.url)

		return taskResult{asset: &result}

	default:
		panic("unknown task type")
	}
}

func (c *Crawler) processWebPage(ctx context.Context, url *url.URL, depth int) pageTaskResult {
	slog.DebugContext(
		ctx,
		"Processing page task...",
		"url", url,
		"depth", depth,
	)

	result := pageTaskResult{
		url:          url,
		depth:        depth,
		discoveredAt: timeutils.UTCNowPretty(c.now),
	}

	shouldCheckOnly := depth >= max(1, c.maxDepth)
	method := lo.Ternary(shouldCheckOnly, http.MethodHead, http.MethodGet)

	response, err := c.fetcher.Fetch(ctx, url, method)
	if err == nil && shouldCheckOnly && !httptools.IsMethodSupported(response.StatusCode) {
		_ = response.Body.Close()
		response, err = c.fetcher.Fetch(ctx, url, http.MethodGet)
	}

	if err != nil {
		result.err = err
		result.status = statusError

		slog.DebugContext(
			ctx,
			"Failed to fetch url",
			"url", url.String(),
			"err", err.Error(),
		)

		return result
	}

	defer func() { _ = response.Body.Close() }()

	result.httpStatus = response.StatusCode
	result.status = statusOK

	if response.StatusCode >= http.StatusBadRequest {
		result.status = statusError
	}

	if shouldCheckOnly {
		return result
	}

	html, err := htmlparser.Parse(response.Body, url)
	if err != nil {
		result.err = err
		result.status = statusError

		return result
	}

	result.html = html

	return result
}

func (c *Crawler) processAsset(
	ctx context.Context,
	assetURL *url.URL,
) assetTaskResult {
	return c.processAssetWithMethod(ctx, assetURL, http.MethodHead)
}

func (c *Crawler) processAssetWithMethod(
	ctx context.Context,
	assetURL *url.URL,
	method string,
) assetTaskResult {
	slog.DebugContext(ctx,
		"Processing asset",
		"url", assetURL.String(),
		"method", method,
	)

	result := assetTaskResult{url: assetURL}

	response, err := c.fetcher.Fetch(ctx, assetURL, method)
	if err != nil {
		result.err = err

		return result
	}

	result.httpStatus = response.StatusCode

	if method == http.MethodHead {
		sizeUnknown := response.StatusCode < http.StatusBadRequest && response.ContentLength < 0

		if !httptools.IsMethodSupported(response.StatusCode) || sizeUnknown {
			_ = response.Body.Close()

			return c.processAssetWithMethod(ctx, assetURL, http.MethodGet)
		}
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		result.err = fmt.Errorf("unexpected HTTP status: %d", response.StatusCode)

		return result
	}

	if method == http.MethodHead {
		result.sizeBytes = int(response.ContentLength)

		return result
	}

	size, err := io.Copy(io.Discard, response.Body)
	if err != nil {
		result.err = fmt.Errorf("failed to measure asset size: %w", err)

		return result
	}

	result.sizeBytes = int(size)

	return result
}
