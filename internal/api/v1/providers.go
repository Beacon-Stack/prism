package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/provider"
)

// Providers API — rotates the baked-in TMDB / Trakt keys without
// rebuilding the image. Read endpoints return a redacted preview and
// the source ("default" | "override"); the actual key value is never
// sent back through any endpoint.
//
// Changes take effect on the next Prism restart — the TMDB client is
// constructed once at startup.

type providerStatusInput struct {
	Name string `path:"name" doc:"Provider name: tmdb | trakt"`
}

type providerStatusOutput struct {
	Body *providerStatusBody
}

type providerStatusBody struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Preview     string `json:"preview"`
	HasDefault  bool   `json:"hasDefault"`
	HasOverride bool   `json:"hasOverride"`
}

type setProviderOverrideInput struct {
	Name string `path:"name"`
	Body struct {
		Value string `json:"value"`
	}
}

// RegisterProviderRoutes wires the /api/v1/settings/providers/{name}
// endpoints onto the given Huma API.
func RegisterProviderRoutes(api huma.API, r *provider.Resolver) {
	huma.Register(api, huma.Operation{
		OperationID: "get-provider",
		Method:      http.MethodGet,
		Path:        "/api/v1/settings/providers/{name}",
		Summary:     "Get status of a third-party provider key",
		Description: "Returns where the effective key came from (baked default or UI override) plus a redacted preview. The full key value is never returned.",
		Tags:        []string{"Settings"},
	}, func(ctx context.Context, input *providerStatusInput) (*providerStatusOutput, error) {
		return buildStatus(ctx, r, input.Name)
	})

	huma.Register(api, huma.Operation{
		OperationID: "set-provider-override",
		Method:      http.MethodPut,
		Path:        "/api/v1/settings/providers/{name}",
		Summary:     "Set a Settings-UI override for a third-party provider key",
		Description: "Stores the supplied value as an override, replacing the baked default at the next Prism restart.",
		Tags:        []string{"Settings"},
	}, func(ctx context.Context, input *setProviderOverrideInput) (*providerStatusOutput, error) {
		if err := r.SetOverride(ctx, input.Name, input.Body.Value); err != nil {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}
		return buildStatus(ctx, r, input.Name)
	})

	huma.Register(api, huma.Operation{
		OperationID: "clear-provider-override",
		Method:      http.MethodDelete,
		Path:        "/api/v1/settings/providers/{name}",
		Summary:     "Clear the Settings-UI override for a third-party provider key",
		Description: "Reverts to the baked default at the next Prism restart.",
		Tags:        []string{"Settings"},
	}, func(ctx context.Context, input *providerStatusInput) (*providerStatusOutput, error) {
		if err := r.ClearOverride(ctx, input.Name); err != nil {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}
		return buildStatus(ctx, r, input.Name)
	})
}

func buildStatus(ctx context.Context, r *provider.Resolver, name string) (*providerStatusOutput, error) {
	hasOverride, err := r.HasOverride(ctx, name)
	if err != nil {
		return nil, huma.NewError(http.StatusNotFound, "unknown provider: "+name)
	}
	preview, source, err := r.Preview(ctx, name)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "previewing provider key", err)
	}
	effective, _, err := r.EffectiveKey(ctx, name)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "resolving effective key", err)
	}
	hasDefault := !hasOverride && effective != ""
	_ = errors.New // keep import available in case we need it later
	return &providerStatusOutput{Body: &providerStatusBody{
		Name:        name,
		Source:      string(source),
		Preview:     preview,
		HasDefault:  hasDefault,
		HasOverride: hasOverride,
	}}, nil
}
