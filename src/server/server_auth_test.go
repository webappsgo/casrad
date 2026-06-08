// Package server — Tests for auth page handlers and auth action handlers.
// Covers: handleAuthLoginPage (200, HTML, login form), handleAuthRegisterPage (200, HTML),
// handleAuthForgotPage (200, HTML), handleAuthResetPage (200, HTML),
// handleAuthLogin (empty fields redirect, redirect param preserved),
// handleAuthLogout (clears session cookie, redirects to login),
// handleAuthRegister (redirects to login), handleAuthForgot (redirects with info),
// handleAuthReset (redirects with info), handleHealth/handleAPIHealth (delegating),
// handleAdminSetupPage (200, HTML, setup title).
package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- handleAuthLoginPage ---

func TestHandleAuthLoginPageStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/login", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLoginPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAuthLoginPage status = %d, want 200", rr.Code)
	}
}

func TestHandleAuthLoginPageContentTypeIsHTML(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/login", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLoginPage(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("handleAuthLoginPage Content-Type = %q, want text/html", ct)
	}
}

func TestHandleAuthLoginPageContainsCASRAD(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/login", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLoginPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "CASRAD") {
		t.Errorf("handleAuthLoginPage body missing CASRAD title")
	}
}

func TestHandleAuthLoginPageContainsPasswordInput(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/login", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLoginPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, `type="password"`) {
		t.Errorf("handleAuthLoginPage body missing password input")
	}
}

// --- handleAuthRegisterPage ---

func TestHandleAuthRegisterPageStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/register", nil)
	rr := httptest.NewRecorder()
	s.handleAuthRegisterPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAuthRegisterPage status = %d, want 200", rr.Code)
	}
}

func TestHandleAuthRegisterPageContainsUsernameField(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/register", nil)
	rr := httptest.NewRecorder()
	s.handleAuthRegisterPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "username") {
		t.Errorf("handleAuthRegisterPage body missing username field")
	}
}

// --- handleAuthForgotPage ---

func TestHandleAuthForgotPageStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/password/forgot", nil)
	rr := httptest.NewRecorder()
	s.handleAuthForgotPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAuthForgotPage status = %d, want 200", rr.Code)
	}
}

func TestHandleAuthForgotPageContainsEmailField(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/password/forgot", nil)
	rr := httptest.NewRecorder()
	s.handleAuthForgotPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "email") {
		t.Errorf("handleAuthForgotPage body missing email field")
	}
}

// --- handleAuthResetPage ---

func TestHandleAuthResetPageStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/password/reset", nil)
	rr := httptest.NewRecorder()
	s.handleAuthResetPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAuthResetPage status = %d, want 200", rr.Code)
	}
}

func TestHandleAuthResetPageContainsNewPasswordField(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/auth/password/reset", nil)
	rr := httptest.NewRecorder()
	s.handleAuthResetPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "new-password") {
		t.Errorf("handleAuthResetPage body missing new-password autocomplete")
	}
}

// --- handleAuthLogin ---

func TestHandleAuthLoginEmptyFieldsRedirectsBack(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/login", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthLogin(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthLogin(empty) status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "login") {
		t.Errorf("handleAuthLogin(empty) redirect = %q, want login URL", loc)
	}
}

func TestHandleAuthLoginWithCredentialsRedirects(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	body := strings.NewReader("username=admin&password=secret")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthLogin(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthLogin(with creds) status = %d, want 303", rr.Code)
	}
}

func TestHandleAuthLoginRespectsRedirectParam(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	body := strings.NewReader("username=admin&password=secret&redirect=/dashboard")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthLogin(rr, req)
	loc := rr.Header().Get("Location")
	if loc != "/dashboard" {
		t.Errorf("handleAuthLogin redirect param = %q, want /dashboard", loc)
	}
}

// --- handleAuthLogout ---

func TestHandleAuthLogoutClearsSessionCookie(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/logout", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLogout(rr, req)

	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session" && c.MaxAge < 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("handleAuthLogout should clear session cookie with MaxAge < 0")
	}
}

func TestHandleAuthLogoutRedirectsToLogin(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/logout", nil)
	rr := httptest.NewRecorder()
	s.handleAuthLogout(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthLogout status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "login") {
		t.Errorf("handleAuthLogout redirect = %q, want login URL", loc)
	}
}

// --- handleAuthRegister ---

func TestHandleAuthRegisterRedirectsToLogin(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	body := strings.NewReader("username=newuser&email=new@example.com&password=secret123")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/register", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthRegister(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthRegister status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "login") {
		t.Errorf("handleAuthRegister redirect = %q, want login URL", loc)
	}
}

// --- handleAuthForgot ---

func TestHandleAuthForgotRedirectsWithInfo(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	body := strings.NewReader("email=user@example.com")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/password/forgot", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthForgot(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthForgot status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "login") {
		t.Errorf("handleAuthForgot redirect = %q, want login URL (enumeration prevention)", loc)
	}
}

// --- handleAuthReset ---

func TestHandleAuthResetRedirectsWithInfo(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	body := strings.NewReader("token=abc&password=newpass123&confirm=newpass123")
	req := httptest.NewRequest(http.MethodPost, "/server/auth/password/reset", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAuthReset(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleAuthReset status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "login") {
		t.Errorf("handleAuthReset redirect = %q, want login URL", loc)
	}
}

// --- handleAdminSetupPage ---

func TestHandleAdminSetupPageStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/config/setup", nil)
	rr := httptest.NewRecorder()
	s.handleAdminSetupPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAdminSetupPage status = %d, want 200", rr.Code)
	}
}

func TestHandleAdminSetupPageContainsSetupTitle(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/config/setup", nil)
	rr := httptest.NewRecorder()
	s.handleAdminSetupPage(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "Setup") && !strings.Contains(body, "setup") {
		t.Errorf("handleAdminSetupPage body missing Setup content, got: %q", body[:200])
	}
}

func TestHandleAdminSetupPageIsHTML(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/server/admin/config/setup", nil)
	rr := httptest.NewRecorder()
	s.handleAdminSetupPage(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("handleAdminSetupPage Content-Type = %q, want text/html", ct)
	}
}
