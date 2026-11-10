package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"code/internal/common/httptools"
	"code/internal/common/timeutils"
	"code/internal/crawler/htmlparser"
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

	case linkCheckTask:
		result := c.processLinkCheck(ctx, t.url, t.depth)

		return taskResult{link: &result}

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
		"url", httptools.SanitizeURL(url),
		"depth", depth,
	)

	result := pageTaskResult{
		url:          url,
		depth:        depth,
		discoveredAt: timeutils.UTCTruncatedToSeconds(c.now),
	}

	response, err := c.fetcher.Fetch(ctx, url, http.MethodGet)
	if err != nil {
		result.err = err
		result.status = statusError

		slog.DebugContext(
			ctx,
			"Failed to fetch url",
			"url", httptools.SanitizeURL(url),
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

	documentURL := url
	if response.Request != nil && response.Request.URL != nil {
		documentURL = response.Request.URL
	}

	if !httptools.IsSameDomain(url, documentURL) {
		slog.DebugContext(
			ctx,
			"Skipping page parsing after cross-domain redirect",
			"url", httptools.SanitizeURL(url),
			"redirect_url", httptools.SanitizeURL(documentURL),
		)

		return result
	}

	if !httptools.IsMarkupResponse(response) {
		slog.DebugContext(
			ctx,
			"Response is not HTML or XML",
			"url", httptools.SanitizeURL(url),
			"content_type", response.Header.Get("Content-Type"),
		)

		return result
	}

	body, err := httptools.DecodeMarkupResponse(response)
	if err != nil {
		result.err = err
		result.status = statusError

		return result
	}

	shouldParseXMLLinks := httptools.IsXMLResponse(response)

	html, err := htmlparser.Parse(body, documentURL, shouldParseXMLLinks)
	if err != nil {
		result.err = err
		result.status = statusError

		return result
	}

	result.html = html

	return result
}

func (c *Crawler) processLinkCheck(ctx context.Context, url *url.URL, depth int) linkTaskResult {
	slog.DebugContext(
		ctx,
		"Processing link check...",
		"url", httptools.SanitizeURL(url),
		"depth", depth,
	)

	result := linkTaskResult{url: url, depth: depth}

	response, err := c.fetcher.Fetch(ctx, url, http.MethodHead)
	if err == nil && !httptools.IsMethodSupported(response.StatusCode) {
		_ = response.Body.Close()
		response, err = c.fetcher.Fetch(ctx, url, http.MethodGet)
	}

	if err != nil {
		result.err = err

		return result
	}

	defer func() { _ = response.Body.Close() }()

	result.httpStatus = response.StatusCode

	return result
}

func (c *Crawler) processAsset(ctx context.Context, assetURL *url.URL) assetTaskResult {
	result := assetTaskResult{url: assetURL}

	slog.DebugContext(ctx, "Processing asset", "url", httptools.SanitizeURL(assetURL), "method", http.MethodHead)

	response, err := c.fetcher.Fetch(ctx, assetURL, http.MethodHead)
	if err != nil {
		result.err = err

		return result
	}

	isSizeUnknown := response.StatusCode < http.StatusBadRequest && response.ContentLength < 0

	if !httptools.IsMethodSupported(response.StatusCode) || isSizeUnknown {
		_ = response.Body.Close()

		slog.DebugContext(
			ctx,
			"Processing asset",
			"url", httptools.SanitizeURL(assetURL),
			"method", http.MethodGet,
		)

		status, size, err := c.fetcher.FetchSize(ctx, assetURL)
		result.httpStatus = status

		if err != nil {
			result.err = fmt.Errorf("failed to measure asset size: %w", err)

			return result
		}

		if status >= http.StatusBadRequest {
			result.err = fmt.Errorf("unexpected HTTP status: %d", status)

			return result
		}

		result.sizeBytes = int(size)

		return result
	}

	defer func() { _ = response.Body.Close() }()

	result.httpStatus = response.StatusCode

	if response.StatusCode >= http.StatusBadRequest {
		result.err = fmt.Errorf("unexpected HTTP status: %d", response.StatusCode)

		return result
	}

	result.sizeBytes = int(response.ContentLength)

	return result
}
