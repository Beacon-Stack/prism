package movie

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/beacon-stack/prism/internal/core/dbutil"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

// MetadataProvider fetches movie metadata from an external source.
type MetadataProvider interface {
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error)
	GetMovie(ctx context.Context, tmdbID int) (*tmdb.MovieDetail, error)
}

// SetMetadataProvider replaces the metadata provider at runtime without
// restarting the server. Safe for concurrent use.
func (s *Service) SetMetadataProvider(p MetadataProvider) {
	s.mu.Lock()
	s.meta = p
	s.mu.Unlock()
}

// HasMetadataProvider reports whether a metadata provider is currently wired up.
func (s *Service) HasMetadataProvider() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta != nil
}

// provider returns the current metadata provider under the read lock.
func (s *Service) provider() MetadataProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta
}

// MatchToTMDB associates an existing unmatched movie record with a TMDB entry.
// It updates the movie's tmdb_id and then refreshes all metadata fields from
// TMDB in a single step. Returns ErrAlreadyExists if another movie in the
// library already owns the supplied tmdbID. Returns ErrTMDBNotConfigured when
// no metadata provider is wired up.
func (s *Service) MatchToTMDB(ctx context.Context, movieID string, tmdbID int) (Movie, error) {
	meta := s.provider()
	if meta == nil {
		return Movie{}, ErrTMDBNotConfigured
	}

	// Ensure the target movie exists.
	if _, err := s.q.GetMovie(ctx, movieID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", movieID, err)
	}

	// Check that no other movie already owns this TMDB ID.
	dup, err := s.q.GetMovieByTMDBID(ctx, int64(tmdbID))
	if err == nil && dup.ID != movieID {
		return Movie{}, ErrAlreadyExists
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Movie{}, fmt.Errorf("checking for duplicate tmdb_id %d: %w", tmdbID, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.q.UpdateMovieTMDBID(ctx, dbgen.UpdateMovieTMDBIDParams{
		TmdbID:    int64(tmdbID),
		UpdatedAt: now,
		ID:        movieID,
	}); err != nil {
		return Movie{}, fmt.Errorf("updating tmdb_id for movie %q: %w", movieID, err)
	}

	// RefreshMetadata reads the updated tmdb_id from the DB and fetches full
	// metadata from TMDB.
	return s.RefreshMetadata(ctx, movieID)
}

// Lookup searches TMDB without adding anything to the library.
// If req.TMDBID is non-zero, it fetches that specific movie by ID.
// Otherwise it performs a text search using req.Query and optional req.Year.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) Lookup(ctx context.Context, req LookupRequest) ([]tmdb.SearchResult, error) {
	meta := s.provider()
	if meta == nil {
		return nil, ErrTMDBNotConfigured
	}

	if req.TMDBID != 0 {
		detail, err := meta.GetMovie(ctx, req.TMDBID)
		if err != nil {
			return nil, fmt.Errorf("fetching movie by tmdb_id %d: %w", req.TMDBID, err)
		}
		return []tmdb.SearchResult{
			{
				ID:            detail.ID,
				Title:         detail.Title,
				OriginalTitle: detail.OriginalTitle,
				Overview:      detail.Overview,
				ReleaseDate:   detail.ReleaseDate,
				Year:          detail.Year,
				PosterPath:    detail.PosterPath,
				BackdropPath:  detail.BackdropPath,
			},
		}, nil
	}

	results, err := meta.SearchMovies(ctx, req.Query, req.Year)
	if err != nil {
		return nil, fmt.Errorf("searching TMDB: %w", err)
	}
	return results, nil
}

// SuggestMatches parses the stored filename/title of an unmatched movie,
// searches TMDB with the extracted title and year, and returns ranked results.
// Returns ErrNotFound if the movie does not exist.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) SuggestMatches(ctx context.Context, id string) ([]tmdb.SearchResult, ParsedFilename, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, ParsedFilename{}, err
	}

	meta := s.provider()
	if meta == nil {
		return nil, ParsedFilename{}, ErrTMDBNotConfigured
	}

	// Use the actual video file path if one is attached (richer signal than the
	// stored title).  m.Path is the movie *directory*, not the filename, so we
	// query movie_files instead.  Fall back to m.Title (the raw filename stem
	// stored at import time) if no file record exists yet.
	source := m.Title
	if files, ferr := s.ListFiles(ctx, id); ferr == nil && len(files) > 0 {
		source = files[0].Path
	}
	parsed := ParseFilename(source)

	results, err := meta.SearchMovies(ctx, parsed.Title, parsed.Year)
	if err != nil {
		return nil, parsed, fmt.Errorf("searching TMDB: %w", err)
	}

	// If no results, retry by progressively dropping words from the front of
	// the title (up to 3 attempts).  This handles filenames where the parser
	// cannot split concatenated words — e.g. "The Hungergames Mockingjay Part
	// 1" yields nothing, but "Mockingjay Part 1" does.
	if len(results) == 0 {
		words := strings.Fields(parsed.Title)
		maxRetries := min(3, len(words)-1)
		for i := 1; i <= maxRetries && len(results) == 0; i++ {
			suffix := strings.Join(words[i:], " ")
			if len(strings.TrimSpace(suffix)) < 3 {
				break
			}
			results, _ = meta.SearchMovies(ctx, suffix, parsed.Year)
		}
	}

	return results, parsed, nil
}

// RefreshMetadata re-fetches TMDB data and updates the movie record in place.
// Returns ErrNotFound if the movie does not exist.
// Returns ErrTMDBNotConfigured if no metadata provider is wired up.
func (s *Service) RefreshMetadata(ctx context.Context, id string) (Movie, error) {
	meta := s.provider()
	if meta == nil {
		return Movie{}, ErrTMDBNotConfigured
	}

	existing, err := s.q.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Movie{}, ErrNotFound
		}
		return Movie{}, fmt.Errorf("fetching movie %q: %w", id, err)
	}

	detail, err := meta.GetMovie(ctx, int(existing.TmdbID))
	if err != nil {
		return Movie{}, fmt.Errorf("fetching updated TMDB detail for movie %q: %w", id, err)
	}

	genresJSON, err := marshalGenres(detail.Genres)
	if err != nil {
		return Movie{}, err
	}

	var runtimeMinutes *int64
	if detail.RuntimeMinutes > 0 {
		runtimeMinutes = ptrInt64(int64(detail.RuntimeMinutes))
	}

	posterURL := dbutil.NullStringFromString(detail.PosterPath)
	fanartURL := dbutil.NullStringFromString(detail.BackdropPath)

	now := time.Now().UTC().Format(time.RFC3339)

	row, err := s.q.UpdateMovie(ctx, dbgen.UpdateMovieParams{
		ID:                  id,
		Title:               detail.Title,
		OriginalTitle:       detail.OriginalTitle,
		Year:                int64(detail.Year),
		Overview:            detail.Overview,
		RuntimeMinutes:      runtimeMinutes,
		GenresJson:          genresJSON,
		PosterUrl:           posterURL,
		FanartUrl:           fanartURL,
		Status:              detail.Status,
		Monitored:           existing.Monitored,
		LibraryID:           existing.LibraryID,
		QualityProfileID:    existing.QualityProfileID,
		MinimumAvailability: existing.MinimumAvailability,
		ReleaseDate:         detail.ReleaseDate,
		UpdatedAt:           now,
	})
	if err != nil {
		return Movie{}, fmt.Errorf("updating movie %q after metadata refresh: %w", id, err)
	}

	// Record when the metadata was last refreshed.
	if err := s.q.UpdateMovieMetadataRefreshed(ctx, dbgen.UpdateMovieMetadataRefreshedParams{
		MetadataRefreshedAt: ptrString(now),
		UpdatedAt:           now,
		ID:                  id,
	}); err != nil {
		// Non-fatal: the data was updated; only the timestamp tracking failed.
		s.logger.WarnContext(ctx, "failed to update metadata_refreshed_at",
			slog.String("movie_id", id),
			slog.Any("error", err),
		)
	}

	// Re-fetch to get the final state including the refreshed timestamp.
	return s.Get(ctx, row.ID)
}

// tmdbImageURL converts a TMDB relative path (e.g. "/abc.jpg") to a full CDN
// URL using the given size string (e.g. "w500", "w1280"). Paths that already
// start with "http" are returned unchanged so the function is idempotent.
func tmdbImageURL(path, size string) string {
	if path == "" || strings.HasPrefix(path, "http") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "https://image.tmdb.org/t/p/" + size + path
}
