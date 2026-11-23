package notifications

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/beacon-stack/pulse/pkg/notify"
	_ "modernc.org/sqlite"

	"github.com/beacon-stack/prism/internal/core/health"
	db "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
)

type fakeQ struct {
	db.Querier
	configs []db.NotificationConfig
}

func (fakeQ) GetMovie(_ context.Context, id string) (db.Movie, error) {
	if id != "m1" {
		return db.Movie{}, sql.ErrNoRows
	}
	poster := "https://image.tmdb.org/t/p/w342/dune.jpg"
	return db.Movie{ID: "m1", Title: "Dune: Part Two", Year: 2024, TmdbID: 693134, PosterUrl: &poster}, nil
}

func (fakeQ) GetGrabByID(_ context.Context, id string) (db.GrabHistory, error) {
	if id != "g1" {
		return db.GrabHistory{}, errors.New("no grab")
	}
	return db.GrabHistory{ID: "g1", MovieID: "m1", ReleaseTitle: "Dune.Part.Two.2024.2160p.UHD.BluRay"}, nil
}

func (q fakeQ) ListNotificationConfigs(context.Context) ([]db.NotificationConfig, error) {
	return q.configs, nil
}

type recorder struct{ got []notify.Event }

func (r *recorder) Publish(e notify.Event) { r.got = append(r.got, e) }

func TestTranslate(t *testing.T) {
	ctx := context.Background()
	rec := &recorder{}
	tr := &translator{q: fakeQ{}, out: rec}

	tr.handle(ctx, events.Event{Type: events.TypeImportComplete, MovieID: "m1", Data: map[string]any{"grab_id": "g1", "quality": "2160p Remux"}})
	e := rec.got[0]
	if e.Type != notify.TypeImported || e.Title != "Imported: Dune: Part Two (2024)" || e.Message != "Dune.Part.Two.2024.2160p.UHD.BluRay" ||
		e.Fields["Quality"] != "2160p Remux" || e.URL != "https://www.themoviedb.org/movie/693134" || e.ImageURL == "" {
		t.Errorf("imported %+v", e)
	}

	rec.got = nil
	tr.handle(ctx, events.Event{Type: events.TypeGrabStarted, MovieID: "m1", Data: map[string]any{"release_title": "Dune.Part.Two.1080p", "indexer": "Torrentio"}})
	if len(rec.got) != 1 || rec.got[0].Title != "Grabbed: Dune: Part Two (2024)" || rec.got[0].Fields["Indexer"] != "Torrentio" {
		t.Errorf("grabbed %+v", rec.got)
	}

	// One message per failed grab.
	rec.got = nil
	for i := 0; i < 2; i++ {
		tr.handle(ctx, events.Event{Type: events.TypeImportFailed, MovieID: "m1", Data: map[string]any{"grab_id": "g1", "reason": "no video file"}})
	}
	if len(rec.got) != 1 || rec.got[0].Level != notify.LevelError {
		t.Errorf("import failed %+v", rec.got)
	}

	rec.got = nil
	tr.handle(ctx, events.Event{Type: events.TypeGrabFailed, Data: map[string]any{"release_title": "Some.Movie.2024", "reason": "download client rejected release"}})
	if len(rec.got) != 1 || rec.got[0].Type != notify.TypeDownloadProblem || rec.got[0].Title != "Download failed: Some.Movie.2024" {
		t.Errorf("grab failed %+v", rec.got)
	}

	rec.got = nil
	tr.handle(ctx, events.Event{Type: events.TypeBulkSearchProgress})
	tr.handle(ctx, events.Event{Type: events.TypeTaskStarted})
	if len(rec.got) != 0 {
		t.Errorf("internal events produced %+v", rec.got)
	}
}

func TestHealthCheck(t *testing.T) {
	check := HealthCheck(func(context.Context) health.Report {
		return health.Report{Checks: []health.CheckResult{
			{Name: "Libraries", Status: health.StatusHealthy},
			{Name: "Indexers", Status: health.StatusDegraded, Message: "1 indexer offline"},
			{Name: "Download clients", Status: health.StatusUnhealthy, Message: "Haul unreachable"},
		}}
	})
	got := check(context.Background())
	if len(got) != 2 || got[0].Key != "Indexers" || got[0].Level != notify.LevelWarning || got[1].Level != notify.LevelError || got[1].Message != "Download clients: Haul unreachable" {
		t.Errorf("problems %+v", got)
	}
}

func TestSetup_BringsOverOldChannels(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "prism.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	q := fakeQ{configs: []db.NotificationConfig{
		{Name: "Phone", Kind: "ntfy", Enabled: true, Settings: `{"url":"https://ntfy.sh","topic":"t"}`, OnEvents: `["import_done","health_issue"]`},
		{Name: "Script", Kind: "command", Enabled: true, Settings: `{}`, OnEvents: `[]`},
	}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store, _, err := Setup(ctx, Deps{DB: sqlDB, Q: q, Bus: events.New(log), Log: log})
	if err != nil {
		t.Fatal(err)
	}
	list, _ := store.List(ctx)
	if len(list) != 1 || list[0].Name != "Phone" || len(list[0].Events) != 2 || list[0].Events[0] != notify.TypeImported {
		t.Errorf("brought over %+v", list)
	}
}
