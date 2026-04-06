package library_test

import (
	"context"
	"errors"
	"testing"

	"github.com/beacon-media/prism/internal/core/library"
	"github.com/beacon-media/prism/internal/core/quality"
	"github.com/beacon-media/prism/internal/testutil"
	"github.com/beacon-media/prism/pkg/plugin"
)

func newServices(t *testing.T) (*library.Service, *quality.Service) {
	t.Helper()
	q := testutil.NewTestDB(t)
	return library.NewService(q, nil, nil), quality.NewService(q, nil)
}

// createQualityProfile is a test helper that inserts a quality profile and
// returns its ID so library tests can satisfy the foreign-key constraint.
func createQualityProfile(t *testing.T, qSvc *quality.Service) string {
	t.Helper()
	cutoff := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX264, HDR: plugin.HDRNone}
	p, err := qSvc.Create(context.Background(), quality.CreateRequest{
		Name:      "Test Profile",
		Cutoff:    cutoff,
		Qualities: []plugin.Quality{cutoff},
	})
	if err != nil {
		t.Fatalf("createQualityProfile: %v", err)
	}
	return p.ID
}

func sampleCreateRequest(profileID string) library.CreateRequest {
	format := "{Title} ({Year})"
	return library.CreateRequest{
		Name:                    "My Movies",
		RootPath:                "/tmp/movies",
		DefaultQualityProfileID: profileID,
		NamingFormat:            &format,
		MinFreeSpaceGB:          10,
		Tags:                    []string{"main", "hd"},
	}
}

func TestService_Create(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()
	profileID := createQualityProfile(t, qSvc)

	req := sampleCreateRequest(profileID)
	lib, err := libSvc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if lib.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if lib.Name != req.Name {
		t.Errorf("Name = %q, want %q", lib.Name, req.Name)
	}
	if lib.RootPath != req.RootPath {
		t.Errorf("RootPath = %q, want %q", lib.RootPath, req.RootPath)
	}
	if lib.MinFreeSpaceGB != req.MinFreeSpaceGB {
		t.Errorf("MinFreeSpaceGB = %d, want %d", lib.MinFreeSpaceGB, req.MinFreeSpaceGB)
	}
	if len(lib.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(lib.Tags))
	}
	if lib.NamingFormat == nil || *lib.NamingFormat != *req.NamingFormat {
		t.Errorf("NamingFormat = %v, want %q", lib.NamingFormat, *req.NamingFormat)
	}
	if lib.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestService_Get(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()
	profileID := createQualityProfile(t, qSvc)

	created, err := libSvc.Create(ctx, sampleCreateRequest(profileID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := libSvc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID = %q, want %q", fetched.ID, created.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	libSvc, _ := newServices(t)
	_, err := libSvc.Get(context.Background(), "nonexistent-id")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Get() err = %v, want ErrNotFound", err)
	}
}

func TestService_List(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()

	list, err := libSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() count = %d, want 0", len(list))
	}

	profileID := createQualityProfile(t, qSvc)
	req := sampleCreateRequest(profileID)
	if _, err := libSvc.Create(ctx, req); err != nil {
		t.Fatalf("Create() first: %v", err)
	}
	req.Name = "Second Library"
	if _, err := libSvc.Create(ctx, req); err != nil {
		t.Fatalf("Create() second: %v", err)
	}

	list, err = libSvc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List() count = %d, want 2", len(list))
	}
}

func TestService_Update(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()
	profileID := createQualityProfile(t, qSvc)

	created, err := libSvc.Create(ctx, sampleCreateRequest(profileID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := libSvc.Update(ctx, created.ID, library.UpdateRequest{
		Name:                    "Updated Library",
		RootPath:                "/tmp/updated",
		DefaultQualityProfileID: profileID,
		MinFreeSpaceGB:          20,
		Tags:                    []string{"updated"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Library" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Library")
	}
	if updated.RootPath != "/tmp/updated" {
		t.Errorf("RootPath = %q, want %q", updated.RootPath, "/tmp/updated")
	}
	if updated.MinFreeSpaceGB != 20 {
		t.Errorf("MinFreeSpaceGB = %d, want 20", updated.MinFreeSpaceGB)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "updated" {
		t.Errorf("Tags = %v, want [updated]", updated.Tags)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	libSvc, qSvc := newServices(t)
	profileID := createQualityProfile(t, qSvc)
	_, err := libSvc.Update(context.Background(), "no-such-id", sampleCreateRequest(profileID))
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Update() err = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()
	profileID := createQualityProfile(t, qSvc)

	created, err := libSvc.Create(ctx, sampleCreateRequest(profileID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := libSvc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = libSvc.Get(ctx, created.ID)
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Get() after delete: err = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	libSvc, _ := newServices(t)
	err := libSvc.Delete(context.Background(), "does-not-exist")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Delete() err = %v, want ErrNotFound", err)
	}
}

func TestService_Stats(t *testing.T) {
	libSvc, qSvc := newServices(t)
	ctx := context.Background()
	profileID := createQualityProfile(t, qSvc)

	// Use /tmp which is guaranteed to exist on Linux.
	req := sampleCreateRequest(profileID)
	req.RootPath = "/tmp"
	req.MinFreeSpaceGB = 0

	created, err := libSvc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stats, err := libSvc.Stats(ctx, created.ID)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.MovieCount != 0 {
		t.Errorf("MovieCount = %d, want 0", stats.MovieCount)
	}
	if stats.TotalSizeBytes != 0 {
		t.Errorf("TotalSizeBytes = %d, want 0", stats.TotalSizeBytes)
	}
	// /tmp should always be accessible on Linux.
	if stats.FreeSpaceBytes < 0 {
		t.Errorf("FreeSpaceBytes = %d, expected >= 0 for /tmp", stats.FreeSpaceBytes)
	}
	if !stats.HealthOK {
		t.Errorf("HealthOK = false, want true (MinFreeSpaceGB=0 should always pass)")
	}
}

func TestService_Stats_NotFound(t *testing.T) {
	libSvc, _ := newServices(t)
	_, err := libSvc.Stats(context.Background(), "bad-id")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Stats() err = %v, want ErrNotFound", err)
	}
}
