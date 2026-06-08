// Package handler — Tests for user handler unauthenticated paths and pure helpers.
// Covers: Profile, ProfileUpdate, Settings, Security, Tokens, TokenCreate, TokenDelete,
// APIMe, APIUpdateMe (all return 401 without auth), selected helper.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- selected helper ---

func TestSelectedMatchReturnsSelected(t *testing.T) {
	t.Parallel()
	got := selected("dark", "dark")
	if got != " selected" {
		t.Errorf("selected(dark,dark) = %q, want \" selected\"", got)
	}
}

func TestSelectedMismatchReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := selected("light", "dark")
	if got != "" {
		t.Errorf("selected(light,dark) = %q, want \"\"", got)
	}
}

func TestSelectedBothEmptyReturnsSelected(t *testing.T) {
	t.Parallel()
	got := selected("", "")
	if got != " selected" {
		t.Errorf("selected(\"\",\"\") = %q, want \" selected\"", got)
	}
}

// --- Profile (unauthenticated → 401) ---

func TestProfileUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()
	h.Profile(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Profile(unauth) status = %d, want 401", rr.Code)
	}
}

// --- ProfileUpdate (unauthenticated → 401) ---

func TestProfileUpdateUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader("display_name=Alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ProfileUpdate(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ProfileUpdate(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Settings (unauthenticated → 401) ---

func TestSettingsUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/users/settings", nil)
	rr := httptest.NewRecorder()
	h.Settings(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Settings(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Security (unauthenticated → 401) ---

func TestSecurityUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/users/security", nil)
	rr := httptest.NewRecorder()
	h.Security(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Security(unauth) status = %d, want 401", rr.Code)
	}
}

// --- Tokens (unauthenticated → 401) ---

func TestTokensUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/users/tokens", nil)
	rr := httptest.NewRecorder()
	h.Tokens(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Tokens(unauth) status = %d, want 401", rr.Code)
	}
}

// --- TokenCreate (unauthenticated → 401) ---

func TestTokenCreateUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/users/tokens",
		strings.NewReader(`{"name":"my-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.TokenCreate(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("TokenCreate(unauth) status = %d, want 401", rr.Code)
	}
}

// --- TokenDelete (unauthenticated → 401) ---

func TestTokenDeleteUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/users/tokens/delete", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.TokenDelete(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("TokenDelete(unauth) status = %d, want 401", rr.Code)
	}
}

// --- APIMe (unauthenticated → 401) ---

func TestAPIMeUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rr := httptest.NewRecorder()
	h.APIMe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("APIMe(unauth) status = %d, want 401", rr.Code)
	}
}

// --- APIUpdateMe (unauthenticated → 401) ---

func TestAPIUpdateMeUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me",
		strings.NewReader(`{"display_name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APIUpdateMe(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("APIUpdateMe(unauth) status = %d, want 401", rr.Code)
	}
}
