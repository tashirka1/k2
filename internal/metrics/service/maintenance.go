package service

import (
	"context"
	"log/slog"
	"time"
)

func (s *Metrics) RunMaintenance(ctx context.Context) error {
	s.runTick(ctx)

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.runTick(ctx)
		}
	}
}

func (s *Metrics) runTick(ctx context.Context) {
	if err := s.maintenanceTick(ctx); err != nil {
		slog.Warn("maintenance tick failed", "error", err)
		return
	}
	slog.Info("maintenance tick completed")
}

func (s *Metrics) maintenanceTick(ctx context.Context) error {
	return s.r.PurgeOlderThan(ctx, s.retention)
}
