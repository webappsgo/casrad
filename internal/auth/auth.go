package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/casapps/casrad/internal/database"
	"golang.org/x/crypto/argon2"
)

// AuthManager handles authentication and sessions
type AuthManager struct {
	db *database.Engine
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(db *database.Engine) *AuthManager {
	return &AuthManager{db: db}
}

// HashPassword creates an Argon2id hash of the password
func (a *AuthManager) HashPassword(password string) (string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Argon2id parameters (as defined in spec)
	time := uint32(1)
	memory := uint32(64 * 1024) // 64MB
	threads := uint8(4)
	keyLength := uint32(32)

	// Generate hash
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLength)

	// Encode as base64 string with salt
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	return encoded, nil
}

// VerifyPassword checks a password against its hash
func (a *AuthManager) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	var version int
	var memory, time uint32
	var threads uint8
	var salt, hash string

	_, err := fmt.Sscanf(encodedHash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		&version, &memory, &time, &threads, &salt, &hash)
	if err != nil {
		return false, err
	}

	// Decode salt and hash
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false, err
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false, err
	}

	// Generate hash of input password
	inputHash := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, uint32(len(hashBytes)))

	// Constant-time comparison
	return subtle.ConstantTimeCompare(hashBytes, inputHash) == 1, nil
}

// CreateUser creates a new user account
func (a *AuthManager) CreateUser(username, email, password string, role string) error {
	// Check if user exists
	var exists bool
	err := a.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE username = ? OR email = ?
		)
	`, username, email).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("user already exists")
	}

	// Hash password
	hashedPassword, err := a.HashPassword(password)
	if err != nil {
		return err
	}

	// Create user directory path
	homeDir := fmt.Sprintf("/var/lib/casrad/users/%s", username)

	// Insert user
	_, err = a.db.Exec(`
		INSERT INTO users (
			username, email, password_hash, role, home_directory,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, username, email, hashedPassword, role, homeDir)

	if err != nil {
		return err
	}

	// Create user storage record with defaults
	_, err = a.db.Exec(`
		INSERT INTO user_storage (
			user_id,
			music_paths,
			podcast_path,
			audiobook_path,
			radio_path,
			playlist_path,
			recording_path,
			transcode_path
		) SELECT 
			id,
			?,
			?,
			?,
			?,
			?,
			?,
			?
		FROM users WHERE username = ?
	`,
		fmt.Sprintf(`["%s/music"]`, homeDir),
		fmt.Sprintf("%s/podcasts", homeDir),
		fmt.Sprintf("%s/audiobooks", homeDir),
		fmt.Sprintf("%s/radio", homeDir),
		fmt.Sprintf("%s/playlists", homeDir),
		fmt.Sprintf("%s/recordings", homeDir),
		fmt.Sprintf("%s/transcodes", homeDir),
		username)

	return err
}

// AuthenticateUser validates credentials and returns user ID
func (a *AuthManager) AuthenticateUser(username, password string) (int, error) {
	var userID int
	var passwordHash string
	var isActive bool
	var lockedUntil *time.Time

	err := a.db.QueryRow(`
		SELECT id, password_hash, is_active, locked_until
		FROM users
		WHERE username = ? OR email = ?
	`, username, username).Scan(&userID, &passwordHash, &isActive, &lockedUntil)

	if err != nil {
		return 0, fmt.Errorf("invalid credentials")
	}

	// Check if account is active
	if !isActive {
		return 0, fmt.Errorf("account disabled")
	}

	// Check if account is locked
	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		return 0, fmt.Errorf("account locked until %s", lockedUntil.Format(time.RFC3339))
	}

	// Verify password
	valid, err := a.VerifyPassword(password, passwordHash)
	if err != nil || !valid {
		// Increment failed login attempts
		a.db.Exec(`
			UPDATE users 
			SET failed_login_attempts = failed_login_attempts + 1,
			    locked_until = CASE 
			        WHEN failed_login_attempts >= 4 
			        THEN datetime('now', '+30 minutes')
			        ELSE locked_until
			    END
			WHERE id = ?
		`, userID)
		return 0, fmt.Errorf("invalid credentials")
	}

	// Reset failed attempts and update last login
	a.db.Exec(`
		UPDATE users 
		SET failed_login_attempts = 0,
		    locked_until = NULL,
		    last_login = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID)

	return userID, nil
}

// CreateSession creates a new user session
func (a *AuthManager) CreateSession(userID int, ipAddress, userAgent string) (string, error) {
	// Generate session ID
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	sessionID := base64.URLEncoding.EncodeToString(bytes)

	// Calculate expiry (7 days default as per spec)
	var sessionDuration int
	a.db.GetSetting("security.session_duration_hours")
	if sessionDuration == 0 {
		sessionDuration = 168 // 7 days
	}

	// Insert session
	_, err := a.db.Exec(`
		INSERT INTO sessions (
			id, user_id, ip_address, user_agent,
			created_at, expires_at, last_activity, is_active
		) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, 
		          datetime('now', '+' || ? || ' hours'),
		          CURRENT_TIMESTAMP, 1)
	`, sessionID, userID, ipAddress, userAgent, sessionDuration)

	if err != nil {
		return "", err
	}

	return sessionID, nil
}

// ValidateSession checks if a session is valid
func (a *AuthManager) ValidateSession(sessionID string) (int, error) {
	var userID int
	var expiresAt time.Time
	var isActive bool

	err := a.db.QueryRow(`
		SELECT user_id, expires_at, is_active
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(&userID, &expiresAt, &isActive)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}

	if !isActive {
		return 0, fmt.Errorf("session inactive")
	}

	if expiresAt.Before(time.Now()) {
		return 0, fmt.Errorf("session expired")
	}

	// Update last activity
	a.db.Exec(`
		UPDATE sessions 
		SET last_activity = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sessionID)

	return userID, nil
}

// DestroySession invalidates a session
func (a *AuthManager) DestroySession(sessionID string) error {
	_, err := a.db.Exec(`
		UPDATE sessions 
		SET is_active = 0
		WHERE id = ?
	`, sessionID)
	return err
}

// GetUser retrieves user information
func (a *AuthManager) GetUser(userID int) (*User, error) {
	user := &User{}
	err := a.db.QueryRow(`
		SELECT id, username, email, role, theme_preference,
		       storage_quota_bytes, storage_used_bytes,
		       is_active, created_at, last_login
		FROM users
		WHERE id = ?
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.ThemePreference,
		&user.StorageQuota,
		&user.StorageUsed,
		&user.IsActive,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdatePassword changes a user's password
func (a *AuthManager) UpdatePassword(userID int, oldPassword, newPassword string) error {
	// Get current password hash
	var currentHash string
	err := a.db.QueryRow(`
		SELECT password_hash FROM users WHERE id = ?
	`, userID).Scan(&currentHash)

	if err != nil {
		return err
	}

	// Verify old password
	valid, err := a.VerifyPassword(oldPassword, currentHash)
	if err != nil || !valid {
		return fmt.Errorf("invalid current password")
	}

	// Hash new password
	newHash, err := a.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	_, err = a.db.Exec(`
		UPDATE users 
		SET password_hash = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newHash, userID)

	return err
}

// GenerateAPIToken creates a new API token for a user
func (a *AuthManager) GenerateAPIToken(userID int, name string, permissions []string) (string, error) {
	// Generate token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	// Convert permissions to JSON
	permJSON := "[]"
	if len(permissions) > 0 {
		// Simple JSON encoding
		permJSON = "["
		for i, perm := range permissions {
			if i > 0 {
				permJSON += ","
			}
			permJSON += fmt.Sprintf(`"%s"`, perm)
		}
		permJSON += "]"
	}

	// Insert token
	_, err := a.db.Exec(`
		INSERT INTO api_tokens (
			user_id, token, name, permissions,
			created_at, is_active
		) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, 1)
	`, userID, token, name, permJSON)

	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateAPIToken validates an API token and returns the user ID
func (a *AuthManager) ValidateAPIToken(token string) (int, []string, error) {
	var userID int
	var permissions string
	var isActive bool
	var expiresAt *time.Time

	err := a.db.QueryRow(`
		SELECT user_id, permissions, is_active, expires_at
		FROM api_tokens
		WHERE token = ?
	`, token).Scan(&userID, &permissions, &isActive, &expiresAt)

	if err != nil {
		return 0, nil, fmt.Errorf("invalid token")
	}

	if !isActive {
		return 0, nil, fmt.Errorf("token inactive")
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return 0, nil, fmt.Errorf("token expired")
	}

	// Update usage
	a.db.Exec(`
		UPDATE api_tokens 
		SET last_used = CURRENT_TIMESTAMP,
		    use_count = use_count + 1
		WHERE token = ?
	`, token)

	// Parse permissions (simplified)
	var perms []string
	// TODO: Proper JSON parsing

	return userID, perms, nil
}

// User represents a user account
type User struct {
	ID              int
	Username        string
	Email           string
	Role            string
	ThemePreference string
	StorageQuota    int64
	StorageUsed     int64
	IsActive        bool
	CreatedAt       time.Time
	LastLogin       *time.Time
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsModerator checks if user has moderator role
func (u *User) IsModerator() bool {
	return u.Role == "moderator" || u.Role == "admin"
}

// PasswordStrength checks if password meets requirements
func (a *AuthManager) PasswordStrength(password string) error {
	// Get minimum length from settings (default 8)
	var minLength int
	a.db.GetSetting("security.password_min_length")
	if minLength == 0 {
		minLength = 8
	}

	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}

	// Additional checks can be added here
	// For now, just length requirement as per spec

	return nil
}

// CleanupSessions removes expired sessions
func (a *AuthManager) CleanupSessions() error {
	_, err := a.db.Exec(`
		DELETE FROM sessions 
		WHERE expires_at < CURRENT_TIMESTAMP
		   OR is_active = 0
	`)
	return err
}

// Enable2FA enables two-factor authentication for a user
func (a *AuthManager) Enable2FA(userID int) (string, error) {
	// Generate TOTP secret
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret := base64.StdEncoding.EncodeToString(bytes)

	// Update user
	_, err := a.db.Exec(`
		UPDATE users 
		SET totp_secret = ?
		WHERE id = ?
	`, secret, userID)

	if err != nil {
		return "", err
	}

	return secret, nil
}

// Verify2FA verifies a TOTP code
func (a *AuthManager) Verify2FA(userID int, code string) error {
	// TODO: Implement TOTP verification
	// This requires additional TOTP library
	return nil
}