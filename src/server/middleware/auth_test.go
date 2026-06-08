// Package middleware — Tests for AuthMiddleware context helpers and middleware behavior.
// Covers: GetUserID, GetAdminID, IsAdmin, GetSessionID (empty context, populated context),
// NewAuthMiddleware, RequireAuth (unauthenticated JSON → 401, unauthenticated web → redirect),
// RequireAdmin (no session → 403 JSON, redirect web), OptionalAuth (no cookie passes through).
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casapps/casrad/src/server/service"
	"github.com/casapps/casrad/src/server/store"
)

// --- Context helpers with empty context ---

func TestGetUserIDEmptyContext(t *testing.T) {
	t.Parallel()
	if id := GetUserID(context.Background()); id != 0 {
		t.Errorf("GetUserID(empty) = %d, want 0", id)
	}
}

func TestGetAdminIDEmptyContext(t *testing.T) {
	t.Parallel()
	if id := GetAdminID(context.Background()); id != 0 {
		t.Errorf("GetAdminID(empty) = %d, want 0", id)
	}
}

func TestIsAdminEmptyContext(t *testing.T) {
	t.Parallel()
	if IsAdmin(context.Background()) {
		t.Error("IsAdmin(empty) should be false")
	}
}

func TestGetSessionIDEmptyContext(t *testing.T) {
	t.Parallel()
	if id := GetSessionID(context.Background()); id != "" {
		t.Errorf("GetSessionID(empty) = %q, want empty string", id)
	}
}

// --- Context helpers with populated context ---

func TestGetUserIDPopulated(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), UserIDKey, int64(42))
	if id := GetUserID(ctx); id != 42 {
		t.Errorf("GetUserID = %d, want 42", id)
	}
}

func TestGetAdminIDPopulated(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), AdminIDKey, int64(7))
	if id := GetAdminID(ctx); id != 7 {
		t.Errorf("GetAdminID = %d, want 7", id)
	}
}

func TestIsAdminTrue(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), IsAdminKey, true)
	if !IsAdmin(ctx) {
		t.Error("IsAdmin should be true when IsAdminKey=true")
	}
}

func TestGetSessionIDPopulated(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), SessionIDKey, "my-session-id")
	if id := GetSessionID(ctx); id != "my-session-id" {
		t.Errorf("GetSessionID = %q, want my-session-id", id)
	}
}

// --- NewAuthMiddleware ---

func TestNewAuthMiddlewareNotNil(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	if m == nil {
		t.Fatal("NewAuthMiddleware returned nil")
	}
}

// --- RequireAuth (unauthenticated) ---

func TestRequireAuthUnauthenticatedJSONReturns401(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated JSON status = %d, want 401", rr.Code)
	}
}

func TestRequireAuthUnauthenticatedWebRedirects(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should redirect (301, 302, or 303) to login
	if rr.Code < 300 || rr.Code >= 400 {
		t.Errorf("unauthenticated web status = %d, want a 3xx redirect", rr.Code)
	}
}

// --- RequireAdmin (unauthenticated) ---

func TestRequireAdminUnauthenticatedJSONReturns403(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	handler := m.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("unauthenticated admin JSON status = %d, want 403", rr.Code)
	}
}

func TestRequireAdminUnauthenticatedWebRedirects(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	handler := m.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/server/admin/settings", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code < 300 || rr.Code >= 400 {
		t.Errorf("unauthenticated admin web status = %d, want a 3xx redirect", rr.Code)
	}
}

// --- OptionalAuth ---

func TestOptionalAuthNoCookiePassesThrough(t *testing.T) {
	t.Parallel()
	ms := store.NewMemoryStore()
	auth := service.NewAuthService(ms)
	m := NewAuthMiddleware(auth, ms)
	called := false
	handler := m.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("OptionalAuth should call next even without a cookie")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
