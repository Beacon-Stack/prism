package v1

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-media/prism/internal/core/mediamanagement"
)

// ── Response / request shapes ─────────────────────────────────────────────────

type mediaManagementBody struct {
	RenameMovies           bool   `json:"rename_movies"            doc:"Rename movie files on import"`
	StandardMovieFormat    string `json:"standard_movie_format"    doc:"File naming format template"`
	MovieFolderFormat      string `json:"movie_folder_format"      doc:"Folder naming format template"`
	ColonReplacement       string `json:"colon_replacement"        doc:"How to handle colons in titles: delete, dash, space-dash, smart"`
	ImportExtraFiles       bool   `json:"import_extra_files"       doc:"Copy extra files (subtitles, NFOs) alongside the video"`
	ExtraFileExtensions    string `json:"extra_file_extensions"    doc:"Comma-separated list of extra file extensions to import"`
	UnmonitorDeletedMovies bool   `json:"unmonitor_deleted_movies" doc:"Unmonitor movies whose files are deleted from disk"`
}

type mediaManagementOutput struct {
	Body *mediaManagementBody
}

type mediaManagementInput struct {
	Body *mediaManagementBody
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func settingsToBody(s mediamanagement.Settings) *mediaManagementBody {
	return &mediaManagementBody{
		RenameMovies:           s.RenameMovies,
		StandardMovieFormat:    s.StandardMovieFormat,
		MovieFolderFormat:      s.MovieFolderFormat,
		ColonReplacement:       s.ColonReplacement,
		ImportExtraFiles:       s.ImportExtraFiles,
		ExtraFileExtensions:    strings.Join(s.ExtraFileExtensions, ","),
		UnmonitorDeletedMovies: s.UnmonitorDeletedMovies,
	}
}

func bodyToSettings(b *mediaManagementBody) mediamanagement.Settings {
	exts := []string{}
	for _, e := range strings.Split(b.ExtraFileExtensions, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			exts = append(exts, e)
		}
	}
	return mediamanagement.Settings{
		RenameMovies:           b.RenameMovies,
		StandardMovieFormat:    b.StandardMovieFormat,
		MovieFolderFormat:      b.MovieFolderFormat,
		ColonReplacement:       b.ColonReplacement,
		ImportExtraFiles:       b.ImportExtraFiles,
		ExtraFileExtensions:    exts,
		UnmonitorDeletedMovies: b.UnmonitorDeletedMovies,
	}
}

// ── Route registration ────────────────────────────────────────────────────────

// RegisterMediaManagementRoutes registers /api/v1/media-management endpoints.
func RegisterMediaManagementRoutes(api huma.API, svc *mediamanagement.Service) {
	// GET /api/v1/media-management
	huma.Register(api, huma.Operation{
		OperationID: "get-media-management",
		Method:      http.MethodGet,
		Path:        "/api/v1/media-management",
		Summary:     "Get media management settings",
		Tags:        []string{"Media Management"},
	}, func(ctx context.Context, _ *struct{}) (*mediaManagementOutput, error) {
		s, err := svc.Get(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get media management settings", err)
		}
		return &mediaManagementOutput{Body: settingsToBody(s)}, nil
	})

	// PUT /api/v1/media-management
	huma.Register(api, huma.Operation{
		OperationID: "update-media-management",
		Method:      http.MethodPut,
		Path:        "/api/v1/media-management",
		Summary:     "Update media management settings",
		Tags:        []string{"Media Management"},
	}, func(ctx context.Context, input *mediaManagementInput) (*mediaManagementOutput, error) {
		updated, err := svc.Update(ctx, bodyToSettings(input.Body))
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update media management settings", err)
		}
		return &mediaManagementOutput{Body: settingsToBody(updated)}, nil
	})
}
