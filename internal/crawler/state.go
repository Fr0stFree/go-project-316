package crawler

import (
	"container/list"
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
	clonedURL := *url
	if clonedURL.Path == "" {
		clonedURL.Path = "/"
	}

	key := clonedURL.String()

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
