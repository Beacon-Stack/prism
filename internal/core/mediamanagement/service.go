// Package mediamanagement provides access to global media management settings
// (movie naming formats, colon replacement, extra file importing, etc.).
package mediamanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/beacon-media/prism/internal/core/dbutil"
	dbsqlite "github.com/beacon-media/prism/internal/db/generated/sqlite"
)

// Settings is the application-level view of the media_management table.
type Settings struct {
	RenameMovies           bool
	StandardMovieFormat    string
	MovieFolderFormat      string
	ColonReplacement       string // "delete" | "dash" | "space-dash" | "smart"
	ImportExtraFiles       bool
	ExtraFileExtensions    []string // parsed from comma-separated DB string
	UnmonitorDeletedMovies bool
}

// Service exposes read/write access to the single media_management row.
type Service struct {
	q dbsqlite.Querier
}

// NewService creates a new Service backed by the given Querier.
func NewService(q dbsqlite.Querier) *Service {
	return &Service{q: q}
}

// Get returns the current media management settings.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	row, err := s.q.GetMediaManagement(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("media_management: get: %w", err)
	}
	return fromRow(row), nil
}

// Update persists new settings and returns the saved values.
func (s *Service) Update(ctx context.Context, settings Settings) (Settings, error) {
	row, err := s.q.UpdateMediaManagement(ctx, dbsqlite.UpdateMediaManagementParams{
		RenameMovies:           dbutil.BoolToInt(settings.RenameMovies),
		StandardMovieFormat:    settings.StandardMovieFormat,
		MovieFolderFormat:      settings.MovieFolderFormat,
		ColonReplacement:       settings.ColonReplacement,
		ImportExtraFiles:       dbutil.BoolToInt(settings.ImportExtraFiles),
		ExtraFileExtensions:    strings.Join(settings.ExtraFileExtensions, ","),
		UnmonitorDeletedMovies: dbutil.BoolToInt(settings.UnmonitorDeletedMovies),
	})
	if err != nil {
		return Settings{}, fmt.Errorf("media_management: update: %w", err)
	}
	return fromRow(row), nil
}

// fromRow converts a DB row to a Settings value.
func fromRow(row dbsqlite.MediaManagement) Settings {
	return Settings{
		RenameMovies:           row.RenameMovies != 0,
		StandardMovieFormat:    row.StandardMovieFormat,
		MovieFolderFormat:      row.MovieFolderFormat,
		ColonReplacement:       row.ColonReplacement,
		ImportExtraFiles:       row.ImportExtraFiles != 0,
		ExtraFileExtensions:    parseExtensions(row.ExtraFileExtensions),
		UnmonitorDeletedMovies: row.UnmonitorDeletedMovies != 0,
	}
}

// parseExtensions splits a comma-separated extension string, trims whitespace,
// and drops empty tokens.
func parseExtensions(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
