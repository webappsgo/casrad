// Package server — Tests for pure HTTP handlers that don't require auth or DB.
// Covers: adminPath, handleRobotsTxt, handleSecurityTxt, handleAutodiscover,
// handleChangePassword (no cookie vs with cookie).
package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/config"
)

// newTestServer builds a minimal *Server for handler unit tests.
// It uses a zero-value config so no network bindings occur.
func newTestServer(adminPath, securityContact string) *Server {
	cfg := &config.Config{}
	cfg.Server.AdminPath = adminPath
	cfg.Server.SecurityContact = securityContact
	return &Server{config: cfg}
}

// --- adminPath ---

func TestAdminPathDefaultIsAdmin(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	if got := s.adminPath(); got != "admin" {
		t.Errorf("adminPath() = %q, want admin", got)
	}
}

func TestAdminPathCustomValue(t *testing.T) {
	t.Parallel()
	s := newTestServer("mgmt", "")
	if got := s.adminPath(); got != "mgmt" {
		t.Errorf("adminPath() = %q, want mgmt", got)
	}
}

// --- handleRobotsTxt ---

func TestHandleRobotsTxtStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	s.handleRobotsTxt(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleRobotsTxt status = %d, want 200", rr.Code)
	}
}

func TestHandleRobotsTxtContentType(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	s.handleRobotsTxt(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("handleRobotsTxt Content-Type = %q, want text/plain", ct)
	}
}

func TestHandleRobotsTxtContainsDisallowAdmin(t *testing.T) {
	t.Parallel()
	s := newTestServer("myadmin", "")
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	s.handleRobotsTxt(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "myadmin") {
		t.Errorf("handleRobotsTxt body = %q, should contain admin path myadmin", body)
	}
	if !strings.Contains(body, "Disallow:") {
		t.Errorf("handleRobotsTxt body missing Disallow directive")
	}
}

func TestHandleRobotsTxtDefaultAdminInDisallow(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	s.handleRobotsTxt(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "/server/admin/") {
		t.Errorf("handleRobotsTxt body = %q, should contain /server/admin/", body)
	}
}

// --- handleSecurityTxt ---

func TestHandleSecurityTxtStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	s.handleSecurityTxt(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleSecurityTxt status = %d, want 200", rr.Code)
	}
}

func TestHandleSecurityTxtContentType(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	s.handleSecurityTxt(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("handleSecurityTxt Content-Type = %q, want text/plain", ct)
	}
}

func TestHandleSecurityTxtContainsContact(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "security@myserver.example")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "myserver.example"
	rr := httptest.NewRecorder()
	s.handleSecurityTxt(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "Contact:") {
		t.Errorf("handleSecurityTxt body = %q, missing Contact field", body)
	}
	if !strings.Contains(body, "security@myserver.example") {
		t.Errorf("handleSecurityTxt body = %q, missing configured contact email", body)
	}
}

func TestHandleSecurityTxtDefaultContactFromHost(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "test.example.com"
	rr := httptest.NewRecorder()
	s.handleSecurityTxt(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "test.example.com") {
		t.Errorf("handleSecurityTxt body = %q, should derive contact from Host header", body)
	}
}

func TestHandleSecurityTxtContainsExpires(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	s.handleSecurityTxt(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "Expires:") {
		t.Errorf("handleSecurityTxt body = %q, missing Expires field", body)
	}
}

// --- handleAutodiscover ---

func TestHandleAutodiscoverStatus200(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/autodiscover", nil)
	req.Host = "myserver.example.com"
	rr := httptest.NewRecorder()
	s.handleAutodiscover(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handleAutodiscover status = %d, want 200", rr.Code)
	}
}

func TestHandleAutodiscoverContentTypeIsJSON(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/autodiscover", nil)
	req.Host = "myserver.example.com"
	rr := httptest.NewRecorder()
	s.handleAutodiscover(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("handleAutodiscover Content-Type = %q, want application/json", ct)
	}
}

func TestHandleAutodiscoverResponseContainsOk(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/api/autodiscover", nil)
	req.Host = "myserver.example.com"
	rr := httptest.NewRecorder()
	s.handleAutodiscover(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, `"ok":true`) {
		t.Errorf("handleAutodiscover body = %q, missing ok:true", body)
	}
}

// --- handleChangePassword ---

func TestHandleChangePasswordNoCookieRedirectsToForgot(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/password/change", nil)
	rr := httptest.NewRecorder()
	s.handleChangePassword(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleChangePassword(no cookie) status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "forgot") {
		t.Errorf("handleChangePassword(no cookie) redirect = %q, want forgot URL", loc)
	}
}

func TestHandleChangePasswordWithSessionCookieRedirectsToSecurity(t *testing.T) {
	t.Parallel()
	s := newTestServer("", "")
	req := httptest.NewRequest(http.MethodGet, "/password/change", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "some-session-token"})
	rr := httptest.NewRecorder()
	s.handleChangePassword(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("handleChangePassword(with cookie) status = %d, want 303", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "security") {
		t.Errorf("handleChangePassword(with cookie) redirect = %q, want security URL", loc)
	}
}
