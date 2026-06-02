// Package config - Tests for ParseBool, MustParseBool, IsTruthy, IsFalsy
// Covers: happy path, all truthy/falsy keywords, case insensitivity, whitespace,
// empty-string default propagation, invalid values, and panic guard on MustParseBool.
package config

import (
	"strings"
	"testing"
)

// parseBoolCase holds a single ParseBool test vector.
type parseBoolCase struct {
	input      string
	defaultVal bool
	wantVal    bool
	wantErr    bool
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	cases := []parseBoolCase{
		// Empty string returns default
		{input: "", defaultVal: true, wantVal: true},
		{input: "", defaultVal: false, wantVal: false},
		// Whitespace-only treated as empty
		{input: "   ", defaultVal: true, wantVal: true},
		{input: "\t", defaultVal: false, wantVal: false},

		// Canonical truthy values
		{input: "1", wantVal: true},
		{input: "y", wantVal: true},
		{input: "t", wantVal: true},
		{input: "yes", wantVal: true},
		{input: "true", wantVal: true},
		{input: "on", wantVal: true},
		{input: "ok", wantVal: true},
		{input: "okay", wantVal: true},
		{input: "enable", wantVal: true},
		{input: "enabled", wantVal: true},
		{input: "active", wantVal: true},
		{input: "accept", wantVal: true},
		{input: "accepted", wantVal: true},
		{input: "allow", wantVal: true},
		{input: "allowed", wantVal: true},
		{input: "grant", wantVal: true},
		{input: "affirmative", wantVal: true},

		// Canonical falsy values
		{input: "0", wantVal: false},
		{input: "n", wantVal: false},
		{input: "f", wantVal: false},
		{input: "no", wantVal: false},
		{input: "false", wantVal: false},
		{input: "off", wantVal: false},
		{input: "disable", wantVal: false},
		{input: "disabled", wantVal: false},
		{input: "inactive", wantVal: false},
		{input: "deny", wantVal: false},
		{input: "denied", wantVal: false},
		{input: "reject", wantVal: false},
		{input: "rejected", wantVal: false},
		{input: "never", wantVal: false},
		{input: "block", wantVal: false},
		{input: "revoke", wantVal: false},
		{input: "negative", wantVal: false},

		// Case insensitivity
		{input: "TRUE", wantVal: true},
		{input: "YES", wantVal: true},
		{input: "False", wantVal: false},
		{input: "NO", wantVal: false},
		{input: "ENABLED", wantVal: true},

		// Surrounding whitespace stripped
		{input: "  true  ", wantVal: true},
		{input: " false ", wantVal: false},

		// Invalid values produce errors
		{input: "maybe", wantErr: true},
		{input: "2", wantErr: true},
		{input: "yesno", wantErr: true},
		{input: "null", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		name := "input=" + tc.input
		if tc.input == "" {
			name = "empty_default=" + boolStr(tc.defaultVal)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBool(tc.input, tc.defaultVal)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseBool(%q, %v) expected error, got nil", tc.input, tc.defaultVal)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseBool(%q, %v) unexpected error: %v", tc.input, tc.defaultVal, err)
			}
			if got != tc.wantVal {
				t.Errorf("ParseBool(%q, %v) = %v, want %v", tc.input, tc.defaultVal, got, tc.wantVal)
			}
		})
	}
}

func TestMustParseBool_valid(t *testing.T) {
	t.Parallel()

	if !MustParseBool("yes", false) {
		t.Error("MustParseBool(yes) should return true")
	}
	if MustParseBool("no", true) {
		t.Error("MustParseBool(no) should return false")
	}
	// Empty propagates default
	if !MustParseBool("", true) {
		t.Error("MustParseBool empty should return default true")
	}
}

func TestMustParseBool_panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseBool with invalid input should panic")
		}
	}()
	MustParseBool("maybe", false)
}

func TestIsTruthy(t *testing.T) {
	t.Parallel()

	truthy := []string{"1", "yes", "true", "on", "enable", "enabled", "allow", "allowed", "ok", "okay", "active"}
	for _, v := range truthy {
		v := v
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if !IsTruthy(v) {
				t.Errorf("IsTruthy(%q) = false, want true", v)
			}
			// Case-insensitive
			if !IsTruthy(strings.ToUpper(v)) {
				t.Errorf("IsTruthy(%q) case variant = false, want true", strings.ToUpper(v))
			}
		})
	}

	// Non-truthy values
	nonTruthy := []string{"", "no", "false", "0", "off", "maybe", "2"}
	for _, v := range nonTruthy {
		v := v
		t.Run("not_truthy_"+v, func(t *testing.T) {
			t.Parallel()
			if IsTruthy(v) {
				t.Errorf("IsTruthy(%q) = true, want false", v)
			}
		})
	}
}

func TestIsFalsy(t *testing.T) {
	t.Parallel()

	falsy := []string{"0", "no", "false", "off", "disable", "disabled", "deny", "denied", "reject", "rejected", "inactive"}
	for _, v := range falsy {
		v := v
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if !IsFalsy(v) {
				t.Errorf("IsFalsy(%q) = false, want true", v)
			}
			// Case-insensitive
			if !IsFalsy(strings.ToUpper(v)) {
				t.Errorf("IsFalsy(%q) case variant = false, want true", strings.ToUpper(v))
			}
		})
	}

	// Non-falsy values
	nonFalsy := []string{"", "yes", "true", "1", "on", "maybe"}
	for _, v := range nonFalsy {
		v := v
		t.Run("not_falsy_"+v, func(t *testing.T) {
			t.Parallel()
			if IsFalsy(v) {
				t.Errorf("IsFalsy(%q) = true, want false", v)
			}
		})
	}
}

// TestIsTruthyIsFalsyMutuallyExclusive ensures no value is both truthy and falsy.
func TestIsTruthyIsFalsyMutuallyExclusive(t *testing.T) {
	t.Parallel()

	allValues := []string{
		"1", "y", "t", "yes", "true", "on", "ok", "okay", "enable", "enabled",
		"active", "accept", "accepted", "allow", "allowed", "grant", "affirmative",
		"0", "n", "f", "no", "false", "off", "disable", "disabled", "inactive",
		"deny", "denied", "reject", "rejected", "never", "block", "revoke", "negative",
	}
	for _, v := range allValues {
		v := v
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if IsTruthy(v) && IsFalsy(v) {
				t.Errorf("%q is both truthy and falsy — values must be mutually exclusive", v)
			}
		})
	}
}

// boolStr converts a bool to a string for test naming.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
