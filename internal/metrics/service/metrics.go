package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"
	"github.com/tashirka1/k2/internal/metrics/storage"

	"github.com/moby/moby/client"
)

type MetricsService interface {
	RunCollector(ctx context.Context, interval time.Duration) error
	QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error)
	QueryChartData(ctx context.Context, metricType string, from, to time.Time) (model.ChartData, error)
	QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error)
	QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error)
	SearchResource(ctx context.Context, query string) ([]model.ResourcePoint, error)
	SearchProcess(ctx context.Context, query string, s model.Sort) ([]model.ProcessPoint, error)
	SearchContainer(ctx context.Context, query string, s model.Sort) ([]model.ContainerPoint, error)
	QueryProcessChart(ctx context.Context, pid int, param string, from, to time.Time) (model.ChartData, error)
	QueryContainerChart(ctx context.Context, name, param string, from, to time.Time) (model.ChartData, error)
	RunMaintenance(ctx context.Context) error
}

type Metrics struct {
	r         storage.MetricsStorage
	dc        *client.Client
	retention time.Duration
}

func NewMetrics(r storage.MetricsStorage, dc *client.Client, retention time.Duration) *Metrics {
	return &Metrics{r: r, dc: dc, retention: retention}
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
	if metricType == "disk" {
		return buildDiskChartData(points)
	}

	values := make(map[string]map[string]float64)
	timestamps := make(map[string]bool)

	for _, p := range points {
		if p.Name != "percent" {
			continue
		}
		label := chartLabel(metricType)
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

func chartLabel(metricType string) string {
	switch metricType {
	case "cpu":
		return "CPU %"
	case "ram":
		return "RAM %"
	case "ram_bytes":
		return "RAM Used (MB)"
	}
	return metricType
}

func buildDiskChartData(points []model.ResourcePoint) model.ChartData {
	usedByTs := make(map[string]float64)
	totalByTs := make(map[string]float64)

	for _, p := range points {
		switch p.Name {
		case "used":
			usedByTs[p.Timestamp] += p.Value
		case "total":
			totalByTs[p.Timestamp] += p.Value
		}
	}

	labels := make([]string, 0, len(usedByTs))
	for ts := range usedByTs {
		labels = append(labels, ts)
	}
	sort.Strings(labels)

	series := make([]*float64, 0, len(labels))
	for _, ts := range labels {
		if total := totalByTs[ts]; total > 0 {
			v := usedByTs[ts] / total * 100
			series = append(series, &v)
		} else {
			series = append(series, nil)
		}
	}

	return model.ChartData{Labels: labels, Series: []model.ChartSeries{{Label: "Disk %", Data: series}}}
}

func (s *Metrics) QueryProcessChart(ctx context.Context, pid int, param string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryProcessHistory(ctx, pid, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return buildProcessChartData(param, points), nil
}

func (s *Metrics) QueryContainerChart(ctx context.Context, name, param string, from, to time.Time) (model.ChartData, error) {
	points, err := s.r.QueryContainerHistory(ctx, name, from, to)
	if err != nil {
		return model.ChartData{}, err
	}
	return buildContainerChartData(param, points), nil
}

func buildProcessChartData(param string, points []model.ProcessPoint) model.ChartData {
	labels := make([]string, 0, len(points))
	values := make([]*float64, 0, len(points))
	for _, p := range points {
		labels = append(labels, p.Timestamp)
		values = append(values, processValue(param, p))
	}
	return buildSeriesChartData(param, labels, values)
}

func buildContainerChartData(param string, points []model.ContainerPoint) model.ChartData {
	labels := make([]string, 0, len(points))
	values := make([]*float64, 0, len(points))
	for _, p := range points {
		labels = append(labels, p.Timestamp)
		values = append(values, containerValue(param, p))
	}
	return buildSeriesChartData(param, labels, values)
}

func buildSeriesChartData(param string, labels []string, values []*float64) model.ChartData {
	return model.ChartData{Labels: labels, Series: []model.ChartSeries{{Label: chartLabel(param), Data: values}}}
}

func processValue(param string, p model.ProcessPoint) *float64 {
	switch param {
	case "cpu":
		return &p.CPU
	case "ram":
		return &p.RAM
	case "ram_bytes":
		v := ramBytesToMiB(p.RAMBytes)
		return &v
	}
	return nil
}

func containerValue(param string, p model.ContainerPoint) *float64 {
	switch param {
	case "cpu":
		return &p.CPU
	case "ram":
		return &p.RAM
	case "ram_bytes":
		v := ramBytesToMiB(p.RAMBytes)
		return &v
	}
	return nil
}

func ramBytesToMiB(b int64) float64 {
	return float64(b) / 1048576
}

func (s *Metrics) QueryProcesses(ctx context.Context, from, to time.Time) ([]model.ProcessPoint, error) {
	return s.r.QueryProcesses(ctx, from, to)
}

func (s *Metrics) QueryContainers(ctx context.Context, from, to time.Time) ([]model.ContainerPoint, error) {
	return s.r.QueryContainers(ctx, from, to)
}

func (s *Metrics) SearchResource(ctx context.Context, query string) ([]model.ResourcePoint, error) {
	return s.r.SearchResource(ctx, query)
}

func (s *Metrics) SearchProcess(ctx context.Context, query string, sort model.Sort) ([]model.ProcessPoint, error) {
	var points []model.ProcessPoint
	var err error
	if strings.TrimSpace(query) == "" {
		points, err = s.r.QueryLatestProcesses(ctx)
	} else {
		points, err = s.r.SearchProcess(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	sortProcessPoints(points, sort)
	return points, nil
}

func (s *Metrics) SearchContainer(ctx context.Context, query string, sort model.Sort) ([]model.ContainerPoint, error) {
	var points []model.ContainerPoint
	var err error
	if strings.TrimSpace(query) == "" {
		points, err = s.r.QueryLatestContainers(ctx)
	} else {
		points, err = s.r.SearchContainer(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	sortContainerPoints(points, sort)
	return points, nil
}

func (s *Metrics) RunCollector(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("metrics collector started", "interval", interval)

	if err := s.collectTick(ctx); err != nil {
		slog.Error("initial collector tick failed", "error", err)
	}

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
	}

	processPoints := collectProcessMetrics(now)
	if len(processPoints) > 0 {
		if err := s.r.InsertProcessBatch(ctx, processPoints); err != nil {
			return err
		}
	}

	containerPoints := collectContainerMetrics(ctx, now, s.dc)
	if len(containerPoints) > 0 {
		if err := s.r.InsertContainerBatch(ctx, containerPoints); err != nil {
			return err
		}
	}

	return nil
}

func (s *Metrics) RunMaintenance(ctx context.Context) error {
	s.runTick(ctx)

	for {
		if err := waitUntilMidnight(ctx); err != nil {
			return nil
		}
		s.runTick(ctx)
	}
}

func (s *Metrics) runTick(ctx context.Context) {
	if err := s.maintenanceTick(ctx); err != nil {
		slog.Warn("maintenance tick failed", "error", err)
		return
	}
	slog.Info("maintenance tick completed")
}

func (s *Metrics) maintenanceTick(ctx context.Context) error {
	if err := s.r.PurgeOlderThan(ctx, s.retention); err != nil {
		return err
	}
	return s.r.Vacuum(ctx)
}

func waitUntilMidnight(ctx context.Context) error {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
