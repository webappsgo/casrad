// Package config — Tests for Load and config defaults.
// Covers: Load (returns non-nil, all defaults set), environment variable overrides
// (CASRAD_ADDRESS, CASRAD_PORT, CASRAD_DEBUG, CASRAD_DB_DRIVER, CASRAD_CACHE_DRIVER,
// CASRAD_REGISTRATION, CASRAD_AUDIO_FORMAT, CASRAD_AUDIO_BITRATE, CASRAD_USERS_DISABLED),
// invalid env values silently use defaults, invalid CASRAD_ADMIN_PATH silently uses default.
// Note: Load calls paths.ServerDB() for the SQLite DSN which may reference OS paths
// in the container — we only assert the returned struct fields, not file existence.
package config

import (
	"testing"
)

// --- Load defaults ---

func TestLoadReturnsNonNil(t *testing.T) {
	t.Parallel()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoadDefaultAddress(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Server.Address != "0.0.0.0" {
		t.Errorf("Server.Address = %q, want 0.0.0.0", cfg.Server.Address)
	}
}

func TestLoadDefaultAdminPath(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Server.AdminPath != "admin" {
		t.Errorf("Server.AdminPath = %q, want admin", cfg.Server.AdminPath)
	}
}

func TestLoadDefaultDatabaseDriver(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Database.Driver = %q, want sqlite", cfg.Database.Driver)
	}
}

func TestLoadDefaultCacheDriver(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Cache.Driver != "memory" {
		t.Errorf("Cache.Driver = %q, want memory", cfg.Cache.Driver)
	}
}

func TestLoadDefaultUsersRegistration(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Users.Registration != "disabled" {
		t.Errorf("Users.Registration = %q, want disabled", cfg.Users.Registration)
	}
}

func TestLoadDefaultAudioFormat(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Audio.DefaultFormat != "mp3" {
		t.Errorf("Audio.DefaultFormat = %q, want mp3", cfg.Audio.DefaultFormat)
	}
}

func TestLoadDefaultAudioBitrate(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Audio.DefaultBitrate != 192 {
		t.Errorf("Audio.DefaultBitrate = %d, want 192", cfg.Audio.DefaultBitrate)
	}
}

func TestLoadDefaultRateLimitEnabled(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if !cfg.Server.RateLimit.Enabled {
		t.Error("Server.RateLimit.Enabled should be true by default")
	}
}

func TestLoadDefaultSessionCookieName(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Server.Session.CookieName != "session" {
		t.Errorf("Session.CookieName = %q, want session", cfg.Server.Session.CookieName)
	}
}

func TestLoadDefaultLanguage(t *testing.T) {
	t.Parallel()
	cfg, _ := Load()
	if cfg.Server.DefaultLanguage != "en" {
		t.Errorf("DefaultLanguage = %q, want en", cfg.Server.DefaultLanguage)
	}
}

// --- Environment variable overrides ---
// These tests use t.Setenv so they cannot call t.Parallel().

func TestLoadEnvAddress(t *testing.T) {
	t.Setenv("CASRAD_ADDRESS", "127.0.0.1")
	cfg, _ := Load()
	if cfg.Server.Address != "127.0.0.1" {
		t.Errorf("Server.Address = %q, want 127.0.0.1", cfg.Server.Address)
	}
}

func TestLoadEnvPort(t *testing.T) {
	t.Setenv("CASRAD_PORT", "8080")
	cfg, _ := Load()
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
}

func TestLoadEnvPortInvalid(t *testing.T) {
	t.Setenv("CASRAD_PORT", "notaport")
	cfg, _ := Load()
	// Invalid port silently uses default (0)
	if cfg.Server.Port != 0 {
		t.Errorf("Server.Port with invalid env = %d, want 0 (default)", cfg.Server.Port)
	}
}

func TestLoadEnvDebug(t *testing.T) {
	t.Setenv("CASRAD_DEBUG", "true")
	cfg, _ := Load()
	if !cfg.Server.Debug {
		t.Error("Server.Debug should be true when CASRAD_DEBUG=true")
	}
}

func TestLoadEnvDebugFalse(t *testing.T) {
	t.Setenv("CASRAD_DEBUG", "false")
	cfg, _ := Load()
	if cfg.Server.Debug {
		t.Error("Server.Debug should be false when CASRAD_DEBUG=false")
	}
}

func TestLoadEnvDBDriver(t *testing.T) {
	t.Setenv("CASRAD_DB_DRIVER", "postgres")
	cfg, _ := Load()
	if cfg.Database.Driver != "postgres" {
		t.Errorf("Database.Driver = %q, want postgres", cfg.Database.Driver)
	}
}

func TestLoadEnvCacheDriver(t *testing.T) {
	t.Setenv("CASRAD_CACHE_DRIVER", "valkey")
	cfg, _ := Load()
	if cfg.Cache.Driver != "valkey" {
		t.Errorf("Cache.Driver = %q, want valkey", cfg.Cache.Driver)
	}
}

func TestLoadEnvRegistration(t *testing.T) {
	t.Setenv("CASRAD_REGISTRATION", "open")
	cfg, _ := Load()
	if cfg.Users.Registration != "open" {
		t.Errorf("Users.Registration = %q, want open", cfg.Users.Registration)
	}
}

func TestLoadEnvAudioFormat(t *testing.T) {
	t.Setenv("CASRAD_AUDIO_FORMAT", "ogg")
	cfg, _ := Load()
	if cfg.Audio.DefaultFormat != "ogg" {
		t.Errorf("Audio.DefaultFormat = %q, want ogg", cfg.Audio.DefaultFormat)
	}
}

func TestLoadEnvAudioBitrate(t *testing.T) {
	t.Setenv("CASRAD_AUDIO_BITRATE", "320")
	cfg, _ := Load()
	if cfg.Audio.DefaultBitrate != 320 {
		t.Errorf("Audio.DefaultBitrate = %d, want 320", cfg.Audio.DefaultBitrate)
	}
}

func TestLoadEnvUsersDisabled(t *testing.T) {
	t.Setenv("CASRAD_USERS_DISABLED", "true")
	cfg, _ := Load()
	if cfg.Users.Enabled {
		t.Error("Users.Enabled should be false when CASRAD_USERS_DISABLED=true")
	}
}

func TestLoadEnvAdminPathInvalid(t *testing.T) {
	// Path traversal attempt should be silently rejected, keeping default "admin"
	t.Setenv("CASRAD_ADMIN_PATH", "../etc/passwd")
	cfg, _ := Load()
	if cfg.Server.AdminPath != "admin" {
		t.Errorf("AdminPath with traversal attempt = %q, want admin (default)", cfg.Server.AdminPath)
	}
}
