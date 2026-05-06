// Package server - Theme detection and switching
// See AI.md PART 16 for theme specification
package server

import (
	"net/http"
)

// Theme represents a UI theme
type Theme string

const (
	ThemeDark  Theme = "dark"  // Default theme
	ThemeLight Theme = "light"
	ThemeAuto  Theme = "auto" // Follows system preference
)

// GetTheme returns the current theme for a request
// See AI.md PART 16 - Dark is default, check localStorage/cookie first
func GetTheme(r *http.Request) Theme {
	// 1. Check cookie for user preference
	if cookie, err := r.Cookie("theme"); err == nil {
		switch Theme(cookie.Value) {
		case ThemeDark, ThemeLight, ThemeAuto:
			return Theme(cookie.Value)
		}
	}

	// 2. Default to dark
	return ThemeDark
}

// SetTheme sets the theme preference cookie
func SetTheme(w http.ResponseWriter, theme Theme) {
	http.SetCookie(w, &http.Cookie{
		Name:     "theme",
		Value:    string(theme),
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60, // 1 year
		HttpOnly: false,              // Allow JS access for instant switching
		SameSite: http.SameSiteLaxMode,
	})
}

// ThemeClass returns the CSS class for the HTML element
func ThemeClass(theme Theme) string {
	switch theme {
	case ThemeLight:
		return "theme-light"
	case ThemeAuto:
		return "theme-auto"
	default:
		return "theme-dark"
	}
}
