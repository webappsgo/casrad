# CASRAD

**Complete Audio Streaming, Radio, and Distribution Server**

A revolutionary single-binary audio streaming and broadcasting server that consolidates 50+ different specialized servers into one self-contained, zero-configuration solution.

## Quick Start

```bash
# Docker
docker run -p 80:80 -v casrad-data:/data ghcr.io/casapps/casrad:latest

# Binary
./casrad-linux-amd64
```

## Features

- **Single Binary Deployment**: ~40-50MB static binary containing everything
- **Zero Configuration**: Works immediately upon execution
- **Self-Installing**: Automatically installs as system service
- **Protocol Complete**: MPD, Subsonic, Ampache, WebDAV, RTMP, DLNA
- **Enterprise Ready**: Scales from personal to datacenter deployment
- **Beautiful UI**: Dracula-inspired dark theme (default) and clean light theme
- **Per-User Storage**: Isolated user directories with quotas
- **Web-Based Management**: Everything managed through comprehensive admin UI

## Supported Protocols

| Protocol | Port | Description |
|----------|------|-------------|
| HTTP/HTTPS | 80/443 | Web UI, REST API, Subsonic, Ampache |
| MPD | 6600 | Music Player Daemon protocol |
| RTMP | 1935 | Live streaming |
| DLNA | 1900 | Media discovery |
| WebDAV | 80 | File access |

## Documentation

- [Installation](installation.md) - How to install and run
- [Configuration](configuration.md) - All configuration options
- [API Reference](api.md) - REST API, Swagger, GraphQL
- [CLI Reference](cli.md) - Command line options
- [Admin Panel](admin.md) - Web UI administration
- [Development](development.md) - Contributing guide

## Links

- [Repository](https://github.com/casapps/casrad)
- [Container Registry](https://ghcr.io/casapps/casrad)

## License

MIT - See [LICENSE.md](https://github.com/casapps/casrad/blob/main/LICENSE.md)
