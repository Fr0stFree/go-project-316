package report

import (
	"code/internal/common/timeutils"
	"code/internal/crawler"
	"net/url"
)

func Build(rootURL string, maxDepth int, results map[*url.URL]crawler.TaskResult) *Report {
	report := &Report{
		URL:         rootURL,
		Depth:       maxDepth,
		GeneratedAt: timeutils.UTCNowPretty(),
		Pages:       make([]ReportPage, 0, len(results)),
	}

	for _, result := range results {
		if result.Err != nil || result.HTTPStatus == 0 || result.HTTPStatus >= 400 {
			continue
		}

		report.Pages = append(report.Pages, ReportPage{
			URL:          result.URL.String(),
			Depth:        result.Depth,
			HTTPStatus:   result.HTTPStatus,
			Status:       result.Status,
			BrokenLinks:  brokenLinks(result, results),
			DiscoveredAt: result.DiscoveredAt,
		})
	}

	return report
}

func brokenLinks(current crawler.TaskResult, all map[*url.URL]crawler.TaskResult) []NodeLink {
	brokenLinks := make([]NodeLink, 0)

	for _, url := range current.FoundURLs {
		target, exists := all[url]
		if !exists || target.Err == nil {
			continue
		}

		link := NodeLink{
			URL:        url.String(),
			StatusCode: target.HTTPStatus,
		}

		if target.Err != nil {
			link.Error = target.Err.Error()
		}

		brokenLinks = append(brokenLinks, link)
	}

	return brokenLinks
}
