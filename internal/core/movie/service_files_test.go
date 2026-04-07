package movie_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/testutil"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// newFileTestService returns a Service and seeded movieID using the same testutil DB.
// A real file at tmpPath is attached to the movie.
func newFileTestService(t *testing.T, tmpPath string) (*movie.Service, string) {
	t.Helper()
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	seeded := testutil.SeedMovie(t, q)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// Write a real file so disk-removal tests work.
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(tmpPath, []byte("fake video"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := svc.AttachFile(ctx, seeded.ID, tmpPath, int64(len("fake video")), plugin.Quality{Resolution: "1080p"}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	return svc, seeded.ID
}

func TestDeleteFile_DBOnly(t *testing.T) {
	ctx := context.Background()
	tmpPath := filepath.Join(t.TempDir(), "movie.mkv")
	svc, movieID := newFileTestService(t, tmpPath)

	files, err := svc.ListFiles(ctx, movieID)
	if err != nil || len(files) == 0 {
		t.Fatalf("ListFiles: err=%v len=%d", err, len(files))
	}

	if err := svc.DeleteFile(ctx, files[0].ID, false /* DB only */); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	// File still exists on disk.
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Errorf("file should still exist on disk after DB-only delete: %v", statErr)
	}

	// DB record is gone.
	remaining, err := svc.ListFiles(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 files after delete, got %d", len(remaining))
	}
}

func TestDeleteFile_FromDisk(t *testing.T) {
	ctx := context.Background()
	tmpPath := filepath.Join(t.TempDir(), "movie.mkv")
	svc, movieID := newFileTestService(t, tmpPath)

	files, err := svc.ListFiles(ctx, movieID)
	if err != nil || len(files) == 0 {
		t.Fatalf("ListFiles: err=%v len=%d", err, len(files))
	}

	if err := svc.DeleteFile(ctx, files[0].ID, true /* delete from disk */); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	// File must be gone from disk.
	if _, statErr := os.Stat(tmpPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("expected file to be removed from disk, got: %v", statErr)
	}
}

func TestDeleteFile_ResetMovieStatus(t *testing.T) {
	ctx := context.Background()
	tmpPath := filepath.Join(t.TempDir(), "movie.mkv")
	svc, movieID := newFileTestService(t, tmpPath)

	// Before delete: movie should be "downloaded".
	m, err := svc.Get(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "downloaded" {
		t.Fatalf("expected status=downloaded before delete, got %q", m.Status)
	}

	files, _ := svc.ListFiles(ctx, movieID)
	if err := svc.DeleteFile(ctx, files[0].ID, false); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	// After delete: status should be "wanted", path cleared.
	m, err = svc.Get(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "wanted" {
		t.Errorf("expected status=wanted after last file deleted, got %q", m.Status)
	}
	if m.Path != "" {
		t.Errorf("expected path cleared after last file deleted, got %q", m.Path)
	}
}

func TestDeleteFile_MultipleFiles_StatusPreserved(t *testing.T) {
	ctx := context.Background()
	tmp1 := filepath.Join(t.TempDir(), "movie1.mkv")
	svc, movieID := newFileTestService(t, tmp1)

	// Attach a second file.
	tmp2 := filepath.Join(t.TempDir(), "movie2.mkv")
	if err := os.WriteFile(tmp2, []byte("fake2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.AttachFile(ctx, movieID, tmp2, 5, plugin.Quality{}); err != nil {
		t.Fatalf("AttachFile second: %v", err)
	}

	files, err := svc.ListFiles(ctx, movieID)
	if err != nil || len(files) != 2 {
		t.Fatalf("expected 2 files, got %d (err=%v)", len(files), err)
	}

	// Delete only one.
	if err := svc.DeleteFile(ctx, files[0].ID, false); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	// Movie status should still be "downloaded" (one file remains).
	m, err := svc.Get(ctx, movieID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "downloaded" {
		t.Errorf("expected status=downloaded when files remain, got %q", m.Status)
	}
}

func TestDeleteFile_NotFound(t *testing.T) {
	ctx := context.Background()
	tmpPath := filepath.Join(t.TempDir(), "movie.mkv")
	svc, _ := newFileTestService(t, tmpPath)

	err := svc.DeleteFile(ctx, "nonexistent-id", false)
	if !errors.Is(err, movie.ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}
