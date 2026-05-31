// Package store - SQLite implementation using modernc.org/sqlite (pure Go)
// Per AI.md: MUST use modernc.org/sqlite, NEVER github.com/mattn/go-sqlite3
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/casapps/casrad/src/server/model"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite (modernc.org/sqlite - pure Go)
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Enable WAL mode and foreign keys
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Migrate runs database migrations
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	// Execute schema creation
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

// schema contains the complete database schema
const schema = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER
);

CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    totp_secret VARCHAR(32),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    last_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    totp_secret VARCHAR(32),
    role VARCHAR(20) DEFAULT 'user',
    theme_preference VARCHAR(20) DEFAULT 'dark',
    home_directory TEXT,
    storage_quota_bytes BIGINT DEFAULT 53687091200,
    storage_used_bytes BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    last_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    settings TEXT,
    avatar_url TEXT,
    bio TEXT,
    website VARCHAR(255),
    location VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    admin_id INTEGER REFERENCES admins(id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    theme_name VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    last_activity TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(100),
    permissions TEXT,
    last_used TIMESTAMP,
    last_ip VARCHAR(45),
    use_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_token ON api_tokens(token);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id);

CREATE TABLE IF NOT EXISTS tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT UNIQUE NOT NULL,
    file_hash VARCHAR(64),
    user_id INTEGER REFERENCES users(id),
    is_global BOOLEAN DEFAULT FALSE,
    title VARCHAR(255),
    artist VARCHAR(255),
    album VARCHAR(255),
    album_artist VARCHAR(255),
    genre VARCHAR(100),
    year INTEGER,
    track_number INTEGER,
    disc_number INTEGER,
    duration INTEGER,
    bitrate INTEGER,
    sample_rate INTEGER,
    channels INTEGER,
    codec VARCHAR(20),
    file_type VARCHAR(10),
    file_size BIGINT,
    play_count INTEGER DEFAULT 0,
    skip_count INTEGER DEFAULT 0,
    last_played TIMESTAMP,
    rating INTEGER CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist);
CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album);
CREATE INDEX IF NOT EXISTS idx_tracks_user ON tracks(user_id);

CREATE TABLE IF NOT EXISTS albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255),
    album_artist VARCHAR(255),
    year INTEGER,
    genre VARCHAR(100),
    cover_art_path TEXT,
    total_tracks INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, album_artist)
);

CREATE TABLE IF NOT EXISTS artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) UNIQUE NOT NULL,
    sort_name VARCHAR(255),
    biography TEXT,
    image_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    track_count INTEGER DEFAULT 0,
    duration_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_playlists_user ON playlists(user_id);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (playlist_id, position)
);

CREATE TABLE IF NOT EXISTS broadcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mount_point VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) DEFAULT 'user',
    name VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(100),
    user_id INTEGER REFERENCES users(id),
    stream_key VARCHAR(64),
    bitrate INTEGER DEFAULT 128,
    format VARCHAR(20) DEFAULT 'mp3',
    channels INTEGER DEFAULT 2,
    sample_rate INTEGER DEFAULT 44100,
    is_public BOOLEAN DEFAULT TRUE,
    requires_auth BOOLEAN DEFAULT FALSE,
    max_listeners INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT FALSE,
    is_enabled BOOLEAN DEFAULT TRUE,
    listeners_current INTEGER DEFAULT 0,
    listeners_peak INTEGER DEFAULT 0,
    listeners_total BIGINT DEFAULT 0,
    bytes_sent_total BIGINT DEFAULT 0,
    current_track TEXT,
    started_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_broadcasts_mount ON broadcasts(mount_point);
CREATE INDEX IF NOT EXISTS idx_broadcasts_user ON broadcasts(user_id);

-- Podcasts - IDEA.md Data Models
CREATE TABLE IF NOT EXISTS podcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    feed_url TEXT UNIQUE NOT NULL,
    title VARCHAR(255),
    description TEXT,
    author VARCHAR(255),
    image_url TEXT,
    website VARCHAR(255),
    language VARCHAR(10),
    category VARCHAR(100),
    explicit BOOLEAN DEFAULT FALSE,
    storage_path TEXT,
    auto_download BOOLEAN DEFAULT TRUE,
    download_quality VARCHAR(20) DEFAULT 'original',
    max_episodes INTEGER DEFAULT 100,
    retention_days INTEGER DEFAULT 30,
    is_active BOOLEAN DEFAULT TRUE,
    last_check TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_podcasts_user ON podcasts(user_id);

CREATE TABLE IF NOT EXISTS podcast_episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    podcast_id INTEGER REFERENCES podcasts(id) ON DELETE CASCADE,
    guid VARCHAR(255),
    title VARCHAR(255),
    description TEXT,
    audio_url TEXT,
    website_url TEXT,
    published_at TIMESTAMP,
    duration INTEGER DEFAULT 0,
    file_size BIGINT DEFAULT 0,
    file_path TEXT,
    play_position INTEGER DEFAULT 0,
    is_played BOOLEAN DEFAULT FALSE,
    played_at TIMESTAMP,
    is_downloaded BOOLEAN DEFAULT FALSE,
    downloaded_at TIMESTAMP,
    download_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(podcast_id, guid)
);

CREATE INDEX IF NOT EXISTS idx_podcast_episodes_podcast ON podcast_episodes(podcast_id);

-- Audiobooks - IDEA.md Data Models
CREATE TABLE IF NOT EXISTS audiobooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255),
    narrator VARCHAR(255),
    series VARCHAR(255),
    series_number REAL,
    file_path TEXT,
    cover_path TEXT,
    isbn VARCHAR(20),
    publisher VARCHAR(255),
    published_date DATE,
    language VARCHAR(10),
    description TEXT,
    total_duration INTEGER DEFAULT 0,
    current_position INTEGER DEFAULT 0,
    current_chapter INTEGER DEFAULT 0,
    play_count INTEGER DEFAULT 0,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audiobooks_user ON audiobooks(user_id);

CREATE TABLE IF NOT EXISTS audiobook_chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    audiobook_id INTEGER REFERENCES audiobooks(id) ON DELETE CASCADE,
    chapter_number INTEGER NOT NULL,
    title VARCHAR(255),
    start_time INTEGER DEFAULT 0,
    end_time INTEGER DEFAULT 0,
    file_path TEXT,
    UNIQUE(audiobook_id, chapter_number)
);

-- User storage configuration - IDEA.md Business Rules
CREATE TABLE IF NOT EXISTS user_storage (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    music_paths TEXT,
    podcast_path TEXT,
    audiobook_path TEXT,
    radio_path TEXT,
    playlist_path TEXT,
    recording_path TEXT,
    transcode_path TEXT,
    quota_music_bytes BIGINT DEFAULT 21474836480,
    quota_podcast_bytes BIGINT DEFAULT 10737418240,
    quota_audiobook_bytes BIGINT DEFAULT 10737418240,
    quota_recording_bytes BIGINT DEFAULT 5368709120,
    quota_other_bytes BIGINT DEFAULT 5368709120,
    used_music_bytes BIGINT DEFAULT 0,
    used_podcast_bytes BIGINT DEFAULT 0,
    used_audiobook_bytes BIGINT DEFAULT 0,
    used_recording_bytes BIGINT DEFAULT 0,
    used_other_bytes BIGINT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Global directories - IDEA.md Data Sources
CREATE TABLE IF NOT EXISTS global_directories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL,
    path TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE,
    scan_interval_hours INTEGER DEFAULT 24,
    allow_guest_access BOOLEAN DEFAULT TRUE,
    allow_user_access BOOLEAN DEFAULT TRUE,
    last_scan TIMESTAMP,
    file_count INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_global_dirs_type ON global_directories(type);

-- Playback history - for scrobbling and statistics
CREATE TABLE IF NOT EXISTS playback_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    play_duration INTEGER DEFAULT 0,
    track_duration INTEGER DEFAULT 0,
    source VARCHAR(50),
    source_ip VARCHAR(45),
    user_agent TEXT,
    skipped BOOLEAN DEFAULT FALSE,
    skip_position INTEGER DEFAULT 0,
    playlist_id INTEGER REFERENCES playlists(id),
    broadcast_id INTEGER REFERENCES broadcasts(id)
);

CREATE INDEX IF NOT EXISTS idx_playback_user ON playback_history(user_id, started_at);
CREATE INDEX IF NOT EXISTS idx_playback_track ON playback_history(track_id);

-- Scheduled tasks - IDEA.md Business Rules
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) UNIQUE NOT NULL,
    schedule VARCHAR(50) NOT NULL,
    task_type VARCHAR(50),
    is_enabled BOOLEAN DEFAULT TRUE,
    command TEXT,
    parameters TEXT,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    last_status VARCHAR(20) DEFAULT 'pending',
    last_error TEXT,
    run_count INTEGER DEFAULT 0,
    average_duration_ms INTEGER DEFAULT 0,
    timeout_seconds INTEGER DEFAULT 3600,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    value_type VARCHAR(50),
    category VARCHAR(50),
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    event_type VARCHAR(100),
    event_category VARCHAR(50),
    ip_address VARCHAR(45),
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id, created_at);

CREATE TABLE IF NOT EXISTS setup_state (
    id INTEGER PRIMARY KEY DEFAULT 1,
    setup_completed BOOLEAN DEFAULT FALSE,
    admin_account_id INTEGER REFERENCES admins(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    CHECK (id = 1)
);

INSERT OR IGNORE INTO setup_state (id) VALUES (1);
INSERT OR IGNORE INTO schema_version (version, description) VALUES (1, 'Initial schema');
`

// Admin operations

func (s *SQLiteStore) GetAdminByID(ctx context.Context, id int64) (*model.Admin, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, is_active,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until
		FROM admins WHERE id = ?`

	admin := &model.Admin{}
	var totpSecret, lastIP sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.PasswordHash, &totpSecret,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &lastLogin, &lastIP,
		&admin.FailedLoginAttempts, &lockedUntil,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	if totpSecret.Valid {
		admin.TOTPSecret = totpSecret.String
	}
	if lastLogin.Valid {
		admin.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		admin.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		admin.LockedUntil = lockedUntil.Time
	}

	return admin, nil
}

func (s *SQLiteStore) GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, is_active,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until
		FROM admins WHERE username = ?`

	admin := &model.Admin{}
	var totpSecret, lastIP sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.PasswordHash, &totpSecret,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &lastLogin, &lastIP,
		&admin.FailedLoginAttempts, &lockedUntil,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	if totpSecret.Valid {
		admin.TOTPSecret = totpSecret.String
	}
	if lastLogin.Valid {
		admin.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		admin.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		admin.LockedUntil = lockedUntil.Time
	}

	return admin, nil
}

func (s *SQLiteStore) GetAdminByEmail(ctx context.Context, email string) (*model.Admin, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, is_active,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until
		FROM admins WHERE email = ?`

	admin := &model.Admin{}
	var totpSecret, lastIP sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.PasswordHash, &totpSecret,
		&admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &lastLogin, &lastIP,
		&admin.FailedLoginAttempts, &lockedUntil,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	if totpSecret.Valid {
		admin.TOTPSecret = totpSecret.String
	}
	if lastLogin.Valid {
		admin.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		admin.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		admin.LockedUntil = lockedUntil.Time
	}

	return admin, nil
}

func (s *SQLiteStore) CreateAdmin(ctx context.Context, admin *model.Admin) (int64, error) {
	query := `INSERT INTO admins (username, email, password_hash, totp_secret, is_active,
		created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	var totpSecret sql.NullString
	if admin.TOTPSecret != "" {
		totpSecret = sql.NullString{String: admin.TOTPSecret, Valid: true}
	}

	result, err := s.db.ExecContext(ctx, query,
		admin.Username, admin.Email, admin.PasswordHash, totpSecret,
		admin.IsActive, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create admin: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

func (s *SQLiteStore) UpdateAdmin(ctx context.Context, admin *model.Admin) error {
	query := `UPDATE admins SET username = ?, email = ?, password_hash = ?, totp_secret = ?,
		is_active = ?, updated_at = ?, last_login = ?, last_ip = ?,
		failed_login_attempts = ?, locked_until = ? WHERE id = ?`

	var totpSecret, lastIP sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	if admin.TOTPSecret != "" {
		totpSecret = sql.NullString{String: admin.TOTPSecret, Valid: true}
	}
	if !admin.LastLogin.IsZero() {
		lastLogin = sql.NullTime{Time: admin.LastLogin, Valid: true}
	}
	if admin.LastIP != "" {
		lastIP = sql.NullString{String: admin.LastIP, Valid: true}
	}
	if !admin.LockedUntil.IsZero() {
		lockedUntil = sql.NullTime{Time: admin.LockedUntil, Valid: true}
	}

	_, err := s.db.ExecContext(ctx, query,
		admin.Username, admin.Email, admin.PasswordHash, totpSecret,
		admin.IsActive, time.Now(), lastLogin, lastIP,
		admin.FailedLoginAttempts, lockedUntil, admin.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update admin: %w", err)
	}

	return nil
}

// User operations

func (s *SQLiteStore) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, role, theme_preference,
		home_directory, storage_quota_bytes, storage_used_bytes, is_active, email_verified,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until,
		settings, avatar_url, bio, website, location
		FROM users WHERE id = ?`

	user := &model.User{}
	var totpSecret, homeDir, lastIP, settings, avatarURL, bio, website, location sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &totpSecret,
		&user.Role, &user.ThemePreference, &homeDir, &user.StorageQuotaBytes,
		&user.StorageUsedBytes, &user.IsActive, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin, &lastIP,
		&user.FailedLoginAttempts, &lockedUntil, &settings, &avatarURL,
		&bio, &website, &location,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}
	if homeDir.Valid {
		user.HomeDirectory = homeDir.String
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		user.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		user.LockedUntil = lockedUntil.Time
	}
	if settings.Valid {
		user.Settings = settings.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if website.Valid {
		user.Website = website.String
	}
	if location.Valid {
		user.Location = location.String
	}

	return user, nil
}

func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, role, theme_preference,
		home_directory, storage_quota_bytes, storage_used_bytes, is_active, email_verified,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until,
		settings, avatar_url, bio, website, location
		FROM users WHERE username = ?`

	user := &model.User{}
	var totpSecret, homeDir, lastIP, settings, avatarURL, bio, website, location sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &totpSecret,
		&user.Role, &user.ThemePreference, &homeDir, &user.StorageQuotaBytes,
		&user.StorageUsedBytes, &user.IsActive, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin, &lastIP,
		&user.FailedLoginAttempts, &lockedUntil, &settings, &avatarURL,
		&bio, &website, &location,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}
	if homeDir.Valid {
		user.HomeDirectory = homeDir.String
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		user.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		user.LockedUntil = lockedUntil.Time
	}
	if settings.Valid {
		user.Settings = settings.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if website.Valid {
		user.Website = website.String
	}
	if location.Valid {
		user.Location = location.String
	}

	return user, nil
}

func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, username, email, password_hash, totp_secret, role, theme_preference,
		home_directory, storage_quota_bytes, storage_used_bytes, is_active, email_verified,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until,
		settings, avatar_url, bio, website, location
		FROM users WHERE email = ?`

	user := &model.User{}
	var totpSecret, homeDir, lastIP, settings, avatarURL, bio, website, location sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &totpSecret,
		&user.Role, &user.ThemePreference, &homeDir, &user.StorageQuotaBytes,
		&user.StorageUsedBytes, &user.IsActive, &user.EmailVerified,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin, &lastIP,
		&user.FailedLoginAttempts, &lockedUntil, &settings, &avatarURL,
		&bio, &website, &location,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}
	if homeDir.Valid {
		user.HomeDirectory = homeDir.String
	}
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	if lastIP.Valid {
		user.LastIP = lastIP.String
	}
	if lockedUntil.Valid {
		user.LockedUntil = lockedUntil.Time
	}
	if settings.Valid {
		user.Settings = settings.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if website.Valid {
		user.Website = website.String
	}
	if location.Valid {
		user.Location = location.String
	}

	return user, nil
}

func (s *SQLiteStore) CreateUser(ctx context.Context, user *model.User) (int64, error) {
	query := `INSERT INTO users (username, email, password_hash, totp_secret, role,
		theme_preference, home_directory, storage_quota_bytes, storage_used_bytes,
		is_active, email_verified, created_at, updated_at, settings, avatar_url,
		bio, website, location) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	var totpSecret, homeDir, settings, avatarURL, bio, website, location sql.NullString

	if user.TOTPSecret != "" {
		totpSecret = sql.NullString{String: user.TOTPSecret, Valid: true}
	}
	if user.HomeDirectory != "" {
		homeDir = sql.NullString{String: user.HomeDirectory, Valid: true}
	}
	if user.Settings != "" {
		settings = sql.NullString{String: user.Settings, Valid: true}
	}
	if user.AvatarURL != "" {
		avatarURL = sql.NullString{String: user.AvatarURL, Valid: true}
	}
	if user.Bio != "" {
		bio = sql.NullString{String: user.Bio, Valid: true}
	}
	if user.Website != "" {
		website = sql.NullString{String: user.Website, Valid: true}
	}
	if user.Location != "" {
		location = sql.NullString{String: user.Location, Valid: true}
	}

	// Set defaults
	if user.Role == "" {
		user.Role = "user"
	}
	if user.ThemePreference == "" {
		user.ThemePreference = "dark"
	}
	if user.StorageQuotaBytes == 0 {
		// 50GB default
		user.StorageQuotaBytes = 53687091200
	}

	result, err := s.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, totpSecret, user.Role,
		user.ThemePreference, homeDir, user.StorageQuotaBytes, user.StorageUsedBytes,
		user.IsActive, user.EmailVerified, now, now, settings, avatarURL,
		bio, website, location,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

func (s *SQLiteStore) UpdateUser(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET username = ?, email = ?, password_hash = ?, totp_secret = ?,
		role = ?, theme_preference = ?, home_directory = ?, storage_quota_bytes = ?,
		storage_used_bytes = ?, is_active = ?, email_verified = ?, updated_at = ?,
		last_login = ?, last_ip = ?, failed_login_attempts = ?, locked_until = ?,
		settings = ?, avatar_url = ?, bio = ?, website = ?, location = ?
		WHERE id = ?`

	var totpSecret, homeDir, lastIP, settings, avatarURL, bio, website, location sql.NullString
	var lastLogin, lockedUntil sql.NullTime

	if user.TOTPSecret != "" {
		totpSecret = sql.NullString{String: user.TOTPSecret, Valid: true}
	}
	if user.HomeDirectory != "" {
		homeDir = sql.NullString{String: user.HomeDirectory, Valid: true}
	}
	if !user.LastLogin.IsZero() {
		lastLogin = sql.NullTime{Time: user.LastLogin, Valid: true}
	}
	if user.LastIP != "" {
		lastIP = sql.NullString{String: user.LastIP, Valid: true}
	}
	if !user.LockedUntil.IsZero() {
		lockedUntil = sql.NullTime{Time: user.LockedUntil, Valid: true}
	}
	if user.Settings != "" {
		settings = sql.NullString{String: user.Settings, Valid: true}
	}
	if user.AvatarURL != "" {
		avatarURL = sql.NullString{String: user.AvatarURL, Valid: true}
	}
	if user.Bio != "" {
		bio = sql.NullString{String: user.Bio, Valid: true}
	}
	if user.Website != "" {
		website = sql.NullString{String: user.Website, Valid: true}
	}
	if user.Location != "" {
		location = sql.NullString{String: user.Location, Valid: true}
	}

	_, err := s.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, totpSecret,
		user.Role, user.ThemePreference, homeDir, user.StorageQuotaBytes,
		user.StorageUsedBytes, user.IsActive, user.EmailVerified, time.Now(),
		lastLogin, lastIP, user.FailedLoginAttempts, lockedUntil,
		settings, avatarURL, bio, website, location, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (s *SQLiteStore) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	// Get total count
	var total int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	query := `SELECT id, username, email, password_hash, totp_secret, role, theme_preference,
		home_directory, storage_quota_bytes, storage_used_bytes, is_active, email_verified,
		created_at, updated_at, last_login, last_ip, failed_login_attempts, locked_until,
		settings, avatar_url, bio, website, location
		FROM users ORDER BY id LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		var totpSecret, homeDir, lastIP, settings, avatarURL, bio, website, location sql.NullString
		var lastLogin, lockedUntil sql.NullTime

		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &totpSecret,
			&user.Role, &user.ThemePreference, &homeDir, &user.StorageQuotaBytes,
			&user.StorageUsedBytes, &user.IsActive, &user.EmailVerified,
			&user.CreatedAt, &user.UpdatedAt, &lastLogin, &lastIP,
			&user.FailedLoginAttempts, &lockedUntil, &settings, &avatarURL,
			&bio, &website, &location,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		if totpSecret.Valid {
			user.TOTPSecret = totpSecret.String
		}
		if homeDir.Valid {
			user.HomeDirectory = homeDir.String
		}
		if lastLogin.Valid {
			user.LastLogin = lastLogin.Time
		}
		if lastIP.Valid {
			user.LastIP = lastIP.String
		}
		if lockedUntil.Valid {
			user.LockedUntil = lockedUntil.Time
		}
		if settings.Valid {
			user.Settings = settings.String
		}
		if avatarURL.Valid {
			user.AvatarURL = avatarURL.String
		}
		if bio.Valid {
			user.Bio = bio.String
		}
		if website.Valid {
			user.Website = website.String
		}
		if location.Valid {
			user.Location = location.String
		}

		users = append(users, user)
	}

	return users, total, nil
}

// Session operations

func (s *SQLiteStore) GetSession(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT id, user_id, admin_id, ip_address, user_agent, theme_name,
		created_at, expires_at, last_activity, is_active
		FROM sessions WHERE id = ?`

	session := &model.Session{}
	var userID, adminID sql.NullInt64
	var ipAddress, userAgent, themeName sql.NullString
	var expiresAt, lastActivity sql.NullTime

	// Hash the raw session ID before DB lookup — never store raw tokens
	err := s.db.QueryRowContext(ctx, query, hashForStorage(id)).Scan(
		&session.ID, &userID, &adminID, &ipAddress, &userAgent, &themeName,
		&session.CreatedAt, &expiresAt, &lastActivity, &session.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if userID.Valid {
		session.UserID = userID.Int64
	}
	if adminID.Valid {
		session.AdminID = adminID.Int64
	}
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if themeName.Valid {
		session.ThemeName = themeName.String
	}
	if expiresAt.Valid {
		session.ExpiresAt = expiresAt.Time
	}
	if lastActivity.Valid {
		session.LastActivity = lastActivity.Time
	}

	return session, nil
}

func (s *SQLiteStore) CreateSession(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO sessions (id, user_id, admin_id, ip_address, user_agent, theme_name,
		created_at, expires_at, last_activity, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var userID, adminID sql.NullInt64
	var ipAddress, userAgent, themeName sql.NullString

	if session.UserID != 0 {
		userID = sql.NullInt64{Int64: session.UserID, Valid: true}
	}
	if session.AdminID != 0 {
		adminID = sql.NullInt64{Int64: session.AdminID, Valid: true}
	}
	if session.IPAddress != "" {
		ipAddress = sql.NullString{String: session.IPAddress, Valid: true}
	}
	if session.UserAgent != "" {
		userAgent = sql.NullString{String: session.UserAgent, Valid: true}
	}
	if session.ThemeName != "" {
		themeName = sql.NullString{String: session.ThemeName, Valid: true}
	}

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastActivity.IsZero() {
		session.LastActivity = now
	}

	// Hash the raw session ID before writing to DB — never store raw tokens
	_, err := s.db.ExecContext(ctx, query,
		hashForStorage(session.ID), userID, adminID, ipAddress, userAgent, themeName,
		session.CreatedAt, session.ExpiresAt, session.LastActivity, session.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (s *SQLiteStore) UpdateSession(ctx context.Context, session *model.Session) error {
	query := `UPDATE sessions SET last_activity = ?, is_active = ?, expires_at = ?
		WHERE id = ?`

	// Hash the raw session ID before DB update — never store raw tokens
	_, err := s.db.ExecContext(ctx, query,
		time.Now(), session.IsActive, session.ExpiresAt, hashForStorage(session.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	// Hash the raw session ID before DB delete — never store raw tokens
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", hashForStorage(id))
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// Token operations

func (s *SQLiteStore) GetToken(ctx context.Context, token string) (*model.APIToken, error) {
	query := `SELECT id, user_id, token, name, permissions, last_used, last_ip,
		use_count, expires_at, is_active, created_at
		FROM api_tokens WHERE token = ?`

	apiToken := &model.APIToken{}
	var name, permissions, lastIP sql.NullString
	var lastUsed, expiresAt sql.NullTime

	// Hash the raw token before DB lookup — never store raw tokens
	err := s.db.QueryRowContext(ctx, query, hashForStorage(token)).Scan(
		&apiToken.ID, &apiToken.UserID, &apiToken.Token, &name, &permissions,
		&lastUsed, &lastIP, &apiToken.UseCount, &expiresAt, &apiToken.IsActive,
		&apiToken.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if name.Valid {
		apiToken.Name = name.String
	}
	if permissions.Valid {
		apiToken.Permissions = permissions.String
	}
	if lastUsed.Valid {
		apiToken.LastUsed = lastUsed.Time
	}
	if lastIP.Valid {
		apiToken.LastIP = lastIP.String
	}
	if expiresAt.Valid {
		apiToken.ExpiresAt = expiresAt.Time
	}

	return apiToken, nil
}

func (s *SQLiteStore) GetTokenByID(ctx context.Context, id int64) (*model.APIToken, error) {
	query := `SELECT id, user_id, token, name, permissions, last_used, last_ip,
		use_count, expires_at, is_active, created_at
		FROM api_tokens WHERE id = ?`

	apiToken := &model.APIToken{}
	var name, permissions, lastIP sql.NullString
	var lastUsed, expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&apiToken.ID, &apiToken.UserID, &apiToken.Token, &name, &permissions,
		&lastUsed, &lastIP, &apiToken.UseCount, &expiresAt, &apiToken.IsActive,
		&apiToken.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if name.Valid {
		apiToken.Name = name.String
	}
	if permissions.Valid {
		apiToken.Permissions = permissions.String
	}
	if lastUsed.Valid {
		apiToken.LastUsed = lastUsed.Time
	}
	if lastIP.Valid {
		apiToken.LastIP = lastIP.String
	}
	if expiresAt.Valid {
		apiToken.ExpiresAt = expiresAt.Time
	}

	return apiToken, nil
}

func (s *SQLiteStore) CreateToken(ctx context.Context, token *model.APIToken) (int64, error) {
	query := `INSERT INTO api_tokens (user_id, token, name, permissions, expires_at,
		is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

	var name, permissions sql.NullString
	var expiresAt sql.NullTime

	if token.Name != "" {
		name = sql.NullString{String: token.Name, Valid: true}
	}
	if token.Permissions != "" {
		permissions = sql.NullString{String: token.Permissions, Valid: true}
	}
	if !token.ExpiresAt.IsZero() {
		expiresAt = sql.NullTime{Time: token.ExpiresAt, Valid: true}
	}

	// Hash the raw token before writing to DB — never store raw tokens
	result, err := s.db.ExecContext(ctx, query,
		token.UserID, hashForStorage(token.Token), name, permissions, expiresAt,
		token.IsActive, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

func (s *SQLiteStore) DeleteToken(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM api_tokens WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListUserTokens(ctx context.Context, userID int64) ([]*model.APIToken, error) {
	query := `SELECT id, user_id, token, name, permissions, last_used, last_ip,
		use_count, expires_at, is_active, created_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*model.APIToken
	for rows.Next() {
		apiToken := &model.APIToken{}
		var name, permissions, lastIP sql.NullString
		var lastUsed, expiresAt sql.NullTime

		err := rows.Scan(
			&apiToken.ID, &apiToken.UserID, &apiToken.Token, &name, &permissions,
			&lastUsed, &lastIP, &apiToken.UseCount, &expiresAt, &apiToken.IsActive,
			&apiToken.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}

		if name.Valid {
			apiToken.Name = name.String
		}
		if permissions.Valid {
			apiToken.Permissions = permissions.String
		}
		if lastUsed.Valid {
			apiToken.LastUsed = lastUsed.Time
		}
		if lastIP.Valid {
			apiToken.LastIP = lastIP.String
		}
		if expiresAt.Valid {
			apiToken.ExpiresAt = expiresAt.Time
		}

		tokens = append(tokens, apiToken)
	}

	return tokens, nil
}
