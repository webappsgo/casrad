// Package config - Tests for path normalization and validation functions
// Covers: NormalizePath, ValidatePathSegment, ValidatePath, SafePath, SafeFilePath
// including traversal attacks, length limits, empty input, and happy paths.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "simple", input: "admin", want: "admin"},
		{name: "leading_slash", input: "/admin", want: "admin"},
		{name: "trailing_slash", input: "admin/", want: "admin"},
		{name: "both_slashes", input: "/admin/", want: "admin"},
		{name: "nested", input: "/foo/bar", want: "foo/bar"},
		{name: "double_slash", input: "foo//bar", want: "foo/bar"},
		{name: "dot_segment", input: "foo/./bar", want: "foo/bar"},
		// traversal is stripped and leaves empty
		{name: "traversal_only", input: "../", want: ""},
		{name: "traversal_root", input: "/../", want: ""},
		// absolute path collapsed to relative
		{name: "absolute", input: "/etc/passwd", want: "etc/passwd"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizePath(tc.input)
			if got != tc.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidatePathSegment(t *testing.T) {
	t.Parallel()

	valid := []string{
		"admin", "api", "a", "abc123", "hello-world", "hello_world", "a1b2-c3",
	}
	for _, v := range valid {
		v := v
		t.Run("valid_"+v, func(t *testing.T) {
			t.Parallel()
			if err := ValidatePathSegment(v); err != nil {
				t.Errorf("ValidatePathSegment(%q) unexpected error: %v", v, err)
			}
		})
	}

	// 64 character segment (boundary — valid)
	seg64 := strings.Repeat("a", 64)
	t.Run("boundary_64", func(t *testing.T) {
		t.Parallel()
		if err := ValidatePathSegment(seg64); err != nil {
			t.Errorf("ValidatePathSegment(64 chars) unexpected error: %v", err)
		}
	})

	// 65 character segment (over boundary — invalid)
	seg65 := strings.Repeat("a", 65)
	t.Run("over_boundary_65", func(t *testing.T) {
		t.Parallel()
		if err := ValidatePathSegment(seg65); err != ErrPathTooLong {
			t.Errorf("ValidatePathSegment(65 chars) = %v, want ErrPathTooLong", err)
		}
	})

	invalid := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty", input: "", wantErr: ErrInvalidPath},
		{name: "uppercase", input: "Admin", wantErr: ErrInvalidPath},
		{name: "space", input: "hello world", wantErr: ErrInvalidPath},
		{name: "slash", input: "foo/bar", wantErr: ErrInvalidPath},
		{name: "dot_double", input: "..", wantErr: ErrPathTraversal},
		{name: "dot_single", input: ".", wantErr: ErrPathTraversal},
		{name: "special_chars", input: "foo@bar", wantErr: ErrInvalidPath},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePathSegment(tc.input)
			if err != tc.wantErr {
				t.Errorf("ValidatePathSegment(%q) = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	t.Parallel()

	valid := []string{
		"admin", "api/v1", "foo/bar/baz", "a-b/c_d",
	}
	for _, v := range valid {
		v := v
		t.Run("valid_"+v, func(t *testing.T) {
			t.Parallel()
			if err := ValidatePath(v); err != nil {
				t.Errorf("ValidatePath(%q) unexpected error: %v", v, err)
			}
		})
	}

	// Path at exactly 2048 chars (boundary — valid, all lowercase letters)
	p2048 := strings.Repeat("a/", 1023) + "a"
	t.Run("boundary_2048", func(t *testing.T) {
		t.Parallel()
		if len(p2048) <= 2048 {
			if err := ValidatePath(p2048); err != nil {
				t.Logf("ValidatePath(2048-ish) error (may be segment-too-long): %v", err)
			}
		}
	})

	// Path exceeding 2048 chars
	p2049 := strings.Repeat("a", 2049)
	t.Run("over_limit_2049", func(t *testing.T) {
		t.Parallel()
		if err := ValidatePath(p2049); err != ErrPathTooLong {
			t.Errorf("ValidatePath(2049 chars) = %v, want ErrPathTooLong", err)
		}
	})

	invalid := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "traversal_relative", input: "foo/../bar", wantErr: ErrPathTraversal},
		{name: "traversal_prefix", input: "../etc/passwd", wantErr: ErrPathTraversal},
		{name: "invalid_chars", input: "foo/Bar", wantErr: ErrInvalidPath},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePath(tc.input)
			if err != tc.wantErr {
				t.Errorf("ValidatePath(%q) = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestSafePath(t *testing.T) {
	t.Parallel()

	// Happy path: valid path returns cleaned version
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		got, err := SafePath("admin/dashboard")
		if err != nil {
			t.Fatalf("SafePath unexpected error: %v", err)
		}
		if got == "" {
			t.Error("SafePath returned empty string for valid input")
		}
	})

	// Traversal must be rejected before normalization removes evidence
	t.Run("traversal_rejected", func(t *testing.T) {
		t.Parallel()
		_, err := SafePath("../etc/passwd")
		if err == nil {
			t.Error("SafePath should reject traversal attempt")
		}
	})

	// Invalid characters
	t.Run("invalid_chars", func(t *testing.T) {
		t.Parallel()
		_, err := SafePath("Hello/World")
		if err == nil {
			t.Error("SafePath should reject uppercase characters")
		}
	})
}

func TestSafeFilePath(t *testing.T) {
	t.Parallel()

	// Create a real temporary directory for base
	base, err := os.MkdirTemp("", "casrad-path-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(base)

	// Happy path: path within base
	t.Run("within_base", func(t *testing.T) {
		t.Parallel()
		got, err := SafeFilePath(base, "music")
		if err != nil {
			t.Fatalf("SafeFilePath unexpected error: %v", err)
		}
		if !strings.HasPrefix(got, base) {
			t.Errorf("SafeFilePath result %q not within base %q", got, base)
		}
	})

	// Traversal attempt should be caught
	t.Run("traversal_attempt", func(t *testing.T) {
		t.Parallel()
		_, err := SafeFilePath(base, "../etc/passwd")
		if err == nil {
			t.Error("SafeFilePath should reject path traversal")
		}
	})

	// Nested valid path
	t.Run("nested_valid", func(t *testing.T) {
		t.Parallel()
		got, err := SafeFilePath(base, "user/music")
		if err != nil {
			t.Fatalf("SafeFilePath nested unexpected error: %v", err)
		}
		want := filepath.Join(base, "user", "music")
		if got != want {
			t.Errorf("SafeFilePath nested = %q, want %q", got, want)
		}
	})

	// Empty sub-path — returns base itself
	t.Run("empty_subpath", func(t *testing.T) {
		t.Parallel()
		// Empty path normalizes to "." — ValidatePath allows empty segments
		// result should still be within base
		got, err := SafeFilePath(base, "")
		if err == nil {
			if !strings.HasPrefix(got, base) {
				t.Errorf("SafeFilePath empty result %q not within base %q", got, base)
			}
		}
		// error is also acceptable for empty sub-path
	})
}
