// Package terminal — Tests for terminal size and detection utilities.
// Covers: GetSize (does not panic, returns valid dimensions), IsTerminal.
package terminal

import (
	"os"
	"testing"
)

// TestGetSizeDoesNotPanic verifies GetSize does not panic in a non-TTY environment.
func TestGetSizeDoesNotPanic(t *testing.T) {
	t.Parallel()
	_ = GetSize()
}

// TestGetSizeFallback verifies GetSize returns valid fallback dimensions when
// stdout is not a TTY (standard CI/Docker environment).
func TestGetSizeFallback(t *testing.T) {
	t.Parallel()
	size := GetSize()
	if size.Width <= 0 {
		t.Errorf("Width = %d, want > 0", size.Width)
	}
	if size.Height <= 0 {
		t.Errorf("Height = %d, want > 0", size.Height)
	}
}

// TestGetSizeDefaultFallbackValues verifies the fallback is 80x24 when not a TTY.
func TestGetSizeDefaultFallbackValues(t *testing.T) {
	t.Parallel()
	// In Docker/CI, stdout is not a TTY — GetSize should return fallback 80x24.
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		t.Skip("cannot stat stdout")
	}
	isCharDevice := (fileInfo.Mode() & os.ModeCharDevice) != 0
	if isCharDevice {
		t.Skip("stdout is a TTY; fallback test only valid in non-TTY environment")
	}
	size := GetSize()
	if size.Width != 80 {
		t.Errorf("fallback Width = %d, want 80", size.Width)
	}
	if size.Height != 24 {
		t.Errorf("fallback Height = %d, want 24", size.Height)
	}
}

// TestIsTerminalStdoutCI verifies IsTerminal matches os.ModeCharDevice check in CI.
func TestIsTerminalStdoutCI(t *testing.T) {
	t.Parallel()
	// int(os.Stdout.Fd()) should match the term.IsTerminal result
	fd := int(os.Stdout.Fd())
	got := IsTerminal(fd)
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		t.Skip("cannot stat stdout")
	}
	expected := (fileInfo.Mode() & os.ModeCharDevice) != 0
	if got != expected {
		t.Errorf("IsTerminal(%d) = %v, os.ModeCharDevice check = %v", fd, got, expected)
	}
}
