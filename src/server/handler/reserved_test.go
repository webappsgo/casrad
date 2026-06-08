// Package handler - Tests for reserved name checking.
// Covers: IsReservedName for known reserved names, unknown names, and edge cases.
package handler

import (
	"testing"
)

func TestIsReservedName(t *testing.T) {
	t.Parallel()

	reserved := []string{
		"api", "admin", "static", "assets", "healthz", "metrics",
		"login", "logout", "register", "signup", "signin", "auth",
		"oauth", "callback", "webhook", "webhooks",
		"users", "settings", "dashboard",
		"stream", "streams", "radio", "broadcast", "broadcasts",
		"playlist", "playlists", "track", "tracks", "album", "albums",
		"artist", "artists", "podcast", "podcasts",
		"library", "queue", "nowplaying", "player",
	}

	for _, name := range reserved {
		name := name
		t.Run("reserved_"+name, func(t *testing.T) {
			t.Parallel()
			if !IsReservedName(name) {
				t.Errorf("IsReservedName(%q) = false, want true", name)
			}
		})
	}
}

func TestIsNotReservedName(t *testing.T) {
	t.Parallel()

	notReserved := []string{
		"alice", "bob", "charlie", "myusername", "user123", "band-name",
	}

	for _, name := range notReserved {
		name := name
		t.Run("not_reserved_"+name, func(t *testing.T) {
			t.Parallel()
			if IsReservedName(name) {
				t.Errorf("IsReservedName(%q) = true, want false", name)
			}
		})
	}
}

func TestIsReservedNameCaseSensitive(t *testing.T) {
	t.Parallel()

	// The check is case-sensitive — "Admin" is NOT reserved, only "admin" is
	if IsReservedName("Admin") {
		t.Error("IsReservedName is case-sensitive; 'Admin' should not be reserved")
	}
	if IsReservedName("API") {
		t.Error("IsReservedName is case-sensitive; 'API' should not be reserved")
	}
}

func TestIsReservedNameEmpty(t *testing.T) {
	t.Parallel()

	if IsReservedName("") {
		t.Error("empty string should not be reserved")
	}
}

func TestReservedNamesListNotEmpty(t *testing.T) {
	t.Parallel()

	if len(ReservedNames) == 0 {
		t.Error("ReservedNames list must not be empty")
	}
}
