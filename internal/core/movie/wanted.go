package movie

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// ListMissing returns paginated monitored movies that have no associated file.
func (s *Service) ListMissing(ctx context.Context, page, perPage int) ([]Movie, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	total, err := s.q.CountMonitoredMoviesWithoutFile(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("counting missing movies: %w", err)
	}
	rows, err := s.q.ListMonitoredMoviesWithoutFile(ctx, dbgen.ListMonitoredMoviesWithoutFileParams{
		Limit:  int64(perPage),
		Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listing missing movies: %w", err)
	}
	movies := make([]Movie, 0, len(rows))
	for _, r := range rows {
		m, err := rowToMovie(r)
		if err != nil {
			s.logger.Warn("skipping movie with bad data", "id", r.ID, "err", err)
			continue
		}
		movies = append(movies, m)
	}
	return movies, total, nil
}

// ListCutoffUnmet returns all monitored movies whose best file quality does not
// meet the quality profile cutoff. Filtering is done in Go to avoid complex SQL
// over JSON-encoded quality columns.
func (s *Service) ListCutoffUnmet(ctx context.Context) ([]Movie, error) {
	rows, err := s.q.ListMonitoredMoviesWithFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing movies with files: %w", err)
	}

	type entry struct {
		row    dbgen.ListMonitoredMoviesWithFilesRow
		best   plugin.Quality
		cutoff plugin.Quality
	}
	seen := map[string]*entry{}

	for _, r := range rows {
		var fileQ plugin.Quality
		_ = json.Unmarshal([]byte(r.QualityJson), &fileQ)

		if e, ok := seen[r.ID]; !ok {
			var cutoffQ plugin.Quality
			_ = json.Unmarshal([]byte(r.CutoffJson), &cutoffQ)
			seen[r.ID] = &entry{row: r, best: fileQ, cutoff: cutoffQ}
		} else if fileQ.Score() > e.best.Score() {
			e.best = fileQ
		}
	}

	var result []Movie
	for _, e := range seen {
		if !e.best.AtLeast(e.cutoff) {
			m, err := rowToMovieFromWithFilesRow(e.row)
			if err != nil {
				s.logger.Warn("skipping movie with bad data", "id", e.row.ID, "err", err)
				continue
			}
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result, nil
}

// CountCutoffUnmet returns the number of monitored movies whose best file
// quality does not meet the quality profile cutoff. Unlike ListCutoffUnmet it
// skips the Movie conversion and sorting since only the count is needed.
func (s *Service) CountCutoffUnmet(ctx context.Context) (int64, error) {
	rows, err := s.q.ListMonitoredMoviesWithFiles(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing movies with files: %w", err)
	}

	type entry struct {
		best   plugin.Quality
		cutoff plugin.Quality
	}
	seen := map[string]*entry{}

	for _, r := range rows {
		var fileQ plugin.Quality
		_ = json.Unmarshal([]byte(r.QualityJson), &fileQ)

		if e, ok := seen[r.ID]; !ok {
			var cutoffQ plugin.Quality
			_ = json.Unmarshal([]byte(r.CutoffJson), &cutoffQ)
			seen[r.ID] = &entry{best: fileQ, cutoff: cutoffQ}
		} else if fileQ.Score() > e.best.Score() {
			e.best = fileQ
		}
	}

	var count int64
	for _, e := range seen {
		if !e.best.AtLeast(e.cutoff) {
			count++
		}
	}
	return count, nil
}

// rowToMovieFromWithFilesRow converts a ListMonitoredMoviesWithFilesRow (which
// includes extra quality columns) into the domain Movie type.
func rowToMovieFromWithFilesRow(r dbgen.ListMonitoredMoviesWithFilesRow) (Movie, error) {
	// Delegate to rowToMovie by re-mapping shared fields.
	return rowToMovie(dbgen.Movie{
		ID:                  r.ID,
		TmdbID:              r.TmdbID,
		ImdbID:              r.ImdbID,
		Title:               r.Title,
		OriginalTitle:       r.OriginalTitle,
		Year:                r.Year,
		Overview:            r.Overview,
		RuntimeMinutes:      r.RuntimeMinutes,
		GenresJson:          r.GenresJson,
		PosterUrl:           r.PosterUrl,
		FanartUrl:           r.FanartUrl,
		Status:              r.Status,
		Monitored:           r.Monitored,
		LibraryID:           r.LibraryID,
		QualityProfileID:    r.QualityProfileID,
		Path:                r.Path,
		AddedAt:             r.AddedAt,
		UpdatedAt:           r.UpdatedAt,
		MetadataRefreshedAt: r.MetadataRefreshedAt,
		MinimumAvailability: r.MinimumAvailability,
	})
}
