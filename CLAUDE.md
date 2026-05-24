# Project SPEC

## Project Identity

| Field | Value |
|-------|-------|
| **Name** | casrad |
| **Org** | casapps |
| **Internal Name** | casrad (frozen) |
| **Module** | github.com/casapps/casrad |
| **Site** | https://casrad.casapps.us |

## Rule Files

Detailed implementation rules live in `.claude/rules/` — regenerated from AI.md at session start:

| File | Covers |
|------|--------|
| `ai-rules.md` | AI behavior, spec compliance (PART 0, 1) |
| `project-rules.md` | License, structure, OS paths (PART 2, 3, 4) |
| `config-rules.md` | Configuration, modes, server config (PART 5, 6, 12) |
| `binary-rules.md` | Binary requirements, CLI flags (PART 7, 8, 33) |
| `backend-rules.md` | Error handling, DB, security, logging (PART 9, 10, 11, 32) |
| `api-rules.md` | Health, API structure, SSL/TLS (PART 13, 14, 15) |
| `frontend-rules.md` | Web frontend, admin panel (PART 16, 17) |
| `features-rules.md` | Email, scheduler, GeoIP, metrics, backup (PART 18-23) |
| `service-rules.md` | Privilege escalation, service support (PART 24, 25) |
| `makefile-rules.md` | Makefile patterns (PART 26) |
| `docker-rules.md` | Docker (PART 27) |
| `cicd-rules.md` | CI/CD workflows (PART 28) |
| `testing-rules.md` | Testing, docs, i18n (PART 29, 30, 31) |
| `optional-rules.md` | Multi-user, orgs, custom domains (PART 34-36) |

## Critical Non-Negotiables

- **CGO_ENABLED=0** always — pure Go, no C dependencies
- **Never run `go` on host** — always `make dev` / `make test` / `make build` (Docker internally)
- **`modernc.org/sqlite`** for SQLite — never `mattn/go-sqlite3`
- **Argon2id** for all password hashing — never bcrypt, never MD5/SHA
- **`config.ParseBool()`** — never `strconv.ParseBool()`
- **`golang:alpine`** rolling tag — never pin Go version
- **Config file**: `server.yml` (not `.yaml`)
- **Random port** from 64000–64999 range on first run
- **All user-facing text** must use i18n keys — never hardcoded English

## IDEA.md

Project WHAT (features, data models, business rules): see `IDEA.md`

## Spec

Implementation HOW: see `AI.md` — read only the PART(s) relevant to the current task.
