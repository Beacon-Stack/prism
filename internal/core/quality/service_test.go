package quality_test

import (
	"context"
	"errors"
	"testing"

	"github.com/beacon-media/prism/internal/core/quality"
	"github.com/beacon-media/prism/internal/testutil"
	"github.com/beacon-media/prism/pkg/plugin"
)

// newService is a thin helper that wires up a Service backed by a fresh in-memory DB.
func newService(t *testing.T) *quality.Service {
	t.Helper()
	q := testutil.NewTestDB(t)
	return quality.NewService(q, nil)
}

func sampleCreateRequest() quality.CreateRequest {
	cutoff := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX264, HDR: plugin.HDRNone}
	until := plugin.Quality{Resolution: plugin.Resolution2160p, Source: plugin.SourceBluRay, Codec: plugin.CodecX265, HDR: plugin.HDRNone}
	return quality.CreateRequest{
		Name:           "HD Standard",
		Cutoff:         cutoff,
		Qualities:      []plugin.Quality{cutoff, until},
		UpgradeAllowed: true,
		UpgradeUntil:   &until,
	}
}

func TestService_Create(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	req := sampleCreateRequest()
	p, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if p.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if p.Name != req.Name {
		t.Errorf("Name = %q, want %q", p.Name, req.Name)
	}
	if p.Cutoff.Resolution != req.Cutoff.Resolution {
		t.Errorf("Cutoff.Resolution = %q, want %q", p.Cutoff.Resolution, req.Cutoff.Resolution)
	}
	if !p.UpgradeAllowed {
		t.Error("UpgradeAllowed = false, want true")
	}
	if p.UpgradeUntil == nil {
		t.Error("UpgradeUntil = nil, want non-nil")
	}
	if len(p.Qualities) != 2 {
		t.Errorf("Qualities count = %d, want 2", len(p.Qualities))
	}
}

func TestService_Get(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, sampleCreateRequest())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fetched, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID = %q, want %q", fetched.ID, created.ID)
	}
	if fetched.Name != created.Name {
		t.Errorf("Name = %q, want %q", fetched.Name, created.Name)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	svc := newService(t)
	_, err := svc.Get(context.Background(), "nonexistent-id")
	if !errors.Is(err, quality.ErrNotFound) {
		t.Errorf("Get() err = %v, want ErrNotFound", err)
	}
}

func TestService_List(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	// Empty list initially.
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() count = %d, want 0", len(list))
	}

	// Create two profiles.
	req := sampleCreateRequest()
	if _, err := svc.Create(ctx, req); err != nil {
		t.Fatalf("Create() first: %v", err)
	}
	req.Name = "4K Profile"
	if _, err := svc.Create(ctx, req); err != nil {
		t.Fatalf("Create() second: %v", err)
	}

	list, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List() count = %d, want 2", len(list))
	}
}

func TestService_Update(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, sampleCreateRequest())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, quality.UpdateRequest{
		Name:           "Updated Name",
		Cutoff:         created.Cutoff,
		Qualities:      created.Qualities,
		UpgradeAllowed: false,
		UpgradeUntil:   nil,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Name")
	}
	if updated.UpgradeAllowed {
		t.Error("UpgradeAllowed = true, want false after update")
	}
	if updated.UpgradeUntil != nil {
		t.Error("UpgradeUntil should be nil after update")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := newService(t)
	_, err := svc.Update(context.Background(), "no-such-id", sampleCreateRequest())
	if !errors.Is(err, quality.ErrNotFound) {
		t.Errorf("Update() err = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, sampleCreateRequest())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Confirm it's gone.
	_, err = svc.Get(ctx, created.ID)
	if !errors.Is(err, quality.ErrNotFound) {
		t.Errorf("Get() after delete: err = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := newService(t)
	err := svc.Delete(context.Background(), "does-not-exist")
	if !errors.Is(err, quality.ErrNotFound) {
		t.Errorf("Delete() err = %v, want ErrNotFound", err)
	}
}
