package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/scheduler"
)

// LibraryScan returns a Job that scans all libraries: it confirms tracked
// files, corrects the paths of ones that moved, links files on disk to
// movies that have none, and marks movies whose files are gone. Runs
// every 24 hours.
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
				res, err := svc.Scan(ctx, lib.ID)
				if err != nil {
					logger.Warn("library scan failed",
						"library_id", lib.ID,
						"library_name", lib.Name,
						"error", err,
					)
					scanErrs++
					continue
				}
				logger.Info("library scanned",
					"library_name", lib.Name,
					"present", res.Present,
					"relinked", res.Relinked,
					"attached", res.Attached,
					"missing", res.Missing,
				)
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
