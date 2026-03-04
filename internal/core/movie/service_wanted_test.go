package movie_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/testutil"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func newWantedTestService(t *testing.T) *movie.Service {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	return movie.NewService(q, nil, bus, logger)
}

// TestListMissing verifies that a monitored movie without a file is returned,
// and a movie with a file is excluded.
func TestListMissing(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// Movie with no file — should appear.
	missing := testutil.SeedMovie(t, q, testutil.WithTMDBID(100))

	// Movie with a file — should not appear.
	withFile := testutil.SeedMovie(t, q, testutil.WithTMDBID(101))
	if err := svc.AttachFile(ctx, withFile.ID, "/movies/test.mkv", 1_000_000_000, plugin.Quality{Resolution: "1080p"}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	movies, total, err := svc.ListMissing(ctx, 1, 25)
	if err != nil {
		t.Fatalf("ListMissing: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
	if movies[0].ID != missing.ID {
		t.Errorf("expected movie ID %q, got %q", missing.ID, movies[0].ID)
	}
}

// TestListMissing_OnlyMonitored verifies that unmonitored movies without files
// are excluded from the missing list.
func TestListMissing_OnlyMonitored(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// Unmonitored movie without a file — must NOT appear.
	_ = testutil.SeedMovie(t, q, testutil.WithMonitored(false))

	movies, total, err := svc.ListMissing(ctx, 1, 25)
	if err != nil {
		t.Fatalf("ListMissing: %v", err)
	}
	if total != 0 || len(movies) != 0 {
		t.Errorf("expected 0 movies, got total=%d len=%d", total, len(movies))
	}
}

// TestListMissing_Pagination checks that page/perPage correctly slices the
// result set.
func TestListMissing_Pagination(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// Seed 3 monitored movies with no files.
	testutil.SeedMovie(t, q, testutil.WithTMDBID(200))
	testutil.SeedMovie(t, q, testutil.WithTMDBID(201))
	testutil.SeedMovie(t, q, testutil.WithTMDBID(202))

	movies, total, err := svc.ListMissing(ctx, 1, 2)
	if err != nil {
		t.Fatalf("ListMissing page 1: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(movies) != 2 {
		t.Errorf("expected 2 movies on page 1, got %d", len(movies))
	}

	page2, _, err := svc.ListMissing(ctx, 2, 2)
	if err != nil {
		t.Fatalf("ListMissing page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 movie on page 2, got %d", len(page2))
	}
}

// TestListCutoffUnmet verifies that a movie with a file below the quality
// profile cutoff appears, while a movie meeting the cutoff does not.
func TestListCutoffUnmet(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// SeedQualityProfile cutoff = 1080p Bluray x264.
	// Movie A: 720p file — below cutoff, should appear.
	movieA := testutil.SeedMovie(t, q, testutil.WithTMDBID(300))
	if err := svc.AttachFile(ctx, movieA.ID, "/movies/a.mkv", 500_000_000, plugin.Quality{
		Resolution: "720p",
		Source:     "bluray",
	}); err != nil {
		t.Fatalf("AttachFile movieA: %v", err)
	}

	// Movie B: 1080p Bluray x264 file — meets cutoff, should NOT appear.
	// (Cutoff from SeedQualityProfile includes codec x264, so we must match it.)
	movieB := testutil.SeedMovie(t, q, testutil.WithTMDBID(301))
	if err := svc.AttachFile(ctx, movieB.ID, "/movies/b.mkv", 8_000_000_000, plugin.Quality{
		Resolution: "1080p",
		Source:     "bluray",
		Codec:      "x264",
	}); err != nil {
		t.Fatalf("AttachFile movieB: %v", err)
	}

	movies, err := svc.ListCutoffUnmet(ctx)
	if err != nil {
		t.Fatalf("ListCutoffUnmet: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 cutoff-unmet movie, got %d", len(movies))
	}
	if movies[0].ID != movieA.ID {
		t.Errorf("expected movie A (%q) to be cutoff-unmet, got %q", movieA.ID, movies[0].ID)
	}
}

// TestListCutoffUnmet_MultipleFiles verifies that the best file quality is
// used when a movie has more than one file.
func TestListCutoffUnmet_MultipleFiles(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	// Movie starts with a 720p file (below 1080p cutoff).
	m := testutil.SeedMovie(t, q)
	if err := svc.AttachFile(ctx, m.ID, "/movies/low.mkv", 500_000_000, plugin.Quality{
		Resolution: "720p",
		Source:     "bluray",
	}); err != nil {
		t.Fatalf("AttachFile low: %v", err)
	}

	// Add a second, 1080p x264 file — best quality now meets cutoff.
	if err := svc.AttachFile(ctx, m.ID, "/movies/high.mkv", 8_000_000_000, plugin.Quality{
		Resolution: "1080p",
		Source:     "bluray",
		Codec:      "x264",
	}); err != nil {
		t.Fatalf("AttachFile high: %v", err)
	}

	movies, err := svc.ListCutoffUnmet(ctx)
	if err != nil {
		t.Fatalf("ListCutoffUnmet: %v", err)
	}
	if len(movies) != 0 {
		t.Errorf("expected 0 cutoff-unmet movies (best file meets cutoff), got %d", len(movies))
	}
}

// TestListCutoffUnmet_UnmonitoredExcluded verifies unmonitored movies are not
// returned even if their file quality is below the cutoff.
func TestListCutoffUnmet_UnmonitoredExcluded(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	svc := movie.NewService(q, nil, bus, logger)

	m := testutil.SeedMovie(t, q, testutil.WithMonitored(false))
	if err := svc.AttachFile(ctx, m.ID, "/movies/low.mkv", 500_000_000, plugin.Quality{
		Resolution: "720p",
		Source:     "bluray",
	}); err != nil {
		t.Fatalf("AttachFile: %v", err)
	}

	movies, err := svc.ListCutoffUnmet(ctx)
	if err != nil {
		t.Fatalf("ListCutoffUnmet: %v", err)
	}
	if len(movies) != 0 {
		t.Errorf("expected 0 cutoff-unmet movies for unmonitored movie, got %d", len(movies))
	}
}

// Ensure the helper is referenced (compile check).
var _ = newWantedTestService
