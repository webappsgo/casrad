// Package handler - Tests for content negotiation helpers.
// Covers: isCliTool, getAPIResponseFormat, detectResponseFormat, getFrontendResponseFormat,
// stripTxtExtension with various request configurations.
package handler

import (
	"net/http"
	"testing"
)

// makeReq builds a minimal *http.Request with the given method, path, and headers.
func makeReq(path string, headers map[string]string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "http://localhost"+path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// --- isCliTool ---

func TestIsCliTool(t *testing.T) {
	t.Parallel()

	cli := []struct {
		name string
		ua   string
	}{
		{name: "casrad_cli", ua: "casrad-cli/1.0"},
		{name: "curl", ua: "curl/7.85.0"},
		{name: "wget", ua: "Wget/1.21.3"},
		{name: "httpie", ua: "httpie/3.2.1"},
		{name: "python_requests", ua: "python-requests/2.28.0"},
		{name: "go_http_client", ua: "Go-http-client/1.1"},
		{name: "empty_ua", ua: ""},
	}
	for _, tc := range cli {
		tc := tc
		t.Run("cli_"+tc.name, func(t *testing.T) {
			t.Parallel()
			r := makeReq("/", map[string]string{"User-Agent": tc.ua})
			if !isCliTool(r) {
				t.Errorf("isCliTool UA=%q should be true", tc.ua)
			}
		})
	}

	browser := []struct {
		name string
		ua   string
	}{
		{name: "firefox", ua: "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko Firefox/91.0"},
		{name: "chrome", ua: "Mozilla/5.0 Chrome/96.0 Safari/537"},
	}
	for _, tc := range browser {
		tc := tc
		t.Run("browser_not_cli_"+tc.name, func(t *testing.T) {
			t.Parallel()
			r := makeReq("/", map[string]string{"User-Agent": tc.ua})
			if isCliTool(r) {
				t.Errorf("isCliTool UA=%q should be false for browser", tc.ua)
			}
		})
	}
}

// --- getAPIResponseFormat ---

func TestGetAPIResponseFormat(t *testing.T) {
	t.Parallel()

	t.Run("txt_extension_returns_text", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/users.txt", nil)
		if got := getAPIResponseFormat(r); got != "text" {
			t.Errorf("getAPIResponseFormat .txt = %q, want text", got)
		}
	})

	t.Run("accept_text_plain_returns_text", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/users", map[string]string{"Accept": "text/plain"})
		if got := getAPIResponseFormat(r); got != "text" {
			t.Errorf("getAPIResponseFormat text/plain = %q, want text", got)
		}
	})

	t.Run("curl_returns_text", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/users", map[string]string{"User-Agent": "curl/7.85.0"})
		if got := getAPIResponseFormat(r); got != "text" {
			t.Errorf("getAPIResponseFormat curl = %q, want text", got)
		}
	})

	t.Run("browser_returns_json", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/users", map[string]string{
			"User-Agent": "Mozilla/5.0 Chrome/96",
			"Accept":     "application/json",
		})
		if got := getAPIResponseFormat(r); got != "json" {
			t.Errorf("getAPIResponseFormat browser = %q, want json", got)
		}
	})

	t.Run("default_returns_json", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/users", map[string]string{
			"User-Agent": "MyApp/1.0",
		})
		if got := getAPIResponseFormat(r); got != "json" {
			t.Errorf("getAPIResponseFormat default = %q, want json", got)
		}
	})
}

// --- detectResponseFormat ---

func TestDetectResponseFormat(t *testing.T) {
	t.Parallel()

	t.Run("txt_extension", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/page.txt", nil)
		if got := detectResponseFormat(r); got != "text/plain" {
			t.Errorf("detectResponseFormat .txt = %q, want text/plain", got)
		}
	})

	t.Run("accept_json", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/foo", map[string]string{"Accept": "application/json"})
		if got := detectResponseFormat(r); got != "application/json" {
			t.Errorf("detectResponseFormat accept json = %q, want application/json", got)
		}
	})

	t.Run("accept_text", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/foo", map[string]string{"Accept": "text/plain"})
		if got := detectResponseFormat(r); got != "text/plain" {
			t.Errorf("detectResponseFormat accept text = %q, want text/plain", got)
		}
	})

	t.Run("accept_html", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/foo", map[string]string{"Accept": "text/html"})
		if got := detectResponseFormat(r); got != "text/html" {
			t.Errorf("detectResponseFormat accept html = %q, want text/html", got)
		}
	})

	t.Run("api_path_defaults_json", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/api/v1/tracks", nil)
		if got := detectResponseFormat(r); got != "application/json" {
			t.Errorf("detectResponseFormat api default = %q, want application/json", got)
		}
	})

	t.Run("non_api_path_defaults_html", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/tracks", nil)
		if got := detectResponseFormat(r); got != "text/html" {
			t.Errorf("detectResponseFormat non-api default = %q, want text/html", got)
		}
	})
}

// --- getFrontendResponseFormat ---

func TestGetFrontendResponseFormat(t *testing.T) {
	t.Parallel()

	t.Run("accept_html", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/", map[string]string{"Accept": "text/html"})
		if got := getFrontendResponseFormat(r); got != "html" {
			t.Errorf("getFrontendResponseFormat accept html = %q, want html", got)
		}
	})

	t.Run("accept_text_plain", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/", map[string]string{"Accept": "text/plain"})
		if got := getFrontendResponseFormat(r); got != "text" {
			t.Errorf("getFrontendResponseFormat accept text = %q, want text", got)
		}
	})

	t.Run("browser_ua", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/", map[string]string{"User-Agent": "Mozilla/5.0 Firefox/91"})
		if got := getFrontendResponseFormat(r); got != "html" {
			t.Errorf("getFrontendResponseFormat browser = %q, want html", got)
		}
	})

	t.Run("curl_returns_text", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/", map[string]string{"User-Agent": "curl/7.85.0"})
		if got := getFrontendResponseFormat(r); got != "text" {
			t.Errorf("getFrontendResponseFormat curl = %q, want text", got)
		}
	})

	t.Run("unknown_defaults_html", func(t *testing.T) {
		t.Parallel()
		r := makeReq("/", map[string]string{"User-Agent": "CustomBot/1.0"})
		if got := getFrontendResponseFormat(r); got != "html" {
			t.Errorf("getFrontendResponseFormat unknown = %q, want html", got)
		}
	})
}

// --- stripTxtExtension ---

func TestStripTxtExtension(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"/page.txt", "/page"},
		{"/api/v1/users.txt", "/api/v1/users"},
		{"/page", "/page"},
		{"/page.json", "/page.json"},
		{"", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := stripTxtExtension(tc.input)
			if got != tc.want {
				t.Errorf("stripTxtExtension(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
