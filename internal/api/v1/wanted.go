package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/davidfic/luminarr/internal/core/movie"
)

type wantedInput struct {
	Page    int `query:"page"     default:"1"  minimum:"1"`
	PerPage int `query:"per_page" default:"25" minimum:"1" maximum:"250"`
}

type wantedListBody struct {
	Movies  []*movieBody `json:"movies"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
}

type wantedListOutput struct {
	Body *wantedListBody
}

// RegisterWantedRoutes registers the wanted/missing and cutoff-unmet endpoints.
func RegisterWantedRoutes(api huma.API, svc *movie.Service) {
	// GET /api/v1/wanted/missing — monitored movies with no file
	huma.Register(api, huma.Operation{
		OperationID: "wanted-missing",
		Method:      http.MethodGet,
		Path:        "/api/v1/wanted/missing",
		Summary:     "List monitored movies with no file",
		Tags:        []string{"Wanted"},
	}, func(ctx context.Context, input *wantedInput) (*wantedListOutput, error) {
		movies, total, err := svc.ListMissing(ctx, input.Page, input.PerPage)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list missing movies", err)
		}
		bodies := make([]*movieBody, len(movies))
		for i, m := range movies {
			bodies[i] = movieToBody(m)
		}
		return &wantedListOutput{Body: &wantedListBody{
			Movies:  bodies,
			Total:   total,
			Page:    input.Page,
			PerPage: input.PerPage,
		}}, nil
	})

	// GET /api/v1/wanted/cutoff — monitored movies whose best file is below the quality profile cutoff
	huma.Register(api, huma.Operation{
		OperationID: "wanted-cutoff",
		Method:      http.MethodGet,
		Path:        "/api/v1/wanted/cutoff",
		Summary:     "List monitored movies whose file quality is below the profile cutoff",
		Tags:        []string{"Wanted"},
	}, func(ctx context.Context, _ *struct{}) (*wantedListOutput, error) {
		movies, err := svc.ListCutoffUnmet(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list cutoff-unmet movies", err)
		}
		bodies := make([]*movieBody, len(movies))
		for i, m := range movies {
			bodies[i] = movieToBody(m)
		}
		return &wantedListOutput{Body: &wantedListBody{
			Movies:  bodies,
			Total:   int64(len(movies)),
			Page:    1,
			PerPage: len(movies),
		}}, nil
	})
}
