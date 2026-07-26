## ADDED Requirements

### Requirement: System resource collection
The server SHALL collect system resource metrics every 5-10 seconds.

#### Scenario: CPU metrics collected
- **WHEN** collector tick fires
- **THEN** system measures CPU usage percent (average across all cores)
- **THEN** system writes CPU metric point to `metrics_data`

#### Scenario: RAM metrics collected
- **WHEN** collector tick fires
- **THEN** system measures RAM used, RAM total, RAM percent
- **THEN** system writes RAM metric point to `metrics_data`

#### Scenario: Disk metrics collected
- **WHEN** collector tick fires
- **THEN** system measures disk used, disk total, disk percent
- **THEN** system writes disk metric point to `metrics_data`

### Requirement: Process metrics collection
The server SHALL collect per-process metrics every 5-10 seconds across all running processes.

#### Scenario: Process metrics collected
- **WHEN** collector tick fires
- **THEN** system collects PID, process name, CPU%, RAM% for every process
- **THEN** system writes process metrics to `metrics_data`
- **THEN** system updates FTS5 index with process names and PIDs from latest snapshot

### Requirement: Docker container metrics collection
The server SHALL collect Docker container metrics every 5-10 seconds when Docker is available.

#### Scenario: Docker metrics collected
- **WHEN** collector tick fires and Docker socket is accessible
- **THEN** system collects container name, image, CPU%, RAM% for every container
- **THEN** system writes container metrics to `metrics_data`
- **THEN** system updates FTS5 index with container names and image names from latest snapshot

#### Scenario: Docker unavailable
- **WHEN** collector tick fires and Docker socket is not accessible
- **THEN** system skips Docker metrics collection
- **THEN** system logs warning at startup

### Requirement: FTS5 search
The system SHALL support full-text search across process names, container names, image names, and PIDs.

#### Scenario: Search processes
- **WHEN** user types search query in process search field
- **THEN** system queries FTS5 index matching process names or PIDs
- **THEN** system returns filtered results

#### Scenario: Search containers
- **WHEN** user types search query in container search field
- **THEN** system queries FTS5 index matching container names or image names
- **THEN** system returns filtered results

### Requirement: Chart.js dashboards
The admin UI SHALL display resource metrics using Chart.js line charts.

#### Scenario: Dashboard shows CPU chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders CPU usage chart for selectable periods (1h, 6h, 24h, 7d)

#### Scenario: Dashboard shows RAM chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders RAM usage area chart for selectable periods

#### Scenario: Dashboard shows Disk chart
- **WHEN** user navigates to admin dashboard
- **THEN** system renders disk usage gauge and used/total line chart

#### Scenario: Processes tab shows top processes
- **WHEN** user navigates to processes tab
- **THEN** system displays sortable table of processes with search
- **THEN** system shows mini sparkline charts per process

#### Scenario: Containers tab shows container metrics
- **WHEN** user navigates to containers tab
- **THEN** system displays table of Docker containers with search
- **THEN** system shows per-container CPU/RAM charts

### Requirement: 7-day data retention
The system SHALL automatically delete metric data older than 7 days.

#### Scenario: Old data purged
- **WHEN** collector starts a new tick
- **THEN** system deletes `metrics_data` rows older than 7 days from current time

#### Scenario: Retention on startup
- **WHEN** server starts
- **THEN** system purges data older than 7 days
