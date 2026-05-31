// Package model contains data structures
package model

import "time"

// Admin represents a server admin account (separate from users per CLAUDE.md)
type Admin struct {
	ID                  int64
	Username            string
	Email               string
	PasswordHash        string
	TOTPSecret          string
	// admin, super_admin
	Role                string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastLogin           time.Time
	LastIP              string
	FailedLoginAttempts int
	LockedUntil         time.Time
}

// User represents a regular user account
type User struct {
	ID                  int64
	Username            string
	Email               string
	PasswordHash        string
	TOTPSecret          string
	Role                string
	ThemePreference     string
	HomeDirectory       string
	StorageQuotaBytes   int64
	StorageUsedBytes    int64
	IsActive            bool
	EmailVerified       bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastLogin           time.Time
	LastIP              string
	FailedLoginAttempts int
	LockedUntil         time.Time
	Settings            string
	AvatarURL           string
	Bio                 string
	Website             string
	Location            string
}
