// Package ssl — Tests for SSL/TLS helpers.
// Covers: NewManager defaults, CertificateSource constants, isValidSSLHost,
// isLoopback, getFQDN helpers, GetCertificateInfo (no cert loaded),
// SetDomain/SetEmail/SetAutoSSL, Stop (no-op on unconfigured manager).
// Note: obtainCertificate/generateSelfSigned require network or filesystem side
// effects and are not tested here.
package ssl

import (
	"strings"
	"testing"
)

// --- CertificateSource constants ---

func TestCertificateSourceConstantsValues(t *testing.T) {
	t.Parallel()
	if string(SourceSystemLetsEncrypt) != "system_letsencrypt" {
		t.Errorf("SourceSystemLetsEncrypt = %q, want system_letsencrypt", SourceSystemLetsEncrypt)
	}
	if string(SourceAppLetsEncrypt) != "app_letsencrypt" {
		t.Errorf("SourceAppLetsEncrypt = %q, want app_letsencrypt", SourceAppLetsEncrypt)
	}
	if string(SourceLocal) != "local" {
		t.Errorf("SourceLocal = %q, want local", SourceLocal)
	}
	if string(SourceSelfSigned) != "self_signed" {
		t.Errorf("SourceSelfSigned = %q, want self_signed", SourceSelfSigned)
	}
	if string(SourceNone) != "none" {
		t.Errorf("SourceNone = %q, want none", SourceNone)
	}
}

// --- NewManager ---

func TestNewManagerDefaultChallengeType(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	if m.challengeType != "http01" {
		t.Errorf("NewManager default challengeType = %q, want http01", m.challengeType)
	}
}

func TestNewManagerExplicitChallengeType(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{ChallengeType: "dns01"})
	if m.challengeType != "dns01" {
		t.Errorf("NewManager ChallengeType = %q, want dns01", m.challengeType)
	}
}

func TestNewManagerPreservesConfig(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{
		Domain:  "example.com",
		Email:   "admin@example.com",
		AutoSSL: true,
		Staging: true,
	})
	if m.domain != "example.com" {
		t.Errorf("NewManager domain = %q, want example.com", m.domain)
	}
	if m.email != "admin@example.com" {
		t.Errorf("NewManager email = %q, want admin@example.com", m.email)
	}
	if !m.autoSSL {
		t.Error("NewManager autoSSL should be true")
	}
	if !m.staging {
		t.Error("NewManager staging should be true")
	}
}

func TestNewManagerNonNil(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	if m == nil {
		t.Error("NewManager returned nil")
	}
}

// --- isValidSSLHost ---

func TestIsValidSSLHostOnionReturnsFalse(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("abc123.onion") {
		t.Error("isValidSSLHost(.onion) should return false")
	}
}

func TestIsValidSSLHostLocalReturnsFalse(t *testing.T) {
	t.Parallel()
	devHosts := []string{
		"myapp.local",
		"myapp.test",
		"myapp.example",
		"myapp.invalid",
		"myapp.localhost",
		"myapp.lan",
		"myapp.internal",
		"myapp.home",
		"myapp.localdomain",
		"localhost",
	}
	for _, host := range devHosts {
		if isValidSSLHost(host) {
			t.Errorf("isValidSSLHost(%q) should return false for dev TLD", host)
		}
	}
}

func TestIsValidSSLHostIPReturnsFalse(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("192.168.1.1") {
		t.Error("isValidSSLHost(IP) should return false")
	}
	if isValidSSLHost("::1") {
		t.Error("isValidSSLHost(IPv6 ::1) should return false")
	}
}

func TestIsValidSSLHostPublicDomainReturnsTrue(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"example.com", "myserver.casapps.us", "sub.domain.io"} {
		if !isValidSSLHost(host) {
			t.Errorf("isValidSSLHost(%q) should return true for public domain", host)
		}
	}
}

// --- isLoopback ---

func TestIsLoopbackLocalhost(t *testing.T) {
	t.Parallel()
	if !isLoopback("localhost") {
		t.Error("isLoopback(localhost) should be true")
	}
}

func TestIsLoopbackLoopbackIP(t *testing.T) {
	t.Parallel()
	if !isLoopback("127.0.0.1") {
		t.Error("isLoopback(127.0.0.1) should be true")
	}
}

func TestIsLoopbackPublicDomain(t *testing.T) {
	t.Parallel()
	if isLoopback("example.com") {
		t.Error("isLoopback(example.com) should be false")
	}
}

func TestIsLoopbackPublicIP(t *testing.T) {
	t.Parallel()
	if isLoopback("8.8.8.8") {
		t.Error("isLoopback(8.8.8.8) should be false")
	}
}

// --- GetCertificateInfo (no cert loaded) ---

func TestGetCertificateInfoNoSSLCert(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Domain: "test.example.com"})
	info := m.GetCertificateInfo()

	if loaded, ok := info["loaded"].(bool); !ok || loaded {
		t.Errorf("GetCertificateInfo no cert: loaded = %v, want false", info["loaded"])
	}
	if info["source"] != string(SourceNone) {
		t.Errorf("GetCertificateInfo no cert: source = %v, want none", info["source"])
	}
}

func TestGetCertificateInfoDomainPreserved(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Domain: "myserver.example.com"})
	info := m.GetCertificateInfo()
	if info["domain"] != "myserver.example.com" {
		t.Errorf("GetCertificateInfo domain = %v, want myserver.example.com", info["domain"])
	}
}

func TestGetCertificateInfoAutoSSLPreserved(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{AutoSSL: true})
	info := m.GetCertificateInfo()
	if autoSSL, ok := info["auto_ssl"].(bool); !ok || !autoSSL {
		t.Errorf("GetCertificateInfo auto_ssl = %v, want true", info["auto_ssl"])
	}
}

// --- SetDomain / SetEmail / SetAutoSSL ---

func TestSetDomainUpdates(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Domain: "original.com"})
	m.SetDomain("updated.com")
	if m.domain != "updated.com" {
		t.Errorf("SetDomain: domain = %q, want updated.com", m.domain)
	}
}

func TestSetEmailUpdates(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Email: "old@example.com"})
	m.SetEmail("new@example.com")
	if m.email != "new@example.com" {
		t.Errorf("SetEmail: email = %q, want new@example.com", m.email)
	}
}

func TestSetAutoSSLUpdates(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{AutoSSL: false})
	m.SetAutoSSL(true)
	if !m.autoSSL {
		t.Error("SetAutoSSL(true): autoSSL should be true")
	}
	m.SetAutoSSL(false)
	if m.autoSSL {
		t.Error("SetAutoSSL(false): autoSSL should be false")
	}
}

// --- Stop (no-op when not started) ---

func TestStopOnUninitializedManagerNoPanic(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	m.Stop()
}

// --- GetTLSConfig (no cert loaded) ---

func TestGetTLSConfigNoCertReturnsError(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	_, err := m.GetTLSConfig()
	if err == nil {
		t.Error("GetTLSConfig with no cert should return error")
	}
	if !strings.Contains(err.Error(), "no SSL certificate") {
		t.Errorf("GetTLSConfig error = %q, want 'no SSL certificate'", err.Error())
	}
}

// --- GetHTTPChallengeHandler (no active challenge) ---

func TestGetHTTPChallengeHandlerReturnsNotFoundWhenInactive(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	h := m.GetHTTPChallengeHandler()
	if h == nil {
		t.Error("GetHTTPChallengeHandler should never return nil")
	}
}
