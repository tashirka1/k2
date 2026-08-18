package service

import (
	"testing"

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
