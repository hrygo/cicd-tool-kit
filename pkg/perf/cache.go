// Package perf provides caching utilities
package perf

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// CacheItem represents an item in the cache
type CacheItem[K comparable, V any] struct {
	Key       K
	Value     V
	ExpiresAt time.Time
}

// Cache is a thread-safe LRU cache with TTL support
type Cache[K comparable, V any] struct {
	mu         sync.RWMutex
	items      map[K]*list.Element
	lru        *list.List
	maxSize    int
	ttl        time.Duration
	onEvicted  func(K, V)
	closeCh    chan struct{}
	closed     bool
	closeOnce  sync.Once
}

// NewCache creates a new cache
func NewCache[K comparable, V any](maxSize int, ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items:   make(map[K]*list.Element),
		lru:     list.New(),
		maxSize: maxSize,
		ttl:     ttl,
		closeCh: make(chan struct{}),
	}

	if ttl > 0 {
		go c.cleanupExpired()
	}

	return c
}

// Set adds or updates an item in the cache
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	var expiresAt time.Time
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	item := &CacheItem[K, V]{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}

	if elem, exists := c.items[key]; exists {
		// Update existing item
		elem.Value = item
		c.lru.MoveToFront(elem)
		return
	}

	// Add new item
	elem := c.lru.PushFront(item)
	c.items[key] = elem

	// Check if we need to evict
	if c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

// Get retrieves an item from the cache
// Returns the value and true if found, false if expired or not found
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		var zero V
		return zero, false
	}

	elem, exists := c.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	item := elem.Value.(*CacheItem[K, V])

	// Check expiration
	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		c.removeElement(elem)
		var zero V
		return zero, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)

	return item.Value, true
}

// Delete removes an item from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}
}

// Contains checks if a key exists in the cache (excluding expired items)
func (c *Cache[K, V]) Contains(key K) bool {
	_, ok := c.Get(key)
	return ok
}

// Len returns the number of items in the cache
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lru.Len()
}

// Clear removes all items from the cache
// Note: This does NOT call the onEvicted callback. If you need eviction
// notifications, iterate and Delete each item individually.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*list.Element)
	c.lru.Init()
}

// SetOnEvicted sets the callback function to be called when an item is evicted
func (c *Cache[K, V]) SetOnEvicted(fn func(K, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.onEvicted = fn
}

// Close closes the cache and stops the cleanup goroutine
// Safe to call multiple times - subsequent calls are no-ops
func (c *Cache[K, V]) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.closeCh)
		c.items = make(map[K]*list.Element)
		c.lru.Init()
		c.mu.Unlock()
	})
}

// removeElement removes an element from the cache
func (c *Cache[K, V]) removeElement(elem *list.Element) {
	c.lru.Remove(elem)
	item := elem.Value.(*CacheItem[K, V])
	delete(c.items, item.Key)

	if c.onEvicted != nil {
		c.onEvicted(item.Key, item.Value)
	}
}

// evictOldest evicts the oldest (least recently used) item
func (c *Cache[K, V]) evictOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// cleanupExpired periodically removes expired items
func (c *Cache[K, V]) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			for elem := c.lru.Back(); elem != nil; {
				item := elem.Value.(*CacheItem[K, V])
				next := elem.Prev()

				if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
					c.removeElement(elem)
				}

				elem = next
			}
			c.mu.Unlock()
		case <-c.closeCh:
			return
		}
	}
}

// Keys returns all keys in the cache
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// Range iterates over all items in the cache
func (c *Cache[K, V]) Range(fn func(K, V) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	for elem := c.lru.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*CacheItem[K, V])

		// Skip expired items
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			continue
		}

		if !fn(item.Key, item.Value) {
			break
		}
	}
}

// Stats returns cache statistics
type CacheStats struct {
	Len     int
	MaxSize int
	TTL     time.Duration
	Hits    int64
	Misses  int64
}

// StatsCache is a cache that tracks statistics
type StatsCache[K comparable, V any] struct {
	*Cache[K, V]
	hits   atomic.Int64
	misses atomic.Int64
}

// NewStatsCache creates a new cache with statistics tracking
func NewStatsCache[K comparable, V any](maxSize int, ttl time.Duration) *StatsCache[K, V] {
	return &StatsCache[K, V]{
		Cache: NewCache[K, V](maxSize, ttl),
	}
}

// Get retrieves an item and tracks statistics
func (c *StatsCache[K, V]) Get(key K) (V, bool) {
	value, ok := c.Cache.Get(key)
	if ok {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}
	return value, ok
}

// Stats returns the current cache statistics
func (c *StatsCache[K, V]) Stats() CacheStats {
	return CacheStats{
		Len:     c.Cache.Len(),
		MaxSize: c.Cache.maxSize,
		TTL:     c.Cache.ttl,
		Hits:    c.hits.Load(),
		Misses:  c.misses.Load(),
	}
}

// HitRate returns the cache hit rate (0-1)
func (c *StatsCache[K, V]) HitRate() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

// Reset resets the statistics
func (c *StatsCache[K, V]) ResetStats() {
	c.hits.Store(0)
	c.misses.Store(0)
}
