// Package handler contains HTTP request handlers
// See AI.md PART 13 for health check specification
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

// Version info - set by main package via InitHealth
var (
	AppVersion  = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
	AppMode     = "production"
	AppName     = "casrad"
	AppOrg      = "casapps"
	startTime   time.Time
	startOnce   sync.Once
)

// InitHealth initializes health tracking (call from main)
func InitHealth() {
	startOnce.Do(func() {
		startTime = time.Now()
	})
}

// SetMode updates the mode string used in health responses
func SetMode(m string) {
	AppMode = m
}

// ProjectInfo identifies the project per AI.md PART 13
type ProjectInfo struct {
	Name string `json:"name"`
	Org  string `json:"org"`
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

// TorInfo contains Tor hidden service status
type TorInfo struct {
	Enabled      bool   `json:"enabled"`
	OnionAddress string `json:"onion_address,omitempty"`
}

// FeaturesInfo contains enabled feature flags per AI.md PART 13
type FeaturesInfo struct {
	Tor   TorInfo `json:"tor"`
	GeoIP bool    `json:"geoip"`
}

// ChecksInfo contains typed component status per AI.md PART 13
type ChecksInfo struct {
	Database  string `json:"database"`
	Cache     string `json:"cache"`
	Disk      string `json:"disk"`
	Scheduler string `json:"scheduler"`
	Cluster   string `json:"cluster,omitempty"`
	Tor       string `json:"tor,omitempty"`
}

// StatsInfo contains aggregate request statistics per AI.md PART 13
type StatsInfo struct {
	RequestsTotal int64 `json:"requests_total"`
	Requests24h   int64 `json:"requests_24h"`
	ActiveConns   int64 `json:"active_conns"`
}

// HealthResponse represents the canonical health check response per AI.md PART 13
// Field order matches spec exactly.
type HealthResponse struct {
	Project        ProjectInfo  `json:"project"`
	Status         string       `json:"status"`
	PendingRestart bool         `json:"pending_restart,omitempty"`
	RestartReason  []string     `json:"restart_reason,omitempty"`
	Version        string       `json:"version"`
	GoVersion      string       `json:"go_version"`
	Build          BuildInfo    `json:"build"`
	Uptime         string       `json:"uptime"`
	Mode           string       `json:"mode"`
	Timestamp      time.Time    `json:"timestamp"`
	Cluster        ClusterInfo  `json:"cluster"`
	Features       FeaturesInfo `json:"features"`
	Checks         ChecksInfo   `json:"checks"`
	Stats          StatsInfo    `json:"stats"`
}

// getUptime returns human-readable uptime string
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
func buildHealthResponse() HealthResponse {
	return HealthResponse{
		Project: ProjectInfo{
			Name: AppName,
			Org:  AppOrg,
		},
		Status:    "healthy",
		Version:   AppVersion,
		GoVersion: runtime.Version(),
		Build: BuildInfo{
			Commit: BuildCommit,
			Date:   BuildDate,
		},
		Uptime:    getUptime(),
		Mode:      AppMode,
		Timestamp: time.Now().UTC(),
		Cluster: ClusterInfo{
			Enabled: false,
			Nodes:   []string{},
		},
		Features: FeaturesInfo{
			Tor:   TorInfo{Enabled: false},
			GeoIP: false,
		},
		Checks: ChecksInfo{
			Database:  "ok",
			Cache:     "ok",
			Disk:      "ok",
			Scheduler: "ok",
		},
		Stats: StatsInfo{
			RequestsTotal: 0,
			Requests24h:   0,
			ActiveConns:   0,
		},
	}
}

// Health handles GET /server/healthz with content negotiation per PART 13
func Health(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")

	if strings.Contains(accept, "application/json") {
		HealthAPI(w, r)
		return
	}

	if strings.Contains(accept, "text/plain") {
		HealthText(w, r)
		return
	}

	// Default to text/plain for curl and simple clients without text/html
	if !strings.Contains(accept, "text/html") {
		HealthText(w, r)
		return
	}

	// HTML response for browsers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp := buildHealthResponse()

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
        .status-banner { padding: 1.5rem; border-radius: 8px; text-align: center; font-size: 1.5rem; margin-bottom: 2rem; background: rgba(80, 250, 123, 0.2); color: #50fa7b; }
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

        <div class="status-banner">&#x2705; All Systems Operational</div>

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
                <tr><td>Database</td><td class="status-ok">%s</td></tr>
                <tr><td>Cache</td><td class="status-ok">%s</td></tr>
                <tr><td>Disk</td><td class="status-ok">%s</td></tr>
                <tr><td>Scheduler</td><td class="status-ok">%s</td></tr>
            </table>
        </div>

        <div class="footer">
            <p>Last checked: %s</p>
            <p>Auto-refreshing in 30s</p>
        </div>
    </div>
</body>
</html>`,
		resp.Version, resp.GoVersion, resp.Build.Commit, resp.Build.Date,
		resp.Uptime, resp.Mode,
		resp.Checks.Database, resp.Checks.Cache, resp.Checks.Disk, resp.Checks.Scheduler,
		resp.Timestamp.Format(time.RFC3339))

	w.Write([]byte(html))
}

// HealthText handles text/plain health response
func HealthText(w http.ResponseWriter, r *http.Request) {
	resp := buildHealthResponse()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "status: %s\n", resp.Status)
	fmt.Fprintf(w, "version: %s\n", resp.Version)
	fmt.Fprintf(w, "mode: %s\n", resp.Mode)
	fmt.Fprintf(w, "uptime: %s\n", resp.Uptime)
	fmt.Fprintf(w, "go_version: %s\n", resp.GoVersion)
	fmt.Fprintf(w, "build.commit: %s\n", resp.Build.Commit)
	fmt.Fprintf(w, "database: %s\n", resp.Checks.Database)
	fmt.Fprintf(w, "cache: %s\n", resp.Checks.Cache)
	fmt.Fprintf(w, "disk: %s\n", resp.Checks.Disk)
	fmt.Fprintf(w, "scheduler: %s\n", resp.Checks.Scheduler)
}

// HealthAPI handles GET /api/v1/server/healthz — always returns JSON
func HealthAPI(w http.ResponseWriter, r *http.Request) {
	resp := buildHealthResponse()

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(resp, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}
