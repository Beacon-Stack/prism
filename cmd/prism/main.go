package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beacon-stack/prism/internal/anthropic"
	"github.com/beacon-stack/prism/internal/api"
	"github.com/beacon-stack/prism/internal/api/ws"
	"github.com/beacon-stack/prism/internal/config"
	"github.com/beacon-stack/prism/internal/core/activity"
	"github.com/beacon-stack/prism/internal/core/aicommand"
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
	"github.com/beacon-stack/prism/internal/core/notification"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/internal/core/queue"
	"github.com/beacon-stack/prism/internal/core/seedenforcer"
	"github.com/beacon-stack/prism/internal/core/stats"
	"github.com/beacon-stack/prism/internal/core/tag"
	"github.com/beacon-stack/prism/internal/core/watchsync"
	"github.com/beacon-stack/prism/internal/db"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/logging"
	"github.com/beacon-stack/prism/internal/mediaservers"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
	"github.com/beacon-stack/prism/internal/notifications"
	"github.com/beacon-stack/prism/internal/plexsync"
	"github.com/beacon-stack/prism/internal/pulse"
	"github.com/beacon-stack/prism/internal/radarrimport"
	"github.com/beacon-stack/prism/internal/ratelimit"
	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/scheduler"
	"github.com/beacon-stack/prism/internal/scheduler/jobs"
	"github.com/beacon-stack/prism/internal/trakt"
	"github.com/beacon-stack/prism/internal/version"

	// Blank-import built-in plugins so their init() functions register
	// them with the default registry before any service is constructed.
	_ "github.com/beacon-stack/prism/plugins/downloaders/deluge"
	_ "github.com/beacon-stack/prism/plugins/downloaders/haul"
	_ "github.com/beacon-stack/prism/plugins/downloaders/nzbget"
	_ "github.com/beacon-stack/prism/plugins/downloaders/qbittorrent"
	_ "github.com/beacon-stack/prism/plugins/downloaders/sabnzbd"
	_ "github.com/beacon-stack/prism/plugins/downloaders/transmission"
	_ "github.com/beacon-stack/prism/plugins/indexers/newznab"
	_ "github.com/beacon-stack/prism/plugins/indexers/torznab"
	_ "github.com/beacon-stack/prism/plugins/mediaservers/emby"
	_ "github.com/beacon-stack/prism/plugins/mediaservers/jellyfin"
	_ "github.com/beacon-stack/prism/plugins/mediaservers/plex"
	_ "github.com/beacon-stack/prism/plugins/notifications/command"
	_ "github.com/beacon-stack/prism/plugins/notifications/discord"
	_ "github.com/beacon-stack/prism/plugins/notifications/email"
	_ "github.com/beacon-stack/prism/plugins/notifications/gotify"
	_ "github.com/beacon-stack/prism/plugins/notifications/ntfy"
	_ "github.com/beacon-stack/prism/plugins/notifications/pushover"
	_ "github.com/beacon-stack/prism/plugins/notifications/slack"
	_ "github.com/beacon-stack/prism/plugins/notifications/telegram"
	_ "github.com/beacon-stack/prism/plugins/notifications/webhook"

	// Import list plugins
	_ "github.com/beacon-stack/prism/plugins/importlists/custom_list"
	_ "github.com/beacon-stack/prism/plugins/importlists/mdblist"
	_ "github.com/beacon-stack/prism/plugins/importlists/plex_watchlist"
	_ "github.com/beacon-stack/prism/plugins/importlists/stevenlu"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_collection"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_list"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_now_playing"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_person"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_popular"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_top_rated"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_trending"
	_ "github.com/beacon-stack/prism/plugins/importlists/tmdb_upcoming"
	_ "github.com/beacon-stack/prism/plugins/importlists/trakt_anticipated"
	_ "github.com/beacon-stack/prism/plugins/importlists/trakt_box_office"
	_ "github.com/beacon-stack/prism/plugins/importlists/trakt_list"
	_ "github.com/beacon-stack/prism/plugins/importlists/trakt_popular"
	_ "github.com/beacon-stack/prism/plugins/importlists/trakt_trending"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "", "path to config file (default: ~/.config/prism/config.yaml)")
	flag.Parse()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	logger, logBuffer := logging.New(cfg.Log.Level, cfg.Log.Format)

	// Set the global slog default so packages using the top-level slog
	// functions (slog.Info, slog.Error, etc.) pick up the configured handler.
	slog.SetDefault(logger)

	// Advisory config file permission check — use the resolved path so the
	// warning fires whether the file was specified explicitly or found at the
	// default location (~/.config/prism/config.yaml).
	checkPath := cfg.ConfigFile
	if checkPath == "" {
		checkPath = cfgFile // may also be "" if no file was found at all
	}
	if checkPath != "" {
		if info, statErr := os.Stat(checkPath); statErr == nil {
			if info.Mode()&0o044 != 0 {
				if chmodErr := os.Chmod(checkPath, 0o600); chmodErr != nil {
					logger.Warn("config file is group- or world-readable and chmod failed — please run: chmod 600 "+checkPath,
						"path", checkPath,
						"error", chmodErr,
					)
				} else {
					logger.Info("fixed config file permissions to 0600",
						"path", checkPath,
					)
				}
			}
		}
	}

	// ── API Key ───────────────────────────────────────────────────────────────
	generated, err := config.EnsureAPIKey(cfg)
	if err != nil {
		return fmt.Errorf("ensuring API key: %w", err)
	}
	if generated {
		// Try to persist the key so it survives restarts. This works when
		// the config directory is writable (local installs, Docker with a
		// volume at /config). If it fails we log a warning and continue —
		// the key still works for this session but will change on restart.
		if _, persistErr := config.WriteConfigKey(cfg.ConfigFile, "auth.api_key", cfg.Auth.APIKey.Value()); persistErr != nil {
			// Print the key to stderr directly — it must be visible to the operator
			// so they can configure clients, but we do NOT put it in structured logs
			// (which are often shipped to log aggregators and retained long-term).
			fmt.Fprintf(os.Stderr, "\n  !! API key generated but could not be saved to disk.\n"+
				"  !! It will change on next restart. Set it now in your client:\n"+
				"  !!\n"+
				"  !!   API key: %s\n"+
				"  !!\n"+
				"  !! Hint: mount a writable volume at /config (Docker) or ensure\n"+
				"  !!        ~/.config/prism/ is writable.\n\n",
				cfg.Auth.APIKey.Value())
			logger.Warn("API key generated but could not be persisted — it will change on next restart",
				"hint", "mount a writable volume at /config (Docker) or ensure ~/.config/prism/ is writable",
				"error", persistErr,
			)
		} else {
			logger.Info("API key generated and saved to config — stable across restarts")
		}
	} else {
		key := cfg.Auth.APIKey.Value()
		masked := key
		if len(key) > 4 {
			masked = key[:4] + "****"
		}
		logger.Info("API key loaded", "key_prefix", masked, "source", "config/env")
	}

	// ── Startup banner ────────────────────────────────────────────────────────
	configFile := cfg.ConfigFile
	if configFile == "" {
		configFile = "(none — using defaults/env)"
	}
	logger.Info("Prism starting",
		"version", version.Version,
		"build_time", version.BuildTime,
		"go", version.GoVersion(),
		"db", cfg.Database.Driver,
		"config_file", configFile,
	)

	// Log registered plugins so operators can confirm which plugins are active.
	for _, kind := range registry.Default.IndexerKinds() {
		logger.Info("registered indexer plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.DownloaderKinds() {
		logger.Info("registered downloader plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.NotifierKinds() {
		logger.Info("registered notifier plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.MediaServerKinds() {
		logger.Info("registered media server plugin", "plugin", kind)
	}
	for _, kind := range registry.Default.ImportListKinds() {
		logger.Info("registered import list plugin", "plugin", kind)
	}

	// ── Feature warnings ──────────────────────────────────────────────────────
	if cfg.TMDB.APIKey.IsEmpty() {
		logger.Warn("TMDB API key not configured — movie metadata and search features are disabled",
			"hint", "set tmdb.api_key in config.yaml or PRISM_TMDB_API_KEY env var",
		)
	}
	if cfg.AI.APIKey.IsEmpty() {
		logger.Info("Claude API key not configured — AI features disabled, using rule-based fallbacks")
	}

	// ── Database ──────────────────────────────────────────────────────────────
	database, err := db.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	logger.Info("database connected", "driver", database.Driver)

	if err := db.Migrate(database.SQL, database.Driver); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("database migrations up to date")

	// ── Event bus ─────────────────────────────────────────────────────────────
	bus := events.New(logger)

	// ── WebSocket hub ─────────────────────────────────────────────────────────
	wsHub := ws.NewHub(logger, []byte(cfg.Auth.APIKey.Value()))
	bus.Subscribe(wsHub.HandleEvent)

	// ── Services ──────────────────────────────────────────────────────────────
	queries := dbgen.New(database.SQL)

	qualitySvc := quality.NewService(queries, bus)
	qualityDefSvc := quality.NewDefinitionService(queries)

	var rawTMDB *tmdb.Client
	if !cfg.TMDB.APIKey.IsEmpty() {
		rawTMDB = tmdb.New(cfg.TMDB.APIKey.Value(), logger)
	}
	// tmdbClient is the interface used by movie and library services.
	// Declared separately to keep the nil-interface semantics correct.
	var tmdbClient movie.MetadataProvider
	if rawTMDB != nil {
		tmdbClient = rawTMDB
	}

	librarySvc := library.NewService(queries, bus, tmdbClient)
	movieSvc := movie.NewService(queries, tmdbClient, bus, logger, movie.WithDB(database.SQL))
	blocklistSvc := blocklist.NewService(queries)
	indexerRL := ratelimit.New()
	indexerSvc := indexer.NewService(queries, registry.Default, bus, indexerRL)
	downloaderSvc := downloader.NewService(queries, registry.Default, bus)
	queueSvc := queue.NewService(queries, downloaderSvc, bus, logger)

	tagSvc := tag.NewService(queries)
	cfSvc := customformat.NewService(queries)

	mmSvc := mediamanagement.NewService(queries)
	dhSvc := downloadhandling.NewService(queries)

	// ── MediaInfo scanner ──────────────────────────────────────────────────
	// Resolve scan_timeout; fall back to 30s if unparseable.
	scanTimeout := 30 * time.Second
	if cfg.MediaInfo.ScanTimeout != "" {
		if d, err := time.ParseDuration(cfg.MediaInfo.ScanTimeout); err == nil {
			scanTimeout = d
		}
	}
	mediainfoScanner := mediainfo.New(cfg.MediaInfo.FFprobePath, scanTimeout)
	if mediainfoScanner.Available() {
		logger.Info("mediainfo scanner available", "ffprobe", mediainfoScanner.FFprobePath())
	} else {
		logger.Info("mediainfo scanner unavailable — ffprobe not found; set mediainfo.ffprobe_path in config or install ffprobe")
	}
	mediainfoSvc := mediainfo.NewService(mediainfoScanner, queries, logger)

	activitySvc := activity.NewService(queries, logger)
	activitySvc.Subscribe(bus)

	importerSvc := importer.NewService(queries, bus, logger, mmSvc, dhSvc, mediainfoSvc, database.SQL)
	importerSvc.Subscribe()

	seedEnforcerSvc := seedenforcer.NewService(queries, indexerSvc, downloaderSvc, bus, logger)
	seedEnforcerSvc.Subscribe()

	notifSvc := notification.NewService(queries, registry.Default)
	notifDispatcher := notifications.NewDispatcher(queries, registry.Default, bus, logger, movieSvc)
	notifDispatcher.Subscribe()

	mediaServerSvc := mediaserver.NewService(queries, registry.Default)
	msDispatcher := mediaservers.NewDispatcher(queries, registry.Default, bus, logger)
	msDispatcher.Subscribe()

	plexSyncSvc := plexsync.NewService(mediaServerSvc, movieSvc, queries)
	watchSyncSvc := watchsync.NewService(queries, mediaServerSvc, movieSvc, registry.Default, logger)

	healthSvc := health.NewService(librarySvc, downloaderSvc, indexerSvc, logger)

	radarrImportSvc := radarrimport.NewService(movieSvc, qualitySvc, librarySvc, indexerSvc, downloaderSvc)
	statsSvc := stats.NewService(queries, movieSvc)

	var collectionSvc *collection.Service
	if rawTMDB != nil {
		collectionSvc = collection.NewService(queries, rawTMDB, movieSvc, logger)
	}

	// ── AutoSearch service (used by release routes and import lists) ──────
	var autoSvc *autosearch.Service
	if indexerSvc != nil && movieSvc != nil && downloaderSvc != nil && qualitySvc != nil {
		autoSvc = autosearch.NewService(
			indexerSvc, movieSvc, downloaderSvc,
			blocklistSvc, qualitySvc, cfSvc, tagSvc, bus, logger,
		)
	}

	// ── Trakt client (optional — only created if a client ID is configured) ──
	var traktClient *trakt.Client
	if !cfg.Trakt.ClientID.IsEmpty() {
		traktClient = trakt.New(cfg.Trakt.ClientID.Value(), logger)
	}

	// ── AI command service ──────────────────────────────────────────────────
	// Always created; the Anthropic client is set only when an API key is
	// configured (and can be hot-swapped via the settings UI).
	var aiClient *anthropic.Client
	if !cfg.AI.APIKey.IsEmpty() {
		aiClient = anthropic.New(cfg.AI.APIKey.Value())
	}
	aiCmdSvc := aicommand.NewService(aiClient, movieSvc, statsSvc, autoSvc, librarySvc, qualitySvc, logger)

	// ── Import list service ──────────────────────────────────────────────────
	importListSvc := importlist.NewService(queries, registry.Default, movieSvc, autoSvc, rawTMDB, traktClient, logger)

	// ── Scheduler ─────────────────────────────────────────────────────────────
	// Load queue poll interval from download handling settings. Default to 60s
	// on error so the scheduler always starts.
	queuePollInterval, err := dhSvc.CheckInterval(context.Background())
	if err != nil {
		logger.Warn("failed to load download handling interval, using 60s default", "error", err)
		queuePollInterval = 60 * time.Second
	}

	sched := scheduler.New(logger)
	sched.Add(jobs.QueuePoll(queueSvc, queuePollInterval, logger))
	sched.Add(jobs.LibraryScan(librarySvc, logger))
	sched.Add(jobs.RSSSync(indexerSvc, downloaderSvc, qualitySvc, queries, logger))
	sched.Add(jobs.RefreshMetadata(movieSvc, queries, logger))
	sched.Add(jobs.StatsSnapshot(statsSvc, logger))
	sched.Add(jobs.ImportListSync(importListSvc, logger))
	sched.Add(jobs.ActivityPrune(activitySvc, logger))
	sched.Add(jobs.WatchSync(watchSyncSvc, logger))

	// ── Pulse integration (optional) ────────────────────────────────────
	cfgrrIntegration, err := pulse.New(cfg.Pulse, cfg.Server.Host, cfg.Server.Port, logger)
	if err != nil {
		logger.Warn("pulse integration failed — continuing without it", "error", err)
	}
	if cfgrrIntegration != nil {
		defer cfgrrIntegration.Close()
	}

	// ── HTTP router ───────────────────────────────────────────────────────────
	startTime := time.Now()
	router := api.NewRouter(api.RouterConfig{
		Auth:                     cfg.Auth.APIKey,
		Logger:                   logger,
		StartTime:                startTime,
		DB:                       database.SQL,
		DBType:                   database.Driver,
		DBPath:                   cfg.Database.Path,
		DBDSN:                    cfg.Database.DSN.Value(),
		ConfigFile:               cfg.ConfigFile,
		TMDBKeyIsDefault:         cfg.TMDBKeyIsDefault,
		QualityService:           qualitySvc,
		QualityDefinitionService: qualityDefSvc,
		LibraryService:           librarySvc,
		MovieService:             movieSvc,
		TMDBClient:               rawTMDB,
		IndexerService:           indexerSvc,
		DownloaderService:        downloaderSvc,
		BlocklistService:         blocklistSvc,
		QueueService:             queueSvc,
		Scheduler:                sched,
		NotificationService:      notifSvc,
		HealthService:            healthSvc,
		MediaManagementService:   mmSvc,
		DownloadHandlingService:  dhSvc,
		RadarrImportService:      radarrImportSvc,
		StatsService:             statsSvc,
		MediaInfoService:         mediainfoSvc,
		CollectionService:        collectionSvc,
		MediaServerService:       mediaServerSvc,
		PlexSyncService:          plexSyncSvc,
		TagService:               tagSvc,
		CustomFormatService:      cfSvc,
		AutoSearchService:        autoSvc,
		AICommandService:         aiCmdSvc,
		ActivityService:          activitySvc,
		WatchSyncService:         watchSyncSvc,
		ImportListService:        importListSvc,
		LogBuffer:                logBuffer,
		WSHub:                    wsHub,
		Bus:                      bus,
		PulseSyncHandler:         pulseSyncHandler(cfgrrIntegration, indexerSvc, downloaderSvc),
	})

	// ── HTTP server ───────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ── Start background services ─────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pulse sync — pull assigned indexers, download clients, quality profiles, and shared settings.
	if cfgrrIntegration != nil {
		go cfgrrIntegration.StartSyncLoop(ctx, indexerSvc, downloaderSvc, qualitySvc, mmSvc, 30*time.Second)
	}

	go sched.Start(ctx)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig)
	}

	// Cancel scheduler and in-flight background jobs.
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped cleanly")
	return nil
}

// pulseSyncHandler returns the Pulse sync webhook handler,
// or nil if integration is disabled.
func pulseSyncHandler(integration *pulse.Integration, indexerSvc *indexer.Service, dlSvc *downloader.Service) http.HandlerFunc {
	if integration == nil {
		return nil
	}
	return integration.SyncHandler(indexerSvc, dlSvc)
}
