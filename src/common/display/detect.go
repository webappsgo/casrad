// Package display handles display mode detection
// See AI.md for display detection
package display

import (
	"os"
)

// Mode represents the display mode
type Mode string

const (
	ModeWeb     Mode = "web"     // HTTP server mode
	ModeCLI     Mode = "cli"     // Command-line mode
	ModeTUI     Mode = "tui"     // Terminal UI mode
	ModeGUI     Mode = "gui"     // Graphical UI mode
	ModeService Mode = "service" // Background service mode
)

// Detect detects the current display mode
func Detect() Mode {
	// Check if running as a service
	if os.Getenv("CASRAD_SERVICE") == "1" {
		return ModeService
	}

	// Check if running in a terminal
	if IsTerminal() {
		// Could be CLI or TUI based on flags
		return ModeCLI
	}

	// Default to web server mode
	return ModeWeb
}

// IsTerminal returns true if stdout is a terminal
func IsTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// IsInteractive returns true if running interactively
func IsInteractive() bool {
	return IsTerminal() && os.Getenv("CASRAD_NON_INTERACTIVE") != "1"
}
