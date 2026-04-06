package importlist_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/beacon-media/prism/internal/core/importlist"
	"github.com/beacon-media/prism/internal/core/movie"
	dbsqlite "github.com/beacon-media/prism/internal/db/generated/sqlite"
	"github.com/beacon-media/prism/internal/events"
	"github.com/beacon-media/prism/internal/metadata/tmdb"
	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/internal/testutil"
	"github.com/beacon-media/prism/pkg/plugin"
)

// ── Mock TMDB ────────────────────────────────────────────────────────────────

type mockTMDB struct {
	movies map[int]*tmdb.MovieDetail
}

func (m *mockTMDB) SearchMovies(_ context.Context, _ string, _ int) ([]tmdb.SearchResult, error) {
	return nil, nil
}

func (m *mockTMDB) GetMovie(_ context.Context, id int) (*tmdb.MovieDetail, error) {
	if d, ok := m.movies[id]; ok {
		return d, nil
	}
	return &tmdb.MovieDetail{
		ID:          id,
		Title:       "Unknown Movie",
		ReleaseDate: "2024-01-01",
		Year:        2024,
		Status:      "released",
	}, nil
}

// ── Mock import list plugin ──────────────────────────────────────────────────

type mockImportList struct {
	name    string
	items   []plugin.ImportListItem
	testErr error
}

func (m *mockImportList) Name() string { return m.name }
func (m *mockImportList) Fetch(_ context.Context) ([]plugin.ImportListItem, error) {
	return m.items, nil
}
func (m *mockImportList) Test(_ context.Context) error { return m.testErr }

// ── Helpers ──────────────────────────────────────────────────────────────────

func newTestReg(mock *mockImportList) *registry.Registry {
	reg := registry.New()
	reg.RegisterImportList("mock", func(_ json.RawMessage) (plugin.ImportList, error) {
		return mock, nil
	})
	return reg
}

func newTestService(t *testing.T, reg *registry.Registry) (*importlist.Service, *dbsqlite.Queries, *sql.DB) {
	t.Helper()
	q, sqlDB := testutil.NewTestDBWithSQL(t)
	bus := events.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	movieSvc := movie.NewService(q, &mockTMDB{}, bus, logger)
	svc := importlist.NewService(q, reg, movieSvc, nil, nil, nil, logger)
	return svc, q, sqlDB
}

func seedFixtures(t *testing.T, q *dbsqlite.Queries) (libraryID, profileID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	qp, err := q.CreateQualityProfile(ctx, dbsqlite.CreateQualityProfileParams{
		ID:            "qp-test",
		Name:          "Test Profile",
		CutoffJson:    `{}`,
		QualitiesJson: `[]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("seedFixtures: CreateQualityProfile: %v", err)
	}

	lib, err := q.CreateLibrary(ctx, dbsqlite.CreateLibraryParams{
		ID:                      "lib-test",
		Name:                    "Test Library",
		RootPath:                "/test",
		DefaultQualityProfileID: qp.ID,
		MinFreeSpaceGb:          0,
		TagsJson:                "[]",
		CreatedAt:               now,
		UpdatedAt:               now,
	})
	if err != nil {
		t.Fatalf("seedFixtures: CreateLibrary: %v", err)
	}

	return lib.ID, qp.ID
}

func sampleSettings() json.RawMessage {
	return json.RawMessage(`{}`)
}

// ── CRUD Tests ───────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	cfg, err := svc.Create(ctx, importlist.CreateRequest{
		Name:             "My List",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		SearchOnAdd:      false,
		Monitor:          true,
		MinAvailability:  "released",
		QualityProfileID: "qp-1",
		LibraryID:        "lib-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if cfg.Name != "My List" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My List")
	}
	if cfg.Kind != "mock" {
		t.Errorf("Kind = %q, want %q", cfg.Kind, "mock")
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true")
	}
	if !cfg.Monitor {
		t.Error("Monitor = false, want true")
	}
}

func TestCreate_InvalidKind(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	_, err := svc.Create(ctx, importlist.CreateRequest{
		Name:     "Bad Kind",
		Kind:     "nonexistent",
		Settings: sampleSettings(),
	})
	if err == nil {
		t.Fatal("Create() with invalid kind should fail")
	}
}

func TestGetNotFound(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Get() non-existent should fail")
	}
}

func TestList(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	// Create two lists.
	for _, name := range []string{"List A", "List B"} {
		_, err := svc.Create(ctx, importlist.CreateRequest{
			Name:     name,
			Kind:     "mock",
			Enabled:  true,
			Settings: sampleSettings(),
		})
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
	}

	cfgs, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(cfgs) != 2 {
		t.Errorf("List() returned %d items, want 2", len(cfgs))
	}
}

func TestUpdate(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	cfg, err := svc.Create(ctx, importlist.CreateRequest{
		Name:     "Original",
		Kind:     "mock",
		Enabled:  true,
		Settings: sampleSettings(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(ctx, cfg.ID, importlist.UpdateRequest{
		Name:     "Updated Name",
		Kind:     "mock",
		Enabled:  false,
		Settings: sampleSettings(),
		Monitor:  true,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Name")
	}
	if updated.Enabled {
		t.Error("Enabled = true, want false")
	}
}

func TestDelete(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	cfg, err := svc.Create(ctx, importlist.CreateRequest{
		Name:     "To Delete",
		Kind:     "mock",
		Enabled:  true,
		Settings: sampleSettings(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, cfg.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = svc.Get(ctx, cfg.ID)
	if err == nil {
		t.Error("Get() after delete should fail")
	}
}

func TestTest(t *testing.T) {
	mock := &mockImportList{name: "test", testErr: nil}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	cfg, err := svc.Create(ctx, importlist.CreateRequest{
		Name:     "Test Me",
		Kind:     "mock",
		Enabled:  true,
		Settings: sampleSettings(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Test(ctx, cfg.ID); err != nil {
		t.Errorf("Test() error = %v, want nil", err)
	}
}

// ── Exclusion Tests ──────────────────────────────────────────────────────────

func TestExclusions_CRUD(t *testing.T) {
	mock := &mockImportList{name: "test"}
	svc, _, _ := newTestService(t, newTestReg(mock))
	ctx := context.Background()

	excl, err := svc.CreateExclusion(ctx, 550, "Fight Club", 1999)
	if err != nil {
		t.Fatalf("CreateExclusion() error = %v", err)
	}
	if excl.ID == "" {
		t.Error("CreateExclusion() returned empty ID")
	}
	if excl.TMDbID != 550 {
		t.Errorf("TMDbID = %d, want 550", excl.TMDbID)
	}

	// Duplicate should fail.
	_, err = svc.CreateExclusion(ctx, 550, "Fight Club", 1999)
	if err == nil {
		t.Error("CreateExclusion() duplicate should fail")
	}

	// List should contain the exclusion.
	exclusions, err := svc.ListExclusions(ctx)
	if err != nil {
		t.Fatalf("ListExclusions() error = %v", err)
	}
	if len(exclusions) != 1 {
		t.Fatalf("ListExclusions() returned %d, want 1", len(exclusions))
	}

	// Delete.
	if err := svc.DeleteExclusion(ctx, excl.ID); err != nil {
		t.Fatalf("DeleteExclusion() error = %v", err)
	}

	exclusions, err = svc.ListExclusions(ctx)
	if err != nil {
		t.Fatalf("ListExclusions() error = %v", err)
	}
	if len(exclusions) != 0 {
		t.Errorf("ListExclusions() after delete returned %d, want 0", len(exclusions))
	}
}

// ── Sync Tests ───────────────────────────────────────────────────────────────

func TestSync_AddsNewMovies(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 100, Title: "Movie A", Year: 2024},
		{TMDbID: 200, Title: "Movie B", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, q, _ := newTestService(t, reg)
	ctx := context.Background()

	libID, profID := seedFixtures(t, q)

	// Create an enabled import list config.
	_, err := svc.Create(ctx, importlist.CreateRequest{
		Name:             "My Import",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		Monitor:          true,
		MinAvailability:  "released",
		QualityProfileID: profID,
		LibraryID:        libID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result := svc.Sync(ctx)
	if result.ListsProcessed != 1 {
		t.Errorf("ListsProcessed = %d, want 1", result.ListsProcessed)
	}
	if result.MoviesAdded != 2 {
		t.Errorf("MoviesAdded = %d, want 2", result.MoviesAdded)
	}
	if result.MoviesSkipped != 0 {
		t.Errorf("MoviesSkipped = %d, want 0", result.MoviesSkipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}
}

func TestSync_SkipsExistingMovies(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 100, Title: "Movie A", Year: 2024},
		{TMDbID: 200, Title: "Movie B", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, q, _ := newTestService(t, reg)
	ctx := context.Background()

	libID, profID := seedFixtures(t, q)

	_, err := svc.Create(ctx, importlist.CreateRequest{
		Name:             "My Import",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		Monitor:          true,
		QualityProfileID: profID,
		LibraryID:        libID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// First sync adds movies.
	result1 := svc.Sync(ctx)
	if result1.MoviesAdded != 2 {
		t.Fatalf("first sync: MoviesAdded = %d, want 2", result1.MoviesAdded)
	}

	// Second sync should skip them.
	result2 := svc.Sync(ctx)
	if result2.MoviesAdded != 0 {
		t.Errorf("second sync: MoviesAdded = %d, want 0", result2.MoviesAdded)
	}
	if result2.MoviesSkipped != 2 {
		t.Errorf("second sync: MoviesSkipped = %d, want 2", result2.MoviesSkipped)
	}
}

func TestSync_SkipsExcludedMovies(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 100, Title: "Movie A", Year: 2024},
		{TMDbID: 200, Title: "Movie B", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, q, _ := newTestService(t, reg)
	ctx := context.Background()

	libID, profID := seedFixtures(t, q)

	// Exclude movie 100 before syncing.
	_, err := svc.CreateExclusion(ctx, 100, "Movie A", 2024)
	if err != nil {
		t.Fatalf("CreateExclusion() error = %v", err)
	}

	_, err = svc.Create(ctx, importlist.CreateRequest{
		Name:             "My Import",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		Monitor:          true,
		QualityProfileID: profID,
		LibraryID:        libID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result := svc.Sync(ctx)
	if result.MoviesAdded != 1 {
		t.Errorf("MoviesAdded = %d, want 1 (movie 100 should be excluded)", result.MoviesAdded)
	}
	if result.MoviesSkipped != 1 {
		t.Errorf("MoviesSkipped = %d, want 1", result.MoviesSkipped)
	}
}

func TestSync_SkipsZeroTMDbID(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 0, Title: "No TMDb", Year: 2024},
		{TMDbID: 100, Title: "Valid", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, q, _ := newTestService(t, reg)
	ctx := context.Background()

	libID, profID := seedFixtures(t, q)

	_, err := svc.Create(ctx, importlist.CreateRequest{
		Name:             "My Import",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		Monitor:          true,
		QualityProfileID: profID,
		LibraryID:        libID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result := svc.Sync(ctx)
	if result.MoviesAdded != 1 {
		t.Errorf("MoviesAdded = %d, want 1 (zero TMDb ID should be skipped)", result.MoviesAdded)
	}
}

func TestSync_DisabledListsNotProcessed(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 100, Title: "Movie A", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, _, _ := newTestService(t, reg)
	ctx := context.Background()

	_, err := svc.Create(ctx, importlist.CreateRequest{
		Name:     "Disabled List",
		Kind:     "mock",
		Enabled:  false,
		Settings: sampleSettings(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result := svc.Sync(ctx)
	if result.ListsProcessed != 0 {
		t.Errorf("ListsProcessed = %d, want 0 (disabled list)", result.ListsProcessed)
	}
	if result.MoviesAdded != 0 {
		t.Errorf("MoviesAdded = %d, want 0", result.MoviesAdded)
	}
}

func TestSyncOne(t *testing.T) {
	items := []plugin.ImportListItem{
		{TMDbID: 100, Title: "Movie A", Year: 2024},
	}
	mock := &mockImportList{name: "test", items: items}
	reg := newTestReg(mock)
	svc, q, _ := newTestService(t, reg)
	ctx := context.Background()

	libID, profID := seedFixtures(t, q)

	cfg, err := svc.Create(ctx, importlist.CreateRequest{
		Name:             "Single Sync",
		Kind:             "mock",
		Enabled:          true,
		Settings:         sampleSettings(),
		Monitor:          true,
		QualityProfileID: profID,
		LibraryID:        libID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result := svc.SyncOne(ctx, cfg.ID)
	if result.ListsProcessed != 1 {
		t.Errorf("ListsProcessed = %d, want 1", result.ListsProcessed)
	}
	if result.MoviesAdded != 1 {
		t.Errorf("MoviesAdded = %d, want 1", result.MoviesAdded)
	}
}
