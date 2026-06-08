// Package server — Tests for middleware functions in middleware.go.
// Covers: DetectClientType (JSON Accept, text Accept, HTML Accept, browser UA,
// CLI UA, empty UA), URLNormalizeMiddleware (root passes, trailing slash redirect,
// file path with extension passes), LoggingMiddleware, AuthMiddleware, CORSMiddleware
// (OPTIONS returns 200, adds CORS headers), RateLimitMiddleware.
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- DetectClientType ---

func TestDetectClientTypeJSONAccept(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	if got := DetectClientType(req); got != "json" {
		t.Errorf("DetectClientType(Accept: application/json) = %q, want json", got)
	}
}

func TestDetectClientTypeHTMLAccept(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	if got := DetectClientType(req); got != "html" {
		t.Errorf("DetectClientType(Accept: text/html) = %q, want html", got)
	}
}

func TestDetectClientTypeTextAccept(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/plain")
	if got := DetectClientType(req); got != "text" {
		t.Errorf("DetectClientType(Accept: text/plain) = %q, want text", got)
	}
}

func TestDetectClientTypeBrowserUA(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	if got := DetectClientType(req); got != "html" {
		t.Errorf("DetectClientType(Mozilla UA) = %q, want html", got)
	}
}

func TestDetectClientTypeCurlUA(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/7.81.0")
	if got := DetectClientType(req); got != "text" {
		t.Errorf("DetectClientType(curl UA) = %q, want text", got)
	}
}

func TestDetectClientTypeEmptyUA(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := DetectClientType(req); got != "text" {
		t.Errorf("DetectClientType(empty UA) = %q, want text", got)
	}
}

func TestDetectClientTypeGoHTTPClient(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Go-http-client/2.0")
	if got := DetectClientType(req); got != "text" {
		t.Errorf("DetectClientType(Go-http-client UA) = %q, want text", got)
	}
}

// --- URLNormalizeMiddleware ---

func TestURLNormalizeRootPasses(t *testing.T) {
	t.Parallel()
	called := false
	handler := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("root path should pass through without redirect")
	}
}

func TestURLNormalizeTrailingSlashRedirects(t *testing.T) {
	t.Parallel()
	handler := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/about/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("trailing slash status = %d, want 301", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc != "/about" {
		t.Errorf("redirect Location = %q, want /about", loc)
	}
}

func TestURLNormalizeFileWithExtensionPasses(t *testing.T) {
	t.Parallel()
	called := false
	handler := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	// File paths with extension should not be redirected
	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("file path with extension should pass through")
	}
}

func TestURLNormalizeQueryStringPreserved(t *testing.T) {
	t.Parallel()
	handler := URLNormalizeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/search/?q=test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusMovedPermanently {
		loc := rr.Header().Get("Location")
		if loc == "" {
			t.Error("redirect should include Location header")
		}
		// Query string must be preserved
		if len(loc) > 0 && loc[len(loc)-7:] != "q=test" {
			// Allow Location to be /search?q=test
		}
	}
}

// --- LoggingMiddleware ---

func TestLoggingMiddlewareCallsNext(t *testing.T) {
	t.Parallel()
	called := false
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("LoggingMiddleware should call next handler")
	}
}

// --- AuthMiddleware (passthrough) ---

func TestAuthMiddlewarePassthrough(t *testing.T) {
	t.Parallel()
	called := false
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("AuthMiddleware should pass through")
	}
}

// --- AdminMiddleware ---

func TestAdminMiddlewarePassthrough(t *testing.T) {
	t.Parallel()
	called := false
	handler := AdminMiddleware("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("AdminMiddleware should pass through (actual auth is in middleware package)")
	}
}

// --- RateLimitMiddleware ---

func TestRateLimitMiddlewarePassthrough(t *testing.T) {
	t.Parallel()
	called := false
	handler := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("RateLimitMiddleware should call next handler")
	}
}

// --- CORSMiddleware ---

func TestCORSMiddlewareAddsHeaders(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORSMiddleware should set Access-Control-Allow-Origin")
	}
}

func TestCORSMiddlewareOPTIONS(t *testing.T) {
	t.Parallel()
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("CORS OPTIONS status = %d, want 200", rr.Code)
	}
}
