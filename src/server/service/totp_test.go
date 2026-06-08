// Package service — Tests for TOTP service.
// Covers: NewTOTPService, GenerateSecret, Validate, ValidateWithWindow,
// GetCurrentCode, ValidateBackupCode, FormatBackupCodes, IsValidSecret.
package service

import (
	"strings"
	"testing"
)

func TestNewTOTPServiceDefaultIssuer(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("")
	if svc.issuer != "CASRAD" {
		t.Errorf("empty issuer should default to CASRAD, got %q", svc.issuer)
	}
}

func TestNewTOTPServiceCustomIssuer(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("MyApp")
	if svc.issuer != "MyApp" {
		t.Errorf("issuer = %q, want MyApp", svc.issuer)
	}
}

func TestGenerateSecretReturnsNonEmpty(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, qrURL, backupCodes, err := svc.GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	if secret == "" {
		t.Error("secret should not be empty")
	}
	if qrURL == "" {
		t.Error("qrURL should not be empty")
	}
	if len(backupCodes) == 0 {
		t.Error("backupCodes should not be empty")
	}
}

func TestGenerateSecretBackupCodeCount(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	_, _, backupCodes, err := svc.GenerateSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	// Should generate 8 backup codes
	if len(backupCodes) != 8 {
		t.Errorf("backup codes count = %d, want 8", len(backupCodes))
	}
}

func TestGenerateSecretBackupCodeFormat(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	_, _, backupCodes, err := svc.GenerateSecret("user")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	for _, code := range backupCodes {
		// Format should be XXXX-XXXX (9 chars: 4+dash+4)
		if len(code) != 9 {
			t.Errorf("backup code %q has length %d, want 9", code, len(code))
		}
		if code[4] != '-' {
			t.Errorf("backup code %q missing dash at position 4", code)
		}
	}
}

func TestGenerateSecretQRURLContainsIssuer(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("CASRADTest")
	_, qrURL, _, err := svc.GenerateSecret("alice")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	if !strings.Contains(qrURL, "CASRADTest") {
		t.Errorf("qrURL %q should contain issuer CASRADTest", qrURL)
	}
}

func TestValidateCurrentCode(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	// Get current valid code and verify it validates
	code, err := svc.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode error: %v", err)
	}
	if !svc.Validate(code, secret) {
		t.Error("current TOTP code should validate")
	}
}

func TestValidateWrongCode(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	// Use an obviously wrong code
	if svc.Validate("000000", secret) {
		t.Error("000000 is extremely unlikely to be a valid current code")
	}
}

func TestValidateWithWindowCurrentCode(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("user")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	code, err := svc.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode error: %v", err)
	}
	if !svc.ValidateWithWindow(code, secret, 1) {
		t.Error("current code should validate with window=1")
	}
}

func TestValidateWithWindowNegativeWindow(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("user")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	code, err := svc.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode error: %v", err)
	}
	// Negative window should be treated as window=1
	if !svc.ValidateWithWindow(code, secret, -1) {
		t.Error("current code should validate even with negative window (clamped to 1)")
	}
}

func TestValidateWithWindowInvalidCode(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("user")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	// 000000 is almost certainly invalid
	if svc.ValidateWithWindow("000000", secret, 0) {
		t.Error("000000 should not validate with window=0")
	}
}

func TestGetCurrentCodeFormat(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secret, _, _, err := svc.GenerateSecret("user")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	code, err := svc.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode error: %v", err)
	}
	// TOTP codes are 6 digits
	if len(code) != 6 {
		t.Errorf("TOTP code length = %d, want 6", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("TOTP code %q contains non-digit character %q", code, string(c))
		}
	}
}

func TestIsValidSecretEmpty(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	if svc.IsValidSecret("") {
		t.Error("empty string should not be a valid secret")
	}
}

func TestIsValidSecretValidBase32(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	// Use a known-valid base32 string (A-Z, 2-7 only, proper length)
	// JBSWY3DPEHPK3PXP is a well-known test secret used in TOTP examples
	if !svc.IsValidSecret("JBSWY3DPEHPK3PXP") {
		t.Error("JBSWY3DPEHPK3PXP should be a valid base32 TOTP secret")
	}
}

func TestIsValidSecretInvalidBase32(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	// Base32 uses A-Z and 2-7; '1', '8', '9', '0' are invalid
	if svc.IsValidSecret("NOTBASE32!!!!") {
		t.Error("invalid base32 string should not be a valid secret")
	}
}

func TestValidateBackupCodeValid(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	storedCodes := "ABCD-EFGH,IJKL-MNOP,QRST-UVWX"
	valid, remaining, err := svc.ValidateBackupCode("ABCD-EFGH", storedCodes)
	if err != nil {
		t.Fatalf("ValidateBackupCode error: %v", err)
	}
	if !valid {
		t.Error("ABCD-EFGH should match stored code")
	}
	// After use, remaining should not contain the used code
	if strings.Contains(remaining, "ABCDEFGH") {
		t.Error("used code should be removed from remaining")
	}
}

func TestValidateBackupCodeCaseInsensitive(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	storedCodes := "ABCD-EFGH,IJKL-MNOP"
	valid, _, err := svc.ValidateBackupCode("abcd-efgh", storedCodes)
	if err != nil {
		t.Fatalf("ValidateBackupCode error: %v", err)
	}
	if !valid {
		t.Error("backup code validation should be case-insensitive")
	}
}

func TestValidateBackupCodeInvalid(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	storedCodes := "ABCD-EFGH,IJKL-MNOP"
	valid, remaining, err := svc.ValidateBackupCode("XXXX-XXXX", storedCodes)
	if err != nil {
		t.Fatalf("ValidateBackupCode error: %v", err)
	}
	if valid {
		t.Error("non-matching code should return false")
	}
	if remaining != storedCodes {
		t.Error("remaining codes should be unchanged when no match")
	}
}

func TestValidateBackupCodeEmptyStorage(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	valid, remaining, err := svc.ValidateBackupCode("ABCD-EFGH", "")
	if err != nil {
		t.Fatalf("ValidateBackupCode error: %v", err)
	}
	if valid {
		t.Error("empty storage should return false")
	}
	if remaining != "" {
		t.Error("remaining should be empty when storage is empty")
	}
}

func TestValidateBackupCodeReducesCount(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	storedCodes := "ABCD-EFGH,IJKL-MNOP,QRST-UVWX"
	_, remaining, err := svc.ValidateBackupCode("IJKL-MNOP", storedCodes)
	if err != nil {
		t.Fatalf("ValidateBackupCode error: %v", err)
	}
	remainingCodes := strings.Split(remaining, ",")
	if len(remainingCodes) != 2 {
		t.Errorf("remaining codes count = %d, want 2", len(remainingCodes))
	}
}

func TestFormatBackupCodes(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	codes := []string{"ABCD-EFGH", "IJKL-MNOP", "QRST-UVWX"}
	formatted := svc.FormatBackupCodes(codes)
	lines := strings.Split(formatted, "\n")
	if len(lines) != 3 {
		t.Errorf("formatted codes line count = %d, want 3", len(lines))
	}
	for i, line := range lines {
		if line != codes[i] {
			t.Errorf("line[%d] = %q, want %q", i, line, codes[i])
		}
	}
}

func TestFormatBackupCodesEmpty(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	formatted := svc.FormatBackupCodes(nil)
	if formatted != "" {
		t.Errorf("FormatBackupCodes(nil) = %q, want empty string", formatted)
	}
}

func TestGenerateSecretUniqueness(t *testing.T) {
	t.Parallel()
	svc := NewTOTPService("TestIssuer")
	secrets := make(map[string]bool)
	for i := 0; i < 10; i++ {
		secret, _, _, err := svc.GenerateSecret("user")
		if err != nil {
			t.Fatalf("GenerateSecret iteration %d error: %v", i, err)
		}
		if secrets[secret] {
			t.Errorf("duplicate secret generated on iteration %d", i)
		}
		secrets[secret] = true
	}
}
