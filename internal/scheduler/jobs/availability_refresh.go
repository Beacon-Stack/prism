package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/beacon-stack/prism/internal/availability"
	"github.com/beacon-stack/prism/internal/scheduler"
)

// AvailabilityRefresh returns a Job that re-reads what the media servers
// hold, so a movie shows as available soon after the server picks it up.
// Runs every 2 minutes; it is one request per movie library.
func AvailabilityRefresh(svc *availability.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "availability_refresh",
		Interval: 2 * time.Minute,
		Fn: func(ctx context.Context) {
			if err := svc.Refresh(ctx); err != nil {
				logger.Warn("task failed", "task", "availability_refresh", "error", err)
			}
		},
	}
}
