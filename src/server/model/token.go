// Package model - API token model
package model

import (
	"encoding/json"
	"time"
)

// APIToken represents an API token
type APIToken struct {
	ID          int64
	UserID      int64
	// Only shown once on creation
	Token       string
	Name        string
	// JSON array stored in database
	Permissions string
	LastUsed    time.Time
	LastIP      string
	UseCount    int64
	ExpiresAt   time.Time
	IsActive    bool
	CreatedAt   time.Time
}

// GetPermissions returns the permissions as a slice
func (t *APIToken) GetPermissions() []string {
	if t.Permissions == "" {
		return nil
	}
	var perms []string
	json.Unmarshal([]byte(t.Permissions), &perms)
	return perms
}

// SetPermissions sets the permissions from a slice
func (t *APIToken) SetPermissions(perms []string) {
	if len(perms) == 0 {
		t.Permissions = ""
		return
	}
	data, _ := json.Marshal(perms)
	t.Permissions = string(data)
}

// HasPermission checks if the token has a specific permission
func (t *APIToken) HasPermission(perm string) bool {
	for _, p := range t.GetPermissions() {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}
