// Package middleware — Tests for RateLimiter and getClientIP.
// Covers: NewRateLimiter, Allow (first request, under limit, at limit, window reset),
// Stop, Limit HTTP middleware, getClientIP (X-Forwarded-For, X-Real-IP, RemoteAddr),
// splitIPs, trimSpace.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- NewRateLimiter ---

func TestNewRateLimiterNotNil(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(10, 60)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	rl.Stop()
}

// --- Allow ---

func TestAllowFirstRequest(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(5, 60)
	defer rl.Stop()
	if !rl.Allow("1.2.3.4") {
		t.Error("Allow(first) should return true")
	}
}

func TestAllowUnderLimit(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(5, 60)
	defer rl.Stop()
	for i := 0; i < 4; i++ {
		if !rl.Allow("10.0.0.1") {
			t.Errorf("Allow(%d) should return true", i+1)
		}
	}
}

func TestAllowAtLimitBlocked(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(3, 60)
	defer rl.Stop()
	for i := 0; i < 3; i++ {
		rl.Allow("192.168.1.1")
	}
	if rl.Allow("192.168.1.1") {
		t.Error("Allow after limit should return false")
	}
}

func TestAllowDifferentIPsIndependent(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(2, 60)
	defer rl.Stop()
	rl.Allow("10.0.0.1")
	rl.Allow("10.0.0.1")
	// 10.0.0.1 is now at limit; 10.0.0.2 should still be allowed
	if !rl.Allow("10.0.0.2") {
		t.Error("Different IP should not be affected by other IP's limit")
	}
}

func TestAllowWindowExpiry(t *testing.T) {
	t.Parallel()
	// Use a 1-second window so we can test expiry in a reasonable time
	rl := NewRateLimiter(2, 1)
	defer rl.Stop()
	rl.Allow("172.16.0.1")
	rl.Allow("172.16.0.1")
	// Exhausted the limit
	if rl.Allow("172.16.0.1") {
		t.Error("should be blocked before window expiry")
	}
	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)
	if !rl.Allow("172.16.0.1") {
		t.Error("should be allowed after window expiry")
	}
}

// --- Limit HTTP middleware ---

func TestLimitMiddlewareAllows(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(100, 60)
	defer rl.Stop()
	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestLimitMiddlewareBlocks(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 60)
	defer rl.Stop()
	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "9.9.9.9:1234"
	httptest.NewRecorder()
	rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "9.9.9.9:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want 429", rr2.Code)
	}
	if rr2.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header missing on rate-limited response")
	}
}

// --- getClientIP ---

func TestGetClientIPXForwardedFor(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	req.RemoteAddr = "172.16.0.1:9000"
	ip := getClientIP(req)
	if ip != "203.0.113.1" {
		t.Errorf("getClientIP = %q, want 203.0.113.1", ip)
	}
}

func TestGetClientIPXRealIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.5")
	req.RemoteAddr = "172.16.0.1:9000"
	ip := getClientIP(req)
	if ip != "198.51.100.5" {
		t.Errorf("getClientIP = %q, want 198.51.100.5", ip)
	}
}

func TestGetClientIPRemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:4567"
	ip := getClientIP(req)
	if ip != "192.0.2.10" {
		t.Errorf("getClientIP = %q, want 192.0.2.10", ip)
	}
}

func TestGetClientIPXForwardedForPrecedence(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	req.Header.Set("X-Real-IP", "2.2.2.2")
	ip := getClientIP(req)
	// X-Forwarded-For takes precedence
	if ip != "1.1.1.1" {
		t.Errorf("getClientIP = %q, want 1.1.1.1 (X-Forwarded-For priority)", ip)
	}
}

// --- splitIPs ---

func TestSplitIPsSingle(t *testing.T) {
	t.Parallel()
	result := splitIPs("10.0.0.1")
	if len(result) != 1 || result[0] != "10.0.0.1" {
		t.Errorf("splitIPs = %v, want [10.0.0.1]", result)
	}
}

func TestSplitIPsMultiple(t *testing.T) {
	t.Parallel()
	result := splitIPs("10.0.0.1, 10.0.0.2, 10.0.0.3")
	if len(result) != 3 {
		t.Errorf("splitIPs len = %d, want 3", len(result))
	}
	if result[0] != "10.0.0.1" {
		t.Errorf("splitIPs[0] = %q, want 10.0.0.1", result[0])
	}
}

func TestSplitIPsEmpty(t *testing.T) {
	t.Parallel()
	result := splitIPs("")
	if len(result) != 0 {
		t.Errorf("splitIPs(\"\") = %v, want empty", result)
	}
}

// --- trimSpace ---

func TestTrimSpaceLeadingTrailing(t *testing.T) {
	t.Parallel()
	if got := trimSpace("  hello  "); got != "hello" {
		t.Errorf("trimSpace = %q, want hello", got)
	}
}

func TestTrimSpaceNoOp(t *testing.T) {
	t.Parallel()
	if got := trimSpace("hello"); got != "hello" {
		t.Errorf("trimSpace = %q, want hello", got)
	}
}

func TestTrimSpaceEmpty(t *testing.T) {
	t.Parallel()
	if got := trimSpace(""); got != "" {
		t.Errorf("trimSpace(\"\") = %q, want empty", got)
	}
}
