package engine

import (
	"sync"
	"time"
)

const probeCacheTTL = 30 * time.Second

type cachedProbe struct {
	result  *ProbeResult
	created time.Time
}

// probeCache stores recent probe results so the executor can skip re-probing
// a URL that was just probed by the frontend modal/pre-probe.
type probeCache struct {
	mu    sync.RWMutex
	items map[string]*cachedProbe
}

func newProbeCache() *probeCache {
	return &probeCache{items: make(map[string]*cachedProbe)}
}

// Put stores a probe result for the given URL.
func (pc *probeCache) Put(url string, result *ProbeResult) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.items[url] = &cachedProbe{result: result, created: time.Now()}
}

// Get returns a cached probe if it exists and is still fresh.
func (pc *probeCache) Get(url string) *ProbeResult {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	entry, ok := pc.items[url]
	if !ok || time.Since(entry.created) > probeCacheTTL {
		return nil
	}
	return entry.result
}

// Delete removes a cached probe.
func (pc *probeCache) Delete(url string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.items, url)
}
