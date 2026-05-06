// Package terminal handles terminal utilities
// See AI.md for terminal handling
package terminal

import (
	"os"

	"golang.org/x/term"
)

// Size represents terminal dimensions
type Size struct {
	Width  int
	Height int
}

// GetSize returns the current terminal size
func GetSize() Size {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default fallback size
		return Size{Width: 80, Height: 24}
	}
	return Size{Width: width, Height: height}
}

// IsTerminal returns true if fd is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}
