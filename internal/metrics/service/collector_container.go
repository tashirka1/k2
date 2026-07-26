package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tashirka1/k2/internal/metrics/model"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func collectContainerMetrics(ctx context.Context, now time.Time) []model.ContainerPoint {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}
	defer cli.Close()

	result, err := cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return nil
	}

	ts := now.Format(time.RFC3339)
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

		cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
		cpuPercent := 0.0
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * 100.0
		}

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
			})
		}
	}
	return points
}
