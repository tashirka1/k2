package service

import (
	"context"
	"encoding/json"
	"runtime"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func collectContainerMetrics(ctx context.Context, now time.Time, cli *client.Client) []model.ContainerPoint {
	if cli == nil {
		return nil
	}

	result, err := cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return nil
	}

	ts := now.UTC().Format(time.RFC3339)
	points := make([]model.ContainerPoint, 0, len(result.Items))

	for _, c := range result.Items {
		statsResult, err := cli.ContainerStats(ctx, c.ID, client.ContainerStatsOptions{})
		if err != nil {
			continue
		}
		var v container.StatsResponse
		if err := json.NewDecoder(statsResult.Body).Decode(&v); err != nil {
			statsResult.Body.Close()
			continue
		}
		statsResult.Body.Close()

		cpuPercent := containerCPUPercent(
			v.PreCPUStats.CPUUsage.TotalUsage,
			v.CPUStats.CPUUsage.TotalUsage,
			v.PreCPUStats.SystemUsage,
			v.CPUStats.SystemUsage,
			v.CPUStats.OnlineCPUs,
		)

		ramPercent := 0.0
		if v.MemoryStats.Limit > 0 {
			ramPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
		}

		if len(c.Names) > 0 {
			points = append(points, model.ContainerPoint{
				Timestamp: ts,
				Name:      c.Names[0][1:],
				Image:     c.Image,
				CPU:       cpuPercent,
				RAM:       ramPercent,
				RAMBytes:  int64(v.MemoryStats.Usage),
			})
		}
	}
	return points
}

func containerCPUPercent(prevTotal, total, prevSystem, system uint64, onlineCPUs uint32) float64 {
	if prevTotal == 0 || total < prevTotal || system <= prevSystem {
		return 0
	}
	if onlineCPUs == 0 {
		onlineCPUs = uint32(runtime.NumCPU())
	}
	cpuDelta := float64(total - prevTotal)
	systemDelta := float64(system - prevSystem)
	return cpuDelta / systemDelta * float64(onlineCPUs) * 100.0
}
