## ADDED Requirements

### Requirement: k2 starts the HTTP server
The `k2` command SHALL start the HTTP server with all subsystems (root command).

#### Scenario: Server starts with config from env
- **WHEN** user runs `k2`
- **THEN** system reads config from environment variables with `K2_` prefix
- **THEN** system applies defaults for missing variables
- **THEN** system initializes database
- **THEN** system starts collector goroutine
- **THEN** system starts HTTP server on configured port
- **THEN** system generates admin credentials if first start

#### Scenario: Server fails on missing required env
- **WHEN** `K2_SESSION_KEY` is not set
- **THEN** system prints error and exits with code 1

#### Scenario: Help shows usage
- **WHEN** user runs `k2 --help`
- **THEN** system prints usage with `k2` and `k2 credentials` commands

### Requirement: k2 credentials displays admin credentials
The `k2 credentials` command SHALL display admin URL, username, and password from the database.

#### Scenario: Credentials displayed successfully
- **WHEN** user runs `k2 credentials` and database has admin config
- **THEN** system prints admin URL, username, and password

#### Scenario: No credentials in database
- **WHEN** user runs `k2 credentials` and database is empty or has no admin config
- **THEN** system prints error and exits with code 1

### Requirement: Environment variables use K2_ prefix
All configuration env vars SHALL use the `K2_` prefix.

#### Scenario: K2_PORT sets server port
- **WHEN** `K2_PORT=9000` is set
- **THEN** server listens on port 9000
- **WHEN** `K2_PORT` is not set
- **THEN** server listens on port 8000

#### Scenario: K2_ENV_FILE sets env file path
- **WHEN** `K2_ENV_FILE=/etc/k2/.env` is set
- **THEN** system loads env vars from that file
- **WHEN** `K2_ENV_FILE` is not set
- **THEN** system tries to load `.env` from current directory

#### Scenario: K2_DB_NAME sets database path
- **WHEN** `K2_DB_NAME=/data/k2.db` is set
- **THEN** system uses that path for SQLite database
- **WHEN** `K2_DB_NAME` is not set
- **THEN** system uses `./data/k2.db`

#### Scenario: K2_SESSION_KEY is required
- **WHEN** `K2_SESSION_KEY` is not set
- **THEN** system exits with error

#### Scenario: K2_EXTERNAL_PORT for compose
- **WHEN** `K2_EXTERNAL_PORT` is set in `.env`
- **THEN** compose.yml maps that port to container port 8000
- **WHEN** `K2_EXTERNAL_PORT` is not set
- **THEN** compose.yml uses default 9000
