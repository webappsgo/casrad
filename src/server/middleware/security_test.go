// Package middleware — Tests for SecurityMiddleware and CORS.
// Covers: NewSecurityMiddleware, Headers (security header presence, HSTS only on TLS,
// CSP on non-JSON), CSRF (safe methods pass, Bearer token passes, missing token blocked,
// valid token passes then invalidated), GenerateCSRFToken uniqueness,
// CORS middleware (wildcard, specific origin, OPTIONS preflight, unknown origin).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Headers middleware ---

func TestSecurityHeadersPresent(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	handler := sm.Headers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	required := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Permissions-Policy",
	}
	for _, header := range required {
		if rr.Header().Get(header) == "" {
			t.Errorf("missing security header: %s", header)
		}
	}
}

func TestSecurityHeadersNoHSTSWithoutTLS(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	handler := sm.Headers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// No TLS on test request, so HSTS should NOT be set
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS header should not be set on plain HTTP request")
	}
}

func TestSecurityHeadersCSPOnNonJSON(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	handler := sm.Headers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Error("CSP header should be set on non-JSON request")
	}
}

func TestSecurityHeadersNoCSPForJSON(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	handler := sm.Headers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// JSON API requests should not get CSP
	if rr.Header().Get("Content-Security-Policy") != "" {
		t.Error("CSP header should not be set on JSON API request")
	}
}

// --- CSRF middleware ---

func TestCSRFSafeMethodPasses(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	called := false
	handler := sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		called = false
		req := httptest.NewRequest(method, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if !called {
			t.Errorf("CSRF should pass through safe method %s", method)
		}
	}
}

func TestCSRFBearerTokenBypasses(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	called := false
	handler := sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("CSRF should bypass when Bearer Authorization header is present")
	}
}

func TestCSRFMissingTokenBlocked(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	handler := sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/form", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("missing CSRF token: status = %d, want 403", rr.Code)
	}
}

func TestCSRFValidTokenPasses(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	token := sm.GenerateCSRFToken()
	called := false
	handler := sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/form", nil)
	req.Header.Set("X-CSRF-Token", token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !called {
		t.Errorf("valid CSRF token: status = %d, called = %v; want 200 and called", rr.Code, called)
	}
}

func TestCSRFTokenInvalidatedAfterUse(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	token := sm.GenerateCSRFToken()
	handler := sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// First use succeeds
	req1 := httptest.NewRequest(http.MethodPost, "/form", nil)
	req1.Header.Set("X-CSRF-Token", token)
	sm.CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req1)

	// Second use should fail (token invalidated)
	req2 := httptest.NewRequest(http.MethodPost, "/form", nil)
	req2.Header.Set("X-CSRF-Token", token)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Errorf("reused CSRF token: status = %d, want 403", rr2.Code)
	}
}

// --- GenerateCSRFToken ---

func TestGenerateCSRFTokenUnique(t *testing.T) {
	t.Parallel()
	sm := NewSecurityMiddleware()
	tokens := make(map[string]bool, 10)
	for i := 0; i < 10; i++ {
		tok := sm.GenerateCSRFToken()
		if tok == "" {
			t.Error("GenerateCSRFToken returned empty string")
		}
		if tokens[tok] {
			t.Errorf("duplicate CSRF token generated: %q", tok)
		}
		tokens[tok] = true
	}
}

// --- CORS middleware ---

func TestCORSWildcardOrigin(t *testing.T) {
	t.Parallel()
	handler := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS wildcard: Access-Control-Allow-Origin header missing")
	}
}

func TestCORSSpecificOriginAllowed(t *testing.T) {
	t.Parallel()
	handler := CORS([]string{"https://trusted.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://trusted.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://trusted.com" {
		t.Errorf("CORS allowed origin header = %q, want https://trusted.com",
			rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSUnknownOriginNoHeader(t *testing.T) {
	t.Parallel()
	handler := CORS([]string{"https://trusted.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS should not set Allow-Origin header for untrusted origin")
	}
}

func TestCORSPreflightReturns204(t *testing.T) {
	t.Parallel()
	handler := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS preflight status = %d, want 204", rr.Code)
	}
}
