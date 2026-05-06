# Installation

## Docker (Recommended)

### Standard Image

```bash
docker run -d \
  --name casrad \
  -p 80:80 \
  -p 6600:6600 \
  -p 1935:1935 \
  -v /mnt/Music:/mnt/Music:ro \
  -v casrad-data:/data \
  ghcr.io/casapps/casrad:latest
```

### All-in-One Image (with PostgreSQL, Valkey, Tor)

```bash
docker run -d \
  --name casrad \
  -p 80:80 \
  -p 6600:6600 \
  -p 1935:1935 \
  -v /mnt/Music:/mnt/Music:ro \
  -v casrad-config:/config/casrad \
  -v casrad-data:/data/casrad \
  -v casrad-db:/data/db \
  ghcr.io/casapps/casrad-aio:latest
```

### Docker Compose

```bash
# Standard
docker compose -f docker-compose.yml up -d

# All-in-One
docker compose -f all-in-one.yml up -d
```

## Binary

Download from [releases](https://github.com/casapps/casrad/releases):

```bash
# Linux AMD64
wget https://github.com/casapps/casrad/releases/latest/download/casrad-linux-amd64
chmod +x casrad-linux-amd64
./casrad-linux-amd64
```

### Available Platforms

| Platform | Binary Name |
|----------|-------------|
| Linux AMD64 | `casrad-linux-amd64` |
| Linux ARM64 | `casrad-linux-arm64` |
| macOS AMD64 | `casrad-darwin-amd64` |
| macOS ARM64 | `casrad-darwin-arm64` |
| Windows AMD64 | `casrad-windows-amd64.exe` |
| Windows ARM64 | `casrad-windows-arm64.exe` |
| FreeBSD AMD64 | `casrad-freebsd-amd64` |
| FreeBSD ARM64 | `casrad-freebsd-arm64` |

## Systemd Service

```bash
# Install as system service
sudo ./casrad --service install

# Start the service
sudo systemctl start casrad

# Enable auto-start on boot
sudo systemctl enable casrad

# Check status
sudo systemctl status casrad
```

## First Run

1. Start CASRAD
2. Open browser to `http://localhost` (or configured port)
3. Complete the setup wizard
4. Create admin account
5. Configure music directories
6. Start streaming!

## Configuration

See [Configuration](configuration.md) for all options.
