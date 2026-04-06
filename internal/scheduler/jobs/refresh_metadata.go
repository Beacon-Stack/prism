package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/beacon-media/prism/internal/core/movie"
	dbsqlite "github.com/beacon-media/prism/internal/db/generated/sqlite"
	"github.com/beacon-media/prism/internal/scheduler"
)

// RefreshMetadata returns a Job that re-fetches TMDB metadata for all
// monitored movies. Runs every 24 hours. Silently skips movies that are
// removed mid-run and exits early if TMDB is not configured.
func RefreshMetadata(movieSvc *movie.Service, q dbsqlite.Querier, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:     "refresh_metadata",
		Interval: 24 * time.Hour,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "refresh_metadata")
			start := time.Now()

			movies, err := q.ListMonitoredMovies(ctx)
			if err != nil {
				logger.Warn("task failed",
					"task", "refresh_metadata",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				return
			}

			var refreshed, failed int
			for _, m := range movies {
				if _, err := movieSvc.RefreshMetadata(ctx, m.ID); err != nil {
					if errors.Is(err, movie.ErrTMDBNotConfigured) {
						logger.Info("task skipped: TMDB not configured",
							"task", "refresh_metadata",
							"duration_ms", time.Since(start).Milliseconds(),
						)
						return
					}
					if errors.Is(err, movie.ErrNotFound) {
						continue // deleted mid-run
					}
					logger.Warn("metadata refresh failed",
						"movie_id", m.ID,
						"movie_title", m.Title,
						"error", err,
					)
					failed++
					continue
				}
				refreshed++
			}

			logger.Info("task finished",
				"task", "refresh_metadata",
				"refreshed", refreshed,
				"failed", failed,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		},
	}
}
