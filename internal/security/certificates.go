package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casapps/casrad/internal/database"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// Let's Encrypt endpoints
	LetsEncryptProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	LetsEncryptStagingURL    = "https://acme-staging-v02.api.letsencrypt.org/directory"

	// Certificate renewal
	CertRenewalDays = 30 // Renew 30 days before expiry
)

// CertificateManager handles SSL/TLS certificates
type CertificateManager struct {
	db          *database.Engine
	certDir     string
	acmeClient  *acme.Client
	autocertMgr *autocert.Manager
	httpPort    int
	httpsPort   int
	useStaging  bool
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(db *database.Engine, certDir string, httpPort, httpsPort int) *CertificateManager {
	return &CertificateManager{
		db:        db,
		certDir:   certDir,
		httpPort:  httpPort,
		httpsPort: httpsPort,
	}
}

// Initialize sets up the certificate manager
func (cm *CertificateManager) Initialize(email string) error {
	// Create certificate directory
	if err := os.MkdirAll(cm.certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Determine ACME directory URL
	acmeURL := LetsEncryptProductionURL
	if cm.useStaging {
		acmeURL = LetsEncryptStagingURL
	}

	// Create autocert manager
	cm.autocertMgr = &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		Cache:       autocert.DirCache(filepath.Join(cm.certDir, "autocert")),
		HostPolicy:  cm.hostPolicy,
		Email:       email,
		RenewBefore: time.Duration(CertRenewalDays) * 24 * time.Hour,
		Client: &acme.Client{
			DirectoryURL: acmeURL,
		},
	}

	// Create ACME client for manual operations
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ACME key: %w", err)
	}

	cm.acmeClient = &acme.Client{
		Key:          key,
		DirectoryURL: acmeURL,
	}

	// Register account if needed
	if email != "" {
		ctx := context.Background()
		_, err := cm.acmeClient.Register(ctx, &acme.Account{
			Contact: []string{"mailto:" + email},
		}, acme.AcceptTOS)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Printf("Failed to register ACME account: %v", err)
		}
	}

	return nil
}

// hostPolicy determines which domains are allowed for automatic certificates
func (cm *CertificateManager) hostPolicy(ctx context.Context, host string) error {
	// Check if this is the server's domain
	serverDomain, _ := cm.db.GetSetting("server.domain")
	if host == serverDomain {
		return nil
	}

	// Check user domains
	var count int
	err := cm.db.QueryRow(`
		SELECT COUNT(*) FROM user_domains
		WHERE domain = ? AND is_verified = 1 AND is_active = 1
	`, host).Scan(&count)

	if err == nil && count > 0 {
		return nil
	}

	return fmt.Errorf("domain %s not authorized", host)
}

// GetCertificate returns a certificate for the given domain
func (cm *CertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Use autocert manager if configured
	if cm.autocertMgr != nil && (cm.httpPort == 80 || cm.httpsPort == 443) {
		return cm.autocertMgr.GetCertificate(hello)
	}

	// Check for existing certificate
	certPath := filepath.Join(cm.certDir, hello.ServerName, "cert.pem")
	keyPath := filepath.Join(cm.certDir, hello.ServerName, "key.pem")

	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		// Check expiry
		if !cm.needsRenewal(&cert) {
			return &cert, nil
		}
	}

	// Generate self-signed certificate as fallback
	return cm.generateSelfSignedCert(hello.ServerName)
}

// ObtainCertificate manually obtains a certificate for a domain
func (cm *CertificateManager) ObtainCertificate(domain string) error {
	if cm.httpPort != 80 && cm.httpsPort != 443 {
		return fmt.Errorf("automatic certificates require standard ports (80/443)")
	}

	ctx := context.Background()

	// Create authorization
	authz, err := cm.acmeClient.Authorize(ctx, domain)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Find HTTP-01 challenge
	var chal *acme.Challenge
	for _, c := range authz.Challenges {
		if c.Type == "http-01" {
			chal = c
			break
		}
	}
	if chal == nil {
		return fmt.Errorf("no HTTP-01 challenge found")
	}

	// Prepare challenge response
	resp, err := cm.acmeClient.HTTP01ChallengeResponse(chal.Token)
	if err != nil {
		return fmt.Errorf("challenge response failed: %w", err)
	}

	// Start HTTP server for challenge
	path := "/.well-known/acme-challenge/" + chal.Token
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(resp))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cm.httpPort),
		Handler: mux,
	}

	go server.ListenAndServe()
	defer server.Shutdown(ctx)

	// Accept challenge
	_, err = cm.acmeClient.Accept(ctx, chal)
	if err != nil {
		return fmt.Errorf("challenge acceptance failed: %w", err)
	}

	// Wait for authorization
	_, err = cm.acmeClient.WaitAuthorization(ctx, authz.URI)
	if err != nil {
		return fmt.Errorf("authorization wait failed: %w", err)
	}

	// Generate private key for certificate
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}

	// Create certificate request
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}, certKey)
	if err != nil {
		return fmt.Errorf("CSR creation failed: %w", err)
	}

	// Request certificate
	der, _, err := cm.acmeClient.CreateCert(ctx, csr, 90*24*time.Hour, true)
	if err != nil {
		return fmt.Errorf("certificate creation failed: %w", err)
	}

	// Save certificate and key
	certDir := filepath.Join(cm.certDir, "letsencrypt", domain)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Write certificate
	certOut, err := os.OpenFile(filepath.Join(certDir, "cert.pem"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer certOut.Close()

	for _, d := range der {
		pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: d})
	}

	// Write private key
	keyOut, err := os.OpenFile(filepath.Join(certDir, "key.pem"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(certKey)
	if err != nil {
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Store in database
	cm.db.Exec(`
		INSERT INTO ssl_certificates (domain, type, cert_path, issued_at, expires_at, status)
		VALUES (?, 'server', ?, ?, ?, 'active')
		ON CONFLICT(domain) DO UPDATE SET
			cert_path = excluded.cert_path,
			issued_at = excluded.issued_at,
			expires_at = excluded.expires_at,
			status = excluded.status,
			last_renewal = CURRENT_TIMESTAMP
	`, domain, certDir, time.Now(), time.Now().Add(90*24*time.Hour))

	return nil
}

// CheckRenewals checks all certificates for renewal
func (cm *CertificateManager) CheckRenewals() error {
	rows, err := cm.db.Query(`
		SELECT domain, cert_path, expires_at
		FROM ssl_certificates
		WHERE auto_renew = 1 AND status = 'active'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domain, certPath string
		var expiresAt time.Time

		if err := rows.Scan(&domain, &certPath, &expiresAt); err != nil {
			continue
		}

		// Check if renewal is needed
		if time.Until(expiresAt) < time.Duration(CertRenewalDays)*24*time.Hour {
			log.Printf("Renewing certificate for %s", domain)
			if err := cm.ObtainCertificate(domain); err != nil {
				log.Printf("Failed to renew certificate for %s: %v", domain, err)
				cm.db.Exec(`
					UPDATE ssl_certificates
					SET status = 'failed'
					WHERE domain = ?
				`, domain)
			}
		}
	}

	return nil
}

// needsRenewal checks if a certificate needs renewal
func (cm *CertificateManager) needsRenewal(cert *tls.Certificate) bool {
	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		var err error
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return true
		}
	}

	if cert.Leaf == nil {
		return true
	}

	// Renew if less than 30 days until expiry
	return time.Until(cert.Leaf.NotAfter) < time.Duration(CertRenewalDays)*24*time.Hour
}

// generateSelfSignedCert generates a self-signed certificate
func (cm *CertificateManager) generateSelfSignedCert(domain string) (*tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"CASRAD Self-Signed"},
			CommonName:   domain,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}

	// Add IP addresses if domain looks like an IP
	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}

// VerifyDomain verifies ownership of a domain
func (cm *CertificateManager) VerifyDomain(domain string, method string) (string, string, error) {
	token := cm.generateVerificationToken()

	switch method {
	case "dns":
		// TXT record verification
		return fmt.Sprintf("_casrad-verify.%s", domain), token, nil

	case "http":
		// HTTP file verification
		return fmt.Sprintf("http://%s/.well-known/casrad-verify.txt", domain), token, nil

	case "cname":
		// CNAME verification
		return fmt.Sprintf("verify.%s", domain), token, nil

	default:
		return "", "", fmt.Errorf("unsupported verification method: %s", method)
	}
}

// generateVerificationToken generates a random verification token
func (cm *CertificateManager) generateVerificationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// HTTPSRedirect returns an HTTP handler that redirects to HTTPS
func (cm *CertificateManager) HTTPSRedirect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle ACME challenges
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			// Let autocert handle it
			if cm.autocertMgr != nil {
				cm.autocertMgr.HTTPHandler(nil).ServeHTTP(w, r)
				return
			}
		}

		// Redirect to HTTPS
		host := r.Host
		if cm.httpsPort != 443 {
			host = fmt.Sprintf("%s:%d", r.Host, cm.httpsPort)
		}

		http.Redirect(w, r, "https://"+host+r.RequestURI, http.StatusMovedPermanently)
	})
}