## Context

The System page (`System()` in `internal/metrics/view/metrics.templ`) renders three Chart.js canvases (`cpuChart`, `ramChart`, `diskChart`) fed by `main.js` (`static/js/main.js`). `initCharts()` scans `canvas[id$=Chart]` and builds fetch URLs as `/metrics/chart/{type}?period=...` where `{type}` is the canvas id minus the `Chart` suffix (`cpuChart` → `/metrics/chart/cpu`). `bindPeriod()` reads/writes a `period` query param and re-fetches all charts on change. Chart JSON is produced by the `ChartData` handler → `service.QueryChartData` → storage `QueryResources` over `metrics_resource` (one row per metric type).

Process/container data is structurally different: `metrics_process` and `metrics_container` store one **row per tick** (timestamp, pid/name, cpu, ram, ram_bytes) instead of type-keyed point rows. Existing indexes cover only `timestamp`. There is no per-entity history query, service method, endpoint, or page.

## Goals / Non-Goals

**Goals:**
- PID in the Process table and Name in the Container table become links to per-entity history pages.
- History page mirrors System: period `<select id="period">` + three canvases (CPU %, RAM %, RAM Used) fed by Chart.js JSON.
- History queried from existing `metrics_process`/`metrics_container` rows; no new collection.
- RAM Used displayed in MiB.

**Non-Goals:**
- Downsampling/decimation of large result sets (interval is 30s; longest period 720h ≈ 2880 points/series — acceptable for Chart.js).
- Per-entity aggregation, alerts, or sparklines in the tables.
- Changing how the System page itself works.
- Identity handling beyond the accepted assumption (PID reuse / container name drift is ignored).
- Data collection changes.

## Decisions

### D1: Composite indexes in a new goose migration
Both history queries filter by `pid`/`name` AND `timestamp`; the current `timestamp`-only indexes are insufficient.
- `CREATE INDEX idx_process_pid_ts ON metrics_process(pid, timestamp);`
- `CREATE INDEX idx_container_name_ts ON metrics_container(name, timestamp);`
Migration is a fresh dependent `.sql` file in `/migrations`, not an edit of the existing initial migration (production DBs already applied it).

### D2: New storage queries returning full rows (no storage-side pivot)
Add to `MetricsStorage` interface and `storage.Metrics`:
- `QueryProcessHistory(ctx, pid, from, to) ([]model.ProcessPoint, error)`
- `QueryContainerHistory(ctx, name, from, to) ([]model.ContainerPoint, error)`

Keeps column selection narrow (`timestamp, cpu, ram, ram_bytes`) and follows the existing layering: storage returns rows, service owns chart shaping (same as `QueryResources` + `buildChartData`).

### D3: Service pivots rows into a single-series `ChartData`
New service methods `QueryProcessChart(ctx, pid, from, to)` and `QueryContainerChart(ctx, name, from, to)` return `model.ChartData`:
- labels = distinct `timestamp` values (sorted)
- series = exactly one `ChartSeries` for the requested param (`cpu`, `ram`, `ram_bytes`)
- `ram_bytes` converted to MiB (`value / 1048576`)

The single-series-per-call shape matches how `main.js` renders one canvas per fetch, so no client-side series slicing is needed. Param→label mapping: `cpu`→`CPU %`, `ram`→`RAM %`, `ram_bytes`→`RAM Used (MB)`.

### D4: Detail pages via new handlers and templ views
- `GET /metrics/processes/:pid` → `view.ProcessHistoryPage(pid)`
- `GET /metrics/containers/:name` → `view.ContainerHistoryPage(name)`

Templ pages render `core_view.Base(...)`, the shared period `<select id="period">` (same options as System), and three canvases. Canvas ids keep the `...Chart` suffix so existing JS bindings work.

### D5: Chart JSON endpoints with param
- `GET /metrics/chart/process/:pid/:param`
- `GET /metrics/chart/container/:name/:param`
where `param ∈ {cpu, ram, ram_bytes}` and `:name` is URL-escaped in the link. Kept as new routes (Echo radix tree gives static `process`/`container` segments priority over the existing `:type`), leaving `/metrics/chart/:type` untouched. Period parsed from `?period=`, default `1h`, via existing `parseDuration`. Invalid entity/param → `400`; DB error → `500` JSON.

### D6: `data-chart` attribute in main.js
The System page's URL-building trick (`id` minus `Chart` suffix) cannot carry a PID or container name. Add optional `data-chart` support in `initCharts()`:

```
var url = canvas.dataset.chart
    ? canvas.dataset.chart + '?' + new URLSearchParams({ period: getPeriod() })
    : '/metrics/chart/' + canvas.id.replace('Chart', '') + '?period=' + getPeriod();
```

`bindPeriod()` is unchanged — it reads the `period` query param from `location.search`, which the detail page URL carries. Canvases get explicit `data-chart="/metrics/chart/process/{pid}/cpu"` (and equivalents) in templ.

### D7: Links in existing tables
In `ProcessResults`, PID cell becomes `<a href="/metrics/processes/{p.PID}">`. In `ContainerResults`, Name cell becomes `<a href="/metrics/containers/{url.PathEscape(p.Name)}">`. Plain `<a>` navigation (not htmx) — the 5s-replaced table result div stays unaffected. `format.go` gains a helper for building the hrefs if needed for escaping.

## Risks / Trade-offs

- [PID reuse / container name drift merges histories of different entities into one chart] → Accepted assumption per requirement; the view shows the PID/name as page title so it is self-explanatory.
- [`/metrics/chart/process/:pid/...` could be misread by Echo as existing `:type` route and 404] → Mitigated by Echo's router priority for static segments; verified with a route test.
- [data-chart URL could collide with the `main.js` `canvas.id$=Chart` selector on the System page] → System canvases carry no `data-chart`, so their behavior is unchanged; `data-chart` branch is purely additive.
- [Empty history (unknown PID / container gone) shows a blank chart] → Acceptable; page still renders, canvas stays empty.
- [Delay: no data before first collector tick] → Existing behavior for all pages; no mitigation needed.

## Migration Plan

1. Add goose migration with the two composite indexes.
2. Storage queries → service chart builders → handlers + templ pages + main.js `data-chart`.
3. New routes are additive and auth-protected under the existing `/metrics` group; no route removal, so rollback is a revert of the change.

## Open Questions

- Exact goose migration version/timestamp for the new file (decided at implementation time).
- Whether RAM Used far-axis labelling in Chart.js is needed beyond default numeric y-axis (MiB values render natively; default is fine).