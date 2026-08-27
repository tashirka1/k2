## 1. Data Layer

- [x] 1.1 Add bucket DTOs to `internal/metrics/model/metrics.go` — `ResourceBucket{Timestamp,Avg,Min,Max float64}`, `ProcessBucket{Timestamp string; CPUAvg,CPUMin,CPUMax,RAMAvg,RAMMin,RAMMax float64; RAMBytesAvg,RAMBytesMin,RAMBytesMax int64}`, `ContainerBucket` equivalent.
- [x] 1.2 Change `MetricsStorage` in `internal/metrics/storage/metrics.go:14-22` to `QueryResources(ctx,mtype string,from,to time.Time,bucketSec int)`, `QueryProcessHistory(ctx,pid int,from,to,bucketSec)`, `QueryContainerHistory(ctx,name string,from,to,bucketSec)` — breaking, no shim.
- [x] 1.3 Implement `QueryResources` bucketed branch: when `bucketSec>0` → `SELECT datetime(CAST(strftime('%s',timestamp)/? AS INTEGER)*?,'unixepoch') AS bucket, AVG(value),MIN(value),MAX(value) FROM metrics_resource WHERE type=? AND name='percent' AND timestamp BETWEEN ? AND ? GROUP BY bucket ORDER BY bucket`; disk subquery `used/total*100` per ts then `AVG/MIN/MAX`. When `0` → raw `SELECT timestamp,... WHERE type=? AND timestamp>=? AND timestamp<=? ORDER BY timestamp`. Preserve `defer rows.Close()` + `rows.Err()`.
- [x] 1.4 Implement `QueryProcessHistory`/`QueryContainerHistory` bucketed branches: `SELECT bucket, AVG(cpu),MIN(cpu),MAX(cpu), AVG(ram),MIN(ram),MAX(ram), CAST(AVG(ram_bytes) AS INTEGER),MIN(ram_bytes),MAX(ram_bytes) ... GROUP BY bucket`; raw fallback when `bucketSec==0`. Add `scan*Buckets` scanners.

## 2. Business Logic

- [x] 2.1 Delete `internal/metrics/service/downsample.go` (`downsampleChartData`,`lttbIndices`,`chartThreshold` target 600). Replace with `bucketForPeriod(period time.Duration) (bucketSec,threshold int)` in `service/chart.go` with `target=400` and existing `niceBucket` steps `[30s,1m,5m,15m,30m,1h,2h,4h,6h,12h,24h]`, `threshold=period/bucket`, `threshold<3 → bucketSec=0`.
- [x] 2.2 Refactor `service/chart.go:11-33` `QueryChartData/QueryProcessChart/QueryContainerChart` to compute `sec,thr:=bucketForPeriod(to.Sub(from))`, call storage with `sec`, and build `ChartData` via new `build*Bucketed` when `sec>0` else existing `build*ChartData`.
- [x] 2.3 Add builders `buildResourceBucketedChartData`, `buildProcessBucketedChartData(param,buckets)`, `buildContainerBucketedChartData` — map `bucket.Timestamp` → `Labels`, emit 3 `ChartSeries` (`"CPU %"`, `"CPU % min"`, `"CPU % max"` etc.) with `min ≤ avg ≤ max`, `ram_bytes` via `ramBytesToMiB(float64(avg))`.
- [x] 2.4 Keep `buildChartData/buildDiskChartData/buildSeriesChartData` for raw path; factor `chartLabel` reuse.

## 3. Testing

- [x] 3.1 Remove `TestLttbIndices`/`TestDownsampleChartData` in `service/downsample_test.go` and update `TestChartThreshold` to target 400 expectations (e.g. `6h→360`, `720h→360` with new buckets) or replace with `TestBucketForPeriod` table-driven.
- [x] 3.2 Update `service/metrics_test.go` mock to new 5-arg signatures, add table-driven tests for `QueryChartData` bucketed vs raw branches, `bucketForPeriod`, envelope builders (`min≤avg≤max`, `len(labels)==threshold`, error propagation).
- [x] 3.3 Add `storage/metrics_test.go` in-memory SQLite: insert `30s` ticks, peak value 100 mid-bucket, assert bucketed `len≈threshold`, `max==100`, `avg` within bounds, `min≤avg≤max`; disk `used/total` multi-device aggregation; `ram_bytes` integer rounding check. Verify `strftime('%s',RFC3339)` parsing.

## 4. Interface & Verification

- [x] 4.1 Verify `handler/metrics.go:69-144` still passes `period → from/to` and renders JSON with 3 series when bucketed (no handler code change expected).
- [x] 4.2 Run `make check` (`templ generate + go fmt + golangci-lint + go test ./...`) — green.
- [x] 4.3 Smoke `make dev` + `curl /metrics/chart/cpu?period=1h|24h|168h` → `len(labels)≤400`, envelope present when bucketed, `500` on storage error / `400` on invalid `param` unchanged.
