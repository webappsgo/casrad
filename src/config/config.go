// Package config handles configuration loading and validation
// See AI.md PART 5 for configuration specification
package config

import (
	"os"
	"strconv"

	"github.com/casapps/casrad/src/paths"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Cache    CacheConfig    `yaml:"cache"`
	Users    UsersConfig    `yaml:"users"`
	Audio    AudioConfig    `yaml:"audio"`
}

// ServerConfig holds HTTP server settings per AI.md PART 12
type ServerConfig struct {
	Address         string `yaml:"address"`
	Port            int    `yaml:"port"`
	AdminPath       string `yaml:"admin_path"`
	// For security.txt per PART 11
	SecurityContact string `yaml:"security_contact"`
	Debug           bool   `yaml:"debug"`

	// Request limits per PART 12
	Limits LimitsConfig `yaml:"limits"`

	// Session configuration per PART 12
	Session SessionConfig `yaml:"session"`

	// Rate limiting per PART 12
	RateLimit RateLimitConfig `yaml:"rate_limit"`

	// i18n per PART 12
	DefaultLanguage string `yaml:"default_language"`
}

// LimitsConfig holds request limit settings per AI.md PART 12
type LimitsConfig struct {
	// bytes, default 10MB
	MaxBodySize int64 `yaml:"max_body_size"`
	// seconds, default 30
	ReadTimeout int `yaml:"read_timeout"`
	// seconds, default 30
	WriteTimeout int `yaml:"write_timeout"`
	// seconds, default 120
	IdleTimeout int `yaml:"idle_timeout"`
}

// SessionConfig holds session settings per AI.md PART 12
type SessionConfig struct {
	// seconds, default 30 days (2592000)
	AdminMaxAge int `yaml:"admin_max_age"`
	// seconds, default 24 hours (86400)
	AdminIdleTimeout int `yaml:"admin_idle_timeout"`
	// seconds, default 7 days (604800)
	UserMaxAge int `yaml:"user_max_age"`
	// seconds, default 24 hours (86400)
	UserIdleTimeout int `yaml:"user_idle_timeout"`
	// default "session"
	CookieName string `yaml:"cookie_name"`
	// auto, true, false
	Secure string `yaml:"secure"`
	// strict, lax, none
	SameSite string `yaml:"same_site"`
}

// RateLimitConfig holds rate limiting settings per AI.md PART 12
type RateLimitConfig struct {
	Enabled  bool `yaml:"enabled"`
	// per window, default 120
	Requests int `yaml:"requests"`
	// seconds, default 60
	Window int `yaml:"window"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// CacheConfig holds cache settings
type CacheConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// UsersConfig holds multi-user settings
type UsersConfig struct {
	Enabled      bool   `yaml:"enabled"`
	// disabled, public, private, approval
	Registration string `yaml:"registration"`
}

// AudioConfig holds audio streaming settings
type AudioConfig struct {
	DefaultFormat  string `yaml:"default_format"`
	DefaultBitrate int    `yaml:"default_bitrate"`
}

// Load loads configuration from file and environment
// Per AI.md PART 12: Invalid config warns and uses default, never fails startup
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address:         "0.0.0.0",
			// Auto-select from 64000-64999
			Port: 0,
			AdminPath:       "admin",
			Debug:           false,
			DefaultLanguage: "en",
			// Request limits per PART 12
			Limits: LimitsConfig{
				// 10MB
				MaxBodySize: 10 * 1024 * 1024,
				// 30s
				ReadTimeout: 30,
				// 30s
				WriteTimeout: 30,
				// 120s
				IdleTimeout: 120,
			},
			// Session config per PART 12
			Session: SessionConfig{
				// 30 days
				AdminMaxAge: 2592000,
				// 24 hours
				AdminIdleTimeout: 86400,
				// 7 days
				UserMaxAge: 604800,
				// 24 hours
				UserIdleTimeout: 86400,
				CookieName:       "session",
				Secure:           "auto",
				SameSite:         "lax",
			},
			// Rate limiting per PART 12
			RateLimit: RateLimitConfig{
				Enabled:  true,
				Requests: 120,
				Window:   60,
			},
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
		},
		Cache: CacheConfig{
			Driver: "memory",
		},
		Users: UsersConfig{
			Enabled:      true,
			Registration: "disabled",
		},
		Audio: AudioConfig{
			DefaultFormat:  "mp3",
			DefaultBitrate: 192,
		},
	}

	// Load from environment variables
	if addr := os.Getenv("CASRAD_ADDRESS"); addr != "" {
		cfg.Server.Address = addr
	}
	if port := os.Getenv("CASRAD_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if adminPath := os.Getenv("CASRAD_ADMIN_PATH"); adminPath != "" {
		if safe, err := SafePath(adminPath); err == nil {
			cfg.Server.AdminPath = safe
		}
		// Invalid paths silently ignored, use default
	}
	cfg.Server.Debug = MustParseBool(os.Getenv("CASRAD_DEBUG"), false)

	// Database configuration from environment
	if driver := os.Getenv("CASRAD_DB_DRIVER"); driver != "" {
		cfg.Database.Driver = driver
	}
	if dsn := os.Getenv("CASRAD_DB_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	} else if cfg.Database.Driver == "sqlite" {
		cfg.Database.DSN = paths.ServerDB()
	}

	// Cache configuration from environment
	if driver := os.Getenv("CASRAD_CACHE_DRIVER"); driver != "" {
		cfg.Cache.Driver = driver
	}
	if dsn := os.Getenv("CASRAD_CACHE_DSN"); dsn != "" {
		cfg.Cache.DSN = dsn
	}

	// User configuration from environment
	if IsTruthy(os.Getenv("CASRAD_USERS_DISABLED")) {
		cfg.Users.Enabled = false
	}
	if reg := os.Getenv("CASRAD_REGISTRATION"); reg != "" {
		cfg.Users.Registration = reg
	}

	// Audio configuration from environment
	if format := os.Getenv("CASRAD_AUDIO_FORMAT"); format != "" {
		cfg.Audio.DefaultFormat = format
	}
	if bitrate := os.Getenv("CASRAD_AUDIO_BITRATE"); bitrate != "" {
		if b, err := strconv.Atoi(bitrate); err == nil {
			cfg.Audio.DefaultBitrate = b
		}
	}

	return cfg, nil
}

