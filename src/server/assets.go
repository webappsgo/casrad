// Package server — embedded asset filesystems per AI.md PART 7.
// All web assets (templates, static files) are embedded at compile time so
// the binary is fully self-contained with no runtime filesystem dependency.
package server

import "embed"

// templateFS contains all Go HTML templates.
//
//go:embed template
var templateFS embed.FS

// staticFS contains all static web assets (CSS, JS, icons, manifests).
//
//go:embed static
var staticFS embed.FS
