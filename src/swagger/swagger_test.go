// Package swagger — Tests for Swagger UI handlers and theme CSS.
// Covers: Handler (status, Content-Type), Spec (status, Content-Type, trailing newline),
// ThemeCSS (dark default, light override, distinct outputs).
package swagger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Handler ---

func TestHandlerStatus200(t *testing.T) {
	t.Parallel()
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Handler status = %d, want 200", rr.Code)
	}
}

func TestHandlerContentTypeIsHTML(t *testing.T) {
	t.Parallel()
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Handler Content-Type = %q, want text/html", ct)
	}
}

func TestHandlerBodyContainsDoctype(t *testing.T) {
	t.Parallel()
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("Handler body missing DOCTYPE")
	}
}

func TestHandlerBodyContainsCASRAD(t *testing.T) {
	t.Parallel()
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "CASRAD") {
		t.Errorf("Handler body missing CASRAD title")
	}
}

func TestHandlerBodyContainsSwaggerUI(t *testing.T) {
	t.Parallel()
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Errorf("Handler body missing swagger-ui div")
	}
}

// --- Spec ---

func TestSpecStatus200(t *testing.T) {
	t.Parallel()
	h := Spec()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Spec status = %d, want 200", rr.Code)
	}
}

func TestSpecContentTypeIsJSON(t *testing.T) {
	t.Parallel()
	h := Spec()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Spec Content-Type = %q, want application/json", ct)
	}
}

func TestSpecBodyEndsWithNewline(t *testing.T) {
	t.Parallel()
	h := Spec()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.HasSuffix(body, "\n") {
		t.Errorf("Spec body should end with trailing newline, got %q", body[len(body)-3:])
	}
}

func TestSpecBodyContainsOpenAPIVersion(t *testing.T) {
	t.Parallel()
	h := Spec()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, `"openapi"`) {
		t.Errorf("Spec body missing openapi field")
	}
}

func TestSpecBodyContainsCASRADTitle(t *testing.T) {
	t.Parallel()
	h := Spec()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "CASRAD") {
		t.Errorf("Spec body missing CASRAD title in API info")
	}
}

// --- ThemeCSS ---

func TestThemeCSSLightReturnsDifferentOutput(t *testing.T) {
	t.Parallel()
	dark := ThemeCSS("dark")
	light := ThemeCSS("light")
	if dark == light {
		t.Error("ThemeCSS(dark) and ThemeCSS(light) should return different CSS")
	}
}

func TestThemeCSSDefaultIsDark(t *testing.T) {
	t.Parallel()
	defaultCSS := ThemeCSS("")
	darkCSS := ThemeCSS("dark")
	if defaultCSS != darkCSS {
		t.Error("ThemeCSS(\"\") should return the same CSS as ThemeCSS(\"dark\")")
	}
}

func TestThemeCSSLightContainsLightClass(t *testing.T) {
	t.Parallel()
	css := ThemeCSS("light")
	if !strings.Contains(css, "theme-light") {
		t.Errorf("ThemeCSS(light) = %q..., should contain theme-light class", css[:100])
	}
}

func TestThemeCSSDarkContainsDraculaColors(t *testing.T) {
	t.Parallel()
	css := ThemeCSS("dark")
	if !strings.Contains(css, "#282a36") {
		t.Errorf("ThemeCSS(dark) should contain Dracula background color #282a36")
	}
}

func TestThemeCSSUnknownThemeDefaultsToDark(t *testing.T) {
	t.Parallel()
	unknown := ThemeCSS("solarized")
	dark := ThemeCSS("dark")
	if unknown != dark {
		t.Error("ThemeCSS(unknown) should fall back to dark theme")
	}
}
