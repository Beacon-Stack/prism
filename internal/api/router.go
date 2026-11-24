package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/beacon-stack/pulse/pkg/auth"
	"github.com/beacon-stack/pulse/pkg/auth/authapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	beaconlog "github.com/beacon-stack/pulse/pkg/log"
	"github.com/beacon-stack/pulse/pkg/notify"
	"github.com/beacon-stack/pulse/pkg/notify/notifyapi"

	"github.com/beacon-stack/prism/internal/api/middleware"
	v1 "github.com/beacon-stack/prism/internal/api/v1"
	v3 "github.com/beacon-stack/prism/internal/api/v3"
	"github.com/beacon-stack/prism/internal/api/ws"
	"github.com/beacon-stack/prism/internal/config"
	"github.com/beacon-stack/prism/internal/core/activity"
	"github.com/beacon-stack/prism/internal/core/autosearch"
	"github.com/beacon-stack/prism/internal/core/blocklist"
	"github.com/beacon-stack/prism/internal/core/collection"
	"github.com/beacon-stack/prism/internal/core/customformat"
	"github.com/beacon-stack/prism/internal/core/downloader"
	"github.com/beacon-stack/prism/internal/core/downloadhandling"
	"github.com/beacon-stack/prism/internal/core/health"
	"github.com/beacon-stack/prism/internal/core/importer"
	"github.com/beacon-stack/prism/internal/core/importlist"
	"github.com/beacon-stack/prism/internal/core/indexer"
	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/core/mediainfo"
	"github.com/beacon-stack/prism/internal/core/mediamanagement"
	"github.com/beacon-stack/prism/internal/core/mediaserver"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/provider"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/internal/core/queue"
	"github.com/beacon-stack/prism/internal/core/stats"
	"github.com/beacon-stack/prism/internal/core/tag"
	"github.com/beacon-stack/prism/internal/core/watchsync"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
	"github.com/beacon-stack/prism/internal/plexsync"
	"github.com/beacon-stack/prism/internal/radarrimport"
	"github.com/beacon-stack/prism/internal/scheduler"
	"github.com/beacon-stack/prism/internal/version"
	"github.com/beacon-stack/prism/web"
)

// RouterConfig holds everything the router needs to function.
type RouterConfig struct {
	Auth config.Secret
	// Gate decides who may use the app. When nil, one is built from Auth
	// with default settings: no login, local network only.
	Gate                     *auth.Auth
	Logger                   *slog.Logger
	StartTime                time.Time
	DB                       *sql.DB
	DBType                   string
	DBPath                   string
	ConfigFile               string
	TMDBKeyIsDefault         bool
	QualityService           *quality.Service
	QualityDefinitionService *quality.DefinitionService
	LibraryService           *library.Service
	MovieService             *movie.Service
	TMDBClient               *tmdb.Client
	IndexerService           *indexer.Service
	DownloaderService        *downloader.Service
	BlocklistService         *blocklist.Service
	QueueService             *queue.Service
	Scheduler                *scheduler.Scheduler
	NotifyStore              *notify.Store
	NotifyOutbox             *notify.Outbox
	HealthService            *health.Service
	MediaManagementService   *mediamanagement.Service
	DownloadHandlingService  *downloadhandling.Service
	RadarrImportService      *radarrimport.Service
	StatsService             *stats.Service
	MediaInfoService         *mediainfo.Service
	CollectionService        *collection.Service
	MediaServerService       *mediaserver.Service
	PlexSyncService          *plexsync.Service
	TagService               *tag.Service
	CustomFormatService      *customformat.Service
	AutoSearchService        *autosearch.Service
	ActivityService          *activity.Service
	WatchSyncService         *watchsync.Service
	ImportListService        *importlist.Service
	ImporterService          *importer.Service
	ProviderResolver         *provider.Resolver
	LogSystem                *beaconlog.System
	DockerLogs               *beaconlog.DockerLogsReader
	WSHub                    *ws.Hub
	Bus                      *events.Bus
	PulseSyncHandler         http.HandlerFunc
}

// NewRouter builds and returns the application HTTP handler.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware — applied to every request including /health.
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxRequestBodySize(1 << 20)) // 1 MiB max request body
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(middleware.Recovery(cfg.Logger))

	// Who may use the app (pulse/pkg/auth): by default the local network
	// without a login. Login, logout and status are the only open /api
	// routes.
	gate := cfg.Gate
	if gate == nil {
		gate = auth.New(auth.Options{App: "Prism", APIKey: cfg.Auth.Value()}, auth.Settings{})
	}
	r.Get("/api/v1/auth/status", gate.StatusHandler())
	r.Post("/api/v1/auth/login", gate.LoginHandler())
	r.Post("/api/v1/auth/logout", gate.LogoutHandler())

	// Everything registered on protected goes through the gate. It is
	// attached per route, not by path prefix, so nothing routes around it.
	protected := r.With(gate.Middleware)

	// WebSocket event stream. On chi directly so the upgrade bypasses huma.
	if cfg.WSHub != nil {
		protected.Get("/api/v1/ws", cfg.WSHub.ServeHTTP)
	}

	// Pulse sync webhook, called by Pulse when indexer assignments change.
	if cfg.PulseSyncHandler != nil {
		protected.Post("/api/v1/hooks/pulse/sync", cfg.PulseSyncHandler)
	}

	// Backup / restore, on chi directly (binary body and response).
	if cfg.DB != nil && cfg.DBPath != "" {
		protected.Get("/api/v1/system/backup", v1.BackupHandler(cfg.DB, cfg.Logger))
		protected.Post("/api/v1/system/restore", v1.RestoreHandler(cfg.DBPath, cfg.Logger))
	}

	// Unauthenticated health check for load balancers / container probes.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	humaConfig := huma.DefaultConfig("Prism API", version.Version)
	humaConfig.DocsPath = "/api/docs"
	humaConfig.OpenAPIPath = "/api/openapi"
	humaConfig.SchemasPath = "/api/schemas"
	humaConfig.Info.Description = "Prism movie collection manager API. " +
		"By default requests from the local network need no credentials and others are refused. " +
		"When a login is required, browsers log in with POST /api/v1/auth/login; other clients provide X-Api-Key."

	humaAPI := humachi.New(protected, humaConfig)

	// Register X-Api-Key security scheme so the docs UI shows an Authorize
	// button and Try-it-out requests include the header automatically.
	oapi := humaAPI.OpenAPI()
	if oapi.Components == nil {
		oapi.Components = &huma.Components{}
	}
	if oapi.Components.SecuritySchemes == nil {
		oapi.Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	oapi.Components.SecuritySchemes["ApiKeyAuth"] = &huma.SecurityScheme{
		Type: "apiKey",
		In:   "header",
		Name: "X-Api-Key",
	}
	oapi.Security = []map[string][]string{{"ApiKeyAuth": {}}}

	authapi.Register(humaAPI, gate)
	v1.RegisterSystemRoutes(humaAPI, cfg.StartTime, cfg.DBType, cfg.DBPath, cfg.ConfigFile, cfg.TMDBKeyIsDefault, cfg.Auth.Value(), cfg.MovieService, cfg.Logger)
	v1.RegisterUpdateRoutes(humaAPI)

	if cfg.LogSystem != nil {
		beaconlog.RegisterRoutesWithDocker(humaAPI, cfg.LogSystem, cfg.DockerLogs)
	}

	if cfg.QualityService != nil {
		v1.RegisterQualityProfileRoutes(humaAPI, cfg.QualityService)
	}

	if cfg.QualityDefinitionService != nil {
		v1.RegisterQualityDefinitionRoutes(humaAPI, cfg.QualityDefinitionService)
	}

	if cfg.LibraryService != nil {
		v1.RegisterLibraryRoutes(humaAPI, cfg.LibraryService, cfg.MovieService)
	}

	if cfg.MovieService != nil {
		v1.RegisterMovieRoutes(humaAPI, cfg.MovieService, cfg.TagService, cfg.WatchSyncService)
		v1.RegisterMovieFileRoutes(humaAPI, cfg.MovieService, cfg.MediaManagementService, cfg.MediaInfoService)
		v1.RegisterMovieCreditsRoutes(humaAPI, cfg.MovieService, cfg.TMDBClient)
		v1.RegisterDiscoverRoutes(humaAPI, cfg.MovieService, cfg.TMDBClient)
		v1.RegisterPeopleRoutes(humaAPI, cfg.MovieService, cfg.TMDBClient)
		v1.RegisterWantedRoutes(humaAPI, cfg.MovieService)
	}

	v1.RegisterMediainfoRoutes(humaAPI, cfg.MediaInfoService)

	if cfg.IndexerService != nil {
		v1.RegisterIndexerRoutes(humaAPI, cfg.IndexerService, cfg.TagService)

		autoSvc := cfg.AutoSearchService
		if autoSvc == nil && cfg.MovieService != nil && cfg.DownloaderService != nil && cfg.QualityService != nil {
			autoSvc = autosearch.NewService(
				cfg.IndexerService, cfg.MovieService, cfg.DownloaderService,
				cfg.BlocklistService, cfg.QualityService, cfg.CustomFormatService, cfg.TagService, cfg.Bus, cfg.Logger,
			)
		}
		v1.RegisterReleaseRoutes(humaAPI, cfg.IndexerService, cfg.MovieService, cfg.DownloaderService, cfg.BlocklistService, cfg.QualityService, autoSvc, cfg.Logger)

		v1.RegisterHistoryRoutes(humaAPI, cfg.IndexerService)
	}

	if cfg.BlocklistService != nil {
		v1.RegisterBlocklistRoutes(humaAPI, cfg.BlocklistService)
	}

	if cfg.DownloaderService != nil {
		v1.RegisterDownloadClientRoutes(humaAPI, cfg.DownloaderService, cfg.TagService)
	}

	// Haul history endpoints — needs both downloader (to find a Haul
	// client) and importer (to run the import pipeline).
	if cfg.DownloaderService != nil && cfg.ImporterService != nil {
		v1.RegisterHaulHistoryRoutes(humaAPI, cfg.DownloaderService, cfg.ImporterService)
	}

	if cfg.QueueService != nil {
		v1.RegisterQueueRoutes(humaAPI, cfg.QueueService, cfg.BlocklistService)
	}

	if cfg.Scheduler != nil {
		v1.RegisterTaskRoutes(humaAPI, cfg.Scheduler)
	}

	if cfg.NotifyStore != nil && cfg.NotifyOutbox != nil {
		notifyapi.Register(humaAPI, notifyapi.Options{App: "Prism", Store: cfg.NotifyStore, Outbox: cfg.NotifyOutbox})
	}

	if cfg.MediaServerService != nil {
		v1.RegisterMediaServerRoutes(humaAPI, cfg.MediaServerService)
	}

	if cfg.PlexSyncService != nil {
		v1.RegisterPlexSyncRoutes(humaAPI, cfg.PlexSyncService)
	}

	if cfg.HealthService != nil {
		v1.RegisterHealthRoutes(humaAPI, cfg.HealthService)
	}

	if cfg.MediaManagementService != nil {
		v1.RegisterMediaManagementRoutes(humaAPI, cfg.MediaManagementService)
	}

	if cfg.DownloadHandlingService != nil {
		v1.RegisterDownloadHandlingRoutes(humaAPI, cfg.DownloadHandlingService)
	}

	if cfg.RadarrImportService != nil {
		v1.RegisterImportRoutes(humaAPI, cfg.RadarrImportService)
	}

	if cfg.StatsService != nil {
		v1.RegisterStatsRoutes(humaAPI, cfg.StatsService)
	}

	v1.RegisterCollectionRoutes(humaAPI, cfg.CollectionService)

	if cfg.TagService != nil {
		v1.RegisterTagRoutes(humaAPI, cfg.TagService)
	}

	if cfg.CustomFormatService != nil {
		v1.RegisterCustomFormatRoutes(humaAPI, cfg.CustomFormatService)
	}

	if cfg.ImportListService != nil {
		v1.RegisterImportListRoutes(humaAPI, cfg.ImportListService, cfg.TagService)
	}

	v1.RegisterPlexAuthRoutes(humaAPI)

	if cfg.LibraryService != nil && cfg.MovieService != nil && cfg.Bus != nil && cfg.Scheduler != nil {
		v1.RegisterHookRoutes(humaAPI, cfg.LibraryService, cfg.MovieService, cfg.Bus, cfg.Scheduler)
	}

	v1.RegisterFilesystemRoutes(humaAPI)
	v1.RegisterParseRoutes(humaAPI)

	if cfg.ActivityService != nil {
		v1.RegisterActivityRoutes(humaAPI, cfg.ActivityService)
	}

	if cfg.WatchSyncService != nil {
		v1.RegisterWatchSyncRoutes(humaAPI, cfg.WatchSyncService)
	}

	if cfg.ProviderResolver != nil {
		v1.RegisterProviderRoutes(humaAPI, cfg.ProviderResolver)
	}

	// Radarr v3 API compatibility layer
	// External tools (Overseerr, Homepage, etc.) can point their "Radarr"
	// integration at Prism and it will just work.
	if cfg.DB != nil {
		v3Config := huma.DefaultConfig("Prism Radarr-Compatible API", version.Version)
		v3Config.DocsPath = ""
		v3Config.OpenAPIPath = ""
		v3Config.SchemasPath = ""
		v3API := humachi.New(protected, v3Config)
		v3.RegisterRoutes(v3API, v3.Config{
			DB:             cfg.DB,
			MovieService:   cfg.MovieService,
			QualityService: cfg.QualityService,
			LibraryService: cfg.LibraryService,
			QueueService:   cfg.QueueService,
			TagService:     cfg.TagService,
			Scheduler:      cfg.Scheduler,
		})
	}

	// Serve the embedded React SPA. This handler serves static files when they
	// exist (assets, favicon, etc.) and falls back to index.html for all other
	// paths so React Router can handle client-side navigation. Must come after
	// all API routes so /api/* and /health take precedence.
	r.Handle("/*", web.ServeStatic())

	return r
}
