## 1. Core Infrastructure

- [ ] 1.1 Add `internal/core/config/` — Config struct with env parsing, `K2_` prefix, defaults for PORT, ENV_FILE, DB_NAME, EXTERNAL_PORT; SESSION_KEY required
- [ ] 1.2 Create goose migration — create tables `admin_config`, `admin_user`, `metrics_data`, `metrics_fts` virtual table; drop `auth_user`
- [ ] 1.3 Update `cmd/k2/main.go` to use new config package with godotenv from `K2_ENV_FILE`

## 2. Admin Module — Data Layer

- [ ] 2.1 Create `internal/admin/model/` — `AdminConfig` (path_hash, created_at), `AdminUser` (username, password, login_attempts, locked_until), sentinel errors
- [ ] 2.2 Create `internal/admin/storage/` — `AdminStorage` interface + SQL implementation: `GetConfig()`, `GetUser(username)`, `UpdateAttempts()`, `ResetAttempts()`, `CreateInitialConfig()`
- [ ] 2.3 Create `internal/admin/service/` — `AdminService` interface + struct: `GenerateCredentials()` (random words, 16-char password), `CheckLogin(username, password, now) error` with bruteforce logic, `GetCredentials()`, `GetConfig()`
- [ ] 2.4 Write `internal/admin/service/` tests — table-driven: login success, wrong password, lockout after 3 attempts, lockout expires, credentials display, first-start generation
- [ ] 2.5 Create `internal/admin/handler/` — Echo handlers for login page, login POST, logout, dashboard placeholder
- [ ] 2.6 Create `internal/admin/view/` — `.templ` components for login form, dashboard layout (mobile-first with PicoCSS), admin base layout with logout button

## 3. CLI — cobra

- [ ] 3.1 Add cobra dependency and create `cmd/k2/root.go` — root command with `PersistentPreRun` for config loading
- [ ] 3.2 Create `cmd/k2/server.go` — moves current server setup logic into cobra RunE, injects config, starts collector
- [ ] 3.3 Create `cmd/k2/credentials.go` — reads from DB directly, prints admin URL/username/password
- [ ] 3.4 Update `cmd/k2/main.go` — just calls `cmd.Execute()`

## 4. Metrics Module

- [ ] 4.1 Create `internal/metrics/model/` — `MetricPoint` struct (timestamp, category, name, value, unit), `ProcessInfo` (pid, name, cpu, ram), `ContainerInfo` (name, image, cpu, ram)
- [ ] 4.2 Create `internal/metrics/storage/` — `MetricsStorage` interface + SQL: `InsertBatch([]MetricPoint)`, `QueryRange(category, from, to)`, `PurgeOlderThan(time)`, `RebuildFTS(processes, containers)`, `SearchFTS(query)`, `GetLastProcessSnapshot()`, `GetLastContainerSnapshot()`
- [ ] 4.3 Create `internal/metrics/service/` — `MetricsService` interface + struct: `RunCollector(ctx, interval)` goroutine with ticker, delegates to system/process/docker collectors
- [ ] 4.4 Implement system collector — CPU percent, RAM used/total/percent, disk used/total/percent via gopsutil
- [ ] 4.5 Implement process collector — list all PIDs with name, CPU%, RAM% via gopsutil
- [ ] 4.6 Implement Docker collector — list all containers with stats via Docker SDK; graceful degradation if socket unavailable
- [ ] 4.7 Create `internal/metrics/handler/` — Echo handlers for dashboard page, processes page, containers page, search endpoint, chart data JSON endpoints (aggregated by period)
- [ ] 4.8 Create `internal/metrics/view/` — templ components: dashboard layout, CPU chart, RAM chart, disk chart, process table with FTS5 search, container table with FTS5 search
- [ ] 4.9 Add Chart.js to static assets — `static/js/chart.umd.min.js`, wire into admin view templates

## 5. Wire Everything

- [ ] 5.1 Update `cmd/k2/server.go` — wire config → db → admin → metrics → echo, setup routes with admin group at `/admin/<hash>/`
- [ ] 5.2 Remove old `internal/auth/` module entirely (all files)

## 6. Deployment & Docs

- [ ] 6.1 Update `Dockerfile` if needed for new dependencies (Docker SDK, CGO)
- [ ] 6.2 Create GitHub Actions workflow — `.github/workflows/publish.yml` — build multi-arch on tag `v*`, push to ghcr.io
- [ ] 6.3 Create `docs/installation.md` — wget + systemctl setup
- [ ] 6.4 Create `docs/docker.md` — make up + compose.yml + K2_EXTERNAL_PORT
- [ ] 6.5 Create systemd service file — `contrib/k2.service`
- [ ] 6.6 Update `Makefile` — add targets if needed
