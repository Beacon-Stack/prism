package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/mediainfo"
)

type mediainfoStatusBody struct {
	Available   bool   `json:"available"`
	FFprobePath string `json:"ffprobe_path,omitempty"`
}

type mediainfoStatusOutput struct {
	Body mediainfoStatusBody
}

// RegisterMediainfoRoutes registers the scanner status and bulk scan endpoints.
// mediaSvc may be nil (scanner not configured); status returns available=false.
func RegisterMediainfoRoutes(api huma.API, mediaSvc *mediainfo.Service) {
	// GET /api/v1/mediainfo/status
	huma.Register(api, huma.Operation{
		OperationID: "mediainfo-status",
		Method:      http.MethodGet,
		Path:        "/api/v1/mediainfo/status",
		Summary:     "Return ffprobe scanner availability",
		Tags:        []string{"MediaInfo"},
	}, func(_ context.Context, _ *struct{}) (*mediainfoStatusOutput, error) {
		if mediaSvc == nil || !mediaSvc.Available() {
			return &mediainfoStatusOutput{Body: mediainfoStatusBody{Available: false}}, nil
		}
		return &mediainfoStatusOutput{Body: mediainfoStatusBody{
			Available:   true,
			FFprobePath: mediaSvc.FFprobeVersion(),
		}}, nil
	})

	// POST /api/v1/mediainfo/scan-all — scan every file without existing mediainfo
	huma.Register(api, huma.Operation{
		OperationID:   "mediainfo-scan-all",
		Method:        http.MethodPost,
		Path:          "/api/v1/mediainfo/scan-all",
		Summary:       "Scan all movie files that have not yet been scanned",
		Tags:          []string{"MediaInfo"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if mediaSvc == nil || !mediaSvc.Available() {
			return nil, huma.NewError(http.StatusServiceUnavailable, "mediainfo scanning not available — install ffprobe")
		}
		// Fire in the background; the client polls or watches WebSocket events.
		go func() {
			_, _ = mediaSvc.ScanAll(context.Background())
		}()
		return nil, nil
	})
}
