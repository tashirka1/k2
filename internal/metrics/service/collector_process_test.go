package service

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProcessCPUPercent(t *testing.T) {
	t0 := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(10 * time.Second)

	tests := []struct {
		prevTs      time.Time
		ts          time.Time
		name        string
		prevTotal   float64
		total       float64
		numCPU      int
		wantPercent float64
	}{
		{
			name:        "full core on 8-core host",
			prevTotal:   10,
			total:       20,
			prevTs:      t0,
			ts:          t1,
			numCPU:      8,
			wantPercent: 12.5,
		},
		{
			name:        "full core on 1-core host",
			prevTotal:   10,
			total:       20,
			prevTs:      t0,
			ts:          t1,
			numCPU:      1,
			wantPercent: 100,
		},
		{
			name:        "counter reset",
			prevTotal:   20,
			total:       10,
			prevTs:      t0,
			ts:          t1,
			numCPU:      8,
			wantPercent: 0,
		},
		{
			name:        "zero elapsed",
			prevTotal:   10,
			total:       20,
			prevTs:      t0,
			ts:          t0,
			numCPU:      8,
			wantPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processCPUPercent(tt.prevTotal, tt.total, tt.prevTs, tt.ts, tt.numCPU)

			assert.InDelta(t, tt.wantPercent, got, 0.001)
		})
	}
}

func TestUpdateProcessCPU(t *testing.T) {
	t0 := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(10 * time.Second)

	tests := []struct {
		ts          time.Time
		setup       func() *Metrics
		name        string
		wantCached  procCPUSample
		createTime  int64
		total       float64
		wantPercent float64
		pid         int32
	}{
		{
			name:        "first sample returns zero and caches",
			setup:       func() *Metrics { return NewMetrics(&mockMetricsStorage{}, nil, time.Hour) },
			pid:         42,
			createTime:  1000,
			total:       10,
			ts:          t0,
			wantPercent: 0,
			wantCached:  procCPUSample{total: 10, createTime: 1000, ts: t0},
		},
		{
			name: "pid reuse with new createTime resets",
			setup: func() *Metrics {
				s := NewMetrics(&mockMetricsStorage{}, nil, time.Hour)
				s.procCPU[42] = procCPUSample{total: 10, createTime: 1000, ts: t0}
				return s
			},
			pid:         42,
			createTime:  2000,
			total:       5,
			ts:          t1,
			wantPercent: 0,
			wantCached:  procCPUSample{total: 5, createTime: 2000, ts: t1},
		},
		{
			name: "delta over interval",
			setup: func() *Metrics {
				s := NewMetrics(&mockMetricsStorage{}, nil, time.Hour)
				s.procCPU[42] = procCPUSample{total: 10, createTime: 1000, ts: t0}
				return s
			},
			pid:         42,
			createTime:  1000,
			total:       20,
			ts:          t1,
			wantPercent: processCPUPercent(10, 20, t0, t1, runtime.NumCPU()),
			wantCached:  procCPUSample{total: 20, createTime: 1000, ts: t1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()

			got := s.updateProcessCPU(tt.pid, tt.createTime, tt.total, tt.ts)

			assert.InDelta(t, tt.wantPercent, got, 0.001)
			assert.Equal(t, tt.wantCached, s.procCPU[tt.pid])
		})
	}
}
