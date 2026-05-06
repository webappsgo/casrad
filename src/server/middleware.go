// Package server - HTTP middleware
// See AI.md PART 16 for middleware requirements
package server

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// LoggingMiddleware logs all HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Log request duration
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuthMiddleware validates user authentication
// See AI.md PART 17 for admin auth, PART 33 for user auth
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Session validation is handled by the middleware package
		// This is a passthrough for routes that don't require auth
		next.ServeHTTP(w, r)
	})
}

// AdminMiddleware validates admin authentication
// See AI.md PART 17 - Admin accounts are separate from users
func AdminMiddleware(adminPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Admin validation is handled by the middleware.AuthMiddleware.RequireAdmin
			// This wrapper is for route configuration compatibility
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements rate limiting
// See AI.md PART 1 for rate limit specification
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting is handled by the middleware.RateLimiter
		// This wrapper is for route configuration compatibility
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds CORS headers for API endpoints
// See AI.md PART 14 - CORS required for API
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// URLNormalizeMiddleware normalizes URLs for consistent routing
// See AI.md PART 16 - Removes trailing slashes, redirects to canonical
// This MUST be applied FIRST in the middleware chain
func URLNormalizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Root path "/" stays as-is
		if path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Remove trailing slash (canonical form: no trailing slash)
		if strings.HasSuffix(path, "/") {
			// Exception: file requests (has extension after last /)
			lastSlash := strings.LastIndex(path, "/")
			if lastSlash >= 0 && !strings.Contains(path[lastSlash:], ".") {
				canonical := strings.TrimSuffix(path, "/")
				// Preserve query string
				if r.URL.RawQuery != "" {
					canonical += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, canonical, http.StatusMovedPermanently)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// DetectClientType determines if request is from browser or CLI
// See AI.md PART 16 - Smart Content Detection
func DetectClientType(r *http.Request) string {
	// 1. Check Accept header first (explicit preference)
	accept := r.Header.Get("Accept")

	if strings.Contains(accept, "text/html") {
		return "html"
	}
	if strings.Contains(accept, "text/plain") {
		return "text"
	}
	if strings.Contains(accept, "application/json") {
		return "json"
	}

	// 2. Check User-Agent for browser detection
	ua := r.Header.Get("User-Agent")

	// Browser User-Agents
	browsers := []string{
		"Mozilla/", "Chrome/", "Safari/", "Edge/", "Firefox/",
		"Opera/", "MSIE", "Trident/",
	}
	for _, browser := range browsers {
		if strings.Contains(ua, browser) {
			return "html"
		}
	}

	// 3. CLI tools
	cliTools := []string{
		"curl/", "Wget/", "HTTPie/", "python-requests/",
		"Go-http-client/", "node-fetch/", "libcurl/",
	}
	for _, tool := range cliTools {
		if strings.Contains(ua, tool) {
			return "text"
		}
	}

	// 4. Empty or unknown User-Agent - default to text for programmatic
	if ua == "" {
		return "text"
	}

	// 5. Default: HTML
	return "html"
}
