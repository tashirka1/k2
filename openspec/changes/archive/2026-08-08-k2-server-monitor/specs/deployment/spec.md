## ADDED Requirements

### Requirement: Docker image published to GitHub Container Registry
The project SHALL publish Docker images to `ghcr.io/tashirka1/k2`.

#### Scenario: Image built on tag push
- **WHEN** a git tag matching `v*` is pushed
- **THEN** GitHub Actions builds multi-arch image (linux/amd64, linux/arm64)
- **THEN** image is pushed to `ghcr.io/tashirka1/k2:<tag>`
- **THEN** image is also tagged as `ghcr.io/tashirka1/k2:latest`

### Requirement: docker compose up workflow
The project SHALL support running via `make up` with docker compose.

#### Scenario: Compose starts the server
- **WHEN** user runs `make up`
- **THEN** Docker Compose builds or pulls the image
- **THEN** container starts with `.env` file mounted
- **THEN** `./data/` directory is mounted for SQLite persistence
- **THEN** port mapping uses `K2_EXTERNAL_PORT` env var

### Requirement: systemd service installation
The project SHALL provide a systemd service file for bare-metal deployment.

#### Scenario: Systemd service starts on boot
- **WHEN** systemd service is enabled
- **THEN** k2 starts on system boot
- **THEN** k2 runs as `noroot` user
- **THEN** k2 logs to journald

#### Scenario: Binary installed via wget
- **WHEN** user runs the installation script
- **THEN** latest binary is downloaded from GitHub releases
- **THEN** binary is placed in `/usr/local/bin/k2`
- **THEN** systemd service file is installed
- **THEN** service is started and enabled

### Requirement: Installation documentation
The project SHALL have installation docs covering both Docker and bare-metal deployment.

#### Scenario: Docker install docs
- **WHEN** user reads `docs/docker.md`
- **THEN** they find instructions for `make up` and compose configuration
- **THEN** they find `K2_EXTERNAL_PORT` usage example

#### Scenario: Bare-metal install docs
- **WHEN** user reads `docs/installation.md`
- **THEN** they find wget download command for latest binary
- **THEN** they find systemctl setup instructions
- **THEN** they find env var configuration example
