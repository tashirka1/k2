package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/metrics/model"
	"github.com/tashirka1/k2/internal/metrics/storage"
)

var _ storage.MetricsStorage = (*mockMetricsStorage)(nil)

type mockMetricsStorage struct {
	processHistoryErr   error
	containerHistoryErr error
	purgeErr            error
	latestProcesses     []model.ProcessPoint
	latestContainers    []model.ContainerPoint
	processResults      []model.ProcessPoint
	containerResults    []model.ContainerPoint
	resources           []model.ResourcePoint
	processHistory      []model.ProcessPoint
	containerHistory    []model.ContainerPoint
	searchProcessCall   bool
	searchContainer     bool
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

func (m *mockMetricsStorage) PurgeOlderThan(_ context.Context, _ time.Duration) error {
	return m.purgeErr
}
func (m *mockMetricsStorage) QueryProcessHistory(_ context.Context, _ int, _, _ time.Time) ([]model.ProcessPoint, error) {
	return m.processHistory, m.processHistoryErr
}
func (m *mockMetricsStorage) QueryContainerHistory(_ context.Context, _ string, _, _ time.Time) ([]model.ContainerPoint, error) {
	return m.containerHistory, m.containerHistoryErr
}
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

	points, err := s.SearchProcess(context.Background(), "  ", model.Sort{})

	assert.NoError(t, err)
	assert.Equal(t, r.latestProcesses, points)
	assert.False(t, r.searchProcessCall)
}

func TestSearchProcess_NonEmptyUsesSearch(t *testing.T) {
	r := &mockMetricsStorage{processResults: []model.ProcessPoint{{PID: 1, Name: "nginx"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchProcess(context.Background(), "nginx", model.Sort{})

	assert.NoError(t, err)
	assert.Equal(t, r.processResults, points)
	assert.True(t, r.searchProcessCall)
}

func TestSearchContainer_EmptyQueryReturnsLatest(t *testing.T) {
	r := &mockMetricsStorage{latestContainers: []model.ContainerPoint{{Name: "web"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchContainer(context.Background(), "", model.Sort{})

	assert.NoError(t, err)
	assert.Equal(t, r.latestContainers, points)
	assert.False(t, r.searchContainer)
}

func TestSearchContainer_NonEmptyUsesSearch(t *testing.T) {
	r := &mockMetricsStorage{containerResults: []model.ContainerPoint{{Name: "db"}}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchContainer(context.Background(), "db", model.Sort{})

	assert.NoError(t, err)
	assert.Equal(t, r.containerResults, points)
	assert.True(t, r.searchContainer)
}

func TestSearchProcess_AppliesSort(t *testing.T) {
	r := &mockMetricsStorage{processResults: []model.ProcessPoint{
		{PID: 2, CPU: 5, Name: "b"},
		{PID: 1, CPU: 10, Name: "a"},
	}}
	s := NewMetrics(r, nil, 7*24*time.Hour)

	points, err := s.SearchProcess(context.Background(), "x", model.Sort{Field: "cpu", Desc: true})

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, []int{points[0].PID, points[1].PID})
}

func TestSortProcessPoints(t *testing.T) {
	tests := []struct {
		name    string
		points  []model.ProcessPoint
		sort    model.Sort
		wantPID []int
	}{
		{
			name:    "empty field keeps order",
			points:  []model.ProcessPoint{{PID: 2, CPU: 5, Name: "b"}, {PID: 1, CPU: 10, Name: "a"}},
			sort:    model.Sort{},
			wantPID: []int{2, 1},
		},
		{
			name:    "pid asc",
			points:  []model.ProcessPoint{{PID: 3}, {PID: 1}, {PID: 2}},
			sort:    model.Sort{Field: "pid"},
			wantPID: []int{1, 2, 3},
		},
		{
			name:    "cpu desc",
			points:  []model.ProcessPoint{{PID: 1, CPU: 5}, {PID: 2, CPU: 50}, {PID: 3, CPU: 25}},
			sort:    model.Sort{Field: "cpu", Desc: true},
			wantPID: []int{2, 3, 1},
		},
		{
			name:    "ram asc",
			points:  []model.ProcessPoint{{PID: 1, RAM: 5}, {PID: 2, RAM: 50}, {PID: 3, RAM: 25}},
			sort:    model.Sort{Field: "ram"},
			wantPID: []int{1, 3, 2},
		},
		{
			name:    "ram_bytes desc",
			points:  []model.ProcessPoint{{PID: 1, RAMBytes: 9000}, {PID: 2, RAMBytes: 100}, {PID: 3, RAMBytes: 500}},
			sort:    model.Sort{Field: "ram_bytes", Desc: true},
			wantPID: []int{1, 3, 2},
		},
		{
			name:    "name desc",
			points:  []model.ProcessPoint{{PID: 1, Name: "bash"}, {PID: 2, Name: "nginx"}, {PID: 3, Name: "chronyd"}},
			sort:    model.Sort{Field: "name", Desc: true},
			wantPID: []int{2, 3, 1},
		},
		{
			name:    "unknown field keeps order",
			points:  []model.ProcessPoint{{PID: 2}, {PID: 1}},
			sort:    model.Sort{Field: "nope"},
			wantPID: []int{2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortProcessPoints(tt.points, tt.sort)

			got := make([]int, len(tt.points))
			for i, p := range tt.points {
				got[i] = p.PID
			}
			assert.Equal(t, tt.wantPID, got)
		})
	}
}

func TestSortContainerPoints(t *testing.T) {
	tests := []struct {
		name   string
		points []model.ContainerPoint
		sort   model.Sort
		want   []string
	}{
		{
			name:   "empty field keeps order",
			points: []model.ContainerPoint{{Name: "web", CPU: 5}, {Name: "db", CPU: 10}},
			sort:   model.Sort{},
			want:   []string{"web", "db"},
		},
		{
			name:   "name asc",
			points: []model.ContainerPoint{{Name: "web"}, {Name: "db"}, {Name: "app"}},
			sort:   model.Sort{Field: "name"},
			want:   []string{"app", "db", "web"},
		},
		{
			name:   "image desc",
			points: []model.ContainerPoint{{Name: "web", Image: "nginx"}, {Name: "db", Image: "postgres"}, {Name: "app", Image: "alpine"}},
			sort:   model.Sort{Field: "image", Desc: true},
			want:   []string{"db", "web", "app"},
		},
		{
			name:   "cpu desc",
			points: []model.ContainerPoint{{Name: "web", CPU: 5}, {Name: "db", CPU: 50}, {Name: "app", CPU: 25}},
			sort:   model.Sort{Field: "cpu", Desc: true},
			want:   []string{"db", "app", "web"},
		},
		{
			name:   "ram_bytes asc",
			points: []model.ContainerPoint{{Name: "web", RAMBytes: 9000}, {Name: "db", RAMBytes: 100}, {Name: "app", RAMBytes: 500}},
			sort:   model.Sort{Field: "ram_bytes"},
			want:   []string{"db", "app", "web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortContainerPoints(tt.points, tt.sort)

			got := make([]string, len(tt.points))
			for i, p := range tt.points {
				got[i] = p.Name
			}
			assert.Equal(t, tt.want, got)
		})
	}
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

func TestBuildProcessChartData(t *testing.T) {
	cpu := 42.5
	ram := 63.1
	mib := 500.0

	tests := []struct {
		name       string
		param      string
		points     []model.ProcessPoint
		wantLabels []string
		wantSeries []model.ChartSeries
	}{
		{
			name:  "cpu series",
			param: "cpu",
			points: []model.ProcessPoint{
				{Timestamp: "2026-08-02T10:00:00Z", CPU: cpu},
				{Timestamp: "2026-08-02T10:00:01Z", CPU: cpu + 1},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z", "2026-08-02T10:00:01Z"},
			wantSeries: []model.ChartSeries{{Label: "CPU %", Data: []*float64{&cpu, ptr(cpu + 1)}}},
		},
		{
			name:  "ram series",
			param: "ram",
			points: []model.ProcessPoint{
				{Timestamp: "2026-08-02T10:00:00Z", RAM: ram},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "RAM %", Data: []*float64{&ram}}},
		},
		{
			name:  "ram_bytes converts to MiB",
			param: "ram_bytes",
			points: []model.ProcessPoint{
				{Timestamp: "2026-08-02T10:00:00Z", RAMBytes: 524288000},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "RAM Used (MB)", Data: []*float64{&mib}}},
		},
		{
			name:       "empty points yields empty labels",
			param:      "cpu",
			points:     []model.ProcessPoint{},
			wantLabels: []string{},
			wantSeries: []model.ChartSeries{{Label: "CPU %", Data: []*float64{}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProcessChartData(tt.param, tt.points)

			assert.Equal(t, tt.wantLabels, got.Labels)
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

func TestBuildContainerChartData(t *testing.T) {
	cpu := 10.0
	ram := 20.0
	mib := 100.0

	tests := []struct {
		name       string
		param      string
		points     []model.ContainerPoint
		wantLabels []string
		wantSeries []model.ChartSeries
	}{
		{
			name:  "cpu series",
			param: "cpu",
			points: []model.ContainerPoint{
				{Timestamp: "2026-08-02T10:00:00Z", CPU: cpu},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "CPU %", Data: []*float64{&cpu}}},
		},
		{
			name:  "ram series",
			param: "ram",
			points: []model.ContainerPoint{
				{Timestamp: "2026-08-02T10:00:00Z", RAM: ram},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "RAM %", Data: []*float64{&ram}}},
		},
		{
			name:  "ram_bytes converts to MiB",
			param: "ram_bytes",
			points: []model.ContainerPoint{
				{Timestamp: "2026-08-02T10:00:00Z", RAMBytes: 104857600},
			},
			wantLabels: []string{"2026-08-02T10:00:00Z"},
			wantSeries: []model.ChartSeries{{Label: "RAM Used (MB)", Data: []*float64{&mib}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildContainerChartData(tt.param, tt.points)

			assert.Equal(t, tt.wantLabels, got.Labels)
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

func ptr(v float64) *float64 { return &v }

func TestQueryProcessChart(t *testing.T) {
	points := []model.ProcessPoint{{Timestamp: "2026-08-02T10:00:00Z", CPU: 42.5}}

	tests := []struct {
		name    string
		mock    *mockMetricsStorage
		param   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "success",
			mock:    &mockMetricsStorage{processHistory: points},
			param:   "cpu",
			wantLen: 1,
		},
		{
			name:    "storage error",
			mock:    &mockMetricsStorage{processHistoryErr: context.DeadlineExceeded},
			param:   "cpu",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetrics(tt.mock, nil, 7*24*time.Hour)

			got, err := s.QueryProcessChart(context.Background(), 1, tt.param, time.Now().Add(-time.Hour), time.Now())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, got.Labels, tt.wantLen)
			assert.Equal(t, "CPU %", got.Series[0].Label)
		})
	}
}

func TestQueryContainerChart(t *testing.T) {
	points := []model.ContainerPoint{{Timestamp: "2026-08-02T10:00:00Z", RAMBytes: 104857600}}

	tests := []struct {
		name    string
		mock    *mockMetricsStorage
		param   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "success",
			mock:    &mockMetricsStorage{containerHistory: points},
			param:   "ram_bytes",
			wantLen: 1,
		},
		{
			name:    "storage error",
			mock:    &mockMetricsStorage{containerHistoryErr: context.DeadlineExceeded},
			param:   "ram_bytes",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetrics(tt.mock, nil, 7*24*time.Hour)

			got, err := s.QueryContainerChart(context.Background(), "web", tt.param, time.Now().Add(-time.Hour), time.Now())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, got.Labels, tt.wantLen)
			assert.Equal(t, "RAM Used (MB)", got.Series[0].Label)
		})
	}
}

func TestMaintenanceTick(t *testing.T) {
	tests := []struct {
		mock    *mockMetricsStorage
		name    string
		wantErr bool
	}{
		{
			name: "success",
			mock: &mockMetricsStorage{},
		},
		{
			name:    "purge error",
			mock:    &mockMetricsStorage{purgeErr: context.DeadlineExceeded},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetrics(tt.mock, nil, 7*24*time.Hour)

			err := s.maintenanceTick(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
