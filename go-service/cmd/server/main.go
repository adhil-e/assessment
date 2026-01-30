// Package main is the entry point for the Go PDF Report Generation Service.
// It initializes all components and starts the HTTP server.
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"

	"go-service/internal/cache"
	"go-service/internal/config"
	"go-service/internal/handlers"
	"go-service/internal/middleware"
	"go-service/internal/ratelimiter"
	"go-service/internal/services"
)

func main() {
	// Load configuration from environment variables.
	// This centralizes all config and provides sensible defaults.
	cfg := config.Load()

	// Initialize the cache for storing student data.
	// Cache reduces load on the Node.js backend and improves response times.
	// Parameters: TTL (how long items live), cleanup interval (how often to evict expired items)
	studentCache := cache.New(cfg.CacheTTL, cfg.CacheCleanupInterval)

	// Initialize the rate limiter to protect against abuse.
	// Uses token bucket algorithm for smooth rate limiting with burst support.
	rateLimiter := ratelimiter.New(ratelimiter.Config{
		Rate:            cfg.RateLimitRate,       // Tokens added per second
		BurstSize:       cfg.RateLimitBurst,      // Maximum tokens (burst capacity)
		CleanupInterval: cfg.RateLimitCleanup,    // Cleanup old buckets interval
		InactivityTime:  cfg.RateLimitInactivity, // Remove inactive client buckets after this time
	})

	// Initialize services.
	// Services contain business logic and coordinate between handlers and external systems.
	studentService := services.NewStudentService(cfg.NodeAPIBaseURL, studentCache)
	pdfService := services.NewPDFService()

	// Initialize HTTP handlers.
	// Handlers translate HTTP requests to service calls and format responses.
	reportHandler := handlers.NewReportHandler(studentService, pdfService)

	// Setup HTTP router with gorilla/mux.
	// Mux provides powerful routing with path variables, method matching, etc.
	router := mux.NewRouter()

	// Health check endpoint - excluded from rate limiting.
	// Health checks are used by load balancers and shouldn't be rate limited.
	router.HandleFunc("/health", reportHandler.HealthCheck).Methods(http.MethodGet)

	// API v1 routes with rate limiting.
	// The subrouter groups related routes and allows middleware to be applied to the group.
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Apply rate limiting middleware to all API v1 routes.
	// This protects the service from abuse and ensures fair usage.
	apiV1.Use(middleware.RateLimitMiddleware(rateLimiter))

	// Register the student report endpoint.
	// GET /api/v1/students/{id}/report - generates and downloads a PDF report
	apiV1.HandleFunc("/students/{id}/report", reportHandler.GetStudentReport).Methods(http.MethodGet)

	// Apply CORS middleware to the entire router.
	// CORS headers are needed for browser-based API access.
	handler := corsMiddleware(router)

	// Setup graceful shutdown handling.
	// This ensures cleanup of resources (cache, rate limiter) on shutdown.
	go handleShutdown(studentCache, rateLimiter)

	// Log startup information for debugging and confirmation.
	log.Printf("=== Go PDF Report Generation Service ===")
	log.Printf("Server port: %s", cfg.Port)
	log.Printf("Node.js API URL: %s", cfg.NodeAPIBaseURL)
	log.Printf("Cache TTL: %v", cfg.CacheTTL)
	log.Printf("Rate limit: %.0f req/s (burst: %d)", cfg.RateLimitRate, cfg.RateLimitBurst)
	log.Printf("Endpoints:")
	log.Printf("  GET /health - Health check (no rate limit)")
	log.Printf("  GET /api/v1/students/{id}/report - Generate PDF report")
	log.Printf("=========================================")

	// Start the HTTP server.
	// ListenAndServe blocks until the server shuts down or encounters an error.
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// corsMiddleware adds CORS headers to allow cross-origin requests.
// This is necessary when the API is called from a different domain (e.g., frontend app).
//
// Parameters:
//   - next: The next handler in the chain
//
// Returns:
//   - http.Handler: A handler that adds CORS headers before calling next
//
// CORS headers set:
//   - Access-Control-Allow-Origin: * (allows any origin - restrict in production)
//   - Access-Control-Allow-Methods: GET, OPTIONS
//   - Access-Control-Allow-Headers: Content-Type, Authorization
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers on every response
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, Cookie")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS requests.
		// Browsers send OPTIONS before actual requests to check CORS policy.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proceed to the actual handler
		next.ServeHTTP(w, r)
	})
}

// handleShutdown listens for OS signals and performs graceful cleanup.
// This ensures resources are properly released when the service is stopped.
//
// Parameters:
//   - studentCache: Cache instance to stop cleanup goroutine
//   - rateLimiter: Rate limiter instance to stop cleanup goroutine
//
// Handles signals:
//   - SIGINT (Ctrl+C)
//   - SIGTERM (kill command, container orchestrators)
func handleShutdown(studentCache *cache.Cache, rateLimiter *ratelimiter.RateLimiter) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Register for SIGINT and SIGTERM signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Stop background goroutines to prevent resource leaks
	studentCache.Stop()
	rateLimiter.Stop()

	log.Println("Cleanup complete, exiting")
	os.Exit(0)
}
