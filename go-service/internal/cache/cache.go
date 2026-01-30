// Package cache provides an in-memory caching solution with TTL support.
// It is designed to be thread-safe and efficient for concurrent access.
package cache

import (
	"sync"
	"time"
)

// item represents a single cached entry with its value and expiration time.
// Using a struct allows us to store metadata alongside the actual value.
type item struct {
	value      interface{} // The cached value (generic to support any type)
	expiration int64       // Unix timestamp when this item expires
}

// isExpired checks if the cache item has passed its expiration time.
// Returns true if the item should be evicted from the cache.
func (i *item) isExpired() bool {
	return time.Now().UnixNano() > i.expiration
}

// Cache provides a thread-safe in-memory cache with automatic expiration.
// It uses sync.RWMutex to allow concurrent reads while ensuring safe writes.
type Cache struct {
	items      map[string]*item // Storage map for cached items
	mu         sync.RWMutex     // RWMutex allows multiple readers OR one writer
	defaultTTL time.Duration    // Default time-to-live for cached items
	stopClean  chan struct{}    // Signal channel to stop the cleanup goroutine
}

// New creates a new Cache instance with the specified default TTL.
// It also starts a background goroutine that periodically cleans expired items.
//
// Parameters:
//   - defaultTTL: The default expiration duration for cached items
//   - cleanupInterval: How often to run the cleanup routine
//
// The cleanup goroutine prevents memory leaks by removing expired entries.
func New(defaultTTL, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:      make(map[string]*item),
		defaultTTL: defaultTTL,
		stopClean:  make(chan struct{}),
	}

	// Start background cleanup goroutine to evict expired items.
	// This prevents unbounded memory growth from accumulated expired entries.
	go c.startCleanup(cleanupInterval)

	return c
}

// startCleanup runs a periodic cleanup routine that removes expired items.
// It uses a ticker for consistent intervals and listens for stop signals.
func (c *Cache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopClean:
			return
		}
	}
}

// deleteExpired removes all expired items from the cache.
// It acquires a write lock since we're modifying the map.
// We collect keys first to avoid holding the lock during deletion iteration.
func (c *Cache) deleteExpired() {
	// First pass: collect expired keys with read lock
	// This minimizes write lock duration
	c.mu.RLock()
	expiredKeys := make([]string, 0)
	for key, item := range c.items {
		if item.isExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	c.mu.RUnlock()

	// Second pass: delete expired items with write lock
	// Only acquire write lock if there are items to delete
	if len(expiredKeys) > 0 {
		c.mu.Lock()
		for _, key := range expiredKeys {
			// Re-check expiration in case item was updated between passes
			if item, exists := c.items[key]; exists && item.isExpired() {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// Set stores a value in the cache with the default TTL.
// Thread-safe: acquires write lock before modifying the map.
//
// Parameters:
//   - key: Unique identifier for the cached item
//   - value: The value to cache (can be any type)
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with a custom TTL.
// Allows fine-grained control over individual item expiration.
//
// Parameters:
//   - key: Unique identifier for the cached item
//   - value: The value to cache
//   - ttl: Custom time-to-live for this specific item
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &item{
		value:      value,
		expiration: time.Now().Add(ttl).UnixNano(),
	}
}

// Get retrieves a value from the cache.
// Returns the value and a boolean indicating if the key was found and not expired.
// Thread-safe: uses read lock to allow concurrent reads.
//
// Parameters:
//   - key: The key to look up
//
// Returns:
//   - value: The cached value (nil if not found or expired)
//   - found: true if key exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if the item has expired
	// Expired items are treated as non-existent
	if item.isExpired() {
		return nil, false
	}

	return item.value, true
}

// Delete removes an item from the cache.
// Thread-safe: acquires write lock before deletion.
//
// Parameters:
//   - key: The key to delete
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache.
// Useful for cache invalidation scenarios.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a new map instead of iterating and deleting.
	// This is more efficient and allows the GC to reclaim the old map.
	c.items = make(map[string]*item)
}

// Stop halts the cleanup goroutine.
// Should be called when the cache is no longer needed to prevent goroutine leaks.
func (c *Cache) Stop() {
	close(c.stopClean)
}

// Len returns the current number of items in the cache (including expired ones).
// Useful for monitoring and debugging.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
