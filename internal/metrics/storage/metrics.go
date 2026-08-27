package storage

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
)

type MetricsStorage interface {
	InsertResourceBatch(ctx context.Context, points []model.ResourcePoint) error
	InsertProcessBatch(ctx context.Context, points []model.ProcessPoint) error
	InsertContainerBatch(ctx context.Context, points []model.ContainerPoint) error
	QueryResources(ctx context.Context, metricType string, from, to time.Time, bucketSec int) ([]model.ResourceBucket, error)
	QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error)
	QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error)
	QueryProcessHistory(ctx context.Context, pid int, from, to time.Time, bucketSec int) ([]model.ProcessBucket, error)
	QueryContainerHistory(ctx context.Context, name string, from, to time.Time, bucketSec int) ([]model.ContainerBucket, error)
	PurgeOlderThan(ctx context.Context, age time.Duration) error
	SearchResource(ctx context.Context, query string) ([]model.ResourcePoint, error)
	SearchProcess(ctx context.Context, query string) ([]model.ProcessPoint, error)
	SearchContainer(ctx context.Context, query string) ([]model.ContainerPoint, error)
}

type Metrics struct {
	db *sql.DB
}

func NewMetrics(db *sql.DB) *Metrics {
	return &Metrics{db: db}
}

func (r *Metrics) InsertResourceBatch(ctx context.Context, points []model.ResourcePoint) error {
	if len(points) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics_resource(timestamp, type, name, device, value) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.Timestamp, p.Type, p.Name, p.Device, p.Value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) InsertProcessBatch(ctx context.Context, points []model.ProcessPoint) error {
	if len(points) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics_process(timestamp, pid, name, cpu, ram, ram_bytes) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.Timestamp, p.PID, p.Name, p.CPU, p.RAM, p.RAMBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) InsertContainerBatch(ctx context.Context, points []model.ContainerPoint) error {
	if len(points) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics_container(timestamp, name, image, cpu, ram, ram_bytes) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.Timestamp, p.Name, p.Image, p.CPU, p.RAM, p.RAMBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) QueryResources(ctx context.Context, metricType string, from, to time.Time, bucketSec int) ([]model.ResourceBucket, error) {
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)
	if metricType == "disk" {
		if bucketSec > 0 {
			rows, err := r.db.QueryContext(ctx, `SELECT strftime('%Y-%m-%dT%H:%M:%SZ', CAST(strftime('%s', ts)/? AS INTEGER)*?,'unixepoch') AS bucket, AVG(pct), MIN(pct), MAX(pct) FROM (
				SELECT timestamp AS ts, SUM(CASE WHEN name='used' THEN value ELSE 0 END)*100.0 / NULLIF(SUM(CASE WHEN name='total' THEN value ELSE 0 END),0) AS pct
				FROM metrics_resource WHERE type='disk' AND timestamp BETWEEN ? AND ? GROUP BY timestamp
			) GROUP BY bucket ORDER BY bucket`, bucketSec, bucketSec, fromStr, toStr)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			return scanResourceBuckets(rows)
		}
		rows, err := r.db.QueryContext(ctx, `SELECT timestamp, SUM(CASE WHEN name='used' THEN value ELSE 0 END)*100.0 / NULLIF(SUM(CASE WHEN name='total' THEN value ELSE 0 END),0) AS pct, SUM(CASE WHEN name='used' THEN value ELSE 0 END)*100.0 / NULLIF(SUM(CASE WHEN name='total' THEN value ELSE 0 END),0), SUM(CASE WHEN name='used' THEN value ELSE 0 END)*100.0 / NULLIF(SUM(CASE WHEN name='total' THEN value ELSE 0 END),0)
			FROM metrics_resource WHERE type='disk' AND timestamp BETWEEN ? AND ? GROUP BY timestamp ORDER BY timestamp`, fromStr, toStr)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanResourceBuckets(rows)
	}
	if bucketSec > 0 {
		rows, err := r.db.QueryContext(ctx, `SELECT strftime('%Y-%m-%dT%H:%M:%SZ', CAST(strftime('%s',timestamp)/? AS INTEGER)*?,'unixepoch') AS bucket, AVG(value), MIN(value), MAX(value)
			FROM metrics_resource WHERE type=? AND name='percent' AND timestamp BETWEEN ? AND ? GROUP BY bucket ORDER BY bucket`, bucketSec, bucketSec, metricType, fromStr, toStr)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanResourceBuckets(rows)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, value, value, value FROM metrics_resource WHERE type=? AND name='percent' AND timestamp BETWEEN ? AND ? ORDER BY timestamp`, metricType, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResourceBuckets(rows)
}

func (r *Metrics) QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, pid, name, cpu, ram, ram_bytes FROM metrics_process WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_process) ORDER BY cpu DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProcessPoints(rows)
}

func (r *Metrics) QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, name, image, cpu, ram, ram_bytes FROM metrics_container WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_container) ORDER BY cpu DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanContainerPoints(rows)
}

func (r *Metrics) QueryProcessHistory(ctx context.Context, pid int, from, to time.Time, bucketSec int) ([]model.ProcessBucket, error) {
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)
	if bucketSec > 0 {
		rows, err := r.db.QueryContext(ctx, `SELECT strftime('%Y-%m-%dT%H:%M:%SZ', CAST(strftime('%s',timestamp)/? AS INTEGER)*?,'unixepoch') AS bucket,
			AVG(cpu), MIN(cpu), MAX(cpu), AVG(ram), MIN(ram), MAX(ram), CAST(AVG(ram_bytes) AS INTEGER), MIN(ram_bytes), MAX(ram_bytes)
			FROM metrics_process WHERE pid=? AND timestamp BETWEEN ? AND ? GROUP BY bucket ORDER BY bucket`, bucketSec, bucketSec, pid, fromStr, toStr)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanProcessBuckets(rows)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, cpu, cpu, cpu, ram, ram, ram, ram_bytes, ram_bytes, ram_bytes FROM metrics_process WHERE pid=? AND timestamp BETWEEN ? AND ? ORDER BY timestamp`, pid, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProcessBuckets(rows)
}

func (r *Metrics) QueryContainerHistory(ctx context.Context, name string, from, to time.Time, bucketSec int) ([]model.ContainerBucket, error) {
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)
	if bucketSec > 0 {
		rows, err := r.db.QueryContext(ctx, `SELECT strftime('%Y-%m-%dT%H:%M:%SZ', CAST(strftime('%s',timestamp)/? AS INTEGER)*?,'unixepoch') AS bucket,
			AVG(cpu), MIN(cpu), MAX(cpu), AVG(ram), MIN(ram), MAX(ram), CAST(AVG(ram_bytes) AS INTEGER), MIN(ram_bytes), MAX(ram_bytes)
			FROM metrics_container WHERE name=? AND timestamp BETWEEN ? AND ? GROUP BY bucket ORDER BY bucket`, bucketSec, bucketSec, name, fromStr, toStr)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanContainerBuckets(rows)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, cpu, cpu, cpu, ram, ram, ram, ram_bytes, ram_bytes, ram_bytes FROM metrics_container WHERE name=? AND timestamp BETWEEN ? AND ? ORDER BY timestamp`, name, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContainerBuckets(rows)
}

func (r *Metrics) PurgeOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().UTC().Add(-age).Format(time.RFC3339)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range []string{"metrics_resource", "metrics_process", "metrics_container"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table), cutoff); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) SearchResource(ctx context.Context, query string) ([]model.ResourcePoint, error) {
	q := sanitizeQuery(query)
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, type, name, COALESCE(device, ''), value
		FROM metrics_resource
		WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_resource)
		  AND (name LIKE ? OR type LIKE ? OR COALESCE(device, '') LIKE ?)
		ORDER BY value DESC LIMIT 50`, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanResourcePoints(rows)
}

func (r *Metrics) SearchProcess(ctx context.Context, query string) ([]model.ProcessPoint, error) {
	q := sanitizeQuery(query)
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, pid, name, cpu, ram, ram_bytes
		FROM metrics_process
		WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_process)
		  AND (name LIKE ? OR CAST(pid AS TEXT) LIKE ?)
		ORDER BY cpu DESC LIMIT 50`, "%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProcessPoints(rows)
}

func (r *Metrics) SearchContainer(ctx context.Context, query string) ([]model.ContainerPoint, error) {
	q := sanitizeQuery(query)
	rows, err := r.db.QueryContext(ctx, `SELECT timestamp, name, image, cpu, ram, ram_bytes
		FROM metrics_container
		WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_container)
		  AND (name LIKE ? OR image LIKE ?)
		ORDER BY cpu DESC LIMIT 50`, "%"+q+"%", "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanContainerPoints(rows)
}

var nonWordChars = regexp.MustCompile(`[^a-zA-Z0-9_\-\s]+`)

func sanitizeQuery(query string) string {
	return strings.TrimSpace(nonWordChars.ReplaceAllString(query, " "))
}

func scanResourcePoints(rows *sql.Rows) ([]model.ResourcePoint, error) {
	var points []model.ResourcePoint
	for rows.Next() {
		var p model.ResourcePoint
		if err := rows.Scan(&p.Timestamp, &p.Type, &p.Name, &p.Device, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func scanProcessPoints(rows *sql.Rows) ([]model.ProcessPoint, error) {
	var points []model.ProcessPoint
	for rows.Next() {
		var p model.ProcessPoint
		if err := rows.Scan(&p.Timestamp, &p.PID, &p.Name, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func scanContainerPoints(rows *sql.Rows) ([]model.ContainerPoint, error) {
	var points []model.ContainerPoint
	for rows.Next() {
		var p model.ContainerPoint
		if err := rows.Scan(&p.Timestamp, &p.Name, &p.Image, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func scanResourceBuckets(rows *sql.Rows) ([]model.ResourceBucket, error) {
	var buckets []model.ResourceBucket
	for rows.Next() {
		var b model.ResourceBucket
		if err := rows.Scan(&b.Timestamp, &b.Avg, &b.Min, &b.Max); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func scanProcessBuckets(rows *sql.Rows) ([]model.ProcessBucket, error) {
	var buckets []model.ProcessBucket
	for rows.Next() {
		var b model.ProcessBucket
		if err := rows.Scan(&b.Timestamp, &b.CPUAvg, &b.CPUMin, &b.CPUMax, &b.RAMAvg, &b.RAMMin, &b.RAMMax, &b.RAMBytesAvg, &b.RAMBytesMin, &b.RAMBytesMax); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func scanContainerBuckets(rows *sql.Rows) ([]model.ContainerBucket, error) {
	var buckets []model.ContainerBucket
	for rows.Next() {
		var b model.ContainerBucket
		if err := rows.Scan(&b.Timestamp, &b.CPUAvg, &b.CPUMin, &b.CPUMax, &b.RAMAvg, &b.RAMMin, &b.RAMMax, &b.RAMBytesAvg, &b.RAMBytesMin, &b.RAMBytesMax); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

var _ MetricsStorage = (*Metrics)(nil)
