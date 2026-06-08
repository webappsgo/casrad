// Package handler — Tests for auth handler pure functions and unauthenticated paths.
// Covers: getClientIP (XFF, X-Real-IP, RemoteAddr), NewAuthHandler,
// RegisterPage disabled mode, Register disabled mode,
// Verify (always redirects), PasswordResetPage/PasswordReset without email service,
// APILogout without session, APIRegister disabled mode.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/server/middleware"
	"github.com/casapps/casrad/src/server/service"
)

// newTestSecurityMW creates a SecurityMiddleware for testing
func newTestSecurityMW() *middleware.SecurityMiddleware {
	return middleware.NewSecurityMiddleware()
}

// newTestEmailService creates an unconfigured email service for testing
func newTestEmailService() *service.EmailService {
	return service.NewEmailService(nil)
}

// --- getClientIP ---

func TestGetClientIPXForwardedFor(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	got := getClientIP(req)
	if got != "10.0.0.1" {
		t.Errorf("getClientIP(XFF) = %q, want 10.0.0.1", got)
	}
}

func TestGetClientIPXRealIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.5")
	got := getClientIP(req)
	if got != "192.168.1.5" {
		t.Errorf("getClientIP(X-Real-IP) = %q, want 192.168.1.5", got)
	}
}

func TestGetClientIPRemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := getClientIP(req)
	// httptest.NewRequest sets RemoteAddr to "192.0.2.1:1234"
	if got == "" {
		t.Error("getClientIP should return RemoteAddr when no proxy headers")
	}
}

func TestGetClientIPXFFTakesPrecedence(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "9.9.9.9")
	got := getClientIP(req)
	if got != "1.2.3.4" {
		t.Errorf("XFF should take precedence: got %q, want 1.2.3.4", got)
	}
}

// --- NewAuthHandler ---

func TestNewAuthHandlerReturnsNonNil(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	if h == nil {
		t.Error("NewAuthHandler returned nil")
	}
}

func TestNewAuthHandlerDefaultsRegistrationMode(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "")
	if h.registrationMode != "disabled" {
		t.Errorf("empty registrationMode should default to 'disabled', got %q", h.registrationMode)
	}
}

// --- RegisterPage disabled ---

func TestRegisterPageDisabledReturns404(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
	rr := httptest.NewRecorder()
	h.RegisterPage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("RegisterPage(disabled) status = %d, want 404", rr.Code)
	}
}

func TestRegisterPageEnabledReturns200(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
	rr := httptest.NewRecorder()
	h.RegisterPage(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("RegisterPage(open) status = %d, want 200", rr.Code)
	}
}

// --- Register disabled ---

func TestRegisterDisabledReturns404(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("username=alice&email=a@b.com&password=secret123&confirm_password=secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Register(disabled) status = %d, want 404", rr.Code)
	}
}

// --- Verify ---

func TestVerifyRedirects(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?code=abc123", nil)
	rr := httptest.NewRecorder()
	h.Verify(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Verify status = %d, want 303", rr.Code)
	}
}

// --- PasswordResetPage without email service ---

func TestPasswordResetPageUnconfiguredEmailReturns503(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodGet, "/auth/password/reset", nil)
	rr := httptest.NewRecorder()
	h.PasswordResetPage(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("PasswordResetPage(no email) status = %d, want 503", rr.Code)
	}
}

// --- PasswordReset without email service ---

func TestPasswordResetUnconfiguredEmailReturns503(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", strings.NewReader("email=user@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.PasswordReset(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("PasswordReset(no email) status = %d, want 503", rr.Code)
	}
}

// --- PasswordReset JSON path without email ---

func TestPasswordResetJSONUnconfiguredReturns503(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", strings.NewReader(`{"email":"user@example.com"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PasswordReset(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("PasswordReset JSON(no email) status = %d, want 503", rr.Code)
	}
}

// --- APILogout without session ---

func TestAPILogoutWithoutSessionReturns200(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.APILogout(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("APILogout(no session) status = %d, want 200", rr.Code)
	}
}

// --- APIRegister disabled ---

func TestAPIRegisterDisabledReturns404(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "disabled")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"username":"alice","email":"a@b.com","password":"secret123","confirm_password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APIRegister(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("APIRegister(disabled) status = %d, want 404", rr.Code)
	}
}

// --- APIRegister invalid JSON ---

func TestAPIRegisterInvalidJSONReturns400(t *testing.T) {
	t.Parallel()
	h := NewAuthHandler(nil, nil, newTestEmailService(), newTestSecurityMW(), "open")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader("notjson"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.APIRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("APIRegister(invalid json) status = %d, want 400", rr.Code)
	}
}

// --- NewUserHandler ---

func TestNewUserHandlerReturnsNonNil(t *testing.T) {
	t.Parallel()
	h := NewUserHandler(nil, nil)
	if h == nil {
		t.Error("NewUserHandler returned nil")
	}
}
