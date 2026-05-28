// Package service - SMTP auto-detection and configuration
// See AI.md PART 18 for SMTP specification
package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// TLSMode represents SMTP TLS configuration
type TLSMode string

const (
	TLSModeAuto     TLSMode = "auto"     // Try STARTTLS, fallback to plain
	TLSModeStartTLS TLSMode = "starttls" // Require STARTTLS
	TLSModeTLS      TLSMode = "tls"      // Implicit TLS (port 465)
	TLSModeNone     TLSMode = "none"     // No encryption
)

// SMTPAutoDetectHosts are the hosts to try for auto-detection
var SMTPAutoDetectHosts = []string{
	"localhost",
	"127.0.0.1",
	"172.17.0.1", // Docker host
}

// SMTPAutoDetectPorts are the ports to try for auto-detection
var SMTPAutoDetectPorts = []int{587, 465, 25}

// SMTPSettings holds complete SMTP configuration
type SMTPSettings struct {
	Host       string
	Port       int
	Username   string
	Password   string
	TLSMode    TLSMode
	SkipVerify bool
	FromName   string
	FromEmail  string
}

// LoadSMTPFromEnv loads SMTP settings from environment variables
// Env vars override config file settings per PART 18
func LoadSMTPFromEnv(cfg *SMTPSettings) *SMTPSettings {
	if cfg == nil {
		cfg = &SMTPSettings{}
	}

	// SMTP_HOST
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg.Host = host
	}

	// SMTP_PORT
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			cfg.Port = p
		}
	}

	// SMTP_USERNAME
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		cfg.Username = username
	}

	// SMTP_PASSWORD
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		cfg.Password = password
	}

	// SMTP_TLS
	if tlsMode := os.Getenv("SMTP_TLS"); tlsMode != "" {
		switch strings.ToLower(tlsMode) {
		case "auto":
			cfg.TLSMode = TLSModeAuto
		case "starttls":
			cfg.TLSMode = TLSModeStartTLS
		case "tls":
			cfg.TLSMode = TLSModeTLS
		case "none":
			cfg.TLSMode = TLSModeNone
		}
	}

	// SMTP_FROM_NAME
	if name := os.Getenv("SMTP_FROM_NAME"); name != "" {
		cfg.FromName = name
	}

	// SMTP_FROM_EMAIL
	if email := os.Getenv("SMTP_FROM_EMAIL"); email != "" {
		cfg.FromEmail = email
	}

	return cfg
}

// AutoDetectSMTP attempts to detect a local SMTP server
// Per PART 18: Try localhost, 127.0.0.1, 172.17.0.1, gateway on ports 25, 587, 465
func AutoDetectSMTP() *SMTPSettings {
	hosts := append([]string{}, SMTPAutoDetectHosts...)

	// Add gateway IP if available
	if gateway := getGatewayIP(); gateway != "" {
		hosts = append(hosts, gateway)
	}

	// Try each host/port combination
	for _, host := range hosts {
		for _, port := range SMTPAutoDetectPorts {
			if testSMTPConnection(host, port) {
				return &SMTPSettings{
					Host:    host,
					Port:    port,
					TLSMode: TLSModeAuto,
				}
			}
		}
	}

	return nil
}

// testSMTPConnection tests if an SMTP server is available
func testSMTPConnection(host string, port int) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Try TCP connection with short timeout
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Try SMTP handshake
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return false
	}
	defer client.Close()

	// EHLO to verify it's an SMTP server
	err = client.Hello("localhost")
	if err != nil {
		return false
	}

	return true
}

// TestSMTPSettings tests if the given SMTP settings work
func TestSMTPSettings(cfg *SMTPSettings) error {
	if cfg == nil || cfg.Host == "" {
		return ErrSMTPNotConfigured
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))

	// Connect
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPConnection, err)
	}
	defer conn.Close()

	// Apply TLS if using implicit TLS (port 465 typically)
	if cfg.TLSMode == TLSModeTLS {
		tlsConfig := &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipVerify,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// Create SMTP client
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPConnection, err)
	}
	defer client.Close()

	// EHLO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("%w: EHLO failed: %v", ErrSMTPConnection, err)
	}

	// STARTTLS if required or auto
	if cfg.TLSMode == TLSModeStartTLS || cfg.TLSMode == TLSModeAuto {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName:         cfg.Host,
				InsecureSkipVerify: cfg.SkipVerify,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				if cfg.TLSMode == TLSModeStartTLS {
					return fmt.Errorf("%w: STARTTLS failed: %v", ErrSMTPConnection, err)
				}
				// Auto mode: continue without TLS
			}
		} else if cfg.TLSMode == TLSModeStartTLS {
			return fmt.Errorf("%w: server does not support STARTTLS", ErrSMTPConnection)
		}
	}

	// Authenticate if credentials provided
	if cfg.Username != "" && cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: %v", ErrSMTPAuth, err)
		}
	}

	return client.Quit()
}

// getGatewayIP tries to determine the default gateway IP
func getGatewayIP() string {
	// Try to get default gateway by connecting to a public IP
	// This doesn't actually send any data, just determines the route
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	// The gateway is typically on the same subnet
	ip := localAddr.IP.To4()
	if ip == nil {
		return ""
	}

	// Try common gateway patterns
	gateway := fmt.Sprintf("%d.%d.%d.1", ip[0], ip[1], ip[2])
	return gateway
}

// DefaultFromEmail returns the default from email based on FQDN
func DefaultFromEmail(fqdn string) string {
	if fqdn == "" {
		fqdn = "localhost"
	}
	return fmt.Sprintf("no-reply@%s", fqdn)
}
