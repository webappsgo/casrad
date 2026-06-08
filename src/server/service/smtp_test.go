// Package service — Tests for SMTP configuration helpers.
// Covers: DefaultFromEmail, LoadSMTPFromEnv (env var override), TLSMode constants,
// SMTPAutoDetectHosts and SMTPAutoDetectPorts defaults.
// Note: AutoDetectSMTP and testSMTPConnection make real network calls — not tested here.
package service

import (
	"strings"
	"testing"
)

// --- DefaultFromEmail ---

func TestDefaultFromEmailWithFQDN(t *testing.T) {
	t.Parallel()
	got := DefaultFromEmail("myserver.example.com")
	want := "no-reply@myserver.example.com"
	if got != want {
		t.Errorf("DefaultFromEmail(myserver.example.com) = %q, want %q", got, want)
	}
}

func TestDefaultFromEmailEmptyFQDNUsesLocalhost(t *testing.T) {
	t.Parallel()
	got := DefaultFromEmail("")
	if !strings.Contains(got, "localhost") {
		t.Errorf("DefaultFromEmail(\"\") = %q, should contain localhost", got)
	}
}

func TestDefaultFromEmailHasNoReplyPrefix(t *testing.T) {
	t.Parallel()
	got := DefaultFromEmail("example.com")
	if !strings.HasPrefix(got, "no-reply@") {
		t.Errorf("DefaultFromEmail() = %q, should start with no-reply@", got)
	}
}

// --- TLSMode constants ---

func TestTLSModeConstantsValues(t *testing.T) {
	t.Parallel()
	if string(TLSModeAuto) != "auto" {
		t.Errorf("TLSModeAuto = %q, want auto", TLSModeAuto)
	}
	if string(TLSModeStartTLS) != "starttls" {
		t.Errorf("TLSModeStartTLS = %q, want starttls", TLSModeStartTLS)
	}
	if string(TLSModeTLS) != "tls" {
		t.Errorf("TLSModeTLS = %q, want tls", TLSModeTLS)
	}
	if string(TLSModeNone) != "none" {
		t.Errorf("TLSModeNone = %q, want none", TLSModeNone)
	}
}

// --- SMTPAutoDetectHosts / Ports defaults ---

func TestSMTPAutoDetectHostsContainsLocalhost(t *testing.T) {
	t.Parallel()
	found := false
	for _, h := range SMTPAutoDetectHosts {
		if h == "localhost" || h == "127.0.0.1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SMTPAutoDetectHosts = %v, should contain localhost or 127.0.0.1", SMTPAutoDetectHosts)
	}
}

func TestSMTPAutoDetectPortsContainsCommonPorts(t *testing.T) {
	t.Parallel()
	portSet := make(map[int]bool)
	for _, p := range SMTPAutoDetectPorts {
		portSet[p] = true
	}
	for _, want := range []int{25, 465, 587} {
		if !portSet[want] {
			t.Errorf("SMTPAutoDetectPorts = %v, should contain port %d", SMTPAutoDetectPorts, want)
		}
	}
}

// --- LoadSMTPFromEnv ---

func TestLoadSMTPFromEnvNilReturnsNonNil(t *testing.T) {
	t.Parallel()
	got := LoadSMTPFromEnv(nil)
	if got == nil {
		t.Error("LoadSMTPFromEnv(nil) returned nil")
	}
}

func TestLoadSMTPFromEnvPreservesExistingValues(t *testing.T) {
	t.Parallel()
	cfg := &SMTPSettings{Host: "mail.example.com", Port: 587}
	got := LoadSMTPFromEnv(cfg)
	if got.Host != "mail.example.com" {
		t.Errorf("LoadSMTPFromEnv preserved host = %q, want mail.example.com", got.Host)
	}
}

func TestLoadSMTPFromEnvOverridesHost(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.override.example.com")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	cfg := &SMTPSettings{Host: "original.example.com"}
	got := LoadSMTPFromEnv(cfg)
	if got.Host != "smtp.override.example.com" {
		t.Errorf("LoadSMTPFromEnv SMTP_HOST override = %q, want smtp.override.example.com", got.Host)
	}
}

func TestLoadSMTPFromEnvOverridesPort(t *testing.T) {
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	cfg := &SMTPSettings{Port: 25}
	got := LoadSMTPFromEnv(cfg)
	if got.Port != 465 {
		t.Errorf("LoadSMTPFromEnv SMTP_PORT override = %d, want 465", got.Port)
	}
}

func TestLoadSMTPFromEnvIgnoresInvalidPort(t *testing.T) {
	t.Setenv("SMTP_PORT", "notanumber")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	cfg := &SMTPSettings{Port: 25}
	got := LoadSMTPFromEnv(cfg)
	if got.Port != 25 {
		t.Errorf("LoadSMTPFromEnv bad SMTP_PORT should leave original port 25, got %d", got.Port)
	}
}

func TestLoadSMTPFromEnvTLSModeAuto(t *testing.T) {
	t.Setenv("SMTP_TLS", "auto")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	got := LoadSMTPFromEnv(nil)
	if got.TLSMode != TLSModeAuto {
		t.Errorf("LoadSMTPFromEnv SMTP_TLS=auto mode = %q, want auto", got.TLSMode)
	}
}

func TestLoadSMTPFromEnvTLSModeStartTLS(t *testing.T) {
	t.Setenv("SMTP_TLS", "starttls")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "")

	got := LoadSMTPFromEnv(nil)
	if got.TLSMode != TLSModeStartTLS {
		t.Errorf("LoadSMTPFromEnv SMTP_TLS=starttls mode = %q, want starttls", got.TLSMode)
	}
}

func TestLoadSMTPFromEnvFromEmail(t *testing.T) {
	t.Setenv("SMTP_FROM_EMAIL", "noreply@test.example.com")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_FROM_NAME", "")

	got := LoadSMTPFromEnv(nil)
	if got.FromEmail != "noreply@test.example.com" {
		t.Errorf("LoadSMTPFromEnv SMTP_FROM_EMAIL = %q, want noreply@test.example.com", got.FromEmail)
	}
}
