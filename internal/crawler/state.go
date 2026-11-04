package crawler

import (
	"container/list"
	"net/url"

	"code/internal/common/httptools"
)

type crawlerState struct {
	root    *url.URL
	pending list.List
	running int

	seenURLs   map[string]int
	pageTasks  map[string]struct{}
	seenAssets map[string]struct{}

	pages  map[string]pageTaskResult
	links  map[string]linkTaskResult
	assets map[string]assetTaskResult
}

func newCrawlerState(root *url.URL) *crawlerState {
	return &crawlerState{
		root:       root,
		seenURLs:   make(map[string]int),
		pageTasks:  make(map[string]struct{}),
		seenAssets: make(map[string]struct{}),
		pages:      make(map[string]pageTaskResult),
		links:      make(map[string]linkTaskResult),
		assets:     make(map[string]assetTaskResult),
	}
}

func (s *crawlerState) hasWork() bool {
	return s.pending.Len() > 0 || s.running > 0
}

func (s *crawlerState) recordURL(url *url.URL, depth int) (first, improved bool) {
	key := httptools.URLKey(url)

	previous, exists := s.seenURLs[key]
	if exists && previous <= depth {
		return false, false
	}

	s.seenURLs[key] = depth

	return !exists, true
}

func (s *crawlerState) minDepth(url *url.URL, fallback int) int {
	if depth, exists := s.seenURLs[httptools.URLKey(url)]; exists {
		return depth
	}

	return fallback
}

func (s *crawlerState) markPageScheduled(url *url.URL) bool {
	key := httptools.URLKey(url)
	if _, exists := s.pageTasks[key]; exists {
		return false
	}

	s.pageTasks[key] = struct{}{}

	return true
}

func (s *crawlerState) savePage(result pageTaskResult) pageTaskResult {
	s.markPageScheduled(result.url)
	s.recordURL(result.url, result.depth)
	result.depth = s.minDepth(result.url, result.depth)
	s.pages[httptools.URLKey(result.url)] = result

	return result
}

func (s *crawlerState) markAssetSeen(url *url.URL) bool {
	key := url.String()

	if _, exists := s.seenAssets[key]; exists {
		return false
	}

	s.seenAssets[key] = struct{}{}

	return true
}
