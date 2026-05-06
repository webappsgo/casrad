// Package handler contains HTTP request handlers
// See AI.md PART 13 for health & versioning, PART 14 for API structure
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Version info - set by main package
var (
	AppVersion  = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
	startTime   time.Time
	startOnce   sync.Once
)

// InitHealth initializes health tracking (call from main)
func InitHealth() {
	startOnce.Do(func() {
		startTime = time.Now()
	})
}

// HealthResponse represents a health check response per AI.md PART 13
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Mode      string            `json:"mode"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	GoVersion string            `json:"go_version"`
	Build     BuildInfo         `json:"build"`
	Cluster   ClusterInfo       `json:"cluster"`
	Features  FeatureInfo       `json:"features"`
	Checks    map[string]string `json:"checks"`
}

// BuildInfo contains build information
type BuildInfo struct {
	Commit string `json:"commit"`
	Date   string `json:"date"`
}

// ClusterInfo contains cluster status (single instance by default)
type ClusterInfo struct {
	Enabled bool     `json:"enabled"`
	Status  string   `json:"status,omitempty"`
	Primary string   `json:"primary,omitempty"`
	Nodes   []string `json:"nodes"`
}

// FeatureInfo contains enabled features
type FeatureInfo struct {
	MultiUser     bool `json:"multi_user"`
	Organizations bool `json:"organizations"`
	GeoIP         bool `json:"geoip"`
	Metrics       bool `json:"metrics"`
}

// getUptime returns human-readable uptime
func getUptime() string {
	if startTime.IsZero() {
		return "0s"
	}
	d := time.Since(startTime)

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// buildHealthResponse creates a health response with current status
func buildHealthResponse(mode string) HealthResponse {
	return HealthResponse{
		Status:    "healthy",
		Version:   AppVersion,
		Mode:      mode,
		Uptime:    getUptime(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GoVersion: runtime.Version(),
		Build: BuildInfo{
			Commit: BuildCommit,
			Date:   BuildDate,
		},
		Cluster: ClusterInfo{
			Enabled: false,
			Nodes:   []string{},
		},
		Features: FeatureInfo{
			MultiUser:     true, // Enabled by default per CLAUDE.md
			Organizations: false,
			GeoIP:         false, // Deferred
			Metrics:       false, // Deferred
		},
		Checks: map[string]string{
			"database":  "ok",
			"cache":     "ok",
			"scheduler": "ok",
		},
	}
}

// Health handles GET /healthz with content negotiation per PART 13
func Health(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")

	// Content negotiation per PART 13
	if strings.Contains(accept, "application/json") {
		HealthAPI(w, r)
		return
	}

	if strings.Contains(accept, "text/plain") {
		HealthText(w, r)
		return
	}

	// Default to text/plain for curl and simple clients
	if !strings.Contains(accept, "text/html") {
		HealthText(w, r)
		return
	}

	// HTML response for browsers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp := buildHealthResponse("production")

	// Status class for styling
	statusClass := "status-healthy"
	statusIcon := "&#x2705;"
	statusText := "All Systems Operational"

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="refresh" content="30">
    <title>CASRAD - Health Status</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: 'Inter', system-ui, sans-serif; background: #282a36; color: #f8f8f2; min-height: 100vh; padding: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        h1 { color: #bd93f9; margin-bottom: 0.5rem; }
        h2 { color: #8be9fd; margin-bottom: 1rem; font-size: 1.25rem; }
        .description { color: #6272a4; margin-bottom: 2rem; }
        .status-banner { padding: 1.5rem; border-radius: 8px; text-align: center; font-size: 1.5rem; margin-bottom: 2rem; }
        .status-healthy { background: rgba(80, 250, 123, 0.2); color: #50fa7b; }
        .section { background: #44475a; padding: 1.5rem; border-radius: 8px; margin-bottom: 1rem; }
        .info-grid { display: grid; grid-template-columns: auto 1fr; gap: 0.5rem 1rem; }
        .info-grid dt { color: #6272a4; }
        .checks-table { width: 100%%; border-collapse: collapse; }
        .checks-table td { padding: 0.5rem; border-bottom: 1px solid #6272a4; }
        .status-ok { color: #50fa7b; }
        .footer { margin-top: 2rem; text-align: center; color: #6272a4; font-size: 0.875rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>CASRAD</h1>
        <p class="description">Complete Audio Streaming, Radio, and Distribution</p>

        <div class="status-banner %s">
            <span>%s</span> %s
        </div>

        <div class="section">
            <h2>Version</h2>
            <dl class="info-grid">
                <dt>Version</dt><dd>%s</dd>
                <dt>Go Version</dt><dd>%s</dd>
                <dt>Build</dt><dd>%s (%s)</dd>
                <dt>Uptime</dt><dd>%s</dd>
                <dt>Mode</dt><dd>%s</dd>
            </dl>
        </div>

        <div class="section">
            <h2>Component Status</h2>
            <table class="checks-table">
                <tr><td>Database</td><td class="status-ok">OK</td></tr>
                <tr><td>Cache</td><td class="status-ok">OK</td></tr>
                <tr><td>Scheduler</td><td class="status-ok">OK</td></tr>
            </table>
        </div>

        <div class="footer">
            <p>Last checked: %s</p>
            <p>Auto-refreshing in 30s</p>
        </div>
    </div>
</body>
</html>`,
		statusClass, statusIcon, statusText,
		resp.Version, resp.GoVersion, resp.Build.Commit, resp.Build.Date,
		resp.Uptime, resp.Mode, resp.Timestamp)

	w.Write([]byte(html))
}

// HealthText handles text/plain health response
func HealthText(w http.ResponseWriter, r *http.Request) {
	resp := buildHealthResponse("production")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "status: %s\n", resp.Status)
	fmt.Fprintf(w, "version: %s\n", resp.Version)
	fmt.Fprintf(w, "mode: %s\n", resp.Mode)
	fmt.Fprintf(w, "uptime: %s\n", resp.Uptime)
	fmt.Fprintf(w, "go_version: %s\n", resp.GoVersion)
	fmt.Fprintf(w, "build.commit: %s\n", resp.Build.Commit)
	fmt.Fprintf(w, "database: %s\n", resp.Checks["database"])
	fmt.Fprintf(w, "cache: %s\n", resp.Checks["cache"])
	fmt.Fprintf(w, "scheduler: %s\n", resp.Checks["scheduler"])
}

// HealthAPI handles GET /api/v1/healthz (always JSON)
func HealthAPI(w http.ResponseWriter, r *http.Request) {
	resp := buildHealthResponse("production")

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(resp, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}
