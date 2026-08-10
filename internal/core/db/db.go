package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/tashirka1/k2"

	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
)

func NewDB(path string) (*sql.DB, error) {
	// DSN params apply to EVERY new connection in the pool (unlike db.Exec,
	// which only affects a single connection)
	dsn := fmt.Sprintf(
		"file:%s"+
			"?_pragma=busy_timeout(10000)"+
			"&_pragma=foreign_keys(ON)"+
			"&_pragma=journal_mode(WAL)"+
			"&_pragma=synchronous(NORMAL)"+
			"&_pragma=temp_store(MEMORY)"+
			"&_pragma=cache_size(-1024)"+
			"&_pragma=auto_vacuum(FULL)"+
			"&_pragma=journal_size_limit(67110000)"+
			"&_pragma=page_size(4096)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// small pool keeps per-connection page cache (1 MiB) bounded
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// goose up
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migration: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	slog.Info("run migrations")

	goose.SetBaseFS(k2.EmbeddedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return err
	}

	slog.Info("migrations applied successfully")
	return nil
}
