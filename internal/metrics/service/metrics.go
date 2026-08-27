package service

import (
	"context"
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
	SearchResource(ctx context.Context, query string) ([]model.ResourcePoint, error)
	SearchProcess(ctx context.Context, query string, s model.Sort) ([]model.ProcessPoint, error)
	SearchContainer(ctx context.Context, query string, s model.Sort) ([]model.ContainerPoint, error)
	QueryProcessChart(ctx context.Context, pid int, param string, from, to time.Time) (model.ChartData, error)
	QueryContainerChart(ctx context.Context, name, param string, from, to time.Time) (model.ChartData, error)
	RunMaintenance(ctx context.Context) error
}

type procCPUSample struct {
	ts         time.Time
	total      float64
	createTime int64
}

type containerCPUSample struct {
	ts     time.Time
	total  uint64
	system uint64
}

type Metrics struct {
	r         storage.MetricsStorage
	dc        *client.Client
	procCPU   map[int32]procCPUSample
	contCPU   map[string]containerCPUSample
	retention time.Duration
}

func NewMetrics(r storage.MetricsStorage, dc *client.Client, retention time.Duration) *Metrics {
	return &Metrics{
		r:         r,
		dc:        dc,
		retention: retention,
		procCPU:   make(map[int32]procCPUSample),
		contCPU:   make(map[string]containerCPUSample),
	}
}

func (s *Metrics) QueryResources(ctx context.Context, metricType string, from, to time.Time) ([]model.ResourcePoint, error) {
	buckets, err := s.r.QueryResources(ctx, metricType, from, to, 0)
	if err != nil {
		return nil, err
	}
	points := make([]model.ResourcePoint, len(buckets))
	for i, b := range buckets {
		points[i] = model.ResourcePoint{Timestamp: b.Timestamp, Type: metricType, Name: "percent", Value: b.Avg}
	}
	return points, nil
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
