package mediainfo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	dbsqlite "github.com/beacon-stack/prism/internal/db/generated/sqlite"
)

// Service provides mediainfo scanning backed by the database.
type Service struct {
	scanner *Scanner
	q       dbsqlite.Querier
	logger  *slog.Logger
}

// NewService creates a Service. scanner may be a disabled Scanner (Available()
// returns false); in that case ScanFile and ScanAll are no-ops.
func NewService(scanner *Scanner, q dbsqlite.Querier, logger *slog.Logger) *Service {
	return &Service{scanner: scanner, q: q, logger: logger}
}

// Available reports whether the underlying scanner is operational.
func (s *Service) Available() bool {
	return s.scanner.Available()
}

// FFprobeVersion returns the resolved ffprobe path for display, or "".
func (s *Service) FFprobeVersion() string {
	return s.scanner.FFprobePath()
}

// ScanFile scans one file by its database ID and file path, updating the
// movie_files row with the resulting mediainfo JSON.
func (s *Service) ScanFile(ctx context.Context, fileID, filePath string) error {
	if !s.scanner.Available() {
		return nil
	}

	result, err := s.scanner.Scan(ctx, filePath)
	if err != nil {
		return fmt.Errorf("scanning %q: %w", filePath, err)
	}

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling mediainfo: %w", err)
	}

	now := time.Now().UTC()
	return s.q.UpdateMovieFileMediainfo(ctx, dbsqlite.UpdateMovieFileMediainfoParams{
		MediainfoJson:      string(b),
		MediainfoScannedAt: &now,
		ID:                 fileID,
	})
}

// ScanAll scans every movie_file where mediainfo_json is empty. It returns
// the number of files successfully scanned. Intended for on-demand "scan all"
// from the settings page.
func (s *Service) ScanAll(ctx context.Context) (int, error) {
	if !s.scanner.Available() {
		return 0, nil
	}

	rows, err := s.q.ListUnscannedMovieFiles(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing unscanned files: %w", err)
	}

	var count int
	for _, row := range rows {
		if err := s.ScanFile(ctx, row.ID, row.Path); err != nil {
			s.logger.Debug("mediainfo scan failed", "file_id", row.ID, "path", row.Path, "error", err)
			continue
		}
		count++
	}

	return count, nil
}
