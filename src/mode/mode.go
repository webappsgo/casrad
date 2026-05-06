// Package mode handles application mode (production/development) and debug flag
// See AI.md PART 6 for mode specification
package mode

import (
	"os"
	"strings"

	"github.com/casapps/casrad/src/config"
)

// Mode represents the application mode
type Mode string

const (
	Production  Mode = "production"
	Development Mode = "development"
)

var (
	currentMode  Mode = Production
	debugEnabled bool = false
)

// Set sets the application mode
func Set(m Mode) {
	currentMode = m
}

// Get returns the current application mode
func Get() Mode {
	return currentMode
}

// SetDebug sets the debug flag
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// IsDebug returns true if debug mode is enabled
func IsDebug() bool {
	return debugEnabled
}

// IsDevelopment returns true if in development mode
func IsDevelopment() bool {
	return currentMode == Development
}

// IsProduction returns true if in production mode
func IsProduction() bool {
	return currentMode == Production
}

// FromString converts a string to Mode with shortcuts
// Accepts: dev, development, prod, production
func FromString(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dev", "development":
		return Development
	case "prod", "production":
		return Production
	default:
		return Production
	}
}

// FromEnv detects mode from environment variable
func FromEnv() Mode {
	return FromString(os.Getenv("MODE"))
}

// DebugFromEnv detects debug flag from environment variable
func DebugFromEnv() bool {
	return config.IsTruthy(os.Getenv("DEBUG"))
}

// Init initializes the mode and debug flag from environment
// This is called first; CLI flags can override later
func Init() {
	currentMode = FromEnv()
	debugEnabled = DebugFromEnv()
}
