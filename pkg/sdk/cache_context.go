package sdk

import (
	"strings"
	"sync"
	"time"
)

// CacheContext provides a generic TTL-based cache for expensive operations.
type CacheContext struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
}

type cacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// NewCacheContext creates an empty cache.
func NewCacheContext() *CacheContext {
	return &CacheContext{
		store: make(map[string]cacheEntry),
	}
}

// Get retrieves a cached value by key. Returns (value, true) if found and not expired.
func (c *CacheContext) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok {
		return nil, false
	}
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

// CacheGetTyped is a generic helper for type-safe cache retrieval.
func CacheGetTyped[T any](c *CacheContext, key string) (T, bool) {
	v, ok := c.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

// Set stores a value with the given TTL. A zero TTL means no expiration.
func (c *CacheContext) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.store[key] = cacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

// Invalidate removes a single key from the cache.
func (c *CacheContext) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

// InvalidatePrefix removes all keys with the given prefix.
func (c *CacheContext) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
		}
	}
}

// Clear removes all entries from the cache.
func (c *CacheContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]cacheEntry)
}

// Cleanup removes all expired entries. Call periodically if needed.
func (c *CacheContext) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, entry := range c.store {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			delete(c.store, k)
		}
	}
}
