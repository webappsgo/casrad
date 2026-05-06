// Package service - Backup and Restore service
// See AI.md PART 22 for backup specification
package service

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrBackupNotFound       = errors.New("backup file not found")
	ErrBackupCorrupted      = errors.New("backup file corrupted")
	ErrBackupPasswordNeeded = errors.New("encrypted backup requires password")
	ErrBackupInvalidPassword = errors.New("invalid backup password")
	ErrBackupVersionMismatch = errors.New("backup version incompatible")
	ErrComplianceNoPassword = errors.New("compliance mode requires backup encryption password")
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeDaily       BackupType = "daily"
	BackupTypeHourly      BackupType = "hourly"
)

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusVerified  BackupStatus = "verified"
)

// BackupManifest contains metadata about the backup
type BackupManifest struct {
	Version          string    `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
	AppVersion       string    `json:"app_version"`
	Contents         []string  `json:"contents"`
	Encrypted        bool      `json:"encrypted"`
	EncryptionMethod string    `json:"encryption_method,omitempty"`
	Checksum         string    `json:"checksum"`
	BackupType       string    `json:"backup_type"`
	BaseBackup       string    `json:"base_backup,omitempty"`
}

// RetentionConfig holds backup retention settings per PART 22
type RetentionConfig struct {
	MaxBackups  int `json:"max_backups"`  // Daily full backups to keep (default: 1)
	KeepWeekly  int `json:"keep_weekly"`  // Weekly backups (Sunday) - 0 = disabled
	KeepMonthly int `json:"keep_monthly"` // Monthly backups (1st) - 0 = disabled
	KeepYearly  int `json:"keep_yearly"`  // Yearly backups (Jan 1st) - 0 = disabled
}

// DefaultRetentionConfig returns the default retention configuration
func DefaultRetentionConfig() *RetentionConfig {
	return &RetentionConfig{
		MaxBackups:  1, // Yesterday only
		KeepWeekly:  0, // Disabled
		KeepMonthly: 0, // Disabled
		KeepYearly:  0, // Disabled
	}
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	Dir           string           `json:"dir"`
	Retention     *RetentionConfig `json:"retention"`
	Encrypted     bool             `json:"encrypted"`
	PasswordHint  string           `json:"password_hint,omitempty"`
	IncludeSSL    bool             `json:"include_ssl"`
	IncludeData   bool             `json:"include_data"`
	ComplianceMode bool            `json:"compliance_mode"`
}

// DefaultBackupConfig returns the default backup configuration
func DefaultBackupConfig() *BackupConfig {
	return &BackupConfig{
		Dir:           "/etc/casrad/backups",
		Retention:     DefaultRetentionConfig(),
		Encrypted:     false,
		IncludeSSL:    false,
		IncludeData:   false,
		ComplianceMode: false,
	}
}

// BackupInfo represents a backup file and its metadata
type BackupInfo struct {
	Filename    string       `json:"filename"`
	Path        string       `json:"path"`
	Size        int64        `json:"size"`
	CreatedAt   time.Time    `json:"created_at"`
	BackupType  BackupType   `json:"backup_type"`
	Encrypted   bool         `json:"encrypted"`
	Verified    bool         `json:"verified"`
	RetentionTag string      `json:"retention_tag,omitempty"` // daily, weekly, monthly, yearly
}

// BackupService provides backup and restore functionality
type BackupService struct {
	config      *BackupConfig
	appName     string
	appVersion  string
	configDir   string
	dataDir     string
	mu          sync.RWMutex
}

// NewBackupService creates a new backup service
func NewBackupService(appName, appVersion, configDir, dataDir string, config *BackupConfig) *BackupService {
	if config == nil {
		config = DefaultBackupConfig()
	}
	if config.Retention == nil {
		config.Retention = DefaultRetentionConfig()
	}

	return &BackupService{
		config:     config,
		appName:    appName,
		appVersion: appVersion,
		configDir:  configDir,
		dataDir:    dataDir,
	}
}

// Configure updates the backup configuration
func (s *BackupService) Configure(config *BackupConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// CreateBackup creates a backup with optional encryption
// Per PART 22: Returns filename of created backup
func (s *BackupService) CreateBackup(backupType BackupType, password string, createdBy string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check compliance mode
	if s.config.ComplianceMode && !s.config.Encrypted && password == "" {
		return "", ErrComplianceNoPassword
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.config.Dir, 0750); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate filename per PART 22 spec
	timestamp := time.Now().Format("2006-01-02_150405")
	var filename string
	switch backupType {
	case BackupTypeFull:
		filename = fmt.Sprintf("%s_backup_%s.tar.gz", s.appName, timestamp)
	case BackupTypeDaily:
		filename = fmt.Sprintf("%s-daily.tar.gz", s.appName)
	case BackupTypeHourly:
		filename = fmt.Sprintf("%s-hourly.tar.gz", s.appName)
	default:
		filename = fmt.Sprintf("%s_backup_%s.tar.gz", s.appName, timestamp)
	}

	// Add .enc extension if encrypted
	encrypted := s.config.Encrypted || password != ""
	if encrypted {
		filename += ".enc"
	}

	backupPath := filepath.Join(s.config.Dir, filename)

	// Collect files to backup
	contents, err := s.collectBackupContents()
	if err != nil {
		return "", fmt.Errorf("failed to collect backup contents: %w", err)
	}

	// Create manifest
	manifest := &BackupManifest{
		Version:    "1.0.0",
		CreatedAt:  time.Now(),
		CreatedBy:  createdBy,
		AppVersion: s.appVersion,
		Contents:   contents,
		Encrypted:  encrypted,
		BackupType: string(backupType),
	}

	if encrypted {
		manifest.EncryptionMethod = "AES-256-GCM"
	}

	// Create tar.gz archive in memory
	archiveData, checksum, err := s.createArchive(contents, manifest)
	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	manifest.Checksum = "sha256:" + checksum

	// Encrypt if needed
	var finalData []byte
	if encrypted {
		finalData, err = s.encryptArchive(archiveData, password)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt backup: %w", err)
		}
	} else {
		finalData = archiveData
	}

	// Write to file
	if err := os.WriteFile(backupPath, finalData, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	// Verify backup immediately after creation per PART 22
	if err := s.VerifyBackup(backupPath, password); err != nil {
		// Delete failed backup
		os.Remove(backupPath)
		return "", fmt.Errorf("backup verification failed: %w", err)
	}

	return filename, nil
}

// collectBackupContents returns list of files/dirs to backup per PART 22
func (s *BackupService) collectBackupContents() ([]string, error) {
	var contents []string

	// Always include: server.yml, server.db, users.db
	configFiles := []string{"server.yml", "server.db", "users.db"}
	for _, f := range configFiles {
		path := filepath.Join(s.configDir, f)
		if _, err := os.Stat(path); err == nil {
			contents = append(contents, f)
		}
	}

	// Include custom templates if exist
	templateDir := filepath.Join(s.configDir, "template")
	if info, err := os.Stat(templateDir); err == nil && info.IsDir() {
		contents = append(contents, "template/")
	}

	// Include custom themes if exist
	themeDir := filepath.Join(s.configDir, "theme")
	if info, err := os.Stat(themeDir); err == nil && info.IsDir() {
		contents = append(contents, "theme/")
	}

	// Include SSL certificates if configured
	if s.config.IncludeSSL {
		sslDir := filepath.Join(s.configDir, "ssl")
		if info, err := os.Stat(sslDir); err == nil && info.IsDir() {
			contents = append(contents, "ssl/")
		}
	}

	// Include data directory if configured
	if s.config.IncludeData {
		if info, err := os.Stat(s.dataDir); err == nil && info.IsDir() {
			contents = append(contents, "data/")
		}
	}

	return contents, nil
}

// createArchive creates a tar.gz archive and returns the data and checksum
func (s *BackupService) createArchive(contents []string, manifest *BackupManifest) ([]byte, string, error) {
	// Create buffer for archive
	var buf strings.Builder
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add manifest first
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, "", err
	}

	if err := s.addToTar(tarWriter, "manifest.json", manifestJSON); err != nil {
		return nil, "", err
	}

	// Add each content item
	for _, item := range contents {
		var srcPath string
		if strings.HasPrefix(item, "data/") {
			srcPath = filepath.Join(s.dataDir, strings.TrimPrefix(item, "data/"))
		} else {
			srcPath = filepath.Join(s.configDir, item)
		}

		if strings.HasSuffix(item, "/") {
			// Directory - walk and add all files
			if err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				relPath, _ := filepath.Rel(s.configDir, path)
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				return s.addToTar(tarWriter, relPath, data)
			}); err != nil {
				return nil, "", err
			}
		} else {
			// Single file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return nil, "", err
			}
			if err := s.addToTar(tarWriter, item, data); err != nil {
				return nil, "", err
			}
		}
	}

	// Close writers
	if err := tarWriter.Close(); err != nil {
		return nil, "", err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", err
	}

	archiveData := []byte(buf.String())

	// Calculate checksum
	hash := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(hash[:])

	return archiveData, checksum, nil
}

// addToTar adds a file to the tar archive
func (s *BackupService) addToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0600,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write(data)
	return err
}

// encryptArchive encrypts data using AES-256-GCM with Argon2id key derivation
// Per PART 22 spec
func (s *BackupService) encryptArchive(data []byte, password string) ([]byte, error) {
	// Generate salt for Argon2id
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key using Argon2id per PART 22
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Prepend salt and nonce to ciphertext
	result := make([]byte, len(salt)+len(nonce)+len(ciphertext))
	copy(result, salt)
	copy(result[len(salt):], nonce)
	copy(result[len(salt)+len(nonce):], ciphertext)

	return result, nil
}

// decryptArchive decrypts data using AES-256-GCM with Argon2id key derivation
func (s *BackupService) decryptArchive(data []byte, password string) ([]byte, error) {
	if len(data) < 28 { // 16 (salt) + 12 (nonce) minimum
		return nil, ErrBackupCorrupted
	}

	// Extract salt
	salt := data[:16]

	// Derive key using Argon2id
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < 16+nonceSize {
		return nil, ErrBackupCorrupted
	}
	nonce := data[16 : 16+nonceSize]
	ciphertext := data[16+nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrBackupInvalidPassword
	}

	return plaintext, nil
}

// VerifyBackup verifies a backup file per PART 22 verification requirements
func (s *BackupService) VerifyBackup(backupPath string, password string) error {
	// Check file exists
	info, err := os.Stat(backupPath)
	if err != nil {
		return ErrBackupNotFound
	}

	// Check size > 0
	if info.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// Read file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Check if encrypted
	encrypted := strings.HasSuffix(backupPath, ".enc")

	var archiveData []byte
	if encrypted {
		if password == "" {
			return ErrBackupPasswordNeeded
		}
		archiveData, err = s.decryptArchive(data, password)
		if err != nil {
			return err
		}
	} else {
		archiveData = data
	}

	// Parse gzip
	gzReader, err := gzip.NewReader(strings.NewReader(string(archiveData)))
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}
	defer gzReader.Close()

	// Parse tar and find manifest
	tarReader := tar.NewReader(gzReader)
	var manifest *BackupManifest
	var calculatedChecksum string

	// First pass - read manifest
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid tar format: %w", err)
		}

		if header.Name == "manifest.json" {
			manifestData := make([]byte, header.Size)
			if _, err := io.ReadFull(tarReader, manifestData); err != nil {
				return fmt.Errorf("failed to read manifest: %w", err)
			}
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				return fmt.Errorf("invalid manifest: %w", err)
			}
		}
	}

	if manifest == nil {
		return fmt.Errorf("manifest not found in backup")
	}

	// Verify checksum
	hash := sha256.Sum256(archiveData)
	calculatedChecksum = "sha256:" + hex.EncodeToString(hash[:])
	if manifest.Checksum != calculatedChecksum {
		return ErrBackupCorrupted
	}

	// Test extract all files to temp dir
	tempDir, err := os.MkdirTemp("", "backup-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Re-read archive for extraction
	gzReader2, _ := gzip.NewReader(strings.NewReader(string(archiveData)))
	tarReader2 := tar.NewReader(gzReader2)

	for {
		header, err := tarReader2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}

		targetPath := filepath.Join(tempDir, header.Name)
		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(targetPath, 0750)
		} else {
			os.MkdirAll(filepath.Dir(targetPath), 0750)
			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}
			if _, err := io.Copy(f, tarReader2); err != nil {
				f.Close()
				return fmt.Errorf("extraction failed: %w", err)
			}
			f.Close()
		}
	}
	gzReader2.Close()

	// Verify database integrity (SQLite)
	dbPath := filepath.Join(tempDir, "server.db")
	if _, err := os.Stat(dbPath); err == nil {
		// Simple integrity check - try to open and read
		data, err := os.ReadFile(dbPath)
		if err != nil {
			return fmt.Errorf("database integrity check failed: %w", err)
		}
		// Check SQLite header magic bytes
		if len(data) < 16 || string(data[:16]) != "SQLite format 3\x00" {
			return fmt.Errorf("database integrity check failed: invalid SQLite header")
		}
	}

	return nil
}

// RestoreBackup restores from a backup file
// Per PART 22: Requires password for encrypted backups
func (s *BackupService) RestoreBackup(backupPath string, password string) error {
	// Verify backup first
	if err := s.VerifyBackup(backupPath, password); err != nil {
		return err
	}

	// Read file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	// Decrypt if needed
	var archiveData []byte
	if strings.HasSuffix(backupPath, ".enc") {
		archiveData, err = s.decryptArchive(data, password)
		if err != nil {
			return err
		}
	} else {
		archiveData = data
	}

	// Extract to config directory
	gzReader, err := gzip.NewReader(strings.NewReader(string(archiveData)))
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip manifest
		if header.Name == "manifest.json" {
			continue
		}

		var targetPath string
		if strings.HasPrefix(header.Name, "data/") {
			targetPath = filepath.Join(s.dataDir, strings.TrimPrefix(header.Name, "data/"))
		} else {
			targetPath = filepath.Join(s.configDir, header.Name)
		}

		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(targetPath, 0750)
		} else {
			os.MkdirAll(filepath.Dir(targetPath), 0750)
			f, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return err
			}
			f.Close()
			os.Chmod(targetPath, 0600)
		}
	}

	return nil
}

// ListBackups returns list of available backups
func (s *BackupService) ListBackups() ([]*BackupInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var backups []*BackupInfo

	entries, err := os.ReadDir(s.config.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return backups, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".tar.gz.enc") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backup := &BackupInfo{
			Filename:  name,
			Path:      filepath.Join(s.config.Dir, name),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Encrypted: strings.HasSuffix(name, ".enc"),
		}

		// Determine backup type from filename
		if strings.HasPrefix(name, s.appName+"-daily") {
			backup.BackupType = BackupTypeDaily
		} else if strings.HasPrefix(name, s.appName+"-hourly") {
			backup.BackupType = BackupTypeHourly
		} else {
			backup.BackupType = BackupTypeFull
		}

		backups = append(backups, backup)
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// ApplyRetention applies retention policy per PART 22
// Only call after successful backup creation and verification
func (s *BackupService) ApplyRetention() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	backups, err := s.ListBackups()
	if err != nil {
		return err
	}

	// Separate by type
	var fullBackups []*BackupInfo
	for _, b := range backups {
		if b.BackupType == BackupTypeFull {
			fullBackups = append(fullBackups, b)
		}
	}

	// Tag backups per PART 22 retention priority
	now := time.Now()
	var toKeep []*BackupInfo
	var toDelete []*BackupInfo

	// Tag yearly (Jan 1st) - highest priority
	yearlyKept := 0
	for _, b := range fullBackups {
		if b.CreatedAt.Month() == time.January && b.CreatedAt.Day() == 1 {
			if yearlyKept < s.config.Retention.KeepYearly {
				b.RetentionTag = "yearly"
				toKeep = append(toKeep, b)
				yearlyKept++
			}
		}
	}

	// Tag monthly (1st of month)
	monthlyKept := 0
	for _, b := range fullBackups {
		if b.RetentionTag != "" {
			continue
		}
		if b.CreatedAt.Day() == 1 {
			if monthlyKept < s.config.Retention.KeepMonthly {
				b.RetentionTag = "monthly"
				toKeep = append(toKeep, b)
				monthlyKept++
			}
		}
	}

	// Tag weekly (Sunday)
	weeklyKept := 0
	for _, b := range fullBackups {
		if b.RetentionTag != "" {
			continue
		}
		if b.CreatedAt.Weekday() == time.Sunday {
			if weeklyKept < s.config.Retention.KeepWeekly {
				b.RetentionTag = "weekly"
				toKeep = append(toKeep, b)
				weeklyKept++
			}
		}
	}

	// Tag daily (up to max_backups)
	dailyKept := 0
	for _, b := range fullBackups {
		if b.RetentionTag != "" {
			continue
		}
		// Only keep backups within max_backups days
		daysDiff := int(now.Sub(b.CreatedAt).Hours() / 24)
		if daysDiff < s.config.Retention.MaxBackups {
			b.RetentionTag = "daily"
			toKeep = append(toKeep, b)
			dailyKept++
		}
	}

	// Everything not tagged is deleted
	for _, b := range fullBackups {
		if b.RetentionTag == "" {
			toDelete = append(toDelete, b)
		}
	}

	// Delete old backups
	for _, b := range toDelete {
		if err := os.Remove(b.Path); err != nil {
			return fmt.Errorf("failed to delete old backup %s: %w", b.Filename, err)
		}
	}

	return nil
}

// DeleteBackup deletes a specific backup file
func (s *BackupService) DeleteBackup(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.config.Dir, filename)

	// Verify it exists and is a backup file
	if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".tar.gz.enc") {
		return fmt.Errorf("invalid backup filename")
	}

	return os.Remove(path)
}

// GetBackupDir returns the backup directory path
func (s *BackupService) GetBackupDir() string {
	return s.config.Dir
}

// IsEncryptionEnabled returns whether backup encryption is enabled
func (s *BackupService) IsEncryptionEnabled() bool {
	return s.config.Encrypted
}

// SetEncryptionEnabled enables or disables backup encryption
// Per PART 22: Cannot disable if compliance mode is enabled
func (s *BackupService) SetEncryptionEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.ComplianceMode && !enabled {
		return ErrComplianceNoPassword
	}

	s.config.Encrypted = enabled
	return nil
}

// ValidateRetentionConfig validates retention settings per PART 22
// Returns warnings but doesn't error (server must start)
func ValidateRetentionConfig(cfg *RetentionConfig) []string {
	var warnings []string

	if cfg.MaxBackups < 1 {
		warnings = append(warnings, fmt.Sprintf("max_backups: %d invalid, using default 1", cfg.MaxBackups))
		cfg.MaxBackups = 1
	}

	if cfg.KeepWeekly < 0 {
		warnings = append(warnings, fmt.Sprintf("keep_weekly: %d invalid, using default 0", cfg.KeepWeekly))
		cfg.KeepWeekly = 0
	}

	if cfg.KeepMonthly < 0 {
		warnings = append(warnings, fmt.Sprintf("keep_monthly: %d invalid, using default 0", cfg.KeepMonthly))
		cfg.KeepMonthly = 0
	}

	if cfg.KeepYearly < 0 {
		warnings = append(warnings, fmt.Sprintf("keep_yearly: %d invalid, using default 0", cfg.KeepYearly))
		cfg.KeepYearly = 0
	}

	// Warning thresholds per PART 22
	if cfg.MaxBackups > 7 {
		warnings = append(warnings, fmt.Sprintf("max_backups: %d exceeds recommended 7 (%d days of daily backups)", cfg.MaxBackups, cfg.MaxBackups))
	}

	if cfg.KeepWeekly > 8 {
		warnings = append(warnings, fmt.Sprintf("keep_weekly: %d exceeds recommended 8 (more than 2 months of weekly backups)", cfg.KeepWeekly))
	}

	if cfg.KeepMonthly > 12 {
		warnings = append(warnings, fmt.Sprintf("keep_monthly: %d exceeds recommended 12 (more than a year of monthly backups)", cfg.KeepMonthly))
	}

	if cfg.KeepYearly > 2 {
		warnings = append(warnings, fmt.Sprintf("keep_yearly: %d exceeds recommended 2 (more than 2 years of yearly backups)", cfg.KeepYearly))
	}

	return warnings
}
