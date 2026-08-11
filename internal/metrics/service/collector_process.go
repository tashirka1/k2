package service

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/tashirka1/k2/internal/metrics/model"
)

func collectProcessMetrics(now time.Time) []model.ProcessPoint {
	ts := now.UTC().Format(time.RFC3339)
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	points := make([]model.ProcessPoint, 0, len(procs))
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			continue
		}
		// gopsutil v4 CPUPercent returns percentage relative to all cores
		cpuPercent = cpuPercent / float64(runtime.NumCPU())
		ramPercent, err := p.MemoryPercent()
		if err != nil {
			continue
		}
		memInfo, err := p.MemoryInfo()
		if err != nil {
			continue
		}
		points = append(points, model.ProcessPoint{
			Timestamp: ts,
			PID:       int(p.Pid),
			Name:      name,
			CPU:       cpuPercent,
			RAM:       float64(ramPercent),
			RAMBytes:  int64(memInfo.RSS),
		})
	}
	return points
}
