## Context

Current chart pipeline is `handler(parse period) → service.Query*Chart(ctx,pid/param/from,to) → storage.Query* (full scan) → service.buildChartData → service.downsampleChartData(LTTB) → handler JSON`. `storage/metrics.go:109,139,149` fetch every row; `service/downsample.go:10-33` computes `target=600`, `bucket=niceBucket(period/600)`, `threshold=period/bucket`, then `lttbIndices` selects `threshold` points preserving shape. Collection is `30s` (`internal/core/config/config.go:47`), retention `168h`; 7d chart loads ~20k rows × 3 system charts + history charts. The Go-side `ChartData` + LTTB allocations dominate heap on minimal hardware (`debug.SetMemoryLimit(40MiB)`, `hard_heap_limit 16MiB` in `internal/core/db/db.go:28`) and add SQLite IO for data that will be discarded. LTTB also cannot run in SQLite, but bucketed `AVG/MIN/MAX` in SQL gives envelope fidelity with 1 query and `≤400` returned rows.

Constraints: `AGENTS.md` — flat-modular + layered (`internal/core` forbidden to import modules, modules isolated via interfaces wired in `cmd/k2/main.go`), `database/sql` only, no ORM, no `any/reflect/panic/init()`, `ctx` first, `(*T,nil)=not found`, `max 2 return values`, `max 3 nesting`, `bin/` builds.

## Goals / Non-Goals

**Goals:**
- Move downsampling from `service/downsample.go` to `storage/metrics.go` via `GROUP BY` bucketed SQL with `AVG/MIN/MAX` envelope.
- Lower target `600 → 400`, recompute thresholds, remove LTTB entirely.
- Breaking API change to `MetricsStorage` (`bucketSec int` param) — no compatibility shim.
- Preserve shape via envelope (per bucket `min ≤ avg ≤ max`), allow integer rounding for `ram_bytes`.
- Keep layered flow: handler parses `period`, service computes `bucketSec/threshold`, storage does bucketed scan, service builds `ChartData` (3 series when bucketed, 1 when raw).

**Non-Goals:**
- New DB migration/index — existing `idx_resource_type_ts`, `idx_process_pid_ts`, `idx_container_name_ts` in `migrations/20260724193038_create_tables.sql` stay.
- Frontend redesign — `Chart.js` in `static/js/chart.umd.js` already renders multiple series; `view/metrics.templ` + `handler/metrics.go` API unchanged except series count.
- FTS/search or retention changes.

## Decisions

**1. Bucket expression: `datetime(CAST(strftime('%s',timestamp)/? AS INTEGER)*?,'unixepoch')`**
- *Why:* `timestamp` is `TEXT RFC3339 UTC` (`storage/metrics.go:110` formatting). `strftime('%s',timestamp)` parses RFC3339 in SQLite and yields unix epoch; integer division truncates to bucket start. No schema change needed. Verified `modernc.org/sqlite` supports `strftime` + `CAST`.
- *Alternative rejected:* store extra `INTEGER unixepoch` column + generated column — requires migration, writes every row; overkill for ~400 groups.
- *Alternative rejected:* `julianday` math — less precise, same expressiveness.

**2. Aggregation: `AVG/MIN/MAX` per bucket, not pure `AVG` or LTTB**
- *Why:* User requires envelope. `AVG` alone smooths spikes; `MIN/MAX` preserves them without `LTTB` procedural logic. 3 values → 3 `ChartSeries` (`avg`, `min`, `max`) keeps Chart.js simple vs. single representative point.
- *Alternative rejected:* M4 (first/last/min/max per bucket, 4 points) — 4× points, needs `UNION` + window functions, over-engineered for target 400.
- *Alternative rejected:* `ROW_NUMBER() % step` uniform sample — skips peaks entirely.

**3. Interface: `QueryResources(ctx, type, from, to, bucketSec int)` etc., `bucketSec==0` → raw**
- *Why:* Single method preserves call-site clarity; service computes `bucketSec` via `bucketForPeriod(to.Sub(from))` and passes through. Breaking change allowed, avoids duplicate `Bucketed` variants.
- *Alternative rejected:* storage computing `niceBucket` itself — would import service constants or duplicate target/steps, violating layering.
- *Envelope DTO:* introduce `model.ResourceBucket{Timestamp,Avg,Min,Max}` etc. or keep `ResourcePoint` with `Value=avg` plus extra fields. Chose new bucket structs to keep `Insert*Batch` and `Search` DTOs clean; builders `buildBucketedChartData` map buckets → 3 series.

**4. Disk handling: percent per timestamp then bucket AVG/MIN/MAX**
- Current `buildDiskChartData:95` aggregates `SUM(used)/SUM(total)*100` per timestamp across devices. Bucketed: per-timestamp percent `used/total*100` as subquery, then bucketed `AVG/MIN/MAX` on that percent (or `SUM(used)/SUM(total)` per bucket → percent). Chose subquery `SELECT datetime(...) AS bucket, used,total` then `GROUP BY bucket` with `SUM(used)/SUM(total)*100` for `avg` and `AVG(used/total)*100` for `min/max` envelope — simplest SQL, matches current device-aggregation.

**5. Target 400 + `niceBucket` steps**
- `period/target` with steps `[30s,1m,5m,15m,30m,1h,2h,4h,6h,12h,24h]` keeps human-readable buckets. `400` reduces max series from `~360` to `~240-400` (e.g. `1h: 3600/400=9s → 30s bucket → 120 thr` stays, `24h:86400/400=216s → 5m bucket → 288 thr`, `7d:604800/400=1512s → 30m bucket → 336 thr → 400 steps would be 30m still). Validated against old `downsample_test.go:18-22` expectations.

**6. Remove `downsample.go` fully**
- No fallback LTTB; `chartThreshold` replaced by `bucketForPeriod` returning `(bucketSec,threshold)`. `thr<3 → bucketSec=0` fallback to raw query handles tiny periods (mirrors old `return 0` behavior).

## Risks / Trade-offs

- **`strftime('%s',timestamp)` parsing failure on non-UTC or malformed RFC3339** → Mitigation: use `from.UTC().Format(time.RFC3339)` consistently (`storage:110`), add unit test with fixed RFC3339 string; log parse edge.
- **`GROUP BY` on expression not index-backed** → Mitigation: `WHERE timestamp BETWEEN ? AND ?` still uses `idx_*` for pruning; grouping 400 buckets is cheap (hash agg).
- **Envelope inflates JSON ~3× series count** → Mitigation: still `≤1200` floats vs. `20k` before; negligible vs. old payload; Chart.js handles 3 datasets per canvas.
- **Integer rounding for `ram_bytes` loses <0.5 MiB** → Mitigation: explicit requirement, `CAST(AVG(ram_bytes) AS INTEGER)` then `ramBytesToMiB` in service; acceptable per spec.
- **Breaking storage interface breaks mocks** → Mitigation: update `metrics_test.go` mocks in same change, no external consumers.

## Migration Plan

1. Deploy new binary via `make build-bin` (`bin/k2`), rolling restart — no DB migration. Old rows stay, new queries read same tables.
2. Rollback: redeploy previous `bin/k2`; no schema to revert.
3. Verification: `make check` (templ generate + go fmt + golangci-lint + go test), `make dev` + `curl /metrics/chart/cpu?period=24h` checks `len(labels) ≤400` and `min≤avg≤max`.

## Open Questions

- None — all decisions resolved by user input (envelope, removal, breaking, rounding, target).

