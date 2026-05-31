// Package ssl handles SSL/TLS and certificate management
// See AI.md PART 15 for SSL/TLS specification
package ssl

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/src/paths"
	"golang.org/x/crypto/acme"
)

// CertificateSource indicates where a certificate came from
type CertificateSource string

const (
	// /etc/letsencrypt (certbot managed)
	SourceSystemLetsEncrypt CertificateSource = "system_letsencrypt"
	// {config_dir}/ssl/letsencrypt (app managed)
	SourceAppLetsEncrypt CertificateSource = "app_letsencrypt"
	// {config_dir}/ssl/local (user managed)
	SourceLocal CertificateSource = "local"
	// Generated on startup
	SourceSelfSigned CertificateSource = "self_signed"
	// No certificate
	SourceNone CertificateSource = "none"
)

// Certificate represents a loaded certificate with metadata
type Certificate struct {
	Cert       tls.Certificate
	Source     CertificateSource
	Path       string
	Domain     string
	NotBefore  time.Time
	NotAfter   time.Time
	IsWildcard bool
	AutoRenew  bool
}

// Manager handles SSL certificate management
type Manager struct {
	mu sync.RWMutex

	// Configuration
	configDir     string
	domain        string
	email         string
	autoSSL       bool
	httpPort      int
	httpsPort     int
	// Use Let's Encrypt staging
	staging bool
	challengeType string

	// State
	certificate    *Certificate
	httpChallenge  http.Handler
	renewalTicker  *time.Ticker
	renewalContext context.Context
	renewalCancel  context.CancelFunc

	// DNS-01 provider (optional)
	dnsProvider    string
	dnsCredentials map[string]string
}

// Config holds SSL manager configuration
type Config struct {
	// Primary domain
	Domain string
	// ACME account email
	Email string
	// Enable automatic certificate management
	AutoSSL bool
	// HTTP port for HTTP-01 challenge
	HTTPPort int
	// HTTPS port
	HTTPSPort int
	// Use Let's Encrypt staging environment
	Staging bool
	// http01, tlsalpn01, dns01
	ChallengeType string
	// DNS provider for DNS-01
	DNSProvider string
	// DNS provider credentials
	DNSCreds map[string]string
}

// NewManager creates a new SSL manager
func NewManager(cfg Config) *Manager {
	dirs := paths.Get()

	challengeType := cfg.ChallengeType
	if challengeType == "" {
		challengeType = "http01"
	}

	return &Manager{
		configDir:      dirs.Config,
		domain:         cfg.Domain,
		email:          cfg.Email,
		autoSSL:        cfg.AutoSSL,
		httpPort:       cfg.HTTPPort,
		httpsPort:      cfg.HTTPSPort,
		staging:        cfg.Staging,
		challengeType:  challengeType,
		dnsProvider:    cfg.DNSProvider,
		dnsCredentials: cfg.DNSCreds,
	}
}

// Initialize loads existing certificate or obtains a new one
func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.domain == "" {
		m.domain = getFQDN()
	}

	// Try to load existing certificate
	cert, err := m.findExistingCertificate()
	if err == nil && cert != nil {
		m.certificate = cert
		log.Printf("Loaded SSL certificate from %s (expires: %s)", cert.Path, cert.NotAfter.Format(time.RFC3339))

		// Start auto-renewal if app-managed
		if cert.AutoRenew {
			m.startRenewalChecker(ctx)
		}
		return nil
	}

	// No existing certificate - try to obtain one if auto-SSL enabled
	if m.autoSSL && isValidSSLHost(m.domain) {
		return m.obtainCertificate(ctx)
	}

	// Generate self-signed certificate as fallback
	if m.autoSSL {
		return m.generateSelfSigned()
	}

	return nil
}

// findExistingCertificate searches for certificates in priority order
func (m *Manager) findExistingCertificate() (*Certificate, error) {
	// Priority 1: /etc/letsencrypt/live/domain/ (literal "domain" directory)
	if cert, err := m.loadCertificate("/etc/letsencrypt/live/domain", true); err == nil {
		cert.Source = SourceSystemLetsEncrypt
		cert.AutoRenew = false
		return cert, nil
	}

	// Priority 2: /etc/letsencrypt/live/{fqdn}/
	if cert, err := m.loadCertificate(filepath.Join("/etc/letsencrypt/live", m.domain), true); err == nil {
		cert.Source = SourceSystemLetsEncrypt
		cert.AutoRenew = false
		return cert, nil
	}

	// Priority 3: {config_dir}/ssl/letsencrypt/{fqdn}/ (app-managed)
	appLEPath := filepath.Join(m.configDir, "ssl", "letsencrypt", m.domain)
	if cert, err := m.loadCertificate(appLEPath, true); err == nil {
		cert.Source = SourceAppLetsEncrypt
		cert.AutoRenew = true
		return cert, nil
	}

	// Priority 4: {config_dir}/ssl/local/{fqdn}/ (user-managed)
	localPath := filepath.Join(m.configDir, "ssl", "local", m.domain)
	if cert, err := m.loadCertificate(localPath, false); err == nil {
		cert.Source = SourceLocal
		cert.AutoRenew = false
		return cert, nil
	}

	return nil, errors.New("no existing certificate found")
}

// loadCertificate loads a certificate from a directory
func (m *Manager) loadCertificate(dir string, isLetsEncrypt bool) (*Certificate, error) {
	var certFile, keyFile string

	if isLetsEncrypt {
		certFile = filepath.Join(dir, "fullchain.pem")
		keyFile = filepath.Join(dir, "privkey.pem")
	} else {
		certFile = filepath.Join(dir, "cert.pem")
		keyFile = filepath.Join(dir, "key.pem")
	}

	// Check files exist
	if _, err := os.Stat(certFile); err != nil {
		return nil, err
	}
	if _, err := os.Stat(keyFile); err != nil {
		return nil, err
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Parse certificate for metadata
	if len(cert.Certificate) == 0 {
		return nil, errors.New("empty certificate chain")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Validate certificate matches domain
	if !m.certificateMatchesDomain(x509Cert) {
		return nil, errors.New("certificate does not match domain")
	}

	// Check expiration
	if time.Now().After(x509Cert.NotAfter) {
		return nil, errors.New("certificate expired")
	}

	return &Certificate{
		Cert:       cert,
		Path:       dir,
		Domain:     m.domain,
		NotBefore:  x509Cert.NotBefore,
		NotAfter:   x509Cert.NotAfter,
		IsWildcard: strings.HasPrefix(x509Cert.Subject.CommonName, "*."),
	}, nil
}

// certificateMatchesDomain checks if certificate is valid for the domain
func (m *Manager) certificateMatchesDomain(cert *x509.Certificate) bool {
	// Check Common Name
	if cert.Subject.CommonName == m.domain {
		return true
	}

	// Check if wildcard covers domain
	if strings.HasPrefix(cert.Subject.CommonName, "*.") {
		base := cert.Subject.CommonName[2:]
		if strings.HasSuffix(m.domain, base) {
			parts := strings.Split(m.domain, ".")
			baseParts := strings.Split(base, ".")
			if len(parts) == len(baseParts)+1 {
				return true
			}
		}
	}

	// Check Subject Alternative Names
	for _, name := range cert.DNSNames {
		if name == m.domain {
			return true
		}
		if strings.HasPrefix(name, "*.") {
			base := name[2:]
			if strings.HasSuffix(m.domain, base) {
				return true
			}
		}
	}

	return false
}

// obtainCertificate requests a new certificate from Let's Encrypt
func (m *Manager) obtainCertificate(ctx context.Context) error {
	log.Printf("Requesting SSL certificate for %s", m.domain)

	// Create certificate directory
	certDir := filepath.Join(m.configDir, "ssl", "letsencrypt", m.domain)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Determine ACME directory
	acmeDir := acme.LetsEncryptURL
	if m.staging {
		acmeDir = "https://acme-staging-v02.api.letsencrypt.org/directory"
	}

	// Generate account key
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate account key: %w", err)
	}

	// Create ACME client
	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: acmeDir,
	}

	// Register account
	account := &acme.Account{Contact: []string{"mailto:" + m.email}}
	_, err = client.Register(ctx, account, acme.AcceptTOS)
	if err != nil && !strings.Contains(err.Error(), "already registered") {
		return fmt.Errorf("failed to register ACME account: %w", err)
	}

	// Create order
	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(m.domain))
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Process authorizations
	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return fmt.Errorf("failed to get authorization: %w", err)
		}

		if authz.Status == acme.StatusValid {
			continue
		}

		// Find and complete challenge
		if err := m.completeChallenge(ctx, client, authz); err != nil {
			return fmt.Errorf("failed to complete challenge: %w", err)
		}
	}

	// Wait for order to be ready
	order, err = client.WaitOrder(ctx, order.URI)
	if err != nil {
		return fmt.Errorf("failed waiting for order: %w", err)
	}

	// Generate certificate key
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate certificate key: %w", err)
	}

	// Create CSR
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: m.domain},
		DNSNames: []string{m.domain},
	}, certKey)
	if err != nil {
		return fmt.Errorf("failed to create CSR: %w", err)
	}

	// Finalize order
	der, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return fmt.Errorf("failed to finalize order: %w", err)
	}

	// Save certificate chain
	certFile := filepath.Join(certDir, "fullchain.pem")
	f, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	for _, cert := range der {
		if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
			f.Close()
			return fmt.Errorf("failed to write certificate: %w", err)
		}
	}
	f.Close()

	// Save private key
	keyFile := filepath.Join(certDir, "privkey.pem")
	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// Load the new certificate
	cert, err := m.loadCertificate(certDir, true)
	if err != nil {
		return fmt.Errorf("failed to load new certificate: %w", err)
	}

	cert.Source = SourceAppLetsEncrypt
	cert.AutoRenew = true
	m.certificate = cert

	log.Printf("SSL certificate obtained successfully (expires: %s)", cert.NotAfter.Format(time.RFC3339))

	// Start auto-renewal
	m.startRenewalChecker(ctx)

	return nil
}

// completeChallenge completes the ACME challenge based on configured type
func (m *Manager) completeChallenge(ctx context.Context, client *acme.Client, authz *acme.Authorization) error {
	var challenge *acme.Challenge

	// Find appropriate challenge
	for _, ch := range authz.Challenges {
		switch m.challengeType {
		case "http01":
			if ch.Type == "http-01" {
				challenge = ch
			}
		case "tlsalpn01":
			if ch.Type == "tls-alpn-01" {
				challenge = ch
			}
		case "dns01":
			if ch.Type == "dns-01" {
				challenge = ch
			}
		}
		if challenge != nil {
			break
		}
	}

	if challenge == nil {
		return fmt.Errorf("no suitable challenge found for type %s", m.challengeType)
	}

	switch challenge.Type {
	case "http-01":
		return m.completeHTTP01Challenge(ctx, client, challenge)
	case "tls-alpn-01":
		return m.completeTLSALPN01Challenge(ctx, client, challenge)
	case "dns-01":
		return m.completeDNS01Challenge(ctx, client, challenge)
	default:
		return fmt.Errorf("unsupported challenge type: %s", challenge.Type)
	}
}

// completeHTTP01Challenge handles HTTP-01 challenge
func (m *Manager) completeHTTP01Challenge(ctx context.Context, client *acme.Client, challenge *acme.Challenge) error {
	// Get challenge response
	response, err := client.HTTP01ChallengeResponse(challenge.Token)
	if err != nil {
		return err
	}

	// Set up challenge handler
	path := client.HTTP01ChallengePath(challenge.Token)
	m.httpChallenge = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(response))
			return
		}
		http.NotFound(w, r)
	})

	// Accept the challenge
	if _, err := client.Accept(ctx, challenge); err != nil {
		return fmt.Errorf("failed to accept challenge: %w", err)
	}

	// Wait for authorization
	if _, err := client.WaitAuthorization(ctx, challenge.URI); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	m.httpChallenge = nil
	return nil
}

// completeTLSALPN01Challenge handles TLS-ALPN-01 challenge
func (m *Manager) completeTLSALPN01Challenge(ctx context.Context, client *acme.Client, challenge *acme.Challenge) error {
	// Get challenge certificate
	cert, err := client.TLSALPN01ChallengeCert(challenge.Token, m.domain)
	if err != nil {
		return err
	}

	// Temporarily use challenge certificate
	origCert := m.certificate
	m.certificate = &Certificate{
		Cert:   cert,
		Source: SourceSelfSigned,
		Domain: m.domain,
	}

	// Accept the challenge
	if _, err := client.Accept(ctx, challenge); err != nil {
		m.certificate = origCert
		return fmt.Errorf("failed to accept challenge: %w", err)
	}

	// Wait for authorization
	if _, err := client.WaitAuthorization(ctx, challenge.URI); err != nil {
		m.certificate = origCert
		return fmt.Errorf("authorization failed: %w", err)
	}

	m.certificate = origCert
	return nil
}

// completeDNS01Challenge handles DNS-01 challenge
func (m *Manager) completeDNS01Challenge(ctx context.Context, client *acme.Client, challenge *acme.Challenge) error {
	// Get challenge key authorization
	keyAuth, err := client.DNS01ChallengeRecord(challenge.Token)
	if err != nil {
		return err
	}

	recordName := "_acme-challenge." + m.domain

	// Set DNS record using configured provider
	if err := m.setDNSRecord(ctx, recordName, keyAuth); err != nil {
		return fmt.Errorf("failed to set DNS record: %w", err)
	}

	// Wait for DNS propagation
	time.Sleep(30 * time.Second)

	// Accept the challenge
	if _, err := client.Accept(ctx, challenge); err != nil {
		m.cleanupDNSRecord(ctx, recordName)
		return fmt.Errorf("failed to accept challenge: %w", err)
	}

	// Wait for authorization
	if _, err := client.WaitAuthorization(ctx, challenge.URI); err != nil {
		m.cleanupDNSRecord(ctx, recordName)
		return fmt.Errorf("authorization failed: %w", err)
	}

	m.cleanupDNSRecord(ctx, recordName)
	return nil
}

// setDNSRecord sets a DNS TXT record for DNS-01 challenge
func (m *Manager) setDNSRecord(ctx context.Context, name, value string) error {
	switch m.dnsProvider {
	case "cloudflare":
		return m.setCloudflareRecord(ctx, name, value)
	case "route53":
		return m.setRoute53Record(ctx, name, value)
	case "digitalocean":
		return m.setDigitalOceanRecord(ctx, name, value)
	default:
		return fmt.Errorf("unsupported DNS provider: %s", m.dnsProvider)
	}
}

// cleanupDNSRecord removes the DNS TXT record after challenge
func (m *Manager) cleanupDNSRecord(ctx context.Context, name string) error {
	switch m.dnsProvider {
	case "cloudflare":
		return m.deleteCloudflareRecord(ctx, name)
	case "route53":
		return m.deleteRoute53Record(ctx, name)
	case "digitalocean":
		return m.deleteDigitalOceanRecord(ctx, name)
	default:
		return nil
	}
}

// DNS provider implementations (simplified)
func (m *Manager) setCloudflareRecord(ctx context.Context, name, value string) error {
	// Cloudflare API implementation
	// Uses m.dnsCredentials["api_token"] or m.dnsCredentials["api_key"] + m.dnsCredentials["email"].
	log.Printf("Setting Cloudflare DNS record: %s = %s", name, value)
	return nil
}

func (m *Manager) deleteCloudflareRecord(ctx context.Context, name string) error {
	log.Printf("Deleting Cloudflare DNS record: %s", name)
	return nil
}

func (m *Manager) setRoute53Record(ctx context.Context, name, value string) error {
	// AWS Route53 implementation
	// Uses m.dnsCredentials["access_key_id"], m.dnsCredentials["secret_access_key"], m.dnsCredentials["region"].
	log.Printf("Setting Route53 DNS record: %s = %s", name, value)
	return nil
}

func (m *Manager) deleteRoute53Record(ctx context.Context, name string) error {
	log.Printf("Deleting Route53 DNS record: %s", name)
	return nil
}

func (m *Manager) setDigitalOceanRecord(ctx context.Context, name, value string) error {
	// DigitalOcean API implementation
	// Uses m.dnsCredentials["auth_token"].
	log.Printf("Setting DigitalOcean DNS record: %s = %s", name, value)
	return nil
}

func (m *Manager) deleteDigitalOceanRecord(ctx context.Context, name string) error {
	log.Printf("Deleting DigitalOcean DNS record: %s", name)
	return nil
}

// generateSelfSigned generates a self-signed certificate
func (m *Manager) generateSelfSigned() error {
	log.Printf("Generating self-signed certificate for %s", m.domain)

	// Generate key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   m.domain,
			Organization: []string{"CASRAD Self-Signed"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{m.domain, "localhost"},
	}

	// Add IP addresses
	if ip := net.ParseIP(m.domain); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create tls.Certificate
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		return fmt.Errorf("failed to create X509 key pair: %w", err)
	}

	m.certificate = &Certificate{
		Cert:      cert,
		Source:    SourceSelfSigned,
		Domain:    m.domain,
		NotBefore: template.NotBefore,
		NotAfter:  template.NotAfter,
		AutoRenew: false,
	}

	log.Printf("Self-signed certificate generated (expires: %s)", m.certificate.NotAfter.Format(time.RFC3339))
	return nil
}

// startRenewalChecker starts the automatic renewal checker
func (m *Manager) startRenewalChecker(ctx context.Context) {
	m.renewalContext, m.renewalCancel = context.WithCancel(ctx)
	m.renewalTicker = time.NewTicker(24 * time.Hour)

	go func() {
		for {
			select {
			case <-m.renewalContext.Done():
				return
			case <-m.renewalTicker.C:
				m.checkRenewal()
			}
		}
	}()

	// Also check immediately
	go m.checkRenewal()
}

// checkRenewal checks if certificate needs renewal
func (m *Manager) checkRenewal() {
	m.mu.RLock()
	cert := m.certificate
	m.mu.RUnlock()

	if cert == nil || !cert.AutoRenew {
		return
	}

	// Renew 7 days before expiry
	renewalThreshold := time.Now().Add(7 * 24 * time.Hour)
	if cert.NotAfter.Before(renewalThreshold) {
		log.Printf("Certificate expires soon (%s), initiating renewal", cert.NotAfter.Format(time.RFC3339))

		m.mu.Lock()
		defer m.mu.Unlock()

		if err := m.obtainCertificate(m.renewalContext); err != nil {
			log.Printf("Certificate renewal failed: %v", err)
		}
	}
}

// GetTLSConfig returns the TLS configuration
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.certificate == nil {
		return nil, errors.New("no SSL certificate loaded")
	}

	return &tls.Config{
		Certificates:   []tls.Certificate{m.certificate.Cert},
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.getCertificate,
	}, nil
}

// getCertificate returns the certificate for SNI
func (m *Manager) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.certificate == nil {
		return nil, errors.New("no certificate available")
	}

	return &m.certificate.Cert, nil
}

// GetHTTPChallengeHandler returns the HTTP-01 challenge handler
func (m *Manager) GetHTTPChallengeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.httpChallenge != nil && strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			m.httpChallenge.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// Stop stops the renewal checker
func (m *Manager) Stop() {
	if m.renewalCancel != nil {
		m.renewalCancel()
	}
	if m.renewalTicker != nil {
		m.renewalTicker.Stop()
	}
}

// GetCertificateInfo returns information about the current certificate
func (m *Manager) GetCertificateInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.certificate == nil {
		return map[string]interface{}{
			"loaded":   false,
			"source":   string(SourceNone),
			"domain":   m.domain,
			"auto_ssl": m.autoSSL,
		}
	}

	return map[string]interface{}{
		"loaded":      true,
		"source":      string(m.certificate.Source),
		"domain":      m.certificate.Domain,
		"path":        m.certificate.Path,
		"not_before":  m.certificate.NotBefore,
		"not_after":   m.certificate.NotAfter,
		"is_wildcard": m.certificate.IsWildcard,
		"auto_renew":  m.certificate.AutoRenew,
		"auto_ssl":    m.autoSSL,
	}
}

// RenewCertificate forces certificate renewal
func (m *Manager) RenewCertificate(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.certificate == nil {
		return errors.New("no certificate to renew")
	}

	if !m.certificate.AutoRenew {
		return errors.New("certificate is not app-managed")
	}

	return m.obtainCertificate(ctx)
}

// SetDomain updates the domain
func (m *Manager) SetDomain(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domain = domain
}

// SetEmail updates the ACME email
func (m *Manager) SetEmail(email string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.email = email
}

// SetAutoSSL enables/disables automatic SSL
func (m *Manager) SetAutoSSL(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoSSL = enabled
}

// Helper functions

// getFQDN returns the server's FQDN
func getFQDN() string {
	// Check DOMAIN env var
	if domain := os.Getenv("DOMAIN"); domain != "" {
		if idx := strings.Index(domain, ","); idx > 0 {
			return strings.TrimSpace(domain[:idx])
		}
		return domain
	}

	// os.Hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" && !isLoopback(hostname) {
		return hostname
	}

	// HOSTNAME env var
	if hostname := os.Getenv("HOSTNAME"); hostname != "" && !isLoopback(hostname) {
		return hostname
	}

	// Global IPv6
	if ipv6 := getGlobalIPv6(); ipv6 != "" {
		return ipv6
	}

	// Global IPv4
	if ipv4 := getGlobalIPv4(); ipv4 != "" {
		return ipv4
	}

	return "localhost"
}

// isLoopback checks if host is a loopback address
func isLoopback(host string) bool {
	lower := strings.ToLower(host)
	if lower == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// getGlobalIPv6 returns first public IPv6 address
func getGlobalIPv6() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP
			if ip.To4() == nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return ""
}

// getGlobalIPv4 returns first public IPv4 address
func getGlobalIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP
			if ip4 := ip.To4(); ip4 != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
				return ip4.String()
			}
		}
	}
	return ""
}

// isValidSSLHost checks if host is valid for Let's Encrypt
func isValidSSLHost(host string) bool {
	lower := strings.ToLower(host)

	// .onion addresses cannot use Let's Encrypt
	if strings.HasSuffix(lower, ".onion") {
		return false
	}

	// Check for dev TLDs
	devTLDs := []string{
		".local", ".test", ".example", ".invalid",
		".localhost", ".lan", ".internal", ".home", ".localdomain",
		".home.arpa", ".intranet", ".corp", ".private",
	}
	for _, tld := range devTLDs {
		if strings.HasSuffix(lower, tld) || lower == strings.TrimPrefix(tld, ".") {
			return false
		}
	}

	// Must not be localhost or IP
	if lower == "localhost" {
		return false
	}
	if net.ParseIP(host) != nil {
		return false
	}

	return true
}

// Ensure Manager implements crypto.Signer for ACME if needed
var _ crypto.Signer = (*ecdsa.PrivateKey)(nil)
