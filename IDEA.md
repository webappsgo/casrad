# CASRAD - Project Idea

## Project description

CASRAD (Complete Audio Streaming, Radio, and Distribution) is a single-binary, zero-configuration audio streaming and broadcasting server that consolidates 50+ specialized servers into one self-contained solution. It provides complete protocol support for all major audio streaming standards while requiring no external dependencies or configuration.

### Target Users

- Home users who want a simple, self-hosted music streaming solution
- Radio station operators needing a complete broadcasting platform
- Content creators managing podcasts and audiobooks
- Small to enterprise organizations needing centralized audio management
- Anyone wanting to replace multiple audio servers with one solution

### Design Philosophy

- **Security by Default**: Invisible but comprehensive protection
- **Mobile-First**: Responsive design throughout
- **User-Friendly**: No technical knowledge required
- **Queue-Preserving**: Adds to queue by default, never destroys user selections
- **Intelligent Defaults**: Every setting has a sane default
- **Progressive Disclosure**: Complexity only when needed
- **Accessibility First**: WCAG 2.1 AA compliant
- **Self-Explanatory**: Intuitive interface with helpful tooltips
- **Minimal CLI**: Web UI for all management tasks

### Replaces

CASRAD completely replaces the following 50+ servers:

**Streaming Servers**
- Icecast2, Shoutcast v1/v2, Liquidsoap
- Azuracast, LibreTime/Airtime
- Radio.co (self-hosted), Mixxx (server components)

**Music Servers**
- Subsonic, Airsonic/Airsonic-Advanced, Navidrome
- Ampache, Jellyfin (audio only), Plex (audio only), Emby (audio only)
- Funkwhale, Koel, Mopidy, Beets (server component), Madsonic
- Music Assistant, Volumio (server), moOde (server)
- RuneAudio (server), piCorePlayer (server)

**Protocol Servers**
- MPD (Music Player Daemon), forked-daapd
- Rygel, miniDLNA/ReadyMedia, Gerbera
- Universal Media Server (audio)

**Broadcasting Tools**
- OpenBroadcaster (server), Rivendell (streaming)
- StationPlaylist (server), RadioDJ (server)
- BUTT (server-side), Rocket Broadcaster

**Podcast Servers**
- Podify, Podsync, Podcast Generator, Castopod

**File Servers (for music)**
- WebDAV servers, FTP servers (via WebDAV)
- Nextcloud (music app only)

**Development Tools**
- Live555 Media Server, Node Media Server
- nginx-rtmp-module, Simple RTMP Server
- GStreamer RTSP Server

**Works With**
- **Clients**: All existing clients for the protocols above (ncmpcpp, DSub, Ultrasonic, etc.)
- **Infrastructure**: Standard web infrastructure (reverse proxies, CDNs, load balancers)

### Notes

Download, make executable, run. Complete the setup wizard in your browser.

- **Zero configuration required** — Everything has sane defaults
- **Single binary** — No dependencies, no installation complexity
- **Automatic everything** — Service installation, SSL certificates, user creation
- **Beautiful UI** — Dark theme by default, light theme option
- **Web-based management** — No command line needed after initial start
- **Complete protocol support** — Works with all existing clients
- **Per-user storage** — Isolated directories with quotas
- **Enterprise features** — With home user simplicity
- **Invisible security** — Comprehensive protection by default

**One Binary. No Dependencies. No Configuration. Just Works.**

---

## Project variables

| Variable | Value |
|----------|-------|
| `project_name` | casrad |
| `project_org` | casapps |
| `internal_name` | casrad (**frozen** — never change after initial setup) |
| `plist_name` | io.github.casapps.casrad |
| `module` | github.com/casapps/casrad |
| `site` | https://casrad.casapps.us |

---

## Business logic

### Core Features

- **Single Binary**: ~40-50MB static binary with all assets embedded
- **Zero Configuration**: Works immediately with sane defaults
- **Self-Installing**: Automatic service installation on supported platforms
- **Cross-Platform**: Linux, Windows, macOS, BSD support (8 platform builds)

### Protocol Support

- **MPD**: Full Music Player Daemon protocol for music player clients
- **Subsonic API**: Compatibility with Subsonic/Airsonic clients
- **Ampache API**: Compatibility with Ampache clients
- **WebDAV**: Direct file access for any WebDAV client
- **RTMP**: Live audio broadcasting and streaming
- **DLNA/UPnP**: Media server for smart TVs and devices

### Audio Management

- **Music Library**: Full metadata management with MusicBrainz integration
- **Podcasts**: Automatic subscription and download management
- **Audiobooks**: Chapter tracking with position memory
- **Playlists**: Standard and smart playlist support
- **Broadcasting**: Live streaming with AutoDJ fallback

### User Features

- **Per-User Storage**: Isolated directories with configurable quotas
- **API Tokens**: Scoped tokens for programmatic access
- **Social Features**: Follow users, activity feeds, comments
- **White Labeling**: Custom domains and branding per user
- **Scrobbling**: Last.fm, Libre.fm, ListenBrainz support

### Administration

- **Web-Based Management**: Full admin UI for all configuration
- **Migration Tool**: Import from Icecast, Subsonic, Ampache, etc.
- **Backup/Restore**: Scheduled backups with one-click restore
- **Metrics**: Real-time monitoring and Prometheus export
- **Compliance**: Optional GDPR, CCPA, DMCA frameworks (disabled by default)

### Theming

- **Three Themes**: Dark (default), Light, Auto (system preference)
- **Project-Wide**: Same theme across all interfaces
- **Responsive**: Mobile and desktop optimized
- **Accessible**: WCAG AA compliant contrast ratios

---

### Data Models

#### Server Admins
Administrative accounts for server management — separate from regular users.
- Unique identifier, username, email
- Password (hashed Argon2id), optional 2FA with backup codes
- Account status, login timestamps

#### Users
Regular user accounts for application features.
- Unique identifier, username, email
- Password (hashed Argon2id), optional 2FA with backup codes
- Role (user/moderator), theme preference
- Storage quota and usage tracking
- Notification preferences

#### Tracks
Audio files in the library.
- File location and content hash for deduplication
- Owner (user or global library)
- Metadata: title, artist, album, album artist, genre, year
- Technical: duration, bitrate, sample rate, codec, file size
- MusicBrainz ID and acoustic fingerprint
- User data: lyrics, play count, skip count, rating (0-5)
- Audio analysis: ReplayGain values, BPM, musical key

#### Albums
Album groupings for tracks.
- Title, artist, album artist, year, genre
- Cover art, MusicBrainz ID
- Track and disc counts, record label

#### Artists
Artist entities.
- Name, sort name, MusicBrainz ID
- Biography, image, country of origin

#### Playlists
User-created track collections.
- Owner, name, description
- Public/private visibility, collaborative editing
- Smart playlist support with filter criteria
- Track count and total duration

#### Broadcasts (Mount Points)
Live audio streams and radio stations.
- Mount point identifier, stream type (live/autodj/relay/user)
- Name, description, genre
- Audio settings: bitrate, format
- Access control: public/private, authentication required
- Listener limits and statistics

#### Podcasts
Podcast subscriptions.
- Owner, feed URL
- Podcast metadata: title, author, image
- Download settings: auto-download, max episodes, retention period
- Sync status and errors

#### Audiobooks
Audiobook content with progress tracking.
- Owner, title, author, narrator
- Series information
- Total duration, current playback position, current chapter
- Completion status

---

### Business Rules

#### Storage
- Default user quota: 50GB (configurable)
- Quota breakdown: Music 20GB, Podcast 10GB, Audiobook 10GB, Other 10GB
- Global directories accessible to all users (admin-configured)
- Per-user isolated storage with quota enforcement

#### Authentication
- Password: 8 character minimum
- Session duration: 7 days default (configurable)
- Failed login lockout: 5 attempts, 30 minute lockout
- Login accepts: username, user ID, or email (auto-detected)
- Optional TOTP 2FA with 10 backup codes
- Optional Passkeys/WebAuthn support

#### Username Validation
- Length: 3-32 characters
- Allowed: lowercase letters, numbers, underscore, hyphen
- Must start with letter, cannot end with underscore or hyphen
- Reserved names blocked (admin, root, system, api, etc.)

#### Registration Modes
- **Disabled**: Admin creates all accounts
- **Public**: Anyone can register (email verification required)
- **Private**: Invite code required
- **Approval**: Admin approval required

#### Streaming
- Default format: MP3 192kbps
- Concurrent streams per user: 10
- Transcoding on-demand with 7-day cache retention

#### Rate Limiting
- Requests per minute per IP: 60
- Requests per hour per IP: 1000
- Downloads per day per user: 100
- Concurrent transcodes per user: 5

#### Scheduled Tasks
- Temp cleanup: Hourly (24h retention)
- Cache cleanup: Every 6 hours (10GB max, 7 day TTL)
- Log rotation: Daily at 3 AM (30 day retention)
- Database backup: Daily at 2 AM (7 backups kept)
- Podcast updates: Every 6 hours
- Library scan: Daily at 3 AM (incremental)
- GeoIP update: Weekly

---

### Endpoints Summary

#### What Users Can Do
- **Authentication**: Login, logout, register, reset password, verify email
- **Profile Management**: View/update profile, change settings, manage security (2FA, passkeys)
- **API Tokens**: Create, list, and revoke personal API tokens
- **Invites**: Generate and manage invite codes (when private registration enabled)

#### What Users Can Access
- **Library**: Browse and search tracks, albums, artists
- **Streaming**: Stream audio with on-demand transcoding
- **Playlists**: Create, edit, share playlists; add/remove tracks
- **Broadcasts**: Listen to live streams, create personal mount points
- **Podcasts**: Subscribe to feeds, browse episodes, stream/download
- **Audiobooks**: Browse library, stream with position tracking

#### What Admins Can Do
- **Server Settings**: Configure all server options via web UI
- **User Management**: Create, edit, disable, delete user accounts
- **Approval Queue**: Review and approve pending registrations
- **Backup/Restore**: Create backups, restore from backup
- **Migration**: Import data from other platforms (Icecast, Subsonic, etc.)
- **Monitoring**: View metrics, logs, audit trails
- **Security**: Configure authentication, firewall, GeoIP blocking

#### Protocol Endpoints
- **MPD**: Music Player Daemon protocol for MPD clients
- **Subsonic**: Subsonic API for Subsonic-compatible apps
- **Ampache**: Ampache API for Ampache-compatible apps
- **WebDAV**: File access for WebDAV clients
- **RTMP**: Live streaming for broadcasting software
- **DLNA**: Media discovery for smart devices

#### System Endpoints
- **Health Check**: Server health status
- **Version**: Server version information
- **Well-Known**: Standard well-known URLs (security.txt, change-password)

---

### Data Sources

#### Internal
- SQLite database (default, zero-config)
- PostgreSQL/MariaDB/MySQL (optional, enterprise scale)
- Memory cache (default) or Valkey/Redis (optional)

#### External Integrations
- **MusicBrainz**: Metadata enrichment and tagging
- **AcoustID**: Audio fingerprinting for identification
- **GeoIP**: Geographic IP database for access control
- **Let's Encrypt**: Automatic SSL certificates

#### File Sources
- Local filesystem paths (global and per-user)
- Podcast RSS feeds
- RTMP streams (for relay)

---

### Reserved Names

Blocked from use as usernames or organization names:
- System: admin, root, system, api, server, host, localhost
- Routes: auth, users, orgs, settings, profile, static, assets
- Project-specific: casrad, casapps
