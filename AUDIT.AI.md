# CASRAD Project Audit

Started: 2026-05-28

Scope: comprehensive 6-pass audit against `AI.md` (source of truth) and `.claude/rules/*.md` (non-negotiable). Note: `CLAUDE.md` is outdated — `AI.md` overrides it on all conflicts.

Project state: very early scaffold. Substantial spec surface is unimplemented. This audit records every divergence from spec found; mechanical/structural issues are fixed inline. Subsystem-level gaps (entire protocol servers, scheduler tasks, migration UI, etc.) are listed for user decision — implementing them is multi-PR work, not an audit fix.

---

## Pass 1: Security

- [ ] auth/login: response paths for "unknown user" vs "bad password" not verified identical-timing or identical message (PART 11). Inspect `src/server/handler/auth.go` and `src/server/service/auth.go` to confirm both produce the same `UNAUTHORIZED`/"Invalid credentials" output with constant-time compare.
- [ ] CSRF: no CSRF middleware visible in `src/server/middleware/`. Spec PART 11 requires CSRF tokens on all state-changing forms.
- [ ] Session tokens: confirm tokens are SHA-256 hashed in DB (PART 11). `src/server/model/session.go` + store implementation must be reviewed; not verified.
- [ ] Setup token: PART 17 requires the first-run setup token to be displayed in console only and consumed once. Not located in source.
- [ ] GeoIP: PART 20 mandates runtime download from `ip-location-db` (CC0/PDDL). RESOLVED: AI.md is truth — ip-location-db is the correct source. No GeoIP downloader exists yet.
- [ ] Security databases: blocklists, CVE DBs not downloaded at runtime. PART 7 forbids embedding; current code embeds none and downloads none.

## Pass 2: Code Quality

- [ ] No NO_COLOR support anywhere in `src/`. PART 8 / binary-rules: ALL binaries MUST respect NO_COLOR. Grep returns zero hits.
- [ ] `src/main.go` flag parser hand-rolled, missing required flags from PART 8: `--shell`, `--color`, `--lang`, `--service`, `--maintenance`. Also missing version-output format from PART 8: `casrad VERSION (COMMIT) built ... at ... TZ`.
- [ ] Backup `IDEA.md.preMigration.bak` committed at repo root. Should be removed (or moved out of tree); root must not contain stale backups (project-files rules).
- [x] `docker/all-in-one.yml` deleted. `docker/Dockerfile.aio` retained as the all-in-one image definition.

## Pass 3: Logic and Correctness

- [ ] chi route trailing-slash: `r.Route("/tracks", ...)` with `r.Get("/", ...)` produces `/tracks/` — violates "no trailing slashes" (PART 14). Applies to `/albums`, `/artists`, `/playlists`, etc.
- [ ] Makefile dev/local/build targets shell out `\$$(go env GOOS)` while inside the Docker container — that's fine, but the comment in the Makefile claiming the host triggers the build is misleading; verify all `go env` calls execute inside container (they do; just noting).
- [ ] Makefile `test` target enforces 100% coverage — CI/CD spec PART 28 says 60% threshold. Mismatch.
- [ ] `release.txt` value `0.1.0` — Makefile applies `v` prefix? Inspection shows it does NOT add `v` to the binary version string; PART 26 says add `v` ONLY to numeric semver tags (release artifacts), not text. The Makefile currently never adds `v`; release tag creation passes `$(VERSION)` raw → tag is `0.1.0`, not `v0.1.0`. Violates PART 26.
- [ ] `docker-compose.yml` port mapping is `"80:80"` — must be `"172.17.0.1:{random_port}:80"` (PART 27, docker-rules).
- [ ] `docker-compose.yml` ports `6600:6600` and `1935:1935` likewise unbound to docker bridge. Same fix needed.

## Pass 4: Documentation Completeness

- [ ] `docs/` exists with MkDocs files (good per PART 30), but missing required coverage areas (admin surface, API surface, configuration, public protocols depth). Spot-check shows top-level pages only.
- [ ] `mkdocs.yml` and `.readthedocs.yaml` present (good).
- [ ] README.md exists and is current-ish; verify lists CLI flags actually implemented.
- [ ] `LICENSE.md` present; verify Embedded Licenses table covers all `go.sum` deps — quick scan shows only the explicit `require` block is likely listed. PART 2 requires every transitive too.
- [ ] No `.github/workflows/` files exist — all 6 required workflows missing (ci.yml, build-toolchain.yml, release.yml, beta.yml, daily.yml, docker.yml). PART 28 violation.
- [ ] No `renovate.json` at repo root. PART 28 requires Renovate.

## Pass 5: Spec and Rules Compliance

### Mandatory files / structure
- [ ] `.github/workflows/` directory is empty — ALL 6 required workflows missing.
- [ ] `src/data/` directory missing — PART 7 requires it for application data assets.
- [ ] `src/agent/` and `src/client/` missing — PART 7/33 require `casrad-agent` and `casrad-cli` binaries.
- [ ] No `locales/{es,zh,fr,ar,de,ja}.json` — only `en.json` present. PART 31 requires all 7 with identical keys.
- [ ] No `manifest.json` / service worker → PWA support missing (PART 16).
- [ ] No theme files / CSS in `src/server/static/` beyond `css/` directory; verify dark/light/auto themes implemented.
- [ ] No `IDEA.md.preMigration.bak` permitted at repo root.
- [ ] No `renovate.json`.

### Binary rules (PART 7, 8, 33)
- [ ] CGO_ENABLED=0: confirmed in Makefile and Dockerfile (OK).
- [ ] `modernc.org/sqlite`: confirmed (OK).
- [ ] Missing CLI flags: `--shell`, `--color`, `--lang`, `--service`, `--maintenance`.
- [ ] Version output format mismatched.
- [ ] No NO_COLOR handling.

### Config rules (PART 5, 6, 12)
- [ ] Config filename: verify `server.yml` (not `.yaml`) — needs inspection of `src/config/config.go`.
- [ ] `config.ParseBool()` is implemented and used in one spot (config.go:157). Other env-var bool reads not audited.
- [ ] Random port 64000–64999: verify in port-selection code (not seen).
- [ ] Path normalization helpers: verify `normalizePath`/`validatePath` exist in `src/config/path.go`.

### API rules (PART 13, 14, 15)
- [ ] All routes versioned at `/api/v1/...` — OK structurally, but the spec uses `{api_version}` placeholder; routes are hard-coded `/api/v1`.
- [ ] Trailing-slash bug from chi `Route("/x") + Get("/")` pattern (PART 14 violation).
- [ ] `/server/healthz` HTML route missing — only `/healthz` exists. PART 13/14 requires `/server/healthz` (HTML) + `/api/v1/server/healthz` (JSON).
- [ ] No `/server/{admin_path}/...` admin route tree present (PART 17 entirely unimplemented).
- [ ] No SSL/Let's Encrypt module wired (PART 15).
- [ ] No Swagger/GraphQL routes mounted at the spec's URLs (`/server/docs/swagger`, etc.).

### Backend rules (PART 9, 10, 11, 32)
- [ ] Argon2id: present (OK).
- [ ] `CREATE TABLE IF NOT EXISTS`: verify in `src/server/store/sqlite.go`.
- [ ] Idempotent schema, no migration files: verify (likely OK, no migrations/ dir present).
- [ ] No Tor module (PART 32).
- [ ] No audit logging service (PART 11) confirmed.

### Frontend rules (PART 16, 17)
- [ ] Three themes (dark/light/auto): theme directory `src/common/theme` exists but coverage not verified.
- [ ] Admin panel isolation: NOT implemented — there is no `/server/{admin_path}` route tree.
- [ ] No CDN scripts: not verified in templates.
- [ ] WCAG 2.1 AA: not auditable without rendered output.
- [ ] PWA missing.

### Docker rules (PART 27)
- [x] `docker/all-in-one.yml` deleted. `docker/Dockerfile.aio` retained.
- [ ] `docker-compose.yml` lacks `172.17.0.1:{random}:80` binding pattern.
- [ ] Tini chain present (OK).
- [ ] Tor package present in Dockerfile install list (OK, even though Tor not implemented in code).

### CI/CD rules (PART 28)
- [ ] NO workflows present. Bootstrap order: `Dockerfile.build` exists; `build-toolchain.yml` must be the first workflow created and dispatched before `ci.yml`/`release.yml`. Currently none exist.

### Testing rules (PART 29, 30, 31)
- [ ] `docker-compose.test.yml` exists (good).
- [ ] No actual `go test` test files inspected; `tests/` contains shell scripts only (not Go test runner). PART 29 expects Docker-based unit tests in `golang:alpine`.
- [ ] i18n: only 1 of 7 required locale files.

### Service rules (PART 24, 25)
- [ ] `src/service/service.go` exists, but no `--service --install/--uninstall/--enable/--disable/--status/--restart/--start/--stop` flags wired in `main.go`. All 8 service flags missing.
- [ ] Privilege drop after port bind: not verified.

### Features rules (PART 18, 19, 20, 21, 22, 23)
- [ ] Scheduler exists at `src/scheduler/scheduler.go` — verify all PART 19 default tasks scheduled. Spec lists 14 tasks; need code review.
- [ ] SMTP auto-detection: not located.
- [ ] GeoIP runtime download: not located.
- [ ] Metrics export at `/api/v1/server/metrics` Prometheus: not located (`src/server/metrics/` exists, route binding not verified).
- [ ] Backup/restore `--maintenance backup/restore/update` flags: not implemented.

### Optional rules (PART 34, 35, 36)
- [ ] PART 34 (multi-user) is required for casrad per IDEA.md. Registration mode, quotas, lockout — not fully verified.
- [ ] PART 35 (orgs) and PART 36 (custom domains) NOT needed for casrad — confirmed in rules; verify nothing accidentally added.

## Pass 6: Code Flow Trace

- [ ] Env var inventory: scan `os.Getenv` / `os.LookupEnv` uses and reconcile with README/IDEA.md and docker-compose defaults. Found: `CASRAD_DEBUG`, `CASRAD_PORT`, `CASRAD_ADDRESS`, `CASRAD_DATA`, `MODE`, `DEBUG`. Verify all are documented + defaulted in compose.
- [ ] Visibility audit: pending.
- [ ] Input validation audit on track/album/playlist handlers: pending.

---

## Fixed in this audit pass

- Makefile test coverage threshold 100% → 60% (PART 28 alignment).
- `docker/docker-compose.yml` port bindings switched to `172.17.0.1:<port>:<internal>` form (PART 27).
- `docker/all-in-one.yml` deleted (`Dockerfile.aio` retained).
- `IDEA.md.preMigration.bak` deleted from repo root.
- Audit scope header updated: AI.md is source of truth, CLAUDE.md is outdated.

## Requires user decision (cannot fix without input)

1. **Massive missing surface** — entire protocol servers (MPD, Subsonic, Ampache, WebDAV, RTMP, DLNA), admin panel, migration UI, scheduler task implementations, backup/restore CLI, service flags, casrad-cli, casrad-agent, 6 GitHub workflows, all i18n locales, PWA, SSL/Let's Encrypt module. These are subsystem implementations, not audit fixes.
2. **GeoIP source** — RESOLVED: AI.md is truth; ip-location-db (CC0/PDDL).
3. **`docker/all-in-one.yml`** — RESOLVED: deleted. `Dockerfile.aio` retained.
4. **`IDEA.md.preMigration.bak`** — RESOLVED: deleted.
5. **Re-run audit with implementation focus** — most remaining items are "implement subsystem X to spec," which the audit skill cannot do unilaterally per the audit's red-flag rules ("stub/TODO implements core business logic that isn't specified anywhere" → ask).

## Completed

- (none yet — fixes below)
