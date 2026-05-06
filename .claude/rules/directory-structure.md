# Directory Structure Rules

@AI.md PART 3: PROJECT STRUCTURE

## Required Structure
- `src/` - ALL Go code (server, client, agent)
- `src/client/` - CLI client (if exists)
- `src/agent/` - Agent (if exists)
- `docker/` - Dockerfile, docker-compose, rootfs
- `tests/` - Test scripts (run_tests.sh, docker.sh, incus.sh)
- `binaries/` - Build output (gitignored)

## Forbidden
- NO `cmd/` directory (use `src/`)
- NO `internal/` (everything in `src/`)
- NO `pkg/` (not a library)
- NO root-level `data/`, `config/`, `logs/`
- NO CHANGELOG.md (use GitHub releases)
- NO AUDIT.md, REPORT.md, ANALYSIS.md

## Allowed Root Files
- AI.md, IDEA.md, CLAUDE.md, README.md, LICENSE.md
- Makefile, go.mod, go.sum, release.txt
- .gitignore, .dockerignore, mkdocs.yml
- TODO.AI.md, PLAN.AI.md (if needed)

## File Naming
- Lowercase only, snake_case for multi-word
- Singular directory names (handler/, model/, service/)
- Match package name (config/config.go, server/server.go)
