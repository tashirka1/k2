## Context

K2 is a Go server monitoring tool at early development stage. Currently it has:
- A single `internal/auth/` module with email/password registration and session login
- Hardcoded config (no env prefix, no defaults, port hardcoded to `:8000`)
- Direct `Run()` function in `cmd/k2/main.go` — no CLI structure
- No monitoring capabilities, no metrics collection, no admin panel
- SQLite database with stub migrations

The project follows flat modular architecture: `internal/core/` (infrastructure), isolated modules (`internal/auth/`) communicating via interfaces wired in `main.go`.

## Goals / Non-Goals

**Goals:**
- Replace auth module with admin module (auto-generated credentials, path hash, bruteforce protection)
- Add cobra CLI with `k2 server` and `k2 credentials` subcommands
- Add env config with `K2_` prefix and sensible defaults
- Add system resource monitoring collector (goroutine, 5-10s interval)
- Add monitoring UI with Chart.js charts and FTS5 search
- Add Docker image publish to ghcr.io and systemd deployment support
- Add installation documentation

**Non-Goals:**
- Multi-user admin management (admin_user table supports it but no UI for it yet)
- Real-time alerts or notifications
- Prometheus/OpenTelemetry integration
- Cloud provider monitoring
- Distributed or clustered deployment

## Decisions

### 1. cobra for CLI
**Why:** Standard in Go ecosystem, supports subcommands naturally (`k2 server`, `k2 credentials`), auto-generates help, bash completion. Replaces the current flat `Run()`.

### 2. Plaintext password in DB, no bcrypt
**Why:** `k2 credentials` must display the current password. bcrypt is one-way. Writing to separate file adds complexity. The password is auto-generated, single-use-per-deploy, and access to the SQLite file implies full system access already.
**Alternative considered:** bcrypt hash in DB + plaintext in `~/.k2/credentials` file. Rejected as unnecessary complexity for this use case.

### 3. Collector inside server process
**Why:** Single binary deployment (no separate agent process). Simpler: one `go run`, one process to monitor, one DB connection. The collector runs as background goroutine with its own ticker.
**Risk:** Heavy `procfs` scans on every tick. Mitigation: scan duration is measured and logged; if scan takes longer than interval, next tick is skipped.

### 4. SQLite for metrics storage
**Why:** The project already uses SQLite. No need for a separate time-series DB. WAL mode handles concurrent writes (collector) and reads (UI). 7-day retention keeps DB bounded (~3M rows/week ≈ 300MB).
**Risk:** Write contention at scale. Mitigation: 64-connection pool, WAL mode, batch inserts inside transactions, separate collector DB connection.

### 5. Chart.js for visualization
**Why:** Responsive out of the box (mobile-first), good touch support, 70KB gzipped, wide browser support. SVG server-side rendering would require custom interactivity code.

### 6. Docker SDK (official) for container stats
**Why:** Type-safe API, handles socket connection, streaming stats. Alternative `docker stats` CLI parsing is brittle. Direct HTTP on socket is more work for same result.
**Risk:** Docker socket must be mounted into container. Document this requirement.

### 7. Full process list (not top-N)
**Why:** Monitoring all processes enables FTS5 search across all names. The data volume per interval is small (~200-500 rows × ~50 bytes each ≈ 25KB per tick).

### 8. FTS5 on latest snapshot only
**Why:** Rebuilding FTS index every 5-10s on the full dataset is expensive. Instead, maintain a separate FTS table that is repopulated from the latest metrics snapshot on each tick.

### 9. Module naming: `internal/admin/`, `internal/metrics/`
**Why:** `admin_` prefix matches module name for DB tables. `metrics_` prefix for monitoring tables is short and unambiguous.

### 10. Separate `internal/core/config/` for env parsing
**Why:** Config parsing is infrastructure, not business logic — fits `internal/core/` scope. Keeps env logic out of `main.go`.

## Risks / Trade-offs

- **[Security]** Plaintext password in SQLite — if DB file is exposed, credentials are compromised. Mitigation: document file permissions (chmod 600 on DB dir).
- **[Performance]** Full process scan every 5-10s may be heavy on large systems (1000+ processes). Mitigation: monitor scan duration, log warnings if approaching interval.
- **[Portability]** Docker SDK + Docker socket — feature only works when Docker is available. Mitigation: collector gracefully degrades (no container data if socket unavailable).
- **[Data loss]** 7-day retention means no long-term history. Mitigation: documented limitation; aggregation could be added later.

## Migration Plan

1. Create goose migration with new tables (`admin_config`, `admin_user`, `metrics_data`, FTS5 virtual tables)
2. Remove old `auth_user` table in migration
3. Delete `internal/auth/` module
4. Implement modules in order: core/config → admin → cli → metrics → docs
5. Update `cmd/k2/main.go` structure to cobra
