package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
	"github.com/tashirka1/k2/internal/metrics/storage"
)

type MetricsService interface {
	RunCollector(ctx context.Context, interval time.Duration) error
	QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error)
	QueryChartData(ctx context.Context, metricType string, from, to time.Time) (model.ChartData, error)
	QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error)
	QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error)
	QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error)
	QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error)
	SearchResourceFTS(ctx context.Context, query string) ([]model.ResourcePoint, error)
	SearchProcessFTS(ctx context.Context, query string) ([]model.ProcessPoint, error)
	SearchContainerFTS(ctx context.Context, query string) ([]model.ContainerPoint, error)
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

func (s *Metrics) QueryChartData(ctx context.Context, metricType string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryResources(ctx, metricType, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return buildChartData(metricType, points), nil
}

func buildChartData(metricType string, points []model.ResourcePoint) model.ChartData {
	values := make(map[string]map[string]float64)
	timestamps := make(map[string]bool)

	for _, p := range points {
		if p.Name != "percent" {
			continue
		}
		label := metricType
		if metricType == "disk" {
			label = p.Device
		}
		if values[label] == nil {
			values[label] = make(map[string]float64)
		}
		values[label][p.Timestamp] = p.Value
		timestamps[p.Timestamp] = true
	}

	labels := make([]string, 0, len(timestamps))
	for ts := range timestamps {
		labels = append(labels, ts)
	}
	sort.Strings(labels)

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	data := model.ChartData{Labels: labels, Series: []model.ChartSeries{}}
	for _, name := range names {
		series := make([]*float64, 0, len(labels))
		for _, ts := range labels {
			if v, ok := values[name][ts]; ok {
				val := v
				series = append(series, &val)
			} else {
				series = append(series, nil)
			}
		}
		data.Series = append(data.Series, model.ChartSeries{Label: name, Data: series})
	}
	return data
}

func (s *Metrics) QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error) {
	return s.r.QueryProcesses(ctx, from, to)
}

func (s *Metrics) QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error) {
	return s.r.QueryContainers(ctx, from, to)
}

func (s *Metrics) QueryLatestProcesses(ctx context.Context) ([]model.ProcessPoint, error) {
	return s.r.QueryLatestProcesses(ctx)
}

func (s *Metrics) QueryLatestContainers(ctx context.Context) ([]model.ContainerPoint, error) {
	return s.r.QueryLatestContainers(ctx)
}

func (s *Metrics) SearchResourceFTS(ctx context.Context, query string) ([]model.ResourcePoint, error) {
	return s.r.SearchResourceFTS(ctx, query)
}

func (s *Metrics) SearchProcessFTS(ctx context.Context, query string) ([]model.ProcessPoint, error) {
	if strings.TrimSpace(query) == "" {
		return s.r.QueryLatestProcesses(ctx)
	}
	return s.r.SearchProcessFTS(ctx, query)
}

func (s *Metrics) SearchContainerFTS(ctx context.Context, query string) ([]model.ContainerPoint, error) {
	if strings.TrimSpace(query) == "" {
		return s.r.QueryLatestContainers(ctx)
	}
	return s.r.SearchContainerFTS(ctx, query)
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
