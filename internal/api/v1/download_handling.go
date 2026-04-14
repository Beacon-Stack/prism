package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/downloadhandling"
)

// ── Settings shapes ───────────────────────────────────────────────────────────

type downloadHandlingBody struct {
	EnableCompleted             bool  `json:"enable_completed"               doc:"Automatically import completed downloads"`
	CheckIntervalMinutes        int32 `json:"check_interval_minutes"         doc:"How often to check for finished downloads (minutes)"`
	RedownloadFailed            bool  `json:"redownload_failed"              doc:"Automatically redownload failed downloads"`
	RedownloadFailedInteractive bool  `json:"redownload_failed_interactive"  doc:"Redownload failed downloads found via interactive search"`
}

type downloadHandlingOutput struct {
	Body *downloadHandlingBody
}

type downloadHandlingInput struct {
	Body *downloadHandlingBody
}

// ── Remote path mapping shapes ─────────────────────────────────────────────────

type remotePathMappingBody struct {
	ID         string `json:"id"          doc:"Unique mapping ID"`
	Host       string `json:"host"        doc:"Download client hostname"`
	RemotePath string `json:"remote_path" doc:"Path as seen by the download client"`
	LocalPath  string `json:"local_path"  doc:"Corresponding local path on this host"`
}

type remotePathMappingListOutput struct {
	Body []*remotePathMappingBody
}

type remotePathMappingOutput struct {
	Body *remotePathMappingBody
}

type createRemotePathMappingInput struct {
	Body *struct {
		Host       string `json:"host"        required:"true" doc:"Download client hostname"`
		RemotePath string `json:"remote_path" required:"true" doc:"Path as seen by the download client"`
		LocalPath  string `json:"local_path"  required:"true" doc:"Corresponding local path on this host"`
	}
}

type deleteRemotePathMappingInput struct {
	ID string `path:"id"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func dhSettingsToBody(s downloadhandling.Settings) *downloadHandlingBody {
	return &downloadHandlingBody{
		EnableCompleted:             s.EnableCompleted,
		CheckIntervalMinutes:        s.CheckIntervalMinutes,
		RedownloadFailed:            s.RedownloadFailed,
		RedownloadFailedInteractive: s.RedownloadFailedInteractive,
	}
}

func dhBodyToSettings(b *downloadHandlingBody) downloadhandling.Settings {
	return downloadhandling.Settings{
		EnableCompleted:             b.EnableCompleted,
		CheckIntervalMinutes:        b.CheckIntervalMinutes,
		RedownloadFailed:            b.RedownloadFailed,
		RedownloadFailedInteractive: b.RedownloadFailedInteractive,
	}
}

func mappingToBody(m downloadhandling.RemotePathMapping) *remotePathMappingBody {
	return &remotePathMappingBody{
		ID:         m.ID,
		Host:       m.Host,
		RemotePath: m.RemotePath,
		LocalPath:  m.LocalPath,
	}
}

// ── Route registration ─────────────────────────────────────────────────────────

// RegisterDownloadHandlingRoutes registers download handling settings and
// remote path mapping endpoints.
func RegisterDownloadHandlingRoutes(api huma.API, svc *downloadhandling.Service) {
	// GET /api/v1/download-handling
	huma.Register(api, huma.Operation{
		OperationID: "get-download-handling",
		Method:      http.MethodGet,
		Path:        "/api/v1/download-handling",
		Summary:     "Get download handling settings",
		Tags:        []string{"Download Handling"},
	}, func(ctx context.Context, _ *struct{}) (*downloadHandlingOutput, error) {
		s, err := svc.Get(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to get download handling settings", err)
		}
		return &downloadHandlingOutput{Body: dhSettingsToBody(s)}, nil
	})

	// PUT /api/v1/download-handling
	huma.Register(api, huma.Operation{
		OperationID: "update-download-handling",
		Method:      http.MethodPut,
		Path:        "/api/v1/download-handling",
		Summary:     "Update download handling settings",
		Tags:        []string{"Download Handling"},
	}, func(ctx context.Context, input *downloadHandlingInput) (*downloadHandlingOutput, error) {
		updated, err := svc.Update(ctx, dhBodyToSettings(input.Body))
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update download handling settings", err)
		}
		return &downloadHandlingOutput{Body: dhSettingsToBody(updated)}, nil
	})

	// GET /api/v1/download-handling/remote-path-mappings
	huma.Register(api, huma.Operation{
		OperationID: "list-remote-path-mappings",
		Method:      http.MethodGet,
		Path:        "/api/v1/download-handling/remote-path-mappings",
		Summary:     "List remote path mappings",
		Tags:        []string{"Download Handling"},
	}, func(ctx context.Context, _ *struct{}) (*remotePathMappingListOutput, error) {
		mappings, err := svc.ListRemotePathMappings(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list remote path mappings", err)
		}
		bodies := make([]*remotePathMappingBody, len(mappings))
		for i, m := range mappings {
			bodies[i] = mappingToBody(m)
		}
		return &remotePathMappingListOutput{Body: bodies}, nil
	})

	// POST /api/v1/download-handling/remote-path-mappings
	huma.Register(api, huma.Operation{
		OperationID:   "create-remote-path-mapping",
		Method:        http.MethodPost,
		Path:          "/api/v1/download-handling/remote-path-mappings",
		Summary:       "Create a remote path mapping",
		Tags:          []string{"Download Handling"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *createRemotePathMappingInput) (*remotePathMappingOutput, error) {
		m, err := svc.CreateRemotePathMapping(ctx, input.Body.Host, input.Body.RemotePath, input.Body.LocalPath)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to create remote path mapping", err)
		}
		return &remotePathMappingOutput{Body: mappingToBody(m)}, nil
	})

	// DELETE /api/v1/download-handling/remote-path-mappings/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "delete-remote-path-mapping",
		Method:        http.MethodDelete,
		Path:          "/api/v1/download-handling/remote-path-mappings/{id}",
		Summary:       "Delete a remote path mapping",
		Tags:          []string{"Download Handling"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *deleteRemotePathMappingInput) (*struct{}, error) {
		if err := svc.DeleteRemotePathMapping(ctx, input.ID); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to delete remote path mapping", err)
		}
		return nil, nil
	})
}
