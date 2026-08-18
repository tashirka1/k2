package service

import (
	"context"
	"log/slog"
	"time"
)

func (s *Metrics) RunCollector(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("metrics collector started", "interval", interval)

	if err := s.collectTick(ctx); err != nil {
		slog.Error("initial collector tick failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("metrics collector stopped")
			return nil
		case <-ticker.C:
			if err := s.collectTick(ctx); err != nil {
				slog.Error("collector tick failed", "error", err)
			}
		}
	}
}

func (s *Metrics) collectTick(ctx context.Context) error {
	now := time.Now()

	resourcePoints := collectSystemMetrics(now)
	if len(resourcePoints) > 0 {
		if err := s.r.InsertResourceBatch(ctx, resourcePoints); err != nil {
			return err
		}
	}

	processPoints := s.collectProcessMetrics(now)
	if len(processPoints) > 0 {
		if err := s.r.InsertProcessBatch(ctx, processPoints); err != nil {
			return err
		}
	}

	containerPoints := collectContainerMetrics(ctx, now, s.dc)
	if len(containerPoints) > 0 {
		if err := s.r.InsertContainerBatch(ctx, containerPoints); err != nil {
			return err
		}
	}

	return nil
}
