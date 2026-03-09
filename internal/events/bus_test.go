package events

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBus_PublishDelivers(t *testing.T) {
	bus := New(slog.Default())

	var received atomic.Bool
	var gotEvent Event

	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe(func(_ context.Context, e Event) {
		gotEvent = e
		received.Store(true)
		wg.Done()
	})

	bus.Publish(context.Background(), Event{
		Type:    TypeGrabStarted,
		MovieID: "movie-123",
		Data:    map[string]any{"title": "Inception"},
	})

	wg.Wait()

	if !received.Load() {
		t.Fatal("handler was not called")
	}
	if gotEvent.Type != TypeGrabStarted {
		t.Errorf("got type %q, want %q", gotEvent.Type, TypeGrabStarted)
	}
	if gotEvent.MovieID != "movie-123" {
		t.Errorf("got movie_id %q, want %q", gotEvent.MovieID, "movie-123")
	}
	if gotEvent.Timestamp.IsZero() {
		t.Error("timestamp should be auto-set when zero")
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := New(slog.Default())

	var count atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		bus.Subscribe(func(_ context.Context, e Event) {
			count.Add(1)
			wg.Done()
		})
	}

	bus.Publish(context.Background(), Event{Type: TypeMovieAdded})

	wg.Wait()

	if got := count.Load(); got != 5 {
		t.Errorf("got %d handler calls, want 5", got)
	}
}

func TestBus_PanickingHandlerDoesNotAffectOthers(t *testing.T) {
	bus := New(slog.Default())

	var received atomic.Bool
	var wg sync.WaitGroup
	wg.Add(2)

	// First handler panics
	bus.Subscribe(func(_ context.Context, e Event) {
		defer wg.Done()
		panic("boom")
	})

	// Second handler should still run
	bus.Subscribe(func(_ context.Context, e Event) {
		received.Store(true)
		wg.Done()
	})

	bus.Publish(context.Background(), Event{Type: TypeHealthIssue})

	wg.Wait()

	if !received.Load() {
		t.Fatal("second handler should have been called despite first panicking")
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := New(slog.Default())
	// Should not panic
	bus.Publish(context.Background(), Event{Type: TypeMovieAdded})
}

func TestBus_TimestampPreserved(t *testing.T) {
	bus := New(slog.Default())

	ts := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	var gotTS time.Time

	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe(func(_ context.Context, e Event) {
		gotTS = e.Timestamp
		wg.Done()
	})

	bus.Publish(context.Background(), Event{Type: TypeMovieAdded, Timestamp: ts})

	wg.Wait()

	if !gotTS.Equal(ts) {
		t.Errorf("got timestamp %v, want %v", gotTS, ts)
	}
}

func TestBus_ContextNotCancelled(t *testing.T) {
	bus := New(slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var handlerCtxDone atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe(func(ctx context.Context, e Event) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			handlerCtxDone.Store(true)
		default:
			handlerCtxDone.Store(false)
		}
	})

	bus.Publish(ctx, Event{Type: TypeGrabStarted})

	wg.Wait()

	if handlerCtxDone.Load() {
		t.Error("handler context should not be cancelled (WithoutCancel)")
	}
}
