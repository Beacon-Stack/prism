package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/conflict"
	"github.com/beacon-stack/prism/internal/core/indexer"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/internal/core/titlematch"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// interactiveSearchTimeout bounds a whole search a person is watching.
// Struggling indexers are cut at 6s and everyone else at 25s, so this is
// only a backstop.
const interactiveSearchTimeout = 60 * time.Second

// indexerOutcomeBody is how one indexer did, so the search window can say
// "still searching", "timed out, will be cached" or the error, instead of
// a bare "no releases found".
type indexerOutcomeBody struct {
	ID        string `json:"indexer_id"`
	Name      string `json:"indexer_name"`
	Results   int    `json:"results"`
	Error     string `json:"error,omitempty"`
	Pending   bool   `json:"pending,omitempty"`
	Searching bool   `json:"searching,omitempty"`
}

// releaseStreamBody is one line of the stream: everything found so far.
type releaseStreamBody struct {
	Releases []*releaseBody       `json:"releases"`
	Indexers []indexerOutcomeBody `json:"indexers"`
	Done     bool                 `json:"done"`
}

// releaseSearch holds what a movie search needs besides the indexers'
// answers, loaded once per search: the movie, its quality profile, the
// file it has, and what was grabbed before.
type releaseSearch struct {
	movie          movie.Movie
	query          plugin.SearchQuery
	prof           *quality.Profile
	currentQuality *plugin.Quality
	currentEdition string
	grabsByGUID    map[string]dbgen.GrabHistory
}

func newReleaseSearch(ctx context.Context, movieID string, indexerSvc *indexer.Service, movieSvc *movie.Service, qualitySvc *quality.Service) (*releaseSearch, error) {
	m, err := movieSvc.Get(ctx, movieID)
	if err != nil {
		if errors.Is(err, movie.ErrNotFound) {
			return nil, huma.Error404NotFound("movie not found")
		}
		return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
	}
	rs := &releaseSearch{
		movie: m,
		query: plugin.SearchQuery{TMDBID: m.TMDBID, IMDBID: m.IMDBID, Query: m.Title, Year: m.Year},
	}
	if qualitySvc != nil {
		if p, err := qualitySvc.Get(ctx, m.QualityProfileID); err == nil {
			rs.prof = &p
		}
	}
	if files, fErr := movieSvc.ListFiles(ctx, movieID); fErr == nil && len(files) > 0 {
		best := bestFileQuality(files)
		rs.currentQuality = &best
		rs.currentEdition = bestFileEdition(files)
	}
	// guid → most recent grab, to badge releases grabbed before.
	if grabRows, gErr := indexerSvc.GrabHistory(ctx, movieID); gErr == nil {
		rs.grabsByGUID = indexLatestGrabsByGUID(grabRows)
	}
	return rs, nil
}

// bodies turns search results into rows: drops releases for other movies
// (indexers sometimes return unrelated hits), then scores and badges.
func (rs *releaseSearch) bodies(results []indexer.SearchResult) []*releaseBody {
	out := make([]*releaseBody, 0, len(results))
	for _, r := range results {
		if !titlematch.Matches(r.Title, rs.movie.Title, rs.movie.Year) {
			continue
		}
		if rs.prof != nil {
			r.QualityScore, r.ScoreBreakdown = rs.prof.ScoreWithBreakdown(r.Quality)
		}
		b := indexerResultToBody(r)
		if rs.currentQuality != nil {
			b.Conflicts = conflict.Compare(*rs.currentQuality, r.Quality, rs.currentEdition, r.Edition)
		}
		if prior, ok := rs.grabsByGUID[r.GUID]; ok {
			b.AlreadyGrabbedAt = prior.GrabbedAt
			b.AlreadyGrabbedID = prior.ID
			b.AlreadyGrabbedStatus = prior.DownloadStatus
		}
		out = append(out, b)
	}
	return out
}

func (rs *releaseSearch) streamBody(snap indexer.Snapshot) releaseStreamBody {
	b := releaseStreamBody{Releases: rs.bodies(snap.Results), Done: snap.Done, Indexers: make([]indexerOutcomeBody, 0, len(snap.Outcomes))}
	for _, o := range snap.Outcomes {
		b.Indexers = append(b.Indexers, indexerOutcomeBody{ID: o.IndexerID, Name: o.IndexerName, Results: o.Results, Error: o.Error, Pending: o.Pending, Searching: o.Searching})
	}
	return b
}

// streamReleases runs the search and writes one JSON line each time an
// indexer answers, holding everything found so far; the last has done=true.
func streamReleases(hctx huma.Context, indexerSvc *indexer.Service, rs *releaseSearch) {
	ctx := hctx.Context()
	hctx.SetHeader("Content-Type", "application/x-ndjson")
	hctx.SetHeader("Cache-Control", "no-store")
	hctx.SetHeader("X-Accel-Buffering", "no") // don't let a proxy hold the lines back
	w := hctx.BodyWriter()
	enc := json.NewEncoder(w)
	send := func(snap indexer.Snapshot) error {
		if err := enc.Encode(rs.streamBody(snap)); err != nil {
			return err
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return nil
	}

	searchCtx, cancel := context.WithTimeout(ctx, interactiveSearchTimeout)
	defer cancel()

	var (
		mu      sync.Mutex
		latest  indexer.Snapshot
		updates = make(chan struct{}, 1)
		done    = make(chan indexer.Snapshot, 1)
	)
	go func() {
		snap, _ := indexerSvc.SearchInteractive(searchCtx, rs.query, func(s indexer.Snapshot) {
			mu.Lock()
			latest = s
			mu.Unlock()
			select {
			case updates <- struct{}{}:
			default: // one already waiting; it will carry this snapshot
			}
		})
		done <- snap
	}()

	for {
		select {
		case <-updates:
			mu.Lock()
			snap := latest
			mu.Unlock()
			if err := send(snap); err != nil {
				cancel() // the window closed; stop asking indexers
				<-done
				return
			}
		case snap := <-done:
			_ = send(snap)
			return
		}
	}
}

// registerReleaseStream adds GET /api/v1/movies/{id}/releases/stream.
func registerReleaseStream(api huma.API, indexerSvc *indexer.Service, movieSvc *movie.Service, qualitySvc *quality.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "stream-movie-releases",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/releases/stream",
		Summary:     "Search for releases, streaming results as each indexer answers (NDJSON)",
		Description: "Newline-delimited JSON: one {releases, indexers, done} object per line, each holding everything found so far. The last line has done=true.",
		Tags:        []string{"Releases"},
	}, func(ctx context.Context, input *releasesSearchInput) (*huma.StreamResponse, error) {
		rs, err := newReleaseSearch(ctx, input.MovieID, indexerSvc, movieSvc, qualitySvc)
		if err != nil {
			return nil, err
		}
		return &huma.StreamResponse{Body: func(hctx huma.Context) {
			streamReleases(hctx, indexerSvc, rs)
		}}, nil
	})
}
