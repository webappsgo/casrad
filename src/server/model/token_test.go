// Package model - Tests for APIToken helper methods.
// Covers: GetPermissions, SetPermissions, HasPermission with various inputs.
package model

import (
	"testing"
)

func TestAPITokenPermissions(t *testing.T) {
	t.Parallel()

	t.Run("empty_permissions_returns_nil", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		if got := tok.GetPermissions(); got != nil {
			t.Errorf("GetPermissions empty = %v, want nil", got)
		}
	})

	t.Run("set_and_get_permissions", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"user:library", "user:playlists"})
		perms := tok.GetPermissions()
		if len(perms) != 2 {
			t.Fatalf("GetPermissions len = %d, want 2", len(perms))
		}
		if perms[0] != "user:library" {
			t.Errorf("perms[0] = %q, want user:library", perms[0])
		}
		if perms[1] != "user:playlists" {
			t.Errorf("perms[1] = %q, want user:playlists", perms[1])
		}
	})

	t.Run("set_nil_clears_permissions", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"user:library"})
		tok.SetPermissions(nil)
		if tok.Permissions != "" {
			t.Errorf("SetPermissions nil should clear, got %q", tok.Permissions)
		}
	})

	t.Run("set_empty_slice_clears_permissions", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"user:library"})
		tok.SetPermissions([]string{})
		if tok.Permissions != "" {
			t.Errorf("SetPermissions empty slice should clear, got %q", tok.Permissions)
		}
	})
}

func TestAPITokenHasPermission(t *testing.T) {
	t.Parallel()

	t.Run("has_specific_perm", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"user:library", "admin:users"})
		if !tok.HasPermission("user:library") {
			t.Error("HasPermission should return true for existing permission")
		}
		if !tok.HasPermission("admin:users") {
			t.Error("HasPermission should return true for existing permission")
		}
	})

	t.Run("missing_perm_returns_false", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"user:library"})
		if tok.HasPermission("admin:settings") {
			t.Error("HasPermission should return false for missing permission")
		}
	})

	t.Run("wildcard_grants_all", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		tok.SetPermissions([]string{"*"})
		if !tok.HasPermission("user:library") {
			t.Error("wildcard * should grant all permissions")
		}
		if !tok.HasPermission("admin:users") {
			t.Error("wildcard * should grant admin permissions too")
		}
	})

	t.Run("empty_permissions_denies_all", func(t *testing.T) {
		t.Parallel()
		tok := &APIToken{}
		if tok.HasPermission("user:library") {
			t.Error("empty permissions should deny all")
		}
	})
}

func TestSessionIsAdminSession(t *testing.T) {
	t.Parallel()

	t.Run("admin_session", func(t *testing.T) {
		t.Parallel()
		s := &Session{AdminID: 5}
		if !s.IsAdminSession() {
			t.Error("Session with AdminID should be an admin session")
		}
	})

	t.Run("user_session", func(t *testing.T) {
		t.Parallel()
		s := &Session{UserID: 3}
		if s.IsAdminSession() {
			t.Error("Session with only UserID should not be an admin session")
		}
	})

	t.Run("zero_admin_id_not_admin", func(t *testing.T) {
		t.Parallel()
		s := &Session{AdminID: 0, UserID: 1}
		if s.IsAdminSession() {
			t.Error("AdminID=0 should not be admin session")
		}
	})
}
