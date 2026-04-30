// Package activity provides a persistent activity log that records events
// from the in-memory event bus so they survive restarts and are queryable.
package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/dbutil"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
)

// Category groups related event types for filtering. Mirrors Pilot's
// success/failure split so the Activity-page "Needs attention" rail
// can find failed grabs/imports without re-parsing the type column.
type Category string

const (
	CategoryGrabSucceeded   Category = "grab_succeeded"
	CategoryGrabFailed      Category = "grab_failed"
	CategoryImportSucceeded Category = "import_succeeded"
	CategoryImportFailed    Category = "import_failed"
	CategoryTask            Category = "task"
	CategoryHealth          Category = "health"
	CategoryMovie           Category = "movie"

	// Legacy values kept in the validator so existing API clients that
	// filter by the old coarse names ("grab"/"import") don't break.
	// Migration 00005_activity_categories.sql back-fills DB rows to
	// the new vocabulary; these constants exist only to accept legacy
	// query strings.
	CategoryGrabLegacy   Category = "grab"
	CategoryImportLegacy Category = "import"
)

// ValidCategory returns true if c is a known category.
func ValidCategory(c string) bool {
	switch Category(c) {
	case CategoryGrabSucceeded, CategoryGrabFailed,
		CategoryImportSucceeded, CategoryImportFailed,
		CategoryTask, CategoryHealth, CategoryMovie,
		CategoryGrabLegacy, CategoryImportLegacy:
		return true
	}
	return false
}

// Activity is the domain representation of an activity log entry.
type Activity struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Category  string         `json:"category"`
	MovieID   *string        `json:"movie_id,omitempty"`
	Title     string         `json:"title"`
	Detail    map[string]any `json:"detail,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// ListResult is the paginated response from List.
type ListResult struct {
	Activities []Activity `json:"activities"`
	Total      int64      `json:"total"`
}

// Service records and queries activity log entries.
type Service struct {
	q      dbgen.Querier
	logger *slog.Logger
}

// NewService creates a new activity service.
func NewService(q dbgen.Querier, logger *slog.Logger) *Service {
	return &Service{q: q, logger: logger}
}

// Subscribe registers the service as an event bus handler. Call this once
// during startup after constructing the service.
func (s *Service) Subscribe(bus *events.Bus) {
	bus.Subscribe(s.handleEvent)
}

// handleEvent converts an event bus event into a persistent activity record.
func (s *Service) handleEvent(ctx context.Context, e events.Event) {
	cat, title := s.classify(e)
	if cat == "" {
		return // unknown event type — skip
	}

	var detailStr sql.NullString
	if len(e.Data) > 0 {
		b, err := json.Marshal(e.Data)
		if err == nil {
			detailStr = sql.NullString{String: string(b), Valid: true}
		}
	}

	err := s.q.InsertActivity(ctx, dbgen.InsertActivityParams{
		ID:        uuid.New().String(),
		Type:      string(e.Type),
		Category:  string(cat),
		MovieID:   dbutil.NullStringFromString(e.MovieID),
		Title:     title,
		Detail:    detailStr,
		CreatedAt: e.Timestamp.UTC().Format(time.RFC3339),
	})
	if err != nil {
		s.logger.Warn("failed to record activity", "type", e.Type, "error", err)
	}
}

// classify maps an event type to a category and human-readable title.
func (s *Service) classify(e events.Event) (Category, string) {
	data := e.Data
	str := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	switch e.Type {
	case events.TypeGrabStarted:
		release := str("release_title")
		indexer := str("indexer")
		if indexer != "" {
			return CategoryGrabSucceeded, fmt.Sprintf("Grabbed %s from %s", release, indexer)
		}
		return CategoryGrabSucceeded, fmt.Sprintf("Grabbed %s", release)

	case events.TypeGrabFailed:
		release := str("release_title")
		reason := str("reason")
		if reason != "" {
			return CategoryGrabFailed, fmt.Sprintf("Grab failed for %s: %s", release, reason)
		}
		return CategoryGrabFailed, fmt.Sprintf("Grab failed for %s", release)

	case events.TypeDownloadDone:
		release := str("release_title")
		return CategoryImportSucceeded, fmt.Sprintf("Download complete: %s", release)

	case events.TypeImportComplete:
		title := str("movie_title")
		quality := str("quality")
		if quality != "" {
			return CategoryImportSucceeded, fmt.Sprintf("Imported %s — %s", title, quality)
		}
		return CategoryImportSucceeded, fmt.Sprintf("Imported %s", title)

	case events.TypeImportFailed:
		title := str("movie_title")
		reason := str("reason")
		if reason != "" {
			return CategoryImportFailed, fmt.Sprintf("Import failed for %s: %s", title, reason)
		}
		return CategoryImportFailed, fmt.Sprintf("Import failed for %s", title)

	case events.TypeMovieAdded:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Added %s to library", title)

	case events.TypeMovieDeleted:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Deleted %s", title)

	case events.TypeMovieUpdated:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Updated %s", title)

	case events.TypeTaskStarted:
		task := str("task")
		return CategoryTask, fmt.Sprintf("%s started", task)

	case events.TypeTaskFinished:
		task := str("task")
		return CategoryTask, fmt.Sprintf("%s completed", task)

	case events.TypeHealthIssue:
		check := str("check")
		message := str("message")
		return CategoryHealth, fmt.Sprintf("%s: %s", check, message)

	case events.TypeHealthOK:
		check := str("check")
		return CategoryHealth, fmt.Sprintf("%s: recovered", check)

	case events.TypeBulkSearchComplete:
		return CategoryGrabSucceeded, "Bulk search completed"

	default:
		return "", ""
	}
}

// List returns activity entries matching the given filters.
func (s *Service) List(ctx context.Context, category *string, since *string, limit int64) (*ListResult, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	catFilter := dbutil.NullString(category)
	sinceFilter := dbutil.NullString(since)

	rows, err := s.q.ListActivities(ctx, dbgen.ListActivitiesParams{
		Category: catFilter,
		Since:    sinceFilter,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("listing activities: %w", err)
	}

	total, err := s.q.CountActivities(ctx, dbgen.CountActivitiesParams{
		Category: catFilter,
		Since:    sinceFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("counting activities: %w", err)
	}

	activities := make([]Activity, 0, len(rows))
	for _, r := range rows {
		a := Activity{
			ID:        r.ID,
			Type:      r.Type,
			Category:  r.Category,
			MovieID:   dbutil.NullStringPtr(r.MovieID),
			Title:     r.Title,
			CreatedAt: r.CreatedAt,
		}
		if r.Detail.Valid {
			_ = json.Unmarshal([]byte(r.Detail.String), &a.Detail)
		}
		activities = append(activities, a)
	}

	return &ListResult{
		Activities: activities,
		Total:      total,
	}, nil
}

// Prune deletes activity entries older than the given duration.
func (s *Service) Prune(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	return s.q.PruneActivities(ctx, cutoff)
}

// AttentionItem is one entry in the "Needs attention" rail. Each item is
// either a failed or removed grab (from grab_history.download_status) or
// a failed import (from activity_log.category=import_failed).
type AttentionItem struct {
	Kind         string `json:"kind"` // "grab_failed" | "import_failed"
	GrabID       string `json:"grab_id,omitempty"`
	MovieID      string `json:"movie_id,omitempty"`
	ReleaseTitle string `json:"release_title"`
	Detail       string `json:"detail,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// AttentionResult is the response shape for /api/v1/activity/needs-attention.
type AttentionResult struct {
	Items []AttentionItem `json:"items"`
	// Counts breaks the items down by kind so the UI can show a summary
	// without re-counting client-side.
	Counts struct {
		GrabFailed   int `json:"grab_failed"`
		ImportFailed int `json:"import_failed"`
	} `json:"counts"`
}

// NeedsAttention returns recent failures for the Activity-page "Needs
// attention" rail. window controls how far back to look; perKind caps
// each bucket so a flood of one type doesn't drown out the others.
//
// Mirrors Pilot's NeedsAttention shape (see pilot/internal/core/activity
// /service.go) — minus the "stalled" bucket, which Prism doesn't have a
// stallwatcher for. Add stalled here if/when one ships.
func (s *Service) NeedsAttention(ctx context.Context, window time.Duration, perKind int) (*AttentionResult, error) {
	if window <= 0 {
		window = 48 * time.Hour
	}
	if perKind <= 0 || perKind > 200 {
		perKind = 50
	}
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)

	out := &AttentionResult{Items: []AttentionItem{}}

	// Failed grabs from grab_history (download client rejected, mid-
	// flight failure, or user-removed).
	for _, status := range []string{"failed", "removed"} {
		rows, err := s.q.ListGrabHistoryByStatusSince(ctx, dbgen.ListGrabHistoryByStatusSinceParams{
			Status: status,
			Since:  since,
			Limit:  int32(perKind),
		})
		if err != nil {
			return nil, fmt.Errorf("listing grab history (%s): %w", status, err)
		}
		for _, r := range rows {
			out.Items = append(out.Items, AttentionItem{
				Kind:         "grab_failed",
				GrabID:       r.ID,
				MovieID:      r.MovieID,
				ReleaseTitle: r.ReleaseTitle,
				Detail:       fmt.Sprintf("Download %s", r.DownloadStatus),
				CreatedAt:    r.GrabbedAt,
			})
			out.Counts.GrabFailed++
		}
	}

	// Failed imports from the activity_log split-category vocabulary.
	cat := dbutil.NullString(stringPtr(string(CategoryImportFailed)))
	sinceNS := dbutil.NullString(&since)
	imports, err := s.q.ListActivities(ctx, dbgen.ListActivitiesParams{
		Category: cat,
		Since:    sinceNS,
		Limit:    int32(perKind),
	})
	if err != nil {
		return nil, fmt.Errorf("listing import failures: %w", err)
	}
	for _, r := range imports {
		var detail string
		if r.Detail.Valid {
			var d map[string]any
			if json.Unmarshal([]byte(r.Detail.String), &d) == nil {
				if v, ok := d["reason"].(string); ok {
					detail = v
				}
			}
		}
		movieID := ""
		if r.MovieID.Valid {
			movieID = r.MovieID.String
		}
		out.Items = append(out.Items, AttentionItem{
			Kind:         "import_failed",
			MovieID:      movieID,
			ReleaseTitle: r.Title,
			Detail:       detail,
			CreatedAt:    r.CreatedAt,
		})
		out.Counts.ImportFailed++
	}

	return out, nil
}

// stringPtr is a tiny helper for embedding string-literal pointers
// inline. Prism uses *string heavily as a "nullable filter" pattern.
func stringPtr(s string) *string { return &s }
