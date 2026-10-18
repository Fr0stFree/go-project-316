// Package report provides utilities for building crawl reports from collected page results.
package report

import (
	"net/url"
	"slices"
	"strings"

	"github.com/samber/lo"

	"code/internal/common/timeutils"
	"code/internal/crawler"
	"code/internal/crawler/htmlparser"
)

// Build creates a crawl report from the collected task results.
func Build(rootURL string, maxDepth int, results map[string]crawler.Result) *Report {
	report := &Report{
		RootURL:     rootURL,
		Depth:       maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]Page, 0, len(results)),
	}

	for _, result := range results {
		if result.Depth >= maxDepth {
			continue
		}

		page := Page{
			URL:          result.URL.String(),
			Depth:        result.Depth,
			HTTPStatus:   result.HTTPStatus,
			Status:       result.Status,
			Error:        errToStr(result.Err),
			DiscoveredAt: result.DiscoveredAt,
			SEO:          buildSEO(result.HTML),
		}
		if result.Err == nil {
			page.BrokenLinks = buildBrokenLinks(result, results)
			page.Assets = buildAssets(result.Assets)
		}

		report.Pages = append(report.Pages, page)
	}

	slices.SortFunc(report.Pages, func(a, b Page) int {
		return strings.Compare(a.URL, b.URL)
	})

	return report
}

func buildSEO(html *htmlparser.ParsedHTML) SEO {
	if html == nil {
		return SEO{}
	}

	return SEO{
		HasTitle:       html.Title != nil,
		Title:          lo.FromPtr(html.Title),
		HasDescription: html.Description != nil,
		Description:    lo.FromPtr(html.Description),
		HasH1:          html.H1 != nil,
	}
}

func buildBrokenLinks(current crawler.Result, all map[string]crawler.Result) []NodeLink {
	if current.HTML == nil {
		return []NodeLink{}
	}

	return lo.FilterMap(current.HTML.Links, func(link *url.URL, _ int) (NodeLink, bool) {
		target, exists := all[link.String()]
		if !exists || (target.Err == nil && target.HTTPStatus < 400) {
			return NodeLink{}, false
		}

		return NodeLink{
			URL:        link.String(),
			StatusCode: target.HTTPStatus,
			Error:      errToStr(target.Err),
		}, true
	})
}

func buildAssets(assets []crawler.AssetResult) []Asset {
	return lo.Map(assets, func(asset crawler.AssetResult, _ int) Asset {
		return Asset{
			URL:        asset.URL,
			Type:       string(asset.Type),
			StatusCode: asset.StatusCode,
			SizeBytes:  asset.SizeBytes,
			Error:      errToStr(asset.Error),
		}
	})
}

func errToStr(err error) string {
	switch err {
	case nil:
		return ""
	default:
		return err.Error()
	}
}
