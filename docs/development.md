# Development Guide

## Prerequisites

- Go 1.21+
- Make
- Docker (for builds and testing)
- Incus (optional, for full integration tests)

## Build

```bash
git clone https://github.com/casapps/casrad
cd casrad

# Development build
make dev

# Production build (current platform)
make host

# Full release (all 8 platforms)
make build
```

## Testing

```bash
# Unit tests
make test

# Integration tests
./tests/run_tests.sh

# Docker testing
./tests/docker.sh

# Incus testing (with systemd)
./tests/incus.sh
```

## Project Structure

```
casrad/
├── src/                    # Go source code
│   ├── main.go
│   ├── config/             # Configuration
│   ├── server/             # HTTP server
│   │   ├── handler/        # Request handlers
│   │   ├── service/        # Business logic
│   │   ├── model/          # Data models
│   │   ├── store/          # Database layer
│   │   ├── template/       # HTML templates
│   │   └── static/         # Static assets
│   ├── mode/               # Runtime modes
│   ├── paths/              # Path resolution
│   └── ssl/                # SSL certificates
├── docker/                 # Docker files
│   ├── Dockerfile
│   ├── Dockerfile.aio
│   ├── docker-compose.yml
│   └── rootfs/             # Container filesystem
├── tests/                  # Test scripts
├── docs/                   # Documentation (MkDocs)
├── Makefile
├── go.mod
└── go.sum
```

## Code Style

- Follow Go standard formatting (`go fmt`)
- Use meaningful variable names
- Add comments for complex logic
- Keep functions focused and small

## Testing Requirements

- 100% code coverage for Go unit tests
- 100% endpoint coverage for integration tests
- Test both happy path and error cases

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/name`)
3. Make changes with tests
4. Run `make test` and `./tests/run_tests.sh`
5. Commit changes
6. Push to your fork
7. Submit pull request

## Release Process

1. Update version in `release.txt`
2. Create git tag (`v1.0.0` or `1.0.0`)
3. Push tag to trigger release workflow
4. GitHub Actions builds and publishes

## Docker Images

| Image | Description |
|-------|-------------|
| `ghcr.io/casapps/casrad:latest` | Standard image |
| `ghcr.io/casapps/casrad-aio:latest` | All-in-One with PostgreSQL, Valkey, Tor |
| `ghcr.io/casapps/casrad:devel` | Development builds |
| `ghcr.io/casapps/casrad:beta` | Beta releases |
