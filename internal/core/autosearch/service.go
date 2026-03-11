// Package autosearch implements on-demand automatic search: given a movie,
// search all indexers, pick the best release that satisfies the quality profile,
// and submit it to a download client. Used by both the single-movie search
// button and the bulk "Search All" action on the Wanted page.
package autosearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// Result status constants.
const (
	StatusGrabbed = "grabbed"
	StatusNoMatch = "no_match"
	StatusSkipped = "skipped"
)

// Result describes the outcome of an auto-search for a single movie.
type Result struct {
	MovieID string    `json:"movie_id"`
	Status  string    `json:"result"` // "grabbed", "no_match", "skipped"
	Reason  string    `json:"reason,omitempty"`
	Grab    *GrabInfo `json:"grab,omitempty"`
}

// GrabInfo is the subset of grab history returned to the caller.
type GrabInfo struct {
	ID           string          `json:"id"`
	MovieID      string          `json:"movie_id"`
	ReleaseTitle string          `json:"release_title"`
	Protocol     string          `json:"protocol"`
	Size         int64           `json:"size"`
	GrabbedAt    string          `json:"grabbed_at"`
	Breakdown    json.RawMessage `json:"score_breakdown,omitempty"`
}

// BulkResult summarises a bulk auto-search operation.
type BulkResult struct {
	Searched int       `json:"searched"`
	Grabbed  int       `json:"grabbed"`
	Results  []*Result `json:"results"`
}

// MaxBulkMovies is the maximum number of movie IDs accepted in a single bulk
// search request.
const MaxBulkMovies = 100

// bulkConcurrency is the maximum number of movies searched concurrently during
// a bulk operation.
const bulkConcurrency = 2

// bulkStagger is the delay between starting successive movie searches in a
// bulk operation to avoid hammering indexers.
const bulkStagger = 1 * time.Second

// Service orchestrates automatic search and grab for movies.
type Service struct {
	indexerSvc    *indexer.Service
	movieSvc      *movie.Service
	downloaderSvc *downloader.Service
	blocklistSvc  *blocklist.Service
	qualitySvc    *quality.Service
	bus           *events.Bus
	logger        *slog.Logger
}

// NewService creates a new auto-search Service.
func NewService(
	indexerSvc *indexer.Service,
	movieSvc *movie.Service,
	downloaderSvc *downloader.Service,
	blocklistSvc *blocklist.Service,
	qualitySvc *quality.Service,
	bus *events.Bus,
	logger *slog.Logger,
) *Service {
	return &Service{
		indexerSvc:    indexerSvc,
		movieSvc:      movieSvc,
		downloaderSvc: downloaderSvc,
		blocklistSvc:  blocklistSvc,
		qualitySvc:    qualitySvc,
		bus:           bus,
		logger:        logger,
	}
}

// SearchMovie performs a full indexer search for a single movie, picks the best
// release satisfying the quality profile, and submits it to a download client.
// Works on both monitored and unmonitored movies (explicit user action).
func (s *Service) SearchMovie(ctx context.Context, movieID string) (*Result, error) {
	// 1. Fetch the movie.
	mov, err := s.movieSvc.Get(ctx, movieID)
	if err != nil {
		if errors.Is(err, movie.ErrNotFound) {
			return nil, movie.ErrNotFound
		}
		return nil, fmt.Errorf("fetching movie: %w", err)
	}

	// 2. Full indexer search.
	query := plugin.SearchQuery{
		TMDBID: mov.TMDBID,
		IMDBID: mov.IMDBID,
		Query:  mov.Title,
		Year:   mov.Year,
	}
	results, searchErr := s.indexerSvc.Search(ctx, query)
	if len(results) == 0 {
		if searchErr != nil {
			return nil, fmt.Errorf("all indexers failed: %w", searchErr)
		}
		return &Result{
			MovieID: movieID,
			Status:  StatusNoMatch,
			Reason:  "no releases found from any indexer",
		}, nil
	}

	// 3. Load quality profile.
	prof, err := s.qualitySvc.Get(ctx, mov.QualityProfileID)
	if err != nil {
		return nil, fmt.Errorf("loading quality profile: %w", err)
	}

	// 4. Determine current file quality on disk (nil = no file).
	var currentQuality *plugin.Quality
	if files, fErr := s.movieSvc.ListFiles(ctx, movieID); fErr == nil && len(files) > 0 {
		best := bestFileQuality(files)
		currentQuality = &best
	}

	// 5. Iterate candidates (sorted best→worst), try each.
	for _, r := range results {
		// Skip blocklisted releases.
		if s.blocklistSvc != nil {
			blocked, blErr := s.blocklistSvc.IsBlocklisted(ctx, r.GUID)
			if blErr != nil {
				s.logger.Warn("auto-search: blocklist check failed", "guid", r.GUID, "error", blErr)
			} else if blocked {
				continue
			}
		}

		// Skip releases the quality profile doesn't want.
		if !prof.WantRelease(r.Quality, currentQuality) {
			continue
		}

		// Try submitting to a download client.
		if s.downloaderSvc == nil {
			return nil, fmt.Errorf("no download client service configured")
		}
		dcID, itemID, addErr := s.downloaderSvc.Add(ctx, r.Release)
		if addErr != nil {
			if errors.Is(addErr, downloader.ErrNoCompatibleClient) {
				return nil, fmt.Errorf("no download client configured for protocol %s", r.Protocol)
			}
			// Download client rejected this release — auto-blocklist and try next.
			s.logger.Warn("auto-search: download client rejected release, trying next",
				"movie_id", movieID,
				"release", r.Title,
				"error", addErr,
			)
			if s.blocklistSvc != nil {
				blErr := s.blocklistSvc.Add(ctx, movieID, r.GUID, r.Title,
					r.IndexerID, string(r.Protocol), r.Size, "auto-search: download client rejected")
				if blErr != nil && !errors.Is(blErr, blocklist.ErrAlreadyBlocklisted) {
					s.logger.Warn("auto-search: failed to auto-blocklist",
						"guid", r.GUID, "error", blErr)
				}
			}
			continue
		}

		// Compute score breakdown for history.
		_, breakdown := prof.ScoreWithBreakdown(r.Quality)
		breakdownJSON, _ := json.Marshal(breakdown)

		// Record grab in history. The unique partial index on grab_history
		// prevents duplicate active grabs for the same movie.
		history, grabErr := s.indexerSvc.Grab(ctx, movieID, r.IndexerID, r.Release, dcID, itemID, string(breakdownJSON))
		if grabErr != nil {
			// If this is a unique constraint violation, another grab is active.
			if isUniqueViolation(grabErr) {
				return &Result{
					MovieID: movieID,
					Status:  StatusSkipped,
					Reason:  "already downloading",
				}, nil
			}
			return nil, fmt.Errorf("recording grab history: %w", grabErr)
		}

		s.logger.Info("auto-search: grabbed release",
			"movie_id", movieID,
			"movie_title", mov.Title,
			"release", r.Title,
			"quality_score", r.QualityScore,
		)

		var bd json.RawMessage
		if len(breakdownJSON) > 0 {
			bd = breakdownJSON
		}

		return &Result{
			MovieID: movieID,
			Status:  StatusGrabbed,
			Grab: &GrabInfo{
				ID:           history.ID,
				MovieID:      history.MovieID,
				ReleaseTitle: history.ReleaseTitle,
				Protocol:     history.Protocol,
				Size:         history.Size,
				GrabbedAt:    history.GrabbedAt,
				Breakdown:    bd,
			},
		}, nil
	}

	// All candidates exhausted.
	return &Result{
		MovieID: movieID,
		Status:  StatusNoMatch,
		Reason:  "no releases satisfy quality profile",
	}, nil
}

// SearchMovies runs SearchMovie for each movie ID with bounded concurrency.
// Progress events are published to the event bus for WebSocket delivery.
func (s *Service) SearchMovies(ctx context.Context, movieIDs []string) *BulkResult {
	bulk := &BulkResult{
		Searched: len(movieIDs),
		Results:  make([]*Result, len(movieIDs)),
	}

	var mu sync.Mutex
	sem := make(chan struct{}, bulkConcurrency)

	var wg sync.WaitGroup
	for i, id := range movieIDs {
		wg.Add(1)
		go func(idx int, movieID string) {
			defer wg.Done()

			// Stagger starts to avoid hammering indexers.
			if idx > 0 {
				select {
				case <-ctx.Done():
					mu.Lock()
					bulk.Results[idx] = &Result{
						MovieID: movieID,
						Status:  StatusSkipped,
						Reason:  "cancelled",
					}
					mu.Unlock()
					return
				case <-time.After(time.Duration(idx) * bulkStagger):
				}
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := s.SearchMovie(ctx, movieID)
			if err != nil {
				result = &Result{
					MovieID: movieID,
					Status:  StatusNoMatch,
					Reason:  err.Error(),
				}
			}

			mu.Lock()
			bulk.Results[idx] = result
			if result.Status == StatusGrabbed {
				bulk.Grabbed++
			}
			mu.Unlock()

			// Publish progress event for WebSocket clients.
			if s.bus != nil {
				s.bus.Publish(ctx, events.Event{
					Type:    events.TypeBulkSearchProgress,
					MovieID: movieID,
					Data: map[string]any{
						"result":  result.Status,
						"reason":  result.Reason,
						"current": idx + 1,
						"total":   len(movieIDs),
					},
				})
			}
		}(i, id)
	}

	wg.Wait()

	// Publish completion event.
	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type: events.TypeBulkSearchComplete,
			Data: map[string]any{
				"searched": bulk.Searched,
				"grabbed":  bulk.Grabbed,
			},
		})
	}

	return bulk
}

// bestFileQuality returns the highest-scoring quality among the given files.
func bestFileQuality(files []movie.FileInfo) plugin.Quality {
	var best plugin.Quality
	for _, f := range files {
		if f.Quality.BetterThan(best) {
			best = f.Quality
		}
	}
	return best
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
