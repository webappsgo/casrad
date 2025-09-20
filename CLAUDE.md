# CASRAD - Complete Audio Streaming and Broadcasting Server
## Full Technical Specification v1.0 - Final Complete Edition

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Core Architecture](#core-architecture)
3. [Database Design](#database-design)
4. [Protocol Implementations](#protocol-implementations)
5. [User Interface & Theming](#user-interface-theming)
6. [Security Architecture](#security-architecture)
7. [Installation & Deployment](#installation-deployment)
8. [API Specification](#api-specification)
9. [Features](#features)
10. [Configuration & Settings](#configuration-settings)
11. [Documentation System](#documentation-system)
12. [Compliance Framework](#compliance-framework)
13. [Build System](#build-system)
14. [Performance & Scaling](#performance-scaling)
15. [Migration System](#migration-system)
16. [Backup & Restore](#backup-restore)
17. [Monitoring & Metrics](#monitoring-metrics)
18. [Complete Replacement List](#replacement-list)

---

## 1. Executive Summary

CASRAD (Complete Audio Streaming, Radio, and Distribution) is a revolutionary single-binary audio streaming and broadcasting server that consolidates the functionality of over 50 different specialized servers into one self-contained, zero-configuration solution.

### Key Innovations
- **Single Binary Deployment**: ~40-50MB static binary containing everything
- **Zero Configuration**: Works immediately upon execution
- **Self-Installing**: Automatically installs as system service
- **Self-Escalating**: Obtains privileges when available
- **Cross-Platform**: Native support for Linux, Windows, macOS, BSD
- **Protocol Complete**: MPD, Subsonic, Ampache, WebDAV, RTMP, DLNA
- **Enterprise Ready**: Scales from personal to datacenter deployment
- **Beautiful UI**: Dracula-inspired dark theme (default) and clean light theme
- **Per-User Storage**: Isolated user directories with quotas
- **Web-Based Management**: Everything managed through comprehensive admin UI

### Design Philosophy
- **Security by Default**: Invisible but comprehensive
- **Mobile-First**: Responsive design throughout
- **User-Friendly**: No technical knowledge required
- **Queue-Preserving**: Adds to queue by default, never destroys
- **Intelligent Defaults**: Every setting has a sane default
- **Progressive Disclosure**: Complexity only when needed
- **Accessibility First**: WCAG 2.1 AA compliant
- **Self-Explanatory**: Intuitive interface with helpful tooltips
- **Minimal CLI**: Web UI for all management tasks

---

## 2. Core Architecture

### 2.1 Binary Architecture

```go
// Single binary contains everything
type CASRADBinary struct {
    // Core Components
    WebServer        *HTTPServer
    Database         *DatabaseEngine
    Cache           *CacheLayer
    ThemeEngine     *ThemeManager
    Scheduler       *TaskScheduler  // Built-in cron-like scheduler
    
    // Protocol Servers
    MPDServer       *MPDProtocolServer
    SubsonicAPI     *SubsonicAPIServer
    AmpacheAPI      *AmpacheAPIServer
    RTMPServer      *RTMPStreamServer
    WebDAVServer    *WebDAVFileServer
    DLNAServer      *DLNAMediaServer
    
    // Management Systems
    ServiceManager   *ServiceInstaller
    UserManager     *SystemUserManager
    CertManager     *ACMECertificateManager
    FFMPEGManager   *FFMPEGDownloader
    StorageManager  *UserStorageManager
    MigrationManager *MigrationImporter
    BackupManager   *BackupRestoreManager
    MetricsManager  *MetricsCollector
    
    // Feature Modules
    Transcoder      *AudioTranscoder
    MusicBrainz     *MusicBrainzTagger
    PodcastManager  *PodcastDownloader
    AutoDJ          *AutoDJEngine
    
    // Embedded Assets
    WebAssets       embed.FS
    Themes          embed.FS
    Documentation   embed.FS
    Migrations      embed.FS
    Templates       embed.FS
}
```

### 2.2 Command Line Interface (Minimal by Design)

```bash
# CASRAD has minimal command line flags by design
# Everything is managed through the web UI

casrad [flags]

Flags:
  -h, --help        Show help
  -v, --version     Show version
  -p, --port PORT   Override default port (default: auto 64000-64999)
  -d, --data PATH   Override data directory (default: OS-specific)
  --debug           Enable debug logging
```

### 2.3 Platform Detection & Adaptation

```go
func NewOSHandler() OSHandler {
    switch runtime.GOOS {
    case "linux":
        return detectLinuxDistro()
    case "darwin":
        return &MacOSHandler{}
    case "windows":
        return &WindowsHandler{}
    case "freebsd", "openbsd", "netbsd":
        return &BSDHandler{variant: runtime.GOOS}
    default:
        return &GenericUnixHandler{}
    }
}

func detectLinuxDistro() OSHandler {
    // Intelligent detection of init system
    if _, err := os.Stat("/run/systemd/system"); err == nil {
        return &SystemdLinuxHandler{}
    }
    if _, err := os.Stat("/sbin/openrc"); err == nil {
        return &OpenRCLinuxHandler{}
    }
    if _, err := os.Stat("/etc/init.d"); err == nil {
        return &SysVLinuxHandler{}
    }
    if isRunningInContainer() {
        return &ContainerHandler{}
    }
    return &GenericLinuxHandler{}
}

func isRunningInContainer() bool {
    // Check for Docker
    if _, err := os.Stat("/.dockerenv"); err == nil {
        return true
    }
    
    // Check for Kubernetes
    if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
        return true
    }
    
    // Check for Podman
    if os.Getenv("container") == "podman" {
        return true
    }
    
    // Check PID 1 process name
    if data, err := os.ReadFile("/proc/1/comm"); err == nil {
        pid1 := strings.TrimSpace(string(data))
        containerInits := []string{
            "tini", "docker-init", "containerd", "sh", "bash",
        }
        for _, init := range containerInits {
            if pid1 == init {
                return true
            }
        }
    }
    
    // Check cgroup for container signatures
    if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
        cgroupData := string(data)
        if strings.Contains(cgroupData, "docker") ||
           strings.Contains(cgroupData, "kubepods") ||
           strings.Contains(cgroupData, "containerd") ||
           strings.Contains(cgroupData, "lxc") {
            return true
        }
    }
    
    return false
}
```

### 2.4 Directory Structure (AS DEFINED)

#### Linux (Privileged Mode)
```
/etc/casrad/
├── server.db           # SQLite database (if using SQLite)
├── backups/           # Database backups
│   └── auto/
├── certs/             # SSL certificates
│   └── letsencrypt/
│       ├── {server_fqdn}/
│       └── {user_domain}/
└── security/          # Security data
    ├── geoip/         # GeoIP databases (auto-downloaded)
    ├── blocklists/    # IP/UA/referrer blocks
    └── wordlists/     # Banned terms

/var/lib/casrad/
└── users/             # Per-user directories
    └── {username}/
        ├── music/     # User's personal music (supports multiple paths)
        ├── podcasts/  # User's subscribed podcasts
        ├── audiobooks/ # User's audiobooks
        ├── radio/     # User's radio recordings
        ├── playlists/ # User's playlists
        ├── recordings/ # User's live recordings
        └── transcodes/ # User's transcoded files

# Global directories (configured in settings, NOT symlinked)
/mnt/Music/Mp3/        # Default global music directory
/mnt/Podcasts/         # Default global podcast directory
/mnt/Audiobooks/       # Default global audiobook directory
/mnt/Playlists/        # Default global playlist directory

# OS Standard directories used
/tmp/casrad/           # Temporary files (auto-cleaned)
/var/cache/casrad/     # Cache files (auto-cleaned)
/var/log/casrad/       # Log files (auto-rotated)
```

#### Linux (User Mode)
```
~/.local/share/casrad/
├── server.db
└── users/
    └── {username}/
        ├── music/
        ├── podcasts/
        ├── audiobooks/
        ├── radio/
        ├── playlists/
        ├── recordings/
        └── transcodes/

~/.cache/casrad/       # Cache directory
~/.config/casrad/      # Config directory
└── backups/
└── security/
    ├── geoip/
    ├── blocklists/
    └── wordlists/

~/.local/state/casrad/
└── logs/
```

#### Windows (System Mode)
```
%PROGRAMDATA%\casrad\
├── server.db
├── users\
│   └── {username}\
│       ├── music\
│       ├── podcasts\
│       ├── audiobooks\
│       ├── radio\
│       ├── playlists\
│       ├── recordings\
│       └── transcodes\
├── backups\
├── certs\
├── security\
│   ├── geoip\
│   ├── blocklists\
│   └── wordlists\
└── logs\

# OS Standard directories used
%TEMP%\casrad\        # Temporary files
%LOCALAPPDATA%\casrad\cache\ # Cache files
```

#### Windows (User Mode)
```
%LOCALAPPDATA%\casrad\
├── server.db
├── users\
│   └── {username}\
│       ├── music\
│       ├── podcasts\
│       ├── audiobooks\
│       ├── radio\
│       ├── playlists\
│       ├── recordings\
│       └── transcodes\
├── cache\
├── logs\
├── backups\
└── security\
    ├── geoip\
    ├── blocklists\
    └── wordlists\
```

#### macOS (System Mode)
```
/Library/Application Support/casrad/
├── server.db
├── users/
│   └── {username}/
│       ├── music/
│       ├── podcasts/
│       ├── audiobooks/
│       ├── radio/
│       ├── playlists/
│       ├── recordings/
│       └── transcodes/
├── backups/
└── security/
    ├── geoip/
    ├── blocklists/
    └── wordlists/

/Library/Caches/casrad/  # OS cache directory
/Library/Logs/casrad/    # OS log directory
/private/tmp/casrad/     # OS temp directory

/etc/casrad/
└── certs/              # SSL certificates
```

#### macOS (User Mode)
```
~/Library/Application Support/casrad/
├── server.db
├── users/
│   └── {username}/
│       ├── music/
│       ├── podcasts/
│       ├── audiobooks/
│       ├── radio/
│       ├── playlists/
│       ├── recordings/
│       └── transcodes/
├── backups/
└── security/
    ├── geoip/
    ├── blocklists/
    └── wordlists/

~/Library/Caches/casrad/
~/Library/Logs/casrad/
```

### 2.5 Built-in Scheduler (WITH DEFINED DEFAULTS)

```go
// Self-cleaning scheduler with sane defaults
type TaskScheduler struct {
    tasks []ScheduledTask
}

func (s *TaskScheduler) Initialize() {
    // All schedules have sane defaults as defined
    
    // Temp file cleanup - every hour (default: 24 hour retention)
    s.Schedule("0 * * * *", s.cleanTempFiles)
    
    // Cache cleanup - every 6 hours (default: 10GB max, 7 day TTL)
    s.Schedule("0 */6 * * *", s.cleanCache)
    
    // Log rotation - daily at 3 AM (default: 30 day retention, 1GB max)
    s.Schedule("0 3 * * *", s.rotateLogs)
    
    // Transcode cache cleanup - daily at 4 AM (default: 7 day retention)
    s.Schedule("0 4 * * *", s.cleanTranscodes)
    
    // Database backup - daily at 2 AM (default: 7 backups kept)
    s.Schedule("0 2 * * *", s.backupDatabase)
    
    // Quota check - every 30 minutes (default: 50GB per user)
    s.Schedule("*/30 * * * *", s.checkUserQuotas)
    
    // Certificate renewal check - daily at 1 AM (default: renew 30 days before expiry)
    s.Schedule("0 1 * * *", s.checkCertificates)
    
    // Podcast updates - every 6 hours (default: check all active feeds)
    s.Schedule("0 */6 * * *", s.updatePodcasts)
    
    // Library scan - daily at 3 AM (default: incremental scan)
    s.Schedule("0 3 * * *", s.scanLibraries)
    
    // GeoIP database update - weekly (default: P3TERX source)
    s.Schedule("0 2 * * 0", s.updateGeoIP)
    
    // Security lists update - daily at 3 AM
    s.Schedule("0 3 * * *", s.updateSecurityLists)
    
    // FFMPEG update check - weekly (default: check for new version)
    s.Schedule("0 3 * * 0", s.checkFFMPEGUpdate)
    
    // Metrics aggregation - every 5 minutes (default: 1 year retention)
    s.Schedule("*/5 * * * *", s.aggregateMetrics)
    
    // Schema version check - on startup and daily (default: auto-migrate)
    s.Schedule("0 0 * * *", s.checkSchemaVersion)
}
```

---

## 3. Database Design

### 3.1 Multi-Database Support

```go
// Database abstraction layer with sane defaults
type DatabaseDriver string

const (
    SQLite     DatabaseDriver = "sqlite"     // Default, embedded, zero-config
    PostgreSQL DatabaseDriver = "postgres"   // Enterprise scale
    MariaDB    DatabaseDriver = "mariadb"   // MySQL compatible
    MySQL      DatabaseDriver = "mysql"      // Uses MariaDB driver
)

// Cache layer with sane defaults
type CacheDriver string

const (
    Memory  CacheDriver = "memory"   // Default, in-process, auto-sizing
    NoCache CacheDriver = "none"     // Disable caching
    Valkey  CacheDriver = "valkey"  // Redis-compatible
    Redis   CacheDriver = "redis"   // Native Redis
)

// Rate limiting with defined defaults
type RateLimiter struct {
    // Per-IP limits (defaults)
    RequestsPerMinute int `default:"60"`
    RequestsPerHour   int `default:"1000"`
    
    // Per-user limits (defaults)
    StreamLimit      int `default:"10"`     // Concurrent streams
    DownloadLimit    int `default:"100"`    // Downloads per day
    UploadLimit      int `default:"1000"`   // Uploads per day
    TranscodeLimit   int `default:"5"`      // Concurrent transcodes
    
    // Global limits (defaults)
    MaxConnections   int `default:"10000"`
    MaxStreams       int `default:"1000"`
    MaxTranscodes    int `default:"100"`
}
```

### 3.2 Complete Database Schema

```sql
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
    role VARCHAR(20) DEFAULT 'user', -- user, moderator, admin
    theme_preference VARCHAR(20) DEFAULT 'dark', -- dark, light, auto
    
    -- Storage configuration
    home_directory TEXT, -- /var/lib/casrad/users/{username}
    storage_quota_bytes BIGINT DEFAULT 53687091200, -- 50GB default
    storage_used_bytes BIGINT DEFAULT 0,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    last_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    
    -- Preferences
    settings TEXT, -- JSON user preferences including UI settings
    avatar_url TEXT,
    bio TEXT,
    website VARCHAR(255),
    location VARCHAR(100),
    
    CONSTRAINT email_format CHECK (email LIKE '%_@_%._%')
);

-- User storage configuration
CREATE TABLE user_storage (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    
    -- Music directories (can have multiple)
    music_paths TEXT, -- JSON array of paths
    
    -- Single directories for other types
    podcast_path TEXT,
    audiobook_path TEXT,
    radio_path TEXT,
    playlist_path TEXT,
    recording_path TEXT,
    transcode_path TEXT,
    
    -- Quota management (with defaults)
    quota_music_bytes BIGINT DEFAULT 21474836480, -- 20GB default
    quota_podcast_bytes BIGINT DEFAULT 10737418240, -- 10GB default
    quota_audiobook_bytes BIGINT DEFAULT 10737418240, -- 10GB default
    quota_recording_bytes BIGINT DEFAULT 5368709120, -- 5GB default
    quota_other_bytes BIGINT DEFAULT 5368709120, -- 5GB default
    
    -- Usage tracking
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
    type VARCHAR(20) NOT NULL, -- music, podcast, audiobook, playlist
    path TEXT NOT NULL,
    
    -- Configuration with defaults
    is_active BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE,
    scan_interval_hours INTEGER DEFAULT 24,
    
    -- Permissions with defaults
    allow_guest_access BOOLEAN DEFAULT TRUE,
    allow_user_access BOOLEAN DEFAULT TRUE,
    
    -- Statistics
    last_scan TIMESTAMP,
    file_count INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_type (type)
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
    
    -- Dracula theme colors (default dark theme)
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
    
    -- UI specific colors
    color_selection VARCHAR(7) DEFAULT '#44475a',
    color_border VARCHAR(7) DEFAULT '#6272a4',
    color_shadow VARCHAR(20) DEFAULT 'rgba(0,0,0,0.3)',
    color_hover VARCHAR(7) DEFAULT '#50fa7b',
    color_active VARCHAR(7) DEFAULT '#bd93f9',
    color_success VARCHAR(7) DEFAULT '#50fa7b',
    color_warning VARCHAR(7) DEFAULT '#f1fa8c',
    color_error VARCHAR(7) DEFAULT '#ff5555',
    color_info VARCHAR(7) DEFAULT '#8be9fd',
    
    -- Typography
    font_family_primary VARCHAR(100) DEFAULT 'Inter, system-ui, sans-serif',
    font_family_mono VARCHAR(100) DEFAULT 'JetBrains Mono, monospace',
    font_size_base VARCHAR(10) DEFAULT '16px',
    font_weight_normal VARCHAR(10) DEFAULT '400',
    font_weight_bold VARCHAR(10) DEFAULT '600',
    
    -- Spacing & Layout
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
    
    -- Custom overrides (NULL = use theme default)
    custom_css TEXT,
    color_background VARCHAR(7),
    color_foreground VARCHAR(7),
    color_primary VARCHAR(7),
    color_secondary VARCHAR(7),
    
    -- Accessibility
    font_size VARCHAR(10) DEFAULT 'medium', -- small, medium, large, x-large
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
    theme_name VARCHAR(20), -- Current theme for session
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    last_activity TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Media library with full metadata
CREATE TABLE tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT UNIQUE NOT NULL,
    file_hash VARCHAR(64), -- SHA256 for deduplication
    user_id INTEGER REFERENCES users(id), -- NULL for global tracks
    is_global BOOLEAN DEFAULT FALSE, -- From global directories
    
    -- Basic metadata
    title VARCHAR(255),
    artist VARCHAR(255),
    album VARCHAR(255),
    album_artist VARCHAR(255),
    
    -- Extended metadata
    genre VARCHAR(100),
    year INTEGER,
    date DATE,
    composer VARCHAR(255),
    performer VARCHAR(255),
    conductor VARCHAR(255),
    remixer VARCHAR(255),
    
    -- Track information
    track_number INTEGER,
    track_total INTEGER,
    disc_number INTEGER,
    disc_total INTEGER,
    
    -- Technical metadata
    duration INTEGER, -- milliseconds
    bitrate INTEGER, -- kbps
    sample_rate INTEGER, -- Hz
    channels INTEGER,
    bits_per_sample INTEGER,
    codec VARCHAR(20),
    file_type VARCHAR(10),
    file_size BIGINT,
    
    -- MusicBrainz integration
    mbid VARCHAR(36),
    album_mbid VARCHAR(36),
    artist_mbid VARCHAR(36),
    acoustid_fingerprint TEXT,
    
    -- Additional metadata
    isrc VARCHAR(12), -- International Standard Recording Code
    barcode VARCHAR(20),
    catalog_number VARCHAR(50),
    media_type VARCHAR(20),
    country VARCHAR(2),
    label VARCHAR(255),
    copyright TEXT,
    license TEXT,
    
    -- Lyrics and descriptions
    lyrics TEXT,
    comment TEXT,
    description TEXT,
    
    -- User metadata
    rating INTEGER CHECK (rating >= 0 AND rating <= 5),
    tags TEXT, -- JSON array
    color_palette TEXT, -- JSON dominant colors for UI theming
    
    -- Statistics
    play_count INTEGER DEFAULT 0,
    skip_count INTEGER DEFAULT 0,
    last_played TIMESTAMP,
    
    -- Analysis data
    replaygain_track_gain REAL,
    replaygain_track_peak REAL,
    replaygain_album_gain REAL,
    replaygain_album_peak REAL,
    bpm REAL,
    key VARCHAR(10),
    mood VARCHAR(50),
    energy REAL CHECK (energy >= 0 AND energy <= 1),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    analyzed_at TIMESTAMP,
    
    -- Indexing
    INDEX idx_artist (artist),
    INDEX idx_album (album),
    INDEX idx_title (title),
    INDEX idx_user (user_id),
    INDEX idx_hash (file_hash),
    INDEX idx_global (is_global)
);

-- Scheduler tasks
CREATE TABLE scheduled_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) UNIQUE NOT NULL,
    schedule VARCHAR(50) NOT NULL, -- Cron format
    task_type VARCHAR(50), -- cleanup, backup, scan, update
    
    -- Configuration with defaults
    is_enabled BOOLEAN DEFAULT TRUE,
    command TEXT, -- Function or script to run
    parameters TEXT, -- JSON parameters
    
    -- Execution tracking
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    last_status VARCHAR(20) DEFAULT 'pending', -- success, failed, running, pending
    last_error TEXT,
    run_count INTEGER DEFAULT 0,
    
    -- Timing
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
    cover_art_colors TEXT, -- JSON of dominant colors for theming
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
    image_colors TEXT, -- JSON of dominant colors for theming
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
    cover_colors TEXT, -- JSON of dominant colors for theming
    is_public BOOLEAN DEFAULT FALSE,
    is_collaborative BOOLEAN DEFAULT FALSE,
    is_smart BOOLEAN DEFAULT FALSE,
    smart_criteria TEXT, -- JSON for smart playlist rules
    sort_order VARCHAR(50) DEFAULT 'custom',
    play_count INTEGER DEFAULT 0,
    follower_count INTEGER DEFAULT 0,
    duration_ms BIGINT DEFAULT 0,
    track_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_played TIMESTAMP,
    INDEX idx_user_playlist (user_id),
    INDEX idx_public (is_public)
);

-- Playlist tracks with custom ordering
CREATE TABLE playlist_tracks (
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_by INTEGER REFERENCES users(id),
    PRIMARY KEY (playlist_id, position),
    INDEX idx_track (track_id)
);

-- Broadcasting/Streaming (Icecast-style mount points)
CREATE TABLE broadcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mount_point VARCHAR(255) UNIQUE NOT NULL,
    type VARCHAR(50) DEFAULT 'user', -- live, autodj, relay, user, radio
    name VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(100),
    
    -- Stream configuration with defaults
    source_url TEXT, -- For relay streams
    fallback_mount VARCHAR(255),
    user_id INTEGER REFERENCES users(id),
    stream_key VARCHAR(64),
    
    -- Technical settings with defaults
    bitrate INTEGER DEFAULT 128,
    format VARCHAR(20) DEFAULT 'mp3', -- mp3, aac, opus, ogg, flac
    channels INTEGER DEFAULT 2,
    sample_rate INTEGER DEFAULT 44100,
    
    -- Access control with defaults
    is_public BOOLEAN DEFAULT TRUE,
    requires_auth BOOLEAN DEFAULT FALSE,
    allowed_ips TEXT, -- JSON array
    max_listeners INTEGER DEFAULT 0, -- 0 = unlimited
    
    -- Status
    is_active BOOLEAN DEFAULT FALSE,
    is_enabled BOOLEAN DEFAULT TRUE,
    
    -- Statistics
    listeners_current INTEGER DEFAULT 0,
    listeners_peak INTEGER DEFAULT 0,
    listeners_total BIGINT DEFAULT 0,
    bytes_sent_total BIGINT DEFAULT 0,
    
    -- Metadata
    current_track TEXT,
    metadata_url TEXT,
    website VARCHAR(255),
    
    -- Timestamps
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_mount (mount_point),
    INDEX idx_active (is_active),
    INDEX idx_user_broadcast (user_id)
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
    average_listening_time INTEGER DEFAULT 0, -- seconds
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
    
    -- Storage
    storage_path TEXT, -- User-specific or global path
    
    -- Sync settings with defaults
    auto_download BOOLEAN DEFAULT TRUE,
    download_quality VARCHAR(20) DEFAULT 'original',
    max_episodes INTEGER DEFAULT 100,
    retention_days INTEGER DEFAULT 30,
    
    -- Status
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
    duration INTEGER DEFAULT 0, -- seconds
    file_size BIGINT DEFAULT 0,
    file_path TEXT, -- Local path if downloaded
    
    -- Playback tracking
    play_position INTEGER DEFAULT 0, -- seconds
    is_played BOOLEAN DEFAULT FALSE,
    played_at TIMESTAMP,
    
    -- Download status
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
    
    -- Storage
    file_path TEXT,
    cover_path TEXT,
    
    -- Metadata
    isbn VARCHAR(20),
    publisher VARCHAR(255),
    published_date DATE,
    language VARCHAR(10),
    description TEXT,
    
    -- Progress tracking
    total_duration INTEGER DEFAULT 0, -- seconds
    current_position INTEGER DEFAULT 0,
    current_chapter INTEGER DEFAULT 0,
    
    -- Statistics
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
    start_time INTEGER DEFAULT 0, -- seconds
    end_time INTEGER DEFAULT 0, -- seconds
    file_path TEXT, -- If chapters are separate files
    
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
    entity_type VARCHAR(20), -- track, album, playlist, broadcast
    entity_id INTEGER NOT NULL,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_edited BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    likes_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_entity (entity_type, entity_id)
);

-- Rating system
CREATE TABLE ratings (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(20), -- track, album, artist
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
    type VARCHAR(50), -- played, liked, followed, commented, broadcast_started
    entity_type VARCHAR(20),
    entity_id INTEGER,
    metadata TEXT, -- JSON additional data
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_activity (user_id, created_at),
    INDEX idx_public_activity (is_public, created_at)
);

-- API tokens with scopes
CREATE TABLE api_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(100),
    permissions TEXT, -- JSON array of scopes
    
    -- Usage tracking
    last_used TIMESTAMP,
    last_ip VARCHAR(45),
    use_count INTEGER DEFAULT 0,
    
    -- Expiration
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_token (token),
    INDEX idx_user_token (user_id)
);

-- Settings storage (configuration in database) with defaults
CREATE TABLE settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT,
    value_type VARCHAR(50), -- boolean, integer, string, json, float
    category VARCHAR(50),
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE, -- Visible to non-admins
    is_readonly BOOLEAN DEFAULT FALSE, -- Cannot be changed via UI
    default_value TEXT, -- Store the default value
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id)
);

-- Default settings for directories and storage
INSERT INTO settings (key, value, default_value, value_type, category, description) VALUES
-- Global directories
('storage.global_music_path', '/mnt/Music/Mp3', '/mnt/Music/Mp3', 'string', 'storage', 'Global music directory'),
('storage.global_podcast_path', '/mnt/Podcasts', '/mnt/Podcasts', 'string', 'storage', 'Global podcast directory'),
('storage.global_audiobook_path', '/mnt/Audiobooks', '/mnt/Audiobooks', 'string', 'storage', 'Global audiobook directory'),
('storage.global_playlist_path', '/mnt/Playlists', '/mnt/Playlists', 'string', 'storage', 'Global playlist directory'),

-- User storage defaults
('storage.user_base_path', '/var/lib/casrad/users', '/var/lib/casrad/users', 'string', 'storage', 'Base path for user directories'),
('storage.default_user_quota', '53687091200', '53687091200', 'integer', 'storage', 'Default user quota in bytes (50GB)'),

-- Cleanup schedules
('cleanup.temp_retention_hours', '24', '24', 'integer', 'cleanup', 'Hours to keep temp files'),
('cleanup.cache_max_size_gb', '10', '10', 'integer', 'cleanup', 'Maximum cache size in GB'),
('cleanup.cache_ttl_days', '7', '7', 'integer', 'cleanup', 'Days to keep cached files'),
('cleanup.transcode_retention_days', '7', '7', 'integer', 'cleanup', 'Days to keep transcoded files'),
('cleanup.log_retention_days', '30', '30', 'integer', 'cleanup', 'Days to keep log files'),
('cleanup.log_max_size_mb', '1024', '1024', 'integer', 'cleanup', 'Maximum log file size in MB'),

-- Backup settings
('backup.retention_count', '7', '7', 'integer', 'backup', 'Number of backups to keep'),
('backup.compression', 'true', 'true', 'boolean', 'backup', 'Compress backups'),
('backup.encryption', 'false', 'false', 'boolean', 'backup', 'Encrypt backups'),

-- Network settings
('network.port', '0', '0', 'integer', 'network', 'HTTP port (0=auto 64000-64999)'),
('network.https_port', '0', '0', 'integer', 'network', 'HTTPS port (0=auto)'),
('network.bind_address', '0.0.0.0', '0.0.0.0', 'string', 'network', 'Bind address'),
('network.behind_proxy', 'false', 'false', 'boolean', 'network', 'Behind reverse proxy'),

-- Rate limiting defaults
('ratelimit.requests_per_minute', '60', '60', 'integer', 'ratelimit', 'Requests per minute per IP'),
('ratelimit.requests_per_hour', '1000', '1000', 'integer', 'ratelimit', 'Requests per hour per IP'),
('ratelimit.stream_limit', '10', '10', 'integer', 'ratelimit', 'Concurrent streams per user'),
('ratelimit.download_limit', '100', '100', 'integer', 'ratelimit', 'Downloads per day per user'),
('ratelimit.upload_limit', '1000', '1000', 'integer', 'ratelimit', 'Uploads per day per user'),
('ratelimit.transcode_limit', '5', '5', 'integer', 'ratelimit', 'Concurrent transcodes per user'),

-- Performance settings
('perf.max_connections', '10000', '10000', 'integer', 'performance', 'Maximum connections'),
('perf.max_streams', '1000', '1000', 'integer', 'performance', 'Maximum concurrent streams'),
('perf.max_transcodes', '100', '100', 'integer', 'performance', 'Maximum concurrent transcodes'),
('perf.worker_threads', '0', '0', 'integer', 'performance', 'Worker threads (0=auto)'),
('perf.cache_enabled', 'true', 'true', 'boolean', 'performance', 'Enable caching'),
('perf.cache_driver', 'memory', 'memory', 'string', 'performance', 'Cache driver (none/memory/valkey/redis)'),
('perf.cache_size_mb', '0', '0', 'integer', 'performance', 'Cache size MB (0=auto)'),

-- Database settings
('db.pool_size', '25', '25', 'integer', 'database', 'Connection pool size'),
('db.max_idle', '5', '5', 'integer', 'database', 'Max idle connections'),
('db.connection_lifetime', '3600', '3600', 'integer', 'database', 'Connection lifetime seconds'),

-- Audio settings
('audio.default_format', 'mp3', 'mp3', 'string', 'audio', 'Default stream format'),
('audio.default_bitrate', '192', '192', 'integer', 'audio', 'Default bitrate kbps'),
('audio.transcode_threads', '4', '4', 'integer', 'audio', 'Transcode threads'),
('audio.crossfade_duration', '5', '5', 'integer', 'audio', 'Crossfade duration seconds'),
('audio.replaygain', 'false', 'false', 'boolean', 'audio', 'Enable ReplayGain'),

-- Protocol settings
('mpd.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable MPD server'),
('mpd.port', '6600', '6600', 'integer', 'protocols', 'MPD port'),
('subsonic.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable Subsonic API'),
('ampache.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable Ampache API'),
('webdav.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable WebDAV server'),
('rtmp.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable RTMP server'),
('rtmp.port', '1935', '1935', 'integer', 'protocols', 'RTMP port'),
('dlna.enabled', 'true', 'true', 'boolean', 'protocols', 'Enable DLNA server'),

-- Security settings
('security.password_min_length', '8', '8', 'integer', 'security', 'Minimum password length'),
('security.session_duration_hours', '168', '168', 'integer', 'security', 'Session duration hours'),
('security.max_login_attempts', '5', '5', 'integer', 'security', 'Max failed login attempts'),
('security.lockout_duration_minutes', '30', '30', 'integer', 'security', 'Account lockout duration'),
('security.require_https', 'false', 'false', 'boolean', 'security', 'Force HTTPS'),
('security.api_rate_limit', '60', '60', 'integer', 'security', 'API requests per minute'),

-- GeoIP settings with P3TERX as default
('geoip.source', 'p3terx', 'p3terx', 'string', 'security', 'GeoIP database source (p3terx/dbip/custom)'),
('geoip.custom_url', '', '', 'string', 'security', 'Custom GeoIP database URL'),
('geoip.enable_dedup', 'true', 'true', 'boolean', 'security', 'Enable GeoIP de-duplication'),
('geoip.cache_size', '10000', '10000', 'integer', 'security', 'GeoIP lookup cache size'),
('geoip.update_day', '0', '0', 'integer', 'security', 'Day to update GeoIP (0=Sunday)'),
('geoip.update_hour', '2', '2', 'integer', 'security', 'Hour to update GeoIP (0-23)'),

-- UI settings
('ui.theme', 'dark', 'dark', 'string', 'ui', 'Default theme (dark/light)'),
('ui.language', 'en', 'en', 'string', 'ui', 'Default language'),
('ui.timezone', 'UTC', 'UTC', 'string', 'ui', 'Default timezone'),
('ui.items_per_page', '50', '50', 'integer', 'ui', 'Items per page'),
('ui.enable_animations', 'true', 'true', 'boolean', 'ui', 'Enable UI animations'),

-- Metrics settings
('metrics.enabled', 'true', 'true', 'boolean', 'metrics', 'Enable metrics collection'),
('metrics.retention_days', '365', '365', 'integer', 'metrics', 'Metrics retention days'),
('metrics.aggregation_interval', '300', '300', 'integer', 'metrics', 'Aggregation interval seconds'),
('metrics.prometheus_enabled', 'false', 'false', 'boolean', 'metrics', 'Enable Prometheus endpoint');

-- Queue persistence
CREATE TABLE user_queues (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    queue_data TEXT NOT NULL, -- JSON array of track IDs
    current_index INTEGER DEFAULT 0,
    shuffle_mode VARCHAR(20) DEFAULT 'off', -- off, tracks, albums
    repeat_mode VARCHAR(20) DEFAULT 'off', -- off, one, all
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Playback history with detailed tracking
CREATE TABLE playback_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE,
    
    -- Playback details
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    play_duration INTEGER DEFAULT 0, -- seconds actually listened
    track_duration INTEGER DEFAULT 0, -- total track duration
    
    -- Context
    source VARCHAR(50), -- web, api, mpd, subsonic, dlna
    source_ip VARCHAR(45),
    user_agent TEXT,
    
    -- Behavior tracking
    skipped BOOLEAN DEFAULT FALSE,
    skip_position INTEGER DEFAULT 0, -- Position when skipped
    
    -- Additional context
    playlist_id INTEGER REFERENCES playlists(id),
    broadcast_id INTEGER REFERENCES broadcasts(id),
    
    INDEX idx_user_history (user_id, started_at),
    INDEX idx_track_history (track_id)
);

-- Scrobbling support
CREATE TABLE scrobbles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    service VARCHAR(20), -- lastfm, librefm, listenbrainz
    track_id INTEGER REFERENCES tracks(id),
    scrobbled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending', -- pending, success, failed
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

-- Custom user domains and white labeling
CREATE TABLE user_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    domain VARCHAR(255) UNIQUE NOT NULL,
    subdomain VARCHAR(100),
    
    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verification_token VARCHAR(100),
    verification_method VARCHAR(20) DEFAULT 'dns', -- dns, http, cname
    verified_at TIMESTAMP,
    
    -- SSL
    ssl_enabled BOOLEAN DEFAULT FALSE,
    ssl_certificate_path TEXT,
    ssl_key_path TEXT,
    ssl_expires_at TIMESTAMP,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_checked TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- White label branding
CREATE TABLE user_branding (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    
    -- Basic branding
    site_title VARCHAR(255),
    site_tagline TEXT,
    logo_url TEXT,
    favicon_url TEXT,
    
    -- Colors (hex values) - can override theme
    primary_color VARCHAR(7),
    secondary_color VARCHAR(7),
    background_color VARCHAR(7),
    text_color VARCHAR(7),
    accent_color VARCHAR(7),
    
    -- Advanced customization
    custom_css TEXT,
    custom_js TEXT,
    custom_head_html TEXT,
    
    -- Footer
    footer_text TEXT,
    hide_powered_by BOOLEAN DEFAULT FALSE,
    
    -- SEO
    meta_description TEXT,
    meta_keywords TEXT,
    og_image_url TEXT,
    
    -- Analytics
    analytics_code TEXT,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Service installation tracking
CREATE TABLE service_status (
    id INTEGER PRIMARY KEY DEFAULT 1,
    is_installed BOOLEAN DEFAULT FALSE,
    service_type VARCHAR(50), -- systemd, windows, launchd, openrc
    service_name VARCHAR(100) DEFAULT 'casrad',
    service_user_uid INTEGER DEFAULT 963,
    service_user_gid INTEGER DEFAULT 963,
    install_path TEXT,
    
    -- Status tracking
    installed_at TIMESTAMP,
    last_start TIMESTAMP,
    last_stop TIMESTAMP,
    restart_count INTEGER DEFAULT 0,
    
    -- Configuration with defaults
    auto_start BOOLEAN DEFAULT TRUE,
    restart_on_failure BOOLEAN DEFAULT TRUE,
    max_restart_attempts INTEGER DEFAULT 10,
    
    CHECK (id = 1) -- Ensure only one row
);

-- Audit logging for security
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    event_type VARCHAR(100), -- login, logout, failed_login, settings_change, etc
    event_category VARCHAR(50), -- auth, admin, security, data
    
    -- Event details
    entity_type VARCHAR(50),
    entity_id INTEGER,
    old_value TEXT,
    new_value TEXT,
    
    -- Context
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(36),
    
    -- Risk assessment
    risk_level VARCHAR(20) DEFAULT 'low', -- low, medium, high, critical
    
    -- Additional data
    metadata TEXT, -- JSON
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_audit (user_id, created_at),
    INDEX idx_event_audit (event_type, created_at),
    INDEX idx_risk_audit (risk_level, created_at)
);

-- Admin actions tracking
CREATE TABLE admin_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id INTEGER,
    target_user_id INTEGER REFERENCES users(id),
    
    -- Change tracking
    previous_value TEXT,
    new_value TEXT,
    
    -- Context
    reason TEXT,
    ip_address VARCHAR(45),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_admin_actions (admin_id, created_at),
    INDEX idx_target_actions (target_type, target_id)
);

-- Support system
CREATE TABLE support_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    
    -- Ticket details
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) DEFAULT 'general',
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent
    status VARCHAR(20) DEFAULT 'open', -- open, in_progress, resolved, closed
    
    -- Assignment
    assigned_to INTEGER REFERENCES users(id),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    closed_at TIMESTAMP,
    
    INDEX idx_user_tickets (user_id),
    INDEX idx_status_tickets (status)
);

-- Documentation and knowledge base
CREATE TABLE documentation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL, -- Markdown with {variables}
    category VARCHAR(100) DEFAULT 'general',
    parent_id INTEGER REFERENCES documentation(id),
    
    -- Metadata with defaults
    is_dynamic BOOLEAN DEFAULT TRUE, -- Supports variable substitution
    is_public BOOLEAN DEFAULT TRUE,
    requires_auth BOOLEAN DEFAULT FALSE,
    
    -- Analytics
    view_count INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    
    -- Search
    tags TEXT, -- JSON array
    search_keywords TEXT,
    
    -- Versioning
    version INTEGER DEFAULT 1,
    author_id INTEGER REFERENCES users(id),
    
    -- Ordering
    sort_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    
    INDEX idx_slug_doc (slug),
    INDEX idx_category_doc (category)
);

-- Compliance tracking (all disabled by default)
CREATE TABLE compliance_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    compliance_type VARCHAR(50) UNIQUE NOT NULL, -- gdpr, ccpa, coppa, dmca
    enabled BOOLEAN DEFAULT FALSE, -- ALL disabled by default
    config TEXT, -- JSON configuration
    override_priority INTEGER DEFAULT 0,
    
    -- Audit trail
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
    data_type VARCHAR(50) NOT NULL, -- logs, history, uploads, etc
    retention_days INTEGER NOT NULL,
    compliance_types TEXT, -- JSON array of compliance types requiring this
    is_active BOOLEAN DEFAULT TRUE,
    last_cleanup TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Security reporting
CREATE TABLE security_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token VARCHAR(100) UNIQUE NOT NULL,
    
    -- Reporter info
    reporter_email VARCHAR(255),
    reporter_name VARCHAR(100),
    
    -- Report details
    severity VARCHAR(20) DEFAULT 'low', -- low, medium, high, critical
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    steps_to_reproduce TEXT,
    impact TEXT,
    
    -- Status
    status VARCHAR(20) DEFAULT 'new', -- new, investigating, resolved, invalid
    assigned_to INTEGER REFERENCES users(id),
    
    -- Response
    admin_notes TEXT,
    resolution TEXT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    resolved_at TIMESTAMP,
    
    INDEX idx_status_security (status),
    INDEX idx_severity_security (severity)
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
    backup_type VARCHAR(20) DEFAULT 'full', -- full, incremental, differential
    backup_path TEXT NOT NULL,
    backup_size BIGINT DEFAULT 0,
    
    -- Verification
    is_verified BOOLEAN DEFAULT FALSE,
    verification_checksum VARCHAR(64),
    verified_at TIMESTAMP,
    
    -- Restoration testing
    test_restore_success BOOLEAN,
    test_restore_at TIMESTAMP,
    
    -- Status
    status VARCHAR(20) DEFAULT 'completed', -- running, completed, failed
    error_message TEXT,
    
    -- Retention
    expires_at TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    
    INDEX idx_status_backup (status),
    INDEX idx_expires_backup (expires_at)
);

-- Migration from other platforms (via Admin UI)
CREATE TABLE migration_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_type VARCHAR(50), -- icecast, subsonic, ampache, etc
    source_version VARCHAR(20),
    
    -- Import method
    import_method VARCHAR(20) DEFAULT 'upload', -- upload, paste
    
    -- Source data
    source_config TEXT, -- Pasted/uploaded config
    source_database BLOB, -- Uploaded database file
    
    -- Migration progress
    status VARCHAR(20) DEFAULT 'pending', -- pending, running, completed, failed
    items_total INTEGER DEFAULT 0,
    items_migrated INTEGER DEFAULT 0,
    items_failed INTEGER DEFAULT 0,
    
    -- Error tracking
    errors TEXT, -- JSON array of errors
    warnings TEXT, -- JSON array of warnings
    
    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Mapping
    mapping_rules TEXT, -- JSON mapping configuration
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    initiated_by INTEGER REFERENCES users(id)
);

-- First-run downloads tracking
CREATE TABLE component_downloads (
    component VARCHAR(50) PRIMARY KEY, -- ffmpeg, geoip, etc
    status VARCHAR(20) DEFAULT 'pending', -- pending, downloading, completed, failed
    download_url TEXT,
    
    -- Version tracking
    current_version VARCHAR(50),
    latest_version VARCHAR(50),
    
    -- Progress tracking
    file_size BIGINT DEFAULT 0,
    bytes_downloaded BIGINT DEFAULT 0,
    ```sql
    progress_percent INTEGER DEFAULT 0,
    
    -- Verification
    expected_checksum VARCHAR(64),
    actual_checksum VARCHAR(64),
    
    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    last_check TIMESTAMP,
    next_check TIMESTAMP,
    
    -- Error handling
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
-- Public scopes (no auth required)
('public:read', 'Browse and stream public content', FALSE),
('public:search', 'Search public content', FALSE),
('public:metadata', 'View metadata', FALSE),

-- User scopes
('user:profile', 'Manage own profile', TRUE),
('user:library', 'Manage own library', TRUE),
('user:playlists', 'Manage own playlists', TRUE),
('user:upload', 'Upload content', TRUE),
('user:broadcast', 'Create broadcasts', TRUE),

-- Admin scopes
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
    current_position INTEGER DEFAULT 0, -- seconds
    queue TEXT, -- JSON array
    shuffle_mode VARCHAR(20) DEFAULT 'off', -- off, tracks, albums
    repeat_mode VARCHAR(20) DEFAULT 'off', -- off, one, all
    volume INTEGER DEFAULT 70, -- 0-100
    quality_preference VARCHAR(20) DEFAULT 'auto',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Metrics collection
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type VARCHAR(50), -- cpu, memory, connections, streams, etc
    metric_name VARCHAR(100),
    metric_value REAL,
    metric_unit VARCHAR(20), -- percent, bytes, count, etc
    
    -- Dimensions
    dimensions TEXT, -- JSON key-value pairs
    
    -- Timestamp
    collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_metric_type (metric_type, collected_at),
    INDEX idx_metric_name (metric_name, collected_at)
);

-- Aggregated metrics
CREATE TABLE metrics_aggregated (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type VARCHAR(50),
    metric_name VARCHAR(100),
    
    -- Aggregation period
    period VARCHAR(20), -- minute, hour, day, week, month
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    
    -- Aggregated values
    min_value REAL,
    max_value REAL,
    avg_value REAL,
    sum_value REAL,
    count_value INTEGER,
    
    -- Percentiles
    p50_value REAL,
    p90_value REAL,
    p95_value REAL,
    p99_value REAL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_agg_type (metric_type, period, period_start),
    INDEX idx_agg_name (metric_name, period, period_start)
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
    consent_types TEXT DEFAULT '["necessary"]', -- JSON array: ["necessary", "analytics", "marketing"]
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User data requests (for compliance)
CREATE TABLE user_data_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    request_type VARCHAR(50), -- export, delete, rectify, restrict
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, completed
    compliance_type VARCHAR(50), -- Which law required this
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
    compliance_types TEXT, -- JSON array of applicable compliance
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Compliance overrides
CREATE TABLE compliance_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    setting_key VARCHAR(255) NOT NULL,
    original_value TEXT,
    compliance_value TEXT,
    compliance_types TEXT, -- JSON array of compliance types requiring this
    reason TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

-- Admin content management
CREATE TABLE admin_content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL DEFAULT 'page', -- 'doc', 'kb', 'faq', 'page'
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    status VARCHAR(20) DEFAULT 'draft', -- draft, published
    author_id INTEGER REFERENCES users(id),
    is_dynamic BOOLEAN DEFAULT TRUE, -- Support {variables}
    meta_description TEXT,
    meta_keywords TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP
);

-- Support content
CREATE TABLE support_content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL DEFAULT 'doc', -- 'doc', 'kb', 'faq'
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
    tags TEXT, -- JSON array
    metadata TEXT, -- JSON metadata
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
    type VARCHAR(20) DEFAULT 'server', -- 'server' or 'user'
    user_id INTEGER REFERENCES users(id), -- NULL for server cert
    cert_path TEXT, -- /etc/casrad/certs/letsencrypt/{domain}/
    issued_at TIMESTAMP,
    expires_at TIMESTAMP,
    last_renewal TIMESTAMP,
    auto_renew BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) DEFAULT 'pending', -- active, expired, pending, failed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Protocol Implementations (AS DEFINED)

### 4.1 MPD (Music Player Daemon) Protocol

**Port**: 6600 (configurable, default)
**Implementation**: Full MPD protocol v0.23.5
**Default**: Enabled

```go
// MPD Protocol Implementation with sane defaults
type MPDServer struct {
    port     int    // Default: 6600
    database *Database
    player   *Player
    enabled  bool   // Default: true
}

func (m *MPDServer) HandleCommand(cmd string, args []string) Response {
    switch cmd {
    // Playback commands
    case "play", "pause", "stop", "next", "previous":
        return m.handlePlayback(cmd, args)
    
    // Queue commands - ALWAYS adds to queue
    case "add", "addid", "clear", "delete", "move", "playlist":
        return m.handleQueue(cmd, args)
    
    // Database commands
    case "find", "search", "list", "listall", "lsinfo":
        return m.handleDatabase(cmd, args)
    
    // Status commands
    case "status", "stats", "currentsong":
        return m.handleStatus(cmd, args)
    }
}
```

**Client Compatibility**:
- ncmpcpp (Linux/macOS)
- Cantata (Cross-platform)
- MPDroid (Android)
- Maximum MPD (iOS)
- Stylophone (iOS)
- malp (Android)
- All other MPD clients

### 4.2 Subsonic API

**Endpoint**: `/subsonic/rest/*`
**Version**: 1.16.1 (latest)
**Default**: Enabled

**Client Compatibility**:
- DSub (Android)
- Ultrasonic (Android)
- play:Sub (iOS)
- Substreamer (iOS)
- Sonixd (Desktop)
- Sublime Music (Linux)
- Strawberry (Desktop)
- Clementine (Desktop)

### 4.3 Ampache API

**Endpoint**: `/ampache/server/*`
**Version**: 6.0.0
**Default**: Enabled

### 4.4 WebDAV Implementation

**Endpoint**: `/webdav/*`
**Default**: Enabled
**Features**: Full WebDAV support

### 4.5 RTMP Streaming Server

**Port**: 1935 (configurable)
**Default**: Enabled
**Features**: Full RTMP support for broadcasting

### 4.6 DLNA/UPnP Server

**Port**: 1900 (SSDP)
**Default**: Enabled
**Features**: Full DLNA media server

---

## 5. User Interface & Theming (AS DEFINED)

### 5.1 Theme System Architecture

```css
/* CSS Variables for Dynamic Theming with Defaults */
:root {
    /* Dracula Dark Theme (Default) */
    --color-background: #282a36;
    --color-current-line: #44475a;
    --color-foreground: #f8f8f2;
    --color-comment: #6272a4;
    --color-cyan: #8be9fd;
    --color-green: #50fa7b;
    --color-orange: #ffb86c;
    --color-pink: #ff79c6;
    --color-purple: #bd93f9;
    --color-red: #ff5555;
    --color-yellow: #f1fa8c;
    
    /* UI Specific */
    --color-selection: #44475a;
    --color-border: #6272a4;
    --color-shadow: rgba(0, 0, 0, 0.3);
    --color-hover: #50fa7b;
    --color-active: #bd93f9;
    
    /* Status Colors */
    --color-success: #50fa7b;
    --color-warning: #f1fa8c;
    --color-error: #ff5555;
    --color-info: #8be9fd;
    
    /* Typography (with fallbacks) */
    --font-primary: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    --font-mono: 'JetBrains Mono', 'Fira Code', 'Consolas', 'Monaco', monospace;
    --font-size-base: 16px;
    --font-weight-normal: 400;
    --font-weight-bold: 600;
    
    /* Spacing & Layout (8px grid) */
    --spacing-xs: 4px;
    --spacing-sm: 8px;
    --spacing-md: 16px;
    --spacing-lg: 24px;
    --spacing-xl: 32px;
    
    /* Borders & Shadows */
    --border-radius: 6px;
    --border-width: 1px;
    --shadow-sm: 0 2px 4px var(--color-shadow);
    --shadow-md: 0 4px 8px var(--color-shadow);
    --shadow-lg: 0 8px 16px var(--color-shadow);
    
    /* Transitions (respects prefers-reduced-motion) */
    --transition-fast: 150ms ease;
    --transition-normal: 250ms ease;
    --transition-slow: 350ms ease;
}

/* Light Theme */
[data-theme="light"] {
    --color-background: #ffffff;
    --color-current-line: #f5f5f5;
    --color-foreground: #2e3440;
    --color-comment: #6c757d;
    --color-cyan: #0969da;
    --color-green: #1a7f37;
    --color-orange: #fb8500;
    --color-pink: #bf3989;
    --color-purple: #8250df;
    --color-red: #cf222e;
    --color-yellow: #d4a72c;
    
    --color-selection: #e1e4e8;
    --color-border: #d0d7de;
    --color-shadow: rgba(0, 0, 0, 0.1);
    --color-hover: #0969da;
    --color-active: #8250df;
}
```

### 5.2 Mobile-First Responsive Design

All UI components are mobile-first with progressive enhancement for larger screens.

### 5.3 Admin Interface

The comprehensive admin interface provides:

- **Dashboard**: Real-time statistics and metrics
- **User Management**: Create, edit, delete users
- **Storage Management**: Monitor quotas and usage
- **Library Management**: Scan, organize, edit metadata
- **Migration Tool**: Import from other platforms via upload/paste
- **Backup/Restore**: One-click backup and restore
- **Settings**: All configuration through UI
- **Logs**: View and search logs
- **Security**: Audit logs, active sessions, API tokens
- **Compliance**: Enable/disable compliance frameworks

---

## 6. Security Architecture (AS DEFINED)

### 6.1 Invisible Security (All Enabled by Default)

Security features that work without user configuration:

```go
// Automatic security measures with defined defaults
type SecurityManager struct {
    rateLimiter     *RateLimiter     // Default: 60 req/min per IP
    bruteForce      *BruteForceProtection // Default: 5 attempts, 30 min lockout
    injectionFilter *InjectionProtection  // Default: enabled
    sessionManager  *SessionManager       // Default: 7 day sessions
    csrfProtection  *CSRFProtection      // Default: enabled
    geoIPManager    *GeoIPManager        // Default: P3TERX source
}

// GeoIP Management with P3TERX as primary
func (s *SecurityManager) updateGeoIP() {
    // P3TERX Mirror - Primary source
    sources := []GeoIPSource{
        {
            Name: "P3TERX-City",
            URL:  "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb",
        },
        {
            Name: "P3TERX-ASN", 
            URL:  "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb",
        },
        {
            Name: "P3TERX-Country",
            URL:  "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb",
        },
    }
    
    // Download and de-duplicate
    s.downloadGeoIPDatabases(sources)
    s.deduplicateGeoIPData()
}

// Security list updates
func (s *SecurityManager) updateSecurityLists() {
    // IP blocklists
    s.updateBlocklist("ips.txt", []string{
        "https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt",
        "https://www.spamhaus.org/drop/drop.txt",
        "https://rules.emergingthreats.net/blockrules/compromised-ips.txt",
    })
    
    // Bad bot user agents
    s.updateBlocklist("user-agents.txt", []string{
        "https://raw.githubusercontent.com/mitchellkrogza/nginx-ultimate-bad-bot-blocker/master/_generator_lists/bad-user-agents.list",
    })
    
    // Referrer spam
    s.updateBlocklist("referrers.txt", []string{
        "https://raw.githubusercontent.com/matomo-org/referrer-spam-list/master/spammers.txt",
    })
    
    // Common passwords (SecLists)
    s.updateWordlist("passwords.txt", []string{
        "https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/10k-most-common.txt",
    })
}
```

### 6.2 Authentication & Authorization

- **Default Password Requirements**: 8 character minimum
- **Session Duration**: 7 days default
- **Failed Login Attempts**: 5 before 30 minute lockout
- **2FA Support**: Optional TOTP
- **API Tokens**: Scoped permissions

### 6.3 Encryption

- **Password Hashing**: Argon2id (default)
- **Session Encryption**: AES-256-GCM
- **TLS Support**: Automatic with Let's Encrypt
- **Database Encryption**: Optional (disabled by default)

### 6.4 Let's Encrypt Integration

Automatic SSL certificates when running on ports 80/443:

```go
func (c *CertificateManager) ObtainCertificate(domain string) error {
    // Automatic certificate management
    // Default: Renew 30 days before expiry
    // Default: Production Let's Encrypt
    // Default: HTTP-01 challenge
}
```

---

## 7. Installation & Deployment (AS DEFINED)

### 7.1 Zero-Configuration Startup

```bash
# Download and run - that's it!
wget https://github.com/casapps/casrad/releases/latest/download/casrad
chmod +x casrad
./casrad

# The server automatically:
# 1. Detects the OS and adapts
# 2. Finds available port (64000-64999 if not root)
# 3. Creates necessary directories
# 4. Initializes SQLite database
# 5. Downloads FFMPEG if needed
# 6. Starts web server
# 7. Opens browser to admin setup (first run)
```

### 7.2 Automatic Service Installation

When run with privileges, CASRAD automatically:

1. Creates system user (UID/GID 963 preferred)
2. Sets up directories with correct permissions
3. Installs appropriate service (systemd/Windows Service/launchd)
4. Starts service
5. Enables auto-start on boot

### 7.3 First Run Wizard

On first access, web-based setup wizard:

1. **Welcome**: Introduction and system check
2. **Admin Account**: Create admin user
3. **Storage**: Configure storage paths (or use defaults)
4. **Protocols**: Enable/disable protocols (all enabled by default)
5. **Network**: Set ports and binding (or use defaults)
6. **SSL**: Configure Let's Encrypt (if on standard ports)
7. **Complete**: Summary and start using

### 7.4 Docker Support

```bash
# Simple Docker run with defaults
docker run -d \
  -p 80:80 \
  -v /mnt/Music:/mnt/Music \
  -v casrad-data:/var/lib/casrad \
  ghcr.io/casapps/casrad:latest

# Everything auto-configures inside container
```

---

## 8. API Specification

### 8.1 RESTful API Structure

Base URL: `/api/v1/`
Default Rate Limit: 60 requests/minute per IP

### 8.2 Authentication Endpoints

```
POST   /api/v1/auth/login         - User login
POST   /api/v1/auth/logout        - User logout
POST   /api/v1/auth/register      - User registration
POST   /api/v1/auth/refresh       - Refresh token
POST   /api/v1/auth/forgot        - Password reset request
POST   /api/v1/auth/reset         - Password reset confirm
GET    /api/v1/auth/verify/{token} - Email verification
```

### 8.3 Admin Endpoints

All admin operations are available through the web UI:

```
GET    /api/v1/admin/dashboard    - Dashboard metrics
POST   /api/v1/admin/migrate      - Import from other platform
POST   /api/v1/admin/backup       - Create backup
POST   /api/v1/admin/restore      - Restore from backup
GET    /api/v1/admin/metrics      - Prometheus metrics
```

---

## 9. Features (AS DEFINED)

### 9.1 MusicBrainz Integration

- **Default**: Enabled for metadata enhancement
- **Automatic**: Tags files on import
- **AcoustID**: Fingerprinting for identification
- **Rate Limit**: Respects MusicBrainz rate limits

### 9.2 Podcast Management

- **Default Update Interval**: Every 6 hours
- **Default Retention**: 30 days
- **Default Max Episodes**: 100 per feed
- **Automatic Download**: Enabled by default

### 9.3 AutoDJ

- **Default**: Disabled (enable in admin UI)
- **Crossfade**: 5 seconds default
- **Rules**: No repeat artist in 30 min, no repeat track in 2 hours
- **Algorithm**: BPM matching, harmonic mixing

### 9.4 Social Features

- **Default**: Enabled
- **Features**: Follow users, activity feed, comments
- **Privacy**: Users control visibility

### 9.5 White Labeling

- **Default**: Disabled
- **Custom Domains**: Configure in admin UI
- **Branding**: Upload logos, set colors
- **SSL**: Automatic for verified domains

---

## 10. Configuration & Settings (AS DEFINED)

All settings have sane defaults and are configurable through the admin UI. No configuration files needed.

### 10.1 Default Settings Summary

- **Port**: 0 (auto 64000-64999 unprivileged, 80/443 privileged)
- **Storage**: 50GB quota per user
- **Audio**: MP3 192kbps default streaming
- **Protocols**: All enabled
- **Security**: All protections enabled
- **Cache**: Memory cache enabled (auto-sizing)
- **Theme**: Dracula dark theme
- **Language**: English
- **Timezone**: UTC (auto-detect from browser)
- **GeoIP**: P3TERX source (no registration)

---

## 11. Documentation System

### 11.1 Built-in Documentation

All documentation is embedded and accessible at `/support`:

- **Quick Start Guide**: Auto-populated with server details
- **User Guide**: Complete user documentation
- **Admin Guide**: Administration documentation
- **API Documentation**: Interactive API docs
- **Protocol Guides**: Setup guides for each protocol

### 11.2 Dynamic Variables

Documentation automatically includes server-specific information:

- `{SERVER_URL}` - Your server's URL
- `{SERVER_TITLE}` - Your server's name
- `{MPD_PORT}` - MPD port number
- `{ADMIN_EMAIL}` - Admin contact email

---

## 12. Compliance Framework (AS DEFINED)

### 12.1 All Compliance Disabled by Default

No compliance frameworks are enabled by default. Enable only what you need through the admin UI:

- GDPR - European data protection
- CCPA - California privacy
- COPPA - Children's online privacy
- DMCA - Copyright protection
- Others available but disabled

### 12.2 Boolean Value Parsing

The system accepts various boolean values for maximum compatibility:

**True values**: true, yes, on, enable, enabled, 1, active, y, t, ok, okay, accept, accepted, allow, allowed

**False values**: false, no, off, disable, disabled, 0, inactive, n, f, deny, denied, reject, rejected

---

## 13. Build System

### 13.1 Single Binary Build

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Output: Single ~40-50MB static binary
```

### 13.2 No External Dependencies

The binary contains:
- Web server
- All protocol implementations
- SQLite database
- Web UI assets
- Themes
- Documentation
- FFMPEG downloader

---

## 14. Performance & Scaling (AS DEFINED)

### 14.1 Default Performance Settings

- **Max Connections**: 10,000
- **Max Concurrent Streams**: 1,000
- **Max Concurrent Transcodes**: 100
- **Worker Threads**: Auto (CPU cores)
- **Cache**: Memory cache enabled (auto-sizing)

### 14.2 Automatic Optimization

- **Database**: Auto-vacuum enabled
- **Indexes**: Created automatically
- **Connection Pooling**: 25 connections default
- **Request Coalescing**: Duplicate requests merged
- **Static Asset Caching**: 1 year cache headers

---

## 15. Migration System (AS DEFINED)

### 15.1 Web-Based Migration Tool

Access via Admin UI → Tools → Migration

**Supported Platforms**:
- Icecast/Icecast2
- Shoutcast
- Subsonic/Airsonic/Navidrome
- Ampache
- Jellyfin/Emby/Plex (audio only)
- MPD
- Funkwhale

**Import Methods**:
1. **Upload Config**: Upload configuration files
2. **Paste Config**: Paste configuration text
3. **Upload Database**: Upload database file
4. **API Import**: Connect via API (if source is running)

### 15.2 Automatic Mapping

The migration system automatically:
- Maps users to CASRAD users
- Converts playlists
- Preserves play counts
- Maintains folder structure
- Converts metadata

---

## 16. Backup & Restore (AS DEFINED)

### 16.1 Web-Based Backup System

Access via Admin UI → Tools → Backup

**Backup Options**:
- **Full Backup**: Database + config + user data
- **Database Only**: Just the database
- **Config Only**: Settings and configuration

**Automatic Backups**:
- **Default Schedule**: Daily at 2 AM
- **Default Retention**: 7 backups
- **Default Compression**: Enabled
- **Default Location**: `/etc/casrad/backups/auto/`

### 16.2 One-Click Restore

Upload backup file through admin UI to restore:
- Automatic version checking
- Selective restore options
- Progress tracking
- Rollback on failure

---

## 17. Monitoring & Metrics (AS DEFINED)

### 17.1 Built-in Metrics Collection

**Default**: Enabled (1 year retention)

**Metrics Collected**:
- CPU usage
- Memory usage
- Disk usage
- Active connections
- Active streams
- Bandwidth usage
- Request rates
- Error rates

### 17.2 Endpoints

```
GET /api/v1/metrics         - JSON metrics
GET /api/v1/metrics/prometheus - Prometheus format (disabled by default)
```

### 17.3 Admin Dashboard

Real-time metrics displayed in admin UI:
- Current listeners
- Popular tracks
- User activity
- System resources
- Error logs
- Recent activities

---

## 18. Complete Replacement List

### 18.1 What CASRAD Replaces

CASRAD completely replaces the following 50+ servers:

**Streaming Servers**:
- Icecast2
- Shoutcast v1/v2
- Liquidsoap
- Azuracast
- LibreTime/Airtime
- Radio.co (self-hosted)
- Mixxx (server components)

**Music Servers**:
- Subsonic
- Airsonic/Airsonic-Advanced
- Navidrome
- Ampache
- Jellyfin (audio only)
- Plex (audio only)
- Emby (audio only)
- Funkwhale
- Koel
- Mopidy
- Beets (server component)
- Madsonic
- Music Assistant
- Volumio (server)
- moOde (server)
- RuneAudio (server)
- piCorePlayer (server)

**Protocol Servers**:
- MPD (Music Player Daemon)
- forked-daapd
- Rygel
- miniDLNA/ReadyMedia
- Gerbera
- Universal Media Server (audio)

**Broadcasting Tools**:
- OpenBroadcaster (server)
- Rivendell (streaming)
- StationPlaylist (server)
- RadioDJ (server)
- BUTT (server-side)
- Rocket Broadcaster

**Podcast Servers**:
- Podify
- Podsync
- Podcast Generator
- Castopod

**File Servers** (for music):
- WebDAV servers
- FTP servers (via WebDAV)
- Nextcloud (music app only)

**Development Tools**:
- Live555 Media Server
- Node Media Server
- nginx-rtmp-module
- Simple RTMP Server
- GStreamer RTSP Server

### 18.2 What CASRAD Works With

**Clients**: All existing clients for the protocols above
**Infrastructure**: Standard web infrastructure (reverse proxies, CDNs, etc.)

---

## Summary

CASRAD is a complete, production-ready audio streaming and broadcasting server that consolidates 50+ different servers into a single ~40-50MB binary. 

**Key Points**:
- **Zero configuration required** - Everything has sane defaults
- **Single binary** - No dependencies, no installation complexity
- **Automatic everything** - Service installation, SSL, user creation
- **Beautiful UI** - Dracula dark theme by default, clean light theme option
- **Web-based management** - No command line needed after starting
- **Complete protocol support** - Works with all existing clients
- **Per-user storage** - Isolated directories with quotas
- **Enterprise features** - With home user simplicity
- **Invisible security** - Comprehensive protection by default
- **Memory cache by default** - Optimal performance out of the box
- **P3TERX GeoIP** - No registration required
- **Smart service detection** - Knows when to install as service

**Getting Started**:
```bash
# That's literally it:
wget https://github.com/casapps/casrad/releases/latest/download/casrad
chmod +x casrad
./casrad
```

Open your browser, complete the 2-minute setup wizard, and start streaming!

**One Binary. No Dependencies. No Configuration. Just Works.™**

---

*End of CASRAD Full Technical Specification v1.0 - Final Complete Edition*

*This specification is complete and comprehensive. All settings have defined defaults. Nothing is left undefined.*

