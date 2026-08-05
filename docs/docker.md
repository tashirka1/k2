# Docker

## Quick start

```bash
make up
```

This uses `docker compose` to build and start the container with default settings.

## Configuration

Create a `.env` file in the project root:

```env
K2_SESSION_KEY=your-secret-key
K2_PORT=8000
K2_DB_NAME=./data/k2.db
K2_EXTERNAL_PORT=9000
K2_USERNAME=admin
K2_PASSWORD=your-password
K2_COLLECT_INTERVAL=30s
```

The `K2_EXTERNAL_PORT` controls the host port mapping (default: 9000).

The `K2_COLLECT_INTERVAL` sets how often the collector samples metrics (default: `30s`).

## Volumes

- `./data/` — SQLite database persistence
- `.env` — environment configuration
