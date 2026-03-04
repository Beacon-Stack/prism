package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/indexer"
)

type historyItemBody struct {
	ID                string          `json:"id"`
	MovieID           string          `json:"movie_id"`
	ReleaseTitle      string          `json:"release_title"`
	ReleaseSource     string          `json:"release_source,omitempty"`
	ReleaseResolution string          `json:"release_resolution,omitempty"`
	Protocol          string          `json:"protocol"`
	Size              int64           `json:"size"`
	DownloadStatus    string          `json:"download_status"`
	GrabbedAt         time.Time       `json:"grabbed_at"`
	ScoreBreakdown    json.RawMessage `json:"score_breakdown,omitempty"`
}

type historyListInput struct {
	Limit          int    `query:"limit"           default:"100" minimum:"1" maximum:"1000"`
	DownloadStatus string `query:"download_status" doc:"Filter by status: completed, failed, queued, downloading, paused, removed"`
	Protocol       string `query:"protocol"        doc:"Filter by protocol: torrent, nzb"`
}

type historyListOutput struct {
	Body []*historyItemBody
}

// RegisterHistoryRoutes registers the global grab history endpoint and the
// per-movie grab history endpoint.
func RegisterHistoryRoutes(api huma.API, svc *indexer.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-history",
		Method:      http.MethodGet,
		Path:        "/api/v1/history",
		Summary:     "List grab history",
		Tags:        []string{"History"},
	}, func(ctx context.Context, input *historyListInput) (*historyListOutput, error) {
		limit := input.Limit
		if limit == 0 {
			limit = 100
		}
		// Fetch with a higher cap when filters are active, to avoid filtering
		// on an already-truncated result set.
		fetchLimit := limit
		if input.DownloadStatus != "" || input.Protocol != "" {
			fetchLimit = 1000
		}
		rows, err := svc.ListHistory(ctx, fetchLimit)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list history", err)
		}

		items := make([]*historyItemBody, 0, len(rows))
		for _, r := range rows {
			if input.DownloadStatus != "" && !strings.EqualFold(r.DownloadStatus, input.DownloadStatus) {
				continue
			}
			if input.Protocol != "" && !strings.EqualFold(r.Protocol, input.Protocol) {
				continue
			}
			grabbedAt, _ := time.Parse(time.RFC3339, r.GrabbedAt)
			item := &historyItemBody{
				ID:                r.ID,
				MovieID:           r.MovieID,
				ReleaseTitle:      r.ReleaseTitle,
				ReleaseSource:     r.ReleaseSource,
				ReleaseResolution: r.ReleaseResolution,
				Protocol:          r.Protocol,
				Size:              r.Size,
				DownloadStatus:    r.DownloadStatus,
				GrabbedAt:         grabbedAt,
			}
			if r.ScoreBreakdown != "" {
				item.ScoreBreakdown = json.RawMessage(r.ScoreBreakdown)
			}
			items = append(items, item)
			if len(items) == limit {
				break
			}
		}
		return &historyListOutput{Body: items}, nil
	})

	type movieHistoryInput struct {
		ID string `path:"id"`
	}

	huma.Register(api, huma.Operation{
		OperationID: "list-movie-history",
		Method:      http.MethodGet,
		Path:        "/api/v1/movies/{id}/history",
		Summary:     "List grab history for a specific movie",
		Tags:        []string{"History"},
	}, func(ctx context.Context, input *movieHistoryInput) (*historyListOutput, error) {
		rows, err := svc.GrabHistory(ctx, input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list movie history", err)
		}
		items := make([]*historyItemBody, len(rows))
		for i, r := range rows {
			grabbedAt, _ := time.Parse(time.RFC3339, r.GrabbedAt)
			item := &historyItemBody{
				ID:                r.ID,
				MovieID:           r.MovieID,
				ReleaseTitle:      r.ReleaseTitle,
				ReleaseSource:     r.ReleaseSource,
				ReleaseResolution: r.ReleaseResolution,
				Protocol:          r.Protocol,
				Size:              r.Size,
				DownloadStatus:    r.DownloadStatus,
				GrabbedAt:         grabbedAt,
			}
			if r.ScoreBreakdown != "" {
				item.ScoreBreakdown = json.RawMessage(r.ScoreBreakdown)
			}
			items[i] = item
		}
		return &historyListOutput{Body: items}, nil
	})
}
