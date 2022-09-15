package caching

import lru "github.com/hashicorp/golang-lru"

type Metrics interface {
	CacheAdd(label string, cacheSize int, evicted bool)
	CacheGet(label string, hit bool)
}

// LRUCache wraps hashicorp *lru.Cache and tracks cache metrics
type LRUCache struct {
	m     Metrics
	label string
	inner *lru.Cache
}

func (c *LRUCache) Get(key any) (value any, ok bool) {
	value, ok = c.inner.Get(key)
	if c.m != nil {
		c.m.CacheGet(c.label, ok)
	}
	return value, ok
}

func (c *LRUCache) Add(key, value any) (evicted bool) {
	evicted = c.inner.Add(key, value)
	if c.m != nil {
		c.m.CacheAdd(c.label, c.inner.Len(), evicted)
	}
	return evicted
}

// NewLRUCache creates a LRU cache with the given metrics, labeling the cache adds/gets.
// Metrics are optional: no metrics will be tracked if m == nil.
func NewLRUCache(m Metrics, label string, maxSize int) *LRUCache {
	// no errors if the size is positive
	cache, _ := lru.New(maxSize)
	return &LRUCache{
		m:     m,
		label: label,
		inner: cache,
	}
}
