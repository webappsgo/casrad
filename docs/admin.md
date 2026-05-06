# Admin Panel

## Access

- **URL**: `/admin` (configurable)
- **First-run**: Setup wizard creates admin account
- **Authentication**: Session-based with optional 2FA

## Dashboard

The admin dashboard provides real-time overview:

- Active listeners and streams
- System resource usage (CPU, Memory, Disk)
- Recent activity feed
- Quick actions

## Features

### User Management

- Create, edit, delete users
- Set storage quotas
- Manage permissions
- Reset passwords
- View user activity

### Library Management

- Scan music directories
- Edit metadata
- MusicBrainz tagging
- Cover art management
- Duplicate detection

### Storage Management

- Monitor disk usage
- Configure global directories
- Set default quotas
- Clean up old files

### Protocol Configuration

- Enable/disable protocols
- Configure ports
- Set protocol-specific options

### Broadcasting

- Manage mount points
- AutoDJ configuration
- Stream statistics
- Recording settings

### Podcasts

- Add/remove feeds
- Download settings
- Update schedules

### Security

- View active sessions
- Manage API tokens
- Audit logs
- GeoIP blocking
- Rate limit settings

### Backup & Restore

- Create manual backups
- Configure auto-backup
- Restore from backup
- Export/import data

### SSL/TLS

- Let's Encrypt integration
- Custom certificates
- Certificate status
- Auto-renewal settings

### Logs

- Server logs
- Access logs
- Error logs
- Search and filter

## Admin API

Programmatic access via `/api/v1/admin/` endpoints.

### Authentication

```bash
# Generate API token in admin panel
curl -H "Authorization: Bearer <token>" \
  http://localhost/api/v1/admin/server/settings
```

### Common Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/admin/server/settings` | GET/PUT | Server configuration |
| `/api/v1/admin/server/logs` | GET | View logs |
| `/api/v1/admin/users` | GET/POST | User management |
| `/api/v1/admin/backup` | POST | Create backup |
| `/api/v1/admin/library/scan` | POST | Trigger scan |
