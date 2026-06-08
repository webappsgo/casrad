// Package display — Tests for display mode detection.
// Covers: Detect, IsTerminal, IsInteractive.
// Note: actual terminal detection depends on the environment; tests verify
// the logic paths rather than specific return values.
package display

import (
	"os"
	"testing"
)

// TestDetectServiceMode and TestDetectDefaultsToWebOrCLI use t.Setenv so they
// cannot be marked t.Parallel() — Go's testing framework forbids the combination.

func TestDetectServiceMode(t *testing.T) {
	t.Setenv("CASRAD_SERVICE", "1")
	if got := Detect(); got != ModeService {
		t.Errorf("Detect() = %q, want %q", got, ModeService)
	}
}

func TestDetectDefaultsToWebOrCLI(t *testing.T) {
	t.Setenv("CASRAD_SERVICE", "")
	mode := Detect()
	if mode != ModeWeb && mode != ModeCLI {
		t.Errorf("Detect() = %q, want ModeWeb or ModeCLI", mode)
	}
}

func TestIsTerminalDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = IsTerminal()
}

func TestIsTerminalFalseInCI(t *testing.T) {
	t.Parallel()
	// In Docker CI, stdout is not a TTY
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		t.Skip("cannot stat stdout")
	}
	isCharDevice := (fileInfo.Mode() & os.ModeCharDevice) != 0
	if IsTerminal() != isCharDevice {
		t.Errorf("IsTerminal() = %v, but os.ModeCharDevice check = %v", IsTerminal(), isCharDevice)
	}
}

func TestIsInteractiveNonInteractiveEnv(t *testing.T) {
	t.Setenv("CASRAD_NON_INTERACTIVE", "1")
	if IsInteractive() {
		t.Error("IsInteractive() should be false when CASRAD_NON_INTERACTIVE=1")
	}
}

func TestIsInteractiveDefault(t *testing.T) {
	t.Parallel()
	_ = IsInteractive()
}

func TestModeConstants(t *testing.T) {
	t.Parallel()
	expected := map[Mode]string{
		ModeWeb:     "web",
		ModeCLI:     "cli",
		ModeTUI:     "tui",
		ModeGUI:     "gui",
		ModeService: "service",
	}
	for mode, want := range expected {
		if string(mode) != want {
			t.Errorf("Mode constant %q = %q, want %q", mode, string(mode), want)
		}
	}
}
