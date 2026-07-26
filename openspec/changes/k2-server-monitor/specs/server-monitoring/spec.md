## ADDED Requirements

### Migration: monitoring tables MUST be written to `migrations/20260724193038_create_tables.sql`

The following tables SHALL be created in the existing migration file `migrations/20260724193038_create_tables.sql` (Up section):

```sql
CREATE TABLE metrics_resource (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    type      TEXT NOT NULL,       -- cpu, ram, disk
    name      TEXT NOT NULL,       -- percent, used, total, available
    device    TEXT,                -- mount point for disk (e.g. /, /data)
    value     REAL NOT NULL
);
CREATE INDEX idx_resource_ts ON metrics_resource(timestamp);

CREATE TABLE metrics_process (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    pid       INTEGER NOT NULL,
    name      TEXT NOT NULL,
    cpu       REAL NOT NULL,
    ram       REAL NOT NULL
);
CREATE INDEX idx_process_ts ON metrics_process(timestamp);

CREATE TABLE metrics_container (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    name      TEXT NOT NULL,
    image     TEXT NOT NULL,
    cpu       REAL NOT NULL,
    ram       REAL NOT NULL
);
CREATE INDEX idx_container_ts ON metrics_container(timestamp);

CREATE VIRTUAL TABLE metrics_resource_fts USING fts5(type, name, device, content='');
CREATE VIRTUAL TABLE metrics_process_fts USING fts5(name, pid, content='');
CREATE VIRTUAL TABLE metrics_container_fts USING fts5(name, image, content='');
```

### Requirement: System resource collection
The server SHALL collect system resource metrics every 5-10 seconds.

#### Scenario: CPU metrics collected
- **WHEN** collector tick fires
- **THEN** system measures CPU usage percent (average across all cores)
- **THEN** system writes CPU metric point to `metrics_resource`

#### Scenario: RAM metrics collected
- **WHEN** collector tick fires
- **THEN** system measures RAM used, RAM total, RAM percent
- **THEN** system writes RAM metric point to `metrics_resource`

#### Scenario: Disk metrics collected
- **WHEN** collector tick fires
- **THEN** system measures disk used, disk total, disk percent, disk available per mount point
- **THEN** system writes disk metric point to `metrics_resource` with `device` set to mount point

### Requirement: Process metrics collection
The server SHALL collect per-process metrics every 5-10 seconds across all running processes.

#### Scenario: Process metrics collected
- **WHEN** collector tick fires
- **THEN** system collects PID, process name, CPU%, RAM% for every process
- **THEN** system writes one row to `metrics_process` per process
- **THEN** system repopulates `metrics_process_fts` with process names and PIDs from latest snapshot
- **THEN** system repopulates `metrics_resource_fts` with metric types, names, and devices from latest resource snapshot

### Requirement: Docker container metrics collection
The server SHALL collect Docker container metrics every 5-10 seconds when Docker is available.

#### Scenario: Docker metrics collected
- **WHEN** collector tick fires and Docker socket is accessible
- **THEN** system collects container name, image, CPU%, RAM% for every container
- **THEN** system writes one row to `metrics_container` per container
- **THEN** system repopulates `metrics_container_fts` with container names and image names from latest snapshot

#### Scenario: Docker unavailable
- **WHEN** collector tick fires and Docker socket is not accessible
- **THEN** system skips Docker metrics collection
- **THEN** system logs warning at startup

### Requirement: FTS5 search
The system SHALL support full-text search across resource types, process names, and container names.

#### Scenario: Search resources
- **WHEN** user types search query in resource search field
- **THEN** system queries `metrics_resource_fts` matching metric types, names, or devices
- **THEN** system returns filtered results

#### Scenario: Search processes
- **WHEN** user types search query in process search field
- **THEN** system queries `metrics_process_fts` matching process names or PIDs
- **THEN** system returns filtered results

#### Scenario: Search containers
- **WHEN** user types search query in container search field
- **THEN** system queries `metrics_container_fts` matching container names or image names
- **THEN** system returns filtered results

### Requirement: Chart.js dashboards
The admin UI SHALL display resource metrics using Chart.js line charts.

#### Scenario: Dashboard shows CPU chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders CPU usage chart from `metrics_resource` where `type='cpu'` for selectable periods (1h, 6h, 24h, 7d)

#### Scenario: Dashboard shows RAM chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders RAM usage area chart from `metrics_resource` where `type='ram'` for selectable periods

#### Scenario: Dashboard shows Disk chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders disk usage gauge and used/total line chart from `metrics_resource` where `type='disk'`

#### Scenario: Processes tab shows top processes
- **WHEN** user navigates to processes tab
- **THEN** system displays sortable table from `metrics_process` with search
- **THEN** system shows mini sparkline charts per process

#### Scenario: Containers tab shows container metrics
- **WHEN** user navigates to containers tab
- **THEN** system displays table of Docker containers from `metrics_container` with search
- **THEN** system shows per-container CPU/RAM charts

### Requirement: 7-day data retention
The system SHALL automatically delete metric data older than 7 days.

#### Scenario: Old data purged
- **WHEN** collector starts a new tick
- **THEN** system deletes rows older than 7 days from `metrics_resource`, `metrics_process`, and `metrics_container`

#### Scenario: Retention on startup
- **WHEN** server starts
- **THEN** system purges data older than 7 days from all three metric tables
