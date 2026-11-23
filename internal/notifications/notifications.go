// Package notifications connects Prism to the shared Beacon notifications
// (pulse/pkg/notify): its channels and routing live in notify's store, and
// Prism's internal events are turned into the shared events here.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/beacon-stack/pulse/pkg/notify"

	"github.com/beacon-stack/prism/internal/core/health"
	db "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
)

// Deps is what Setup needs from Prism.
type Deps struct {
	DB  *sql.DB
	Q   db.Querier
	Bus *events.Bus
	// Hub returns Pulse when Prism is connected to it, nil otherwise.
	Hub func() notify.Hub
	Log *slog.Logger
}

// Setup opens the store, brings over channels set up before the shared
// notifications existed (once), and starts turning Prism's events into
// notifications. Run the returned outbox's Run in the background.
func Setup(ctx context.Context, d Deps) (*notify.Store, *notify.Outbox, error) {
	store, err := notify.NewStore(ctx, d.DB)
	if err != nil {
		return nil, nil, err
	}
	importOld(ctx, store, d.Q, d.Log)
	out := &notify.Outbox{App: "Prism", Store: store, Hub: d.Hub, Log: d.Log}
	t := &translator{q: d.Q, out: out}
	d.Bus.Subscribe(t.handle)
	return store, out, nil
}

func importOld(ctx context.Context, store *notify.Store, q db.Querier, log *slog.Logger) {
	rows, err := q.ListNotificationConfigs(ctx)
	if err != nil {
		log.Warn("notifications: couldn't read the old channels to bring them over", "error", err)
		return
	}
	old := make([]notify.LegacyChannel, 0, len(rows))
	for _, r := range rows {
		var evts []string
		_ = json.Unmarshal([]byte(r.OnEvents), &evts)
		old = append(old, notify.LegacyChannel{Name: r.Name, Kind: r.Kind, Enabled: r.Enabled, Settings: []byte(r.Settings), Events: evts})
	}
	n, skipped, err := store.ImportLegacy(ctx, old)
	if err != nil {
		log.Warn("notifications: bringing over old channels failed", "error", err)
		return
	}
	if n > 0 {
		log.Info("notifications: brought over existing channels", "count", n)
	}
	for _, s := range skipped {
		log.Warn("notifications: a custom-script channel was not brought over; scripts are no longer supported", "channel", s)
	}
}

// HealthCheck adapts Prism's health report for notify.HealthWatcher: one
// problem per subsystem (libraries, download clients, indexers).
func HealthCheck(check func(ctx context.Context) health.Report) func(ctx context.Context) []notify.Problem {
	return func(ctx context.Context) []notify.Problem {
		var out []notify.Problem
		for _, c := range check(ctx).Checks {
			if c.Status == health.StatusHealthy {
				continue
			}
			level := notify.LevelWarning
			if c.Status == health.StatusUnhealthy {
				level = notify.LevelError
			}
			out = append(out, notify.Problem{Key: c.Name, Message: c.Name + ": " + c.Message, Level: level})
		}
		return out
	}
}

type publisher interface {
	Publish(notify.Event)
}

// translator turns Prism's events into the shared notification events.
type translator struct {
	q   db.Querier
	out publisher

	mu     sync.Mutex
	failed map[string]time.Time // grab IDs already told about
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return strings.TrimSpace(s)
}

// movie looks up the film behind an event for its name, poster and link.
func (t *translator) movie(ctx context.Context, e events.Event) (db.Movie, bool) {
	id := e.MovieID
	if id == "" {
		id = str(e.Data, "movie_id")
	}
	if id == "" {
		if gid := str(e.Data, "grab_id"); gid != "" {
			if g, err := t.q.GetGrabByID(ctx, gid); err == nil {
				id = g.MovieID
			}
		}
	}
	if id == "" {
		return db.Movie{}, false
	}
	m, err := t.q.GetMovie(ctx, id)
	return m, err == nil
}

func name(m db.Movie, ok bool, fallback string) string {
	if ok && m.Title != "" {
		if m.Year > 0 {
			return fmt.Sprintf("%s (%d)", m.Title, m.Year)
		}
		return m.Title
	}
	if fallback != "" {
		return fallback
	}
	return "a movie"
}

func dress(ev *notify.Event, m db.Movie, ok bool) {
	if !ok {
		return
	}
	if m.PosterUrl != nil {
		ev.ImageURL = *m.PosterUrl
	}
	if m.TmdbID > 0 {
		ev.URL = fmt.Sprintf("https://www.themoviedb.org/movie/%d", m.TmdbID)
	}
}

func (t *translator) firstFailure(grabID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failed == nil {
		t.failed = map[string]time.Time{}
	}
	now := time.Now()
	for id, at := range t.failed {
		if now.Sub(at) > 24*time.Hour {
			delete(t.failed, id)
		}
	}
	if _, seen := t.failed[grabID]; seen {
		return false
	}
	t.failed[grabID] = now
	return true
}

func (t *translator) handle(ctx context.Context, e events.Event) {
	if ev, ok := t.translate(ctx, e); ok {
		t.out.Publish(ev)
	}
}

func (t *translator) translate(ctx context.Context, e events.Event) (notify.Event, bool) {
	m, ok := t.movie(ctx, e)
	var ev notify.Event
	switch e.Type {
	case events.TypeMovieAdded:
		ev = notify.Event{Type: notify.TypeAdded, Level: notify.LevelInfo, Title: "Added: " + name(m, ok, str(e.Data, "title"))}

	case events.TypeGrabStarted:
		release := str(e.Data, "release_title")
		ev = notify.Event{Type: notify.TypeGrabbed, Level: notify.LevelInfo, Title: "Grabbed: " + name(m, ok, release), Message: release}
		if ix := str(e.Data, "indexer"); ix != "" {
			ev.Fields = map[string]string{"Indexer": ix}
		}

	case events.TypeDownloadDone:
		release := str(e.Data, "release_title")
		ev = notify.Event{Type: notify.TypeDownloaded, Level: notify.LevelInfo, Title: "Downloaded: " + name(m, ok, release), Message: release}

	case events.TypeImportComplete:
		ev = notify.Event{Type: notify.TypeImported, Level: notify.LevelSuccess, Title: "Imported: " + name(m, ok, str(e.Data, "movie_title"))}
		if gid := str(e.Data, "grab_id"); gid != "" {
			if g, err := t.q.GetGrabByID(ctx, gid); err == nil {
				ev.Message = g.ReleaseTitle
			}
		}
		if q := str(e.Data, "quality"); q != "" {
			ev.Fields = map[string]string{"Quality": q}
		}

	case events.TypeImportFailed:
		if gid := str(e.Data, "grab_id"); gid != "" && !t.firstFailure(gid) {
			return ev, false
		}
		ev = notify.Event{Type: notify.TypeImportFailed, Level: notify.LevelError, Title: "Import failed: " + name(m, ok, str(e.Data, "movie_title")),
			Message: strings.TrimSpace(str(e.Data, "release_title") + "\n" + str(e.Data, "reason"))}

	case events.TypeGrabFailed:
		release := str(e.Data, "release_title")
		ev = notify.Event{Type: notify.TypeDownloadProblem, Level: notify.LevelError, Title: "Download failed: " + name(m, ok, release),
			Message: strings.TrimSpace(release + "\n" + str(e.Data, "reason"))}

	default:
		return ev, false
	}
	dress(&ev, m, ok)
	return ev, true
}
