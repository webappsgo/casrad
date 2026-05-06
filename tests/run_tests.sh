#!/bin/bash
# CASRAD Test Runner - Auto-detects incus/docker
# See AI.md PART 29 for testing requirements

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=== CASRAD Test Runner ==="
echo ""

# Auto-detect container runtime
if command -v incus &>/dev/null; then
    echo -e "${GREEN}Detected: Incus (PREFERRED)${NC}"
    exec "${SCRIPT_DIR}/incus.sh" "$@"
elif command -v docker &>/dev/null; then
    echo -e "${YELLOW}Detected: Docker (fallback)${NC}"
    exec "${SCRIPT_DIR}/docker.sh" "$@"
else
    echo -e "${RED}ERROR: No container runtime found!${NC}"
    echo "Please install Incus (PREFERRED) or Docker."
    echo ""
    echo "Incus installation: https://linuxcontainers.org/incus/docs/main/installing/"
    echo "Docker installation: https://docs.docker.com/get-docker/"
    exit 1
fi
