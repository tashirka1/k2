## MODIFIED Requirements

### Requirement: History chart data from stored metric rows

The system SHALL return series data by querying rows from `metrics_process` (for processes, filtered by `pid`) and `metrics_container` (for containers, filtered by `name`) within the requested time range using SQLite bucketed aggregation. When `threshold = period / niceBucket(period/400) ≥ 3`, the query SHALL be `GROUP BY datetime(CAST(strftime('%s',timestamp)/bucketSec AS INTEGER)*bucketSec,'unixepoch')` returning `AVG/MIN/MAX` per bucket (with `CAST(AVG(ram_bytes) AS INTEGER)` rounding); otherwise it SHALL be the raw `WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp`. Timestamps (or bucket starts) SHALL be the chart x-axis labels, and the response SHALL contain envelope series (avg/min/max) when bucketed, otherwise a single series.

#### Scenario: Query process history bucketed
- **WHEN** system receives `/metrics/chart/process/{pid}/{param}?period=7d` and threshold is ≥3
- **THEN** system calls `QueryProcessHistory` with `bucketSec=niceBucket(period/400).Seconds()` and storage executes `SELECT bucket, AVG(param), MIN(param), MAX(param) FROM metrics_process WHERE pid=? AND timestamp BETWEEN ? AND ? GROUP BY bucket ORDER BY bucket`
- **THEN** system returns JSON with bucket labels and 3 series (avg/min/max) for `param`

#### Scenario: Query container history bucketed
- **WHEN** system receives `/metrics/chart/container/{name}/{param}?period=7d` and threshold is ≥3
- **THEN** system calls `QueryContainerHistory` with `bucketSec` and storage executes the equivalent bucketed `AVG/MIN/MAX` query on `metrics_container`
- **THEN** system returns JSON with bucket labels and envelope series for `param`

#### Scenario: Raw fallback for small period
- **WHEN** system receives a history request with period yielding `threshold < 3` (e.g. short `1h` that would give fewer than 3 buckets)
- **THEN** storage executes the raw `SELECT timestamp, cpu, ram, ram_bytes FROM metrics_process|metrics_container WHERE pid|name=? AND timestamp BETWEEN ? AND ? ORDER BY timestamp`
- **THEN** system returns JSON with one series per request param

#### Scenario: Query process history (raw reference retained for compatibility)
- **WHEN** system receives a history request where bucketing is disabled
- **THEN** behavior matches the original scenario: system queries `metrics_process` where `pid` matches and `timestamp` is within the period and returns timestamp labels with one series of values for `param`

