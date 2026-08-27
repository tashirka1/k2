## Why

Chart endpoints currently load every row in the requested period (`SELECT ... WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp`) and downsample in Go via LTTB (`internal/metrics/service/downsample.go`). For `CollectInterval=30s` and `Retention=168h` a 7-day chart loads ~20k rows per series (3 charts in parallel), allocates `ChartData` for all points, then runs `LTTB` to ~300 points. This wastes SQLite IO, CPU and heap on minimal hardware (`hard_heap_limit 16MiB`, `GOMEMLIMIT 40MiB` in `cmd/k2/main.go`), contradicting the project's single-binary, low-resource promise.

## What Changes

- Replace Go LTTB downsampling with SQLite bucketed aggregation (AVG/MIN/MAX per time bucket) for all chart queries (`metrics_resource`, `metrics_process`, `metrics_container`).
- Lower downsampling target from `600` to `400` (`downsample.go:14` → `400`) and recompute `niceBucket` thresholds; `threshold = period / bucket`, disabled when `<3`.
- Require **MIN/MAX envelope** per bucket (AVG + MIN + MAX) so spikes are not smoothed away.
- **BREAKING** Remove `internal/metrics/service/downsample.go` entirely (`downsampleChartData`, `lttbIndices`, `chartThreshold` replaced by `bucketForPeriod`). No compatibility shim for the old full-scan API.
- Change `MetricsStorage` signatures to `QueryResources(ctx, metricType, from, to, bucketSec int)` etc.; `bucketSec==0` means raw (period < niceBucket threshold), otherwise bucketed query via `GROUP BY datetime(CAST(strftime('%s',timestamp)/bucketSec AS INTEGER)*bucketSec,'unixepoch')`.
- Allow integer rounding for `ram_bytes` bucket averages (`CAST(AVG(ram_bytes) AS INTEGER)`).
- Update `ChartData` builders to emit 3 series per metric when bucketed (avg/min/max envelope) instead of 1.

## Capabilities

### New Capabilities
- `sqlite-chart-downsampling`: SQLite-level time-bucketed downsampling for resource, process and container charts with MIN/MAX envelope and target 400.

### Modified Capabilities
- `entity-history-charts`: Update "History chart data from stored metric rows" requirement to mandate bucketed SQL aggregation with envelope instead of full-scan + Go LTTB; adjust period/threshold math for `target=400`.

## Impact

- `internal/metrics/model/` — new `ResourceBucket`/`ProcessBucket`/`ContainerBucket` DTOs (or enriched `*Point` with `Avg/Min/Max`).
- `internal/metrics/storage/metrics.go:14-22,109-157` — breaking `MetricsStorage` change, new `GROUP BY` queries, `strftime('%s',timestamp)` bucket expression, `AVG/MIN/MAX` aggregates, `defer rows.Close()` + `rows.Err()` preserved.
- `internal/metrics/service/chart.go:11-33` + `downsample.go:1-133` — remove LTTB, introduce `bucketForPeriod`, route to bucketed storage, builders for envelope series.
- `internal/metrics/service/*_test.go`, `storage/*_test.go` — table-driven tests for thresholds (`400`), bucket SQL, envelope invariants (`min <= avg <= max`), disk `used/total*100` per bucket.
- `internal/metrics/handler/` — no API change (JSON shape gains extra series when downsampled).
- `migrations/20260724193038_create_tables.sql` — no new migration; existing `(type,timestamp)`, `(pid,timestamp)`, `(name,timestamp)` indexes remain sufficient for `WHERE` pruning.
