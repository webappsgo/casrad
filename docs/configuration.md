# Configuration

CASRAD follows a "zero configuration" philosophy - everything works with sensible defaults. All settings can be modified through the web-based admin panel.

## Default Directories

### Linux (Privileged Mode)

| Path | Purpose |
|------|---------|
| `/etc/casrad/` | Configuration, database, certs |
| `/var/lib/casrad/users/` | Per-user directories |
| `/var/log/casrad/` | Log files |
| `/var/cache/casrad/` | Cache files |
| `/tmp/casrad/` | Temporary files |

### Linux (User Mode)

| Path | Purpose |
|------|---------|
| `~/.local/share/casrad/` | Data and database |
| `~/.config/casrad/` | Configuration |
| `~/.local/state/casrad/logs/` | Log files |
| `~/.cache/casrad/` | Cache files |

## Environment Variables

All settings can be overridden via environment variables:

```bash
# Server settings
CASRAD_ADDRESS=0.0.0.0
CASRAD_PORT=80

# Database
CASRAD_DB_TYPE=sqlite           # sqlite, postgres, mariadb
CASRAD_DB_PATH=/data/server.db  # SQLite path
CASRAD_DB_HOST=localhost        # PostgreSQL/MariaDB host
CASRAD_DB_PORT=5432
CASRAD_DB_NAME=casrad
CASRAD_DB_USER=casrad
CASRAD_DB_PASSWORD=secret

# Cache
CASRAD_CACHE_DRIVER=memory      # memory, valkey, redis, none
CASRAD_CACHE_ADDRESS=localhost:6379

# Mode
MODE=production                 # production, development
DEBUG=false
TZ=America/New_York
```

## Default Ports

| Service | Port | Configurable |
|---------|------|--------------|
| HTTP | 80 (root) / 64000-64999 (user) | Yes |
| HTTPS | 443 | Yes |
| MPD | 6600 | Yes |
| RTMP | 1935 | Yes |

## Protocol Settings

All protocols are enabled by default:

| Protocol | Default | Admin Setting |
|----------|---------|---------------|
| Subsonic API | Enabled | `/admin/protocols` |
| Ampache API | Enabled | `/admin/protocols` |
| MPD | Enabled | `/admin/protocols` |
| WebDAV | Enabled | `/admin/protocols` |
| RTMP | Enabled | `/admin/protocols` |
| DLNA | Enabled | `/admin/protocols` |

## Storage Quotas

Default user storage quota: 50GB

| Storage Type | Default Quota |
|--------------|---------------|
| Music | 20GB |
| Podcasts | 10GB |
| Audiobooks | 10GB |
| Recordings | 5GB |
| Other | 5GB |

## Security Settings

| Setting | Default |
|---------|---------|
| Password minimum length | 8 characters |
| Session duration | 7 days |
| Failed login attempts | 5 (30 min lockout) |
| Rate limit | 60 req/min per IP |

## Admin Panel

All settings are configurable via the web UI at `/admin`.
