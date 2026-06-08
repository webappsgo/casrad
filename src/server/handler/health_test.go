// Package handler — Tests for health check handlers.
// Covers: InitHealth, SetMode, getUptime, buildHealthResponse,
// Health (JSON via Accept header, text/plain, HTML, default),
// HealthText, HealthAPI.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- InitHealth and getUptime ---

func TestInitHealthSetsStartTime(t *testing.T) {
	t.Parallel()
	// InitHealth is idempotent due to sync.Once; just verify it doesn't panic.
	InitHealth()
	InitHealth()
}

func TestGetUptimeBeforeInit(t *testing.T) {
	t.Parallel()
	// If startTime is zero, uptime should be "0s"
	saved := startTime
	startTime = time.Time{}
	defer func() { startTime = saved }()
	if got := getUptime(); got != "0s" {
		t.Errorf("getUptime (zero time) = %q, want 0s", got)
	}
}

func TestGetUptimeDays(t *testing.T) {
	t.Parallel()
	saved := startTime
	startTime = time.Now().Add(-49 * time.Hour)
	defer func() { startTime = saved }()
	got := getUptime()
	if !strings.Contains(got, "d") {
		t.Errorf("getUptime (2 days) = %q, want days format", got)
	}
}

func TestGetUptimeHours(t *testing.T) {
	t.Parallel()
	saved := startTime
	startTime = time.Now().Add(-3 * time.Hour)
	defer func() { startTime = saved }()
	got := getUptime()
	if !strings.Contains(got, "h") {
		t.Errorf("getUptime (3 hours) = %q, want hours format", got)
	}
}

func TestGetUptimeMinutes(t *testing.T) {
	t.Parallel()
	saved := startTime
	startTime = time.Now().Add(-5 * time.Minute)
	defer func() { startTime = saved }()
	got := getUptime()
	if !strings.Contains(got, "m") {
		t.Errorf("getUptime (5 minutes) = %q, want minutes format", got)
	}
}

// --- SetMode ---

func TestSetMode(t *testing.T) {
	t.Parallel()
	saved := AppMode
	defer func() { AppMode = saved }()
	SetMode("development")
	if AppMode != "development" {
		t.Errorf("AppMode = %q, want development", AppMode)
	}
}

// --- buildHealthResponse ---

func TestBuildHealthResponseFields(t *testing.T) {
	t.Parallel()
	saved := AppVersion
	AppVersion = "9.9.9"
	defer func() { AppVersion = saved }()

	resp := buildHealthResponse()

	if resp.Status != "healthy" {
		t.Errorf("Status = %q, want healthy", resp.Status)
	}
	if resp.Version != "9.9.9" {
		t.Errorf("Version = %q, want 9.9.9", resp.Version)
	}
	if resp.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if resp.Checks.Database != "ok" {
		t.Errorf("Checks.Database = %q, want ok", resp.Checks.Database)
	}
	if resp.Checks.Cache != "ok" {
		t.Errorf("Checks.Cache = %q, want ok", resp.Checks.Cache)
	}
	if resp.Cluster.Nodes == nil {
		t.Error("Cluster.Nodes should not be nil")
	}
}

// --- HealthAPI ---

func TestHealthAPIReturnsJSON(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	HealthAPI(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var resp HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON unmarshal error: %v\nbody: %s", err, rr.Body.String())
	}
	if resp.Status != "healthy" {
		t.Errorf("JSON status = %q, want healthy", resp.Status)
	}
}

func TestHealthAPIResponseFields(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/healthz", nil)
	rr := httptest.NewRecorder()
	HealthAPI(rr, req)

	var resp HealthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Project.Name == "" {
		t.Error("project.name should not be empty")
	}
	if resp.Project.Org == "" {
		t.Error("project.org should not be empty")
	}
	if resp.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

// --- HealthText ---

func TestHealthTextReturnsPlainText(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	rr := httptest.NewRecorder()
	HealthText(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "status: healthy") {
		t.Errorf("text health response missing status: %q", body)
	}
	if !strings.Contains(body, "database: ok") {
		t.Errorf("text health response missing database: %q", body)
	}
}

// --- Health (content negotiation) ---

func TestHealthJSONAcceptDispatchesToAPI(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	Health(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHealthTextAcceptDispatchesToText(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	req.Header.Set("Accept", "text/plain")
	rr := httptest.NewRecorder()
	Health(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestHealthHTMLAcceptReturnsHTML(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	Health(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestHealthDefaultNoAcceptHeader(t *testing.T) {
	t.Parallel()
	// Without Accept header, should default to text/plain
	req := httptest.NewRequest(http.MethodGet, "/server/healthz", nil)
	rr := httptest.NewRecorder()
	Health(rr, req)
	// Should not panic and should return 200
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
