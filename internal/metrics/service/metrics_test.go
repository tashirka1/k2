package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/metrics/model"
)

type mockMetricsStorage struct {
	latestProcesses   []model.ProcessPoint
	latestContainers  []model.ContainerPoint
	processResults    []model.ProcessPoint
	containerResults  []model.ContainerPoint
	resources         []model.ResourcePoint
	searchProcessCall bool
	searchContainer   bool
}

func (m *mockMetricsStorage) InsertResourceBatch(_ context.Context, _ []model.ResourcePoint) error {
	return nil
}
func (m *mockMetricsStorage) InsertProcessBatch(_ context.Context, _ []model.ProcessPoint) error {
	return nil
}
func (m *mockMetricsStorage) InsertContainerBatch(_ context.Context, _ []model.ContainerPoint) error {
	return nil
}
func (m *mockMetricsStorage) QueryResources(_ context.Context, _ string, _, _ time.Time) ([]model.ResourcePoint, error) {
	return m.resources, nil
}
func (m *mockMetricsStorage) QueryProcesses(_ context.Context, _, _ time.Time) ([]model.ProcessPoint, error) {
	return nil, nil
}
func (m *mockMetricsStorage) QueryContainers(_ context.Context, _, _ time.Time) ([]model.ContainerPoint, error) {
	return nil, nil
}
func (m *mockMetricsStorage) QueryLatestProcesses(_ context.Context) ([]model.ProcessPoint, error) {
	return m.latestProcesses, nil
}
func (m *mockMetricsStorage) QueryLatestContainers(_ context.Context) ([]model.ContainerPoint, error) {
	return m.latestContainers, nil
}
func (m *mockMetricsStorage) PurgeOlderThan(_ context.Context, _ time.Duration) error { return nil }
func (m *mockMetricsStorage) SearchResource(_ context.Context, _ string) ([]model.ResourcePoint, error) {
	return nil, nil
}
func (m *mockMetricsStorage) SearchProcess(_ context.Context, _ string) ([]model.ProcessPoint, error) {
	m.searchProcessCall = true
	return m.processResults, nil
}
func (m *mockMetricsStorage) SearchContainer(_ context.Context, _ string) ([]model.ContainerPoint, error) {
	m.searchContainer = true
	return m.containerResults, nil
}

func TestSearchProcess_EmptyQueryReturnsLatest(t *testing.T) {
	r := &mockMetricsStorage{latestProcesses: []model.ProcessPoint{{PID: 1, Name: "bash"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchProcess(context.Background(), "  ")

	assert.NoError(t, err)
	assert.Equal(t, r.latestProcesses, points)
	assert.False(t, r.searchProcessCall)
}

func TestSearchProcess_NonEmptyUsesSearch(t *testing.T) {
	r := &mockMetricsStorage{processResults: []model.ProcessPoint{{PID: 1, Name: "nginx"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchProcess(context.Background(), "nginx")

	assert.NoError(t, err)
	assert.Equal(t, r.processResults, points)
	assert.True(t, r.searchProcessCall)
}

func TestSearchContainer_EmptyQueryReturnsLatest(t *testing.T) {
	r := &mockMetricsStorage{latestContainers: []model.ContainerPoint{{Name: "web"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchContainer(context.Background(), "")

	assert.NoError(t, err)
	assert.Equal(t, r.latestContainers, points)
	assert.False(t, r.searchContainer)
}

func TestSearchContainer_NonEmptyUsesSearch(t *testing.T) {
	r := &mockMetricsStorage{containerResults: []model.ContainerPoint{{Name: "db"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchContainer(context.Background(), "db")

	assert.NoError(t, err)
	assert.Equal(t, r.containerResults, points)
	assert.True(t, r.searchContainer)
}

func TestBuildChartData(t *testing.T) {
	cpu := 42.5
	ram := 63.1
	diskUsed := 50.0

	tests := []struct {
		name       string
		metricType string
		points     []model.ResourcePoint
		wantLabels []string
		wantSeries []model.ChartSeries
	}{
		{
			name:       "cpu keeps only percent",
			metricType: "cpu",
			points: []model.ResourcePoint{
				{Timestamp: "2026-08-02T10:00:00Z", Type: "cpu", Name: "percent", Value: cpu},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "cpu", Name: "used", Value: 1},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "CPU %", Data: []*float64{&cpu}}},
		},
		{
			name:       "ram keeps only percent",
			metricType: "ram",
			points: []model.ResourcePoint{
				{Timestamp: "2026-08-02T10:00:00Z", Type: "ram", Name: "percent", Value: ram},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "ram", Name: "total", Value: 8192},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "ram", Name: "used", Value: 4096},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "RAM %", Data: []*float64{&ram}}},
		},
		{
			name:       "no points yields empty non-nil series",
			metricType: "ram",
			points:     []model.ResourcePoint{},
			wantLabels: []string{},
			wantSeries: []model.ChartSeries{},
		},
		{
			name:       "disk aggregates used/total across devices",
			metricType: "disk",
			points: []model.ResourcePoint{
				{Timestamp: "2026-08-02T10:00:00Z", Type: "disk", Name: "used", Device: "/", Value: 50},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "disk", Name: "total", Device: "/", Value: 100},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "disk", Name: "used", Device: "/data", Value: 25},
				{Timestamp: "2026-08-02T10:00:00Z", Type: "disk", Name: "total", Device: "/data", Value: 50},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "Disk %", Data: []*float64{&diskUsed}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildChartData(tt.metricType, tt.points)

			assert.Equal(t, tt.wantLabels, got.Labels)
			assert.NotNil(t, got.Series)
			assert.Len(t, got.Series, len(tt.wantSeries))
			for i, want := range tt.wantSeries {
				assert.Equal(t, want.Label, got.Series[i].Label)
				assert.Len(t, got.Series[i].Data, len(want.Data))
				for j := range want.Data {
					assert.Equal(t, *want.Data[j], *got.Series[i].Data[j])
				}
			}
		})
	}
}
