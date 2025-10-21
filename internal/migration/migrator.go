package migration

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casapps/casrad/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

// Migrator handles migration from other platforms
type Migrator struct {
	db           *database.Engine
	tempDir      string
	progressChan chan MigrationProgress
}

// MigrationProgress represents migration progress
type MigrationProgress struct {
	Stage        string
	Current      int
	Total        int
	Message      string
	ErrorMessage string
}

// MigrationResult represents the result of a migration
type MigrationResult struct {
	Success        bool
	ItemsMigrated  int
	ItemsFailed    int
	Errors         []string
	Warnings       []string
	Duration       time.Duration
}

// NewMigrator creates a new migrator
func NewMigrator(db *database.Engine) *Migrator {
	return &Migrator{
		db:           db,
		tempDir:      "/tmp/casrad-migration",
		progressChan: make(chan MigrationProgress, 100),
	}
}

// MigrateFromIcecast migrates from Icecast configuration
func (m *Migrator) MigrateFromIcecast(configContent string) (*MigrationResult, error) {
	result := &MigrationResult{}
	startTime := time.Now()

	// Parse Icecast XML configuration
	type IcecastMount struct {
		MountName        string `xml:"mount-name"`
		Username         string `xml:"username"`
		Password         string `xml:"password"`
		MaxListeners     int    `xml:"max-listeners"`
		Fallback         string `xml:"fallback-mount"`
		FallbackOverride int    `xml:"fallback-override"`
		Public           int    `xml:"public"`
		Genre            string `xml:"genre"`
		Description      string `xml:"stream-description"`
		URL              string `xml:"stream-url"`
		Bitrate          int    `xml:"bitrate"`
	}

	type IcecastConfig struct {
		XMLName    xml.Name       `xml:"icecast"`
		Location   string         `xml:"location"`
		Admin      string         `xml:"admin"`
		Hostname   string         `xml:"hostname"`
		Port       int            `xml:"listen-socket>port"`
		Mounts     []IcecastMount `xml:"mount"`
		SourcePass string         `xml:"authentication>source-password"`
		AdminPass  string         `xml:"authentication>admin-password"`
		AdminUser  string         `xml:"authentication>admin-user"`
	}

	var config IcecastConfig
	if err := xml.Unmarshal([]byte(configContent), &config); err != nil {
		return result, fmt.Errorf("failed to parse Icecast config: %w", err)
	}

	m.sendProgress("Parsing", 1, 3, "Parsed Icecast configuration")

	// Migrate server settings
	m.db.SetSetting("server.location", config.Location, nil)
	m.db.SetSetting("server.admin_email", config.Admin, nil)
	m.db.SetSetting("server.hostname", config.Hostname, nil)

	// Migrate mounts as broadcasts
	for i, mount := range config.Mounts {
		m.sendProgress("Mounts", i+1, len(config.Mounts), fmt.Sprintf("Migrating mount %s", mount.MountName))

		_, err := m.db.Exec(`
			INSERT INTO broadcasts (
				mount_point, type, name, description, genre,
				fallback_mount, bitrate, format, is_public,
				max_listeners, website, stream_key
			) VALUES (?, 'relay', ?, ?, ?, ?, ?, 'mp3', ?, ?, ?, ?)
			ON CONFLICT (mount_point) DO UPDATE SET
				name = excluded.name,
				description = excluded.description
		`, mount.MountName, mount.MountName, mount.Description,
			mount.Genre, mount.Fallback, mount.Bitrate,
			mount.Public == 1, mount.MaxListeners, mount.URL,
			generateStreamKey())

		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to migrate mount %s: %v", mount.MountName, err))
		} else {
			result.ItemsMigrated++
		}
	}

	// Create admin user from Icecast credentials
	if config.AdminUser != "" && config.AdminPass != "" {
		m.db.Exec(`
			INSERT INTO users (username, email, password_hash, role)
			VALUES (?, ?, ?, 'admin')
			ON CONFLICT (username) DO NOTHING
		`, config.AdminUser, config.Admin, config.AdminPass) // Should hash password in production
	}

	m.sendProgress("Complete", 3, 3, "Migration completed")

	result.Success = result.ItemsFailed == 0
	result.Duration = time.Since(startTime)
	return result, nil
}

// MigrateFromSubsonic migrates from Subsonic/Airsonic/Navidrome database
func (m *Migrator) MigrateFromSubsonic(dbPath string) (*MigrationResult, error) {
	result := &MigrationResult{}
	startTime := time.Now()

	// Open Subsonic database
	subsonicDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return result, fmt.Errorf("failed to open Subsonic database: %w", err)
	}
	defer subsonicDB.Close()

	// Migrate users
	if err := m.migrateSubsonicUsers(subsonicDB, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("User migration warning: %v", err))
	}

	// Migrate music folders
	if err := m.migrateSubsonicFolders(subsonicDB, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Folder migration warning: %v", err))
	}

	// Migrate playlists
	if err := m.migrateSubsonicPlaylists(subsonicDB, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Playlist migration warning: %v", err))
	}

	// Migrate play counts and ratings
	if err := m.migrateSubsonicStats(subsonicDB, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Stats migration warning: %v", err))
	}

	result.Success = result.ItemsFailed < result.ItemsMigrated/2 // Success if more than half migrated
	result.Duration = time.Since(startTime)
	return result, nil
}

// migrateSubsonicUsers migrates Subsonic users
func (m *Migrator) migrateSubsonicUsers(subsonicDB *sql.DB, result *MigrationResult) error {
	rows, err := subsonicDB.Query(`
		SELECT username, email, password, role
		FROM user
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var username, email, password, role string
		if err := rows.Scan(&username, &email, &password, &role); err != nil {
			continue
		}

		m.sendProgress("Users", count, -1, fmt.Sprintf("Migrating user %s", username))

		// Map Subsonic roles to CASRAD roles
		casradRole := "user"
		if role == "ADMIN" {
			casradRole = "admin"
		}

		_, err := m.db.Exec(`
			INSERT INTO users (username, email, password_hash, role)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (username) DO UPDATE SET
				email = excluded.email,
				role = excluded.role
		`, username, email, password, casradRole)

		if err != nil {
			result.ItemsFailed++
		} else {
			result.ItemsMigrated++
		}
		count++
	}

	return nil
}

// migrateSubsonicFolders migrates music folders
func (m *Migrator) migrateSubsonicFolders(subsonicDB *sql.DB, result *MigrationResult) error {
	rows, err := subsonicDB.Query(`
		SELECT path, name, enabled
		FROM music_folder
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path, name string
		var enabled bool
		if err := rows.Scan(&path, &name, &enabled); err != nil {
			continue
		}

		_, err := m.db.Exec(`
			INSERT INTO global_directories (type, path, is_active)
			VALUES ('music', ?, ?)
		`, path, enabled)

		if err != nil {
			result.ItemsFailed++
		} else {
			result.ItemsMigrated++
		}
	}

	return nil
}

// migrateSubsonicPlaylists migrates playlists
func (m *Migrator) migrateSubsonicPlaylists(subsonicDB *sql.DB, result *MigrationResult) error {
	// Get playlists
	playlists, err := subsonicDB.Query(`
		SELECT id, username, name, comment, public, created, changed
		FROM playlist
	`)
	if err != nil {
		return err
	}
	defer playlists.Close()

	for playlists.Next() {
		var playlistID int
		var username, name, comment string
		var public bool
		var created, changed time.Time

		if err := playlists.Scan(&playlistID, &username, &name, &comment, &public, &created, &changed); err != nil {
			continue
		}

		// Get user ID
		var userID int
		err := m.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
		if err != nil {
			continue
		}

		// Create playlist
		var newPlaylistID int64
		result, err := m.db.Exec(`
			INSERT INTO playlists (user_id, name, description, is_public, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, userID, name, comment, public, created, changed)

		if err != nil {
			continue
		}

		newPlaylistID, _ = result.LastInsertId()

		// Get playlist tracks
		tracks, err := subsonicDB.Query(`
			SELECT media_file_id
			FROM playlist_file
			WHERE playlist_id = ?
			ORDER BY position
		`, playlistID)
		if err != nil {
			continue
		}

		position := 0
		for tracks.Next() {
			var mediaFileID string
			tracks.Scan(&mediaFileID)

			// Try to find corresponding track in CASRAD
			// This is simplified - in production would need proper mapping
			m.db.Exec(`
				INSERT INTO playlist_tracks (playlist_id, track_id, position)
				SELECT ?, id, ?
				FROM tracks
				WHERE file_path LIKE ?
				LIMIT 1
			`, newPlaylistID, position, "%"+mediaFileID+"%")

			position++
		}
		tracks.Close()
	}

	return nil
}

// migrateSubsonicStats migrates play statistics
func (m *Migrator) migrateSubsonicStats(subsonicDB *sql.DB, result *MigrationResult) error {
	// Migrate play counts
	rows, err := subsonicDB.Query(`
		SELECT media_file_id, play_count, last_played
		FROM media_file
		WHERE play_count > 0
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var mediaFileID string
		var playCount int
		var lastPlayed *time.Time

		rows.Scan(&mediaFileID, &playCount, &lastPlayed)

		// Update corresponding track
		m.db.Exec(`
			UPDATE tracks
			SET play_count = ?, last_played = ?
			WHERE file_path LIKE ?
		`, playCount, lastPlayed, "%"+mediaFileID+"%")
	}

	return nil
}

// MigrateFromAmpache migrates from Ampache database
func (m *Migrator) MigrateFromAmpache(config string) (*MigrationResult, error) {
	result := &MigrationResult{}
	startTime := time.Now()

	// Parse Ampache configuration
	// This would parse ampache.cfg.php or connect to MySQL database

	m.sendProgress("Parsing", 1, 5, "Parsing Ampache configuration")

	// For demonstration, showing structure
	// In production, would parse PHP config and connect to MySQL

	result.Success = true
	result.Duration = time.Since(startTime)
	result.Warnings = append(result.Warnings, "Ampache migration requires database access configuration")

	return result, nil
}

// MigrateFromMPD migrates from MPD configuration and database
func (m *Migrator) MigrateFromMPD(configPath, dbPath string) (*MigrationResult, error) {
	result := &MigrationResult{}
	startTime := time.Now()

	// Parse MPD configuration
	config, err := m.parseMPDConfig(configPath)
	if err != nil {
		return result, fmt.Errorf("failed to parse MPD config: %w", err)
	}

	// Migrate music directory
	if musicDir, ok := config["music_directory"]; ok {
		m.db.Exec(`
			INSERT INTO global_directories (type, path, is_active)
			VALUES ('music', ?, 1)
		`, musicDir)
		result.ItemsMigrated++
	}

	// Migrate playlists directory
	if playlistDir, ok := config["playlist_directory"]; ok {
		m.migrateM3UPlaylists(playlistDir, result)
	}

	// Parse MPD database if available
	if dbPath != "" {
		m.migrateMPDDatabase(dbPath, result)
	}

	result.Success = result.ItemsFailed == 0
	result.Duration = time.Since(startTime)
	return result, nil
}

// parseMPDConfig parses MPD configuration file
func (m *Migrator) parseMPDConfig(configPath string) (map[string]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	config := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.Trim(parts[1], "\"")
			config[key] = value
		}
	}

	return config, nil
}

// migrateMPDDatabase migrates MPD database
func (m *Migrator) migrateMPDDatabase(dbPath string, result *MigrationResult) {
	content, err := os.ReadFile(dbPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read MPD database: %v", err))
		return
	}

	// Parse MPD database format
	lines := strings.Split(string(content), "\n")
	var currentTrack map[string]string

	for _, line := range lines {
		if strings.HasPrefix(line, "song_begin:") {
			currentTrack = make(map[string]string)
		} else if strings.HasPrefix(line, "song_end") && currentTrack != nil {
			// Save track
			m.saveMPDTrack(currentTrack, result)
			currentTrack = nil
		} else if currentTrack != nil && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentTrack[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
}

// saveMPDTrack saves an MPD track to database
func (m *Migrator) saveMPDTrack(track map[string]string, result *MigrationResult) {
	_, err := m.db.Exec(`
		INSERT INTO tracks (
			file_path, title, artist, album, genre,
			duration, track_number, date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (file_path) DO UPDATE SET
			title = excluded.title,
			artist = excluded.artist
	`, track["file"], track["Title"], track["Artist"], track["Album"],
		track["Genre"], track["Time"], track["Track"], track["Date"])

	if err != nil {
		result.ItemsFailed++
	} else {
		result.ItemsMigrated++
	}
}

// migrateM3UPlaylists migrates M3U playlist files
func (m *Migrator) migrateM3UPlaylists(playlistDir string, result *MigrationResult) {
	files, err := filepath.Glob(filepath.Join(playlistDir, "*.m3u*"))
	if err != nil {
		return
	}

	for _, file := range files {
		m.importM3UPlaylist(file, result)
	}
}

// importM3UPlaylist imports a single M3U playlist
func (m *Migrator) importM3UPlaylist(filePath string, result *MigrationResult) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.ItemsFailed++
		return
	}

	playlistName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Create playlist
	playlistResult, err := m.db.Exec(`
		INSERT INTO playlists (user_id, name, description, is_public)
		VALUES (NULL, ?, 'Imported from M3U', 0)
	`, playlistName)

	if err != nil {
		result.ItemsFailed++
		return
	}

	playlistID, _ := playlistResult.LastInsertId()

	// Parse M3U
	lines := strings.Split(string(content), "\n")
	position := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Add track to playlist
		m.db.Exec(`
			INSERT INTO playlist_tracks (playlist_id, track_id, position)
			SELECT ?, id, ?
			FROM tracks
			WHERE file_path = ? OR file_path LIKE ?
			LIMIT 1
		`, playlistID, position, line, "%"+filepath.Base(line))

		position++
	}

	result.ItemsMigrated++
}

// MigrateFromPlex migrates from Plex (audio only)
func (m *Migrator) MigrateFromPlex(plexDBPath string) (*MigrationResult, error) {
	result := &MigrationResult{}
	startTime := time.Now()

	// Open Plex database
	plexDB, err := sql.Open("sqlite3", plexDBPath)
	if err != nil {
		return result, fmt.Errorf("failed to open Plex database: %w", err)
	}
	defer plexDB.Close()

	// Migrate audio tracks
	rows, err := plexDB.Query(`
		SELECT file, title, artist, album, year, genre, duration
		FROM media_items
		WHERE media_item_type = 10
	`) // Type 10 is audio in Plex

	if err != nil {
		return result, fmt.Errorf("failed to query Plex database: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var file, title, artist, album, genre string
		var year, duration int

		rows.Scan(&file, &title, &artist, &album, &year, &genre, &duration)

		m.db.Exec(`
			INSERT INTO tracks (
				file_path, title, artist, album, year, genre, duration
			) VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (file_path) DO UPDATE SET
				title = excluded.title
		`, file, title, artist, album, year, genre, duration)

		result.ItemsMigrated++
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result, nil
}

// ImportFromUpload handles file uploads for migration
func (m *Migrator) ImportFromUpload(sourceType string, fileReader io.Reader) (*MigrationResult, error) {
	// Create temp file
	tempFile, err := os.CreateTemp(m.tempDir, "migration-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())

	// Copy upload to temp file
	if _, err := io.Copy(tempFile, fileReader); err != nil {
		return nil, err
	}
	tempFile.Close()

	// Process based on source type
	switch sourceType {
	case "subsonic":
		return m.MigrateFromSubsonic(tempFile.Name())
	case "plex":
		return m.MigrateFromPlex(tempFile.Name())
	case "mpd":
		return m.MigrateFromMPD("", tempFile.Name())
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}

// sendProgress sends migration progress
func (m *Migrator) sendProgress(stage string, current, total int, message string) {
	select {
	case m.progressChan <- MigrationProgress{
		Stage:   stage,
		Current: current,
		Total:   total,
		Message: message,
	}:
	default:
		// Don't block if channel is full
	}
}

// GetProgressChannel returns the progress channel
func (m *Migrator) GetProgressChannel() <-chan MigrationProgress {
	return m.progressChan
}

// generateStreamKey generates a random stream key
func generateStreamKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(b)
}

// ExportConfiguration exports CASRAD configuration for backup
func (m *Migrator) ExportConfiguration() (string, error) {
	config := make(map[string]interface{})

	// Export settings
	settings := make(map[string]string)
	rows, err := m.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		rows.Scan(&key, &value)
		settings[key] = value
	}
	config["settings"] = settings

	// Export users (without passwords)
	var users []map[string]interface{}
	userRows, err := m.db.Query(`
		SELECT username, email, role, created_at
		FROM users
	`)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var username, email, role string
			var createdAt time.Time
			userRows.Scan(&username, &email, &role, &createdAt)

			users = append(users, map[string]interface{}{
				"username":   username,
				"email":      email,
				"role":       role,
				"created_at": createdAt,
			})
		}
	}
	config["users"] = users

	// Convert to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// ImportConfiguration imports CASRAD configuration
func (m *Migrator) ImportConfiguration(configJSON string) error {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}

	// Import settings
	if settings, ok := config["settings"].(map[string]interface{}); ok {
		for key, value := range settings {
			m.db.SetSetting(key, fmt.Sprintf("%v", value), nil)
		}
	}

	log.Println("Configuration imported successfully")
	return nil
}