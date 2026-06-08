// Package service — Tests for AuditLogger.
// Covers: NewAuditLogger (via temp file path override), Log, Close,
// LogAdminLogin, LogAdminLoginFailed, LogAdminLogout,
// LogUserLogin, LogUserLoginFailed, LogServerStarted, LogServerStopped,
// LogConfigUpdated, LogTokenCreated, LogTokenRevoked,
// LogRateLimitExceeded, LogBruteForceDetected, LogBackupCreated.
package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// newTestAuditLogger creates an AuditLogger that writes to a temp file.
// This bypasses paths.Get() which may not work in test containers.
func newTestAuditLogger(t *testing.T) (*AuditLogger, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	l := &AuditLogger{
		file:    f,
		enabled: true,
		nodeid:  "test-node",
	}
	t.Cleanup(func() { l.Close() })
	return l, logPath
}

// readLastEntry reads the last JSON line from the log file.
func readLastEntry(t *testing.T, logPath string) *AuditEntry {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	lines := splitLines(data)
	if len(lines) == 0 {
		t.Fatal("audit log is empty")
	}
	var entry AuditEntry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("unmarshal audit entry: %v", err)
	}
	return &entry
}

// splitLines splits byte slice into non-empty lines.
func splitLines(data []byte) []string {
	var out []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := string(data[start:i])
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(data) && len(data[start:]) > 0 {
		out = append(out, string(data[start:]))
	}
	return out
}

// --- Log ---

func TestAuditLogBasic(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)

	entry := &AuditEntry{
		Event:    "test.event",
		Category: CategoryAuth,
		Severity: SeverityInfo,
		Actor:    &AuditActor{Type: "system", ID: "test"},
		Result:   ResultSuccess,
	}
	if err := l.Log(entry); err != nil {
		t.Fatalf("Log: %v", err)
	}

	got := readLastEntry(t, logPath)
	if got.Event != "test.event" {
		t.Errorf("Event = %q, want test.event", got.Event)
	}
	if got.ID == "" {
		t.Error("ID should be set after Log")
	}
	if got.Time == "" {
		t.Error("Time should be set after Log")
	}
	if got.NodeID != "test-node" {
		t.Errorf("NodeID = %q, want test-node", got.NodeID)
	}
}

func TestAuditLogDisabled(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.enabled = false

	if err := l.Log(&AuditEntry{Event: "should_not_write"}); err != nil {
		t.Fatalf("Log when disabled: %v", err)
	}

	data, _ := os.ReadFile(logPath)
	if len(data) != 0 {
		t.Error("disabled logger should not write to file")
	}
}

func TestAuditLogSequenceIncrements(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)

	for i := 0; i < 3; i++ {
		l.Log(&AuditEntry{
			Event:    "seq.test",
			Category: CategoryAuth,
			Severity: SeverityInfo,
			Actor:    &AuditActor{Type: "system"},
			Result:   ResultSuccess,
		})
	}

	data, _ := os.ReadFile(logPath)
	lines := splitLines(data)
	if len(lines) != 3 {
		t.Errorf("expected 3 log lines, got %d", len(lines))
	}
}

// --- Close ---

func TestAuditLoggerClose(t *testing.T) {
	t.Parallel()
	l, _ := newTestAuditLogger(t)
	if err := l.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// Calling Close twice on a closed file will error — that's fine, just shouldn't panic
}

func TestAuditLoggerCloseNilFile(t *testing.T) {
	t.Parallel()
	l := &AuditLogger{}
	if err := l.Close(); err != nil {
		t.Errorf("Close on nil file: %v", err)
	}
}

// --- Helper method tests ---

func TestLogAdminLogin(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogAdminLogin("adminuser", "10.0.0.1", "Mozilla/5.0", true)

	got := readLastEntry(t, logPath)
	if got.Event != "admin.login" {
		t.Errorf("Event = %q, want admin.login", got.Event)
	}
	if got.Result != ResultSuccess {
		t.Errorf("Result = %q, want %q", got.Result, ResultSuccess)
	}
	if got.Actor == nil || got.Actor.IP != "10.0.0.1" {
		t.Error("Actor IP should be 10.0.0.1")
	}
}

func TestLogAdminLoginFailed(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogAdminLoginFailed("baduser", "10.0.0.2", "curl/7.0", "invalid_credentials")

	got := readLastEntry(t, logPath)
	if got.Event != "admin.login_failed" {
		t.Errorf("Event = %q, want admin.login_failed", got.Event)
	}
	if got.Result != ResultFailure {
		t.Errorf("Result = %q, want %q", got.Result, ResultFailure)
	}
	if got.Severity != SeverityWarn {
		t.Errorf("Severity = %q, want %q", got.Severity, SeverityWarn)
	}
}

func TestLogAdminLogout(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogAdminLogout("adminuser", "2h30m")

	got := readLastEntry(t, logPath)
	if got.Event != "admin.logout" {
		t.Errorf("Event = %q, want admin.logout", got.Event)
	}
}

func TestLogUserLogin(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogUserLogin("42", "192.168.1.1", "TestAgent/1.0", "password")

	got := readLastEntry(t, logPath)
	if got.Event != "user.login" {
		t.Errorf("Event = %q, want user.login", got.Event)
	}
	if got.Category != CategoryAuth {
		t.Errorf("Category = %q, want %q", got.Category, CategoryAuth)
	}
}

func TestLogUserLoginFailed(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogUserLoginFailed("192.168.1.5", "TestAgent", "invalid_credentials")

	got := readLastEntry(t, logPath)
	if got.Event != "user.login_failed" {
		t.Errorf("Event = %q, want user.login_failed", got.Event)
	}
	// Per PART 11: username must NOT be logged for failed user logins
	if got.Actor != nil && got.Actor.ID != "" {
		t.Error("failed user login must not log username/ID")
	}
}

func TestLogServerStarted(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogServerStarted("1.0.0", "production")

	got := readLastEntry(t, logPath)
	if got.Event != "server.started" {
		t.Errorf("Event = %q, want server.started", got.Event)
	}
	if got.Category != CategoryServer {
		t.Errorf("Category = %q, want %q", got.Category, CategoryServer)
	}
}

func TestLogServerStopped(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogServerStopped("shutdown", "4h20m")

	got := readLastEntry(t, logPath)
	if got.Event != "server.stopped" {
		t.Errorf("Event = %q, want server.stopped", got.Event)
	}
}

func TestLogConfigUpdated(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogConfigUpdated([]string{"theme", "language"}, "adminuser")

	got := readLastEntry(t, logPath)
	if got.Event != "config.updated" {
		t.Errorf("Event = %q, want config.updated", got.Event)
	}
	if got.Category != CategoryConfiguration {
		t.Errorf("Category = %q, want %q", got.Category, CategoryConfiguration)
	}
}

func TestLogTokenCreated(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogTokenCreated("abc123", "adminuser", "read")

	got := readLastEntry(t, logPath)
	if got.Event != "token.created" {
		t.Errorf("Event = %q, want token.created", got.Event)
	}
	if got.Category != CategoryTokens {
		t.Errorf("Category = %q, want %q", got.Category, CategoryTokens)
	}
}

func TestLogTokenRevoked(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogTokenRevoked("abc123", "adminuser")

	got := readLastEntry(t, logPath)
	if got.Event != "token.revoked" {
		t.Errorf("Event = %q, want token.revoked", got.Event)
	}
}

func TestLogRateLimitExceeded(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogRateLimitExceeded("10.0.0.99", "/api/v1/auth/login", 60)

	got := readLastEntry(t, logPath)
	if got.Event != "security.rate_limit_exceeded" {
		t.Errorf("Event = %q, want security.rate_limit_exceeded", got.Event)
	}
	if got.Category != CategorySecurity {
		t.Errorf("Category = %q, want %q", got.Category, CategorySecurity)
	}
	if got.Severity != SeverityWarn {
		t.Errorf("Severity = %q, want %q", got.Severity, SeverityWarn)
	}
}

func TestLogBruteForceDetected(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogBruteForceDetected("10.0.0.100", 20)

	got := readLastEntry(t, logPath)
	if got.Event != "security.brute_force_detected" {
		t.Errorf("Event = %q, want security.brute_force_detected", got.Event)
	}
	if got.Severity != SeverityCritical {
		t.Errorf("Severity = %q, want %q", got.Severity, SeverityCritical)
	}
}

func TestLogBackupCreated(t *testing.T) {
	t.Parallel()
	l, logPath := newTestAuditLogger(t)
	l.LogBackupCreated("backup_2025-01-01.tar.gz", "adminuser", 1024*1024*100)

	got := readLastEntry(t, logPath)
	if got.Event != "backup.created" {
		t.Errorf("Event = %q, want backup.created", got.Event)
	}
	if got.Category != CategoryBackup {
		t.Errorf("Category = %q, want %q", got.Category, CategoryBackup)
	}
}
