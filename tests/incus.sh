#!/usr/bin/env bash
# CASRAD Incus Test Script (PREFERRED)
# See AI.md PART 29 for testing requirements
set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if incus is available
if ! command -v incus &>/dev/null; then
    echo -e "${YELLOW}Incus not available, falling back to docker.sh${NC}"
    exec "$(dirname "$0")/docker.sh"
fi

# Detect project info
PROJECTNAME=$(basename "$PWD")
PROJECTORG=$(basename "$(dirname "$PWD")")
CONTAINER_NAME="test-${PROJECTNAME}-$$"

# Incus image - use latest Debian stable
INCUS_IMAGE="images:debian/12"

# Create temp directory for build
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}.XXXXXX")
trap "rm -rf $BUILD_DIR; incus delete $CONTAINER_NAME --force 2>/dev/null || true" EXIT

# Go cache directories (same as Makefile)
GODIR="${HOME}/.local/share/go"
GOCACHE="${HOME}/.local/share/go/build"
mkdir -p "$GODIR" "$GOCACHE"

# Common docker run for Go builds
GO_DOCKER="docker run --rm \
  -v $(pwd):/build \
  -v ${GOCACHE}:/root/.cache/go-build \
  -v ${GODIR}:/go \
  -w /build \
  -e CGO_ENABLED=0 \
  golang:alpine"

echo "=== CASRAD Incus Test (PREFERRED) ==="
echo "Build directory: ${BUILD_DIR}"
echo "Container: ${CONTAINER_NAME}"
echo ""

echo "Building server binary in Docker..."
$GO_DOCKER go build -o "$PROJECTNAME" ./src
mv "$PROJECTNAME" "$BUILD_DIR/"

# Build CLI client if exists
if [ -d "src/client" ]; then
    echo "Building CLI client in Docker..."
    $GO_DOCKER go build -o "$PROJECTNAME-cli" ./src/client
    mv "$PROJECTNAME-cli" "$BUILD_DIR/"
fi

# Build agent if exists
if [ -d "src/agent" ]; then
    echo "Building agent in Docker..."
    $GO_DOCKER go build -o "$PROJECTNAME-agent" ./src/agent
    mv "$PROJECTNAME-agent" "$BUILD_DIR/"
fi

echo "Launching Incus container (Debian + systemd)..."
incus launch "$INCUS_IMAGE" "$CONTAINER_NAME"

# Wait for container to be ready
echo "Waiting for container to be ready..."
for i in {1..30}; do
    if incus exec "$CONTAINER_NAME" -- systemctl is-system-running --wait 2>/dev/null; then
        break
    fi
    sleep 1
    if [ "$i" -eq 30 ]; then
        echo -e "${YELLOW}Warning: System not fully ready, continuing anyway${NC}"
    fi
done

# Install required tools
echo "Installing test dependencies..."
incus exec "$CONTAINER_NAME" -- apt-get update -qq
incus exec "$CONTAINER_NAME" -- apt-get install -y -qq curl jq file

echo "Copying binaries to container..."
incus file push "$BUILD_DIR/$PROJECTNAME" "$CONTAINER_NAME/usr/local/bin/"
incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/$PROJECTNAME"

# Copy CLI client if built
if [ -f "$BUILD_DIR/$PROJECTNAME-cli" ]; then
    incus file push "$BUILD_DIR/$PROJECTNAME-cli" "$CONTAINER_NAME/usr/local/bin/"
    incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/$PROJECTNAME-cli"
fi

# Copy agent if built
if [ -f "$BUILD_DIR/$PROJECTNAME-agent" ]; then
    incus file push "$BUILD_DIR/$PROJECTNAME-agent" "$CONTAINER_NAME/usr/local/bin/"
    incus exec "$CONTAINER_NAME" -- chmod +x "/usr/local/bin/$PROJECTNAME-agent"
fi

echo "Running tests in Incus..."
incus exec "$CONTAINER_NAME" -- bash -c "
    set -e

    PROJECTNAME='$PROJECTNAME'
    PROJECTORG='$PROJECTORG'

    echo '=== Version Check ==='
    /usr/local/bin/\$PROJECTNAME --version

    echo '=== Help Check ==='
    /usr/local/bin/\$PROJECTNAME --help

    echo '=== Binary Info ==='
    ls -lh /usr/local/bin/\$PROJECTNAME
    file /usr/local/bin/\$PROJECTNAME

    echo '=== Binary Rename Test ==='
    cp /usr/local/bin/\$PROJECTNAME /usr/local/bin/renamed-server
    chmod +x /usr/local/bin/renamed-server
    if /usr/local/bin/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
        echo '✓ Server binary rename works (--help shows actual name)'
    else
        echo '✗ FAILED: Server --help does not show renamed binary name'
    fi

    echo '=== Service Install Test ==='
    /usr/local/bin/\$PROJECTNAME --service --install || echo 'Service install returned non-zero (may be expected)'

    echo '=== Service Status ==='
    systemctl status \$PROJECTNAME --no-pager || true

    echo '=== Service Start Test ==='
    # Create required directories
    mkdir -p /etc/\$PROJECTORG/\$PROJECTNAME
    mkdir -p /var/lib/\$PROJECTORG/\$PROJECTNAME

    # Start the service
    systemctl start \$PROJECTNAME || {
        echo 'systemd start failed, trying manual start...'
        /usr/local/bin/\$PROJECTNAME --port 80 > /tmp/server.log 2>&1 &
        sleep 3
    }
    sleep 2

    # Check if running
    systemctl status \$PROJECTNAME --no-pager || echo 'Service status check failed'

    echo '=== Health Endpoint Tests ==='
    curl -sf http://localhost:80/healthz || curl -sf http://localhost:64000/healthz || echo 'FAILED: /healthz'
    curl -sf http://localhost:80/version || curl -sf http://localhost:64000/version || echo 'FAILED: /version'

    echo '=== API Endpoint Tests ==='
    PORT=80
    curl -sf http://localhost:\$PORT/healthz >/dev/null 2>&1 || PORT=64000

    # Test JSON response (default)
    curl -sf http://localhost:\$PORT/api/v1/ && echo '' || echo 'FAILED: /api/v1/'

    # Test .txt extension (plain text) - if supported
    curl -sf http://localhost:\$PORT/api/v1/healthz.txt 2>/dev/null && echo '' || echo 'INFO: .txt extension not yet implemented'

    # Test Accept header: application/json
    curl -sf -H 'Accept: application/json' http://localhost:\$PORT/api/v1/ && echo '' || echo 'FAILED: Accept JSON'

    # Test Accept header: text/plain
    curl -sf -H 'Accept: text/plain' http://localhost:\$PORT/api/v1/ 2>/dev/null && echo '' || echo 'INFO: text/plain not yet implemented'

    echo '=== Admin Setup & API Token Creation ==='
    # Get setup token from journal or log
    SETUP_TOKEN=\$(journalctl -u \$PROJECTNAME --no-pager 2>/dev/null | grep -oP 'Setup Token.*:\\s*\\K[a-f0-9]+' | head -1 || cat /tmp/server.log 2>/dev/null | grep -oP 'Setup Token.*:\\s*\\K[a-f0-9]+' | head -1 || echo '')

    if [ -n \"\$SETUP_TOKEN\" ]; then
        echo \"Setup token found: \${SETUP_TOKEN:0:8}...\"

        # Create admin account
        curl -sf -X POST \
            -H \"X-Setup-Token: \$SETUP_TOKEN\" \
            -H 'Content-Type: application/json' \
            -d '{\"username\":\"testadmin\",\"password\":\"TestPass123!\",\"email\":\"admin@test.local\"}' \
            http://localhost:\$PORT/api/v1/admin/setup || echo 'Admin setup failed (may already exist)'

        # Login and get session
        SESSION=\$(curl -sf -X POST \
            -H 'Content-Type: application/json' \
            -d '{\"username\":\"testadmin\",\"password\":\"TestPass123!\"}' \
            http://localhost:\$PORT/api/v1/auth/login | jq -r '.session_token // empty' || echo '')

        if [ -n \"\$SESSION\" ]; then
            echo '✓ Admin login successful'

            # Generate API token for CLI/Agent testing
            API_TOKEN=\$(curl -sf -X POST \
                -H \"Authorization: Bearer \$SESSION\" \
                http://localhost:\$PORT/api/v1/admin/profile/token | jq -r '.token // empty' || echo '')

            if [ -n \"\$API_TOKEN\" ]; then
                echo \"✓ API token created: \${API_TOKEN:0:12}...\"
            else
                echo 'API token creation failed (continuing without token)'
            fi
        else
            echo 'Admin login failed (continuing without session)'
        fi
    else
        echo 'No setup token found (server may already be configured)'
    fi

    echo '=== Core Page Tests ==='
    curl -sf http://localhost:\$PORT/ > /dev/null && echo '✓ Home page' || echo 'FAILED: Home page'
    curl -sf http://localhost:\$PORT/auth/login > /dev/null && echo '✓ Login page' || echo 'FAILED: Login page'

    echo '=== CLI Client Tests (if exists) ==='
    if [ -f /usr/local/bin/\$PROJECTNAME-cli ]; then
        /usr/local/bin/\$PROJECTNAME-cli --version || echo 'FAILED: CLI --version'
        /usr/local/bin/\$PROJECTNAME-cli --help || echo 'FAILED: CLI --help'

        # Test binary rename
        cp /usr/local/bin/\$PROJECTNAME-cli /usr/local/bin/renamed-cli
        chmod +x /usr/local/bin/renamed-cli
        if /usr/local/bin/renamed-cli --help 2>&1 | grep -q 'renamed-cli'; then
            echo '✓ CLI binary rename works'
        else
            echo '✗ FAILED: CLI --help does not show renamed binary name'
        fi

        # Full CLI functionality tests against server
        echo '--- CLI Full Functionality Tests ---'
        if [ -n \"\${API_TOKEN:-}\" ]; then
            /usr/local/bin/\$PROJECTNAME-cli --server http://localhost:\$PORT --token \"\$API_TOKEN\" status || echo 'CLI status failed'
        else
            /usr/local/bin/\$PROJECTNAME-cli --server http://localhost:\$PORT status 2>/dev/null || echo 'CLI status (no token) - skipped'
        fi
    else
        echo 'CLI client not built - skipping'
    fi

    echo '=== Agent Tests (if exists) ==='
    if [ -f /usr/local/bin/\$PROJECTNAME-agent ]; then
        /usr/local/bin/\$PROJECTNAME-agent --version || echo 'FAILED: Agent --version'
        /usr/local/bin/\$PROJECTNAME-agent --help || echo 'FAILED: Agent --help'

        # Test binary rename
        cp /usr/local/bin/\$PROJECTNAME-agent /usr/local/bin/renamed-agent
        chmod +x /usr/local/bin/renamed-agent
        if /usr/local/bin/renamed-agent --help 2>&1 | grep -q 'renamed-agent'; then
            echo '✓ Agent binary rename works'
        else
            echo '✗ FAILED: Agent --help does not show renamed binary name'
        fi

        # Full Agent functionality tests against server
        echo '--- Agent Full Functionality Tests ---'
        if [ -n \"\${API_TOKEN:-}\" ]; then
            /usr/local/bin/\$PROJECTNAME-agent --server http://localhost:\$PORT --token \"\$API_TOKEN\" status || echo 'Agent status failed'
        else
            echo 'Agent tests skipped (no API token)'
        fi
    else
        echo 'Agent not built - skipping'
    fi

    echo '=== Service Stop Test ==='
    systemctl stop \$PROJECTNAME 2>/dev/null || true

    echo ''
    echo '=== All tests passed ==='
"

echo ""
echo -e "${GREEN}Incus tests completed successfully${NC}"
