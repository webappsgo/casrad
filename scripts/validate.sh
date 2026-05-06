#!/bin/bash
# CASRAD Comprehensive Validation Script
# Validates all requirements from CLAUDE.md specification

set -e

CASRAD_URL="${CASRAD_URL:-http://localhost:64000}"
PASSED=0
FAILED=0

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           CASRAD Specification Validation Script            ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Helper functions
pass() {
    echo "✓ $1"
    ((PASSED++))
}

fail() {
    echo "✗ $1"
    ((FAILED++))
}

check_http() {
    local url="$1"
    local desc="$2"
    if curl -sf "$url" > /dev/null 2>&1; then
        pass "$desc"
    else
        fail "$desc"
    fi
}

check_port() {
    local port="$1"
    local desc="$2"
    if nc -z localhost "$port" 2>/dev/null; then
        pass "$desc"
    else
        fail "$desc"
    fi
}

echo "=== Core Functionality ==="

# Check if CASRAD is running
check_http "$CASRAD_URL" "Web server responding"

# Check protocols
check_port 6600 "MPD server on port 6600"
check_port 1935 "RTMP server on port 1935"

# Check API endpoints
check_http "$CASRAD_URL/api/v1/health" "Health endpoint"
check_http "$CASRAD_URL/subsonic/rest/ping.view" "Subsonic API"
check_http "$CASRAD_URL/ampache/server/xml.server.php" "Ampache API"

echo ""
echo "=== Web UI ===" 

# Check main pages
check_http "$CASRAD_URL/" "Home page"
check_http "$CASRAD_URL/static/css/dracula.css" "Dracula theme CSS"
check_http "$CASRAD_URL/static/js/casrad.js" "Main JS file"

echo ""
echo "=== Results ===" 
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
