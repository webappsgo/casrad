// Package service contains business logic
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
	"golang.org/x/crypto/argon2"
)

// Argon2id parameters per AI.md (OWASP 2023)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// Session duration per CLAUDE.md - 7 days default
const defaultSessionDuration = 7 * 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrInvalidSession     = errors.New("invalid session")
	ErrSessionExpired     = errors.New("session expired")
)

// identifierType represents the type of login identifier
type identifierType int

const (
	identifierUsername identifierType = iota
	identifierEmail
	identifierUserID
)

// emailRegex validates email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// userIDRegex matches numeric IDs
var userIDRegex = regexp.MustCompile(`^[0-9]+$`)

// Store interface for auth operations
type AuthStore interface {
	GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error)
	GetAdminByEmail(ctx context.Context, email string) (*model.Admin, error)
	GetAdminByID(ctx context.Context, id int64) (*model.Admin, error)
	UpdateAdmin(ctx context.Context, admin *model.Admin) error

	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error

	GetSession(ctx context.Context, id string) (*model.Session, error)
	CreateSession(ctx context.Context, session *model.Session) error
	UpdateSession(ctx context.Context, session *model.Session) error
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID int64) error
}

// AuthService handles authentication logic
type AuthService struct {
	store AuthStore
}

// NewAuthService creates a new auth service
func NewAuthService(s AuthStore) *AuthService {
	return &AuthService{store: s}
}

// NewAuthServiceWithStore creates a new auth service with SQLiteStore
func NewAuthServiceWithStore(s *store.SQLiteStore) *AuthService {
	return &AuthService{store: s}
}

// detectIdentifierType detects whether identifier is username, email, or user_id
func detectIdentifierType(identifier string) identifierType {
	if emailRegex.MatchString(identifier) {
		return identifierEmail
	}
	if userIDRegex.MatchString(identifier) {
		return identifierUserID
	}
	return identifierUsername
}

// Authenticate validates user credentials
// identifier can be: username, user_id, or email
// Returns userID, adminID (one will be 0), and error
func (s *AuthService) Authenticate(ctx context.Context, identifier, password, ip string) (userID, adminID int64, err error) {
	// Validate login input - trim identifier, check password whitespace
	result := ValidateLogin(identifier, password)
	if result.HasErrors() {
		// Return first error
		for _, errMsg := range result.Errors {
			return 0, 0, errors.New(errMsg)
		}
	}

	// Trim identifier for lookup
	identifier = TrimInput(identifier)

	idType := detectIdentifierType(identifier)

	// Check admins table first
	var admin *model.Admin
	switch idType {
	case identifierEmail:
		admin, err = s.store.GetAdminByEmail(ctx, identifier)
	case identifierUsername:
		admin, err = s.store.GetAdminByUsername(ctx, identifier)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("database error: %w", err)
	}

	if admin != nil {
		// Check if account is locked
		if !admin.LockedUntil.IsZero() && time.Now().Before(admin.LockedUntil) {
			return 0, 0, ErrAccountLocked
		}
		// Check if account is disabled
		if !admin.IsActive {
			return 0, 0, ErrAccountDisabled
		}
		// Verify password
		if s.VerifyPassword(password, admin.PasswordHash) {
			// Reset failed attempts, update last login
			admin.FailedLoginAttempts = 0
			admin.LastLogin = time.Now()
			admin.LastIP = ip
			admin.LockedUntil = time.Time{}
			s.store.UpdateAdmin(ctx, admin)
			return 0, admin.ID, nil
		}
		// Wrong password - increment failed attempts
		admin.FailedLoginAttempts++
		if admin.FailedLoginAttempts >= 5 {
			admin.LockedUntil = time.Now().Add(30 * time.Minute)
		}
		s.store.UpdateAdmin(ctx, admin)
		return 0, 0, ErrInvalidCredentials
	}

	// Check users table
	var user *model.User
	switch idType {
	case identifierEmail:
		user, err = s.store.GetUserByEmail(ctx, identifier)
	case identifierUsername:
		user, err = s.store.GetUserByUsername(ctx, identifier)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("database error: %w", err)
	}

	if user != nil {
		// Check if account is locked
		if !user.LockedUntil.IsZero() && time.Now().Before(user.LockedUntil) {
			return 0, 0, ErrAccountLocked
		}
		// Check if account is disabled
		if !user.IsActive {
			return 0, 0, ErrAccountDisabled
		}
		// Verify password
		if s.VerifyPassword(password, user.PasswordHash) {
			// Reset failed attempts, update last login
			user.FailedLoginAttempts = 0
			user.LastLogin = time.Now()
			user.LastIP = ip
			user.LockedUntil = time.Time{}
			s.store.UpdateUser(ctx, user)
			return user.ID, 0, nil
		}
		// Wrong password - increment failed attempts
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			user.LockedUntil = time.Now().Add(30 * time.Minute)
		}
		s.store.UpdateUser(ctx, user)
		return 0, 0, ErrInvalidCredentials
	}

	return 0, 0, ErrInvalidCredentials
}

// CreateSession creates a new session for authenticated user
func (s *AuthService) CreateSession(ctx context.Context, userID, adminID int64, ip, userAgent string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &model.Session{
		ID:           sessionID,
		UserID:       userID,
		AdminID:      adminID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		ThemeName:    "dark", // Default theme
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(defaultSessionDuration),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// ValidateSession validates a session token
// Returns userID, adminID (one will be 0), and error
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (userID, adminID int64, err error) {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return 0, 0, fmt.Errorf("database error: %w", err)
	}
	if session == nil {
		return 0, 0, ErrInvalidSession
	}
	if !session.IsActive {
		return 0, 0, ErrInvalidSession
	}
	if time.Now().After(session.ExpiresAt) {
		return 0, 0, ErrSessionExpired
	}

	// Update last activity
	session.LastActivity = time.Now()
	s.store.UpdateSession(ctx, session)

	return session.UserID, session.AdminID, nil
}

// InvalidateSession invalidates a session
func (s *AuthService) InvalidateSession(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *AuthService) InvalidateAllUserSessions(ctx context.Context, userID int64) error {
	return s.store.DeleteUserSessions(ctx, userID)
}

// HashPassword hashes a password using Argon2id
// Format: $argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
func (s *AuthService) HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Encode as PHC string format
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword verifies a password against a hash
func (s *AuthService) VerifyPassword(password, encodedHash string) bool {
	// Parse the PHC format hash
	params, salt, hash, err := parseArgon2Hash(encodedHash)
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, params.keyLen)

	// Constant-time comparison
	return subtle.ConstantTimeCompare(hash, computedHash) == 1
}

// argon2Params holds the parameters extracted from a hash
type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
}

// parseArgon2Hash parses a PHC format Argon2id hash
func parseArgon2Hash(encodedHash string) (params argon2Params, salt, hash []byte, err error) {
	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return params, nil, nil, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return params, nil, nil, errors.New("unsupported algorithm")
	}

	var version int
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return params, nil, nil, fmt.Errorf("failed to parse version: %w", err)
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads)
	if err != nil {
		return params, nil, nil, fmt.Errorf("failed to parse params: %w", err)
	}
	params.keyLen = argonKeyLen

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	return params, salt, hash, nil
}

// generateSessionID generates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Token prefixes per AI.md PART 11
const (
	TokenPrefixAdmin = "adm_"
	TokenPrefixUser  = "usr_"
	TokenPrefixOrg   = "org_"
)

// alphanumeric characters for token generation
const alphanumeric = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateAPIToken generates a new API token with the specified prefix
// Returns the raw token (to show to user once), its hash (to store), and prefix (first 8 chars)
func GenerateAPIToken() (token, hash string, err error) {
	return GenerateAPITokenWithPrefix(TokenPrefixAdmin)
}

// GenerateAPITokenWithPrefix generates a new API token with a custom prefix
// Token format: {prefix}_{random_32_alphanumeric} per AI.md PART 11
func GenerateAPITokenWithPrefix(prefix string) (token, hash string, err error) {
	// Generate 32 random alphanumeric characters
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	// Convert to alphanumeric
	randomChars := make([]byte, 32)
	for i := 0; i < 32; i++ {
		randomChars[i] = alphanumeric[int(randomBytes[i])%len(alphanumeric)]
	}

	token = prefix + string(randomChars)
	hash = HashToken(token)
	return token, hash, nil
}

// GetTokenPrefix returns the display prefix (first 8 chars) for a token
func GetTokenPrefix(token string) string {
	if len(token) >= 8 {
		return token[:8]
	}
	return token
}

// HashToken hashes an API token using SHA-256
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(h[:])
}

// VerifyToken verifies an API token against its hash
func VerifyToken(token, storedHash string) bool {
	computedHash := HashToken(token)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// Registration errors
var (
	ErrUsernameExists = errors.New("username already exists")
	ErrEmailExists    = errors.New("email already exists")
)

// RegisterStore interface for registration operations
type RegisterStore interface {
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
}

// RegisterUser registers a new user with full validation
// Returns the created user ID or validation/storage errors
func (s *AuthService) RegisterUser(ctx context.Context, username, email, password, confirmPassword string) (int64, *ValidationResult, error) {
	// Validate all registration input
	result := ValidateRegistration(username, email, password, confirmPassword)
	if result.HasErrors() {
		return 0, result, nil
	}

	// Normalize inputs (trim and lowercase email)
	username = TrimInput(username)
	email = strings.ToLower(TrimInput(email))

	// Check if username exists
	existingUser, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, nil, fmt.Errorf("database error checking username: %w", err)
	}
	if existingUser != nil {
		result.AddError("username", ErrUsernameExists.Error())
		return 0, result, nil
	}

	// Check if email exists
	existingUser, err = s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, nil, fmt.Errorf("database error checking email: %w", err)
	}
	if existingUser != nil {
		result.AddError("email", ErrEmailExists.Error())
		return 0, result, nil
	}

	// Hash password
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user model
	user := &model.User{
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		Role:              "user",
		ThemePreference:   "dark",
		StorageQuotaBytes: 53687091200, // 50GB default
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Store user
	if err := s.store.(RegisterStore).CreateUser(ctx, user); err != nil {
		return 0, nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user.ID, nil, nil
}

// ChangePassword changes a user's password with validation
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword, confirmPassword string) (*ValidationResult, error) {
	result := NewValidationResult()

	// Validate new password
	if err := ValidatePassword(newPassword); err != nil {
		result.AddError("new_password", err.Error())
	}

	// Check confirmation matches
	if newPassword != confirmPassword {
		result.AddError("confirm_password", "passwords do not match")
	}

	if result.HasErrors() {
		return result, nil
	}

	// Get user
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Verify current password
	if !s.VerifyPassword(currentPassword, user.PasswordHash) {
		result.AddError("current_password", "current password is incorrect")
		return result, nil
	}

	// Hash new password
	passwordHash, err := s.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate all sessions for security
	if err := s.store.DeleteUserSessions(ctx, userID); err != nil {
		// Log but don't fail
		fmt.Printf("WARN: Failed to invalidate sessions after password change: %v\n", err)
	}

	return nil, nil
}
