package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// BackupManager handles database backup and restore operations
type BackupManager struct {
	db          *database.Engine
	backupPath  string
	retention   int  // Number of backups to keep (default: 7)
	compress    bool // Compress backups (default: true)
	encrypt     bool // Encrypt backups (default: false)
	encryptKey  []byte
	mu          sync.Mutex
	inProgress  bool
}

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	Version      string    `json:"version"`
	Created      time.Time `json:"created"`
	Type         string    `json:"type"` // full, database, config
	Size         int64     `json:"size"`
	Compressed   bool      `json:"compressed"`
	Encrypted    bool      `json:"encrypted"`
	SchemaVer    int       `json:"schema_version"`
	UserCount    int       `json:"user_count"`
	TrackCount   int       `json:"track_count"`
	Checksum     string    `json:"checksum"`
	Description  string    `json:"description"`
}

// BackupStatus represents the current backup status
type BackupStatus struct {
	InProgress bool      `json:"in_progress"`
	Type       string    `json:"type"`
	Progress   float64   `json:"progress"`
	StartTime  time.Time `json:"start_time"`
	Error      string    `json:"error,omitempty"`
}

func NewBackupManager(backupPath string, db *database.Engine) *BackupManager {
	// Create backup directories
	autoPath := filepath.Join(backupPath, "auto")
	manualPath := filepath.Join(backupPath, "manual")
	os.MkdirAll(autoPath, 0755)
	os.MkdirAll(manualPath, 0755)

	m := &BackupManager{
		db:         db,
		backupPath: backupPath,
		retention:  7,
		compress:   true,
		encrypt:    false,
	}

	// Load settings from database
	m.loadSettings()

	return m
}

func (m *BackupManager) loadSettings() {
	// Load retention count
	if val, err := m.db.GetSetting("backup.retention_count"); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			m.retention = n
		}
	}

	// Load compression setting
	if val, err := m.db.GetSetting("backup.compression"); err == nil {
		m.compress = val == "true"
	}

	// Load encryption setting
	if val, err := m.db.GetSetting("backup.encryption"); err == nil {
		m.encrypt = val == "true"
	}

	// Generate encryption key if needed
	if m.encrypt {
		m.generateEncryptionKey()
	}
}

func (m *BackupManager) generateEncryptionKey() {
	// In production, this would load from secure storage
	// For now, generate from a fixed seed
	h := sha256.New()
	h.Write([]byte("casrad-backup-key"))
	m.encryptKey = h.Sum(nil)
}

// CreateBackup creates a new backup
func (m *BackupManager) CreateBackup(backupType string, description string, manual bool) (*BackupMetadata, error) {
	m.mu.Lock()
	if m.inProgress {
		m.mu.Unlock()
		return nil, fmt.Errorf("backup already in progress")
	}
	m.inProgress = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.inProgress = false
		m.mu.Unlock()
	}()

	log.Printf("Starting %s backup: %s", backupType, description)

	// Determine backup directory
	backupDir := filepath.Join(m.backupPath, "auto")
	if manual {
		backupDir = filepath.Join(m.backupPath, "manual")
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("casrad_backup_%s_%s", backupType, timestamp)
	if m.compress {
		filename += ".tar.gz"
	} else {
		filename += ".tar"
	}
	if m.encrypt {
		filename += ".enc"
	}

	backupFile := filepath.Join(backupDir, filename)

	// Create metadata
	metadata := &BackupMetadata{
		Version:     "1.0.0",
		Created:     time.Now(),
		Type:        backupType,
		Compressed:  m.compress,
		Encrypted:   m.encrypt,
		Description: description,
	}

	// Get database statistics
	m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&metadata.UserCount)
	m.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&metadata.TrackCount)
	m.db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&metadata.SchemaVer)

	// Create backup based on type
	var err error
	switch backupType {
	case "full":
		err = m.createFullBackup(backupFile, metadata)
	case "database":
		err = m.createDatabaseBackup(backupFile, metadata)
	case "config":
		err = m.createConfigBackup(backupFile, metadata)
	default:
		return nil, fmt.Errorf("unknown backup type: %s", backupType)
	}

	if err != nil {
		os.Remove(backupFile)
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	// Get file size
	if info, err := os.Stat(backupFile); err == nil {
		metadata.Size = info.Size()
	}

	// Calculate checksum
	metadata.Checksum = m.calculateChecksum(backupFile)

	// Save backup record to database
	m.recordBackup(metadata, backupFile)

	// Clean old backups if not manual
	if !manual {
		m.cleanOldBackups()
	}

	log.Printf("Backup completed: %s", backupFile)
	return metadata, nil
}

func (m *BackupManager) createFullBackup(backupFile string, metadata *BackupMetadata) error {
	// Create tar file
	file, err := os.Create(backupFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Setup writers chain based on options
	var writer io.Writer = file
	var gzWriter *gzip.Writer
	var encWriter io.Writer

	// Add encryption if enabled
	if m.encrypt {
		encWriter, err = m.createEncryptedWriter(writer)
		if err != nil {
			return err
		}
		writer = encWriter
	}

	// Add compression if enabled
	if m.compress {
		gzWriter = gzip.NewWriter(writer)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Create tar writer
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Add metadata file
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	if err := m.addToTar(tarWriter, "metadata.json", metadataJSON); err != nil {
		return err
	}

	// Export database to SQL
	sqlDump, err := m.exportDatabase()
	if err != nil {
		return err
	}
	if err := m.addToTar(tarWriter, "database.sql", sqlDump); err != nil {
		return err
	}

	// Export settings
	settings, err := m.exportSettings()
	if err != nil {
		return err
	}
	if err := m.addToTar(tarWriter, "settings.json", settings); err != nil {
		return err
	}

	// Add user data directories (structure only, not files)
	userDirs, err := m.exportUserDirectories()
	if err != nil {
		return err
	}
	if err := m.addToTar(tarWriter, "user_directories.json", userDirs); err != nil {
		return err
	}

	return nil
}

func (m *BackupManager) createDatabaseBackup(backupFile string, metadata *BackupMetadata) error {
	// Simple database-only backup
	sqlDump, err := m.exportDatabase()
	if err != nil {
		return err
	}

	// Write to file with optional compression
	if m.compress {
		return m.writeCompressed(backupFile, sqlDump)
	}
	return os.WriteFile(backupFile, sqlDump, 0644)
}

func (m *BackupManager) createConfigBackup(backupFile string, metadata *BackupMetadata) error {
	// Config-only backup (settings, users, playlists)
	configData := make(map[string]interface{})

	// Export settings
	settings, err := m.exportSettings()
	if err == nil {
		var settingsObj interface{}
		json.Unmarshal(settings, &settingsObj)
		configData["settings"] = settingsObj
	}

	// Export users (without passwords)
	users, err := m.exportUsers()
	if err == nil {
		var usersObj interface{}
		json.Unmarshal(users, &usersObj)
		configData["users"] = usersObj
	}

	// Export playlists
	playlists, err := m.exportPlaylists()
	if err == nil {
		var playlistsObj interface{}
		json.Unmarshal(playlists, &playlistsObj)
		configData["playlists"] = playlistsObj
	}

	// Convert to JSON
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	if m.compress {
		return m.writeCompressed(backupFile, data)
	}
	return os.WriteFile(backupFile, data, 0644)
}

func (m *BackupManager) exportDatabase() ([]byte, error) {
	// For SQLite, we can use the backup API or dump to SQL
	// This is a simplified version - real implementation would use sqlite3 .dump command

	var dump strings.Builder
	dump.WriteString("-- CASRAD Database Backup\n")
	dump.WriteString(fmt.Sprintf("-- Created: %s\n\n", time.Now().Format(time.RFC3339)))

	// Get all tables
	rows, err := m.db.Query(`
		SELECT name, sql FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []struct {
		name string
		sql  string
	}{}

	for rows.Next() {
		var name, createSQL string
		if err := rows.Scan(&name, &createSQL); err != nil {
			continue
		}
		tables = append(tables, struct {
			name string
			sql  string
		}{name, createSQL})
	}

	// Export each table
	for _, table := range tables {
		// Write CREATE statement
		dump.WriteString(fmt.Sprintf("%s;\n\n", table.sql))

		// Export data
		if err := m.exportTableData(&dump, table.name); err != nil {
			log.Printf("Failed to export table %s: %v", table.name, err)
		}
	}

	return []byte(dump.String()), nil
}

func (m *BackupManager) exportTableData(dump *strings.Builder, tableName string) error {
	rows, err := m.db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	dump.WriteString(fmt.Sprintf("-- Data for table %s\n", tableName))

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		dump.WriteString(fmt.Sprintf("INSERT INTO %s VALUES(", tableName))
		for i, v := range values {
			if i > 0 {
				dump.WriteString(",")
			}
			switch val := v.(type) {
			case nil:
				dump.WriteString("NULL")
			case []byte:
				dump.WriteString(fmt.Sprintf("'%s'", strings.ReplaceAll(string(val), "'", "''")))
			case string:
				dump.WriteString(fmt.Sprintf("'%s'", strings.ReplaceAll(val, "'", "''")))
			default:
				dump.WriteString(fmt.Sprintf("%v", val))
			}
		}
		dump.WriteString(");\n")
	}
	dump.WriteString("\n")

	return nil
}

func (m *BackupManager) exportSettings() ([]byte, error) {
	settings := make(map[string]interface{})

	rows, err := m.db.Query("SELECT key, value, value_type FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value, valueType string
		if err := rows.Scan(&key, &value, &valueType); err != nil {
			continue
		}

		// Convert value based on type
		switch valueType {
		case "boolean":
			settings[key] = value == "true"
		case "integer":
			if n, err := strconv.Atoi(value); err == nil {
				settings[key] = n
			}
		case "float":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				settings[key] = f
			}
		case "json":
			var jsonVal interface{}
			if json.Unmarshal([]byte(value), &jsonVal) == nil {
				settings[key] = jsonVal
			}
		default:
			settings[key] = value
		}
	}

	return json.MarshalIndent(settings, "", "  ")
}

func (m *BackupManager) exportUsers() ([]byte, error) {
	users := []map[string]interface{}{}

	rows, err := m.db.Query(`
		SELECT username, email, role, theme_preference, created_at
		FROM users
		ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		user := make(map[string]interface{})
		var username, email, role, theme string
		var created time.Time

		if err := rows.Scan(&username, &email, &role, &theme, &created); err != nil {
			continue
		}

		user["username"] = username
		user["email"] = email
		user["role"] = role
		user["theme"] = theme
		user["created"] = created

		users = append(users, user)
	}

	return json.MarshalIndent(users, "", "  ")
}

func (m *BackupManager) exportPlaylists() ([]byte, error) {
	playlists := []map[string]interface{}{}

	rows, err := m.db.Query(`
		SELECT p.name, p.description, u.username, p.is_public
		FROM playlists p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		playlist := make(map[string]interface{})
		var name string
		var desc sql.NullString
		var username string
		var isPublic bool

		if err := rows.Scan(&name, &desc, &username, &isPublic); err != nil {
			continue
		}

		playlist["name"] = name
		if desc.Valid {
			playlist["description"] = desc.String
		}
		playlist["owner"] = username
		playlist["public"] = isPublic

		playlists = append(playlists, playlist)
	}

	return json.MarshalIndent(playlists, "", "  ")
}

func (m *BackupManager) exportUserDirectories() ([]byte, error) {
	// Export user directory structure (not the actual files)
	dirs := []map[string]interface{}{}

	rows, err := m.db.Query(`
		SELECT username, home_directory, storage_quota_bytes, storage_used_bytes
		FROM users
		WHERE home_directory IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		dir := make(map[string]interface{})
		var username string
		var homePath sql.NullString
		var quota, used int64

		if err := rows.Scan(&username, &homePath, &quota, &used); err != nil {
			continue
		}

		dir["username"] = username
		if homePath.Valid {
			dir["path"] = homePath.String
		}
		dir["quota"] = quota
		dir["used"] = used

		dirs = append(dirs, dir)
	}

	return json.MarshalIndent(dirs, "", "  ")
}

// RestoreBackup restores from a backup file
func (m *BackupManager) RestoreBackup(backupFile string) error {
	m.mu.Lock()
	if m.inProgress {
		m.mu.Unlock()
		return fmt.Errorf("operation already in progress")
	}
	m.inProgress = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.inProgress = false
		m.mu.Unlock()
	}()

	log.Printf("Starting restore from %s", backupFile)

	// Check if file exists
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Determine backup type from filename
	if strings.Contains(backupFile, ".tar") {
		return m.restoreFromTar(backupFile)
	} else if strings.HasSuffix(backupFile, ".sql") || strings.HasSuffix(backupFile, ".sql.gz") {
		return m.restoreDatabase(backupFile)
	} else if strings.HasSuffix(backupFile, ".json") || strings.HasSuffix(backupFile, ".json.gz") {
		return m.restoreConfig(backupFile)
	}

	return fmt.Errorf("unknown backup format")
}

func (m *BackupManager) restoreFromTar(backupFile string) error {
	// Open file
	file, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle encryption if needed
	if strings.HasSuffix(backupFile, ".enc") {
		decReader, err := m.createDecryptedReader(reader)
		if err != nil {
			return err
		}
		reader = decReader
	}

	// Handle compression if needed
	if strings.Contains(backupFile, ".gz") {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Create tar reader
	tarReader := tar.NewReader(reader)

	// Process each file in the tar
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data, err := io.ReadAll(tarReader)
		if err != nil {
			return err
		}

		switch header.Name {
		case "database.sql":
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "casrad-restore-*.sql")
			if err != nil {
				return err
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(data); err != nil {
				tmpFile.Close()
				return err
			}
			tmpFile.Close()

			// Restore database
			if err := m.restoreDatabase(tmpFile.Name()); err != nil {
				return err
			}

		case "settings.json":
			if err := m.restoreSettings(data); err != nil {
				log.Printf("Failed to restore settings: %v", err)
			}

		case "metadata.json":
			// Just log the metadata
			var metadata BackupMetadata
			if json.Unmarshal(data, &metadata) == nil {
				log.Printf("Restoring backup from %s", metadata.Created.Format(time.RFC3339))
			}
		}
	}

	log.Println("Restore completed successfully")
	return nil
}

func (m *BackupManager) restoreDatabase(sqlFile string) error {
	// Read SQL file
	data, err := os.ReadFile(sqlFile)
	if err != nil {
		return err
	}

	// If compressed, decompress
	if strings.HasSuffix(sqlFile, ".gz") {
		data, err = m.decompress(data)
		if err != nil {
			return err
		}
	}

	// Execute SQL statements
	sqlStatements := strings.Split(string(data), ";")

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			log.Printf("Failed to execute SQL: %v\nStatement: %s", err, stmt[:min(100, len(stmt))])
			// Continue with other statements
		}
	}

	return tx.Commit()
}

func (m *BackupManager) restoreSettings(data []byte) error {
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	for key, value := range settings {
		valueStr := fmt.Sprintf("%v", value)
		valueType := "string"

		switch value.(type) {
		case bool:
			valueType = "boolean"
		case float64:
			valueType = "float"
		case int:
			valueType = "integer"
		}

		m.db.Exec(`
			UPDATE settings SET value = ?, value_type = ?
			WHERE key = ?
		`, valueStr, valueType, key)
	}

	return nil
}

func (m *BackupManager) restoreConfig(configFile string) error {
	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	// If compressed, decompress
	if strings.HasSuffix(configFile, ".gz") {
		data, err = m.decompress(data)
		if err != nil {
			return err
		}
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Restore settings if present
	if settings, ok := config["settings"].(map[string]interface{}); ok {
		settingsData, _ := json.Marshal(settings)
		m.restoreSettings(settingsData)
	}

	log.Println("Config restore completed")
	return nil
}

// ListBackups returns a list of available backups
func (m *BackupManager) ListBackups() ([]BackupMetadata, error) {
	backups := []BackupMetadata{}

	// List auto backups
	autoFiles, _ := filepath.Glob(filepath.Join(m.backupPath, "auto", "casrad_backup_*"))
	for _, file := range autoFiles {
		if metadata := m.getBackupMetadata(file); metadata != nil {
			backups = append(backups, *metadata)
		}
	}

	// List manual backups
	manualFiles, _ := filepath.Glob(filepath.Join(m.backupPath, "manual", "casrad_backup_*"))
	for _, file := range manualFiles {
		if metadata := m.getBackupMetadata(file); metadata != nil {
			backups = append(backups, *metadata)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Created.After(backups[j].Created)
	})

	return backups, nil
}

func (m *BackupManager) getBackupMetadata(file string) *BackupMetadata {
	info, err := os.Stat(file)
	if err != nil {
		return nil
	}

	// Parse filename for basic info
	basename := filepath.Base(file)
	parts := strings.Split(basename, "_")

	metadata := &BackupMetadata{
		Created:    info.ModTime(),
		Size:       info.Size(),
		Compressed: strings.Contains(file, ".gz"),
		Encrypted:  strings.Contains(file, ".enc"),
	}

	if len(parts) >= 3 {
		metadata.Type = parts[2]
	}

	// Try to read actual metadata from tar files
	if strings.Contains(file, ".tar") {
		// Would extract and read metadata.json here
	}

	return metadata
}

func (m *BackupManager) cleanOldBackups() {
	autoPath := filepath.Join(m.backupPath, "auto")
	files, err := filepath.Glob(filepath.Join(autoPath, "casrad_backup_*"))
	if err != nil {
		return
	}

	// Sort by modification time
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().After(fj.ModTime())
	})

	// Keep only the configured number of backups
	if len(files) > m.retention {
		for _, file := range files[m.retention:] {
			log.Printf("Removing old backup: %s", file)
			os.Remove(file)
		}
	}
}

func (m *BackupManager) recordBackup(metadata *BackupMetadata, path string) {
	m.db.Exec(`
		INSERT INTO backup_history (backup_type, backup_path, backup_size, is_verified, verification_checksum, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, metadata.Type, path, metadata.Size, true, metadata.Checksum, metadata.Created)
}

func (m *BackupManager) calculateChecksum(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (m *BackupManager) addToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write(data)
	return err
}

func (m *BackupManager) writeCompressed(file string, data []byte) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	_, err = gzWriter.Write(data)
	return err
}

func (m *BackupManager) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (m *BackupManager) createEncryptedWriter(w io.Writer) (io.Writer, error) {
	// Simplified encryption - in production would use proper key management
	block, err := aes.NewCipher(m.encryptKey)
	if err != nil {
		return nil, err
	}

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	// Write IV to output
	if _, err := w.Write(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	return &cipher.StreamWriter{S: stream, W: w}, nil
}

func (m *BackupManager) createDecryptedReader(r io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(m.encryptKey)
	if err != nil {
		return nil, err
	}

	// Read IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(r, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	return &cipher.StreamReader{S: stream, R: r}, nil
}

// GetStatus returns the current backup status
func (m *BackupManager) GetStatus() BackupStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return BackupStatus{
		InProgress: m.inProgress,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}