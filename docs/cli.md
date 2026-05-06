# CLI Reference

CASRAD has minimal command line options by design. All configuration is managed through the web UI.

## Basic Usage

```bash
casrad [flags]
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--help` | `-h` | Show help | - |
| `--version` | `-v` | Show version | - |
| `--port` | `-p` | Override port | Auto (64000-64999 or 80) |
| `--address` | `-a` | Bind address | 0.0.0.0 |
| `--data` | `-d` | Data directory | OS-specific |
| `--debug` | | Enable debug logging | false |

## Service Management

| Flag | Description |
|------|-------------|
| `--service install` | Install as system service |
| `--service uninstall` | Remove system service |
| `--service start` | Start service |
| `--service stop` | Stop service |
| `--service restart` | Restart service |
| `--service status` | Show service status |
| `--status` | Show server status |

## Examples

```bash
# Run with defaults
./casrad

# Run on specific port
./casrad --port 8080

# Run with debug logging
./casrad --debug

# Install as service
sudo ./casrad --service install

# Show version
./casrad --version
```

## Output

### Version Output

```
casrad v1.0.0 (abc1234) built 2024-01-01T12:00:00Z
```

### Status Output

```
CASRAD Status:
  Running: yes
  Port: 80
  Uptime: 5d 12h 30m
  Users: 42
  Streams: 15
  CPU: 12%
  Memory: 256MB
```
