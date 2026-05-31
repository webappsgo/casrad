// Package banner provides responsive startup banner
// See AI.md PART 15 for startup banner specification
package banner

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Config holds banner configuration
type Config struct {
	AppName   string
	Version   string
	CommitID  string
	BuildDate string
	// "production" or "development"
	Mode      string
	Debug     bool
	URLs      []string
	ListenURL string
}

// Print prints a responsive startup banner
// Adapts to terminal width per PART 15 spec
func Print(w io.Writer, cfg Config) {
	width := getTerminalWidth()

	switch {
	case width >= 80:
		printFullBanner(w, cfg)
	case width >= 60:
		printCompactBanner(w, cfg)
	case width >= 40:
		printMinimalBanner(w, cfg)
	default:
		printMicroBanner(w, cfg)
	}
}

// getTerminalWidth returns terminal width or 80 as default
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		return 80
	}
	return width
}

// printFullBanner prints full banner with ASCII art (≥80 cols)
func printFullBanner(w io.Writer, cfg Config) {
	fmt.Fprintln(w, "╭─────────────────────────────────────────────────────────────╮")
	fmt.Fprintf(w, "│  🚀 %s · 📦 %s%s│\n", cfg.AppName, cfg.Version, padTo(60-len(cfg.AppName)-len(cfg.Version)-8))
	fmt.Fprintln(w, "├─────────────────────────────────────────────────────────────┤")
	printModeLine(w, cfg.Mode, cfg.Debug, true)
	fmt.Fprintln(w, "├─────────────────────────────────────────────────────────────┤")

	// Print URLs
	for _, url := range cfg.URLs {
		icon := getURLIcon(url)
		fmt.Fprintf(w, "│  %s %s%s│\n", icon, url, padTo(58-len(url)))
	}

	fmt.Fprintln(w, "├─────────────────────────────────────────────────────────────┤")
	if cfg.ListenURL != "" {
		fmt.Fprintf(w, "│  📡 Listening on %s%s│\n", cfg.ListenURL, padTo(44-len(cfg.ListenURL)))
	}
	fmt.Fprintln(w, "╰─────────────────────────────────────────────────────────────╯")
	fmt.Fprintln(w)
}

// printCompactBanner prints compact banner without ASCII art (60-79 cols)
func printCompactBanner(w io.Writer, cfg Config) {
	fmt.Fprintf(w, "🚀 %s v%s\n", cfg.AppName, cfg.Version)
	printModeLineCompact(w, cfg.Mode, cfg.Debug)
	for _, url := range cfg.URLs {
		icon := getURLIcon(url)
		fmt.Fprintf(w, "%s %s\n", icon, url)
	}
	fmt.Fprintln(w)
}

// printMinimalBanner prints minimal banner (40-59 cols)
func printMinimalBanner(w io.Writer, cfg Config) {
	fmt.Fprintf(w, "%s %s\n", cfg.AppName, cfg.Version)
	for _, url := range cfg.URLs {
		fmt.Println(extractHostPort(url))
	}
}

// printMicroBanner prints micro banner (<40 cols)
func printMicroBanner(w io.Writer, cfg Config) {
	if len(cfg.URLs) > 0 {
		fmt.Fprintf(w, "%s %s\n", cfg.AppName, extractHostPort(cfg.URLs[0]))
	} else {
		fmt.Fprintln(w, cfg.AppName)
	}
}

// printModeLine prints the mode line in full format
func printModeLine(w io.Writer, mode string, debug, useIcons bool) {
	icon := "🔒"
	if mode == "development" {
		icon = "🔧"
	}

	modeText := fmt.Sprintf("Running in mode: %s", mode)
	if debug {
		modeText += " [debugging]"
	}

	if useIcons {
		fmt.Fprintf(w, "│  %s %s%s│\n", icon, modeText, padTo(58-len(modeText)))
	} else {
		fmt.Fprintf(w, "│  %s%s│\n", modeText, padTo(59-len(modeText)))
	}
}

// printModeLineCompact prints the mode line in compact format
func printModeLineCompact(w io.Writer, mode string, debug bool) {
	icon := "🔒"
	if mode == "development" {
		icon = "🔧"
	}

	modeText := fmt.Sprintf("Running in mode: %s", mode)
	if debug {
		modeText += " [debugging]"
	}

	fmt.Fprintf(w, "%s %s\n", icon, modeText)
}

// getURLIcon returns the appropriate icon for a URL
func getURLIcon(url string) string {
	switch {
	case strings.Contains(url, ".onion"):
		return "🧅 Tor   "
	case strings.Contains(url, ".i2p"):
		return "🔗 I2P   "
	case strings.HasPrefix(url, "https://"):
		return "🔐 HTTPS "
	case strings.Contains(url, "["):
		return "🌍 IPv6  "
	default:
		return "🌐 HTTP  "
	}
}

// extractHostPort extracts host:port from a URL
func extractHostPort(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}

// padTo returns spaces to pad to n characters
func padTo(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// PrintSetupToken prints the setup token box (first run only)
func PrintSetupToken(w io.Writer, token string) {
	width := getTerminalWidth()
	if width < 60 {
		// Compact for narrow terminals
		fmt.Fprintln(w, "🔑 SETUP REQUIRED")
		fmt.Fprintf(w, "Setup Token: %s\n", token)
		fmt.Fprintln(w, "Go to /admin and enter this token.")
		fmt.Fprintln(w, "This token will only be shown ONCE.")
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "┌─────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(w, "│  🔑 SETUP REQUIRED                                          │")
	fmt.Fprintln(w, "├─────────────────────────────────────────────────────────────┤")
	fmt.Fprintf(w, "│  Setup Token: %s%s│\n", token, padTo(46-len(token)))
	fmt.Fprintln(w, "│                                                             │")
	fmt.Fprintln(w, "│  Go to /admin and enter this token to complete setup.       │")
	fmt.Fprintln(w, "│  This token will only be shown ONCE.                        │")
	fmt.Fprintln(w, "└─────────────────────────────────────────────────────────────┘")
}

// PrintLegacy prints a simple banner (backwards compatible)
func PrintLegacy(w io.Writer, version, commitID, buildDate string) {
	cfg := Config{
		AppName:   "CASRAD",
		Version:   version,
		CommitID:  commitID,
		BuildDate: buildDate,
		Mode:      "production",
	}
	Print(w, cfg)
}
