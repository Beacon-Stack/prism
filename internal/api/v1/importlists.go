package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/importlist"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/registry"
)

// ── Request / response shapes ─────────────────────────────────────────────────

type importListBody struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Kind             string          `json:"kind"               doc:"Plugin kind: tmdb_popular, trakt_list, etc."`
	Enabled          bool            `json:"enabled"`
	Settings         json.RawMessage `json:"settings"           doc:"Plugin-specific settings as JSON"`
	SearchOnAdd      bool            `json:"search_on_add"      doc:"Auto-search for releases when a movie is added"`
	Monitor          bool            `json:"monitor"            doc:"Set added movies as monitored"`
	MinAvailability  string          `json:"min_availability"   doc:"Minimum availability for added movies"`
	QualityProfileID string          `json:"quality_profile_id" doc:"Quality profile for added movies"`
	LibraryID        string          `json:"library_id"         doc:"Target library for added movies"`
	TagIDs           []string        `json:"tag_ids"            doc:"Assigned tag UUIDs"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type importListListOutput struct {
	Body []*importListBody
}

type importListGetOutput struct {
	Body *importListBody
}

type importListInput struct {
	ID string `path:"id"`
}

type importListCreateBody struct {
	Name             string          `json:"name"               minLength:"1"`
	Kind             string          `json:"kind"               minLength:"1"`
	Enabled          bool            `json:"enabled"`
	Settings         json.RawMessage `json:"settings,omitempty"`
	SearchOnAdd      bool            `json:"search_on_add"`
	Monitor          bool            `json:"monitor"`
	MinAvailability  string          `json:"min_availability"`
	QualityProfileID string          `json:"quality_profile_id"`
	LibraryID        string          `json:"library_id"`
	TagIDs           []string        `json:"tag_ids,omitempty"  doc:"Tag UUIDs to assign"`
}

type importListCreateInput struct {
	Body importListCreateBody
}

type importListUpdateInput struct {
	ID   string `path:"id"`
	Body importListCreateBody
}

type importListDeleteInput struct {
	ID string `path:"id"`
}

type importListDeleteOutput struct{}

type importListTestInput struct {
	ID string `path:"id"`
}

type importListTestOutput struct{}

type importListSyncOutput struct {
	Body *importlist.SyncResult
}

// ── Preview shapes ──────────────────────────────────────────────────────────

type importListPreviewBody struct {
	Kind     string          `json:"kind"     minLength:"1"`
	Settings json.RawMessage `json:"settings,omitempty"`
}

type importListPreviewInput struct {
	Body importListPreviewBody
}

type importListPreviewOutput struct {
	Body []importlist.PreviewItem
}

// ── Exclusion shapes ────────────────────────────────────────────────────────

type exclusionBody struct {
	ID        string    `json:"id"`
	TMDbID    int       `json:"tmdb_id"`
	Title     string    `json:"title"`
	Year      int       `json:"year"`
	CreatedAt time.Time `json:"created_at"`
}

type exclusionListOutput struct {
	Body []*exclusionBody
}

type exclusionGetOutput struct {
	Body *exclusionBody
}

type exclusionCreateBody struct {
	TMDbID int    `json:"tmdb_id" minimum:"1"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
}

type exclusionCreateInput struct {
	Body exclusionCreateBody
}

type exclusionDeleteInput struct {
	ID string `path:"id"`
}

type exclusionDeleteOutput struct{}

// ── Helpers ───────────────────────────────────────────────────────────────────

func importListToBody(cfg importlist.Config) *importListBody {
	return &importListBody{
		ID:               cfg.ID,
		Name:             cfg.Name,
		Kind:             cfg.Kind,
		Enabled:          cfg.Enabled,
		Settings:         registry.Default.SanitizeImportListSettings(cfg.Kind, cfg.Settings),
		SearchOnAdd:      cfg.SearchOnAdd,
		Monitor:          cfg.Monitor,
		MinAvailability:  cfg.MinAvailability,
		QualityProfileID: cfg.QualityProfileID,
		LibraryID:        cfg.LibraryID,
		CreatedAt:        cfg.CreatedAt,
		UpdatedAt:        cfg.UpdatedAt,
	}
}

func exclusionToBody(e importlist.Exclusion) *exclusionBody {
	return &exclusionBody{
		ID:        e.ID,
		TMDbID:    e.TMDbID,
		Title:     e.Title,
		Year:      e.Year,
		CreatedAt: e.CreatedAt,
	}
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterImportListRoutes registers the /api/v1/importlists endpoints.
func RegisterImportListRoutes(api huma.API, svc *importlist.Service, tagSvc *tag.Service) {
	// GET /api/v1/importlists
	huma.Register(api, huma.Operation{
		OperationID: "list-import-lists",
		Method:      http.MethodGet,
		Path:        "/api/v1/importlists",
		Summary:     "List all import list configurations",
		Tags:        []string{"Import Lists"},
	}, func(ctx context.Context, _ *struct{}) (*importListListOutput, error) {
		cfgs, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list import lists", err)
		}
		bodies := make([]*importListBody, len(cfgs))
		for i, c := range cfgs {
			b := importListToBody(c)
			if tagSvc != nil {
				b.TagIDs, _ = tagSvc.ImportListTagIDs(ctx, c.ID)
			}
			if b.TagIDs == nil {
				b.TagIDs = []string{}
			}
			bodies[i] = b
		}
		return &importListListOutput{Body: bodies}, nil
	})

	// POST /api/v1/importlists
	huma.Register(api, huma.Operation{
		OperationID:   "create-import-list",
		Method:        http.MethodPost,
		Path:          "/api/v1/importlists",
		Summary:       "Create an import list configuration",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *importListCreateInput) (*importListGetOutput, error) {
		cfg, err := svc.Create(ctx, importlist.CreateRequest{
			Name:             input.Body.Name,
			Kind:             input.Body.Kind,
			Enabled:          input.Body.Enabled,
			Settings:         input.Body.Settings,
			SearchOnAdd:      input.Body.SearchOnAdd,
			Monitor:          input.Body.Monitor,
			MinAvailability:  input.Body.MinAvailability,
			QualityProfileID: input.Body.QualityProfileID,
			LibraryID:        input.Body.LibraryID,
		})
		if err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "failed to create import list", err)
		}
		b := importListToBody(cfg)
		if tagSvc != nil && len(input.Body.TagIDs) > 0 {
			_ = tagSvc.SetImportListTags(ctx, cfg.ID, input.Body.TagIDs)
			b.TagIDs = input.Body.TagIDs
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &importListGetOutput{Body: b}, nil
	})

	// GET /api/v1/importlists/{id}
	huma.Register(api, huma.Operation{
		OperationID: "get-import-list",
		Method:      http.MethodGet,
		Path:        "/api/v1/importlists/{id}",
		Summary:     "Get an import list configuration",
		Tags:        []string{"Import Lists"},
	}, func(ctx context.Context, input *importListInput) (*importListGetOutput, error) {
		cfg, err := svc.Get(ctx, input.ID)
		if err != nil {
			if errors.Is(err, importlist.ErrNotFound) {
				return nil, huma.Error404NotFound("import list not found")
			}
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get import list", err)
		}
		b := importListToBody(cfg)
		if tagSvc != nil {
			b.TagIDs, _ = tagSvc.ImportListTagIDs(ctx, cfg.ID)
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &importListGetOutput{Body: b}, nil
	})

	// PUT /api/v1/importlists/{id}
	huma.Register(api, huma.Operation{
		OperationID: "update-import-list",
		Method:      http.MethodPut,
		Path:        "/api/v1/importlists/{id}",
		Summary:     "Update an import list configuration",
		Tags:        []string{"Import Lists"},
	}, func(ctx context.Context, input *importListUpdateInput) (*importListGetOutput, error) {
		cfg, err := svc.Update(ctx, input.ID, importlist.UpdateRequest{
			Name:             input.Body.Name,
			Kind:             input.Body.Kind,
			Enabled:          input.Body.Enabled,
			Settings:         input.Body.Settings,
			SearchOnAdd:      input.Body.SearchOnAdd,
			Monitor:          input.Body.Monitor,
			MinAvailability:  input.Body.MinAvailability,
			QualityProfileID: input.Body.QualityProfileID,
			LibraryID:        input.Body.LibraryID,
		})
		if err != nil {
			if errors.Is(err, importlist.ErrNotFound) {
				return nil, huma.Error404NotFound("import list not found")
			}
			return nil, huma.NewError(http.StatusUnprocessableEntity, "failed to update import list", err)
		}
		b := importListToBody(cfg)
		if tagSvc != nil {
			if input.Body.TagIDs != nil {
				_ = tagSvc.SetImportListTags(ctx, cfg.ID, input.Body.TagIDs)
				b.TagIDs = input.Body.TagIDs
			} else {
				b.TagIDs, _ = tagSvc.ImportListTagIDs(ctx, cfg.ID)
			}
		}
		if b.TagIDs == nil {
			b.TagIDs = []string{}
		}
		return &importListGetOutput{Body: b}, nil
	})

	// DELETE /api/v1/importlists/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-import-list",
		Method:        http.MethodDelete,
		Path:          "/api/v1/importlists/{id}",
		Summary:       "Delete an import list configuration",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *importListDeleteInput) (*importListDeleteOutput, error) {
		if err := svc.Delete(ctx, input.ID); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete import list", err)
		}
		return &importListDeleteOutput{}, nil
	})

	// POST /api/v1/importlists/{id}/test
	huma.Register(api, huma.Operation{
		OperationID:   "test-import-list",
		Method:        http.MethodPost,
		Path:          "/api/v1/importlists/{id}/test",
		Summary:       "Test an import list connection",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *importListTestInput) (*importListTestOutput, error) {
		if err := svc.Test(ctx, input.ID); err != nil {
			if errors.Is(err, importlist.ErrNotFound) {
				return nil, huma.Error404NotFound("import list not found")
			}
			return nil, huma.NewError(http.StatusBadGateway, "test failed", err)
		}
		return &importListTestOutput{}, nil
	})

	// POST /api/v1/importlists/sync
	huma.Register(api, huma.Operation{
		OperationID:   "sync-all-import-lists",
		Method:        http.MethodPost,
		Path:          "/api/v1/importlists/sync",
		Summary:       "Trigger a sync of all enabled import lists",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, _ *struct{}) (*importListSyncOutput, error) {
		result := svc.Sync(ctx)
		return &importListSyncOutput{Body: &result}, nil
	})

	// POST /api/v1/importlists/{id}/sync
	huma.Register(api, huma.Operation{
		OperationID:   "sync-import-list",
		Method:        http.MethodPost,
		Path:          "/api/v1/importlists/{id}/sync",
		Summary:       "Trigger a sync of a single import list",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *importListInput) (*importListSyncOutput, error) {
		result := svc.SyncOne(ctx, input.ID)
		return &importListSyncOutput{Body: &result}, nil
	})

	// POST /api/v1/importlists/preview
	huma.Register(api, huma.Operation{
		OperationID: "preview-import-list",
		Method:      http.MethodPost,
		Path:        "/api/v1/importlists/preview",
		Summary:     "Preview items from an import list source without saving",
		Tags:        []string{"Import Lists"},
	}, func(ctx context.Context, input *importListPreviewInput) (*importListPreviewOutput, error) {
		items, err := svc.Preview(ctx, input.Body.Kind, input.Body.Settings)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "preview failed", err)
		}
		return &importListPreviewOutput{Body: items}, nil
	})

	// ── Exclusions ────────────────────────────────────────────────────────────

	// GET /api/v1/importlists/exclusions
	huma.Register(api, huma.Operation{
		OperationID: "list-import-exclusions",
		Method:      http.MethodGet,
		Path:        "/api/v1/importlists/exclusions",
		Summary:     "List all import exclusions",
		Tags:        []string{"Import Lists"},
	}, func(ctx context.Context, _ *struct{}) (*exclusionListOutput, error) {
		excls, err := svc.ListExclusions(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list exclusions", err)
		}
		bodies := make([]*exclusionBody, len(excls))
		for i, e := range excls {
			bodies[i] = exclusionToBody(e)
		}
		return &exclusionListOutput{Body: bodies}, nil
	})

	// POST /api/v1/importlists/exclusions
	huma.Register(api, huma.Operation{
		OperationID:   "create-import-exclusion",
		Method:        http.MethodPost,
		Path:          "/api/v1/importlists/exclusions",
		Summary:       "Add a movie to the import exclusion list",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *exclusionCreateInput) (*exclusionGetOutput, error) {
		excl, err := svc.CreateExclusion(ctx, input.Body.TMDbID, input.Body.Title, input.Body.Year)
		if err != nil {
			return nil, huma.NewError(http.StatusUnprocessableEntity, "failed to create exclusion", err)
		}
		return &exclusionGetOutput{Body: exclusionToBody(excl)}, nil
	})

	// DELETE /api/v1/importlists/exclusions/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-import-exclusion",
		Method:        http.MethodDelete,
		Path:          "/api/v1/importlists/exclusions/{id}",
		Summary:       "Remove a movie from the import exclusion list",
		Tags:          []string{"Import Lists"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *exclusionDeleteInput) (*exclusionDeleteOutput, error) {
		if err := svc.DeleteExclusion(ctx, input.ID); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete exclusion", err)
		}
		return &exclusionDeleteOutput{}, nil
	})
}
