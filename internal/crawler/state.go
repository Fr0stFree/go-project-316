package crawler

import "net/url"

type crawlerState struct {
	seenURLs     map[string]struct{}
	results      map[string]TaskResult
	pendingTasks int
}

func newCrawlerState() *crawlerState {
	return &crawlerState{
		seenURLs: make(map[string]struct{}),
		results:  make(map[string]TaskResult),
	}
}

func (c *crawlerState) tryMarkSeen(url *url.URL) bool {
	key := url.String()
	if _, exists := c.seenURLs[key]; exists {
		return false
	}

	c.seenURLs[url.String()] = struct{}{}

	return true
}

func (c *crawlerState) completeTask(result TaskResult) {
	c.results[result.URL.String()] = result
	c.pendingTasks--
}

func (c *crawlerState) increaseTasks() {
	c.pendingTasks++
}

func (c *crawlerState) hasTasks() bool {
	return c.pendingTasks > 0
}
