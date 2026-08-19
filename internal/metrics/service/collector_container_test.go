package service

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContainerCPUPercent(t *testing.T) {
	tests := []struct {
		name        string
		prevTotal   uint64
		total       uint64
		prevSystem  uint64
		system      uint64
		onlineCPUs  uint32
		wantPercent float64
	}{
		{
			name:        "one core busy on 8-core host",
			prevTotal:   7e9,
			total:       8e9,
			prevSystem:  0,
			system:      8e9,
			onlineCPUs:  8,
			wantPercent: 100,
		},
		{
			name:        "half core on 8-core host",
			prevTotal:   4e9,
			total:       8e9,
			prevSystem:  0,
			system:      64e9,
			onlineCPUs:  8,
			wantPercent: 50,
		},
		{
			name:        "full host on 8-core host",
			prevTotal:   64e9,
			total:       128e9,
			prevSystem:  0,
			system:      64e9,
			onlineCPUs:  8,
			wantPercent: 800,
		},
		{
			name:        "two cores busy on 4-core host",
			prevTotal:   2e9,
			total:       4e9,
			prevSystem:  0,
			system:      4e9,
			onlineCPUs:  4,
			wantPercent: 200,
		},
		{
			name:        "first sample skipped",
			prevTotal:   0,
			total:       0,
			prevSystem:  0,
			system:      1e9,
			onlineCPUs:  8,
			wantPercent: 0,
		},
		{
			name:        "counter reset skipped",
			prevTotal:   5e9,
			total:       2e9,
			prevSystem:  1e9,
			system:      3e9,
			onlineCPUs:  8,
			wantPercent: 0,
		},
		{
			name:        "zero system delta skipped",
			prevTotal:   1e9,
			total:       2e9,
			prevSystem:  1e9,
			system:      1e9,
			onlineCPUs:  8,
			wantPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containerCPUPercent(tt.prevTotal, tt.total, tt.prevSystem, tt.system, tt.onlineCPUs)

			assert.InDelta(t, tt.wantPercent, got, 0.001)
		})
	}
}

func TestUpdateContainerCPU(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		firstTotal   uint64
		firstSystem  uint64
		secondTotal  uint64
		secondSystem uint64
		wantPercent  float64
	}{
		{
			name:         "first sample returns zero and stores baseline",
			firstTotal:   0,
			firstSystem:  0,
			secondTotal:  8e9,
			secondSystem: 8e9,
			wantPercent:  0,
		},
		{
			name:         "second tick computes delta",
			firstTotal:   4e9,
			firstSystem:  4e9,
			secondTotal:  8e9,
			secondSystem: 8e9,
			wantPercent:  100 * float64(runtime.NumCPU()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetrics(nil, nil, 0)
			first := s.updateContainerCPU("c1", tt.firstTotal, tt.firstSystem, now)
			assert.Zero(t, first)

			second := s.updateContainerCPU("c1", tt.secondTotal, tt.secondSystem, now.Add(time.Second))
			assert.InDelta(t, tt.wantPercent, second, 0.001)
		})
	}
}

func TestContainerCPUPrunesRemovedContainers(t *testing.T) {
	now := time.Now()
	s := NewMetrics(nil, nil, 0)

	s.updateContainerCPU("gone", 1e9, 1e9, now)
	s.updateContainerCPU("kept", 1e9, 1e9, now)

	assert.Len(t, s.contCPU, 2)
	s.pruneContainerCPU(map[string]bool{"kept": true})
	assert.Len(t, s.contCPU, 1)
	_, ok := s.contCPU["gone"]
	assert.False(t, ok)
}
