## Why

K2 is a skeleton project with only an auth module, hardcoded config, and no monitoring capabilities. It needs to become a standalone server monitoring tool with CLI interface, secure admin panel, resource metrics collection, and deployable via Docker and systemd.

## What Changes

- **BREAKING**: Replace `internal/auth/` with `internal/admin/` — email/password registration removed, replaced with auto-generated admin credentials (username from 2-3 random words, 16-char password with symbols)
- **BREAKING**: Switch from `Run()` to cobra CLI — `k2` starts the server, `k2 credentials` subcommand
- **BREAKING**: All env vars prefixed with `K2_` (`K2_PORT`, `K2_ENV_FILE`, `K2_SESSION_KEY`, `K2_DB_NAME`, `K2_EXTERNAL_PORT`)
- Add admin panel at `/admin/<path_hash>/` with login page and bruteforce protection (3 attempts → 1 min lockout)
- Add resource monitoring collector (goroutine every 5-10s) — CPU, RAM, disk, all processes, Docker containers via official SDK
- Add monitoring UI with Chart.js charts and FTS5 search by resource type/device, process name/PID, container name/image
- Add data retention policy: 7 days, auto-cleanup per table
- Add Docker image publish to ghcr.io/tashirka1/k2
- Add installation docs (wget + systemctl, docker compose)

## Capabilities

### New Capabilities
- `admin-auth`: Admin login with path_hash-protected URL, auto-generated username/password, bruteforce protection, session-based auth, logout
- `server-monitoring`: Resource metrics collection (CPU, RAM, disk, processes, Docker containers), Chart.js charts, FTS5 search across all metric types, 7-day retention per table
- `cli-tool`: Cobra CLI — `k2` starts the server, `k2 credentials` subcommand, env var config with K2_ prefix and defaults
- `deployment`: Docker multi-arch image, GitHub Container Registry, compose.yml, systemd service file, installation docs

### Modified Capabilities
*(none — no existing specs to modify)*

## Impact

- Replace `internal/auth/` entirely with `internal/admin/`
- Add `internal/metrics/` module
- Update `cmd/k2/main.go` — cobra root command (`k2`) starts server, `k2 credentials` subcommand
- Add `internal/core/config/` for env parsing with defaults
- New dependencies: cobra, docker SDK client, gopsutil, chart.js
- New DB tables: `admin_config`, `admin_user`, `metrics_resource`, `metrics_process`, `metrics_container`, 3 FTS5 virtual tables
- Migration: existing `auth_user` table must be replaced
- Docs: `docs/installation.md`, `docs/docker.md`
