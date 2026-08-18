package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/metrics/model"
)

func TestChartThreshold(t *testing.T) {
	tests := []struct {
		name     string
		period   time.Duration
		wantZero bool
		want     int
	}{
		{name: "one hour no downsample", period: time.Hour, want: 120},
		{name: "six hours", period: 6 * time.Hour, want: 360},
		{name: "one day", period: 24 * time.Hour, want: 288},
		{name: "seven days", period: 168 * time.Hour, want: 336},
		{name: "one month", period: 720 * time.Hour, want: 360},
		{name: "zero period", period: 0, wantZero: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chartThreshold(tt.period)

			if tt.wantZero {
				assert.Zero(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLttbIndices(t *testing.T) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = float64(i)
	}

	tests := []struct {
		name      string
		values    []float64
		threshold int
		wantNil   bool
		wantLen   int
		wantFirst int
		wantLast  int
	}{
		{
			name:      "threshold greater than length",
			values:    values,
			threshold: 2000,
			wantNil:   true,
		},
		{
			name:      "threshold equals length",
			values:    values,
			threshold: 1000,
			wantNil:   true,
		},
		{
			name:      "threshold below three",
			values:    values,
			threshold: 2,
			wantNil:   true,
		},
		{
			name:      "downsampled preserves endpoints",
			values:    values,
			threshold: 100,
			wantLen:   100,
			wantFirst: 0,
			wantLast:  999,
		},
		{
			name:      "flat line",
			values:    make([]float64, 100),
			threshold: 10,
			wantLen:   10,
			wantFirst: 0,
			wantLast:  99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lttbIndices(tt.values, tt.threshold)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Len(t, got, tt.wantLen)
			assert.Equal(t, tt.wantFirst, got[0])
			assert.Equal(t, tt.wantLast, got[len(got)-1])
			for i := 1; i < len(got); i++ {
				assert.Greater(t, got[i], got[i-1])
			}
		})
	}
}

func TestDownsampleChartData(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }

	longLabels := make([]string, 1000)
	longSeries := make([]*float64, 1000)
	for i := range longLabels {
		longLabels[i] = time.Unix(int64(i), 0).UTC().Format(time.RFC3339)
		v := float64(i)
		longSeries[i] = &v
	}

	tests := []struct {
		name      string
		data      model.ChartData
		threshold int
		wantSame  bool
		wantLen   int
	}{
		{
			name:      "empty data unchanged",
			data:      model.ChartData{},
			threshold: 100,
			wantSame:  true,
		},
		{
			name:      "short data unchanged",
			data:      model.ChartData{Labels: longLabels[:100], Series: []model.ChartSeries{{Data: longSeries[:100]}}},
			threshold: 100,
			wantSame:  true,
		},
		{
			name:      "downsampled",
			data:      model.ChartData{Labels: longLabels, Series: []model.ChartSeries{{Label: "CPU %", Data: longSeries}}},
			threshold: 100,
			wantLen:   100,
		},
		{
			name: "nil values preserved",
			data: model.ChartData{
				Labels: longLabels[:200],
				Series: []model.ChartSeries{{Label: "Disk %", Data: append([]*float64{ptr(1)}, longSeries[1:200]...)}},
			},
			threshold: 50,
			wantLen:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := downsampleChartData(tt.data, tt.threshold)

			if tt.wantSame {
				assert.Equal(t, tt.data, got)
				return
			}
			assert.Len(t, got.Labels, tt.wantLen)
			for _, s := range got.Series {
				assert.Len(t, s.Data, tt.wantLen)
			}
			assert.Equal(t, tt.data.Labels[0], got.Labels[0])
			assert.Equal(t, tt.data.Labels[len(tt.data.Labels)-1], got.Labels[len(got.Labels)-1])
		})
	}
}
