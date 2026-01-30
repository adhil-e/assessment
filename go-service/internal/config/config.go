// Package config handles application configuration management.
// It loads configuration from environment variables with sensible defaults.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values for the application.
// Using a struct allows type-safe access and easy testing with mock configs.
type Config struct {
	// Server settings
	Port string // HTTP server port

	// External service URLs
	NodeAPIBaseURL string // URL of the Node.js backend API

	// Cache settings
	CacheTTL             time.Duration // How long to cache student data
	CacheCleanupInterval time.Duration // How often to clean expired cache entries

	// Rate limiter settings
	RateLimitRate       float64       // Requests allowed per second per client
	RateLimitBurst      int           // Maximum burst size (token bucket capacity)
	RateLimitCleanup    time.Duration // How often to clean up inactive rate limit buckets
	RateLimitInactivity time.Duration // Remove buckets inactive for this duration
}

// Load reads configuration from environment variables.
// Falls back to sensible defaults for development environments.
//
// Environment variables:
//   - PORT: Server port (default: "8080")
//   - NODE_API_URL: Node.js backend URL (default: "http://localhost:5007")
//   - CACHE_TTL_SECONDS: Cache TTL in seconds (default: 300 = 5 minutes)
//   - CACHE_CLEANUP_SECONDS: Cache cleanup interval (default: 60 = 1 minute)
//   - RATE_LIMIT_PER_SECOND: Requests per second (default: 10)
//   - RATE_LIMIT_BURST: Burst size (default: 20)
//
// Returns:
//   - *Config: Populated configuration struct
func Load() *Config {
	return &Config{
		// Server configuration
		Port:           getEnvOrDefault("PORT", "8080"),
		NodeAPIBaseURL: getEnvOrDefault("NODE_API_URL", "http://localhost:5007"),

		// Cache configuration
		// Default: 5 minute TTL, cleanup every minute
		CacheTTL:             getDurationEnv("CACHE_TTL_SECONDS", 300) * time.Second,
		CacheCleanupInterval: getDurationEnv("CACHE_CLEANUP_SECONDS", 60) * time.Second,

		// Rate limiter configuration
		// Default: 10 req/s with burst of 20, cleanup every 5 minutes
		RateLimitRate:       getFloatEnv("RATE_LIMIT_PER_SECOND", 10),
		RateLimitBurst:      getIntEnv("RATE_LIMIT_BURST", 20),
		RateLimitCleanup:    getDurationEnv("RATE_LIMIT_CLEANUP_SECONDS", 300) * time.Second,
		RateLimitInactivity: getDurationEnv("RATE_LIMIT_INACTIVITY_SECONDS", 600) * time.Second,
	}
}

// getEnvOrDefault returns the environment variable value or a default.
// This is a common pattern for configuration with fallbacks.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv parses an integer from an environment variable.
// Returns the default value if the variable is not set or invalid.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getFloatEnv parses a float64 from an environment variable.
// Returns the default value if the variable is not set or invalid.
func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getDurationEnv parses an integer as seconds from an environment variable.
// Returns the default value if the variable is not set or invalid.
// The caller multiplies by time.Second to get a time.Duration.
func getDurationEnv(key string, defaultSeconds int) time.Duration {
	return time.Duration(getIntEnv(key, defaultSeconds))
}
