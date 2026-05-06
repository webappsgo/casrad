// Package middleware - Rate limiting middleware
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter provides IP-based rate limiting
type RateLimiter struct {
	requests      map[string]*clientInfo
	mu            sync.RWMutex
	requestsLimit int
	windowSeconds int64
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
}

type clientInfo struct {
	count      int
	windowStart int64
}

// NewRateLimiter creates a new rate limiter
// requestsLimit: max requests per window
// windowSeconds: time window in seconds
func NewRateLimiter(requestsLimit int, windowSeconds int64) *RateLimiter {
	rl := &RateLimiter{
		requests:      make(map[string]*clientInfo),
		requestsLimit: requestsLimit,
		windowSeconds: windowSeconds,
		stopCh:        make(chan struct{}),
	}

	// Start cleanup goroutine
	rl.cleanupTicker = time.NewTicker(time.Minute)
	go rl.cleanup()

	return rl
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
	rl.cleanupTicker.Stop()
}

// cleanup removes expired entries
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.stopCh:
			return
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now().Unix()
			for ip, info := range rl.requests {
				if now-info.windowStart > rl.windowSeconds*2 {
					delete(rl.requests, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().Unix()

	info, exists := rl.requests[ip]
	if !exists {
		rl.requests[ip] = &clientInfo{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// Check if window has expired
	if now-info.windowStart >= rl.windowSeconds {
		info.count = 1
		info.windowStart = now
		return true
	}

	// Check if limit exceeded
	if info.count >= rl.requestsLimit {
		return false
	}

	info.count++
	return true
}

// Limit returns HTTP middleware that applies rate limiting
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if !rl.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}` + "\n"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from a request
// Handles X-Forwarded-For and X-Real-IP headers for reverse proxies
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For header (may contain multiple IPs)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// First IP in the list is the client
		ips := splitIPs(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Try X-Real-IP header
	xrip := r.Header.Get("X-Real-IP")
	if xrip != "" {
		return xrip
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// splitIPs splits a comma-separated list of IPs
func splitIPs(s string) []string {
	var ips []string
	for _, part := range []string{s} {
		for i := 0; i < len(part); i++ {
			if part[i] == ',' {
				ip := trimSpace(part[:i])
				if ip != "" {
					ips = append(ips, ip)
				}
				part = part[i+1:]
				i = -1
			}
		}
		ip := trimSpace(part)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
