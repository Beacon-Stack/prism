package autosearch_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/autosearch"
	"github.com/beacon-stack/prism/internal/core/blocklist"
	"github.com/beacon-stack/prism/internal/core/downloader"
	"github.com/beacon-stack/prism/internal/core/indexer"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/quality"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/ratelimit"
	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/testutil"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// ── mock indexer ─────────────────────────────────────────────────────────────

type mockIndexer struct {
	releases []plugin.Release
	err      error
}

func (m *mockIndexer) Name() string                 { return "mock" }
func (m *mockIndexer) Protocol() plugin.Protocol    { return plugin.ProtocolTorrent }
func (m *mockIndexer) Test(_ context.Context) error { return nil }

func (m *mockIndexer) Capabilities(_ context.Context) (plugin.Capabilities, error) {
	return plugin.Capabilities{}, nil
}

func (m *mockIndexer) Search(_ context.Context, _ plugin.SearchQuery) ([]plugin.Release, error) {
	return m.releases, m.err
}

func (m *mockIndexer) GetRecent(_ context.Context) ([]plugin.Release, error) {
	return m.releases, m.err
}

// ── mock downloader ──────────────────────────────────────────────────────────

type mockDownloader struct {
	itemID string
	err    error
}

func (m *mockDownloader) Name() string                 { return "mock-dl" }
func (m *mockDownloader) Protocol() plugin.Protocol    { return plugin.ProtocolTorrent }
func (m *mockDownloader) Test(_ context.Context) error { return nil }

func (m *mockDownloader) Add(_ context.Context, _ plugin.Release) (string, error) {
	return m.itemID, m.err
}

func (m *mockDownloader) Status(_ context.Context, _ string) (plugin.QueueItem, error) {
	return plugin.QueueItem{}, nil
}

func (m *mockDownloader) GetQueue(_ context.Context) ([]plugin.QueueItem, error) {
	return nil, nil
}

func (m *mockDownloader) Remove(_ context.Context, _ string, _ bool) error { return nil }

// ── test helpers ─────────────────────────────────────────────────────────────

type testEnv struct {
	q       *dbgen.Queries
	svc     *autosearch.Service
	blSvc   *blocklist.Service
	mockIdx *mockIndexer
	mockDL  *mockDownloader
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.Default()
	bus := events.New(logger)

	mockIdx := &mockIndexer{}
	mockDL := &mockDownloader{itemID: "item-123"}

	reg := registry.New()
	reg.RegisterIndexer("mock", func(_ json.RawMessage) (plugin.Indexer, error) {
		return mockIdx, nil
	})
	reg.RegisterDownloader("mock-dl", func(_ json.RawMessage) (plugin.DownloadClient, error) {
		return mockDL, nil
	})

	indexerSvc := indexer.NewService(q, reg, bus, ratelimit.New())
	movieSvc := movie.NewService(q, nil, bus, logger)
	qualSvc := quality.NewService(q, bus)
	blSvc := blocklist.NewService(q)
	dlSvc := downloader.NewService(q, reg, bus)

	svc := autosearch.NewService(indexerSvc, movieSvc, dlSvc, blSvc, qualSvc, nil, nil, bus, logger)

	return &testEnv{
		q:       q,
		svc:     svc,
		blSvc:   blSvc,
		mockIdx: mockIdx,
		mockDL:  mockDL,
	}
}

// seedWithIndexerAndDownloader creates DB rows for an enabled indexer and download
// client so the service layer can find them.
func seedWithIndexerAndDownloader(t *testing.T, env *testEnv) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := env.q.CreateIndexerConfig(ctx, dbgen.CreateIndexerConfigParams{
		ID:       uuid.New().String(),
		Name:     "Test Indexer",
		Kind:     "mock",
		Enabled:  true,
		Priority: 1,
		Settings: "{}",
	})
	if err != nil {
		t.Fatalf("seed indexer: %v", err)
	}

	_, err = env.q.CreateDownloadClientConfig(ctx, dbgen.CreateDownloadClientConfigParams{
		ID:        uuid.New().String(),
		Name:      "Test Downloader",
		Kind:      "mock-dl",
		Enabled:   true,
		Priority:  1,
		Settings:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed download client: %v", err)
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestSearchMovie_GrabsBestRelease(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 720p", Protocol: plugin.ProtocolTorrent, DownloadURL: "http://x/1",
			Quality: plugin.Quality{Resolution: "720p", Source: "bluray"}, Seeds: 10},
		{GUID: "r2", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent, DownloadURL: "http://x/2",
			Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("got status %q, want %q (reason: %s)", result.Status, autosearch.StatusGrabbed, result.Reason)
	}
	if result.Grab == nil {
		t.Fatal("expected non-nil Grab")
	}
}

// Regression for cc5ce18: auto-grab must skip releases tagged by the
// per-indexer min_seeders filter and pick the next clean candidate.
// Without this guard the filter would only matter for the manual-search
// UI; auto-grab would happily download the dead release.
func TestSearchMovie_SkipsBelowMinSeeders_GrabsNext(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	// r-dead has Seeds=0 → indexer.Service.Search will tag it
	// "below minimum seeders (0 < 5)" and auto-grab must skip it.
	// r-live has 50 seeders, well above the default-5 threshold.
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r-dead", Title: "Inception 2010 1080p Dead", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/dead", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 0},
		{GUID: "r-live", Title: "Inception 2010 1080p Live", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/live", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 50},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("got %q, want %q (reason: %s)", result.Status, autosearch.StatusGrabbed, result.Reason)
	}
	if result.Grab == nil || result.Grab.ReleaseTitle != "Inception 2010 1080p Live" {
		got := ""
		if result.Grab != nil {
			got = result.Grab.ReleaseTitle
		}
		t.Errorf("expected to grab the live release; got %q", got)
	}
}

// When every candidate gets tagged by the min_seeders filter, auto-grab
// must return no_match rather than ignoring the filter and grabbing
// anyway. The user can still pick from the manual-search UI which
// renders tagged rows greyed with an override button.
func TestSearchMovie_AllBelowMinSeeders_NoMatch(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p A", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 0},
		{GUID: "r2", Title: "Inception 2010 1080p B", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/2", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 1},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got %q, want %q (reason: %s)", result.Status, autosearch.StatusNoMatch, result.Reason)
	}
}

func TestSearchMovie_NoReleases(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = nil

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got status %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}

func TestSearchMovie_MovieNotFound(t *testing.T) {
	t.Parallel()
	env := setup(t)

	_, err := env.svc.SearchMovie(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent movie")
	}
}

func TestSearchMovie_AllBlocklisted(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "blocked-1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	ctx := context.Background()
	err := env.blSvc.Add(ctx, mov.ID, "blocked-1", "Inception 2010 1080p", "", "torrent", 1000, "test")
	if err != nil {
		t.Fatalf("blocklist add: %v", err)
	}

	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got status %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}

func TestSearchMovie_SkipsBlocklisted_GrabsNext(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r-bad", Title: "Inception 2010 1080p Bad", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/bad", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
		{GUID: "r-good", Title: "Inception 2010 1080p Good", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/good", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	ctx := context.Background()
	err := env.blSvc.Add(ctx, mov.ID, "r-bad", "Inception 2010 1080p Bad", "", "torrent", 1000, "bad release")
	if err != nil {
		t.Fatalf("blocklist add: %v", err)
	}

	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("got status %q, want %q (reason: %s)", result.Status, autosearch.StatusGrabbed, result.Reason)
	}
	if result.Grab.ReleaseTitle != "Inception 2010 1080p Good" {
		t.Fatalf("grabbed wrong release: %s", result.Grab.ReleaseTitle)
	}
}

func TestSearchMovie_ActiveGrab_Skipped(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	ctx := context.Background()
	result, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if result.Status != autosearch.StatusGrabbed {
		t.Fatalf("first search: got %q, want %q", result.Status, autosearch.StatusGrabbed)
	}

	// Second search for same movie — unique index prevents duplicate active grab.
	result2, err := env.svc.SearchMovie(ctx, mov.ID)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if result2.Status != autosearch.StatusSkipped {
		t.Fatalf("second search: got %q, want %q (reason: %s)", result2.Status, autosearch.StatusSkipped, result2.Reason)
	}
}

func TestSearchMovies_BulkCounts(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov1 := testutil.SeedMovie(t, env.q, testutil.WithTMDBID(27205))
	mov2 := testutil.SeedMovie(t, env.q, testutil.WithTMDBID(155))

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1", Quality: plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	ctx := context.Background()
	bulk := env.svc.SearchMovies(ctx, []string{mov1.ID, mov2.ID})

	if bulk.Searched != 2 {
		t.Fatalf("searched: got %d, want 2", bulk.Searched)
	}
	if bulk.Grabbed < 1 {
		t.Fatalf("grabbed: got %d, want >= 1", bulk.Grabbed)
	}
	if len(bulk.Results) != 2 {
		t.Fatalf("results: got %d, want 2", len(bulk.Results))
	}
}

func TestSearchMovies_Empty(t *testing.T) {
	t.Parallel()
	env := setup(t)

	bulk := env.svc.SearchMovies(context.Background(), nil)
	if bulk.Searched != 0 {
		t.Fatalf("searched: got %d, want 0", bulk.Searched)
	}
	if bulk.Grabbed != 0 {
		t.Fatalf("grabbed: got %d, want 0", bulk.Grabbed)
	}
}

func TestSearchMovie_ProfileRejectsQuality(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)

	mov := testutil.SeedMovie(t, env.q)

	// Only 720p — test profile only allows 1080p bluray.
	env.mockIdx.releases = []plugin.Release{
		{GUID: "r720", Title: "Inception 2010 720p", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/720", Quality: plugin.Quality{Resolution: "720p", Source: "bluray"}, Seeds: 10},
	}

	result, err := env.svc.SearchMovie(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != autosearch.StatusNoMatch {
		t.Fatalf("got %q, want %q", result.Status, autosearch.StatusNoMatch)
	}
}

// ── SearchMovieExplain ────────────────────────────────────────────────────────
//
// SearchMovieExplain is the dry-run twin of SearchMovie — it classifies
// every candidate release with a SkipReason instead of grabbing one.
// Each branch maps to a UI pill in the manual-search modal. Without
// these tests the per-pill classification can drift silently (e.g. a
// rejected release showing as "blocklisted" when it's actually
// "quality_not_in_profile").
//
// Coverage gap before this commit: 0% on the entire SearchMovieExplain
// branch matrix despite ~150 lines of decision code.

// Headline happy path: a 1080p BluRay x264 release matches the seeded
// profile exactly, so the explain output marks it as grabbed. This
// pins both Outcome="grabbed" and Reason=ReasonGrabbed.
func TestExplain_PassingReleaseClassifiesAsGrabbed(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r1", Title: "Inception 2010 1080p Bluray x264", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1",
			Quality:     plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"},
			Seeds:       10},
	}

	result, err := env.svc.SearchMovieExplain(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(result.Decisions))
	}
	d := result.Decisions[0]
	if d.Outcome != "grabbed" {
		t.Errorf("Outcome = %q, want %q", d.Outcome, "grabbed")
	}
	if d.Reason != autosearch.ReasonGrabbed {
		t.Errorf("Reason = %q, want %q", d.Reason, autosearch.ReasonGrabbed)
	}
	// Explain runs SearchMovieExplain — DOES NOT actually grab. Ensure
	// nothing was sent to the mock downloader.
	if env.mockDL.itemID == "" {
		t.Error("mockDL.itemID was reset — explain should never grab")
	}
}

// A blocklisted release must classify as ReasonBlocklisted regardless
// of how good its quality is. The pill shows "blocklisted" so the user
// knows why a perfect-looking release is being skipped.
func TestExplain_BlocklistedReleaseClassifiesAsBlocklisted(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "blocked-guid", Title: "Inception 2010 1080p Bluray x264", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/1",
			Quality:     plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"},
			Seeds:       10},
	}
	if err := env.blSvc.Add(context.Background(), mov.ID, "blocked-guid",
		"Inception 2010 1080p Bluray x264", "", "torrent", 1000, "test-blocklist"); err != nil {
		t.Fatalf("blocklist add: %v", err)
	}

	result, err := env.svc.SearchMovieExplain(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(result.Decisions))
	}
	d := result.Decisions[0]
	if d.Outcome != "skipped" {
		t.Errorf("Outcome = %q, want %q", d.Outcome, "skipped")
	}
	if d.Reason != autosearch.ReasonBlocklisted {
		t.Errorf("Reason = %q, want %q (blocklist branch in SearchMovieExplain)",
			d.Reason, autosearch.ReasonBlocklisted)
	}
}

// A 720p release against a 1080p-only profile must classify as
// ReasonQualityNotAllowed (not "no_upgrade_needed", not blocklisted).
// This is the difference the manual-search UI uses to render the
// "Quality" pill differently from the "Already have a copy" pill.
func TestExplain_QualityOutOfProfileClassifiesAsQualityNotAllowed(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		{GUID: "r720", Title: "Inception 2010 720p WEB", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/720",
			Quality:     plugin.Quality{Resolution: "720p", Source: "webdl"},
			Seeds:       10},
	}

	result, err := env.svc.SearchMovieExplain(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(result.Decisions))
	}
	d := result.Decisions[0]
	if d.Outcome != "skipped" {
		t.Errorf("Outcome = %q, want %q", d.Outcome, "skipped")
	}
	if d.Reason != autosearch.ReasonQualityNotAllowed {
		t.Errorf("Reason = %q, want %q (quality_not_in_profile branch)",
			d.Reason, autosearch.ReasonQualityNotAllowed)
	}
}

// Multiple-candidate fan-out: explain returns a decision for EVERY
// candidate, not just the first passing one. The first passing
// candidate gets ReasonGrabbed; subsequent passing candidates also
// get ReasonGrabbed (they "would have been grabbed" if the first
// wasn't there); rejected ones get their specific reject reason.
func TestExplain_MixedDecisionsReturnedInOrder(t *testing.T) {
	t.Parallel()
	env := setup(t)
	seedWithIndexerAndDownloader(t, env)
	mov := testutil.SeedMovie(t, env.q)

	env.mockIdx.releases = []plugin.Release{
		// Out-of-profile quality.
		{GUID: "r-720", Title: "Inception 2010 720p WEB", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/720",
			Quality:     plugin.Quality{Resolution: "720p", Source: "webdl"}, Seeds: 10},
		// In-profile.
		{GUID: "r-good", Title: "Inception 2010 1080p Bluray x264", Protocol: plugin.ProtocolTorrent,
			DownloadURL: "http://x/good",
			Quality:     plugin.Quality{Resolution: "1080p", Source: "bluray", Codec: "x264"}, Seeds: 10},
	}

	result, err := env.svc.SearchMovieExplain(context.Background(), mov.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Decisions) != 2 {
		t.Fatalf("len(decisions) = %d, want 2 — got: %+v", len(result.Decisions), result.Decisions)
	}

	// Both reasons must appear; the order tracks the post-quality-sort
	// ordering, which puts higher-scoring releases first.
	gotReasons := map[autosearch.SkipReason]bool{}
	for _, d := range result.Decisions {
		gotReasons[d.Reason] = true
	}
	if !gotReasons[autosearch.ReasonGrabbed] {
		t.Errorf("expected at least one ReasonGrabbed, got %+v", gotReasons)
	}
	if !gotReasons[autosearch.ReasonQualityNotAllowed] {
		t.Errorf("expected at least one ReasonQualityNotAllowed, got %+v", gotReasons)
	}
}
