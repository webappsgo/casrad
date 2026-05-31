// Package service provides server services
// See AI.md - Input validation and sanitization
package service

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// Validation errors
var (
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong           = errors.New("password cannot exceed 128 characters")
	ErrPasswordLeadingWhitespace = errors.New("password cannot start with whitespace")
	ErrPasswordTrailingWhitespace = errors.New("password cannot end with whitespace")
	ErrPasswordCommonWord        = errors.New("password is too common")
	ErrUsernameTooShort          = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong           = errors.New("username cannot exceed 32 characters")
	ErrUsernameInvalidChars      = errors.New("username can only contain letters, numbers, underscores, and hyphens")
	ErrUsernameReserved          = errors.New("username is reserved")
	ErrEmailInvalidFormat        = errors.New("invalid email address")
	ErrEmailTooLong              = errors.New("email cannot exceed 255 characters")
	ErrInputTooLong              = errors.New("input is too long")
	ErrInputTooShort             = errors.New("input is too short")
	ErrInputEmpty                = errors.New("input cannot be empty")
)

// Regex patterns for validation
var (
	usernameValidRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	emailValidRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Reserved usernames that cannot be used
var reservedUsernames = map[string]bool{
	"admin":      true,
	"administrator": true,
	"root":       true,
	"system":     true,
	"api":        true,
	"www":        true,
	"mail":       true,
	"ftp":        true,
	"support":    true,
	"help":       true,
	"info":       true,
	"contact":    true,
	"security":   true,
	"abuse":      true,
	"postmaster": true,
	"webmaster":  true,
	"hostmaster": true,
	"noreply":    true,
	"no-reply":   true,
	"null":       true,
	"undefined":  true,
	"anonymous":  true,
	"guest":      true,
	"test":       true,
	"demo":       true,
	"example":    true,
	"casrad":     true,
	"casapps":    true,
}

// Common passwords that should be rejected
var commonPasswords = map[string]bool{
	"password":    true,
	"password1":   true,
	"password123": true,
	"12345678":    true,
	"123456789":   true,
	"1234567890":  true,
	"qwerty":      true,
	"qwerty123":   true,
	"letmein":     true,
	"welcome":     true,
	"admin":       true,
	"admin123":    true,
	"login":       true,
	"abc123":      true,
	"monkey":      true,
	"master":      true,
	"dragon":      true,
	"111111":      true,
	"baseball":    true,
	"iloveyou":    true,
	"trustno1":    true,
	"sunshine":    true,
	"princess":    true,
	"football":    true,
	"shadow":      true,
	"superman":    true,
	"michael":     true,
	"jennifer":    true,
	"hunter":      true,
	"buster":      true,
}

// TrimInput trims leading and trailing whitespace from input
// This should be called on ALL user input fields
func TrimInput(input string) string {
	return strings.TrimSpace(input)
}

// TrimInputs trims whitespace from all provided inputs
func TrimInputs(inputs ...*string) {
	for _, input := range inputs {
		if input != nil {
			*input = strings.TrimSpace(*input)
		}
	}
}

// SanitizeInput trims whitespace and normalizes internal whitespace
func SanitizeInput(input string) string {
	// Trim leading/trailing
	input = strings.TrimSpace(input)
	// Normalize internal whitespace (collapse multiple spaces to one)
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(input, " ")
}

// ValidatePassword validates a password
// - Minimum 8 characters
// - Maximum 128 characters
// - Cannot start or end with whitespace
// - Cannot be a common password
func ValidatePassword(password string) error {
	// Check for leading whitespace
	if len(password) > 0 && unicode.IsSpace(rune(password[0])) {
		return ErrPasswordLeadingWhitespace
	}

	// Check for trailing whitespace
	if len(password) > 0 && unicode.IsSpace(rune(password[len(password)-1])) {
		return ErrPasswordTrailingWhitespace
	}

	// Check minimum length
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Check maximum length
	if len(password) > 128 {
		return ErrPasswordTooLong
	}

	// Check against common passwords (case-insensitive)
	if commonPasswords[strings.ToLower(password)] {
		return ErrPasswordCommonWord
	}

	return nil
}

// ValidateUsername validates a username
// - Minimum 3 characters
// - Maximum 32 characters
// - Only letters, numbers, underscores, hyphens
// - Not a reserved username
// Input is automatically trimmed
func ValidateUsername(username string) (string, error) {
	// Trim whitespace
	username = strings.TrimSpace(username)

	// Check minimum length
	if len(username) < 3 {
		return "", ErrUsernameTooShort
	}

	// Check maximum length
	if len(username) > 32 {
		return "", ErrUsernameTooLong
	}

	// Check character validity
	if !usernameValidRegex.MatchString(username) {
		return "", ErrUsernameInvalidChars
	}

	// Check reserved names (case-insensitive)
	if reservedUsernames[strings.ToLower(username)] {
		return "", ErrUsernameReserved
	}

	return username, nil
}

// ValidateEmail validates an email address
// Input is automatically trimmed and lowercased
func ValidateEmail(email string) (string, error) {
	// Trim whitespace
	email = strings.TrimSpace(email)

	// Convert to lowercase
	email = strings.ToLower(email)

	// Check if empty
	if email == "" {
		return "", ErrInputEmpty
	}

	// Check maximum length
	if len(email) > 255 {
		return "", ErrEmailTooLong
	}

	// Check format
	if !emailValidRegex.MatchString(email) {
		return "", ErrEmailInvalidFormat
	}

	return email, nil
}

// ValidateString validates a general string input
// - Trims whitespace
// - Checks maximum length
// - Optionally checks minimum length and emptiness
func ValidateString(input string, maxLen int, required bool) (string, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check if required
	if required && input == "" {
		return "", ErrInputEmpty
	}

	// Check maximum length
	if maxLen > 0 && len(input) > maxLen {
		return "", ErrInputTooLong
	}

	return input, nil
}

// ValidateStringWithMin validates a string with min/max length
func ValidateStringWithMin(input string, minLen, maxLen int) (string, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check minimum length
	if len(input) < minLen {
		return "", ErrInputTooShort
	}

	// Check maximum length
	if maxLen > 0 && len(input) > maxLen {
		return "", ErrInputTooLong
	}

	return input, nil
}

// SanitizeFilename sanitizes a filename by removing dangerous characters
func SanitizeFilename(filename string) string {
	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove other dangerous characters
	dangerous := []string{"..", "~", "|", "<", ">", ":", "\"", "?", "*"}
	for _, d := range dangerous {
		filename = strings.ReplaceAll(filename, d, "")
	}

	return filename
}

// SanitizeURL sanitizes a URL by removing dangerous characters
func SanitizeURL(url string) string {
	// Trim whitespace
	url = strings.TrimSpace(url)

	// Prevent javascript: URLs
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "javascript:") {
		return ""
	}
	if strings.HasPrefix(lower, "data:") {
		return ""
	}
	if strings.HasPrefix(lower, "vbscript:") {
		return ""
	}

	return url
}

// NormalizeNewlines converts all newline variations to \n
func NormalizeNewlines(input string) string {
	// Replace \r\n with \n
	input = strings.ReplaceAll(input, "\r\n", "\n")
	// Replace remaining \r with \n
	input = strings.ReplaceAll(input, "\r", "\n")
	return input
}

// StripHTML removes HTML tags from input (basic sanitization)
func StripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// EscapeHTML escapes HTML special characters
func EscapeHTML(input string) string {
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	return input
}

// ValidationResult holds multiple validation errors
type ValidationResult struct {
	Valid  bool
	Errors map[string]string
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}
}

// AddError adds an error to the validation result
func (v *ValidationResult) AddError(field, message string) {
	v.Valid = false
	v.Errors[field] = message
}

// HasErrors returns true if there are validation errors
func (v *ValidationResult) HasErrors() bool {
	return !v.Valid
}

// ValidateRegistration validates user registration input
func ValidateRegistration(username, email, password, confirmPassword string) *ValidationResult {
	result := NewValidationResult()

	// Validate username
	if validUsername, err := ValidateUsername(username); err != nil {
		result.AddError("username", err.Error())
	} else {
		username = validUsername
	}

	// Validate email
	if validEmail, err := ValidateEmail(email); err != nil {
		result.AddError("email", err.Error())
	} else {
		email = validEmail
	}

	// Validate password
	if err := ValidatePassword(password); err != nil {
		result.AddError("password", err.Error())
	}

	// Check password confirmation
	if password != confirmPassword {
		result.AddError("confirm_password", "passwords do not match")
	}

	return result
}

// ValidateLogin validates login input
func ValidateLogin(identifier, password string) *ValidationResult {
	result := NewValidationResult()

	// Trim identifier
	identifier = strings.TrimSpace(identifier)

	if identifier == "" {
		result.AddError("identifier", "username or email is required")
	}

	// NOTE: We do NOT trim password - user may have intentional internal spaces
	// But we check for leading/trailing whitespace
	if password == "" {
		result.AddError("password", "password is required")
	} else {
		if len(password) > 0 && unicode.IsSpace(rune(password[0])) {
			result.AddError("password", "password cannot start with whitespace")
		}
		if len(password) > 0 && unicode.IsSpace(rune(password[len(password)-1])) {
			result.AddError("password", "password cannot end with whitespace")
		}
	}

	return result
}
