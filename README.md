# K2

Server monitoring tool for processes, containers, and system metrics. Built with Go + Echo, htmx + templ + PicoCSS, SQLite.

## Install

### Bare-metal (systemd)

```bash
curl -fsSL https://raw.githubusercontent.com/tashirka1/k2/main/scripts/install.sh | sudo bash
```

The script downloads the latest release binary, installs it to `/usr/local/bin/k2`, and sets up a systemd service. Data lives in `/opt/k2`:

- `/opt/k2/k2.env` — environment configuration (generated on first install)
- `/opt/k2/k2.db` — SQLite database

Re-running the script updates the binary without touching your config or data.

### Docker

```bash
cp env-example .env
make up
```

The container runs on port 8000, mapped to the host via `K2_EXTERNAL_PORT` (default `9000`). The `./db/` directory and `.env` are mounted for persistence.

## Development

```bash
cp env-example .env
make dev   # run dev server with autoreload (air)
```

## Build

```bash
make build-bin          # dev binary to bin/k2
make build-linux-amd64  # static release binary to bin/k2-linux-amd64
make check              # lint + format + tests
```

## Configuration

Create a `.env` file in the project root (see `env-example`):

| Variable               | Default          | Description                              |
| ---------------------- | ---------------- | ---------------------------------------- |
| `K2_SESSION_KEY`       | *(required)*     | Session signing key                      |
| `K2_PORT`              | `8000`           | Server listen port                       |
| `K2_EXTERNAL_PORT`     | `9000`           | Host port for Docker (compose only)      |
| `K2_DB_NAME`           | `./data/k2.db`   | SQLite database path                     |
| `K2_USERNAME`          | —                | Admin username                           |
| `K2_PASSWORD`          | —                | Admin password                           |
| `K2_COLLECT_INTERVAL`  | `30s`            | Metrics collection interval              |
| `K2_RETENTION`         | `168h`           | How long metrics are kept before purging |

## Usage

`k2` — start the server. `k2 credentials` — print the admin username and password.
