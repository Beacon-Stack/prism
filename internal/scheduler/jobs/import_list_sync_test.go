package jobs

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/beacon-media/prism/internal/core/importlist"
	"github.com/beacon-media/prism/internal/core/movie"
	"github.com/beacon-media/prism/internal/events"
	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/internal/testutil"
)

func TestImportListSync_JobMetadata(t *testing.T) {
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	reg := registry.New()

	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := importlist.NewService(q, reg, movieSvc, nil, nil, nil, logger)

	job := ImportListSync(svc, logger)

	if job.Name != "import_list_sync" {
		t.Errorf("Name = %q, want %q", job.Name, "import_list_sync")
	}
	if job.Interval != 6*time.Hour {
		t.Errorf("Interval = %v, want %v", job.Interval, 6*time.Hour)
	}
	if job.Fn == nil {
		t.Fatal("Fn is nil")
	}
}

func TestImportListSync_RunsWithoutPanic(t *testing.T) {
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	reg := registry.New()

	movieSvc := movie.NewService(q, nil, bus, logger)
	svc := importlist.NewService(q, reg, movieSvc, nil, nil, nil, logger)

	job := ImportListSync(svc, logger)

	// Running the job function should not panic even with no configured lists.
	job.Fn(context.Background())
}
