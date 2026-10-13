package crawler

import (
	"container/list"
	"errors"
	"net/url"
)

type crawlerState struct {
	pending list.List
	running int

	seenPages  map[string]struct{}
	seenAssets map[string]struct{}

	pages  map[string]pageTaskResult
	assets map[string]assetTaskResult
}

func newCrawlerState() *crawlerState {
	return &crawlerState{
		seenPages:  make(map[string]struct{}),
		seenAssets: make(map[string]struct{}),
		pages:      make(map[string]pageTaskResult),
		assets:     make(map[string]assetTaskResult),
	}
}

func (s *crawlerState) hasWork() bool {
	return s.pending.Len() > 0 || s.running > 0
}

func (s *crawlerState) markPageSeen(url *url.URL) bool {
	key := url.String()

	if _, exists := s.seenPages[key]; exists {
		return false
	}

	s.seenPages[key] = struct{}{}

	return true
}

func (s *crawlerState) markAssetSeen(url *url.URL) bool {
	key := url.String()

	if _, exists := s.seenAssets[key]; exists {
		return false
	}

	s.seenAssets[key] = struct{}{}

	return true
}

func (s *crawlerState) mergedResults() map[string]Result {
	results := make(map[string]Result, len(s.pages))

	for key, page := range s.pages {
		result := Result{
			URL:          page.url,
			Depth:        page.depth,
			HTTPStatus:   page.httpStatus,
			Status:       page.status,
			DiscoveredAt: page.discoveredAt,
			HTML:         page.html,
			Assets:       make([]AssetResult, 0),
			Err:          page.err,
		}

		if page.html != nil {
			for _, asset := range page.html.Assets {
				assetURL := asset.URL.String()

				item := AssetResult{
					URL:  assetURL,
					Type: asset.Type,
				}

				if fetched, ok := s.assets[assetURL]; ok {
					item.StatusCode = fetched.httpStatus
					item.SizeBytes = fetched.sizeBytes

					if fetched.err != nil {
						item.Error = fetched.err
					}
				} else {
					item.Error = errors.New("asset was not fetched")
				}

				result.Assets = append(result.Assets, item)
			}
		}

		results[key] = result
	}

	return results
}
