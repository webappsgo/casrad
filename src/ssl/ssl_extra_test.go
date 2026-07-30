// Package ssl — Additional tests targeting uncovered branches in ssl.go.
// Covers: generateSelfSigned (full path including IP domain), GetTLSConfig
// (cert loaded), getCertificate (SNI callback), certificateMatchesDomain
// (all branches via real x509 certs), getFQDN (env overrides),
// getGlobalIPv4/getGlobalIPv6 (network scan, skipped if unavailable),
// RenewCertificate (no cert / non-auto-renew), checkRenewal (various states),
// Stop (with active ticker), setDNSRecord/cleanupDNSRecord (all providers
// plus unknown), DNS stub log methods (direct call), GetCertificateInfo
// (cert loaded), isLoopback extra branches, isValidSSLHost remaining TLDs.
package ssl

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// generateForTest calls generateSelfSigned on a new manager and fatals on error.
func generateForTest(t *testing.T, domain string) *Manager {
	t.Helper()
	m := NewManager(Config{Domain: domain, AutoSSL: false})
	if err := m.generateSelfSigned(); err != nil {
		t.Fatalf("generateSelfSigned(%q): %v", domain, err)
	}
	return m
}

// buildX509Cert creates a real *x509.Certificate with the given Common Name and
// optional DNS SANs. It generates a throwaway key, signs a certificate, and
// parses it back so DNSNames are correctly populated for testing
// certificateMatchesDomain.
func buildX509Cert(t *testing.T, commonName string, dnsNames []string) *x509.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("buildX509Cert ecdsa.GenerateKey: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		t.Fatalf("buildX509Cert rand.Int: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("buildX509Cert x509.CreateCertificate: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("buildX509Cert MarshalECPrivateKey: %v", err)
	}

	// Round-trip through tls.X509KeyPair ensures Certificate[0] is populated
	// and leaf is the parsed cert.
	tlsCert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		t.Fatalf("buildX509Cert tls.X509KeyPair: %v", err)
	}

	parsed, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("buildX509Cert x509.ParseCertificate: %v", err)
	}

	return parsed
}

// --- generateSelfSigned ---

func TestGenerateSelfSignedPopulatesCertificate(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "test.casrad.local")

	if m.certificate == nil {
		t.Fatal("generateSelfSigned: m.certificate is nil")
	}
	if m.certificate.Source != SourceSelfSigned {
		t.Errorf("certificate.Source = %q, want %q", m.certificate.Source, SourceSelfSigned)
	}
	if m.certificate.Domain != "test.casrad.local" {
		t.Errorf("certificate.Domain = %q, want test.casrad.local", m.certificate.Domain)
	}
}

func TestGenerateSelfSignedNotAfterInFuture(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "future.casrad.local")

	if !m.certificate.NotAfter.After(time.Now()) {
		t.Errorf("certificate.NotAfter %s is not in the future", m.certificate.NotAfter)
	}
}

func TestGenerateSelfSignedAutoRenewIsFalse(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "norenew.casrad.local")

	if m.certificate.AutoRenew {
		t.Error("self-signed cert should have AutoRenew=false")
	}
}

func TestGenerateSelfSignedIPDomain(t *testing.T) {
	t.Parallel()
	// When domain is an IP, the code adds it to template.IPAddresses
	m := NewManager(Config{Domain: "127.0.0.1"})
	if err := m.generateSelfSigned(); err != nil {
		t.Fatalf("generateSelfSigned(IP domain): %v", err)
	}
	if m.certificate == nil {
		t.Fatal("expected certificate for IP domain")
	}
}

func TestGenerateSelfSignedEmptyDomain(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Domain: ""})
	if err := m.generateSelfSigned(); err != nil {
		t.Fatalf("generateSelfSigned(empty): %v", err)
	}
	if m.certificate == nil {
		t.Fatal("expected certificate even with empty domain")
	}
}

// --- GetTLSConfig with cert loaded ---

func TestGetTLSConfigWithCertReturnsValidConfig(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "tlsconfig.local")

	cfg, err := m.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig(): %v", err)
	}
	if cfg == nil {
		t.Fatal("GetTLSConfig returned nil")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %v, want TLS 1.2", cfg.MinVersion)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("len(Certificates) = %d, want 1", len(cfg.Certificates))
	}
	if cfg.GetCertificate == nil {
		t.Error("GetCertificate callback should be non-nil")
	}
}

// --- getCertificate (SNI callback) ---

func TestGetCertificateCallbackReturnsCert(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "snicert.local")

	hello := &tls.ClientHelloInfo{ServerName: "snicert.local"}
	cert, err := m.getCertificate(hello)
	if err != nil {
		t.Fatalf("getCertificate(): %v", err)
	}
	if cert == nil {
		t.Fatal("getCertificate returned nil")
	}
}

func TestGetCertificateCallbackNoCertReturnsError(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{Domain: "empty.local"})

	_, err := m.getCertificate(&tls.ClientHelloInfo{})
	if err == nil {
		t.Error("getCertificate with no cert should return error")
	}
	if !strings.Contains(err.Error(), "no certificate") {
		t.Errorf("getCertificate error = %q, want 'no certificate'", err.Error())
	}
}

// --- certificateMatchesDomain ---

func TestCertificateMatchesDomainExactCN(t *testing.T) {
	t.Parallel()
	m := &Manager{domain: "example.com"}
	if !m.certificateMatchesDomain(buildX509Cert(t, "example.com", nil)) {
		t.Error("exact CN match should return true")
	}
}

func TestCertificateMatchesDomainWildcardCNOneLevel(t *testing.T) {
	t.Parallel()
	// *.example.com matches api.example.com (exactly one subdomain level)
	m := &Manager{domain: "api.example.com"}
	if !m.certificateMatchesDomain(buildX509Cert(t, "*.example.com", nil)) {
		t.Error("single-level wildcard CN should match")
	}
}

func TestCertificateMatchesDomainWildcardCNTwoLevelsFails(t *testing.T) {
	t.Parallel()
	// *.example.com does NOT cover deep.api.example.com
	m := &Manager{domain: "deep.api.example.com"}
	if m.certificateMatchesDomain(buildX509Cert(t, "*.example.com", nil)) {
		t.Error("single-level wildcard CN should not match two-level subdomain")
	}
}

func TestCertificateMatchesDomainSANExact(t *testing.T) {
	t.Parallel()
	m := &Manager{domain: "san.example.com"}
	if !m.certificateMatchesDomain(buildX509Cert(t, "other.com", []string{"san.example.com"})) {
		t.Error("SAN exact match should return true")
	}
}

func TestCertificateMatchesDomainSANWildcard(t *testing.T) {
	t.Parallel()
	m := &Manager{domain: "sub.example.com"}
	if !m.certificateMatchesDomain(buildX509Cert(t, "other.com", []string{"*.example.com"})) {
		t.Error("SAN wildcard match should return true")
	}
}

func TestCertificateMatchesDomainNoMatch(t *testing.T) {
	t.Parallel()
	m := &Manager{domain: "totally.different.org"}
	if m.certificateMatchesDomain(buildX509Cert(t, "example.com", []string{"api.example.com"})) {
		t.Error("non-matching cert should return false")
	}
}

// --- getFQDN ---
// Note: t.Setenv is incompatible with t.Parallel(), so these tests run serially.

func TestGetFQDNWithDomainEnvVar(t *testing.T) {
	t.Setenv("DOMAIN", "myfqdn.example.com")
	if got := getFQDN(); got != "myfqdn.example.com" {
		t.Errorf("getFQDN() = %q, want myfqdn.example.com", got)
	}
}

func TestGetFQDNWithDomainEnvVarCommaList(t *testing.T) {
	t.Setenv("DOMAIN", "first.example.com, second.example.com")
	got := getFQDN()
	if got != "first.example.com" {
		t.Errorf("getFQDN() with comma list = %q, want first.example.com", got)
	}
}

func TestGetFQDNFallbackNonEmpty(t *testing.T) {
	t.Setenv("DOMAIN", "")
	t.Setenv("HOSTNAME", "")
	if got := getFQDN(); got == "" {
		t.Error("getFQDN() should never return empty string")
	}
}

// --- getGlobalIPv4 / getGlobalIPv6 ---

func TestGetGlobalIPv4DoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = getGlobalIPv4()
}

func TestGetGlobalIPv6DoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = getGlobalIPv6()
}

func TestGetGlobalIPv4ReturnedValueIsValidIPv4(t *testing.T) {
	t.Parallel()
	s := getGlobalIPv4()
	if s == "" {
		t.Skip("no public IPv4 in test environment")
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		t.Errorf("getGlobalIPv4() = %q, expected valid IPv4", s)
	}
}

func TestGetGlobalIPv6ReturnedValueIsValidIPv6(t *testing.T) {
	t.Parallel()
	s := getGlobalIPv6()
	if s == "" {
		t.Skip("no public IPv6 in test environment")
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() != nil {
		t.Errorf("getGlobalIPv6() = %q, expected IPv6 (non-IPv4-mappable)", s)
	}
}

// --- isLoopback additional branches ---

func TestIsLoopbackIPv6Loopback(t *testing.T) {
	t.Parallel()
	if !isLoopback("::1") {
		t.Error("isLoopback(::1) should be true")
	}
}

func TestIsLoopback127Range(t *testing.T) {
	t.Parallel()
	if !isLoopback("127.0.0.2") {
		t.Error("isLoopback(127.0.0.2) should be true")
	}
}

func TestIsLoopbackLocalhostCaseInsensitive(t *testing.T) {
	t.Parallel()
	if !isLoopback("LOCALHOST") {
		t.Error("isLoopback(LOCALHOST) should be true (case-insensitive)")
	}
}

func TestIsLoopbackPrivateIPNotLoopback(t *testing.T) {
	t.Parallel()
	if isLoopback("192.168.0.1") {
		t.Error("isLoopback(192.168.0.1) should be false")
	}
}

// --- isValidSSLHost remaining dev TLD branches ---

func TestIsValidSSLHostHomeDotArpa(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("router.home.arpa") {
		t.Error("isValidSSLHost(.home.arpa) should return false")
	}
}

func TestIsValidSSLHostCorp(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("server.corp") {
		t.Error("isValidSSLHost(.corp) should return false")
	}
}

func TestIsValidSSLHostIntranet(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("app.intranet") {
		t.Error("isValidSSLHost(.intranet) should return false")
	}
}

func TestIsValidSSLHostPrivate(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("host.private") {
		t.Error("isValidSSLHost(.private) should return false")
	}
}

func TestIsValidSSLHostIPv6Addr(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("2001:db8::1") {
		t.Error("isValidSSLHost(IPv6) should return false")
	}
}

func TestIsValidSSLHostLAN(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("device.lan") {
		t.Error("isValidSSLHost(.lan) should return false")
	}
}

func TestIsValidSSLHostLocalDomain(t *testing.T) {
	t.Parallel()
	if isValidSSLHost("host.localdomain") {
		t.Error("isValidSSLHost(.localdomain) should return false")
	}
}

// --- RenewCertificate ---

func TestRenewCertificateNoCertReturnsError(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	err := m.RenewCertificate(context.Background())
	if err == nil {
		t.Error("RenewCertificate with no cert should return error")
	}
	if !strings.Contains(err.Error(), "no certificate") {
		t.Errorf("RenewCertificate error = %q, want 'no certificate'", err.Error())
	}
}

func TestRenewCertificateNonAutoRenewReturnsError(t *testing.T) {
	t.Parallel()
	// Self-signed certs have AutoRenew=false by design
	m := generateForTest(t, "nonrenew.local")
	err := m.RenewCertificate(context.Background())
	if err == nil {
		t.Error("RenewCertificate on non-auto-renew cert should return error")
	}
	if !strings.Contains(err.Error(), "not app-managed") {
		t.Errorf("RenewCertificate error = %q, want 'not app-managed'", err.Error())
	}
}

// --- checkRenewal ---

func TestCheckRenewalNilCertIsNoOp(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	// Must not panic when no cert is loaded
	m.checkRenewal()
}

func TestCheckRenewalAutoRenewFalseIsNoOp(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "checkrenew.local")
	// AutoRenew=false: checkRenewal should return early without trying to renew
	m.checkRenewal()
}

func TestCheckRenewalFarFutureDoesNotAttemptRenewal(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "futurerenew.local")
	// Set AutoRenew true but NotAfter far in future (> 7 day threshold)
	m.mu.Lock()
	m.certificate.AutoRenew = true
	m.certificate.NotAfter = time.Now().Add(90 * 24 * time.Hour)
	m.mu.Unlock()
	// Should return without attempting network calls
	m.checkRenewal()
}

// --- Stop with active ticker ---

func TestStopWithActiveRenewalTicker(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "stopticker.local")

	m.mu.Lock()
	m.certificate.AutoRenew = true
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.startRenewalChecker(ctx)

	time.Sleep(5 * time.Millisecond)
	// Must not panic
	m.Stop()
}

func TestStopIdempotentNoPanic(t *testing.T) {
	t.Parallel()
	m := NewManager(Config{})
	m.Stop()
	m.Stop()
}

// --- setDNSRecord / cleanupDNSRecord ---

func TestSetDNSRecordCloudflare(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "cloudflare", dnsCredentials: map[string]string{}}
	if err := m.setDNSRecord(context.Background(), "_acme.test", "v1"); err != nil {
		t.Errorf("setDNSRecord(cloudflare): %v", err)
	}
}

func TestSetDNSRecordRoute53(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "route53", dnsCredentials: map[string]string{}}
	if err := m.setDNSRecord(context.Background(), "_acme.test", "v2"); err != nil {
		t.Errorf("setDNSRecord(route53): %v", err)
	}
}

func TestSetDNSRecordDigitalOcean(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "digitalocean", dnsCredentials: map[string]string{}}
	if err := m.setDNSRecord(context.Background(), "_acme.test", "v3"); err != nil {
		t.Errorf("setDNSRecord(digitalocean): %v", err)
	}
}

func TestSetDNSRecordUnknownReturnsError(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "badprovider"}
	err := m.setDNSRecord(context.Background(), "_acme.test", "v")
	if err == nil {
		t.Fatal("setDNSRecord with unknown provider should error")
	}
	if !strings.Contains(err.Error(), "unsupported DNS provider") {
		t.Errorf("setDNSRecord error = %q, want 'unsupported DNS provider'", err.Error())
	}
}

func TestCleanupDNSRecordCloudflare(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "cloudflare"}
	if err := m.cleanupDNSRecord(context.Background(), "_acme.test"); err != nil {
		t.Errorf("cleanupDNSRecord(cloudflare): %v", err)
	}
}

func TestCleanupDNSRecordRoute53(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "route53"}
	if err := m.cleanupDNSRecord(context.Background(), "_acme.test"); err != nil {
		t.Errorf("cleanupDNSRecord(route53): %v", err)
	}
}

func TestCleanupDNSRecordDigitalOcean(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "digitalocean"}
	if err := m.cleanupDNSRecord(context.Background(), "_acme.test"); err != nil {
		t.Errorf("cleanupDNSRecord(digitalocean): %v", err)
	}
}

func TestCleanupDNSRecordUnknownIsNoOp(t *testing.T) {
	t.Parallel()
	m := &Manager{dnsProvider: "noprovider"}
	// default case returns nil
	if err := m.cleanupDNSRecord(context.Background(), "_acme.test"); err != nil {
		t.Errorf("cleanupDNSRecord(unknown) = %v, want nil", err)
	}
}

// --- DNS stub direct-call coverage ---

func TestSetCloudflareRecord(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).setCloudflareRecord(context.Background(), "n", "v"); err != nil {
		t.Errorf("setCloudflareRecord: %v", err)
	}
}

func TestDeleteCloudflareRecord(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).deleteCloudflareRecord(context.Background(), "n"); err != nil {
		t.Errorf("deleteCloudflareRecord: %v", err)
	}
}

func TestSetRoute53Record(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).setRoute53Record(context.Background(), "n", "v"); err != nil {
		t.Errorf("setRoute53Record: %v", err)
	}
}

func TestDeleteRoute53Record(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).deleteRoute53Record(context.Background(), "n"); err != nil {
		t.Errorf("deleteRoute53Record: %v", err)
	}
}

func TestSetDigitalOceanRecord(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).setDigitalOceanRecord(context.Background(), "n", "v"); err != nil {
		t.Errorf("setDigitalOceanRecord: %v", err)
	}
}

func TestDeleteDigitalOceanRecord(t *testing.T) {
	t.Parallel()
	if err := (&Manager{}).deleteDigitalOceanRecord(context.Background(), "n"); err != nil {
		t.Errorf("deleteDigitalOceanRecord: %v", err)
	}
}

// --- GetCertificateInfo with cert loaded ---

func TestGetCertificateInfoWithCert(t *testing.T) {
	t.Parallel()
	m := generateForTest(t, "infoloaded.local")
	info := m.GetCertificateInfo()

	loaded, ok := info["loaded"].(bool)
	if !ok || !loaded {
		t.Errorf("GetCertificateInfo loaded = %v, want true", info["loaded"])
	}
	if info["source"] != string(SourceSelfSigned) {
		t.Errorf("GetCertificateInfo source = %v, want self_signed", info["source"])
	}
	for _, key := range []string{"not_before", "not_after", "is_wildcard", "auto_renew", "path", "domain"} {
		if _, present := info[key]; !present {
			t.Errorf("GetCertificateInfo missing key %q", key)
		}
	}
}
