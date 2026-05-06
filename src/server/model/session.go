// Package model - Session model
package model

import "time"

// Session represents a user or admin session
type Session struct {
	ID           string
	UserID       int64
	AdminID      int64
	IPAddress    string
	UserAgent    string
	ThemeName    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	LastActivity time.Time
	IsActive     bool
}

// IsAdminSession returns true if this is an admin session
func (s *Session) IsAdminSession() bool {
	return s.AdminID != 0
}
