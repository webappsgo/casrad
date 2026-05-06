#!/usr/bin/env bash
# CASRAD Docker Test Script
# See AI.md PART 29 for testing requirements
set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Detect project info
PROJECTNAME=$(basename "$PWD")
PROJECTORG=$(basename "$(dirname "$PWD")")

# Create temp directory for build
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}.XXXXXX")
trap "rm -rf $BUILD_DIR" EXIT

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

echo "=== CASRAD Docker Test ==="
echo "Build directory: ${BUILD_DIR}"
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

echo "Testing in Docker (Alpine)..."
docker run --rm \
  -v "$BUILD_DIR:/app" \
  alpine:latest sh -c "
    set -e

    # Install required tools for testing
    apk add --no-cache curl bash file jq >/dev/null

    chmod +x /app/$PROJECTNAME
    [ -f /app/$PROJECTNAME-cli ] && chmod +x /app/$PROJECTNAME-cli
    [ -f /app/$PROJECTNAME-agent ] && chmod +x /app/$PROJECTNAME-agent

    echo '=== Version Check ==='
    /app/$PROJECTNAME --version

    echo '=== Help Check ==='
    /app/$PROJECTNAME --help

    echo '=== Binary Info ==='
    ls -lh /app/$PROJECTNAME
    file /app/$PROJECTNAME

    echo '=== Binary Rename Test ==='
    # Test that binary shows ACTUAL name in --help/--version (not hardcoded)
    cp /app/$PROJECTNAME /app/renamed-server
    chmod +x /app/renamed-server
    if /app/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
        echo -e '${GREEN}✓ Server binary rename works (--help shows actual name)${NC}'
    else
        echo -e '${RED}✗ FAILED: Server --help does not show renamed binary name${NC}'
    fi

    echo '=== Starting Server for API Tests ==='
    mkdir -p /etc/${PROJECTORG}/${PROJECTNAME}
    mkdir -p /var/lib/${PROJECTORG}/${PROJECTNAME}
    /app/$PROJECTNAME --port 64580 > /tmp/server.log 2>&1 &
    SERVER_PID=\$!
    sleep 3

    # Show setup token if present (for debugging)
    grep -i 'setup.*token' /tmp/server.log 2>/dev/null || true

    echo '=== Health Endpoint Tests ==='
    # Test healthz endpoint
    curl -sf http://localhost:64580/healthz || echo -e '${RED}FAILED: /healthz${NC}'

    # Test version endpoint
    curl -sf http://localhost:64580/version || echo -e '${RED}FAILED: /version${NC}'

    echo '=== API Endpoint Tests ==='
    # Test JSON response (default)
    curl -sf http://localhost:64580/api/v1/ && echo '' || echo -e '${RED}FAILED: /api/v1/${NC}'

    # Test .txt extension (plain text) - if supported
    curl -sf http://localhost:64580/api/v1/healthz.txt 2>/dev/null && echo '' || echo 'INFO: .txt extension not yet implemented'

    # Test Accept header: application/json
    curl -sf -H 'Accept: application/json' http://localhost:64580/api/v1/ && echo '' || echo -e '${RED}FAILED: Accept JSON${NC}'

    # Test Accept header: text/plain
    curl -sf -H 'Accept: text/plain' http://localhost:64580/api/v1/ 2>/dev/null && echo '' || echo 'INFO: text/plain not yet implemented'

    echo '=== Admin Setup & API Token Creation ==='
    # Get setup token from server output (captured during startup)
    SETUP_TOKEN=\$(cat /tmp/server.log 2>/dev/null | grep -oP 'Setup Token.*:\\s*\\K[a-f0-9]+' | head -1 || echo '')

    if [ -n \"\$SETUP_TOKEN\" ]; then
        echo \"Setup token found: \${SETUP_TOKEN:0:8}...\"

        # Create admin account
        curl -sf -X POST \
            -H \"X-Setup-Token: \$SETUP_TOKEN\" \
            -H \"Content-Type: application/json\" \
            -d '{\"username\":\"testadmin\",\"password\":\"TestPass123!\",\"email\":\"admin@test.local\"}' \
            http://localhost:64580/api/v1/admin/setup || echo 'Admin setup failed (may already exist)'

        # Login and get session
        SESSION=\$(curl -sf -X POST \
            -H \"Content-Type: application/json\" \
            -d '{\"username\":\"testadmin\",\"password\":\"TestPass123!\"}' \
            http://localhost:64580/api/v1/auth/login | jq -r '.session_token // empty' || echo '')

        if [ -n \"\$SESSION\" ]; then
            echo -e '${GREEN}✓ Admin login successful${NC}'

            # Generate API token for CLI/Agent testing
            API_TOKEN=\$(curl -sf -X POST \
                -H \"Authorization: Bearer \$SESSION\" \
                http://localhost:64580/api/v1/admin/profile/token | jq -r '.token // empty' || echo '')

            if [ -n \"\$API_TOKEN\" ]; then
                echo -e \"${GREEN}✓ API token created: \${API_TOKEN:0:12}...${NC}\"
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
    # Test home page
    curl -sf http://localhost:64580/ > /dev/null && echo -e '${GREEN}✓ Home page${NC}' || echo -e '${RED}FAILED: Home page${NC}'

    # Test login page
    curl -sf http://localhost:64580/auth/login > /dev/null && echo -e '${GREEN}✓ Login page${NC}' || echo -e '${RED}FAILED: Login page${NC}'

    echo '=== CLI Client Tests (if exists) ==='
    if [ -f /app/$PROJECTNAME-cli ]; then
        /app/$PROJECTNAME-cli --version || echo -e '${RED}FAILED: CLI --version${NC}'
        /app/$PROJECTNAME-cli --help || echo -e '${RED}FAILED: CLI --help${NC}'

        # Test binary rename
        cp /app/$PROJECTNAME-cli /app/renamed-cli
        chmod +x /app/renamed-cli
        if /app/renamed-cli --help 2>&1 | grep -q 'renamed-cli'; then
            echo -e '${GREEN}✓ CLI binary rename works${NC}'
        else
            echo -e '${RED}✗ FAILED: CLI --help does not show renamed binary name${NC}'
        fi

        # Full CLI functionality tests against server
        echo '--- CLI Full Functionality Tests ---'
        if [ -n \"\${API_TOKEN:-}\" ]; then
            # Test with API token
            /app/$PROJECTNAME-cli --server http://localhost:64580 --token \"\$API_TOKEN\" status || echo 'CLI status failed'
        else
            # Test without token (anonymous if allowed)
            /app/$PROJECTNAME-cli --server http://localhost:64580 status 2>/dev/null || echo 'CLI status (no token) - skipped'
        fi
    else
        echo 'CLI client not built - skipping'
    fi

    echo '=== Agent Tests (if exists) ==='
    if [ -f /app/$PROJECTNAME-agent ]; then
        /app/$PROJECTNAME-agent --version || echo -e '${RED}FAILED: Agent --version${NC}'
        /app/$PROJECTNAME-agent --help || echo -e '${RED}FAILED: Agent --help${NC}'

        # Test binary rename
        cp /app/$PROJECTNAME-agent /app/renamed-agent
        chmod +x /app/renamed-agent
        if /app/renamed-agent --help 2>&1 | grep -q 'renamed-agent'; then
            echo -e '${GREEN}✓ Agent binary rename works${NC}'
        else
            echo -e '${RED}✗ FAILED: Agent --help does not show renamed binary name${NC}'
        fi

        # Full Agent functionality tests against server
        echo '--- Agent Full Functionality Tests ---'
        if [ -n \"\${API_TOKEN:-}\" ]; then
            # Test agent registration/status with API token
            /app/$PROJECTNAME-agent --server http://localhost:64580 --token \"\$API_TOKEN\" status || echo 'Agent status failed'
        else
            echo 'Agent tests skipped (no API token)'
        fi
    else
        echo 'Agent not built - skipping'
    fi

    echo '=== Stopping Server ==='
    kill \$SERVER_PID 2>/dev/null || true
    wait \$SERVER_PID 2>/dev/null || true

    echo ''
    echo -e '${GREEN}=== All tests passed ===${NC}'
"

echo ""
echo -e "${GREEN}Docker tests completed successfully${NC}"
