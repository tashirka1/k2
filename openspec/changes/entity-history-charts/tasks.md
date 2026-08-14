## 1. Data layer (migration + storage)

- [x] 1.1 Add `idx_process_pid_ts` on `metrics_process(pid, timestamp)` and `idx_container_name_ts` on `metrics_container(name, timestamp)` to the last migration `/migrations/20260724193038_create_tables.sql` (no new migration file)
- [x] 1.2 Add `QueryProcessHistory(ctx, pid, from, to)` and `QueryContainerHistory(ctx, name, from, to)` to the `MetricsStorage` interface in `internal/metrics/storage/metrics.go`
- [x] 1.3 Implement `QueryProcessHistory` in `storage.Metrics` (select `timestamp, cpu, ram, ram_bytes` from `metrics_process` where `pid = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`)
- [x] 1.4 Implement `QueryContainerHistory` in `storage.Metrics` (select `timestamp, cpu, ram, ram_bytes` from `metrics_container` where `name = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`)

## 2. Service layer

- [x] 2.1 Add `QueryProcessChart(ctx, pid, from, to)` and `QueryContainerChart(ctx, name, from, to)` to the `MetricsService` interface in `internal/metrics/service/metrics.go`
- [x] 2.2 Implement chart pivot helpers in service (rows → `model.ChartData`, single series; `ram_bytes` → MiB; label mapping `cpu`→`CPU %`, `ram`→`RAM %`, `ram_bytes`→`RAM Used (MB)`)
- [x] 2.3 Implement `QueryProcessChart` and `QueryContainerChart` delegating to storage queries and pivot helpers

## 3. Tests (service layer)

- [x] 3.1 Write table-driven tests for the chart pivot helper covering cpu, ram, ram_bytes (MiB conversion) series
- [x] 3.2 Write tests for `QueryProcessChart`/`QueryContainerChart` with a lightweight storage mock (success path + storage error path)

## 4. Views + handler

- [x] 4.1 Add `ProcessHistoryPage(pid)` and `ContainerHistoryPage(name)` templ components mirroring `System()` (period select with `id="period"` + three canvases `cpuChart`, `ramChart`, `ramBytesChart` with `data-chart` attributes)
- [x] 4.2 Update `ProcessResults` templ so PID renders as `<a href="/metrics/processes/{pid}">`
- [x] 4.3 Update `ContainerResults` templ so Name renders as `<a href="/metrics/containers/{escaped name}">`
- [x] 4.4 Add handler `ProcessHistory` and `ContainerHistory` rendering the detail pages; add chart JSON handlers for `/metrics/chart/process/:pid/:param` and `/metrics/chart/container/:name/:param` (parse `period` default `1h`, validate param, 400/500 errors)
- [x] 4.5 Register new routes under the authenticated `/metrics` group in `SetupHandlers`

## 5. Frontend

- [x] 5.1 Update `static/js/main.js` `initCharts()` to prefer `canvas.dataset.chart` over the `id`-based URL when present, appending `period` from `getPeriod()`
- [x] 5.2 Regenerate templ (`templ generate`) and build static assets; verify `main.js` version query string bump in `Base` if caching requires it

## 6. Verification

- [x] 6.1 Run `make check` and ensure lint, format, and tests pass
- [ ] 6.2 Run a manual smoke test: navigate from PID/Name links, change period on both history pages, confirm charts redraw