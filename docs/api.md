# API Reference

## REST API

Base URL: `/api/v1/`

### Authentication

```bash
# Login and get session
POST /api/v1/auth/login
Content-Type: application/json
{"username": "user", "password": "pass"}

# Use API token (header)
Authorization: Bearer <token>
```

### Response Formats

| Extension | Content-Type | Description |
|-----------|--------------|-------------|
| `.json` | application/json | JSON (default for API) |
| `.txt` | text/plain | Plain text |
| (none) | Auto-detect | Based on Accept header |

### Health & Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/healthz` | GET | Health check |
| `/api/v1/server/status` | GET | Server status |
| `/api/v1/server/version` | GET | Version info |

### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | User login |
| `/api/v1/auth/logout` | POST | User logout |
| `/api/v1/auth/register` | POST | User registration |
| `/api/v1/auth/refresh` | POST | Refresh token |

### Library

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/tracks` | GET | List tracks |
| `/api/v1/tracks/{id}` | GET | Get track |
| `/api/v1/albums` | GET | List albums |
| `/api/v1/albums/{id}` | GET | Get album |
| `/api/v1/artists` | GET | List artists |
| `/api/v1/artists/{id}` | GET | Get artist |
| `/api/v1/playlists` | GET | List playlists |
| `/api/v1/playlists/{id}` | GET | Get playlist |

### Streaming

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/stream/{id}` | GET | Stream track |
| `/api/v1/stream/{id}/transcode` | GET | Transcoded stream |
| `/api/v1/broadcasts` | GET | List broadcasts |
| `/api/v1/broadcasts/{mount}` | GET | Get broadcast |

### Admin Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/server/settings` | GET/PUT | Server settings |
| `/api/v1/admin/server/logs` | GET | Server logs |
| `/api/v1/admin/users` | GET | List users |
| `/api/v1/admin/users/{id}` | GET/PUT/DELETE | Manage user |
| `/api/v1/admin/backup` | POST | Create backup |
| `/api/v1/admin/restore` | POST | Restore backup |

## Subsonic API

Endpoint: `/subsonic/rest/`

Full compatibility with Subsonic API v1.16.1

### Example

```bash
curl "http://localhost/subsonic/rest/ping.view?u=user&p=pass&v=1.16.1&c=myclient&f=json"
```

## Ampache API

Endpoint: `/ampache/server/`

Full compatibility with Ampache API v6.0.0

### Example

```bash
curl "http://localhost/ampache/server/json.server.php?action=handshake&auth=xxx&version=6.0.0"
```

## MPD Protocol

Port: 6600

Standard MPD protocol v0.23.5 over TCP

### Example

```bash
nc localhost 6600
status
currentsong
```

## WebDAV

Endpoint: `/webdav/`

Standard WebDAV protocol for file access

### Example

```bash
curl -u user:pass http://localhost/webdav/music/
```

## RTMP Streaming

Port: 1935

### Broadcast URL

```
rtmp://localhost/live/{stream_key}
```

### Playback URL

```
rtmp://localhost/live/{mount_point}
```
