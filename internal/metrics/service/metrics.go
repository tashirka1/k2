package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
	"github.com/tashirka1/k2/internal/metrics/storage"
)

type MetricsService interface {
	RunCollector(ctx context.Context, interval time.Duration) error
	QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error)
	QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error)
	QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error)
	SearchFTS(ctx context.Context, category string, query string) (interface{}, error)
}

type Metrics struct {
	r storage.MetricsStorage
}

func NewMetrics(r storage.MetricsStorage) *Metrics {
	return &Metrics{r: r}
}

func (s *Metrics) QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error) {
	return s.r.QueryResources(ctx, metricType, from, to)
}

func (s *Metrics) QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error) {
	return s.r.QueryProcesses(ctx, from, to)
}

func (s *Metrics) QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error) {
	return s.r.QueryContainers(ctx, from, to)
}

func (s *Metrics) SearchFTS(ctx context.Context, category string, query string) (interface{}, error) {
	switch category {
	case "resource":
		return s.r.SearchResourceFTS(ctx, query)
	case "process":
		return s.r.SearchProcessFTS(ctx, query)
	case "container":
		return s.r.SearchContainerFTS(ctx, query)
	default:
		return nil, nil
	}
}

func (s *Metrics) RunCollector(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("metrics collector started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("metrics collector stopped")
			return nil
		case <-ticker.C:
			if err := s.collectTick(ctx); err != nil {
				slog.Error("collector tick failed", "error", err)
			}
		}
	}
}

func (s *Metrics) collectTick(ctx context.Context) error {
	now := time.Now()

	resourcePoints := collectSystemMetrics(now)
	if len(resourcePoints) > 0 {
		if err := s.r.InsertResourceBatch(ctx, resourcePoints); err != nil {
			return err
		}
		if err := s.r.RebuildResourceFTS(ctx); err != nil {
			slog.Warn("rebuild resource fts failed", "error", err)
		}
	}

	processPoints := collectProcessMetrics(now)
	if len(processPoints) > 0 {
		if err := s.r.InsertProcessBatch(ctx, processPoints); err != nil {
			return err
		}
		if err := s.r.RebuildProcessFTS(ctx); err != nil {
			slog.Warn("rebuild process fts failed", "error", err)
		}
	}

	containerPoints := collectContainerMetrics(ctx, now)
	if len(containerPoints) > 0 {
		if err := s.r.InsertContainerBatch(ctx, containerPoints); err != nil {
			return err
		}
		if err := s.r.RebuildContainerFTS(ctx); err != nil {
			slog.Warn("rebuild container fts failed", "error", err)
		}
	}

	if err := s.r.PurgeOlderThan(ctx, 7*24*time.Hour); err != nil {
		slog.Warn("purge old data failed", "error", err)
	}

	return nil
}
