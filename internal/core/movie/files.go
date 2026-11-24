package movie

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/dbutil"
	"github.com/beacon-stack/prism/internal/core/edition"
	"github.com/beacon-stack/prism/internal/core/renamer"
	"github.com/beacon-stack/prism/internal/db"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// RenamePreviewItem describes a single proposed or completed file rename.
type RenamePreviewItem struct {
	FileID  string
	OldPath string
	NewPath string
}

// RenameSettings carries the naming format options used by RenameFiles.
type RenameSettings struct {
	Format           string
	ColonReplacement renamer.ColonReplacement
}

// SetRenameFunc replaces the function used to rename files on disk. Intended
// for use in tests to inject a mock without touching the real filesystem.
func (s *Service) SetRenameFunc(fn func(oldPath, newPath string) error) {
	s.renameFile = fn
}

// FileInfo is the domain representation of a movie_files record.
type FileInfo struct {
	ID            string
	MovieID       string
	Path          string
	SizeBytes     int64
	Quality       plugin.Quality
	Edition       string
	ImportedAt    time.Time
	IndexedAt     time.Time
	MediainfoJSON string // raw JSON from ffprobe scan; empty if not yet scanned
}

// ListFiles returns all file records for a movie, newest import first.
func (s *Service) ListFiles(ctx context.Context, movieID string) ([]FileInfo, error) {
	rows, err := s.q.ListMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("listing files for movie %q: %w", movieID, err)
	}
	files := make([]FileInfo, 0, len(rows))
	for _, row := range rows {
		fi, err := rowToFileInfo(row)
		if err != nil {
			return nil, err
		}
		files = append(files, fi)
	}
	return files, nil
}

// GetFile returns the FileInfo for a single movie_file record.
func (s *Service) GetFile(ctx context.Context, fileID string) (FileInfo, error) {
	row, err := s.q.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FileInfo{}, ErrFileNotFound
		}
		return FileInfo{}, fmt.Errorf("fetching movie file %q: %w", fileID, err)
	}
	return rowToFileInfo(row)
}

// DeleteFile removes the movie_files record identified by fileID.
// If deleteFromDisk is true, the file at the stored path is also removed from
// disk (errors are logged but do not fail the operation).
// After the DB record is removed, if no files remain for the movie, the movie's
// path is cleared and its status is reset to "wanted".
func (s *Service) DeleteFile(ctx context.Context, fileID string, deleteFromDisk bool) error {
	row, err := s.q.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFileNotFound
		}
		return fmt.Errorf("fetching movie file %q: %w", fileID, err)
	}

	if deleteFromDisk {
		if removeErr := os.Remove(row.Path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.logger.WarnContext(ctx, "failed to delete movie file from disk",
				slog.String("path", row.Path),
				slog.Any("error", removeErr),
			)
		}
	}

	do := func(q dbgen.Querier) error {
		if err := q.DeleteMovieFile(ctx, fileID); err != nil {
			return fmt.Errorf("deleting movie file record %q: %w", fileID, err)
		}

		remaining, err := q.ListMovieFiles(ctx, row.MovieID)
		if err != nil {
			return fmt.Errorf("listing remaining files for movie %q: %w", row.MovieID, err)
		}
		if len(remaining) == 0 {
			now := time.Now().UTC().Format(time.RFC3339)
			if _, err := q.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{
				Path:      nil,
				UpdatedAt: now,
				ID:        row.MovieID,
			}); err != nil {
				return fmt.Errorf("clearing path for movie %q: %w", row.MovieID, err)
			}
			if _, err := q.UpdateMovieStatus(ctx, dbgen.UpdateMovieStatusParams{
				Status:    "wanted",
				UpdatedAt: now,
				ID:        row.MovieID,
			}); err != nil {
				return fmt.Errorf("resetting status for movie %q: %w", row.MovieID, err)
			}
		}
		return nil
	}

	if s.sqlDB != nil {
		return db.RunInTx(ctx, s.sqlDB, do)
	}
	return do(s.q)
}

// AttachFile links a file on disk to an existing movie record. It sets
// movies.path to the full file path, creates a movie_file record,
// and marks the movie status as "downloaded".
//
// Callers are responsible for ensuring the file actually exists on disk before
// calling this method.
func (s *Service) AttachFile(ctx context.Context, movieID, filePath string, sizeBytes int64, quality plugin.Quality) error {
	qualityJSON, err := json.Marshal(quality)
	if err != nil {
		return fmt.Errorf("marshaling quality: %w", err)
	}

	// Auto-detect edition from the filename.
	var editionNS *string
	if ed := edition.Parse(filepath.Base(filePath)); ed != nil {
		editionNS = ptrString(ed.Name)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	do := func(q dbgen.Querier) error {
		if _, err := q.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{
			Path:      ptrString(filePath),
			UpdatedAt: now,
			ID:        movieID,
		}); err != nil {
			return fmt.Errorf("updating movie path for %q: %w", movieID, err)
		}

		if _, err := q.CreateMovieFile(ctx, dbgen.CreateMovieFileParams{
			ID:          uuid.New().String(),
			MovieID:     movieID,
			Path:        filePath,
			SizeBytes:   sizeBytes,
			QualityJson: string(qualityJSON),
			Edition:     editionNS,
			ImportedAt:  now,
			IndexedAt:   now,
		}); err != nil {
			return fmt.Errorf("creating movie file for %q: %w", movieID, err)
		}

		if _, err := q.UpdateMovieStatus(ctx, dbgen.UpdateMovieStatusParams{
			Status:    "downloaded",
			UpdatedAt: now,
			ID:        movieID,
		}); err != nil {
			return fmt.Errorf("updating movie status for %q: %w", movieID, err)
		}
		return nil
	}

	if s.sqlDB != nil {
		return db.RunInTx(ctx, s.sqlDB, do)
	}
	return do(s.q)
}

// RenameFiles computes the standard-format destination filename for every file
// belonging to movieID using the supplied settings. Files whose current name
// already matches the computed name are skipped.
//
// If dryRun is true, no disk or DB operations are performed; the returned slice
// describes what would happen. If dryRun is false, each file is renamed on disk
// and the DB record is updated. Errors for individual files are logged and
// skipped; a combined error is returned if any file could not be renamed.
//
// Returns ErrNotFound if the movie does not exist.
func (s *Service) RenameFiles(ctx context.Context, movieID string, settings RenameSettings, dryRun bool) ([]RenamePreviewItem, error) {
	movie, err := s.q.GetMovie(ctx, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("fetching movie %q: %w", movieID, err)
	}

	files, err := s.q.ListMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("listing files for movie %q: %w", movieID, err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	format := settings.Format
	if format == "" {
		format = renamer.DefaultFileFormat
	}
	colon := settings.ColonReplacement
	if colon == "" {
		colon = renamer.ColonDelete
	}

	rm := renamer.Movie{
		Title:         movie.Title,
		OriginalTitle: movie.OriginalTitle,
		Year:          int(movie.Year),
	}

	var items []RenamePreviewItem
	for _, f := range files {
		var qual plugin.Quality
		_ = json.Unmarshal([]byte(f.QualityJson), &qual)

		// Set per-file edition so {Edition} renders correctly for each file.
		fileRM := rm
		if f.Edition != nil {
			fileRM.Edition = *f.Edition
		}

		ext := filepath.Ext(f.Path)
		newFilename := renamer.ApplyWithOptions(format, fileRM, qual, colon) + ext
		newPath := filepath.Join(filepath.Dir(f.Path), newFilename)

		if newPath == f.Path {
			continue // already correctly named
		}
		items = append(items, RenamePreviewItem{
			FileID:  f.ID,
			OldPath: f.Path,
			NewPath: newPath,
		})
	}

	if dryRun || len(items) == 0 {
		return items, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var errs []string
	var done []RenamePreviewItem

	for _, item := range items {
		if _, statErr := os.Stat(item.NewPath); statErr == nil {
			// Target already exists — skip to avoid clobbering.
			s.logger.WarnContext(ctx, "rename skipped: target path already exists",
				slog.String("old", item.OldPath),
				slog.String("new", item.NewPath),
			)
			errs = append(errs, fmt.Sprintf("target exists: %s", item.NewPath))
			continue
		}

		if renameErr := s.renameFile(item.OldPath, item.NewPath); renameErr != nil {
			s.logger.ErrorContext(ctx, "rename failed",
				slog.String("old", item.OldPath),
				slog.String("new", item.NewPath),
				slog.Any("error", renameErr),
			)
			errs = append(errs, renameErr.Error())
			continue
		}

		if dbErr := s.q.UpdateMovieFilePath(ctx, dbgen.UpdateMovieFilePathParams{
			Path: item.NewPath,
			ID:   item.FileID,
		}); dbErr != nil {
			s.logger.ErrorContext(ctx, "failed to update movie_files path after rename",
				slog.String("file_id", item.FileID),
				slog.Any("error", dbErr),
			)
			errs = append(errs, dbErr.Error())
			continue
		}

		// Keep movies.path in sync if this file's old path matches it.
		if movie.Path != nil && *movie.Path == item.OldPath {
			if _, pathErr := s.q.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{
				Path:      ptrString(item.NewPath),
				UpdatedAt: now,
				ID:        movieID,
			}); pathErr != nil {
				s.logger.WarnContext(ctx, "failed to sync movies.path after rename",
					slog.String("movie_id", movieID),
					slog.Any("error", pathErr),
				)
			}
		}

		done = append(done, item)
	}

	if len(errs) > 0 {
		return done, fmt.Errorf("rename: %d file(s) failed: %s", len(errs), strings.Join(errs, "; "))
	}
	return done, nil
}

// rowToFileInfo converts a DB movie_files row into the FileInfo domain type.
func rowToFileInfo(row dbgen.MovieFile) (FileInfo, error) {
	var qual plugin.Quality
	if err := json.Unmarshal([]byte(row.QualityJson), &qual); err != nil {
		// Non-fatal: return zero Quality if JSON is malformed.
		qual = plugin.Quality{}
	}

	importedAt, err := time.Parse(time.RFC3339, row.ImportedAt)
	if err != nil {
		importedAt = time.Time{}
	}
	indexedAt, err := time.Parse(time.RFC3339, row.IndexedAt)
	if err != nil {
		indexedAt = time.Time{}
	}

	edition := dbutil.NullStringValue(row.Edition)

	return FileInfo{
		ID:            row.ID,
		MovieID:       row.MovieID,
		Path:          row.Path,
		SizeBytes:     row.SizeBytes,
		Quality:       qual,
		Edition:       edition,
		ImportedAt:    importedAt,
		IndexedAt:     indexedAt,
		MediainfoJSON: row.MediainfoJson,
	}, nil
}
