// Package mode - Tests for mode parsing, environment detection, and flag accessors.
// Note: currentMode and debugEnabled are package-level globals. Tests that mutate them
// reset the state at the end to avoid polluting parallel runs in other packages.
// Within this package tests are run sequentially (no t.Parallel on mutating tests).
package mode

import (
	"os"
	"testing"
)

// --- FromString ---

func TestFromString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  Mode
	}{
		// Canonical values
		{input: "production", want: Production},
		{input: "development", want: Development},
		// Shortcuts
		{input: "prod", want: Production},
		{input: "dev", want: Development},
		// Case-insensitive
		{input: "PRODUCTION", want: Production},
		{input: "DEVELOPMENT", want: Development},
		{input: "DEV", want: Development},
		{input: "PROD", want: Production},
		// Whitespace stripped
		{input: "  dev  ", want: Development},
		{input: "  prod  ", want: Production},
		// Unknown falls back to Production
		{input: "", want: Production},
		{input: "staging", want: Production},
		{input: "test", want: Production},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := FromString(tc.input)
			if got != tc.want {
				t.Errorf("FromString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- Set / Get / IsProduction / IsDevelopment ---

func TestSetGet(t *testing.T) {
	// Mutates global — not parallel
	original := currentMode
	defer func() { currentMode = original }()

	Set(Development)
	if Get() != Development {
		t.Errorf("Get() = %q, want Development after Set(Development)", Get())
	}
	if !IsDevelopment() {
		t.Error("IsDevelopment() should be true after Set(Development)")
	}
	if IsProduction() {
		t.Error("IsProduction() should be false after Set(Development)")
	}

	Set(Production)
	if Get() != Production {
		t.Errorf("Get() = %q, want Production after Set(Production)", Get())
	}
	if !IsProduction() {
		t.Error("IsProduction() should be true after Set(Production)")
	}
	if IsDevelopment() {
		t.Error("IsDevelopment() should be false after Set(Production)")
	}
}

// --- SetDebug / IsDebug ---

func TestSetDebugIsDebug(t *testing.T) {
	// Mutates global — not parallel
	original := debugEnabled
	defer func() { debugEnabled = original }()

	SetDebug(true)
	if !IsDebug() {
		t.Error("IsDebug() should be true after SetDebug(true)")
	}

	SetDebug(false)
	if IsDebug() {
		t.Error("IsDebug() should be false after SetDebug(false)")
	}
}

// --- FromEnv ---

func TestFromEnv(t *testing.T) {
	// Mutates env — not parallel

	t.Run("dev_env", func(t *testing.T) {
		os.Setenv("MODE", "dev")
		defer os.Unsetenv("MODE")
		if got := FromEnv(); got != Development {
			t.Errorf("FromEnv() with MODE=dev = %q, want Development", got)
		}
	})

	t.Run("prod_env", func(t *testing.T) {
		os.Setenv("MODE", "production")
		defer os.Unsetenv("MODE")
		if got := FromEnv(); got != Production {
			t.Errorf("FromEnv() with MODE=production = %q, want Production", got)
		}
	})

	t.Run("unset_defaults_production", func(t *testing.T) {
		os.Unsetenv("MODE")
		if got := FromEnv(); got != Production {
			t.Errorf("FromEnv() with MODE unset = %q, want Production", got)
		}
	})
}

// --- DebugFromEnv ---

func TestDebugFromEnv(t *testing.T) {
	// Mutates env — not parallel

	t.Run("true_env", func(t *testing.T) {
		os.Setenv("DEBUG", "true")
		defer os.Unsetenv("DEBUG")
		if !DebugFromEnv() {
			t.Error("DebugFromEnv() with DEBUG=true should be true")
		}
	})

	t.Run("enabled_env", func(t *testing.T) {
		os.Setenv("DEBUG", "enabled")
		defer os.Unsetenv("DEBUG")
		if !DebugFromEnv() {
			t.Error("DebugFromEnv() with DEBUG=enabled should be true")
		}
	})

	t.Run("false_env", func(t *testing.T) {
		os.Setenv("DEBUG", "false")
		defer os.Unsetenv("DEBUG")
		if DebugFromEnv() {
			t.Error("DebugFromEnv() with DEBUG=false should be false")
		}
	})

	t.Run("unset_is_false", func(t *testing.T) {
		os.Unsetenv("DEBUG")
		if DebugFromEnv() {
			t.Error("DebugFromEnv() with DEBUG unset should be false")
		}
	})
}

// --- Init ---

func TestInit(t *testing.T) {
	// Mutates globals and env — not parallel
	originalMode := currentMode
	originalDebug := debugEnabled
	defer func() {
		currentMode = originalMode
		debugEnabled = originalDebug
	}()

	os.Setenv("MODE", "development")
	os.Setenv("DEBUG", "1")
	defer os.Unsetenv("MODE")
	defer os.Unsetenv("DEBUG")

	Init()

	if currentMode != Development {
		t.Errorf("Init() mode = %q, want Development", currentMode)
	}
	if !debugEnabled {
		t.Error("Init() debug should be true when DEBUG=1")
	}
}

// --- Mode constants have expected string values ---

func TestModeStringValues(t *testing.T) {
	t.Parallel()

	if string(Production) != "production" {
		t.Errorf("Production = %q, want %q", Production, "production")
	}
	if string(Development) != "development" {
		t.Errorf("Development = %q, want %q", Development, "development")
	}
}
