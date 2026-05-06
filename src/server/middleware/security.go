// Package middleware - Security middleware
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
)

// SecurityMiddleware adds security headers and CSRF protection
type SecurityMiddleware struct {
	csrfTokens map[string]bool
	mu         sync.RWMutex
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{
		csrfTokens: make(map[string]bool),
	}
}

// Headers adds security headers to responses
func (m *SecurityMiddleware) Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers per AI.md PART 11
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS header when SSL is enabled (1 year = 31536000 seconds)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy for web pages
		if r.Header.Get("Accept") != "application/json" {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' https://unpkg.com; "+
					"style-src 'self' 'unsafe-inline' https://unpkg.com; "+
					"img-src 'self' data: https:; "+
					"font-src 'self' https:; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'")
		}

		next.ServeHTTP(w, r)
	})
}

// CSRF provides CSRF token generation and validation
func (m *SecurityMiddleware) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods and API requests with Authorization header
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for API requests with Bearer token
		if r.Header.Get("Authorization") != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token for form submissions
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = r.FormValue("csrf_token")
		}

		if token == "" || !m.validateCSRFToken(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"invalid CSRF token"}` + "\n"))
			return
		}

		// Invalidate used token
		m.invalidateCSRFToken(token)

		next.ServeHTTP(w, r)
	})
}

// GenerateCSRFToken generates a new CSRF token
func (m *SecurityMiddleware) GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := base64.RawURLEncoding.EncodeToString(bytes)

	m.mu.Lock()
	m.csrfTokens[token] = true
	m.mu.Unlock()

	return token
}

// validateCSRFToken checks if a CSRF token is valid
func (m *SecurityMiddleware) validateCSRFToken(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.csrfTokens[token]
}

// invalidateCSRFToken removes a used CSRF token
func (m *SecurityMiddleware) invalidateCSRFToken(token string) {
	m.mu.Lock()
	delete(m.csrfTokens, token)
	m.mu.Unlock()
}

// CORS adds CORS headers for API requests
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
