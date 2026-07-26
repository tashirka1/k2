## 1. Core Infrastructure

- [x] 1.1 Add `internal/core/config/` — Config struct with env parsing, `K2_` prefix, defaults for PORT, ENV_FILE, DB_NAME, EXTERNAL_PORT; SESSION_KEY required
- [x] 1.2 Write goose migration in `migrations/20260724193038_create_tables.sql` — create tables `admin_user`, `metrics_resource`, `metrics_process`, `metrics_container`, FTS5 virtual tables `metrics_resource_fts`, `metrics_process_fts`, `metrics_container_fts`; drop `auth_user`
- [x] 1.3 Update `cmd/k2/main.go` to use new config package with godotenv from `K2_ENV_FILE`

## 2. Admin Module — Data Layer

- [x] 2.1 Create `internal/admin/model/` — `AdminUser` (username, password, login_attempts, locked_until), sentinel errors
- [x] 2.2 Create `internal/admin/storage/` — `AdminStorage` interface + SQL implementation: `GetUser(username)`, `UpdateAttempts()`, `ResetAttempts()`, `CreateInitialUser()`
- [x] 2.3 Create `internal/admin/service/` — `AdminService` interface + struct: `EnsureCredentials()` (random words, 16-char password), `CheckLogin(username, password, now) error` with bruteforce logic, `GetCredentials()`
- [x] 2.4 Write `internal/admin/service/` tests — table-driven: login success, wrong password, lockout after 3 attempts, lockout expires, credentials display, first-start generation
- [x] 2.5 Create `internal/admin/handler/` — Echo handlers for root `/`, login page, login POST, logout, dashboard
- [x] 2.6 Create `internal/admin/view/` — `.templ` components for login form, root page, dashboard

## 3. CLI — cobra

- [x] 3.1 Add cobra dependency and rewrite `cmd/k2/main.go` — root command (`k2`) starts server with `PersistentPreRun` for config loading; add `k2 credentials` subcommand that reads DB and prints username/password

## 4. Metrics Module

- [x] 4.1 Create `internal/metrics/model/` — `ResourcePoint` (timestamp, type, name, device, value), `ProcessPoint` (timestamp, pid, name, cpu, ram), `ContainerPoint` (timestamp, name, image, cpu, ram)
- [x] 4.2a Create `internal/metrics/storage/` — `MetricsStorage` interface + SQL
- [x] 4.2b Create indexes on timestamp columns
- [x] 4.3 Create `internal/metrics/service/` — collector goroutine
- [x] 4.4 Implement system collector via gopsutil
- [x] 4.5 Implement process collector via gopsutil
- [x] 4.6 Implement Docker collector via moby SDK
- [x] 4.7 Create `internal/metrics/handler/` — Echo handlers under `/metrics/` prefix
- [x] 4.8 Create `internal/metrics/view/` — templ components
- [x] 4.9 Add Chart.js to `static/js/`

## 5. Wire Everything

- [x] 5.1 Update `cmd/k2/main.go` — wire config → db → admin → metrics → echo, setup routes with trailing slash support
- [x] 5.2 Remove old `internal/auth/` module entirely (all files)

## 6. Deployment & Docs

- [x] 6.1 Update `Dockerfile` if needed
- [x] 6.2 Create GitHub Actions workflow
- [x] 6.3 Create `docs/installation.md`
- [x] 6.4 Create `docs/docker.md`
- [x] 6.5 Create systemd service file
- [x] 6.6 Update `Makefile`
