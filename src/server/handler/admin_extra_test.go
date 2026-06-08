// Package handler — Additional tests for admin handler unauthenticated redirect paths
// and authenticated paths that need no real store (Preferences, ServerSettings,
// ServerLogs, ServerBackup, ServerMetrics with JSON Accept header).
// Covers: Dashboard, Profile, Preferences, ServerSettings, ServerUsers, ServerLogs,
// ServerBackup, ServerMetrics — unauthenticated redirect, and authenticated variants.
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/server/middleware"
)

// adminContext returns a request context with admin privileges injected.
func adminContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.AdminIDKey, int64(1))
	return ctx
}

// --- Dashboard ---

func TestDashboardNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rr := httptest.NewRecorder()
	h.Dashboard(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Dashboard(no auth) status = %d, want 303", rr.Code)
	}
	if loc := rr.Header().Get("Location"); !strings.Contains(loc, "login") {
		t.Errorf("Dashboard(no auth) redirect = %q, want login", loc)
	}
}

// --- Profile ---

func TestProfileNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Profile(no auth) status = %d, want 303", rr.Code)
	}
}

// --- Preferences ---

func TestPreferencesNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/preferences", nil)
	rr := httptest.NewRecorder()
	h.Preferences(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Preferences(no auth) status = %d, want 303", rr.Code)
	}
}

func TestPreferencesAdminReturnsHTML(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/preferences", nil)
	req = req.WithContext(adminContext())
	rr := httptest.NewRecorder()
	h.Preferences(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Preferences(admin) status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Preferences Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rr.Body.String(), "Preferences") {
		t.Error("Preferences body missing 'Preferences'")
	}
}

// --- ServerSettings ---

func TestServerSettingsNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerSettings(no auth) status = %d, want 303", rr.Code)
	}
}

func TestServerSettingsAdminReturnsHTML(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	req = req.WithContext(adminContext())
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerSettings(admin) status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Settings") {
		t.Error("ServerSettings body missing 'Settings'")
	}
}

func TestServerSettingsAdminJSONResponse(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	req = req.WithContext(adminContext())
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServerSettings(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerSettings(admin, json) status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("ServerSettings Content-Type = %q, want application/json", ct)
	}
}

// --- ServerLogs ---

func TestServerLogsNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerLogs(no auth) status = %d, want 303", rr.Code)
	}
}

func TestServerLogsAdminReturnsHTML(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	req = req.WithContext(adminContext())
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerLogs(admin) status = %d, want 200", rr.Code)
	}
}

func TestServerLogsAdminJSONResponse(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/logs", nil)
	req = req.WithContext(adminContext())
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServerLogs(rr, req)
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("ServerLogs JSON Content-Type = %q, want application/json", ct)
	}
}

// --- ServerBackup ---

func TestServerBackupNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerBackup(no auth) status = %d, want 303", rr.Code)
	}
}

func TestServerBackupAdminReturnsHTML(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	req = req.WithContext(adminContext())
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerBackup(admin) status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Backup") {
		t.Error("ServerBackup body missing 'Backup'")
	}
}

func TestServerBackupAdminJSONResponse(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/backup", nil)
	req = req.WithContext(adminContext())
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServerBackup(rr, req)
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("ServerBackup JSON Content-Type = %q, want application/json", ct)
	}
}

// --- ServerMetrics ---

func TestServerMetricsNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerMetrics(no auth) status = %d, want 303", rr.Code)
	}
}

func TestServerMetricsAdminReturnsHTML(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	req = req.WithContext(adminContext())
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ServerMetrics(admin) status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Metrics") {
		t.Error("ServerMetrics body missing 'Metrics'")
	}
}

func TestServerMetricsAdminJSONResponse(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/metrics", nil)
	req = req.WithContext(adminContext())
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServerMetrics(rr, req)
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("ServerMetrics JSON Content-Type = %q, want application/json", ct)
	}
}

// --- ServerUsers no admin ---

func TestServerUsersNoAdminRedirectsToLogin(t *testing.T) {
	t.Parallel()
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/server/users", nil)
	rr := httptest.NewRecorder()
	h.ServerUsers(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ServerUsers(no auth) status = %d, want 303", rr.Code)
	}
}
