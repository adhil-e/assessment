// Package ratelimiter provides a token bucket rate limiting implementation.
// It is designed for high-concurrency scenarios with minimal memory allocation.
package ratelimiter

import (
	"sync"
	"time"
)

// bucket represents a token bucket for a single client (identified by key).
// Token bucket algorithm: tokens are added at a fixed rate up to a maximum.
// Each request consumes one token; requests are denied when no tokens remain.
type bucket struct {
	tokens     float64   // Current number of available tokens
	lastRefill time.Time // Last time tokens were added to the bucket
}

// RateLimiter implements a token bucket rate limiter with per-key tracking.
// It automatically cleans up inactive buckets to prevent memory leaks.
type RateLimiter struct {
	buckets        map[string]*bucket // Map of client keys to their token buckets
	mu             sync.Mutex         // Mutex for thread-safe bucket access
	rate           float64            // Tokens added per second
	maxTokens      float64            // Maximum tokens a bucket can hold (burst size)
	cleanupTicker  *time.Ticker       // Ticker for periodic cleanup
	stopCleanup    chan struct{}      // Signal to stop cleanup goroutine
	inactivityTime time.Duration      // Duration after which inactive buckets are removed
}

// Config holds the configuration for creating a new RateLimiter.
// Using a config struct makes the API cleaner and more extensible.
type Config struct {
	Rate            float64       // Requests allowed per second
	BurstSize       int           // Maximum burst size (max tokens)
	CleanupInterval time.Duration // How often to clean up stale buckets
	InactivityTime  time.Duration // Time after which inactive buckets are removed
}

// DefaultConfig returns a sensible default configuration.
// 10 requests/second with burst of 20, cleanup every 5 minutes.
func DefaultConfig() Config {
	return Config{
		Rate:            10,              // 10 requests per second
		BurstSize:       20,              // Allow bursts of up to 20 requests
		CleanupInterval: 5 * time.Minute, // Clean up every 5 minutes
		InactivityTime:  10 * time.Minute, // Remove buckets inactive for 10 minutes
	}
}

// New creates a new RateLimiter with the given configuration.
// It starts a background goroutine for cleaning up inactive buckets.
//
// Parameters:
//   - config: Configuration for the rate limiter
//
// Returns:
//   - A new RateLimiter instance ready for use
func New(config Config) *RateLimiter {
	rl := &RateLimiter{
		buckets:        make(map[string]*bucket),
		rate:           config.Rate,
		maxTokens:      float64(config.BurstSize),
		cleanupTicker:  time.NewTicker(config.CleanupInterval),
		stopCleanup:    make(chan struct{}),
		inactivityTime: config.InactivityTime,
	}

	// Start background cleanup to prevent unbounded memory growth.
	// This removes buckets for clients that haven't made requests recently.
	go rl.cleanup()

	return rl
}

// cleanup periodically removes inactive buckets to free memory.
// A bucket is considered inactive if it hasn't been accessed within inactivityTime.
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.removeInactiveBuckets()
		case <-rl.stopCleanup:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// removeInactiveBuckets removes buckets that haven't been used recently.
// Uses a two-phase approach to minimize lock contention.
func (rl *RateLimiter) removeInactiveBuckets() {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Iterate through all buckets and remove inactive ones.
	// Since we hold the lock, we can safely modify the map.
	for key, b := range rl.buckets {
		if now.Sub(b.lastRefill) > rl.inactivityTime {
			delete(rl.buckets, key)
		}
	}
}

// Allow checks if a request from the given key should be allowed.
// It implements the token bucket algorithm:
// 1. Calculate tokens to add based on time elapsed since last refill
// 2. Add tokens up to the maximum (burst size)
// 3. If at least 1 token available, consume it and allow the request
// 4. Otherwise, deny the request
//
// Parameters:
//   - key: Unique identifier for the client (e.g., IP address)
//
// Returns:
//   - true if the request is allowed, false if rate limited
//
// Thread-safe: uses mutex to protect bucket access.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create bucket for this key.
	// New clients start with a full bucket (allows initial burst).
	b, exists := rl.buckets[key]
	if !exists {
		// Create new bucket with full tokens for new clients
		rl.buckets[key] = &bucket{
			tokens:     rl.maxTokens - 1, // Subtract 1 for current request
			lastRefill: now,
		}
		return true
	}

	// Calculate how many tokens to add based on elapsed time.
	// tokens_to_add = rate * seconds_elapsed
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokensToAdd := elapsed * rl.rate

	// Refill tokens, capped at maxTokens (burst size).
	// This prevents token accumulation beyond the burst limit.
	b.tokens = min(rl.maxTokens, b.tokens+tokensToAdd)
	b.lastRefill = now

	// Check if we have at least one token available
	if b.tokens >= 1 {
		b.tokens-- // Consume one token
		return true
	}

	// No tokens available - rate limited
	return false
}

// Stop halts the cleanup goroutine.
// Should be called when the rate limiter is no longer needed.
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// GetRemainingTokens returns the current token count for a key.
// Useful for including rate limit headers in HTTP responses.
func (rl *RateLimiter) GetRemainingTokens(key string) float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		return rl.maxTokens
	}

	// Calculate current tokens including time-based refill
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	return min(rl.maxTokens, b.tokens+(elapsed*rl.rate))
}
