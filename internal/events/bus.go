package events

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Type identifies what happened.
type Type string

const (
	TypeMovieAdded         Type = "movie_added"
	TypeMovieDeleted       Type = "movie_deleted"
	TypeMovieUpdated       Type = "movie_updated"
	TypeGrabStarted        Type = "grab_started"
	TypeGrabFailed         Type = "grab_failed"
	TypeDownloadDone       Type = "download_done"
	TypeImportComplete     Type = "import_complete"
	TypeImportFailed       Type = "import_failed"
	TypeHealthIssue        Type = "health_issue"
	TypeHealthOK           Type = "health_ok"
	TypeTaskStarted        Type = "task_started"
	TypeTaskFinished       Type = "task_finished"
	TypeBulkSearchProgress Type = "bulk_search_progress"
	TypeBulkSearchComplete Type = "bulk_search_complete"
)

// Event carries the context of something that happened.
type Event struct {
	Type      Type           `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	MovieID   string         `json:"movie_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// Handler is a function that receives events.
type Handler func(ctx context.Context, e Event)

// Bus is a simple in-process publish/subscribe event bus.
// Publish is non-blocking — each handler runs in its own goroutine.
// A panicking handler is recovered and logged; it does not affect other handlers.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
	logger   *slog.Logger
}

// New creates a new Bus.
func New(logger *slog.Logger) *Bus {
	return &Bus{logger: logger}
}

// Subscribe registers a handler to receive all future events.
func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish sends an event to all registered handlers asynchronously.
// It returns immediately; handlers run concurrently in separate goroutines.
// The context passed to handlers is detached from the caller's cancellation
// so that handlers are not aborted when the originating HTTP request ends.
func (b *Bus) Publish(ctx context.Context, e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	// Detach from the caller's cancellation — event handlers outlive HTTP requests.
	handlerCtx := context.WithoutCancel(ctx)

	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	for _, h := range handlers {
		h := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					b.logger.Error("event handler panicked",
						"event_type", e.Type,
						"panic", r,
					)
				}
			}()
			h(handlerCtx, e)
		}()
	}
}
