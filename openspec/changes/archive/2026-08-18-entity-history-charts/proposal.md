## Why

The Process and Container tables show only the latest snapshot. Users cannot inspect how a specific process or container behaved over time. The System page already solves this for CPU/RAM/Disk via historical charts with a period selector; the same experience should be available per process (PID) and per container (name).

## What Changes

- Process table: PID cell becomes a link to a new history page `/metrics/processes/:pid`.
- Container table: Name cell becomes a link to a new history page `/metrics/containers/:name`.
- New detail pages rendering three Chart.js canvases (CPU %, RAM %, RAM Used) plus the same period select (1h, 6h, 24h, 7d, 1mo) as the System page.
- New JSON endpoint(s) returning historical series for a single PID or container name over a time range.
- Combine existing `HTTP status 500`, `HTTP 400` behaviors into API error responses.

## Capabilities

### New Capabilities
- `entity-history-charts`: Historical charts for a single process (by PID) and container (by name) with a period filter, mirroring the System page behavior.

### Modified Capabilities
<!-- None: existing spec (server-monitoring) changes only at implementation level; table cells gain links but route/monitoring behavior is covered by the new capability. -->

## Impact

- `internal/metrics/model/` — reused `ChartData`/`ChartSeries`; no new DTOs expected for process/container series.
- `internal/metrics/storage/` — new history queries on `metrics_process` and `metrics_container`; composite indexes.
- `internal/metrics/service/` — new history chart-building methods (pivot rows to ChartData).
- `internal/metrics/handler/` — new detail-page handlers and chart JSON handler.
- `internal/metrics/view/` — two new templ pages; link-ify PID/Name cells.
- `static/js/main.js` — support `data-chart` canvas attribute for non-resource chart URLs.
- `migrations/20260724193038_create_tables.sql` — add composite indexes `(pid, timestamp)` and `(name, timestamp)` to the last migration; no new migration file.
- Doctrine of System page is the reference: three canvases, shared period select, JSON-fed Chart.js.