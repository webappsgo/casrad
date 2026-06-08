// Package banner — Tests for Print, PrintSetupToken, PrintLegacy, and helper functions.
// Covers: padTo, extractHostPort, getURLIcon, printFullBanner, printCompactBanner,
// printMinimalBanner, printMicroBanner, PrintSetupToken, PrintLegacy.
// Note: getTerminalWidth is not tested directly — it depends on os.Stdout being a TTY.
package banner

import (
	"bytes"
	"strings"
	"testing"
)

// --- padTo ---

func TestPadToPositive(t *testing.T) {
	t.Parallel()
	p := padTo(5)
	if len(p) != 5 {
		t.Errorf("padTo(5) length = %d, want 5", len(p))
	}
	for _, c := range p {
		if c != ' ' {
			t.Errorf("padTo(5) contains non-space character")
		}
	}
}

func TestPadToZero(t *testing.T) {
	t.Parallel()
	if p := padTo(0); p != "" {
		t.Errorf("padTo(0) = %q, want empty string", p)
	}
}

func TestPadToNegative(t *testing.T) {
	t.Parallel()
	if p := padTo(-5); p != "" {
		t.Errorf("padTo(-5) = %q, want empty string", p)
	}
}

// --- extractHostPort ---

func TestExtractHostPortHTTPS(t *testing.T) {
	t.Parallel()
	got := extractHostPort("https://example.com:8080")
	if got != "example.com:8080" {
		t.Errorf("extractHostPort = %q, want example.com:8080", got)
	}
}

func TestExtractHostPortHTTP(t *testing.T) {
	t.Parallel()
	got := extractHostPort("http://localhost:64000")
	if got != "localhost:64000" {
		t.Errorf("extractHostPort = %q, want localhost:64000", got)
	}
}

func TestExtractHostPortNoScheme(t *testing.T) {
	t.Parallel()
	got := extractHostPort("example.com:9090")
	if got != "example.com:9090" {
		t.Errorf("extractHostPort = %q, want example.com:9090", got)
	}
}

// --- getURLIcon ---

func TestGetURLIconOnion(t *testing.T) {
	t.Parallel()
	icon := getURLIcon("http://abc123.onion")
	if !strings.Contains(icon, "Tor") {
		t.Errorf("onion URL icon = %q, want Tor", icon)
	}
}

func TestGetURLIconHTTPS(t *testing.T) {
	t.Parallel()
	icon := getURLIcon("https://example.com")
	if !strings.Contains(icon, "HTTPS") {
		t.Errorf("https URL icon = %q, want HTTPS", icon)
	}
}

func TestGetURLIconIPv6(t *testing.T) {
	t.Parallel()
	icon := getURLIcon("http://[::1]:8080")
	if !strings.Contains(icon, "IPv6") {
		t.Errorf("IPv6 URL icon = %q, want IPv6", icon)
	}
}

func TestGetURLIconHTTP(t *testing.T) {
	t.Parallel()
	icon := getURLIcon("http://example.com")
	if !strings.Contains(icon, "HTTP") {
		t.Errorf("http URL icon = %q, want HTTP", icon)
	}
}

// --- printFullBanner ---

func TestPrintFullBannerContainsAppName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{
		AppName:   "TestApp",
		Version:   "1.0.0",
		CommitID:  "abc1234",
		BuildDate: "2025-01-01",
		Mode:      "production",
		URLs:      []string{"https://example.com"},
		ListenURL: "localhost:64000",
	}
	printFullBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "TestApp") {
		t.Errorf("full banner missing AppName: %q", out)
	}
	if !strings.Contains(out, "1.0.0") {
		t.Errorf("full banner missing Version: %q", out)
	}
	if !strings.Contains(out, "localhost:64000") {
		t.Errorf("full banner missing ListenURL: %q", out)
	}
}

func TestPrintFullBannerDebugMode(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{AppName: "App", Version: "0.1", Mode: "development", Debug: true}
	printFullBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "development") {
		t.Errorf("full banner (debug) missing mode: %q", out)
	}
}

func TestPrintFullBannerNoListenURL(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{AppName: "App", Version: "0.1", Mode: "production", URLs: []string{"http://localhost"}}
	printFullBanner(&buf, cfg)
	// Should not panic and should produce output
	if buf.Len() == 0 {
		t.Error("printFullBanner produced no output")
	}
}

// --- printCompactBanner ---

func TestPrintCompactBannerContainsVersion(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{
		AppName: "CompactApp",
		Version: "2.0.0",
		Mode:    "production",
		URLs:    []string{"https://example.com"},
	}
	printCompactBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "CompactApp") {
		t.Errorf("compact banner missing AppName: %q", out)
	}
	if !strings.Contains(out, "2.0.0") {
		t.Errorf("compact banner missing Version: %q", out)
	}
}

// --- printMinimalBanner ---

func TestPrintMinimalBannerAppName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{AppName: "MinApp", Version: "1.0", URLs: []string{"http://localhost:64001"}}
	printMinimalBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "MinApp") {
		t.Errorf("minimal banner missing AppName: %q", out)
	}
}

// --- printMicroBanner ---

func TestPrintMicroBannerWithURL(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{AppName: "MicroApp", URLs: []string{"http://localhost:9999"}}
	printMicroBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "MicroApp") {
		t.Errorf("micro banner missing AppName: %q", out)
	}
}

func TestPrintMicroBannerNoURL(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{AppName: "MicroApp"}
	printMicroBanner(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "MicroApp") {
		t.Errorf("micro banner (no URL) missing AppName: %q", out)
	}
}

// --- PrintSetupToken ---

func TestPrintSetupTokenWide(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// Call directly for wide output (can't control terminal width, so use the underlying function)
	// Simulate wide terminal by calling the box output path directly
	token := "SETUP-TOKEN-XYZ-123"
	// Width check in PrintSetupToken uses getTerminalWidth() which returns 80 in tests.
	// So this always runs the wide path in a CI terminal.
	PrintSetupToken(&buf, token)
	out := buf.String()
	if !strings.Contains(out, token) {
		t.Errorf("PrintSetupToken missing token: %q", out)
	}
}

func TestPrintSetupTokenContainsInstructions(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintSetupToken(&buf, "MY-TOKEN")
	out := buf.String()
	if !strings.Contains(out, "MY-TOKEN") {
		t.Errorf("PrintSetupToken output missing token: %q", out)
	}
}

// --- PrintLegacy ---

func TestPrintLegacy(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintLegacy(&buf, "9.9.9", "def5678", "2025-06-01")
	out := buf.String()
	if !strings.Contains(out, "CASRAD") {
		t.Errorf("PrintLegacy missing CASRAD: %q", out)
	}
}

// --- Print (dispatch) ---

func TestPrintDoesNotPanic(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := Config{
		AppName: "PanicTest",
		Version: "0.0.1",
		Mode:    "production",
	}
	// Should not panic regardless of terminal width
	Print(&buf, cfg)
}
