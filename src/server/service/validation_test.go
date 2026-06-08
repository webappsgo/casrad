// Package service - Tests for input validation and sanitization functions.
// Covers: ValidatePassword, ValidateUsername, ValidateEmail, ValidateString,
// ValidateStringWithMin, SanitizeFilename, SanitizeURL, SanitizeInput,
// StripHTML, EscapeHTML, NormalizeNewlines, TrimInput, TrimInputs,
// ValidateRegistration, ValidateLogin, ValidationResult helpers.
package service

import (
	"strings"
	"testing"
)

// --- ValidatePassword ---

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid_8chars", input: "abcdefgh"},
		{name: "valid_complex", input: "MyS3cur3P@ss!"},
		{name: "valid_128chars", input: strings.Repeat("a", 128)},
		{name: "too_short_7", input: "abcdefg", wantErr: ErrPasswordTooShort},
		{name: "too_short_empty", input: "", wantErr: ErrPasswordTooShort},
		{name: "too_long_129", input: strings.Repeat("a", 129), wantErr: ErrPasswordTooLong},
		{name: "leading_space", input: " abcdefgh", wantErr: ErrPasswordLeadingWhitespace},
		{name: "trailing_space", input: "abcdefgh ", wantErr: ErrPasswordTrailingWhitespace},
		{name: "leading_tab", input: "\tabcdefgh", wantErr: ErrPasswordLeadingWhitespace},
		{name: "common_password", input: "password", wantErr: ErrPasswordCommonWord},
		{name: "common_password1", input: "password1", wantErr: ErrPasswordCommonWord},
		{name: "common_123456789", input: "123456789", wantErr: ErrPasswordCommonWord},
		// Common password check is case-insensitive
		{name: "common_upper", input: "PASSWORD", wantErr: ErrPasswordCommonWord},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePassword(tc.input)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Errorf("ValidatePassword(%q) = %v, want %v", tc.input, err, tc.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePassword(%q) unexpected error: %v", tc.input, err)
				}
			}
		})
	}
}

// --- ValidateUsername ---

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name     string
		input    string
		wantOut  string
	}{
		{name: "simple", input: "alice", wantOut: "alice"},
		{name: "with_numbers", input: "alice123", wantOut: "alice123"},
		{name: "with_hyphen", input: "alice-bob", wantOut: "alice-bob"},
		{name: "with_underscore", input: "alice_bob", wantOut: "alice_bob"},
		{name: "3chars", input: "abc", wantOut: "abc"},
		{name: "32chars", input: strings.Repeat("a", 32), wantOut: strings.Repeat("a", 32)},
		// Whitespace is trimmed
		{name: "trimmed_spaces", input: "  alice  ", wantOut: "alice"},
	}
	for _, tc := range valid {
		tc := tc
		t.Run("valid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateUsername(tc.input)
			if err != nil {
				t.Errorf("ValidateUsername(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.wantOut {
				t.Errorf("ValidateUsername(%q) = %q, want %q", tc.input, got, tc.wantOut)
			}
		})
	}

	invalid := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "too_short_2", input: "ab", wantErr: ErrUsernameTooShort},
		{name: "empty", input: "", wantErr: ErrUsernameTooShort},
		{name: "too_long_33", input: strings.Repeat("a", 33), wantErr: ErrUsernameTooLong},
		// Spaces and special chars are rejected; uppercase is allowed by the regex
		{name: "space", input: "alice bob", wantErr: ErrUsernameInvalidChars},
		{name: "at_sign", input: "alice@bob", wantErr: ErrUsernameInvalidChars},
		{name: "reserved_admin", input: "admin", wantErr: ErrUsernameReserved},
		{name: "reserved_root", input: "root", wantErr: ErrUsernameReserved},
		{name: "reserved_casrad", input: "casrad", wantErr: ErrUsernameReserved},
		{name: "reserved_api", input: "api", wantErr: ErrUsernameReserved},
		{name: "reserved_login", input: "login", wantErr: ErrUsernameReserved},
		// Reserved check is case-insensitive
		{name: "reserved_admin_upper", input: "ADMIN", wantErr: ErrUsernameReserved},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateUsername(tc.input)
			if err != tc.wantErr {
				t.Errorf("ValidateUsername(%q) = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

// --- ValidateEmail ---

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name    string
		input   string
		wantOut string
	}{
		{name: "simple", input: "user@example.com", wantOut: "user@example.com"},
		{name: "uppercase_lowercased", input: "User@Example.COM", wantOut: "user@example.com"},
		{name: "with_plus", input: "user+tag@example.com", wantOut: "user+tag@example.com"},
		{name: "subdomain", input: "user@mail.example.co.uk", wantOut: "user@mail.example.co.uk"},
		// Whitespace trimmed and lowercased
		{name: "whitespace_trimmed", input: "  USER@EXAMPLE.COM  ", wantOut: "user@example.com"},
	}
	for _, tc := range valid {
		tc := tc
		t.Run("valid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateEmail(tc.input)
			if err != nil {
				t.Errorf("ValidateEmail(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.wantOut {
				t.Errorf("ValidateEmail(%q) = %q, want %q", tc.input, got, tc.wantOut)
			}
		})
	}

	invalid := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrInputEmpty},
		{name: "whitespace_only", input: "   ", wantErr: ErrInputEmpty},
		{name: "missing_at", input: "userexample.com", wantErr: ErrEmailInvalidFormat},
		{name: "missing_domain", input: "user@", wantErr: ErrEmailInvalidFormat},
		{name: "missing_tld", input: "user@example", wantErr: ErrEmailInvalidFormat},
		{name: "double_at", input: "user@@example.com", wantErr: ErrEmailInvalidFormat},
		// 251 a's + "@x.co" = 256 characters total — exceeds the 255 limit
		{name: "too_long_256", input: strings.Repeat("a", 251) + "@x.co", wantErr: ErrEmailTooLong},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateEmail(tc.input)
			if err != tc.wantErr {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

// --- ValidateString ---

func TestValidateString(t *testing.T) {
	t.Parallel()

	t.Run("required_empty_fails", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateString("", 100, true)
		if err != ErrInputEmpty {
			t.Errorf("ValidateString empty required = %v, want ErrInputEmpty", err)
		}
	})

	t.Run("not_required_empty_ok", func(t *testing.T) {
		t.Parallel()
		got, err := ValidateString("", 100, false)
		if err != nil {
			t.Errorf("ValidateString empty optional unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("ValidateString empty optional = %q, want empty", got)
		}
	})

	t.Run("exceeds_max_fails", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateString(strings.Repeat("a", 101), 100, false)
		if err != ErrInputTooLong {
			t.Errorf("ValidateString over-max = %v, want ErrInputTooLong", err)
		}
	})

	t.Run("at_max_ok", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateString(strings.Repeat("a", 100), 100, false)
		if err != nil {
			t.Errorf("ValidateString at-max unexpected error: %v", err)
		}
	})

	t.Run("whitespace_trimmed", func(t *testing.T) {
		t.Parallel()
		got, err := ValidateString("  hello  ", 100, false)
		if err != nil {
			t.Fatalf("ValidateString trim unexpected error: %v", err)
		}
		if got != "hello" {
			t.Errorf("ValidateString trim = %q, want %q", got, "hello")
		}
	})
}

// --- ValidateStringWithMin ---

func TestValidateStringWithMin(t *testing.T) {
	t.Parallel()

	t.Run("below_min", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateStringWithMin("ab", 3, 10)
		if err != ErrInputTooShort {
			t.Errorf("ValidateStringWithMin below min = %v, want ErrInputTooShort", err)
		}
	})

	t.Run("at_min", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateStringWithMin("abc", 3, 10)
		if err != nil {
			t.Errorf("ValidateStringWithMin at min unexpected error: %v", err)
		}
	})

	t.Run("above_max", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateStringWithMin(strings.Repeat("a", 11), 3, 10)
		if err != ErrInputTooLong {
			t.Errorf("ValidateStringWithMin above max = %v, want ErrInputTooLong", err)
		}
	})

	t.Run("zero_max_no_upper_limit", func(t *testing.T) {
		t.Parallel()
		_, err := ValidateStringWithMin(strings.Repeat("a", 1000), 3, 0)
		if err != nil {
			t.Errorf("ValidateStringWithMin zero max unexpected error: %v", err)
		}
	})
}

// --- SanitizeFilename ---

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "clean", input: "track.mp3", want: "track.mp3"},
		{name: "path_sep_removed", input: "../../etc/passwd", want: "etcpasswd"},
		{name: "backslash_removed", input: "foo\\bar", want: "foobar"},
		{name: "null_byte_removed", input: "foo\x00bar", want: "foobar"},
		{name: "tilde_removed", input: "~/secrets", want: "secrets"},
		{name: "pipes_removed", input: "foo|bar", want: "foobar"},
		{name: "angle_brackets", input: "<script>", want: "script"},
		{name: "colon_removed", input: "C:file", want: "Cfile"},
		{name: "question_mark", input: "foo?bar", want: "foobar"},
		{name: "asterisk", input: "foo*bar", want: "foobar"},
		{name: "whitespace_trimmed", input: "  song.mp3  ", want: "song.mp3"},
		// Double-dot pattern removed
		{name: "double_dot_stripped", input: "a..b", want: "ab"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeFilename(tc.input)
			if got != tc.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- SanitizeURL ---

func TestSanitizeURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "http_ok", input: "http://example.com", want: "http://example.com"},
		{name: "https_ok", input: "https://example.com/path", want: "https://example.com/path"},
		{name: "javascript_blocked", input: "javascript:alert(1)", want: ""},
		{name: "javascript_uppercase", input: "JAVASCRIPT:alert(1)", want: ""},
		{name: "data_uri_blocked", input: "data:text/html,<script>", want: ""},
		{name: "vbscript_blocked", input: "vbscript:msgbox(1)", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "whitespace_trimmed", input: "  https://example.com  ", want: "https://example.com"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeURL(tc.input)
			if got != tc.want {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- SanitizeInput ---

func TestSanitizeInput(t *testing.T) {
	t.Parallel()

	t.Run("collapses_internal_whitespace", func(t *testing.T) {
		t.Parallel()
		got := SanitizeInput("hello   world")
		if got != "hello world" {
			t.Errorf("SanitizeInput = %q, want %q", got, "hello world")
		}
	})

	t.Run("trims_outer_whitespace", func(t *testing.T) {
		t.Parallel()
		got := SanitizeInput("  hello  ")
		if got != "hello" {
			t.Errorf("SanitizeInput trim = %q, want %q", got, "hello")
		}
	})

	t.Run("empty_stays_empty", func(t *testing.T) {
		t.Parallel()
		got := SanitizeInput("")
		if got != "" {
			t.Errorf("SanitizeInput empty = %q, want empty", got)
		}
	})
}

// --- TrimInput / TrimInputs ---

func TestTrimInput(t *testing.T) {
	t.Parallel()

	if got := TrimInput("  hello  "); got != "hello" {
		t.Errorf("TrimInput = %q, want %q", got, "hello")
	}
	if got := TrimInput(""); got != "" {
		t.Errorf("TrimInput empty = %q, want empty", got)
	}
}

func TestTrimInputs(t *testing.T) {
	t.Parallel()

	a, b := "  hello  ", "  world  "
	TrimInputs(&a, &b)
	if a != "hello" {
		t.Errorf("TrimInputs a = %q, want %q", a, "hello")
	}
	if b != "world" {
		t.Errorf("TrimInputs b = %q, want %q", b, "world")
	}

	// Nil pointer should not panic
	TrimInputs(nil)
}

// --- StripHTML ---

func TestStripHTML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"<b>bold</b>", "bold"},
		{"<script>alert(1)</script>", "alert(1)"},
		{"no html", "no html"},
		{"<a href='x'>link</a>", "link"},
		{"", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := StripHTML(tc.input)
			if got != tc.want {
				t.Errorf("StripHTML(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- EscapeHTML ---

func TestEscapeHTML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{"no escaping needed", "no escaping needed"},
		{"", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := EscapeHTML(tc.input)
			if got != tc.want {
				t.Errorf("EscapeHTML(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- NormalizeNewlines ---

func TestNormalizeNewlines(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "crlf", input: "a\r\nb", want: "a\nb"},
		{name: "cr_only", input: "a\rb", want: "a\nb"},
		{name: "lf_unchanged", input: "a\nb", want: "a\nb"},
		{name: "mixed", input: "a\r\nb\rc\nd", want: "a\nb\nc\nd"},
		{name: "empty", input: "", want: ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeNewlines(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeNewlines(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- ValidationResult ---

func TestValidationResult(t *testing.T) {
	t.Parallel()

	t.Run("new_is_valid", func(t *testing.T) {
		t.Parallel()
		r := NewValidationResult()
		if !r.Valid {
			t.Error("new ValidationResult should be valid")
		}
		if r.HasErrors() {
			t.Error("new ValidationResult should have no errors")
		}
	})

	t.Run("add_error_marks_invalid", func(t *testing.T) {
		t.Parallel()
		r := NewValidationResult()
		r.AddError("field", "problem")
		if r.Valid {
			t.Error("ValidationResult should be invalid after AddError")
		}
		if !r.HasErrors() {
			t.Error("ValidationResult HasErrors should be true")
		}
		if r.Errors["field"] != "problem" {
			t.Errorf("ValidationResult error = %q, want %q", r.Errors["field"], "problem")
		}
	})

	t.Run("multiple_errors", func(t *testing.T) {
		t.Parallel()
		r := NewValidationResult()
		r.AddError("email", "invalid email")
		r.AddError("password", "too short")
		if len(r.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(r.Errors))
		}
	})
}

// --- ValidateRegistration ---

func TestValidateRegistration(t *testing.T) {
	t.Parallel()

	t.Run("valid_registration", func(t *testing.T) {
		t.Parallel()
		r := ValidateRegistration("alice123", "alice@example.com", "s3cur3P@ss!", "s3cur3P@ss!")
		if r.HasErrors() {
			t.Errorf("valid registration has errors: %v", r.Errors)
		}
	})

	t.Run("password_mismatch", func(t *testing.T) {
		t.Parallel()
		r := ValidateRegistration("alice123", "alice@example.com", "s3cur3P@ss!", "different!")
		if !r.HasErrors() {
			t.Error("mismatched passwords should produce errors")
		}
		if _, ok := r.Errors["confirm_password"]; !ok {
			t.Error("confirm_password error expected")
		}
	})

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		r := ValidateRegistration("admin", "alice@example.com", "s3cur3P@ss!", "s3cur3P@ss!")
		if !r.HasErrors() {
			t.Error("reserved username should produce errors")
		}
		if _, ok := r.Errors["username"]; !ok {
			t.Error("username error expected")
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		t.Parallel()
		r := ValidateRegistration("alice123", "not-an-email", "s3cur3P@ss!", "s3cur3P@ss!")
		if !r.HasErrors() {
			t.Error("invalid email should produce errors")
		}
	})

	t.Run("common_password", func(t *testing.T) {
		t.Parallel()
		r := ValidateRegistration("alice123", "alice@example.com", "password123", "password123")
		if !r.HasErrors() {
			t.Error("common password should produce errors")
		}
	})
}

// --- ValidateLogin ---

func TestValidateLogin(t *testing.T) {
	t.Parallel()

	t.Run("valid_login", func(t *testing.T) {
		t.Parallel()
		r := ValidateLogin("alice123", "s3cur3P@ss!")
		if r.HasErrors() {
			t.Errorf("valid login has errors: %v", r.Errors)
		}
	})

	t.Run("empty_identifier", func(t *testing.T) {
		t.Parallel()
		r := ValidateLogin("", "s3cur3P@ss!")
		if !r.HasErrors() {
			t.Error("empty identifier should produce errors")
		}
		if _, ok := r.Errors["identifier"]; !ok {
			t.Error("identifier error expected")
		}
	})

	t.Run("empty_password", func(t *testing.T) {
		t.Parallel()
		r := ValidateLogin("alice", "")
		if !r.HasErrors() {
			t.Error("empty password should produce errors")
		}
	})

	t.Run("leading_space_password_rejected", func(t *testing.T) {
		t.Parallel()
		r := ValidateLogin("alice", " mypassword")
		if !r.HasErrors() {
			t.Error("password with leading space should produce errors")
		}
	})

	t.Run("trailing_space_password_rejected", func(t *testing.T) {
		t.Parallel()
		r := ValidateLogin("alice", "mypassword ")
		if !r.HasErrors() {
			t.Error("password with trailing space should produce errors")
		}
	})
}
