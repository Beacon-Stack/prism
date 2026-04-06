// Package downloadhandling provides access to download handling settings
// (completed/failed download automation, check interval, remote path mappings).
package downloadhandling

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-media/prism/internal/core/dbutil"
	dbsqlite "github.com/beacon-media/prism/internal/db/generated/sqlite"
)

// Settings is the application-level view of the download_handling table.
type Settings struct {
	EnableCompleted             bool
	CheckIntervalMinutes        int64
	RedownloadFailed            bool
	RedownloadFailedInteractive bool
}

// RemotePathMapping is the application-level view of a remote_path_mappings row.
type RemotePathMapping struct {
	ID         string
	Host       string
	RemotePath string
	LocalPath  string
}

// Service exposes read/write access to download handling settings and
// remote path mapping CRUD.
type Service struct {
	q dbsqlite.Querier
}

// NewService creates a new Service backed by the given Querier.
func NewService(q dbsqlite.Querier) *Service {
	return &Service{q: q}
}

// Get returns the current download handling settings.
func (s *Service) Get(ctx context.Context) (Settings, error) {
	row, err := s.q.GetDownloadHandling(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("download_handling: get: %w", err)
	}
	return fromRow(row), nil
}

// Update persists new settings and returns the saved values.
func (s *Service) Update(ctx context.Context, settings Settings) (Settings, error) {
	row, err := s.q.UpdateDownloadHandling(ctx, dbsqlite.UpdateDownloadHandlingParams{
		EnableCompleted:             dbutil.BoolToInt(settings.EnableCompleted),
		CheckIntervalMinutes:        settings.CheckIntervalMinutes,
		RedownloadFailed:            dbutil.BoolToInt(settings.RedownloadFailed),
		RedownloadFailedInteractive: dbutil.BoolToInt(settings.RedownloadFailedInteractive),
	})
	if err != nil {
		return Settings{}, fmt.Errorf("download_handling: update: %w", err)
	}
	return fromRow(row), nil
}

// CheckInterval returns the polling interval as a time.Duration.
// Falls back to 60 seconds if the stored value is <= 0.
func (s *Service) CheckInterval(ctx context.Context) (time.Duration, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return 60 * time.Second, err
	}
	if settings.CheckIntervalMinutes <= 0 {
		return 60 * time.Second, nil
	}
	return time.Duration(settings.CheckIntervalMinutes) * time.Minute, nil
}

// ListRemotePathMappings returns all configured remote path mappings.
func (s *Service) ListRemotePathMappings(ctx context.Context) ([]RemotePathMapping, error) {
	rows, err := s.q.ListRemotePathMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("download_handling: list remote path mappings: %w", err)
	}
	out := make([]RemotePathMapping, len(rows))
	for i, r := range rows {
		out[i] = mappingFromRow(r)
	}
	return out, nil
}

// CreateRemotePathMapping inserts a new remote path mapping and returns it.
func (s *Service) CreateRemotePathMapping(ctx context.Context, host, remotePath, localPath string) (RemotePathMapping, error) {
	row, err := s.q.CreateRemotePathMapping(ctx, dbsqlite.CreateRemotePathMappingParams{
		ID:         uuid.New().String(),
		Host:       host,
		RemotePath: remotePath,
		LocalPath:  localPath,
	})
	if err != nil {
		return RemotePathMapping{}, fmt.Errorf("download_handling: create remote path mapping: %w", err)
	}
	return mappingFromRow(row), nil
}

// DeleteRemotePathMapping removes a remote path mapping by ID.
func (s *Service) DeleteRemotePathMapping(ctx context.Context, id string) error {
	if err := s.q.DeleteRemotePathMapping(ctx, id); err != nil {
		return fmt.Errorf("download_handling: delete remote path mapping %q: %w", id, err)
	}
	return nil
}

// fromRow converts a DB row to a Settings value.
func fromRow(row dbsqlite.DownloadHandling) Settings {
	return Settings{
		EnableCompleted:             row.EnableCompleted != 0,
		CheckIntervalMinutes:        row.CheckIntervalMinutes,
		RedownloadFailed:            row.RedownloadFailed != 0,
		RedownloadFailedInteractive: row.RedownloadFailedInteractive != 0,
	}
}

// mappingFromRow converts a DB row to a RemotePathMapping value.
func mappingFromRow(row dbsqlite.RemotePathMapping) RemotePathMapping {
	return RemotePathMapping{
		ID:         row.ID,
		Host:       row.Host,
		RemotePath: row.RemotePath,
		LocalPath:  row.LocalPath,
	}
}
