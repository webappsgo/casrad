// Package service — Tests for BackupService pure constructors and configuration.
// Covers: DefaultRetentionConfig, DefaultBackupConfig, NewBackupService,
// Configure, GetBackupDir, IsEncryptionEnabled, SetEncryptionEnabled,
// ValidateRetentionConfig.
package service

import (
	"testing"
)

// --- DefaultRetentionConfig ---

func TestDefaultRetentionConfigReturnsNonNil(t *testing.T) {
	t.Parallel()
	cfg := DefaultRetentionConfig()
	if cfg == nil {
		t.Fatal("DefaultRetentionConfig returned nil")
	}
}

func TestDefaultRetentionConfigMaxBackupsIsOne(t *testing.T) {
	t.Parallel()
	cfg := DefaultRetentionConfig()
	if cfg.MaxBackups != 1 {
		t.Errorf("MaxBackups = %d, want 1", cfg.MaxBackups)
	}
}

func TestDefaultRetentionConfigWeeklyDisabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultRetentionConfig()
	if cfg.KeepWeekly != 0 {
		t.Errorf("KeepWeekly = %d, want 0 (disabled)", cfg.KeepWeekly)
	}
}

func TestDefaultRetentionConfigMonthlyDisabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultRetentionConfig()
	if cfg.KeepMonthly != 0 {
		t.Errorf("KeepMonthly = %d, want 0 (disabled)", cfg.KeepMonthly)
	}
}

func TestDefaultRetentionConfigYearlyDisabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultRetentionConfig()
	if cfg.KeepYearly != 0 {
		t.Errorf("KeepYearly = %d, want 0 (disabled)", cfg.KeepYearly)
	}
}

// --- DefaultBackupConfig ---

func TestDefaultBackupConfigReturnsNonNil(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg == nil {
		t.Fatal("DefaultBackupConfig returned nil")
	}
}

func TestDefaultBackupConfigNotEncrypted(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg.Encrypted {
		t.Error("default backup should not be encrypted")
	}
}

func TestDefaultBackupConfigSSLNotIncluded(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg.IncludeSSL {
		t.Error("default backup should not include SSL")
	}
}

func TestDefaultBackupConfigDataNotIncluded(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg.IncludeData {
		t.Error("default backup should not include data")
	}
}

func TestDefaultBackupConfigComplianceModeDisabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg.ComplianceMode {
		t.Error("default backup should not be in compliance mode")
	}
}

func TestDefaultBackupConfigHasRetention(t *testing.T) {
	t.Parallel()
	cfg := DefaultBackupConfig()
	if cfg.Retention == nil {
		t.Error("default backup config should have a Retention sub-config")
	}
}

// --- NewBackupService ---

func TestNewBackupServiceReturnsNonNil(t *testing.T) {
	t.Parallel()
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", nil)
	if svc == nil {
		t.Fatal("NewBackupService returned nil")
	}
}

func TestNewBackupServiceNilConfigUsesDefaults(t *testing.T) {
	t.Parallel()
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", nil)
	if svc.config == nil {
		t.Error("NewBackupService(nil config) should use default config")
	}
}

func TestNewBackupServiceWithConfigPreservesDir(t *testing.T) {
	t.Parallel()
	cfg := &BackupConfig{
		Dir:       "/custom/backups",
		Retention: DefaultRetentionConfig(),
	}
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", cfg)
	if svc.GetBackupDir() != "/custom/backups" {
		t.Errorf("GetBackupDir() = %q, want /custom/backups", svc.GetBackupDir())
	}
}

func TestNewBackupServiceNilConfigRetentionUsesDefaults(t *testing.T) {
	t.Parallel()
	// Config without Retention should auto-populate
	cfg := &BackupConfig{Dir: "/tmp/backups"}
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", cfg)
	if svc.config.Retention == nil {
		t.Error("NewBackupService should auto-populate Retention when nil")
	}
}

// --- Configure ---

func TestConfigureUpdatesDir(t *testing.T) {
	t.Parallel()
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", nil)
	newCfg := &BackupConfig{
		Dir:       "/new/backups",
		Retention: DefaultRetentionConfig(),
	}
	svc.Configure(newCfg)
	if svc.GetBackupDir() != "/new/backups" {
		t.Errorf("after Configure, GetBackupDir() = %q, want /new/backups", svc.GetBackupDir())
	}
}

// --- IsEncryptionEnabled / SetEncryptionEnabled ---

func TestIsEncryptionEnabledDefaultFalse(t *testing.T) {
	t.Parallel()
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", nil)
	if svc.IsEncryptionEnabled() {
		t.Error("default encryption should be disabled")
	}
}

func TestSetEncryptionEnabledTrue(t *testing.T) {
	t.Parallel()
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", nil)
	if err := svc.SetEncryptionEnabled(true); err != nil {
		t.Fatalf("SetEncryptionEnabled(true) unexpected error: %v", err)
	}
	if !svc.IsEncryptionEnabled() {
		t.Error("encryption should be enabled after SetEncryptionEnabled(true)")
	}
}

func TestSetEncryptionDisabledInComplianceModeReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &BackupConfig{
		Dir:            "/tmp/backups",
		Retention:      DefaultRetentionConfig(),
		Encrypted:      true,
		ComplianceMode: true,
	}
	svc := NewBackupService("casrad", "1.0.0", "/tmp/config", "/tmp/data", cfg)
	err := svc.SetEncryptionEnabled(false)
	if err != ErrComplianceNoPassword {
		t.Errorf("SetEncryptionEnabled(false) in compliance mode = %v, want ErrComplianceNoPassword", err)
	}
}

// --- ValidateRetentionConfig ---

func TestValidateRetentionConfigValidConfig(t *testing.T) {
	t.Parallel()
	cfg := &RetentionConfig{
		MaxBackups:  7,
		KeepWeekly:  4,
		KeepMonthly: 3,
		KeepYearly:  1,
	}
	warnings := ValidateRetentionConfig(cfg)
	if len(warnings) != 0 {
		t.Errorf("ValidateRetentionConfig(valid) warnings = %v, want none", warnings)
	}
}

func TestValidateRetentionConfigMaxBackupsZeroIsFixed(t *testing.T) {
	t.Parallel()
	cfg := &RetentionConfig{MaxBackups: 0}
	warnings := ValidateRetentionConfig(cfg)
	if len(warnings) == 0 {
		t.Error("ValidateRetentionConfig(MaxBackups=0) should warn and fix")
	}
	if cfg.MaxBackups != 1 {
		t.Errorf("MaxBackups after validation = %d, want 1", cfg.MaxBackups)
	}
}

func TestValidateRetentionConfigNegativeWeeklyIsFixed(t *testing.T) {
	t.Parallel()
	cfg := &RetentionConfig{MaxBackups: 1, KeepWeekly: -1}
	warnings := ValidateRetentionConfig(cfg)
	if len(warnings) == 0 {
		t.Error("ValidateRetentionConfig(KeepWeekly=-1) should warn")
	}
	if cfg.KeepWeekly != 0 {
		t.Errorf("KeepWeekly after validation = %d, want 0", cfg.KeepWeekly)
	}
}

func TestValidateRetentionConfigNegativeMonthlyIsFixed(t *testing.T) {
	t.Parallel()
	cfg := &RetentionConfig{MaxBackups: 1, KeepMonthly: -3}
	warnings := ValidateRetentionConfig(cfg)
	if len(warnings) == 0 {
		t.Error("ValidateRetentionConfig(KeepMonthly=-3) should warn")
	}
	if cfg.KeepMonthly != 0 {
		t.Errorf("KeepMonthly after validation = %d, want 0", cfg.KeepMonthly)
	}
}
