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
	QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error)
	QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error)
	QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error)
	QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error)
	QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error)
	QueryProcessHistory(ctx context.Context, pid int, from, to time.Time) ([]model.ProcessPoint, error)
	QueryContainerHistory(ctx context.Context, name string, from, to time.Time) ([]model.ContainerPoint, error)
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

func (r *Metrics) QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, type, name, COALESCE(device, ''), value FROM metrics_resource WHERE type = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp", metricType, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *Metrics) QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, pid, name, cpu, ram, ram_bytes FROM metrics_process WHERE timestamp >= ? AND timestamp <= ? ORDER BY cpu DESC", from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *Metrics) QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, name, image, cpu, ram, ram_bytes FROM metrics_container WHERE timestamp >= ? AND timestamp <= ? ORDER BY cpu DESC", from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *Metrics) QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, pid, name, cpu, ram, ram_bytes FROM metrics_process WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_process) ORDER BY cpu DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *Metrics) QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, name, image, cpu, ram, ram_bytes FROM metrics_container WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_container) ORDER BY cpu DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *Metrics) QueryProcessHistory(ctx context.Context, pid int, from, to time.Time) ([]model.ProcessPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, cpu, ram, ram_bytes FROM metrics_process WHERE pid = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp", pid, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.ProcessPoint
	for rows.Next() {
		var p model.ProcessPoint
		if err := rows.Scan(&p.Timestamp, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *Metrics) QueryContainerHistory(ctx context.Context, name string, from, to time.Time) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, cpu, ram, ram_bytes FROM metrics_container WHERE name = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp", name, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.ContainerPoint
	for rows.Next() {
		var p model.ContainerPoint
		if err := rows.Scan(&p.Timestamp, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
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

	var results []model.ResourcePoint
	for rows.Next() {
		var p model.ResourcePoint
		if err := rows.Scan(&p.Timestamp, &p.Type, &p.Name, &p.Device, &p.Value); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
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

	var results []model.ProcessPoint
	for rows.Next() {
		var p model.ProcessPoint
		if err := rows.Scan(&p.Timestamp, &p.PID, &p.Name, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
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

	var results []model.ContainerPoint
	for rows.Next() {
		var p model.ContainerPoint
		if err := rows.Scan(&p.Timestamp, &p.Name, &p.Image, &p.CPU, &p.RAM, &p.RAMBytes); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

var nonWordChars = regexp.MustCompile(`[^a-zA-Z0-9_\-\s]+`)

func sanitizeQuery(query string) string {
	return strings.TrimSpace(nonWordChars.ReplaceAllString(query, " "))
}

var _ MetricsStorage = (*Metrics)(nil)
