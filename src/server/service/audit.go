// Package service - Audit logging per AI.md PART 11
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/src/paths"
)

// Audit event categories per AI.md PART 11
const (
	CategoryAuth          = "authentication"
	CategoryConfiguration = "configuration"
	CategorySecurity      = "security"
	CategoryTokens        = "tokens"
	CategoryUsers         = "users"
	CategoryBackup        = "backup"
	CategoryServer        = "server"
	CategoryCluster       = "cluster"
)

// Audit severity levels per AI.md PART 11
const (
	SeverityInfo     = "info"
	SeverityWarn     = "warn"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// Audit result values
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// AuditEntry represents a single audit log entry per AI.md PART 11
type AuditEntry struct {
	ID       string            `json:"id"`
	Time     string            `json:"time"`
	Event    string            `json:"event"`
	Category string            `json:"category"`
	Severity string            `json:"severity"`
	Actor    *AuditActor       `json:"actor"`
	Target   *AuditTarget      `json:"target,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
	Result   string            `json:"result"`
	NodeID   string            `json:"node_id,omitempty"`
	Reason   string            `json:"reason,omitempty"`
}

// AuditActor represents who performed the action
type AuditActor struct {
	// admin, user, system
	Type string `json:"type"`
	// username or user_id
	ID string `json:"id"`
	// IP address
	IP string `json:"ip,omitempty"`
	// User agent string
	UserAgent string `json:"user_agent,omitempty"`
}

// AuditTarget represents what was acted upon
type AuditTarget struct {
	// session, user, config, etc.
	Type string `json:"type"`
	// target identifier
	ID string `json:"id"`
}

// AuditLogger handles audit log writing per AI.md PART 11
type AuditLogger struct {
	file     *os.File
	mu       sync.Mutex
	enabled  bool
	nodeid   string
	sequence uint64
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() (*AuditLogger, error) {
	logPath := filepath.Join(paths.Get().Log, "audit.log")

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open audit log file in append-only mode (per PART 11: append-only, no modification)
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &AuditLogger{
		file:    file,
		enabled: true,
		// Single node by default
		nodeid: "node-1",
	}, nil
}

// Log writes an audit entry
func (l *AuditLogger) Log(entry *AuditEntry) error {
	if !l.enabled || l.file == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Generate unique ID (ULID-like format)
	l.sequence++
	entry.ID = fmt.Sprintf("audit_%d_%06d", time.Now().UnixNano(), l.sequence)

	// Set timestamp in ISO 8601 format with milliseconds, UTC
	entry.Time = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// Set node ID
	if entry.NodeID == "" {
		entry.NodeID = l.nodeid
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write JSON line
	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// Close closes the audit log file
func (l *AuditLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Helper methods for common audit events per AI.md PART 11

// LogAdminLogin logs admin login event
func (l *AuditLogger) LogAdminLogin(username, ip, userAgent string, mfaUsed bool) {
	details := map[string]string{}
	if mfaUsed {
		details["mfa_used"] = "true"
	}
	l.Log(&AuditEntry{
		Event:    "admin.login",
		Category: CategoryAuth,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type:      "admin",
			ID:        username,
			IP:        ip,
			UserAgent: userAgent,
		},
		Details: details,
		Result:  ResultSuccess,
	})
}

// LogAdminLoginFailed logs failed admin login attempt
func (l *AuditLogger) LogAdminLoginFailed(attemptedUsername, ip, userAgent, reason string) {
	l.Log(&AuditEntry{
		Event:    "admin.login_failed",
		Category: CategoryAuth,
		Severity: SeverityWarn,
		Actor: &AuditActor{
			Type:      "admin",
			ID:        attemptedUsername,
			IP:        ip,
			UserAgent: userAgent,
		},
		Details: map[string]string{"reason": reason},
		Result:  ResultFailure,
	})
}

// LogAdminLogout logs admin logout event
func (l *AuditLogger) LogAdminLogout(username string, sessionDuration string) {
	l.Log(&AuditEntry{
		Event:    "admin.logout",
		Category: CategoryAuth,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "admin",
			ID:   username,
		},
		Details: map[string]string{"session_duration": sessionDuration},
		Result:  ResultSuccess,
	})
}

// LogUserLogin logs user login event
func (l *AuditLogger) LogUserLogin(userID, ip, userAgent, authMethod string) {
	l.Log(&AuditEntry{
		Event:    "user.login",
		Category: CategoryAuth,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type:      "user",
			ID:        userID,
			IP:        ip,
			UserAgent: userAgent,
		},
		Details: map[string]string{"auth_method": authMethod},
		Result:  ResultSuccess,
	})
}

// LogUserLoginFailed logs failed user login attempt
func (l *AuditLogger) LogUserLoginFailed(ip, userAgent, reason string) {
	// Per PART 11: Do NOT log username/email for failed logins
	l.Log(&AuditEntry{
		Event:    "user.login_failed",
		Category: CategoryAuth,
		Severity: SeverityWarn,
		Actor: &AuditActor{
			Type:      "user",
			IP:        ip,
			UserAgent: userAgent,
		},
		Details: map[string]string{"reason": reason},
		Result:  ResultFailure,
	})
}

// LogServerStarted logs server start event
func (l *AuditLogger) LogServerStarted(version, mode string) {
	l.Log(&AuditEntry{
		Event:    "server.started",
		Category: CategoryServer,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "system",
			ID:   "server",
		},
		Details: map[string]string{
			"version": version,
			"mode":    mode,
		},
		Result: ResultSuccess,
	})
}

// LogServerStopped logs server stop event
func (l *AuditLogger) LogServerStopped(reason, uptime string) {
	l.Log(&AuditEntry{
		Event:    "server.stopped",
		Category: CategoryServer,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "system",
			ID:   "server",
		},
		Details: map[string]string{
			"reason": reason,
			"uptime": uptime,
		},
		Result: ResultSuccess,
	})
}

// LogConfigUpdated logs configuration change
func (l *AuditLogger) LogConfigUpdated(changedKeys []string, changedBy string) {
	l.Log(&AuditEntry{
		Event:    "config.updated",
		Category: CategoryConfiguration,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "admin",
			ID:   changedBy,
		},
		Details: map[string]string{"changed_keys": strings.Join(changedKeys, ",")},
		Result:  ResultSuccess,
	})
}

// LogTokenCreated logs API token creation
func (l *AuditLogger) LogTokenCreated(tokenPrefix, createdBy, scope string) {
	l.Log(&AuditEntry{
		Event:    "token.created",
		Category: CategoryTokens,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "admin",
			ID:   createdBy,
		},
		Target: &AuditTarget{
			Type: "token",
			// Only show prefix per PART 11
			ID: tokenPrefix + "...",
		},
		Details: map[string]string{"scope": scope},
		Result:  ResultSuccess,
	})
}

// LogTokenRevoked logs API token revocation
func (l *AuditLogger) LogTokenRevoked(tokenPrefix, revokedBy string) {
	l.Log(&AuditEntry{
		Event:    "token.revoked",
		Category: CategoryTokens,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "admin",
			ID:   revokedBy,
		},
		Target: &AuditTarget{
			Type: "token",
			ID:   tokenPrefix + "...",
		},
		Result: ResultSuccess,
	})
}

// LogRateLimitExceeded logs rate limit hit
func (l *AuditLogger) LogRateLimitExceeded(ip, endpoint string, limit int) {
	l.Log(&AuditEntry{
		Event:    "security.rate_limit_exceeded",
		Category: CategorySecurity,
		Severity: SeverityWarn,
		Actor: &AuditActor{
			Type: "unknown",
			IP:   ip,
		},
		Details: map[string]string{
			"endpoint": endpoint,
			"limit":    fmt.Sprintf("%d", limit),
		},
		Result: ResultFailure,
	})
}

// LogBruteForceDetected logs brute force detection
func (l *AuditLogger) LogBruteForceDetected(ip string, attemptCount int) {
	l.Log(&AuditEntry{
		Event:    "security.brute_force_detected",
		Category: CategorySecurity,
		Severity: SeverityCritical,
		Actor: &AuditActor{
			Type: "unknown",
			IP:   ip,
		},
		Details: map[string]string{"attempt_count": fmt.Sprintf("%d", attemptCount)},
		Result:  ResultFailure,
	})
}

// LogBackupCreated logs backup creation
func (l *AuditLogger) LogBackupCreated(filename, createdBy string, sizeBytes int64) {
	l.Log(&AuditEntry{
		Event:    "backup.created",
		Category: CategoryBackup,
		Severity: SeverityInfo,
		Actor: &AuditActor{
			Type: "admin",
			ID:   createdBy,
		},
		Target: &AuditTarget{
			Type: "backup",
			ID:   filename,
		},
		Details: map[string]string{"size_bytes": fmt.Sprintf("%d", sizeBytes)},
		Result:  ResultSuccess,
	})
}
