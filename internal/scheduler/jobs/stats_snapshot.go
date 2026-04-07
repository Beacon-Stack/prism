package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/beacon-stack/prism/internal/core/stats"
	"github.com/beacon-stack/prism/internal/scheduler"
)

// StatsSnapshot returns a Job that records a storage snapshot once per day
// and prunes snapshots older than 90 days.
func StatsSnapshot(svc *stats.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "stats_snapshot",
		Interval: 24 * time.Hour,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "stats_snapshot")
			start := time.Now()

			if err := svc.TakeSnapshot(ctx); err != nil {
				logger.Warn("task failed",
					"task", "stats_snapshot",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				return
			}

			if err := svc.PruneSnapshots(ctx, 90*24*time.Hour); err != nil {
				logger.Warn("snapshot prune failed", "task", "stats_snapshot", "error", err)
			}

			logger.Info("task finished",
				"task", "stats_snapshot",
				"duration_ms", time.Since(start).Milliseconds(),
			)
		},
	}
}
