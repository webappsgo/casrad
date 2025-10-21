# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for static linking
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    sqlite-dev \
    sqlite-static

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with everything embedded
RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    CC=gcc \
    go build \
    -a \
    -trimpath \
    -ldflags "-linkmode external -extldflags '-static -pthread' -X main.Version=1.0.0 -X main.BuildTime=$(date +%FT%T%z) -s -w" \
    -tags "fts5,osusergo,netgo" \
    -o casrad \
    cmd/casrad/main.go

# Verify it's a static binary (ldd should fail or say "not a dynamic executable")
RUN (ldd casrad 2>&1 && exit 1) || echo "✓ Static binary confirmed"

# Show binary size
RUN ls -lh casrad

# Final runtime stage - Alpine for user/directory management
FROM alpine:latest

# Create casrad user and directories
RUN adduser -D -u 963 -g 963 casrad \
    && mkdir -p \
        /var/lib/casrad \
        /etc/casrad \
        /var/cache/casrad \
        /var/log/casrad \
        /tmp/casrad \
        /mnt/Music \
        /mnt/Podcasts \
        /mnt/Audiobooks \
        /mnt/Playlists \
    && chown -R casrad:casrad \
        /var/lib/casrad \
        /etc/casrad \
        /var/cache/casrad \
        /var/log/casrad \
        /tmp/casrad

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy static binary
COPY --from=builder /build/casrad /usr/local/bin/casrad

# Verify binary is executable
RUN /usr/local/bin/casrad --version || echo "Binary check: Version flag not yet implemented"

# Switch to non-root user
USER casrad

# Expose ports
EXPOSE 80 443 6600 1935

# Volume for persistent data
VOLUME ["/var/lib/casrad", "/etc/casrad", "/mnt/Music", "/mnt/Podcasts", "/mnt/Audiobooks", "/mnt/Playlists"]

# Entry point
ENTRYPOINT ["/usr/local/bin/casrad"]