package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
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
	PurgeOlderThan(ctx context.Context, age time.Duration) error
	RebuildResourceFTS(ctx context.Context) error
	RebuildProcessFTS(ctx context.Context) error
	RebuildContainerFTS(ctx context.Context) error
	SearchResourceFTS(ctx context.Context, query string) ([]model.ResourcePoint, error)
	SearchProcessFTS(ctx context.Context, query string) ([]model.ProcessPoint, error)
	SearchContainerFTS(ctx context.Context, query string) ([]model.ContainerPoint, error)
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
	defer tx.Rollback()

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
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics_process(timestamp, pid, name, cpu, ram) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.Timestamp, p.PID, p.Name, p.CPU, p.RAM); err != nil {
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
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO metrics_container(timestamp, name, image, cpu, ram) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.Timestamp, p.Name, p.Image, p.CPU, p.RAM); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, type, name, COALESCE(device, ''), value FROM metrics_resource WHERE type = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp", metricType, from.Format(time.RFC3339), to.Format(time.RFC3339))
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
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, pid, name, cpu, ram FROM metrics_process WHERE timestamp >= ? AND timestamp <= ? ORDER BY cpu DESC", from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.ProcessPoint
	for rows.Next() {
		var p model.ProcessPoint
		if err := rows.Scan(&p.Timestamp, &p.PID, &p.Name, &p.CPU, &p.RAM); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *Metrics) QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT timestamp, name, image, cpu, ram FROM metrics_container WHERE timestamp >= ? AND timestamp <= ? ORDER BY cpu DESC", from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.ContainerPoint
	for rows.Next() {
		var p model.ContainerPoint
		if err := rows.Scan(&p.Timestamp, &p.Name, &p.Image, &p.CPU, &p.RAM); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *Metrics) PurgeOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age).Format(time.RFC3339)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"metrics_resource", "metrics_process", "metrics_container"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table), cutoff); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Metrics) RebuildResourceFTS(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT type, name, COALESCE(device, '') FROM metrics_resource WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_resource)")
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM metrics_resource_fts"); err != nil {
		return err
	}

	for rows.Next() {
		var typ, name, device string
		if err := rows.Scan(&typ, &name, &device); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO metrics_resource_fts(type, name, device) VALUES (?, ?, ?)", typ, name, device); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Metrics) RebuildProcessFTS(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT name, CAST(pid AS TEXT) FROM metrics_process WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_process)")
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM metrics_process_fts"); err != nil {
		return err
	}

	for rows.Next() {
		var name, pid string
		if err := rows.Scan(&name, &pid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO metrics_process_fts(name, pid) VALUES (?, ?)", name, pid); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Metrics) RebuildContainerFTS(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT name, image FROM metrics_container WHERE timestamp = (SELECT MAX(timestamp) FROM metrics_container)")
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM metrics_container_fts"); err != nil {
		return err
	}

	for rows.Next() {
		var name, image string
		if err := rows.Scan(&name, &image); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO metrics_container_fts(name, image) VALUES (?, ?)", name, image); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Metrics) SearchResourceFTS(ctx context.Context, query string) ([]model.ResourcePoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT type, name, device FROM metrics_resource_fts WHERE metrics_resource_fts MATCH ?", query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ResourcePoint
	for rows.Next() {
		var p model.ResourcePoint
		if err := rows.Scan(&p.Type, &p.Name, &p.Device); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func (r *Metrics) SearchProcessFTS(ctx context.Context, query string) ([]model.ProcessPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT name, pid FROM metrics_process_fts WHERE metrics_process_fts MATCH ?", query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ProcessPoint
	for rows.Next() {
		var p model.ProcessPoint
		var pidStr string
		if err := rows.Scan(&p.Name, &pidStr); err != nil {
			return nil, err
		}
		p.PID, _ = strconv.Atoi(pidStr)
		results = append(results, p)
	}
	return results, rows.Err()
}

func (r *Metrics) SearchContainerFTS(ctx context.Context, query string) ([]model.ContainerPoint, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT name, image FROM metrics_container_fts WHERE metrics_container_fts MATCH ?", query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ContainerPoint
	for rows.Next() {
		var p model.ContainerPoint
		if err := rows.Scan(&p.Name, &p.Image); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

var _ MetricsStorage = (*Metrics)(nil)
