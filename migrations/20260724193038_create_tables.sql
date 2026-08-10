-- +goose Up

CREATE TABLE IF NOT EXISTS auth_user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until DATETIME,
    created_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS metrics_resource (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    device TEXT,
    value REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_resource_ts ON metrics_resource(timestamp);

CREATE TABLE IF NOT EXISTS metrics_process (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    pid INTEGER NOT NULL,
    name TEXT NOT NULL,
    cpu REAL NOT NULL,
    ram REAL NOT NULL,
    ram_bytes INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_process_ts ON metrics_process(timestamp);

CREATE TABLE IF NOT EXISTS metrics_container (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    cpu REAL NOT NULL,
    ram REAL NOT NULL,
    ram_bytes INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_container_ts ON metrics_container(timestamp);

-- +goose Down

DROP TABLE IF EXISTS auth_user;
DROP TABLE IF EXISTS metrics_resource;
DROP TABLE IF EXISTS metrics_process;
DROP TABLE IF EXISTS metrics_container;
