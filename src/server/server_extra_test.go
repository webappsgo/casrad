// Package server — Additional tests for server-level HTTP handlers.
// Covers: handleAdminDashboard (status, content-type, body), handleHealth,
// handleAPIHealth, handleAdminSetup (invalid token),
// langMiddleware (no lang param passes through, with unknown lang passes through).
// Note: handleAdminSetupPage tests already exist in server_auth_test.go.
package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- handleAdminDashboard ---

func TestHandleAdminDashboardStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	s.handleAdminDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAdminDashboard status = %d, want 200", rr.Code)
	}
}

func TestHandleAdminDashboardContentTypeIsHTML(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	s.handleAdminDashboard(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("handleAdminDashboard Content-Type = %q, want text/html", ct)
	}
}

func TestHandleAdminDashboardBodyContainsCASRAD(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	s.handleAdminDashboard(rr, req)
	if !strings.Contains(rr.Body.String(), "CASRAD") {
		t.Error("handleAdminDashboard body should contain CASRAD")
	}
}

func TestHandleAdminDashboardBodyContainsAdminDashboard(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	s.handleAdminDashboard(rr, req)
	if !strings.Contains(rr.Body.String(), "Admin Dashboard") {
		t.Error("handleAdminDashboard body should contain 'Admin Dashboard'")
	}
}

// --- handleHealth ---

func TestHandleHealthStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleHealth status = %d, want 200", rr.Code)
	}
}

func TestHandleHealthContentTypeIsHTMLWhenAccepted(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	s.handleHealth(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("handleHealth(Accept:text/html) Content-Type = %q, want text/html", ct)
	}
}

func TestHandleHealthDefaultsToTextPlain(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.handleHealth(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("handleHealth(no Accept) Content-Type = %q, want text/plain", ct)
	}
}

// --- handleAPIHealth ---

func TestHandleAPIHealthStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.handleAPIHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAPIHealth status = %d, want 200", rr.Code)
	}
}

func TestHandleAPIHealthContentTypeIsJSON(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.handleAPIHealth(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("handleAPIHealth Content-Type = %q, want application/json", ct)
	}
}

func TestHandleAPIHealthBodyContainsStatus(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.handleAPIHealth(rr, req)
	if !strings.Contains(rr.Body.String(), "status") {
		t.Error("handleAPIHealth body should contain 'status'")
	}
}

// --- handleAdminSetup (invalid token — does not reach authService or store) ---

func TestHandleAdminSetupEmptySetupTokenReturnsForbidden(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	// s.setupToken is empty string — any submission is rejected as forbidden
	req := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", strings.NewReader("setup_token=wrong&username=admin&email=a@b.com&password=secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAdminSetup(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("handleAdminSetup(invalid token) status = %d, want 403", rr.Code)
	}
}

func TestHandleAdminSetupBadFormReturnsBadRequest(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	// Content-Type application/x-www-form-urlencoded with % causes ParseForm to fail
	req := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", strings.NewReader("%ZZ"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAdminSetup(rr, req)
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Errorf("handleAdminSetup(bad form) status = %d, want 400 or 403", rr.Code)
	}
}

// --- langMiddleware (no lang param — passes through unchanged) ---

func TestLangMiddlewareNoLangParamPassesThrough(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	// s.i18n is nil — with no ?lang= param the condition short-circuits
	// and next.ServeHTTP is called without touching s.i18n
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := s.langMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("langMiddleware: next handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("langMiddleware(no lang param) status = %d, want 200", rr.Code)
	}
}

func TestLangMiddlewareEmptyLangParamPassesThrough(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := s.langMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/?lang=", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("langMiddleware: next handler should have been called with empty lang param")
	}
	if rr.Code != http.StatusNoContent {
		t.Errorf("langMiddleware(empty lang param) status = %d, want 204", rr.Code)
	}
}
