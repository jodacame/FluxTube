package extractor

import (
	"sync"
	"time"
)

// entry holds a cached resolve result and its expiry.
type entry struct {
	res *Resolved
	exp time.Time
}

// negEntry records a recent failure so we don't hammer upstream on repeated
// requests for the same broken video.
type negEntry struct {
	err   error
	until time.Time
}

// call coordinates concurrent resolves of the same video (single-flight).
type call struct {
	wg  sync.WaitGroup
	res *Resolved
	err error
}

// cache is a small TTL cache with single-flight de-duplication and a negative
// cache to minimise upstream requests.
type cache struct {
	mu      sync.Mutex
	items   map[string]entry
	neg     map[string]negEntry
	inflight map[string]*call
}

func newCache() *cache {
	return &cache{
		items:    map[string]entry{},
		neg:      map[string]negEntry{},
		inflight: map[string]*call{},
	}
}

// get returns a cached, non-expired result if present.
func (c *cache) get(id string) (*Resolved, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[id]
	if !ok || time.Now().After(e.exp) {
		return nil, false
	}
	return e.res, true
}

func (c *cache) put(id string, res *Resolved, ttl time.Duration) {
	c.mu.Lock()
	c.items[id] = entry{res: res, exp: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *cache) negGet(id string) (error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, ok := c.neg[id]
	if !ok || time.Now().After(n.until) {
		return nil, false
	}
	return n.err, true
}

func (c *cache) negPut(id string, err error, ttl time.Duration) {
	c.mu.Lock()
	c.neg[id] = negEntry{err: err, until: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// drop removes any cached state for a video.
func (c *cache) drop(id string) {
	c.mu.Lock()
	delete(c.items, id)
	delete(c.neg, id)
	c.mu.Unlock()
}
