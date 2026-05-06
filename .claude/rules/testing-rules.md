# Testing Rules

@AI.md PART 29: TESTING & DEVELOPMENT

## Test Scripts
- `./tests/run_tests.sh` - Auto-detects incus/docker
- `./tests/docker.sh` - Docker alpine (quick)
- `./tests/incus.sh` - Incus debian (PREFERRED)

## Tests MUST Include
- Admin setup (setup token -> create admin -> API token)
- Binary rename test (verify --help shows actual name)
- CLI full functionality (with API token)
- Agent full functionality (with API token)
- API endpoint tests (.txt extension, Accept headers)

## Debug Container Tools
`apk add --no-cache curl bash file jq`

## AI as Beta Tester
When AI tests, it acts as a beta tester:
- Find bugs: Try edge cases, invalid inputs
- Break it: Stress test, race conditions
- Fix it: Don't just report - implement the fix
- Verify fix: Re-test to confirm

## Container-Only Development
- NEVER run Go or binaries on host
- ALL development uses containers
- Consistent environment (same as CI/CD and production)
