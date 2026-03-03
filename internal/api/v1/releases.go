package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/downloader"
	"github.com/davidfic/luminarr/internal/core/indexer"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ── Request / response shapes ────────────────────────────────────────────────

type releaseBody struct {
	GUID         string         `json:"guid"`
	Title        string         `json:"title"`
	Indexer      string         `json:"indexer"`
	Protocol     string         `json:"protocol"`
	DownloadURL  string         `json:"download_url"`
	InfoURL      string         `json:"info_url,omitempty"`
	Size         int64          `json:"size"`
	Seeds        int            `json:"seeds,omitempty"`
	Peers        int            `json:"peers,omitempty"`
	AgeDays      float64        `json:"age_days,omitempty"`
	Quality      plugin.Quality `json:"quality"`
	QualityScore int            `json:"quality_score"`
}

type releaseListOutput struct {
	Body []*releaseBody
}

type releasesSearchInput struct {
	MovieID string `path:"id"`
}

// grabHistoryBody is a summary of one recorded grab.
type grabHistoryBody struct {
	ID               string    `json:"id"`
	MovieID          string    `json:"movie_id"`
	IndexerID        *string   `json:"indexer_id,omitempty"`
	ReleaseGUID      string    `json:"release_guid"`
	ReleaseTitle     string    `json:"release_title"`
	Protocol         string    `json:"protocol"`
	Size             int64     `json:"size"`
	DownloadClientID *string   `json:"download_client_id,omitempty"`
	ClientItemID     *string   `json:"client_item_id,omitempty"`
	DownloadStatus   string    `json:"download_status"`
	GrabbedAt        time.Time `json:"grabbed_at"`
}

// grabInput carries the release the client wants to grab.
type grabInput struct {
	MovieID string `path:"id"`
	Body    grabReleaseBody
}

type grabReleaseBody struct {
	GUID        string          `json:"guid"`
	Title       string          `json:"title"`
	IndexerID   string          `json:"indexer_id,omitempty"`
	Protocol    string          `json:"protocol"`
	DownloadURL string          `json:"download_url"`
	Size        int64           `json:"size"`
	Quality     json.RawMessage `json:"quality,omitempty"`
}

type grabOutput struct {
	Body *grabHistoryBody
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func indexerResultToBody(r indexer.SearchResult) *releaseBody {
	return &releaseBody{
		GUID:         r.GUID,
		Title:        r.Title,
		Indexer:      r.Indexer,
		Protocol:     string(r.Protocol),
		DownloadURL:  r.DownloadURL,
		InfoURL:      r.InfoURL,
		Size:         r.Size,
		Seeds:        r.Seeds,
		Peers:        r.Peers,
		AgeDays:      r.AgeDays,
		Quality:      r.Quality,
		QualityScore: r.QualityScore,
	}
}

// ── Route registration ───────────────────────────────────────────────────────

// RegisterReleaseRoutes registers the release search and grab endpoints.
// downloaderSvc may be nil; in that case grabs are recorded to history without
// being sent to a download client (backward-compatible with Phase 2 mode).
func RegisterReleaseRoutes(api huma.API, indexerSvc *indexer.Service, movieSvc *movie.Service, downloaderSvc *downloader.Service, logger *slog.Logger) {
	// GET /api/v1/movies/{id}/releases
	huma.Register(api, huma.Operation{
		OperationID: "search-releases",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/releases",
		Summary:     "Search for releases for a movie across all enabled indexers",
		Tags:        []string{"Releases"},
	}, func(ctx context.Context, input *releasesSearchInput) (*releaseListOutput, error) {
		m, err := movieSvc.Get(ctx, input.MovieID)
		if err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}

		query := plugin.SearchQuery{
			TMDBID: m.TMDBID,
			IMDBID: m.IMDBID,
			Query:  m.Title,
			Year:   m.Year,
		}

		results, searchErr := indexerSvc.Search(ctx, query)
		bodies := make([]*releaseBody, len(results))
		for i, r := range results {
			bodies[i] = indexerResultToBody(r)
		}

		if len(bodies) == 0 && searchErr != nil {
			return nil, huma.NewError(http.StatusBadGateway, searchErr.Error())
		}

		return &releaseListOutput{Body: bodies}, nil
	})

	// POST /api/v1/movies/{id}/releases/grab
	huma.Register(api, huma.Operation{
		OperationID:   "grab-release",
		Method:        http.MethodPost,
		Path:          "/api/v1/movies/{id}/releases/grab",
		Summary:       "Grab a release — submits to a download client and records history",
		Tags:          []string{"Releases"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *grabInput) (*grabOutput, error) {
		if _, err := movieSvc.Get(ctx, input.MovieID); err != nil {
			if errors.Is(err, movie.ErrNotFound) {
				return nil, huma.Error404NotFound("movie not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get movie", err)
		}

		var qual plugin.Quality
		if len(input.Body.Quality) > 0 {
			_ = json.Unmarshal(input.Body.Quality, &qual)
		}

		release := plugin.Release{
			GUID:        input.Body.GUID,
			Title:       input.Body.Title,
			Protocol:    plugin.Protocol(input.Body.Protocol),
			DownloadURL: input.Body.DownloadURL,
			Size:        input.Body.Size,
			Quality:     qual,
		}

		// Submit to download client when one is configured.
		var dcID, itemID string
		if downloaderSvc != nil {
			id, item, err := downloaderSvc.Add(ctx, release)
			if err != nil {
				if errors.Is(err, downloader.ErrNoCompatibleClient) {
					return nil, huma.NewError(http.StatusServiceUnavailable,
						"no download client configured for this protocol — add one at /api/v1/download-clients", err)
				}
				return nil, huma.NewError(http.StatusBadGateway, "download client: "+err.Error())
			}
			dcID = id
			itemID = item
		}

		history, err := indexerSvc.Grab(ctx, input.MovieID, input.Body.IndexerID, release, dcID, itemID)
		if err != nil {
			logger.Error("grab: failed to record grab history",
				"movie_id", input.MovieID,
				"indexer_id", input.Body.IndexerID,
				"dc_id", dcID,
				"item_id", itemID,
				"error", err,
			)
			return nil, huma.NewError(http.StatusInternalServerError, "failed to record grab: "+err.Error())
		}

		grabbedAt, _ := time.Parse(time.RFC3339, history.GrabbedAt)
		out := &grabHistoryBody{
			ID:               history.ID,
			MovieID:          history.MovieID,
			IndexerID:        history.IndexerID,
			ReleaseGUID:      history.ReleaseGuid,
			ReleaseTitle:     history.ReleaseTitle,
			Protocol:         history.Protocol,
			Size:             history.Size,
			DownloadClientID: history.DownloadClientID,
			ClientItemID:     history.ClientItemID,
			DownloadStatus:   history.DownloadStatus,
			GrabbedAt:        grabbedAt,
		}
		return &grabOutput{Body: out}, nil
	})
}
