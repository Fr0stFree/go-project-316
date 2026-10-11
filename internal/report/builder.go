// Package report provides utilities for building crawl reports from collected page results.
package report

import (
	"code/internal/common/timeutils"
	"code/internal/crawler"
	"code/internal/crawler/htmlparser"

	"github.com/samber/lo"
)

// Build creates a crawl report from the collected task results.
func Build(rootURL string, maxDepth int, results map[string]crawler.TaskResult) *Report {
	report := &Report{
		URL:         rootURL,
		Depth:       maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]Page, 0, len(results)),
	}

	for _, result := range results {
		page := Page{
			URL:          result.URL.String(),
			Depth:        result.Depth,
			HTTPStatus:   result.HTTPStatus,
			Status:       result.Status,
			DiscoveredAt: result.DiscoveredAt,
			BrokenLinks:  buildBrokenLinks(result, results),
			SEO:          buildSEO(result.HTML),
		}

		report.Pages = append(report.Pages, page)
	}

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

func buildBrokenLinks(current crawler.TaskResult, all map[string]crawler.TaskResult) []NodeLink {
	links := make([]NodeLink, 0)

	if current.HTML == nil {
		return links
	}

	for _, url := range current.HTML.Links {
		target, exists := all[url.String()]
		if !exists {
			continue
		}

		if target.Err == nil && target.HTTPStatus < 400 {
			continue
		}

		link := NodeLink{
			URL:        url.String(),
			StatusCode: target.HTTPStatus,
		}

		if target.Err != nil {
			link.Error = target.Err.Error()
		}

		links = append(links, link)
	}

	return links
}
