package v3

import (
	"database/sql"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/internal/core/queue"
	"github.com/beacon-stack/prism/internal/core/tag"
	"github.com/beacon-stack/prism/internal/scheduler"
)

// Config holds the dependencies for the Radarr v3 compatibility layer.
type Config struct {
	DB             *sql.DB
	MovieService   *movie.Service
	QualityService *quality.Service
	LibraryService *library.Service
	QueueService   *queue.Service
	TagService     *tag.Service
	Scheduler      *scheduler.Scheduler
}

// RegisterRoutes registers all Radarr v3 compatible endpoints on the given
// huma API instance.
func RegisterRoutes(api huma.API, cfg Config) {
	registerSystemRoutes(api)
	registerMovieRoutes(api, cfg.DB, cfg.MovieService, cfg.LibraryService)
	registerQualityProfileRoutes(api, cfg.DB, cfg.QualityService)
	registerRootFolderRoutes(api, cfg.DB, cfg.LibraryService)
	registerTagRoutes(api, cfg.TagService)
	registerQueueRoutes(api, cfg.QueueService)
	registerCommandRoutes(api, cfg.DB, cfg.MovieService, cfg.Scheduler)
}
