package v1

// haul_history.go — endpoints that proxy to Haul's history index so
// the Prism UI can show "downloaded externally" badges and trigger
// re-imports for files Haul has but Prism's library doesn't know
// about. Mirror of Pilot's counterpart with a movie-shaped surface.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/downloader"
	"github.com/beacon-stack/prism/internal/core/importer"
	"github.com/beacon-stack/prism/plugins/downloaders/haul"
)

type movieHaulHistoryInput struct {
	MovieID string `path:"id" doc:"Movie UUID"`
}

type movieHaulHistoryOutput struct {
	Body struct {
		// Records are Haul history rows associated with this movie
		// via the requester_movie_id metadata. When empty, Haul
		// has no record of any download for this movie.
		Records []haul.HistoryRecord `json:"records"`
	}
}

type importFromHaulInput struct {
	Body struct {
		InfoHash string `json:"info_hash" doc:"Haul info hash to import"`
	}
}

type importFromHaulOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

// RegisterHaulHistoryRoutes wires the two endpoints.
func RegisterHaulHistoryRoutes(api huma.API, downloaderSvc *downloader.Service, importerSvc *importer.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-movie-haul-history",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/haul-history",
		Summary:     "List Haul torrent records associated with this movie",
		Description: "Returns Haul's view of every torrent grabbed against this movie, regardless of whether the corresponding file is linked in the Prism library. Used by the \"Haul has it\" badge.",
		Tags:        []string{"Haul"},
	}, func(ctx context.Context, input *movieHaulHistoryInput) (*movieHaulHistoryOutput, error) {
		client, err := firstHaulClient(ctx, downloaderSvc)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		out := &movieHaulHistoryOutput{}
		out.Body.Records = []haul.HistoryRecord{}
		if client == nil {
			// No Haul client configured. Empty list — UI renders no badges.
			return out, nil
		}
		records, err := client.LookupHistory(ctx, haul.HistoryFilter{
			Service: "prism",
			MovieID: input.MovieID,
			Limit:   100,
		})
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		out.Body.Records = records
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "import-from-haul",
		Method:      http.MethodPost,
		Path:        "/api/v1/import/from-haul",
		Summary:     "Run the import pipeline against an existing Haul torrent",
		Description: "Looks up the Haul history record by info_hash, resolves its movie via requester metadata, and runs the import pipeline against the file on disk.",
		Tags:        []string{"Haul"},
	}, func(ctx context.Context, input *importFromHaulInput) (*importFromHaulOutput, error) {
		hash := strings.TrimSpace(input.Body.InfoHash)
		if hash == "" {
			return nil, huma.Error400BadRequest("info_hash is required")
		}
		client, err := firstHaulClient(ctx, downloaderSvc)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		if client == nil {
			return nil, huma.Error503ServiceUnavailable("no Haul download client configured")
		}
		rec, err := client.LookupHistoryByHash(ctx, hash)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		if rec == nil {
			return nil, huma.Error404NotFound("info_hash not found in Haul history")
		}
		if rec.MovieID == "" {
			return nil, huma.Error409Conflict(
				"Haul record has no movie_id metadata — was the torrent grabbed via Prism? Sideloaded torrents can't be auto-imported")
		}

		// Reconstruct on-disk path; the importer walks directories
		// for multi-file torrents.
		contentPath := filepath.Join(rec.SavePath, rec.Name)

		if err := importerSvc.ImportFromHaulRecord(ctx, rec.MovieID, contentPath); err != nil {
			return nil, huma.Error500InternalServerError("import failed: " + err.Error())
		}

		out := &importFromHaulOutput{}
		out.Body.Status = "imported"
		return out, nil
	})
}

// firstHaulClient finds and returns the first enabled Haul download
// client. Returns (nil, nil) when no Haul client is configured.
func firstHaulClient(ctx context.Context, downloaderSvc *downloader.Service) (*haul.Client, error) {
	configs, err := downloaderSvc.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing download clients: %w", err)
	}
	for _, cfg := range configs {
		if !cfg.Enabled || cfg.Kind != "haul" {
			continue
		}
		client, err := downloaderSvc.ClientFor(ctx, cfg.ID)
		if err != nil {
			continue
		}
		hc, ok := client.(*haul.Client)
		if !ok {
			continue
		}
		return hc, nil
	}
	return nil, nil
}
