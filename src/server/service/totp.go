// Package service - TOTP (Time-based One-Time Password) service
// Per AI.md PART 3 - Authentication requirements
package service

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP errors
var (
	ErrTOTPAlreadyEnabled = errors.New("TOTP is already enabled")
	ErrTOTPNotEnabled     = errors.New("TOTP is not enabled")
	ErrTOTPInvalidCode    = errors.New("invalid TOTP code")
)

// TOTPService handles TOTP operations
type TOTPService struct {
	issuer string // App name shown in authenticator
}

// NewTOTPService creates a new TOTP service
func NewTOTPService(issuer string) *TOTPService {
	if issuer == "" {
		issuer = "CASRAD"
	}
	return &TOTPService{issuer: issuer}
}

// GenerateSecret generates a new TOTP secret for a user
// Returns the secret, QR code URL, and backup codes
func (s *TOTPService) GenerateSecret(accountName string) (secret, qrURL string, backupCodes []string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Generate backup codes
	backupCodes, err = s.generateBackupCodes(8)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	return key.Secret(), key.URL(), backupCodes, nil
}

// Validate validates a TOTP code against a secret
func (s *TOTPService) Validate(code, secret string) bool {
	// Validate with a 30-second window
	return totp.Validate(code, secret)
}

// ValidateWithWindow validates a TOTP code with a configurable window
// Window of 1 means we check current, previous, and next codes
func (s *TOTPService) ValidateWithWindow(code, secret string, window int) bool {
	if window < 0 {
		window = 1
	}

	// Try current time
	if totp.Validate(code, secret) {
		return true
	}

	// Try within window
	now := time.Now()
	for i := 1; i <= window; i++ {
		// Check past
		pastCode, err := totp.GenerateCodeCustom(secret, now.Add(time.Duration(-i*30)*time.Second), totp.ValidateOpts{
			Period:    30,
			Skew:      0,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && code == pastCode {
			return true
		}

		// Check future
		futureCode, err := totp.GenerateCodeCustom(secret, now.Add(time.Duration(i*30)*time.Second), totp.ValidateOpts{
			Period:    30,
			Skew:      0,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && code == futureCode {
			return true
		}
	}

	return false
}

// GetCurrentCode generates the current TOTP code for a secret (for testing)
func (s *TOTPService) GetCurrentCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// generateBackupCodes generates a set of one-time backup codes
func (s *TOTPService) generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 5)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		// Format: XXXX-XXXX (8 chars in 2 groups)
		code := strings.ToUpper(base32.StdEncoding.EncodeToString(bytes)[:8])
		codes[i] = code[:4] + "-" + code[4:]
	}
	return codes, nil
}

// ValidateBackupCode validates and consumes a backup code
// Returns true if valid, the remaining codes, and any error
func (s *TOTPService) ValidateBackupCode(code string, storedCodesJSON string) (valid bool, remaining string, err error) {
	// Normalize code (remove dashes, uppercase)
	code = strings.ToUpper(strings.ReplaceAll(code, "-", ""))

	// Parse stored codes (comma-separated, hashed)
	if storedCodesJSON == "" {
		return false, "", nil
	}

	storedCodes := strings.Split(storedCodesJSON, ",")
	for i, storedCode := range storedCodes {
		normalizedStored := strings.ToUpper(strings.ReplaceAll(storedCode, "-", ""))
		if code == normalizedStored {
			// Remove used code
			remaining := append(storedCodes[:i], storedCodes[i+1:]...)
			return true, strings.Join(remaining, ","), nil
		}
	}

	return false, storedCodesJSON, nil
}

// FormatBackupCodes formats backup codes for display
func (s *TOTPService) FormatBackupCodes(codes []string) string {
	return strings.Join(codes, "\n")
}

// IsValidSecret checks if a string is a valid TOTP secret
func (s *TOTPService) IsValidSecret(secret string) bool {
	if secret == "" {
		return false
	}
	// Base32 decode should work
	_, err := base32.StdEncoding.DecodeString(secret)
	return err == nil
}
