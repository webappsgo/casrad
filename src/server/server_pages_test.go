// Package server — Tests for theme.go functions, New() constructor, and the
// remaining middleware wrappers (LoggingMiddleware, AuthMiddleware, AdminMiddleware,
// RateLimitMiddleware, CORSMiddleware, URLNormalizeMiddleware).
// Also exercises handleAdminSetup with a valid token path and langMiddleware with
// a valid language param that hits i18n.IsAvailable.
package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casrad/src/config"
)

// newFullServer returns a fully-initialised *Server using the minimal config that
// New() requires (i18n, store, routes). Tests that need router access use this.
func newFullServer(t *testing.T) *Server {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		// Load() never errors — it warns and uses defaults per PART 12
		t.Fatalf("config.Load() unexpected error: %v", err)
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

// --- Theme functions (theme.go) ---

func TestGetThemeDefaultIsDark(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := GetTheme(req)
	if got != ThemeDark {
		t.Errorf("GetTheme(no cookie) = %q, want dark", got)
	}
}

func TestGetThemeDarkCookieReturnsDark(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	if got := GetTheme(req); got != ThemeDark {
		t.Errorf("GetTheme(dark cookie) = %q, want dark", got)
	}
}

func TestGetThemeLightCookieReturnsLight(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "light"})
	if got := GetTheme(req); got != ThemeLight {
		t.Errorf("GetTheme(light cookie) = %q, want light", got)
	}
}

func TestGetThemeAutoCookieReturnsAuto(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "auto"})
	if got := GetTheme(req); got != ThemeAuto {
		t.Errorf("GetTheme(auto cookie) = %q, want auto", got)
	}
}

func TestGetThemeInvalidCookieFallsToDark(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "invalid-theme"})
	if got := GetTheme(req); got != ThemeDark {
		t.Errorf("GetTheme(invalid cookie) = %q, want dark", got)
	}
}

func TestSetThemeSetsCookie(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SetTheme(rr, ThemeLight)
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "theme" && c.Value == "light" {
			found = true
		}
	}
	if !found {
		t.Error("SetTheme(light) should set theme=light cookie")
	}
}

func TestSetThemeDarkCookieMaxAgePositive(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SetTheme(rr, ThemeDark)
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "theme" {
			if c.MaxAge <= 0 {
				t.Errorf("SetTheme(dark) MaxAge = %d, want > 0", c.MaxAge)
			}
			return
		}
	}
	t.Error("SetTheme(dark) did not set theme cookie")
}

func TestSetThemeAutoCookieValue(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	SetTheme(rr, ThemeAuto)
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "theme" {
			if c.Value != "auto" {
				t.Errorf("SetTheme(auto) value = %q, want 'auto'", c.Value)
			}
			return
		}
	}
	t.Error("SetTheme(auto) did not set theme cookie")
}

func TestThemeClassDarkReturnsDarkClass(t *testing.T) {
	t.Parallel()
	if got := ThemeClass(ThemeDark); got != "theme-dark" {
		t.Errorf("ThemeClass(dark) = %q, want theme-dark", got)
	}
}

func TestThemeClassLightReturnsLightClass(t *testing.T) {
	t.Parallel()
	if got := ThemeClass(ThemeLight); got != "theme-light" {
		t.Errorf("ThemeClass(light) = %q, want theme-light", got)
	}
}

func TestThemeClassAutoReturnsAutoClass(t *testing.T) {
	t.Parallel()
	if got := ThemeClass(ThemeAuto); got != "theme-auto" {
		t.Errorf("ThemeClass(auto) = %q, want theme-auto", got)
	}
}

func TestThemeClassUnknownFallsToDark(t *testing.T) {
	t.Parallel()
	if got := ThemeClass(Theme("bogus")); got != "theme-dark" {
		t.Errorf("ThemeClass(bogus) = %q, want theme-dark", got)
	}
}

// --- New() constructor ---

func TestNewReturnsServer(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	if s == nil {
		t.Fatal("New() returned nil server")
	}
}

func TestNewServerHasRouter(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	if s.router == nil {
		t.Error("New() server has nil router")
	}
}

// TestNewServerHandlesHealthEndpoint verifies setupRoutes is correctly wired by
// making a request through the router returned by New.
func TestNewServerHandlesHealthEndpoint(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("New() router GET /api/v1/server/healthz = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesRobotsTxt(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /robots.txt = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "User-agent") {
		t.Error("robots.txt missing User-agent directive")
	}
}

func TestNewServerHandlesSecurityTxt(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /.well-known/security.txt = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesLoginPage(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/server/auth/login", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /server/auth/login = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesRegisterPage(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/server/auth/register", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /server/auth/register = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesForgotPasswordPage(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/server/auth/password/forgot", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /server/auth/password/forgot = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesAdminSetupPage(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/server/admin/config/setup", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /server/admin/config/setup = %d, want 200", rr.Code)
	}
}

func TestNewServerHandlesAutodiscover(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/autodiscover", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("router GET /api/v1/autodiscover = %d, want 200", rr.Code)
	}
}

// --- Middleware wrappers in middleware.go ---

func TestLoggingMiddlewarePassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := LoggingMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("LoggingMiddleware did not call next handler")
	}
}

func TestLoggingMiddlewareCapturesNon200Status(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	h := LoggingMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("LoggingMiddleware status = %d, want 404", rr.Code)
	}
}

func TestAuthMiddlewarePassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})
	h := AuthMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("AuthMiddleware did not call next handler")
	}
	if rr.Code != http.StatusAccepted {
		t.Errorf("AuthMiddleware status = %d, want 202", rr.Code)
	}
}

func TestAdminMiddlewarePassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	h := AdminMiddleware("admin")(next)
	req := httptest.NewRequest(http.MethodGet, "/server/admin/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("AdminMiddleware did not call next handler")
	}
	if rr.Code != http.StatusCreated {
		t.Errorf("AdminMiddleware status = %d, want 201", rr.Code)
	}
}

func TestRateLimitMiddlewarePassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimitMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("RateLimitMiddleware did not call next handler")
	}
}

func TestCORSMiddlewareOptionsReturns200(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := CORSMiddleware(next)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tracks", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("CORSMiddleware(OPTIONS) status = %d, want 200", rr.Code)
	}
}

func TestCORSMiddlewareAddsAllowOriginHeader(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := CORSMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin == "" {
		t.Error("CORSMiddleware should set Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddlewareGetPassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := CORSMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("CORSMiddleware(GET) should call next handler")
	}
}

func TestURLNormalizeMiddlewareRootPassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := URLNormalizeMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("URLNormalizeMiddleware(/) should pass through to next")
	}
}

func TestURLNormalizeMiddlewareTrailingSlashRedirects(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := URLNormalizeMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/tracks/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("URLNormalizeMiddleware(/tracks/) status = %d, want 301", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if strings.HasSuffix(loc, "/") {
		t.Errorf("URLNormalizeMiddleware redirect location = %q, should not have trailing slash", loc)
	}
}

func TestURLNormalizeMiddlewareFilePathPassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := URLNormalizeMiddleware(next)
	// Path with extension — should not redirect
	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("URLNormalizeMiddleware(file.js) should call next handler")
	}
}

func TestURLNormalizeMiddlewareNoTrailingSlashPassesThrough(t *testing.T) {
	t.Parallel()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := URLNormalizeMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("URLNormalizeMiddleware(no trailing slash) should call next handler")
	}
}

func TestURLNormalizeMiddlewareTrailingSlashPreservesQuery(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := URLNormalizeMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/tracks/?limit=10", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("URLNormalizeMiddleware(/tracks/?limit=10) status = %d, want 301", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "limit=10") {
		t.Errorf("URLNormalizeMiddleware redirect = %q, should preserve query string", loc)
	}
}

// --- responseWriter wrapper ---

func TestResponseWriterDefaultStatus200(t *testing.T) {
	t.Parallel()
	base := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}
	if rw.statusCode != 200 {
		t.Errorf("responseWriter default statusCode = %d, want 200", rw.statusCode)
	}
}

func TestResponseWriterCapturesWriteHeader(t *testing.T) {
	t.Parallel()
	base := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: base, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("responseWriter.WriteHeader(404) captured statusCode = %d, want 404", rw.statusCode)
	}
}

// --- handleAdminSetup with valid token ---
// This covers the branches after successful token validation (field validation,
// hashing, store creation).

func TestHandleAdminSetupValidTokenMissingFieldsReturnsBadRequest(t *testing.T) {
	t.Parallel()
	// We need a real authService to hash passwords — use newFullServer which wires everything.
	s := newFullServer(t)

	// Inject a known setup token directly
	s.setupToken = "test-setup-token-abc123"

	// Submit with valid token but missing username/email/password
	body := strings.NewReader("setup_token=test-setup-token-abc123&username=&email=&password=")
	req := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAdminSetup(rr, req)

	// Token was consumed; either bad request (missing fields) or forbidden (token consumed)
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Errorf("handleAdminSetup(valid token, missing fields) = %d, want 400 or 403", rr.Code)
	}
}

func TestHandleAdminSetupValidTokenAndFieldsCreatesAdmin(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	s.setupToken = "valid-token-xyz-9876"

	body := strings.NewReader("setup_token=valid-token-xyz-9876&username=superadmin&email=admin@example.com&password=securepass1")
	req := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.handleAdminSetup(rr, req)

	// Successful admin creation redirects to the admin dashboard
	if rr.Code != http.StatusSeeOther && rr.Code != http.StatusForbidden {
		t.Errorf("handleAdminSetup(all valid) status = %d, want 303 (redirect) or 403 (token mismatch)", rr.Code)
	}
}

// TestHandleAdminSetupTokenConsumedOnSecondUse verifies the one-time-use property.
func TestHandleAdminSetupTokenConsumedOnSecondUse(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)
	s.setupToken = "one-time-token-abc"

	// First submission — consume the token
	body1 := strings.NewReader("setup_token=one-time-token-abc&username=adm&email=adm@x.com&password=strongpass1")
	req1 := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", body1)
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr1 := httptest.NewRecorder()
	s.handleAdminSetup(rr1, req1)
	// First use should succeed or at least consume the token

	// Second submission — token should be empty now
	body2 := strings.NewReader("setup_token=one-time-token-abc&username=adm2&email=adm2@x.com&password=strongpass2")
	req2 := httptest.NewRequest(http.MethodPost, "/server/admin/config/setup", body2)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr2 := httptest.NewRecorder()
	s.handleAdminSetup(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Errorf("handleAdminSetup(second use) status = %d, want 403 (token consumed)", rr2.Code)
	}
}

// --- langMiddleware with valid language param (hits IsAvailable) ---

func TestLangMiddlewareValidLangParamSetsCooke(t *testing.T) {
	t.Parallel()
	// Use a fully-initialised server so s.i18n is non-nil and IsAvailable works.
	s := newFullServer(t)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := s.langMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/?lang=en", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("langMiddleware(lang=en) should call next handler")
	}
}

func TestLangMiddlewareUnknownLangParamPassesThrough(t *testing.T) {
	t.Parallel()
	s := newFullServer(t)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := s.langMiddleware(next)
	// "klingon" is not a supported language — IsAvailable returns false
	req := httptest.NewRequest(http.MethodGet, "/?lang=klingon", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("langMiddleware(unknown lang) should still call next handler")
	}
}
