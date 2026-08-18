package service

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/tashirka1/k2/internal/metrics/model"
)

func (s *Metrics) collectProcessMetrics(now time.Time) []model.ProcessPoint {
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
		cpu, err := p.Times()
		if err != nil {
			continue
		}
		createTime, err := p.CreateTime()
		if err != nil {
			continue
		}
		pid := p.Pid
		cpuPercent := s.updateProcessCPU(pid, createTime, cpu.User+cpu.System, now)
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

func (s *Metrics) updateProcessCPU(pid int32, createTime int64, total float64, ts time.Time) float64 {
	prev, ok := s.procCPU[pid]
	if !ok || prev.createTime != createTime {
		s.procCPU[pid] = procCPUSample{total: total, createTime: createTime, ts: ts}
		return 0
	}
	percent := processCPUPercent(prev.total, total, prev.ts, ts, runtime.NumCPU())
	s.procCPU[pid] = procCPUSample{total: total, createTime: createTime, ts: ts}
	return percent
}

func processCPUPercent(prevTotal, total float64, prevTs, ts time.Time, numCPU int) float64 {
	elapsed := ts.Sub(prevTs).Seconds()
	if elapsed <= 0 || total < prevTotal {
		return 0
	}
	return (total - prevTotal) / elapsed * 100 / float64(numCPU)
}
