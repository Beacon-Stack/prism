package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/beacon-media/prism/internal/core/library"
	"github.com/beacon-media/prism/internal/scheduler"
)

// LibraryScan returns a Job that scans all libraries, reconciling tracked
// movie files with what is actually on disk. Runs every 24 hours.
func LibraryScan(svc *library.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "library_scan",
		Interval: 24 * time.Hour,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "library_scan")
			start := time.Now()

			libs, err := svc.List(ctx)
			if err != nil {
				logger.Warn("task failed",
					"task", "library_scan",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				return
			}

			var scanErrs int
			for _, lib := range libs {
				if err := svc.Scan(ctx, lib.ID); err != nil {
					logger.Warn("library scan failed",
						"library_id", lib.ID,
						"library_name", lib.Name,
						"error", err,
					)
					scanErrs++
				}
			}

			if scanErrs > 0 {
				logger.Warn("task finished with errors",
					"task", "library_scan",
					"libraries_scanned", len(libs),
					"errors", scanErrs,
					"duration_ms", time.Since(start).Milliseconds(),
				)
			} else {
				logger.Info("task finished",
					"task", "library_scan",
					"libraries_scanned", len(libs),
					"duration_ms", time.Since(start).Milliseconds(),
				)
			}
		},
	}
}
