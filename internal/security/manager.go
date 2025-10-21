package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
	"golang.org/x/crypto/argon2"
)

type Manager struct {
	db              *database.Engine
	rateLimiter     *RateLimiter
	bruteForce      *BruteForceProtection
	sessionManager  *SessionManager
	geoIPManager    *GeoIPManager
	mu              sync.RWMutex
}

func NewManager(db *database.Engine) *Manager {
	// Get data path for GeoIP
	dataPath := "/etc/casrad/security"
	if setting, err := db.GetSetting("storage.user_base_path"); err == nil {
		dataPath = setting
	}

	return &Manager{
		db:             db,
		rateLimiter:    NewRateLimiter(),
		bruteForce:     NewBruteForceProtection(db),
		sessionManager: NewSessionManager(db),
		geoIPManager:   NewGeoIPManager(dataPath, db),
	}
}

func (m *Manager) RunBackgroundTasks(ctx context.Context) {
	// Run cleanup tasks periodically
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredSessions()
			m.cleanupRateLimiter()
			m.bruteForce.Cleanup()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) cleanupExpiredSessions() {
	m.db.Exec(`
		UPDATE sessions
		SET is_active = 0
		WHERE expires_at < ? AND is_active = 1
	`, time.Now())
}

func (m *Manager) cleanupRateLimiter() {
	m.rateLimiter.Cleanup()
}

// Password handling
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Format: salt:hash
	return fmt.Sprintf("%x:%x", salt, hash), nil
}

func VerifyPassword(password, hash string) bool {
	parts := string(hash)
	if len(parts) != 2 {
		return false
	}

	salt, _ := hex.DecodeString(parts[0:32])
	storedHash, _ := hex.DecodeString(parts[33:])

	computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Constant time comparison
	if len(computedHash) != len(storedHash) {
		return false
	}

	result := byte(0)
	for i := range computedHash {
		result |= computedHash[i] ^ storedHash[i]
	}

	return result == 0
}

// Session generation
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// API token generation
func GenerateAPIToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RateLimiter implementation
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*Visitor
}

type Visitor struct {
	limiter  *time.Ticker
	lastSeen time.Time
	requests int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*Visitor),
	}
}

func (r *RateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, exists := r.visitors[ip]
	if !exists {
		// Default: 60 requests per minute
		v = &Visitor{
			limiter:  time.NewTicker(time.Second),
			lastSeen: time.Now(),
			requests: 0,
		}
		r.visitors[ip] = v
	}

	v.lastSeen = time.Now()
	v.requests++

	// Simple rate limiting: 60 requests per minute
	if v.requests > 60 {
		select {
		case <-v.limiter.C:
			v.requests = 1
			return true
		default:
			return false
		}
	}

	return true
}

func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ip, v := range r.visitors {
		if time.Since(v.lastSeen) > 5*time.Minute {
			v.limiter.Stop()
			delete(r.visitors, ip)
		}
	}
}

// BruteForceProtection implementation
type BruteForceProtection struct {
	db       *database.Engine
	attempts map[string]int
	mu       sync.RWMutex
}

func NewBruteForceProtection(db *database.Engine) *BruteForceProtection {
	return &BruteForceProtection{
		db:       db,
		attempts: make(map[string]int),
	}
}

func (b *BruteForceProtection) RecordFailedAttempt(username string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attempts[username]++

	// Update database
	b.db.Exec(`
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1
		WHERE username = ?
	`, username)

	// Lock account after 5 attempts for 30 minutes
	if b.attempts[username] >= 5 {
		lockUntil := time.Now().Add(30 * time.Minute)
		b.db.Exec(`
			UPDATE users
			SET locked_until = ?
			WHERE username = ?
		`, lockUntil, username)
	}
}

func (b *BruteForceProtection) ResetAttempts(username string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.attempts, username)

	b.db.Exec(`
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until = NULL
		WHERE username = ?
	`, username)
}

func (b *BruteForceProtection) IsLocked(username string) bool {
	var lockedUntil *time.Time
	err := b.db.QueryRow(`
		SELECT locked_until
		FROM users
		WHERE username = ?
	`, username).Scan(&lockedUntil)

	if err != nil || lockedUntil == nil {
		return false
	}

	return lockedUntil.After(time.Now())
}

func (b *BruteForceProtection) Cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset in-memory attempts periodically
	for username := range b.attempts {
		if b.attempts[username] == 0 {
			delete(b.attempts, username)
		}
	}
}

// SessionManager implementation
type SessionManager struct {
	db *database.Engine
}

func NewSessionManager(db *database.Engine) *SessionManager {
	return &SessionManager{db: db}
}

func (s *SessionManager) CreateSession(userID int, ip, userAgent string) (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days default

	_, err = s.db.Exec(`
		INSERT INTO sessions (id, user_id, ip_address, user_agent, expires_at, last_activity)
		VALUES (?, ?, ?, ?, ?, ?)
	`, token, userID, ip, userAgent, expiresAt, time.Now())

	return token, err
}

func (s *SessionManager) ValidateSession(token string) (int, error) {
	var userID int
	var expiresAt time.Time
	var isActive bool

	err := s.db.QueryRow(`
		SELECT user_id, expires_at, is_active
		FROM sessions
		WHERE id = ?
	`, token).Scan(&userID, &expiresAt, &isActive)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}

	if !isActive || expiresAt.Before(time.Now()) {
		return 0, fmt.Errorf("session expired")
	}

	// Update last activity
	s.db.Exec(`
		UPDATE sessions
		SET last_activity = ?
		WHERE id = ?
	`, time.Now(), token)

	return userID, nil
}

func (s *SessionManager) DestroySession(token string) error {
	_, err := s.db.Exec(`
		UPDATE sessions
		SET is_active = 0
		WHERE id = ?
	`, token)
	return err
}

// IP utilities
func GetRealIP(remoteAddr string, headers map[string]string) string {
	// Check X-Forwarded-For header
	if xff := headers["X-Forwarded-For"]; xff != "" {
		ips := splitIPs(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	if xri := headers["X-Real-IP"]; xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, _ := net.SplitHostPort(remoteAddr)
	return ip
}

func splitIPs(ips string) []string {
	var result []string
	for _, ip := range []string{ips} {
		if net.ParseIP(ip) != nil {
			result = append(result, ip)
		}
	}
	return result
}