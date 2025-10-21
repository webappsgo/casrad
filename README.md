# CASRAD - Complete Audio Streaming, Radio, and Distribution

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-green.svg)](https://www.docker.com/)

CASRAD is a revolutionary single-binary audio streaming and broadcasting server that consolidates the functionality of 50+ specialized servers into one self-contained, zero-configuration solution.

## ✨ Key Features

- **Single Binary**: ~40-50MB static binary containing everything
- **Zero Configuration**: Works immediately upon execution
- **6 Protocol Support**: MPD, Subsonic, Ampache, WebDAV, RTMP, DLNA
- **Beautiful UI**: Dracula-themed interface with setup wizard
- **Per-User Storage**: Isolated directories with quotas
- **Auto Everything**: Service installation, SSL certificates, FFMPEG download
- **Enterprise Ready**: Scales from personal to datacenter deployment

## 🚀 Quick Start

### Option 1: Download and Run
```bash
wget https://github.com/casapps/casrad/releases/latest/download/casrad
chmod +x casrad
./casrad
```

### Option 2: Docker
```bash
docker run -d \
  -p 80:80 \
  -p 6600:6600 \
  -p 1935:1935 \
  -v /path/to/music:/mnt/Music/Mp3:ro \
  -v casrad-data:/var/lib/casrad \
  ghcr.io/casapps/casrad:latest
```

Open your browser to `http://localhost` and complete the 2-minute setup wizard!

## 🎵 Protocol Support

| Protocol | Port | Compatible Clients |
|----------|------|-------------------|
| **MPD** | 6600 | ncmpcpp, Cantata, MPDroid, Maximum MPD |
| **Subsonic** | HTTP | DSub, Ultrasonic, play:Sub, Sonixd |
| **Ampache** | HTTP | PowerAmpache, Ampache Player |
| **WebDAV** | HTTP | Any WebDAV client |
| **RTMP** | 1935 | OBS, FFmpeg, VLC |
| **DLNA** | 1900 | Smart TVs, Media Players |

## 🔒 Security

All security features enabled by default:
- Argon2id password hashing
- Rate limiting (60 req/min)
- Brute force protection
- Session management
- GeoIP integration (P3TERX)
- Automatic security updates

## 📁 Directory Structure

- `/etc/casrad/` - Configuration (privileged)
- `/var/lib/casrad/` - Data and users (privileged)
- `~/.local/share/casrad/` - User mode data
- `/mnt/Music/Mp3/` - Default music directory

## Author

🤖 casjay: [Github](https://github.com/casjay) 🤖

**One Binary. No Dependencies. No Configuration. Just Works.™**  
