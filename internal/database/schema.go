package database

// Schema contains the complete CASRAD database schema
const Schema = `
-- Schema version tracking
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER
);

-- Insert initial version
INSERT INTO schema_version (version, description) VALUES
(1, 'Initial CASRAD schema');

-- Core user system with storage paths
CREATE TABLE users (
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
    location VARCHAR(100),

    CONSTRAINT email_format CHECK (email LIKE '%_@_%._%')
);

-- User storage configuration
CREATE TABLE user_storage (
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

-- Global media directories (multiple music dirs supported)
CREATE TABLE global_directories (
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

-- Default global directories
INSERT INTO global_directories (type, path) VALUES
('music', '/mnt/Music/Mp3'),
('podcast', '/mnt/Podcasts'),
('audiobook', '/mnt/Audiobooks'),
('playlist', '/mnt/Playlists');

-- Theme system
CREATE TABLE themes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,

    color_background VARCHAR(7) DEFAULT '#282a36',
    color_current_line VARCHAR(7) DEFAULT '#44475a',
    color_foreground VARCHAR(7) DEFAULT '#f8f8f2',
    color_comment VARCHAR(7) DEFAULT '#6272a4',
    color_cyan VARCHAR(7) DEFAULT '#8be9fd',
    color_green VARCHAR(7) DEFAULT '#50fa7b',
    color_orange VARCHAR(7) DEFAULT '#ffb86c',
    color_pink VARCHAR(7) DEFAULT '#ff79c6',
    color_purple VARCHAR(7) DEFAULT '#bd93f9',
    color_red VARCHAR(7) DEFAULT '#ff5555',
    color_yellow VARCHAR(7) DEFAULT '#f1fa8c',

    color_selection VARCHAR(7) DEFAULT '#44475a',
    color_border VARCHAR(7) DEFAULT '#6272a4',
    color_shadow VARCHAR(20) DEFAULT 'rgba(0,0,0,0.3)',
    color_hover VARCHAR(7) DEFAULT '#50fa7b',
    color_active VARCHAR(7) DEFAULT '#bd93f9',
    color_success VARCHAR(7) DEFAULT '#50fa7b',
    color_warning VARCHAR(7) DEFAULT '#f1fa8c',
    color_error VARCHAR(7) DEFAULT '#ff5555',
    color_info VARCHAR(7) DEFAULT '#8be9fd',

    font_family_primary VARCHAR(100) DEFAULT 'Inter, system-ui, sans-serif',
    font_family_mono VARCHAR(100) DEFAULT 'JetBrains Mono, monospace',
    font_size_base VARCHAR(10) DEFAULT '16px',
    font_weight_normal VARCHAR(10) DEFAULT '400',
    font_weight_bold VARCHAR(10) DEFAULT '600',

    spacing_unit VARCHAR(10) DEFAULT '8px',
    border_radius VARCHAR(10) DEFAULT '6px',
    transition_speed VARCHAR(10) DEFAULT '200ms',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default themes
INSERT INTO themes (name, display_name, description, is_default) VALUES
('dark', 'Dracula Dark', 'Beautiful dark theme based on Dracula', TRUE),
('light', 'Clean Light', 'Clean and modern light theme', FALSE);

-- Light theme colors
UPDATE themes SET
    color_background = '#ffffff',
    color_current_line = '#f5f5f5',
    color_foreground = '#2e3440',
    color_comment = '#6c757d',
    color_cyan = '#0969da',
    color_green = '#1a7f37',
    color_orange = '#fb8500',
    color_pink = '#bf3989',
    color_purple = '#8250df',
    color_red = '#cf222e',
    color_yellow = '#d4a72c',
    color_selection = '#e1e4e8',
    color_border = '#d0d7de',
    color_shadow = 'rgba(0,0,0,0.1)',
    color_hover = '#0969da',
    color_active = '#8250df'
WHERE name = 'light';

-- User theme customization
CREATE TABLE user_themes (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme_id INTEGER REFERENCES themes(id),

    custom_css TEXT,
    color_background VARCHAR(7),
    color_foreground VARCHAR(7),
    color_primary VARCHAR(7),
    color_secondary VARCHAR(7),

    font_size VARCHAR(10) DEFAULT 'medium',
    high_contrast BOOLEAN DEFAULT FALSE,
    reduce_motion BOOLEAN DEFAULT FALSE,

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Session management with theme
CREATE TABLE sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    theme_name VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    last_activity TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Media library with full metadata
CREATE TABLE tracks (
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
    date DATE,
    composer VARCHAR(255),
    performer VARCHAR(255),
    conductor VARCHAR(255),
    remixer VARCHAR(255),

    track_number INTEGER,
    track_total INTEGER,
    disc_number INTEGER,
    disc_total INTEGER,

    duration INTEGER,
    bitrate INTEGER,
    sample_rate INTEGER,
    channels INTEGER,
    bits_per_sample INTEGER,
    codec VARCHAR(20),
    file_type VARCHAR(10),
    file_size BIGINT,

    mbid VARCHAR(36),
    album_mbid VARCHAR(36),
    artist_mbid VARCHAR(36),
    acoustid_fingerprint TEXT,

    isrc VARCHAR(12),
    barcode VARCHAR(20),
    catalog_number VARCHAR(50),
    media_type VARCHAR(20),
    country VARCHAR(2),
    label VARCHAR(255),
    copyright TEXT,
    license TEXT,

    lyrics TEXT,
    comment TEXT,
    description TEXT,

    rating INTEGER CHECK (rating >= 0 AND rating <= 5),
    tags TEXT,
    color_palette TEXT,

    play_count INTEGER DEFAULT 0,
    skip_count INTEGER DEFAULT 0,
    last_played TIMESTAMP,

    replaygain_track_gain REAL,
    replaygain_track_peak REAL,
    replaygain_album_gain REAL,
    replaygain_album_peak REAL,
    bpm REAL,
    key VARCHAR(10),
    mood VARCHAR(50),
    energy REAL CHECK (energy >= 0 AND energy <= 1),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    analyzed_at TIMESTAMP
);

-- Scheduler tasks
CREATE TABLE scheduled_tasks (
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
    max_duration_ms INTEGER DEFAULT 0,
    timeout_seconds INTEGER DEFAULT 3600,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Default scheduled tasks with sane defaults
INSERT INTO scheduled_tasks (name, schedule, task_type, command) VALUES
('cleanup_temp', '0 * * * *', 'cleanup', 'cleanTempFiles'),
('cleanup_cache', '0 */6 * * *', 'cleanup', 'cleanCache'),
('rotate_logs', '0 3 * * *', 'cleanup', 'rotateLogs'),
('cleanup_transcodes', '0 4 * * *', 'cleanup', 'cleanTranscodes'),
('backup_database', '0 2 * * *', 'backup', 'backupDatabase'),
('check_quotas', '*/30 * * * *', 'check', 'checkUserQuotas'),
('renew_certificates', '0 1 * * *', 'update', 'checkCertificates'),
('update_podcasts', '0 */6 * * *', 'update', 'updatePodcasts'),
('scan_libraries', '0 3 * * *', 'scan', 'scanLibraries'),
('update_geoip', '0 2 * * 0', 'update', 'updateGeoIP'),
('update_security_lists', '0 3 * * *', 'update', 'updateSecurityLists'),
('check_ffmpeg', '0 3 * * 0', 'update', 'checkFFMPEGUpdate'),
('aggregate_metrics', '*/5 * * * *', 'metrics', 'aggregateMetrics'),
('check_schema', '0 0 * * *', 'maintenance', 'checkSchemaVersion');

-- Albums as separate entities
CREATE TABLE albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255),
    album_artist VARCHAR(255),
    year INTEGER,
    genre VARCHAR(100),
    cover_art_path TEXT,
    cover_art_url TEXT,
    cover_art_colors TEXT,
    mbid VARCHAR(36),
    total_tracks INTEGER,
    total_discs INTEGER,
    label VARCHAR(255),
    catalog_number VARCHAR(50),
    barcode VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, album_artist)
);

-- Artists as separate entities
CREATE TABLE artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) UNIQUE NOT NULL,
    sort_name VARCHAR(255),
    mbid VARCHAR(36),
    biography TEXT,
    image_url TEXT,
    image_colors TEXT,
    website VARCHAR(255),
    country VARCHAR(2),
    formed_year INTEGER,
    disbanded_year INTEGER,
    genre VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Playlists with smart playlist support
CREATE TABLE playlists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cover_image TEXT,
    cover_colors TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    is_collaborative BOOLEAN DEFAULT FALSE,
    is_smart BOOLEAN DEFAULT FALSE,
    smart_criteria TEXT,
    sort_order VARCHAR(50) DEFAULT 'custom',
    play_count INTEGER DEFAULT 0,
    follower_count INTEGER DEFAULT 0,
    duration_ms BIGINT DEFAULT 0,
    track_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_played TIMESTAMP
);

-- Playlist tracks with custom ordering
CREATE TABLE playlist_tracks (
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_by INTEGER REFERENCES users(id),
    PRIMARY KEY (playlist_id, position)
);

-- Broadcasting/Streaming (Icecast-style mount points)
CREATE TABLE broadcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mount_point VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) DEFAULT 'user',
    name VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(100),

    source_url TEXT,
    fallback_mount VARCHAR(255),
    user_id INTEGER REFERENCES users(id),
    stream_key VARCHAR(64),

    bitrate INTEGER DEFAULT 128,
    format VARCHAR(20) DEFAULT 'mp3',
    channels INTEGER DEFAULT 2,
    sample_rate INTEGER DEFAULT 44100,

    is_public BOOLEAN DEFAULT TRUE,
    requires_auth BOOLEAN DEFAULT FALSE,
    allowed_ips TEXT,
    max_listeners INTEGER DEFAULT 0,

    is_active BOOLEAN DEFAULT FALSE,
    is_enabled BOOLEAN DEFAULT TRUE,

    listeners_current INTEGER DEFAULT 0,
    listeners_peak INTEGER DEFAULT 0,
    listeners_total BIGINT DEFAULT 0,
    bytes_sent_total BIGINT DEFAULT 0,

    current_track TEXT,
    metadata_url TEXT,
    website VARCHAR(255),

    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stream history/statistics
CREATE TABLE broadcast_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    broadcast_id INTEGER REFERENCES broadcasts(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL,
    stopped_at TIMESTAMP,
    peak_listeners INTEGER DEFAULT 0,
    unique_listeners INTEGER DEFAULT 0,
    total_bytes_sent BIGINT DEFAULT 0,
    average_listening_time INTEGER DEFAULT 0,
    tracks_played INTEGER DEFAULT 0
);

-- Podcast support with defaults
CREATE TABLE podcasts (
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

-- Podcast episodes
CREATE TABLE podcast_episodes (
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

-- Audiobooks
CREATE TABLE audiobooks (
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

-- Audiobook chapters
CREATE TABLE audiobook_chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    audiobook_id INTEGER REFERENCES audiobooks(id) ON DELETE CASCADE,
    chapter_number INTEGER NOT NULL,
    title VARCHAR(255),
    start_time INTEGER DEFAULT 0,
    end_time INTEGER DEFAULT 0,
    file_path TEXT,

    UNIQUE(audiobook_id, chapter_number)
);

-- Social features
CREATE TABLE follows (
    follower_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    following_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, following_id),
    CHECK (follower_id != following_id)
);

-- Comments system
CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(20),
    entity_id INTEGER NOT NULL,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_edited BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    likes_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Rating system
CREATE TABLE ratings (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(20),
    entity_id INTEGER NOT NULL,
    rating INTEGER DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, entity_type, entity_id)
);

-- Activity feed
CREATE TABLE activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50),
    entity_type VARCHAR(20),
    entity_id INTEGER,
    metadata TEXT,
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API tokens with scopes
CREATE TABLE api_tokens (
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

-- Settings storage (configuration in database) with defaults
CREATE TABLE settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    value_type VARCHAR(50),
    category VARCHAR(50),
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    is_readonly BOOLEAN DEFAULT FALSE,
    default_value TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id)
);

-- Default settings for directories and storage
INSERT INTO settings (key, value, default_value, value_type, category, description) VALUES
('storage.global_music_path', '/mnt/Music/Mp3', '/mnt/Music/Mp3', 'string', 'storage', 'Global music directory'),
('storage.global_podcast_path', '/mnt/Podcasts', '/mnt/Podcasts', 'string', 'storage', 'Global podcast directory'),
('storage.global_audiobook_path', '/mnt/Audiobooks', '/mnt/Audiobooks', 'string', 'storage', 'Global audiobook directory'),
('storage.global_playlist_path', '/mnt/Playlists', '/mnt/Playlists', 'string', 'storage', 'Global playlist directory'),

('storage.user_base_path', '/var/lib/casrad/users', '/var/lib/casrad/users', 'string', 'storage', 'Base path for user directories'),
('storage.default_user_quota', '53687091200', '53687091200', 'integer', 'storage', 'Default user quota in bytes (50GB)'),

('cleanup.temp_retention_hours', '24', '24', 'integer', 'cleanup', 'Hours to keep temp files'),
('cleanup.cache_max_size_gb', '10', '10', 'integer', 'cleanup', 'Maximum cache size in GB'),
('cleanup.cache_ttl_days', '7', '7', 'integer', 'cleanup', 'Days to keep cached files'),
('cleanup.transcode_retention_days', '7', '7', 'integer', 'cleanup', 'Days to keep transcoded files'),
('cleanup.log_retention_days', '30', '30', 'integer', 'cleanup', 'Days to keep log files'),
('cleanup.log_max_size_mb', '1024', '1024', 'integer', 'cleanup', 'Maximum log file size in MB'),

('backup.retention_count', '7', '7', 'integer', 'backup', 'Number of backups to keep'),
('backup.compression', 'true', 'true', 'boolean', 'backup', 'Compress backups'),
('backup.encryption', 'false', 'false', 'boolean', 'backup', 'Encrypt backups'),

('network.port', '0', '0', 'integer', 'network', 'HTTP port (0=auto 64000-64999)'),
('network.https_port', '0', '0', 'integer', 'network', 'HTTPS port (0=auto)'),
('network.bind_address', '0.0.0.0', '0.0.0.0', 'string', 'network', 'Bind address'),
('network.behind_proxy', 'false', 'false', 'boolean', 'network', 'Behind reverse proxy'),

('ratelimit.requests_per_minute', '60', '60', 'integer', 'ratelimit', 'Requests per minute per IP'),
('ratelimit.requests_per_hour', '1000', '1000', 'integer', 'ratelimit', 'Requests per hour per IP'),
('ratelimit.stream_limit', '10', '10', 'integer', 'ratelimit', 'Concurrent streams per user'),
('ratelimit.download_limit', '100', '100', 'integer', 'ratelimit', 'Downloads per day per user'),
('ratelimit.upload_limit', '1000', '1000', 'integer', 'ratelimit', 'Uploads per day per user'),
('ratelimit.transcode_limit', '5', '5', 'integer', 'ratelimit', 'Concurrent transcodes per user'),

('perf.max_connections', '10000', '10000', 'integer', 'performance', 'Maximum connections'),
('perf.max_streams', '1000', '1000', 'integer', 'performance', 'Maximum concurrent streams'),
('perf.max_transcodes', '100', '100', 'integer', 'performance', 'Maximum concurrent transcodes'),
('perf.worker_threads', '0', '0', 'integer', 'performance', 'Worker threads (0=auto)'),
('perf.cache_enabled', 'true', 'true', 'boolean', 'performance', 'Enable caching'),
('perf.cache_driver', 'memory', 'memory', 'string', 'performance', 'Cache driver (none/memory/valkey/redis)'),
('perf.cache_size_mb', '0', '0', 'integer', 'performance', 'Cache size MB (0=auto)'),

('db.pool_size', '25', '25', 'integer', 'database', 'Connection pool size'),
('db.max_idle', '5', '5', 'integer', 'database', 'Max idle connections'),
('db.connection_lifetime', '3600', '3600', 'integer', 'database', 'Connection lifetime seconds'),

('audio.default_format', 'mp3', 'mp3', 'string', 'audio', 'Default stream format'),
('audio.default_bitrate', '192', '192', 'integer', 'audio', 'Default bitrate kbps'),
('audio.transcode_threads', '4', '4', 'integer', 'audio', 'Transcode threads'),
('audio.crossfade_duration', '5', '5', 'integer', 'audio', 'Crossfade duration seconds'),
('audio.replaygain', 'false', 'false', 'boolean', 'audio', 'Enable ReplayGain'),

('mpd.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable MPD server'),
('mpd.port', '6600', '6600', 'integer', 'protocols', 'MPD port'),
('subsonic.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable Subsonic API'),
('ampache.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable Ampache API'),
('webdav.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable WebDAV server'),
('rtmp.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable RTMP server'),
('rtmp.port', '1935', '1935', 'integer', 'protocols', 'RTMP port'),
('dlna.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable DLNA server'),

('security.password_min_length', '8', '8', 'integer', 'security', 'Minimum password length'),
('security.session_duration_hours', '168', '168', 'integer', 'security', 'Session duration hours'),
('security.max_login_attempts', '5', '5', 'integer', 'security', 'Max failed login attempts'),
('security.lockout_duration_minutes', '30', '30', 'integer', 'security', 'Account lockout duration'),
('security.require_https', 'false', 'false', 'boolean', 'security', 'Force HTTPS'),
('security.api_rate_limit', '60', '60', 'integer', 'security', 'API requests per minute'),

('geoip.source', 'p3terx', 'p3terx', 'string', 'security', 'GeoIP database source (p3terx/dbip/custom)'),
('geoip.custom_url', '', '', 'string', 'security', 'Custom GeoIP database URL'),
('geoip.enable_dedup', 'true', 'true', 'boolean', 'security', 'Enable GeoIP de-duplication'),
('geoip.cache_size', '10000', '10000', 'integer', 'security', 'GeoIP lookup cache size'),
('geoip.update_day', '0', '0', 'integer', 'security', 'Day to update GeoIP (0=Sunday)'),
('geoip.update_hour', '2', '2', 'integer', 'security', 'Hour to update GeoIP (0-23)'),

('ui.theme', 'dark', 'dark', 'string', 'ui', 'Default theme (dark/light)'),
('ui.language', 'en', 'en', 'string', 'ui', 'Default language'),
('ui.timezone', 'UTC', 'UTC', 'string', 'ui', 'Default timezone'),
('ui.items_per_page', '50', '50', 'integer', 'ui', 'Items per page'),
('ui.enable_animations', 'true', 'true', 'boolean', 'ui', 'Enable UI animations'),

('metrics.enabled', 'true', 'true', 'boolean', 'metrics', 'Enable metrics collection'),
('metrics.retention_days', '365', '365', 'integer', 'metrics', 'Metrics retention days'),
('metrics.aggregation_interval', '300', '300', 'integer', 'metrics', 'Aggregation interval seconds'),
('metrics.prometheus_enabled', 'false', 'false', 'boolean', 'metrics', 'Enable Prometheus endpoint');

-- Queue persistence
CREATE TABLE user_queues (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    queue_data TEXT NOT NULL,
    current_index INTEGER DEFAULT 0,
    shuffle_mode VARCHAR(20) DEFAULT 'off',
    repeat_mode VARCHAR(20) DEFAULT 'off',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Playback history with detailed tracking
CREATE TABLE playback_history (
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

-- Scrobbling support
CREATE TABLE scrobbles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    service VARCHAR(20),
    track_id INTEGER REFERENCES tracks(id),
    scrobbled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

-- Custom user domains and white labeling
CREATE TABLE user_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    domain VARCHAR(255) UNIQUE NOT NULL,
    subdomain VARCHAR(100),

    is_verified BOOLEAN DEFAULT FALSE,
    verification_token VARCHAR(100),
    verification_method VARCHAR(20) DEFAULT 'dns',
    verified_at TIMESTAMP,

    ssl_enabled BOOLEAN DEFAULT FALSE,
    ssl_certificate_path TEXT,
    ssl_key_path TEXT,
    ssl_expires_at TIMESTAMP,

    is_active BOOLEAN DEFAULT TRUE,
    last_checked TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- White label branding
CREATE TABLE user_branding (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    site_title VARCHAR(255),
    site_tagline TEXT,
    logo_url TEXT,
    favicon_url TEXT,

    primary_color VARCHAR(7),
    secondary_color VARCHAR(7),
    background_color VARCHAR(7),
    text_color VARCHAR(7),
    accent_color VARCHAR(7),

    custom_css TEXT,
    custom_js TEXT,
    custom_head_html TEXT,

    footer_text TEXT,
    hide_powered_by BOOLEAN DEFAULT FALSE,

    meta_description TEXT,
    meta_keywords TEXT,
    og_image_url TEXT,

    analytics_code TEXT,

    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Service installation tracking
CREATE TABLE service_status (
    id INTEGER PRIMARY KEY DEFAULT 1,
    is_installed BOOLEAN DEFAULT FALSE,
    service_type VARCHAR(50),
    service_name VARCHAR(100) DEFAULT 'casrad',
    service_user_uid INTEGER DEFAULT 963,
    service_user_gid INTEGER DEFAULT 963,
    install_path TEXT,

    installed_at TIMESTAMP,
    last_start TIMESTAMP,
    last_stop TIMESTAMP,
    restart_count INTEGER DEFAULT 0,

    auto_start BOOLEAN DEFAULT TRUE,
    restart_on_failure BOOLEAN DEFAULT TRUE,
    max_restart_attempts INTEGER DEFAULT 10,

    CHECK (id = 1)
);

-- Audit logging for security
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    event_type VARCHAR(100),
    event_category VARCHAR(50),

    entity_type VARCHAR(50),
    entity_id INTEGER,
    old_value TEXT,
    new_value TEXT,

    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(36),

    risk_level VARCHAR(20) DEFAULT 'low',

    metadata TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Admin actions tracking
CREATE TABLE admin_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id INTEGER,
    target_user_id INTEGER REFERENCES users(id),

    previous_value TEXT,
    new_value TEXT,

    reason TEXT,
    ip_address VARCHAR(45),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Support system
CREATE TABLE support_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),

    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) DEFAULT 'general',
    priority VARCHAR(20) DEFAULT 'normal',
    status VARCHAR(20) DEFAULT 'open',

    assigned_to INTEGER REFERENCES users(id),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP
);

-- Documentation and knowledge base
CREATE TABLE documentation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    parent_id INTEGER REFERENCES documentation(id),

    is_dynamic BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE,
    requires_auth BOOLEAN DEFAULT FALSE,

    view_count INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,

    tags TEXT,
    search_keywords TEXT,

    version INTEGER DEFAULT 1,
    author_id INTEGER REFERENCES users(id),

    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP
);

-- Compliance tracking (all disabled by default)
CREATE TABLE compliance_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    compliance_type VARCHAR(50) UNIQUE NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    config TEXT,
    override_priority INTEGER DEFAULT 0,

    last_audit TIMESTAMP,
    last_audit_by INTEGER REFERENCES users(id),
    audit_report TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Default compliance settings (ALL DISABLED)
INSERT INTO compliance_settings (compliance_type, enabled, override_priority) VALUES
('gdpr', FALSE, 100),
('ccpa', FALSE, 90),
('coppa', FALSE, 110),
('dmca', FALSE, 80),
('pipeda', FALSE, 70),
('lgpd', FALSE, 75),
('hipaa', FALSE, 120),
('sox', FALSE, 60),
('pci_dss', FALSE, 85),
('ada', FALSE, 50),
('wcag', FALSE, 45);

-- Data retention policies (for compliance)
CREATE TABLE data_retention_policies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    data_type VARCHAR(50) NOT NULL,
    retention_days INTEGER NOT NULL,
    compliance_types TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_cleanup TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Security reporting
CREATE TABLE security_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token VARCHAR(100) UNIQUE NOT NULL,

    reporter_email VARCHAR(255),
    reporter_name VARCHAR(100),

    severity VARCHAR(20) DEFAULT 'low',
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    steps_to_reproduce TEXT,
    impact TEXT,

    status VARCHAR(20) DEFAULT 'new',
    assigned_to INTEGER REFERENCES users(id),

    admin_notes TEXT,
    resolution TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Security tokens for reporting
CREATE TABLE security_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Backup tracking
CREATE TABLE backup_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    backup_type VARCHAR(20) DEFAULT 'full',
    backup_path TEXT NOT NULL,
    backup_size BIGINT DEFAULT 0,

    is_verified BOOLEAN DEFAULT FALSE,
    verification_checksum VARCHAR(64),
    verified_at TIMESTAMP,

    test_restore_success BOOLEAN,
    test_restore_at TIMESTAMP,

    status VARCHAR(20) DEFAULT 'completed',
    error_message TEXT,

    expires_at TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Migration from other platforms (via Admin UI)
CREATE TABLE migration_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_type VARCHAR(50),
    source_version VARCHAR(20),

    import_method VARCHAR(20) DEFAULT 'upload',

    source_config TEXT,
    source_database BLOB,

    status VARCHAR(20) DEFAULT 'pending',
    items_total INTEGER DEFAULT 0,
    items_migrated INTEGER DEFAULT 0,
    items_failed INTEGER DEFAULT 0,

    errors TEXT,
    warnings TEXT,

    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    mapping_rules TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    initiated_by INTEGER REFERENCES users(id)
);

-- First-run downloads tracking
CREATE TABLE component_downloads (
    component VARCHAR(50) PRIMARY KEY,
    status VARCHAR(20) DEFAULT 'pending',
    download_url TEXT,

    current_version VARCHAR(50),
    latest_version VARCHAR(50),

    file_size BIGINT DEFAULT 0,
    bytes_downloaded BIGINT DEFAULT 0,
    progress_percent INTEGER DEFAULT 0,

    expected_checksum VARCHAR(64),
    actual_checksum VARCHAR(64),

    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    last_check TIMESTAMP,
    next_check TIMESTAMP,

    retry_count INTEGER DEFAULT 0,
    error_message TEXT
);

-- Permission scopes
CREATE TABLE permission_scopes (
    scope VARCHAR(50) PRIMARY KEY,
    description TEXT,
    requires_auth BOOLEAN DEFAULT TRUE
);

INSERT INTO permission_scopes (scope, description, requires_auth) VALUES
('public:read', 'Browse and stream public content', FALSE),
('public:search', 'Search public content', FALSE),
('public:metadata', 'View metadata', FALSE),

('user:profile', 'Manage own profile', TRUE),
('user:library', 'Manage own library', TRUE),
('user:playlists', 'Manage own playlists', TRUE),
('user:upload', 'Upload content', TRUE),
('user:broadcast', 'Create broadcasts', TRUE),

('admin:users', 'Manage all users', TRUE),
('admin:content', 'Manage all content', TRUE),
('admin:settings', 'Modify server settings', TRUE),
('admin:security', 'Security management', TRUE),
('admin:system', 'System operations', TRUE),
('admin:migration', 'Import from other platforms', TRUE),
('admin:backup', 'Backup and restore', TRUE);

-- Player state persistence
CREATE TABLE player_state (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    current_track_id INTEGER,
    current_position INTEGER DEFAULT 0,
    queue TEXT,
    shuffle_mode VARCHAR(20) DEFAULT 'off',
    repeat_mode VARCHAR(20) DEFAULT 'off',
    volume INTEGER DEFAULT 70,
    quality_preference VARCHAR(20) DEFAULT 'auto',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Metrics collection
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type VARCHAR(50),
    metric_name VARCHAR(100),
    metric_value REAL,
    metric_unit VARCHAR(20),

    dimensions TEXT,

    collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Aggregated metrics
CREATE TABLE metrics_aggregated (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type VARCHAR(50),
    metric_name VARCHAR(100),

    period VARCHAR(20),
    period_start TIMESTAMP,
    period_end TIMESTAMP,

    min_value REAL,
    max_value REAL,
    avg_value REAL,
    sum_value REAL,
    count_value INTEGER,

    p50_value REAL,
    p90_value REAL,
    p95_value REAL,
    p99_value REAL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Virtual FTS5 table for search
CREATE VIRTUAL TABLE search_index USING fts5(
    title,
    artist,
    album,
    content,
    content_type UNINDEXED,
    content_id UNINDEXED
);

-- Setup state tracking
CREATE TABLE setup_state (
    id INTEGER PRIMARY KEY DEFAULT 1,
    setup_completed BOOLEAN DEFAULT FALSE,
    first_user_id INTEGER REFERENCES users(id),
    admin_account_id INTEGER REFERENCES users(id),
    wizard_step INTEGER DEFAULT 0,
    wizard_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    CHECK (id = 1)
);

-- Cookie consent tracking (when required by compliance)
CREATE TABLE cookie_consent (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    ip_address VARCHAR(45),
    consent_given BOOLEAN DEFAULT FALSE,
    consent_types TEXT DEFAULT '["necessary"]',
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User data requests (for compliance)
CREATE TABLE user_data_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    request_type VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending',
    compliance_type VARCHAR(50),
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    notes TEXT
);

-- Compliance audit log
CREATE TABLE compliance_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(100),
    data_affected TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    compliance_types TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Compliance overrides
CREATE TABLE compliance_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    setting_key VARCHAR(255) NOT NULL,
    original_value TEXT,
    compliance_value TEXT,
    compliance_types TEXT,
    reason TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

-- Admin content management
CREATE TABLE admin_content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL DEFAULT 'page',
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    status VARCHAR(20) DEFAULT 'draft',
    author_id INTEGER REFERENCES users(id),
    is_dynamic BOOLEAN DEFAULT TRUE,
    meta_description TEXT,
    meta_keywords TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP
);

-- Support content
CREATE TABLE support_content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL DEFAULT 'doc',
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    parent_id INTEGER REFERENCES support_content(id),
    is_dynamic BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    tags TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Support search index
CREATE VIRTUAL TABLE support_search USING fts5(
    title,
    content,
    tags,
    content_id UNINDEXED
);

-- SSL certificate tracking
CREATE TABLE ssl_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(20) DEFAULT 'server',
    user_id INTEGER REFERENCES users(id),
    cert_path TEXT,
    issued_at TIMESTAMP,
    expires_at TIMESTAMP,
    last_renewal TIMESTAMP,
    auto_renew BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_type ON global_directories(type);
CREATE INDEX idx_artist ON tracks(artist);
CREATE INDEX idx_album ON tracks(album);
CREATE INDEX idx_title ON tracks(title);
CREATE INDEX idx_user ON tracks(user_id);
CREATE INDEX idx_hash ON tracks(file_hash);
CREATE INDEX idx_global ON tracks(is_global);
CREATE INDEX idx_user_playlist ON playlists(user_id);
CREATE INDEX idx_public ON playlists(is_public);
CREATE INDEX idx_track ON playlist_tracks(track_id);
CREATE INDEX idx_mount ON broadcasts(mount_point);
CREATE INDEX idx_active ON broadcasts(is_active);
CREATE INDEX idx_user_broadcast ON broadcasts(user_id);
CREATE INDEX idx_entity ON comments(entity_type, entity_id);
CREATE INDEX idx_user_activity ON activities(user_id, created_at);
CREATE INDEX idx_public_activity ON activities(is_public, created_at);
CREATE INDEX idx_token ON api_tokens(token);
CREATE INDEX idx_user_token ON api_tokens(user_id);
CREATE INDEX idx_user_history ON playback_history(user_id, started_at);
CREATE INDEX idx_track_history ON playback_history(track_id);
CREATE INDEX idx_user_audit ON audit_log(user_id, created_at);
CREATE INDEX idx_event_audit ON audit_log(event_type, created_at);
CREATE INDEX idx_risk_audit ON audit_log(risk_level, created_at);
CREATE INDEX idx_admin_actions ON admin_actions(admin_id, created_at);
CREATE INDEX idx_target_actions ON admin_actions(target_type, target_id);
CREATE INDEX idx_user_tickets ON support_tickets(user_id);
CREATE INDEX idx_status_tickets ON support_tickets(status);
CREATE INDEX idx_slug_doc ON documentation(slug);
CREATE INDEX idx_category_doc ON documentation(category);
CREATE INDEX idx_status_security ON security_reports(status);
CREATE INDEX idx_severity_security ON security_reports(severity);
CREATE INDEX idx_status_backup ON backup_history(status);
CREATE INDEX idx_expires_backup ON backup_history(expires_at);
CREATE INDEX idx_metric_type ON metrics(metric_type, collected_at);
CREATE INDEX idx_metric_name ON metrics(metric_name, collected_at);
CREATE INDEX idx_agg_type ON metrics_aggregated(metric_type, period, period_start);
CREATE INDEX idx_agg_name ON metrics_aggregated(metric_name, period, period_start);
`
