// Package movie manages movie records in the Prism library.
package movie

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/dbutil"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
)

// Sentinel errors returned by Service methods.
var (
	ErrNotFound          = errors.New("movie not found")
	ErrAlreadyExists     = errors.New("movie already in library")
	ErrTMDBNotConfigured = errors.New("TMDB API key not configured")
	ErrFileNotFound      = errors.New("movie file not found")
)

// Movie is the domain representation of a movie record.
type Movie struct {
	ID                  string
	TMDBID              int
	IMDBID              string
	Title               string
	OriginalTitle       string
	Year                int
	Overview            string
	RuntimeMinutes      int
	Genres              []string
	PosterURL           string
	FanartURL           string
	Status              string
	Monitored           bool
	LibraryID           string
	QualityProfileID    string
	MinimumAvailability string
	PreferredEdition    string // empty = no preference
	ReleaseDate         string
	Path                string
	AddedAt             time.Time
	UpdatedAt           time.Time
	MetadataRefreshedAt *time.Time
}

// AddRequest carries the fields needed to add a movie to the library.
type AddRequest struct {
	TMDBID              int
	LibraryID           string
	QualityProfileID    string
	Monitored           bool
	MinimumAvailability string // defaults to "released" when empty
	PreferredEdition    string // empty = no preference
}

// AddUnmatchedRequest carries the fields needed to add a file that has no TMDB
// match. The resulting movie record has tmdb_id = 0, monitored = false, and
// uses the caller-supplied title (typically the raw filename stem).
type AddUnmatchedRequest struct {
	Title            string
	LibraryID        string
	QualityProfileID string
}

// ListRequest carries filter and pagination options for listing movies.
type ListRequest struct {
	LibraryID string // empty = all libraries
	Page      int    // 1-indexed; defaults to 1
	PerPage   int    // defaults to 50, max 250
}

// ListResult is the paginated response from List.
type ListResult struct {
	Movies  []Movie
	Total   int64
	Page    int
	PerPage int
}

// UpdateRequest carries the mutable fields for updating a movie.
type UpdateRequest struct {
	Title               string
	Monitored           bool
	LibraryID           string
	QualityProfileID    string
	MinimumAvailability string  // preserved from existing when empty
	PreferredEdition    *string // nil = don't change; pointer to "" clears it
}

// LookupRequest carries parameters for searching TMDB without adding to the library.
type LookupRequest struct {
	Query  string
	TMDBID int // if set, fetch exact movie; Query is ignored
	Year   int // optional year filter for query search
}

// Service manages movie records.
type Service struct {
	q          dbgen.Querier
	sqlDB      *sql.DB // for transactions; nil in tests that don't need them
	meta       MetadataProvider
	mu         sync.RWMutex
	bus        *events.Bus
	logger     *slog.Logger
	renameFile func(oldPath, newPath string) error // injectable for tests; defaults to os.Rename
}

// NewService creates a new Service backed by the given querier, metadata
// provider, event bus, and logger. meta may be nil when TMDB is not configured;
// methods that require it return ErrTMDBNotConfigured.
// sqlDB may be nil in tests; when non-nil, multi-step mutations run in a
// database transaction.
func NewService(q dbgen.Querier, meta MetadataProvider, bus *events.Bus, logger *slog.Logger, opts ...ServiceOption) *Service {
	s := &Service{q: q, meta: meta, bus: bus, logger: logger, renameFile: os.Rename}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ServiceOption configures optional Service dependencies.
type ServiceOption func(*Service)

// WithDB provides a *sql.DB for transaction support. When set, multi-step
// mutations (AttachFile, DeleteFile) run atomically.
func WithDB(sqlDB *sql.DB) ServiceOption {
	return func(s *Service) { s.sqlDB = sqlDB }
}

// Add fetches movie details from TMDB and inserts the movie into the library.
// When no metadata provider is configured (TMDB key absent), the movie is still
// added as a stub record with a placeholder title so that it can be monitored
// and grabbed immediately. Calling RefreshMetadata later will populate the full
// metadata once a key is configured.
// Returns ErrAlreadyExists if a movie with the same TMDB ID is already present.
func (s *Service) Add(ctx context.Context, req AddRequest) (Movie, error) {
	// Check for duplicates before hitting TMDB (or creating a stub).
	if _, err := s.q.GetMovieByTMDBID(ctx, int64(req.TMDBID)); err == nil {
		return Movie{}, ErrAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Movie{}, fmt.Errorf("checking for existing movie with tmdb_id %d: %w", req.TMDBID, err)
	}

	meta := s.provider()
	if meta == nil {
		return s.addStub(ctx, req)
	}

	detail, err := meta.GetMovie(ctx, req.TMDBID)
	if err != nil {
		return Movie{}, fmt.Errorf("fetching movie detail from TMDB: %w", err)
	}

	genresJSON, err := marshalGenres(detail.Genres)
	if err != nil {
		return Movie{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	imdbID := dbutil.NullStringFromString(detail.IMDBId)

	var runtimeMinutes *int64
	if detail.RuntimeMinutes > 0 {
		runtimeMinutes = ptrInt64(int64(detail.RuntimeMinutes))
	}

	posterURL := dbutil.NullStringFromString(detail.PosterPath)
	fanartURL := dbutil.NullStringFromString(detail.BackdropPath)

	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = "released"
	}

	row, err := s.q.CreateMovie(ctx, dbgen.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              int64(detail.ID),
		ImdbID:              imdbID,
		Title:               detail.Title,
		OriginalTitle:       detail.OriginalTitle,
		Year:                int64(detail.Year),
		Overview:            detail.Overview,
		RuntimeMinutes:      runtimeMinutes,
		GenresJson:          genresJSON,
		PosterUrl:           posterURL,
		FanartUrl:           fanartURL,
		Status:              detail.Status,
		Monitored:           req.Monitored,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: minAvail,
		ReleaseDate:         detail.ReleaseDate,
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: ptrString(now),
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting movie: %w", err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	if req.PreferredEdition != "" {
		if err := s.q.UpdateMoviePreferredEdition(ctx, dbgen.UpdateMoviePreferredEditionParams{
			ID:               m.ID,
			PreferredEdition: ptrString(req.PreferredEdition),
			UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return Movie{}, fmt.Errorf("setting preferred edition for movie %q: %w", m.ID, err)
		}
		m.PreferredEdition = req.PreferredEdition
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// addStub inserts a placeholder movie record when no TMDB metadata provider is
// configured. The title is set to "tmdb:<id>" so the record is clearly
// identifiable. Calling RefreshMetadata once a key is configured will replace
// all stub fields with real data.
func (s *Service) addStub(ctx context.Context, req AddRequest) (Movie, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = "released"
	}

	row, err := s.q.CreateMovie(ctx, dbgen.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              int64(req.TMDBID),
		ImdbID:              nil,
		Title:               fmt.Sprintf("tmdb:%d", req.TMDBID),
		OriginalTitle:       "",
		Year:                0,
		Overview:            "",
		RuntimeMinutes:      nil,
		GenresJson:          "[]",
		PosterUrl:           nil,
		FanartUrl:           nil,
		Status:              "unknown",
		Monitored:           req.Monitored,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: minAvail,
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: nil,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting stub movie for tmdb_id %d: %w", req.TMDBID, err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	s.logger.WarnContext(ctx, "TMDB not configured — movie added as stub; configure tmdb.api_key and call /refresh to populate metadata",
		slog.String("movie_id", m.ID),
		slog.Int("tmdb_id", req.TMDBID),
	)

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// AddUnmatched inserts a placeholder movie record for a file that has no TMDB
// match. The movie is created with tmdb_id = 0 and monitored = false so it
// appears in the library but is not actively sought for download. Calling
// Update or RefreshMetadata later can associate it with a real TMDB entry.
func (s *Service) AddUnmatched(ctx context.Context, req AddUnmatchedRequest) (Movie, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	row, err := s.q.CreateMovie(ctx, dbgen.CreateMovieParams{
		ID:                  uuid.New().String(),
		TmdbID:              0,
		ImdbID:              nil,
		Title:               req.Title,
		OriginalTitle:       "",
		Year:                0,
		Overview:            "",
		RuntimeMinutes:      nil,
		GenresJson:          "[]",
		PosterUrl:           nil,
		FanartUrl:           nil,
		Status:              "announced",
		Monitored:           false,
		LibraryID:           req.LibraryID,
		QualityProfileID:    req.QualityProfileID,
		MinimumAvailability: "released",
		ReleaseDate:         "",
		Path:                nil,
		AddedAt:             now,
		UpdatedAt:           now,
		MetadataRefreshedAt: nil,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("inserting unmatched movie: %w", err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieAdded,
		MovieID: m.ID,
		Data: map[string]any{
			"movie_id": m.ID,
			"title":    m.Title,
		},
	})

	return m, nil
}

// Get returns a single movie by its internal UUID.
// Returns ErrNotFound if no movie with that ID exists.
func (s *Service) Get(ctx context.Context, id string) (Movie, error) {
	row, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", id, err)
	}
	return rowToMovie(row)
}

// List returns a paginated set of movies, optionally filtered by library.
func (s *Service) List(ctx context.Context, req ListRequest) (ListResult, error) {
	// Normalise pagination inputs.
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 {
		req.PerPage = 50
	}
	if req.PerPage > 250 {
		req.PerPage = 250
	}

	limit := int64(req.PerPage)
	offset := int64((req.Page - 1) * req.PerPage)

	var (
		rows  []dbgen.Movie
		total int64
		err   error
	)

	if req.LibraryID != "" {
		total, err = s.q.CountMoviesByLibrary(ctx, req.LibraryID)
		if err != nil {
			return ListResult{}, fmt.Errorf("counting movies for library %q: %w", req.LibraryID, err)
		}

		rows, err = s.q.ListMoviesByLibrary(ctx, dbgen.ListMoviesByLibraryParams{
			LibraryID: req.LibraryID,
			Limit:     limit,
			Offset:    offset,
		})
	} else {
		total, err = s.q.CountMovies(ctx)
		if err != nil {
			return ListResult{}, fmt.Errorf("counting movies: %w", err)
		}

		rows, err = s.q.ListMovies(ctx, dbgen.ListMoviesParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return ListResult{}, fmt.Errorf("listing movies: %w", err)
	}

	movies := make([]Movie, 0, len(rows))
	for _, row := range rows {
		m, err := rowToMovie(row)
		if err != nil {
			return ListResult{}, err
		}
		movies = append(movies, m)
	}

	return ListResult{
		Movies:  movies,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

// Update replaces the mutable fields of an existing movie.
// Returns ErrNotFound if the movie does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Movie, error) {
	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q for update: %w", id, err)
	}

	libraryID := req.LibraryID
	if libraryID == "" {
		libraryID = existing.LibraryID
	}

	qualityProfileID := req.QualityProfileID
	if qualityProfileID == "" {
		qualityProfileID = existing.QualityProfileID
	}

	title := req.Title
	if title == "" {
		title = existing.Title
	}

	minAvail := req.MinimumAvailability
	if minAvail == "" {
		minAvail = existing.MinimumAvailability
	}

	now := time.Now().UTC().Format(time.RFC3339)

	row, err := s.q.UpdateMovie(ctx, dbgen.UpdateMovieParams{
		ID:                  id,
		Title:               title,
		OriginalTitle:       existing.OriginalTitle,
		Year:                existing.Year,
		Overview:            existing.Overview,
		RuntimeMinutes:      existing.RuntimeMinutes,
		GenresJson:          existing.GenresJson,
		PosterUrl:           existing.PosterUrl,
		FanartUrl:           existing.FanartUrl,
		Status:              existing.Status,
		Monitored:           req.Monitored,
		LibraryID:           libraryID,
		QualityProfileID:    qualityProfileID,
		MinimumAvailability: minAvail,
		ReleaseDate:         existing.ReleaseDate,
		UpdatedAt:           now,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("updating movie %q: %w", id, err)
	}

	m, err := rowToMovie(row)
	if err != nil {
		return Movie{}, err
	}

	if req.PreferredEdition != nil {
		var pe *string
		if *req.PreferredEdition != "" {
			pe = ptrString(*req.PreferredEdition)
		}
		if err := s.q.UpdateMoviePreferredEdition(ctx, dbgen.UpdateMoviePreferredEditionParams{
			ID:               id,
			PreferredEdition: pe,
			UpdatedAt:        now,
		}); err != nil {
			return Movie{}, fmt.Errorf("setting preferred edition for movie %q: %w", id, err)
		}
		m.PreferredEdition = *req.PreferredEdition
	}

	return m, nil
}

// Delete removes a movie by ID. It does not delete files from disk.
// Returns ErrNotFound if the movie does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching movie %q for delete: %w", id, err)
	}

	if err := s.q.DeleteMovie(ctx, id); err != nil {
		return fmt.Errorf("deleting movie %q: %w", id, err)
	}

	// Auto-exclude from import lists so the movie isn't re-added on next sync.
	if existing.TmdbID != 0 {
		_, _ = s.q.CreateImportExclusion(ctx, dbgen.CreateImportExclusionParams{
			ID:        uuid.New().String(),
			TmdbID:    existing.TmdbID,
			Title:     existing.Title,
			Year:      existing.Year,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeMovieDeleted,
		MovieID: id,
		Data: map[string]any{
			"movie_id": id,
			"title":    existing.Title,
		},
	})

	return nil
}

// GetByTMDBID returns a movie by its TMDB ID.
// Returns ErrNotFound if no movie with that TMDB ID exists.
func (s *Service) GetByTMDBID(ctx context.Context, tmdbID int) (Movie, error) {
	row, err := s.q.GetMovieByTMDBID(ctx, int64(tmdbID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie by tmdb_id %d: %w", tmdbID, err)
	}
	return rowToMovie(row)
}

// LibraryIDsByTMDBIDs maps TMDB IDs to library movie IDs for the subset that
// exist in the library, in a single query. Used by Discover to flag
// already-added results without an N+1 of GetByTMDBID per result. TMDB IDs not
// in the library are simply absent from the returned map.
func (s *Service) LibraryIDsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]string, error) {
	out := make(map[int]string, len(tmdbIDs))
	if len(tmdbIDs) == 0 {
		return out, nil
	}
	ids := make([]int64, len(tmdbIDs))
	for i, id := range tmdbIDs {
		ids[i] = int64(id)
	}
	rows, err := s.q.ListMovieIDsByTMDBIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("looking up movies by tmdb_ids: %w", err)
	}
	for _, r := range rows {
		out[int(r.TmdbID)] = r.ID
	}
	return out, nil
}

// Seed is a library movie Discover can base recommendations on.
type Seed struct {
	ID     string
	TMDBID int
	Title  string
}

// RecentlyAdded returns up to limit of the newest library movies that have
// a TMDB ID, newest first.
func (s *Service) RecentlyAdded(ctx context.Context, limit int) ([]Seed, error) {
	rows, err := s.q.ListRecentlyAddedMovies(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("listing recently added movies: %w", err)
	}
	out := make([]Seed, 0, len(rows))
	for _, r := range rows {
		out = append(out, Seed{ID: r.ID, TMDBID: int(r.TmdbID), Title: r.Title})
	}
	return out, nil
}

// GenreCounts returns how many library movies carry each genre name.
func (s *Service) GenreCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.q.ListMovieGenres(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing movie genres: %w", err)
	}
	counts := make(map[string]int)
	for _, raw := range rows {
		var genres []string
		if err := json.Unmarshal([]byte(raw), &genres); err != nil {
			continue // one unreadable row shouldn't hide the rest
		}
		for _, g := range genres {
			counts[g]++
		}
	}
	return counts, nil
}

// GetByFilePath returns the movie that owns the given file path, or
// ErrNotFound if no movie_files row exists for that path. If the
// movie_files row exists but the parent movie has been deleted
// (orphaned row), the orphan is cleaned up and ErrNotFound is returned.
func (s *Service) GetByFilePath(ctx context.Context, filePath string) (Movie, error) {
	mf, err := s.q.GetMovieFileByPath(ctx, filePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("looking up movie file %q: %w", filePath, err)
	}
	m, err := s.Get(ctx, mf.MovieID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Orphaned movie_files row — parent movie was deleted but
			// CASCADE didn't fire. Clean it up.
			_ = s.q.DeleteMovieFile(ctx, mf.ID)
		}
		return Movie{}, err
	}
	return m, nil
}

// marshalGenres encodes a genre slice as a JSON array string.
// A nil or empty slice becomes "[]".
func marshalGenres(genres []string) (string, error) {
	if genres == nil {
		genres = []string{}
	}
	b, err := json.Marshal(genres)
	if err != nil {
		return "", fmt.Errorf("marshaling genres: %w", err)
	}
	return string(b), nil
}

// rowToMovie converts a DB row into the domain Movie type.
func rowToMovie(row dbgen.Movie) (Movie, error) {
	var genres []string
	if err := json.Unmarshal([]byte(row.GenresJson), &genres); err != nil {
		return Movie{}, fmt.Errorf("unmarshaling genres for movie %q: %w", row.ID, err)
	}
	if genres == nil {
		genres = []string{}
	}

	addedAt, err := time.Parse(time.RFC3339, row.AddedAt)
	if err != nil {
		return Movie{}, fmt.Errorf("parsing added_at for movie %q: %w", row.ID, err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return Movie{}, fmt.Errorf("parsing updated_at for movie %q: %w", row.ID, err)
	}

	var metadataRefreshedAt *time.Time
	if row.MetadataRefreshedAt != nil {
		t, err := time.Parse(time.RFC3339, *row.MetadataRefreshedAt)
		if err != nil {
			return Movie{}, fmt.Errorf("parsing metadata_refreshed_at for movie %q: %w", row.ID, err)
		}
		metadataRefreshedAt = &t
	}

	imdbID := dbutil.NullStringValue(row.ImdbID)
	runtimeMinutes := dbutil.NullInt32Value(row.RuntimeMinutes)

	var posterURL string
	if row.PosterUrl != nil {
		posterURL = tmdbImageURL(*row.PosterUrl, "w500")
	}

	var fanartURL string
	if row.FanartUrl != nil {
		fanartURL = tmdbImageURL(*row.FanartUrl, "w1280")
	}

	path := dbutil.NullStringValue(row.Path)
	preferredEdition := dbutil.NullStringValue(row.PreferredEdition)

	return Movie{
		ID:                  row.ID,
		TMDBID:              int(row.TmdbID),
		IMDBID:              imdbID,
		Title:               row.Title,
		OriginalTitle:       row.OriginalTitle,
		Year:                int(row.Year),
		Overview:            row.Overview,
		RuntimeMinutes:      runtimeMinutes,
		Genres:              genres,
		PosterURL:           posterURL,
		FanartURL:           fanartURL,
		Status:              row.Status,
		Monitored:           row.Monitored,
		LibraryID:           row.LibraryID,
		QualityProfileID:    row.QualityProfileID,
		MinimumAvailability: row.MinimumAvailability,
		PreferredEdition:    preferredEdition,
		ReleaseDate:         row.ReleaseDate,
		Path:                path,
		AddedAt:             addedAt,
		UpdatedAt:           updatedAt,
		MetadataRefreshedAt: metadataRefreshedAt,
	}, nil
}

// ptrString returns a pointer to s.
func ptrString(s string) *string {
	return &s
}

// ptrInt64 returns a pointer to v.
func ptrInt64(v int64) *int64 {
	return &v
}
