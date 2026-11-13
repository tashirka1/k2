## ADDED Requirements

### Requirement: SQLite bucketed downsampling with MIN/MAX envelope

The system SHALL perform chart downsampling in SQLite via time-bucketed aggregation (not in Go). When a chart period requires downsampling, storage SHALL group rows by bucket start `datetime(CAST(strftime('%s',timestamp)/bucketSec AS INTEGER)*bucketSec,'unixepoch')` and return `AVG`, `MIN`, `MAX` per bucket for the requested metric.

#### Scenario: Resource chart uses bucketed query
- **WHEN** system receives `/metrics/chart/cpu?period=24h` and the computed threshold is ≥3
- **THEN** system calls `QueryResources` with `bucketSec=niceBucket(period/400).Seconds()` and executes `SELECT bucket, AVG(value), MIN(value), MAX(value) ... GROUP BY bucket ORDER BY bucket`
- **THEN** system returns JSON with bucket-aligned labels and 3 series (avg/min/max) and at most `threshold` labels

#### Scenario: Process chart envelope
- **WHEN** system receives `/metrics/chart/process/{pid}/cpu?period=7d` with bucketed period
- **THEN** system queries `metrics_process` with `AVG(cpu), MIN(cpu), MAX(cpu)` (and equivalent for `ram`/`ram_bytes`) grouped by bucket
- **THEN** response contains `min ≤ avg ≤ max` per bucket label

#### Scenario: Container chart envelope
- **WHEN** system receives `/metrics/chart/container/{name}/ram_bytes?period=7d` with bucketed period
- **THEN** system queries `metrics_container` with `AVG/MIN/MAX` per bucket
- **THEN** response contains envelope series for the requested param

#### Scenario: Raw fallback for small periods
- **WHEN** the period yields `threshold < 3` (e.g. `1h` with `30s` bucket → raw path or computed bucket would give fewer than 3 groups)
- **THEN** system calls storage with `bucketSec=0` and executes the non-bucketed `SELECT ... WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp`
- **THEN** system returns a single series without envelope (1 label per row)

### Requirement: Downsampling target 400 and niceBucket

The system SHALL use `target=400` and bucket steps `[30s, 1m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 24h]` where `bucket=niceBucket(period/target)` (first step ≥ `period/target`, fallback `24h`) and `threshold=period/bucket` (integer). `threshold < 3` disables bucketing.

#### Scenario: Threshold for 6 hours is recomputed
- **WHEN** period is `6h` (21600s)
- **THEN** bucket is `1m` (21600/400=54s → next step 1m) and threshold is `360` (21600/60)

#### Scenario: Threshold for 1 month with target 400
- **WHEN** period is `720h` (2592000s)
- **THEN** bucket is `2h` (2592000/400=6480s → 2h=7200s) and threshold is `360`

#### Scenario: Zero or negative period yields no bucket
- **WHEN** period is `0` or negative
- **THEN** system uses `bucketSec=0` and returns raw data

### Requirement: Disk chart bucketed aggregation

For `metricType=disk`, the system SHALL compute per-timestamp `used/total*100` (summing across devices per timestamp) in a subquery, then bucket that percent with `AVG/MIN/MAX` per bucket.

#### Scenario: Disk percent envelope
- **WHEN** system receives `/metrics/chart/disk?period=24h` with bucketed period
- **THEN** storage computes `used/total*100` per distinct timestamp, then `GROUP BY bucket` returning `AVG/MIN/MAX` of that percent
- **THEN** response series "Disk %" contains avg/min/max per bucket label

### Requirement: Removal of Go LTTB

The system SHALL NOT contain Go-side LTTB downsampling (`downsampleChartData`, `lttbIndices`, `chartThreshold` with target 600). `internal/metrics/service/downsample.go` SHALL be deleted and `chart.go` SHALL NOT call any Go downsampler.

#### Scenario: No LTTB artifact remains
- **WHEN** the codebase is built
- **THEN** `internal/metrics/service/downsample.go` does not exist
- **THEN** `go vet` and `make check` pass without references to `downsampleChartData` or `lttbIndices`

### Requirement: Integer rounding for ram_bytes

Bucketed `ram_bytes` averages SHALL be rounded to integer bytes via `CAST(AVG(ram_bytes) AS INTEGER)` in SQL before conversion to MiB (`/1048576`) in service.

#### Scenario: ram_bytes rounding
- **WHEN** a bucket contains `ram_bytes` values `1000` and `1001`
- **THEN** `AVG` is `1000.5` and storage returns `1000` (or `1001` per SQLite round) as `RAMBytesAvg`
- **THEN** service converts to MiB with `float64(avg)/1048576`

### Requirement: Storage interface with bucketSec

`MetricsStorage` SHALL expose `QueryResources(ctx, metricType, from, to, bucketSec int)`, `QueryProcessHistory(ctx, pid, from, to, bucketSec int)`, `QueryContainerHistory(ctx, name, from, to, bucketSec int)` where `bucketSec==0` means raw full-scan and `bucketSec>0` means bucketed envelope query. This is a **BREAKING** change with no compatibility shim.

#### Scenario: Breaking signature
- **WHEN** a caller invokes `QueryResources` with 5 arguments including `bucketSec`
- **THEN** storage executes the bucketed or raw branch accordingly and returns envelope buckets or raw points

#### Scenario: Envelope ChartData shape
- **WHEN** storage returns bucketed `Avg/Min/Max` per timestamp
- **THEN** service builds `ChartData` with `Labels = bucket timestamps` and `Series = 3` entries (e.g. `"CPU %"`, `"CPU % min"`, `"CPU % max"`)
- **THEN** handler renders JSON with 3 datasets; `make check` table-driven tests assert `min ≤ avg ≤ max` and `len(labels) == threshold`

