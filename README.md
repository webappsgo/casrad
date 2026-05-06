# CASRAD

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)

## About

CASRAD (Complete Audio Streaming, Radio, and Distribution) is a single-binary audio streaming server that consolidates 50+ specialized servers into one zero-configuration solution. Supports MPD, Subsonic, Ampache, WebDAV, RTMP, and DLNA protocols.

## Official Site

https://casrad.casapps.us

## Features

- Single binary deployment (~40-50MB static binary)
- Zero configuration required - works immediately
- Self-installing service (systemd, Windows Service, launchd)
- Web-based admin interface with dark/light themes
- Per-user storage with configurable quotas
- Multiple database backends (SQLite default, PostgreSQL, MariaDB)
- Automatic SSL via Let's Encrypt
- Built-in scheduler for maintenance tasks
- MusicBrainz integration for metadata

## Production

### Binary Installation

```bash
# Download and run
wget https://github.com/casapps/casrad/releases/latest/download/casrad-linux-amd64
chmod +x casrad-linux-amd64
./casrad-linux-amd64

# Or with systemd (as root)
./casrad-linux-amd64 --service install
./casrad-linux-amd64 --service start
```

### Docker

```bash
docker run -d \
  --name casrad \
  -p 80:80 \
  -p 6600:6600 \
  -p 1935:1935 \
  -v /path/to/music:/mnt/Music \
  -v casrad-data:/var/lib/casrad \
  ghcr.io/casapps/casrad:latest
```

### Docker Compose

```yaml
services:
  casrad:
    image: ghcr.io/casapps/casrad:latest
    ports:
      - "80:80"
      - "6600:6600"
      - "1935:1935"
    volumes:
      - /path/to/music:/mnt/Music
      - casrad-data:/var/lib/casrad
    environment:
      - TZ=America/New_York
    restart: unless-stopped

volumes:
  casrad-data:
```

## Configuration

Configuration is via environment variables or web admin panel. No config files required.

| Variable | Default | Description |
|----------|---------|-------------|
| `CASRAD_PORT` | auto (64000-64999) | HTTP port |
| `CASRAD_ADDRESS` | 0.0.0.0 | Bind address |
| `CASRAD_DATA` | /var/lib/casrad | Data directory |
| `CASRAD_DB_DRIVER` | sqlite | Database (sqlite/postgres/mariadb) |
| `CASRAD_ADMIN_PATH` | admin | Admin panel path |
| `CASRAD_DEBUG` | false | Enable debug logging |

## API

### Protocol Endpoints

| Protocol | Port/Path | Description |
|----------|-----------|-------------|
| HTTP/HTTPS | 80/443 | Web interface and REST API |
| MPD | 6600 | Music Player Daemon protocol |
| Subsonic | /subsonic/rest/* | Subsonic API v1.16.1 |
| Ampache | /ampache/server/* | Ampache API v6.0.0 |
| WebDAV | /webdav/* | File access |
| RTMP | 1935 | Live streaming |
| DLNA | 1900 (SSDP) | Media server |

### REST API

| Endpoint | Description |
|----------|-------------|
| GET /api/v1/tracks | List tracks |
| GET /api/v1/albums | List albums |
| GET /api/v1/artists | List artists |
| GET /api/v1/playlists | User playlists |
| GET /api/v1/broadcasts | Stream mounts |
| GET /healthz | Health check |
| GET /version | Version info |

See API documentation at `/openapi` when running.

## Development

```bash
# Development build (to temp dir)
make dev

# Host binary (to binaries/)
make host

# All 8 platforms
make build

# Run tests
make test

# Docker image
make docker

# Release
make release
```

### Requirements

- Docker (for building - no local Go installation needed)
- make

## License

MIT License - see [LICENSE.md](LICENSE.md)
