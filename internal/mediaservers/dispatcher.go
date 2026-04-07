// Package mediaservers wires the event bus to the media server plugin system.
// The Dispatcher subscribes to import_complete events and triggers a library
// refresh on every enabled media server so new files appear immediately.
package mediaservers

import (
	"context"
	"log/slog"

	dbsqlite "github.com/beacon-stack/prism/internal/db/generated/sqlite"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/registry"
)

// Dispatcher subscribes to the event bus and dispatches import_complete events
// to all enabled media server plugins.
type Dispatcher struct {
	q      dbsqlite.Querier
	reg    *registry.Registry
	bus    *events.Bus
	logger *slog.Logger
}

// NewDispatcher creates a Dispatcher. Call Subscribe() to start receiving events.
func NewDispatcher(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{q: q, reg: reg, bus: bus, logger: logger}
}

// Subscribe registers the dispatcher as a handler on the event bus.
func (d *Dispatcher) Subscribe() {
	d.bus.Subscribe(func(ctx context.Context, e events.Event) {
		if e.Type != events.TypeImportComplete {
			return
		}
		d.handle(ctx, e)
	})
}

func (d *Dispatcher) handle(ctx context.Context, e events.Event) {
	destPath, _ := e.Data["dest_path"].(string)
	if destPath == "" {
		d.logger.Warn("mediaserver dispatcher: import_complete event missing dest_path")
		return
	}

	rows, err := d.q.ListEnabledMediaServers(ctx)
	if err != nil {
		d.logger.Warn("mediaserver dispatcher: could not load configs", "error", err)
		return
	}

	for _, row := range rows {
		ms, err := d.reg.NewMediaServer(row.Kind, []byte(row.Settings))
		if err != nil {
			d.logger.Warn("mediaserver dispatcher: could not instantiate",
				"id", row.ID,
				"kind", row.Kind,
				"error", err,
			)
			continue
		}

		d.logger.Info("mediaserver dispatcher: refreshing library",
			"id", row.ID,
			"kind", row.Kind,
			"path", destPath,
		)

		if err := ms.RefreshLibrary(ctx, destPath); err != nil {
			d.logger.Warn("mediaserver dispatcher: refresh failed",
				"id", row.ID,
				"kind", row.Kind,
				"path", destPath,
				"error", err,
			)
		}
	}
}
