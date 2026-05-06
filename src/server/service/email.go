// Package service - Email service for SMTP sending
package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

var (
	ErrSMTPNotConfigured = errors.New("SMTP not configured")
	ErrSMTPConnection    = errors.New("failed to connect to SMTP server")
	ErrSMTPAuth          = errors.New("SMTP authentication failed")
	ErrSMTPSend          = errors.New("failed to send email")
)

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromName   string
	FromEmail  string
	UseTLS     bool
	SkipVerify bool
}

// EmailService handles email sending via SMTP
type EmailService struct {
	config     *SMTPConfig
	configured bool
}

// NewEmailService creates a new email service
func NewEmailService(config *SMTPConfig) *EmailService {
	configured := config != nil && config.Host != "" && config.FromEmail != ""
	return &EmailService{
		config:     config,
		configured: configured,
	}
}

// IsConfigured returns true if SMTP is configured
func (s *EmailService) IsConfigured() bool {
	return s.configured
}

// Configure configures the email service
func (s *EmailService) Configure(config *SMTPConfig) {
	s.config = config
	s.configured = config != nil && config.Host != "" && config.FromEmail != ""
}

// Send sends an email
func (s *EmailService) Send(to, subject, body string, isHTML bool) error {
	if !s.configured {
		return ErrSMTPNotConfigured
	}

	// Build message
	headers := make(map[string]string)
	headers["From"] = s.formatAddress(s.config.FromName, s.config.FromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["Date"] = time.Now().Format(time.RFC1123Z)
	headers["MIME-Version"] = "1.0"

	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return s.sendMail([]string{to}, []byte(msg.String()))
}

// SendVerification sends an email verification email
func (s *EmailService) SendVerification(email, code, baseURL string) error {
	if !s.configured {
		return ErrSMTPNotConfigured
	}

	verifyURL := fmt.Sprintf("%s/auth/verify?code=%s", baseURL, code)

	subject := "Verify your email address"
	body := fmt.Sprintf(`Hello,

Please verify your email address by clicking the link below:

%s

If you did not create an account, you can safely ignore this email.

This link will expire in 24 hours.

Best regards,
CASRAD
`, verifyURL)

	return s.Send(email, subject, body, false)
}

// SendPasswordReset sends a password reset email
func (s *EmailService) SendPasswordReset(email, code, baseURL string) error {
	if !s.configured {
		return ErrSMTPNotConfigured
	}

	resetURL := fmt.Sprintf("%s/auth/reset-password?code=%s", baseURL, code)

	subject := "Reset your password"
	body := fmt.Sprintf(`Hello,

You requested to reset your password. Click the link below to set a new password:

%s

If you did not request a password reset, you can safely ignore this email.

This link will expire in 1 hour.

Best regards,
CASRAD
`, resetURL)

	return s.Send(email, subject, body, false)
}

// SendNotification sends a notification email
func (s *EmailService) SendNotification(email, subject, body string) error {
	return s.Send(email, subject, body, false)
}

// SendWelcome sends a welcome email to new users
func (s *EmailService) SendWelcome(email, username, baseURL string) error {
	if !s.configured {
		return ErrSMTPNotConfigured
	}

	subject := "Welcome to CASRAD"
	body := fmt.Sprintf(`Hello %s,

Welcome to CASRAD! Your account has been created successfully.

You can log in at: %s/login

If you have any questions, feel free to reach out to support.

Best regards,
CASRAD
`, username, baseURL)

	return s.Send(email, subject, body, false)
}

// formatAddress formats an email address with optional name
func (s *EmailService) formatAddress(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

// sendMail sends the email using SMTP
func (s *EmailService) sendMail(to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create SMTP client
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPConnection, err)
	}
	defer conn.Close()

	// Wrap with TLS if configured
	if s.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         s.config.Host,
			InsecureSkipVerify: s.config.SkipVerify,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPConnection, err)
	}
	defer client.Close()

	// Try STARTTLS if available and not already using TLS
	if !s.config.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName:         s.config.Host,
				InsecureSkipVerify: s.config.SkipVerify,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("%w: STARTTLS failed: %v", ErrSMTPConnection, err)
			}
		}
	}

	// Authenticate if credentials provided
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: %v", ErrSMTPAuth, err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("%w: MAIL FROM failed: %v", ErrSMTPSend, err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("%w: RCPT TO failed: %v", ErrSMTPSend, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: DATA failed: %v", ErrSMTPSend, err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("%w: write failed: %v", ErrSMTPSend, err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("%w: close failed: %v", ErrSMTPSend, err)
	}

	return client.Quit()
}
