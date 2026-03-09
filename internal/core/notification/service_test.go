package notification_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/luminarr/luminarr/internal/core/notification"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── Mock notifier ─────────────────────────────────────────────────────────────

type mockNotifier struct {
	testErr error
}

func (m *mockNotifier) Name() string                                               { return "mock" }
func (m *mockNotifier) Notify(_ context.Context, _ plugin.NotificationEvent) error { return nil }
func (m *mockNotifier) Test(_ context.Context) error                               { return m.testErr }

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestReg(mock *mockNotifier) *registry.Registry {
	reg := registry.New()
	reg.RegisterNotifier("mock", func(_ json.RawMessage) (plugin.Notifier, error) {
		return mock, nil
	})
	return reg
}

func newServiceFromSQL(sqlDB *sql.DB, mock *mockNotifier) *notification.Service {
	q := dbsqlite.New(sqlDB)
	return notification.NewService(q, newTestReg(mock))
}

func sampleSettings() json.RawMessage {
	b, _ := json.Marshal(map[string]string{"url": "https://discord.com/api/webhooks/test"})
	return b
}

func sampleCreateReq() notification.CreateRequest {
	return notification.CreateRequest{
		Name:     "My Discord",
		Kind:     "mock",
		Enabled:  true,
		Settings: sampleSettings(),
		OnEvents: []string{"grab_started", "download_done"},
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_Create(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	req := sampleCreateReq()
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if cfg.Name != req.Name {
		t.Errorf("Name = %q, want %q", cfg.Name, req.Name)
	}
	if cfg.Kind != "mock" {
		t.Errorf("Kind = %q, want mock", cfg.Kind)
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true")
	}
	if len(cfg.OnEvents) != 2 {
		t.Errorf("OnEvents len = %d, want 2", len(cfg.OnEvents))
	}
}

func TestService_Create_UnknownKind(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Kind = "does-not-exist"
	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() with unknown kind should return error")
	}
}

func TestService_Create_NilSettings(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Settings = nil
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() with nil settings error = %v", err)
	}
	// Should default to "{}"
	if string(cfg.Settings) != "{}" {
		t.Errorf("Settings = %q, want {}", string(cfg.Settings))
	}
}

func TestService_Get(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != created.Name {
		t.Errorf("Name = %q, want %q", got.Name, created.Name)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	_, err := svc.Get(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestService_List_Empty(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() count = %d, want 0", len(items))
	}
}

func TestService_List_ReturnsCreated(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	req := sampleCreateReq()
	_, _ = svc.Create(ctx, req)
	req.Name = "Second Notification"
	_, _ = svc.Create(ctx, req)

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("List() count = %d, want 2", len(items))
	}
}

func TestService_Update(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	updated, err := svc.Update(ctx, created.ID, notification.UpdateRequest{
		Name:     "Updated Name",
		Kind:     "mock",
		Enabled:  false,
		Settings: created.Settings,
		OnEvents: []string{"health_issue"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want Updated Name", updated.Name)
	}
	if updated.Enabled {
		t.Error("Enabled = true, want false")
	}
	if len(updated.OnEvents) != 1 || updated.OnEvents[0] != "health_issue" {
		t.Errorf("OnEvents = %v, want [health_issue]", updated.OnEvents)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	_, err := svc.Update(ctx, "00000000-0000-0000-0000-000000000000", sampleCreateReq())
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err := svc.Get(ctx, created.ID)
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	err := svc.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestService_Test_Success(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{testErr: nil})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err != nil {
		t.Errorf("Test() error = %v, want nil", err)
	}
}

func TestService_Test_Failure(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{testErr: errors.New("webhook returned 404")})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err == nil {
		t.Error("Test() should return error when notifier test fails")
	}
}

func TestService_Test_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockNotifier{})
	ctx := context.Background()

	err := svc.Test(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("Test() error = %v, want ErrNotFound", err)
	}
}

// ── buildMessage / EventToNotification ────────────────────────────────────────

func TestEventToNotification_AllEventTypes(t *testing.T) {
	tests := []struct {
		eventType string
		data      map[string]any
		wantMsg   string
	}{
		{"movie_added", map[string]any{"title": "Inception"}, "Movie added: Inception"},
		{"movie_added", map[string]any{}, "A new movie was added to the library"},
		{"movie_deleted", map[string]any{"title": "Inception"}, "Movie removed: Inception"},
		{"movie_deleted", map[string]any{}, "A movie was removed from the library"},
		{"grab_started", map[string]any{"title": "Inception"}, "Grabbing release: Inception"},
		{"grab_started", map[string]any{}, "A release was sent to the download client"},
		{"download_done", map[string]any{"title": "Inception"}, "Download complete: Inception"},
		{"download_done", map[string]any{}, "A download completed"},
		{"import_done", map[string]any{"title": "Inception"}, "Imported: Inception"},
		{"import_done", map[string]any{}, "A file was imported into the library"},
		{"import_failed", map[string]any{"title": "Inception"}, "Import failed: Inception"},
		{"import_failed", map[string]any{}, "A file import failed"},
		{"health_issue", map[string]any{"message": "disk full"}, "Health issue: disk full"},
		{"health_issue", map[string]any{}, "A health issue was detected"},
		{"health_ok", map[string]any{"message": "recovered"}, "Health restored: recovered"},
		{"health_ok", map[string]any{}, "A health check recovered"},
		{"unknown_event", map[string]any{}, "Event: unknown_event"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			evt := notification.EventToNotification(tt.eventType, "movie-123", tt.data)
			if evt.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", evt.Message, tt.wantMsg)
			}
			if string(evt.Type) != tt.eventType {
				t.Errorf("Type = %q, want %q", evt.Type, tt.eventType)
			}
			if evt.MovieID != "movie-123" {
				t.Errorf("MovieID = %q, want movie-123", evt.MovieID)
			}
			if evt.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}
		})
	}
}
