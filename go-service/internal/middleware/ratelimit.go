// Package middleware provides HTTP middleware components for the service.
// Middleware functions wrap handlers to add cross-cutting concerns like
// rate limiting, logging, authentication, etc.
package middleware

import (
	"fmt"
	"net/http"

	"go-service/internal/ratelimiter"
)

// RateLimitMiddleware wraps an HTTP handler with rate limiting functionality.
// It uses the client's IP address as the rate limit key.
//
// Parameters:
//   - rl: The rate limiter instance to use for checking requests
//
// Returns:
//   - A middleware function that can wrap HTTP handlers
//
// The middleware adds standard rate limit headers to responses:
//   - X-RateLimit-Remaining: Approximate tokens remaining
//   - Retry-After: Seconds to wait when rate limited (on 429 response)
func RateLimitMiddleware(rl *ratelimiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP for rate limiting.
			// Uses X-Forwarded-For if behind a proxy, otherwise RemoteAddr.
			clientIP := getClientIP(r)

			// Check if the request is allowed under the rate limit
			if !rl.Allow(clientIP) {
				// Request denied - client has exceeded rate limit
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1") // Suggest retry after 1 second
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded","message":"too many requests, please try again later"}`))
				return
			}

			// Add rate limit info to response headers for client visibility.
			// This helps clients implement their own rate limiting logic.
			remaining := rl.GetRemainingTokens(clientIP)
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", remaining))

			// Request allowed - proceed to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client's IP address from the request.
// It checks common proxy headers first, then falls back to RemoteAddr.
//
// Header priority:
// 1. X-Forwarded-For (set by most proxies/load balancers)
// 2. X-Real-IP (set by some proxies like nginx)
// 3. RemoteAddr (direct connection)
//
// Note: These headers can be spoofed, so in high-security scenarios,
// consider using only RemoteAddr or validating proxy headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (most common proxy header)
	// Format: "client, proxy1, proxy2" - we want the first (client) IP
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP (used by nginx and some other proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (format: "IP:port" or "[IPv6]:port")
	// We need to strip the port number
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}

	return addr
}
