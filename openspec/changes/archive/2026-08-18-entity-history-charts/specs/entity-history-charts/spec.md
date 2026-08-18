## ADDED Requirements

### Requirement: Process history page accessible from PID

The Process table SHALL render the PID as a link to a per-process history page at `/metrics/processes/:pid`. The history page SHALL render historical charts for the process with a selectable period, mirroring the System page layout.

#### Scenario: Navigate from PID link
- **WHEN** user clicks the PID in the Process table
- **THEN** system navigates to `/metrics/processes/{pid}`
- **THEN** system renders a history page with the period select (1h, 6h, 24h, 7d, 1mo) and three canvases

### Requirement: Container history page accessible from Name

The Container table SHALL render the container name as a link to a per-container history page at `/metrics/containers/:name`. The history page SHALL render historical charts for the container with a selectable period, mirroring the System page layout.

#### Scenario: Navigate from Name link
- **WHEN** user clicks the container name in the Container table
- **THEN** system navigates to `/metrics/containers/{name}`
- **THEN** system renders a history page with the period select (1h, 6h, 24h, 7d, 1mo) and three canvases

### Requirement: History charts for a single entity

Both history pages SHALL render three Chart.js line charts for the entity: CPU %, RAM %, and RAM Used (MiB). The charts SHALL be populated from `/metrics/chart/process/:pid/:param` and `/metrics/chart/container/:name/:param` JSON endpoints respectively, where `param` is `cpu`, `ram`, or `ram_bytes`.

#### Scenario: CPU chart shown
- **WHEN** user opens the history page
- **THEN** system fetches `cpu` param series for the entity and renders the CPU % chart

#### Scenario: RAM chart shown
- **WHEN** user opens the history page
- **THEN** system fetches `ram` param series for the entity and renders the RAM % chart

#### Scenario: RAM Used chart shown
- **WHEN** user opens the history page
- **THEN** system fetches `ram_bytes` param series and renders the RAM Used chart in MiB

### Requirement: Period filter on history pages

The history pages SHALL support period selection identical to the System page (1h, 6h, 24h, 7d, 1mo default 1h). Changing the period SHALL re-fetch and redraw all charts for the new time range without a full page reload.

#### Scenario: Change period
- **WHEN** user selects a different period in the period select
- **THEN** system updates the URL `period` query param
- **THEN** system re-fetches each chart series over the new period
- **THEN** system redraws the charts

#### Scenario: Default period
- **WHEN** user opens a history page without a `period` param
- **THEN** system uses 1 hour as the period

### Requirement: History chart data from stored metric rows

The system SHALL return series data by querying rows from `metrics_process` (for processes, filtered by `pid`) and `metrics_container` (for containers, filtered by `name`) within the requested time range. Timestamps SHALL be the chart x-axis labels.

#### Scenario: Query process history
- **WHEN** system receives `/metrics/chart/process/{pid}/{param}?period=...`
- **THEN** system queries `metrics_process` where `pid` matches and `timestamp` is within the period
- **THEN** system returns JSON with timestamp labels and one series of values for `param`

#### Scenario: Query container history
- **WHEN** system receives `/metrics/chart/container/{name}/{param}?period=...`
- **THEN** system queries `metrics_container` where `name` matches and `timestamp` is within the period
- **THEN** system returns JSON with timestamp labels and one series of values for `param`

### Requirement: Composite indexes for history queries

The schema SHALL include indexes on `(pid, timestamp)` in `metrics_process` and `(name, timestamp)` in `metrics_container` to serve history queries efficiently.

#### Scenario: Process history indexed
- **WHEN** the database migration runs
- **THEN** an index on `metrics_process(pid, timestamp)` exists

#### Scenario: Container history indexed
- **WHEN** the database migration runs
- **THEN** an index on `metrics_container(name, timestamp)` exists

### Requirement: Invalid entity error handling

The history chart endpoints SHALL return a `400 Bad Request` for an invalid entity identifier or an unsupported `param`, and `500 Internal Server Error` on storage failure.

#### Scenario: Invalid param
- **WHEN** system receives a request with a `param` other than `cpu`, `ram`, or `ram_bytes`
- **THEN** system responds `400`

#### Scenario: Storage failure
- **WHEN** the historical query fails at the storage layer
- **THEN** system logs the error and responds `500`