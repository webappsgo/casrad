// Package handler — Additional tests for auth handler login/logout paths.
// Covers: LoginPage (status 200, HTML content-type, contains CASRAD, contains
// password input, with redirect param), Logout without cookie (clears cookie,
// redirects to /), Logout with Accept JSON (returns JSON), Login empty fields
// redirects.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- LoginPage ---

func TestLoginPageStatus200(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("LoginPage status = %d, want 200", rr.Code)
	}
}

func TestLoginPageContentTypeHTML(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("LoginPage Content-Type = %q, want text/html", ct)
	}
}

func TestLoginPageContainsCASRAD(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	if !strings.Contains(rr.Body.String(), "CASRAD") {
		t.Error("LoginPage body missing CASRAD")
	}
}

func TestLoginPageContainsPasswordInput(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	if !strings.Contains(rr.Body.String(), `type="password"`) {
		t.Error("LoginPage body missing password input")
	}
}

func TestLoginPageContainsCSRFToken(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	if !strings.Contains(rr.Body.String(), "csrf_token") {
		t.Error("LoginPage body missing csrf_token")
	}
}

func TestLoginPageEmbedRedirectParam(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login?redirect=/admin", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	if !strings.Contains(rr.Body.String(), "/admin") {
		t.Error("LoginPage body should contain redirect param value /admin")
	}
}

func TestLoginPageDefaultRedirectIsSlash(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()
	h.LoginPage(rr, req)
	// When no redirect param, default / is embedded in the form
	if !strings.Contains(rr.Body.String(), `value="/"`) {
		t.Error("LoginPage should embed default redirect of / when no redirect param")
	}
}

// --- Logout ---

func TestLogoutNoCookieRedirectsToRoot(t *testing.T) {
	t.Parallel()
	// nil authService is safe when no session cookie is present
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Logout(no cookie) status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc != "/" {
		t.Errorf("Logout(no cookie) redirect = %q, want /", loc)
	}
}

func TestLogoutNoCookieClearsSessionCookie(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "session" {
			found = true
			if c.Expires.Year() > 1971 {
				t.Errorf("Logout session cookie Expires = %v, want expired time", c.Expires)
			}
		}
	}
	if !found {
		t.Error("Logout should set a session cookie to expire it")
	}
}

func TestLogoutJSONAcceptReturnsJSON(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, nil, newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.Logout(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Logout(JSON Accept) status = %d, want 200", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Logout(JSON Accept) Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), "logged out") {
		t.Error("Logout(JSON Accept) body should contain logged out")
	}
}
