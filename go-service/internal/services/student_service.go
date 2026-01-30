// Package services contains the business logic layer of the application.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-service/internal/cache"
	"go-service/internal/models"
)

// AuthInfo holds authentication data to forward to Node.js API.
// The Node.js backend uses cookie-based auth with CSRF protection.
type AuthInfo struct {
	Cookies   string // Cookie header value (accessToken, refreshToken, csrfToken)
	CSRFToken string // X-CSRF-Token header value
}

// StudentService handles fetching student data from the Node.js backend API.
type StudentService struct {
	nodeAPIBaseURL string       // Base URL of the Node.js backend API
	httpClient     *http.Client // Reusable HTTP client with connection pooling
	cache          *cache.Cache // In-memory cache for student data
}

// NewStudentService creates a new StudentService instance.
func NewStudentService(nodeAPIBaseURL string, studentCache *cache.Cache) *StudentService {
	return &StudentService{
		nodeAPIBaseURL: nodeAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		cache: studentCache,
	}
}

// cacheKey generates a consistent cache key for a student.
func cacheKey(studentID string) string {
	return "student:" + studentID
}

// FetchStudent retrieves student data, using cache when available.
// Cache-aside pattern: check cache first, fetch from API on miss.
func (s *StudentService) FetchStudent(id string, authInfo *AuthInfo) (*models.Student, error) {
	key := cacheKey(id)

	// Try to get from cache first (fast path)
	if cached, found := s.cache.Get(key); found {
		if student, ok := cached.(*models.Student); ok {
			return student, nil
		}
		s.cache.Delete(key) // Corrupted entry, remove it
	}

	// Cache miss - fetch from Node.js API
	student, err := s.fetchFromAPI(id, authInfo)
	if err != nil {
		return nil, err
	}

	// Cache the successful response
	s.cache.Set(key, student)

	return student, nil
}

// fetchFromAPI makes the HTTP request to the Node.js backend.
func (s *StudentService) fetchFromAPI(id string, authInfo *AuthInfo) (*models.Student, error) {
	url := fmt.Sprintf("%s/api/v1/students/%s", s.nodeAPIBaseURL, id)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Forward authentication to Node.js API.
	// The Node.js backend uses cookie-based auth with CSRF tokens.
	if authInfo != nil {
		if authInfo.Cookies != "" {
			req.Header.Set("Cookie", authInfo.Cookies)
		}
		if authInfo.CSRFToken != "" {
			req.Header.Set("X-CSRF-Token", authInfo.CSRFToken)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student from API: %w", err)
	}
	defer resp.Body.Close()

	// Handle HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - continue to parse
	case http.StatusNotFound:
		return nil, fmt.Errorf("student not found")
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized: please provide valid authentication")
	case http.StatusForbidden:
		return nil, fmt.Errorf("unauthorized: please provide valid authentication")
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var student models.Student
	if err := json.Unmarshal(body, &student); err != nil {
		return nil, fmt.Errorf("failed to parse student data: %w", err)
	}

	return &student, nil
}

// InvalidateCache removes a student from the cache.
func (s *StudentService) InvalidateCache(id string) {
	s.cache.Delete(cacheKey(id))
}

// ClearCache removes all students from the cache.
func (s *StudentService) ClearCache() {
	s.cache.Clear()
}
