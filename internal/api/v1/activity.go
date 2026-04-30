package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/activity"
)

type listActivitiesInput struct {
	Category string `query:"category" doc:"Filter by category: grab_succeeded, grab_failed, import_succeeded, import_failed, task, health, movie (legacy 'grab' / 'import' also accepted)"`
	Since    string `query:"since"    doc:"Only return activities after this ISO 8601 timestamp"`
	Limit    int64  `query:"limit"    doc:"Max results (default 100, max 500)" default:"100"`
}

type listActivitiesOutput struct {
	Body *activity.ListResult
}

type needsAttentionInput struct {
	Hours   int `query:"hours"    doc:"How many hours back to look (default 48)" default:"48"`
	PerKind int `query:"per_kind" doc:"Cap per bucket so one type doesn't drown the others (default 50, max 200)" default:"50"`
}

type needsAttentionOutput struct {
	Body *activity.AttentionResult
}

// RegisterActivityRoutes registers the /api/v1/activity endpoints.
func RegisterActivityRoutes(api huma.API, svc *activity.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-activity",
		Method:      http.MethodGet,
		Path:        "/api/v1/activity",
		Summary:     "List activity log",
		Description: "Returns a chronological feed of system events. The Activity page uses the dedicated rail endpoints (/queue, /history, /activity/needs-attention) for its main view; this endpoint is the firehose for debugging and the legacy timeline.",
		Tags:        []string{"Activity"},
	}, func(ctx context.Context, input *listActivitiesInput) (*listActivitiesOutput, error) {
		if input.Category != "" && !activity.ValidCategory(input.Category) {
			return nil, huma.NewError(http.StatusBadRequest, "invalid category")
		}

		var catPtr, sincePtr *string
		if input.Category != "" {
			catPtr = &input.Category
		}
		if input.Since != "" {
			sincePtr = &input.Since
		}

		result, err := svc.List(ctx, catPtr, sincePtr, input.Limit)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list activities", err)
		}

		return &listActivitiesOutput{Body: result}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-activity-needs-attention",
		Method:      http.MethodGet,
		Path:        "/api/v1/activity/needs-attention",
		Summary:     "Recent failures and removed grabs",
		Description: "Powers the Activity-page \"Needs attention\" rail. Returns failed/removed grabs from grab_history and import failures from activity_log within the requested window.",
		Tags:        []string{"Activity"},
	}, func(ctx context.Context, input *needsAttentionInput) (*needsAttentionOutput, error) {
		hours := input.Hours
		if hours <= 0 {
			hours = 48
		}
		result, err := svc.NeedsAttention(ctx, time.Duration(hours)*time.Hour, input.PerKind)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to compute needs-attention", err)
		}
		return &needsAttentionOutput{Body: result}, nil
	})
}
