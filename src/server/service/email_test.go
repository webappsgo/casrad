// Package service — Tests for EmailService.
// Covers: NewEmailService (configured/unconfigured), IsConfigured,
// Configure, formatAddress, Send (unconfigured path),
// SendVerification/SendPasswordReset/SendWelcome/SendNotification (unconfigured path),
// ErrSMTPNotConfigured is returned when SMTP not set up.
package service

import (
	"testing"
)

// --- NewEmailService ---

func TestNewEmailServiceNilConfigIsUnconfigured(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	if svc == nil {
		t.Fatal("NewEmailService(nil) returned nil")
	}
	if svc.IsConfigured() {
		t.Error("EmailService with nil config should not be configured")
	}
}

func TestNewEmailServiceEmptyHostIsUnconfigured(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "",
		FromEmail: "test@example.com",
	})
	if svc.IsConfigured() {
		t.Error("EmailService with empty host should not be configured")
	}
}

func TestNewEmailServiceEmptyFromEmailIsUnconfigured(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "smtp.example.com",
		FromEmail: "",
	})
	if svc.IsConfigured() {
		t.Error("EmailService with empty from_email should not be configured")
	}
}

func TestNewEmailServiceValidConfigIsConfigured(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "smtp.example.com",
		Port:      587,
		FromEmail: "noreply@example.com",
	})
	if !svc.IsConfigured() {
		t.Error("EmailService with valid config should be configured")
	}
}

// --- Configure ---

func TestConfigureSetsConfigured(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	if svc.IsConfigured() {
		t.Error("should start unconfigured")
	}
	svc.Configure(&SMTPConfig{
		Host:      "mail.example.com",
		Port:      25,
		FromEmail: "from@example.com",
	})
	if !svc.IsConfigured() {
		t.Error("after Configure with valid config, should be configured")
	}
}

func TestConfigureNilUnconfigures(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(&SMTPConfig{
		Host:      "smtp.example.com",
		FromEmail: "from@example.com",
	})
	svc.Configure(nil)
	if svc.IsConfigured() {
		t.Error("Configure(nil) should set service to unconfigured")
	}
}

// --- formatAddress ---

func TestFormatAddressWithName(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	got := svc.formatAddress("CASRAD", "noreply@example.com")
	want := "CASRAD <noreply@example.com>"
	if got != want {
		t.Errorf("formatAddress = %q, want %q", got, want)
	}
}

func TestFormatAddressWithoutName(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	got := svc.formatAddress("", "noreply@example.com")
	want := "noreply@example.com"
	if got != want {
		t.Errorf("formatAddress(no name) = %q, want %q", got, want)
	}
}

// --- Send returns ErrSMTPNotConfigured when unconfigured ---

func TestSendUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.Send("to@example.com", "Subject", "Body", false)
	if err != ErrSMTPNotConfigured {
		t.Errorf("Send(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}

func TestSendHTMLUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.Send("to@example.com", "Subject", "<b>Body</b>", true)
	if err != ErrSMTPNotConfigured {
		t.Errorf("Send HTML(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}

// --- SendVerification returns ErrSMTPNotConfigured when unconfigured ---

func TestSendVerificationUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.SendVerification("user@example.com", "abc123", "https://example.com")
	if err != ErrSMTPNotConfigured {
		t.Errorf("SendVerification(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}

// --- SendPasswordReset returns ErrSMTPNotConfigured when unconfigured ---

func TestSendPasswordResetUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.SendPasswordReset("user@example.com", "reset123", "https://example.com")
	if err != ErrSMTPNotConfigured {
		t.Errorf("SendPasswordReset(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}

// --- SendWelcome returns ErrSMTPNotConfigured when unconfigured ---

func TestSendWelcomeUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.SendWelcome("user@example.com", "alice", "https://example.com")
	if err != ErrSMTPNotConfigured {
		t.Errorf("SendWelcome(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}

// --- SendNotification delegates to Send ---

func TestSendNotificationUnconfiguredReturnsError(t *testing.T) {
	t.Parallel()
	svc := NewEmailService(nil)
	err := svc.SendNotification("user@example.com", "Notice", "Body text")
	if err != ErrSMTPNotConfigured {
		t.Errorf("SendNotification(unconfigured) error = %v, want ErrSMTPNotConfigured", err)
	}
}
