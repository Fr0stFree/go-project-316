package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"code/internal/common/timeutils"
	"code/internal/crawler/htmlparser"
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
		discoveredAt: timeutils.UTCNowPretty(),
	}

	response, err := c.fetcher.Fetch(ctx, url)
	if err != nil {
		result.err = err
		slog.DebugContext(
			ctx,
			"Failed to fetch url",
			"url", url.String(),
			"err", err.Error(),
		)

		return result
	}

	result.httpStatus = response.StatusCode
	result.status = response.Status

	html, err := htmlparser.Parse(response.Body, url)
	defer func() {
		_ = response.Body.Close()
	}()

	if err != nil {
		result.err = err

		return result
	}

	result.html = html

	return result
}

func (c *Crawler) processAsset(ctx context.Context, assetURL *url.URL) assetTaskResult {
	slog.DebugContext(
		ctx,
		"Processing asset task...",
		"url", assetURL.String(),
	)

	result := assetTaskResult{
		url: assetURL,
	}

	response, err := c.fetcher.Fetch(ctx, assetURL)
	if err != nil {
		slog.DebugContext(
			ctx,
			"Failed to fetch asset",
			"url", assetURL.String(),
			"err", err,
		)

		result.err = err

		return result
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result.httpStatus = response.StatusCode

	if response.StatusCode >= http.StatusBadRequest {
		result.err = fmt.Errorf("unexpected HTTP status: %d", response.StatusCode)
	}

	var size int64
	if response.ContentLength >= 0 {
		size = response.ContentLength
	} else {
		size, err = io.Copy(io.Discard, response.Body)
		if err != nil {
			result.err = err
		}
	}

	result.sizeBytes = int(size)

	if err != nil {
		result.err = errors.Join(result.err, err)
	}

	return result
}
