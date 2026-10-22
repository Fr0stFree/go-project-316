package crawler

import (
	"container/list"
	"net/url"
)

type crawlerState struct {
	pending list.List
	running int

	seenURLs   map[string]struct{}
	seenAssets map[string]struct{}

	pages  map[string]pageTaskResult
	links  map[string]linkTaskResult
	assets map[string]assetTaskResult
}

func newCrawlerState() *crawlerState {
	return &crawlerState{
		seenURLs:   make(map[string]struct{}),
		seenAssets: make(map[string]struct{}),
		pages:      make(map[string]pageTaskResult),
		links:      make(map[string]linkTaskResult),
		assets:     make(map[string]assetTaskResult),
	}
}

func (s *crawlerState) hasWork() bool {
	return s.pending.Len() > 0 || s.running > 0
}

func (s *crawlerState) markURLSeen(url *url.URL) bool {
	clonedURL := *url
	if clonedURL.Path == "" {
		clonedURL.Path = "/"
	}

	key := clonedURL.String()

	if _, exists := s.seenURLs[key]; exists {
		return false
	}

	s.seenURLs[key] = struct{}{}

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
