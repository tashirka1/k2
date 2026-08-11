package service

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/tashirka1/k2/internal/metrics/model"
)

func collectSystemMetrics(now time.Time) []model.ResourcePoint {
	ts := now.UTC().Format(time.RFC3339)
	var points []model.ResourcePoint

	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		points = append(points, model.ResourcePoint{Timestamp: ts, Type: "cpu", Name: "percent", Value: cpuPercent[0]})
	}

	m, err := mem.VirtualMemory()
	if err == nil {
		points = append(points,
			model.ResourcePoint{Timestamp: ts, Type: "ram", Name: "percent", Value: m.UsedPercent},
			model.ResourcePoint{Timestamp: ts, Type: "ram", Name: "used", Value: float64(m.Used)},
			model.ResourcePoint{Timestamp: ts, Type: "ram", Name: "total", Value: float64(m.Total)},
			model.ResourcePoint{Timestamp: ts, Type: "ram", Name: "available", Value: float64(m.Available)},
		)
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		return points
	}
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		points = append(points,
			model.ResourcePoint{Timestamp: ts, Type: "disk", Name: "percent", Device: p.Mountpoint, Value: usage.UsedPercent},
			model.ResourcePoint{Timestamp: ts, Type: "disk", Name: "used", Device: p.Mountpoint, Value: float64(usage.Used)},
			model.ResourcePoint{Timestamp: ts, Type: "disk", Name: "total", Device: p.Mountpoint, Value: float64(usage.Total)},
			model.ResourcePoint{Timestamp: ts, Type: "disk", Name: "available", Device: p.Mountpoint, Value: float64(usage.Free)},
		)
	}

	return points
}
