package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/beacon-stack/prism/internal/core/importlist"
	"github.com/beacon-stack/prism/internal/scheduler"
)

// ImportListSync returns a Job that syncs all enabled import lists every 6 hours.
func ImportListSync(svc *importlist.Service, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "import_list_sync",
		Interval: 6 * time.Hour,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "import_list_sync")
			start := time.Now()

			result := svc.Sync(ctx)

			logger.Info("task finished",
				"task", "import_list_sync",
				"lists_processed", result.ListsProcessed,
				"movies_added", result.MoviesAdded,
				"movies_skipped", result.MoviesSkipped,
				"errors", len(result.Errors),
				"duration_ms", time.Since(start).Milliseconds(),
			)

			for _, e := range result.Errors {
				logger.Warn("import list sync error",
					"task", "import_list_sync",
					"error", e,
				)
			}
		},
	}
}
