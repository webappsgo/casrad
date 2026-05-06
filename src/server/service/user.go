// Package service - User service
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/casapps/casrad/src/server/model"
	"github.com/casapps/casrad/src/server/store"
)

// Default user storage quota: 50GB
const defaultStorageQuota int64 = 53687091200

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUsernameInvalid = errors.New("invalid username format")
	ErrUsernameBlocked = errors.New("username not allowed")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailInvalid    = errors.New("invalid email format")
	ErrEmailTaken      = errors.New("email already taken")
	ErrQuotaExceeded   = errors.New("storage quota exceeded")
	// Note: Password validation errors are in validation.go
)

// usernameRegex validates username format
// 3-32 chars, lowercase, starts with letter
var usernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,31}$`)

// emailValidationRegex validates email format
var emailValidationRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// blockedUsernames contains reserved usernames
var blockedUsernames = map[string]bool{
	"admin": true, "administrator": true, "root": true, "system": true,
	"api": true, "server": true, "auth": true, "users": true,
	"orgs": true, "settings": true, "profile": true, "static": true,
	"assets": true, "casrad": true, "casapps": true, "www": true,
	"mail": true, "smtp": true, "ftp": true, "ssh": true,
	"help": true, "support": true, "login": true, "logout": true,
	"register": true, "signup": true, "signin": true, "signout": true,
	"dashboard": true, "home": true, "index": true, "about": true,
	"contact": true, "privacy": true, "terms": true, "legal": true,
	"music": true, "audio": true, "stream": true, "radio": true,
	"podcast": true, "podcasts": true, "audiobook": true, "audiobooks": true,
	"playlist": true, "playlists": true, "track": true, "tracks": true,
	"album": true, "albums": true, "artist": true, "artists": true,
	"library": true, "upload": true, "download": true, "search": true,
	"browse": true, "explore": true, "discover": true, "popular": true,
	"trending": true, "new": true, "latest": true, "featured": true,
	"null": true, "undefined": true, "test": true, "demo": true,
}

// UserStore interface for user operations
type UserStore interface {
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) (int64, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error)

	GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error)
	GetAdminByEmail(ctx context.Context, email string) (*model.Admin, error)
	CreateAdmin(ctx context.Context, admin *model.Admin) (int64, error)
}

// UserService handles user-related business logic
type UserService struct {
	store       UserStore
	authService *AuthService
	baseDataDir string
}

// NewUserService creates a new user service
func NewUserService(s UserStore, auth *AuthService, baseDataDir string) *UserService {
	return &UserService{
		store:       s,
		authService: auth,
		baseDataDir: baseDataDir,
	}
}

// NewUserServiceWithStore creates a new user service with SQLiteStore
func NewUserServiceWithStore(s *store.SQLiteStore, auth *AuthService, baseDataDir string) *UserService {
	return &UserService{
		store:       s,
		authService: auth,
		baseDataDir: baseDataDir,
	}
}

// ValidateUsername validates a username
func (s *UserService) ValidateUsername(username string) error {
	username = strings.ToLower(strings.TrimSpace(username))

	// Check format
	if !usernameRegex.MatchString(username) {
		return ErrUsernameInvalid
	}

	// Check blocklist
	if blockedUsernames[username] {
		return ErrUsernameBlocked
	}

	// Cannot end with _ or -
	if strings.HasSuffix(username, "_") || strings.HasSuffix(username, "-") {
		return ErrUsernameInvalid
	}

	// No consecutive special chars
	if strings.Contains(username, "__") || strings.Contains(username, "--") ||
		strings.Contains(username, "_-") || strings.Contains(username, "-_") {
		return ErrUsernameInvalid
	}

	return nil
}

// ValidateEmail validates an email address
func (s *UserService) ValidateEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !emailValidationRegex.MatchString(email) {
		return ErrEmailInvalid
	}
	return nil
}

// ValidatePassword validates a password using the comprehensive validation service
func (s *UserService) ValidatePassword(password string) error {
	// Use the validation service for comprehensive password checking
	// This includes: length, whitespace, common password checks
	return ValidatePassword(password)
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, username, email, password string) (int64, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))

	// Validate inputs
	if err := s.ValidateUsername(username); err != nil {
		return 0, err
	}
	if err := s.ValidateEmail(email); err != nil {
		return 0, err
	}
	if err := s.ValidatePassword(password); err != nil {
		return 0, err
	}

	// Check if username taken
	existingUser, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingUser != nil {
		return 0, ErrUsernameTaken
	}

	// Also check admins table
	existingAdmin, err := s.store.GetAdminByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingAdmin != nil {
		return 0, ErrUsernameTaken
	}

	// Check if email taken
	existingUser, err = s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingUser != nil {
		return 0, ErrEmailTaken
	}

	// Also check admins table
	existingAdmin, err = s.store.GetAdminByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingAdmin != nil {
		return 0, ErrEmailTaken
	}

	// Hash password
	passwordHash, err := s.authService.HashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user storage directory
	homeDir := filepath.Join(s.baseDataDir, "users", username)
	if err := s.createUserDirectories(homeDir); err != nil {
		return 0, fmt.Errorf("failed to create user directories: %w", err)
	}

	// Create user
	user := &model.User{
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		Role:              "user",
		ThemePreference:   "dark",
		HomeDirectory:     homeDir,
		StorageQuotaBytes: defaultStorageQuota,
		IsActive:          true,
		EmailVerified:     false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	id, err := s.store.CreateUser(ctx, user)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

// CreateAdmin creates a new admin
func (s *UserService) CreateAdmin(ctx context.Context, username, email, password string) (int64, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))

	// Validate inputs
	if err := s.ValidateUsername(username); err != nil {
		return 0, err
	}
	if err := s.ValidateEmail(email); err != nil {
		return 0, err
	}
	if err := s.ValidatePassword(password); err != nil {
		return 0, err
	}

	// Check if username taken in admins
	existingAdmin, err := s.store.GetAdminByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingAdmin != nil {
		return 0, ErrUsernameTaken
	}

	// Check if username taken in users
	existingUser, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingUser != nil {
		return 0, ErrUsernameTaken
	}

	// Check if email taken in admins
	existingAdmin, err = s.store.GetAdminByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingAdmin != nil {
		return 0, ErrEmailTaken
	}

	// Check if email taken in users
	existingUser, err = s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}
	if existingUser != nil {
		return 0, ErrEmailTaken
	}

	// Hash password
	passwordHash, err := s.authService.HashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin
	admin := &model.Admin{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	id, err := s.store.CreateAdmin(ctx, admin)
	if err != nil {
		return 0, fmt.Errorf("failed to create admin: %w", err)
	}

	return id, nil
}

// createUserDirectories creates the user's storage directories
func (s *UserService) createUserDirectories(homeDir string) error {
	dirs := []string{
		homeDir,
		filepath.Join(homeDir, "music"),
		filepath.Join(homeDir, "podcasts"),
		filepath.Join(homeDir, "audiobooks"),
		filepath.Join(homeDir, "radio"),
		filepath.Join(homeDir, "playlists"),
		filepath.Join(homeDir, "recordings"),
		filepath.Join(homeDir, "transcodes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetUser gets a user by ID
func (s *UserService) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByUsername gets a user by username
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := s.store.GetUserByUsername(ctx, strings.ToLower(username))
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser updates a user's profile
func (s *UserService) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "theme_preference":
			if v, ok := value.(string); ok {
				user.ThemePreference = v
			}
		case "bio":
			if v, ok := value.(string); ok {
				user.Bio = v
			}
		case "website":
			if v, ok := value.(string); ok {
				user.Website = v
			}
		case "location":
			if v, ok := value.(string); ok {
				user.Location = v
			}
		case "avatar_url":
			if v, ok := value.(string); ok {
				user.AvatarURL = v
			}
		case "settings":
			if v, ok := value.(string); ok {
				user.Settings = v
			}
		}
	}

	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(ctx, user)
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	if err := s.ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	passwordHash, err := s.authService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(ctx, user)
}

// DeleteUser deletes a user and their data
func (s *UserService) DeleteUser(ctx context.Context, userID int64) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Delete user from database
	if err := s.store.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Optionally delete user's home directory
	// Note: This is destructive - consider archiving instead
	if user.HomeDirectory != "" {
		os.RemoveAll(user.HomeDirectory)
	}

	return nil
}

// ListUsers lists all users with pagination
func (s *UserService) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.ListUsers(ctx, offset, limit)
}

// VerifyEmail marks a user's email as verified
func (s *UserService) VerifyEmail(ctx context.Context, userID int64) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(ctx, user)
}

// SetUserActive enables or disables a user account
func (s *UserService) SetUserActive(ctx context.Context, userID int64, active bool) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.IsActive = active
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(ctx, user)
}

// UpdateStorageUsed updates a user's storage used bytes
func (s *UserService) UpdateStorageUsed(ctx context.Context, userID int64, bytesUsed int64) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.StorageUsedBytes = bytesUsed
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(ctx, user)
}

// CheckStorageQuota checks if a user has enough storage quota
func (s *UserService) CheckStorageQuota(ctx context.Context, userID int64, additionalBytes int64) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.StorageUsedBytes+additionalBytes > user.StorageQuotaBytes {
		return ErrQuotaExceeded
	}

	return nil
}
