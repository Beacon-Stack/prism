package queue_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/beacon-stack/prism/internal/core/queue"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// testLogger returns a slog.Logger that discards all output so tests don't
// spam stdout but the service code (which unconditionally logs at various
// levels) doesn't panic on a nil pointer.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ── Mocks ────────────────────────────────────────────────────────────────────

// stubQuerier implements just the methods queue.PollAndUpdate calls —
// ListActiveGrabs, UpdateGrabStatus, MarkGrabRemoved — and records what
// happens so tests can assert on the sync side effects. Any other querier
// method panics via the embedded nil interface.
type stubQuerier struct {
	dbgen.Querier

	activeGrabs []dbgen.GrabHistory

	// Recorded calls
	updateStatusCalls []dbgen.UpdateGrabStatusParams
	markRemovedCalls  []string

	// Optional error injection
	markRemovedErr error
}

func (s *stubQuerier) ListActiveGrabs(_ context.Context) ([]dbgen.GrabHistory, error) {
	return s.activeGrabs, nil
}

func (s *stubQuerier) UpdateGrabStatus(_ context.Context, arg dbgen.UpdateGrabStatusParams) error {
	s.updateStatusCalls = append(s.updateStatusCalls, arg)
	return nil
}

func (s *stubQuerier) MarkGrabRemoved(_ context.Context, id string) error {
	s.markRemovedCalls = append(s.markRemovedCalls, id)
	return s.markRemovedErr
}

// stubClient implements plugin.DownloadClient. Only the methods the queue
// service calls (Status) are driven by the test; the rest return empty.
type stubClient struct {
	statusResult plugin.QueueItem
	statusErr    error
	statusCalls  int
}

func (c *stubClient) Name() string              { return "stub" }
func (c *stubClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (c *stubClient) Add(_ context.Context, _ plugin.Release) (string, error) {
	return "", nil
}
func (c *stubClient) Status(_ context.Context, _ string) (plugin.QueueItem, error) {
	c.statusCalls++
	return c.statusResult, c.statusErr
}
func (c *stubClient) GetQueue(_ context.Context) ([]plugin.QueueItem, error) { return nil, nil }
func (c *stubClient) Remove(_ context.Context, _ string, _ bool) error       { return nil }
func (c *stubClient) Test(_ context.Context) error                           { return nil }

// stubDownloader implements queue.DownloaderClient — just returns the one
// canned plugin client for every lookup.
type stubDownloader struct {
	client plugin.DownloadClient
}

func (d *stubDownloader) ClientFor(_ context.Context, _ string) (plugin.DownloadClient, error) {
	return d.client, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func activeGrab(id, movieID string) dbgen.GrabHistory {
	return dbgen.GrabHistory{
		ID:               id,
		MovieID:          movieID,
		ReleaseTitle:     "Arrival 2016 PROPER 2160p BluRay REMUX",
		DownloadClientID: sql.NullString{String: "haul-client", Valid: true},
		ClientItemID:     sql.NullString{String: "abc123infohash", Valid: true},
		DownloadStatus:   "downloading",
		DownloadedBytes:  1000,
		GrabbedAt:        "2026-04-13T10:23:36Z",
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

// TestPollAndUpdate_ItemNotFoundMarksGrabRemoved is the headline regression
// test for the "can't grab another release for this movie" bug. When a user
// removes a torrent directly in Haul (or any other client), the client starts
// returning plugin.ErrItemNotFound for subsequent Status calls. The queue
// sync must detect this and mark the grab as 'removed' so the partial unique
// index idx_grab_history_active_movie frees up and the user can grab a
// different release without manually editing the database.
//
// Before this fix, the queue service silently continued past the error,
// leaving grab_history in status='downloading' forever. The resulting
// user-visible symptom was a 409 on every subsequent grab attempt:
//
//	"already downloading another release for this movie (…)"
//
// If this test fails, that bug is back.
func TestPollAndUpdate_ItemNotFoundMarksGrabRemoved(t *testing.T) {
	q := &stubQuerier{
		activeGrabs: []dbgen.GrabHistory{activeGrab("g1", "m1")},
	}
	c := &stubClient{
		// Wrap the sentinel the same way the real haul plugin does, so
		// we're also testing that errors.Is unwrapping still works.
		statusErr: errors.Join(errors.New("haul: extra context"), plugin.ErrItemNotFound),
	}
	svc := queue.NewService(q, &stubDownloader{client: c}, nil, testLogger())

	if err := svc.PollAndUpdate(context.Background()); err != nil {
		t.Fatalf("PollAndUpdate returned error: %v", err)
	}

	if len(q.markRemovedCalls) != 1 {
		t.Fatalf("expected exactly one MarkGrabRemoved call, got %d", len(q.markRemovedCalls))
	}
	if q.markRemovedCalls[0] != "g1" {
		t.Errorf("expected MarkGrabRemoved('g1'), got %q", q.markRemovedCalls[0])
	}
	if len(q.updateStatusCalls) != 0 {
		t.Errorf("expected no UpdateGrabStatus calls, got %d", len(q.updateStatusCalls))
	}
}

// TestPollAndUpdate_TransientErrorKeepsGrab asserts that a transient network
// error (anything that is NOT plugin.ErrItemNotFound) leaves the grab alone
// so the next poll cycle can retry. Without this, a brief haul restart would
// nuke the queue and the user would lose tracked downloads.
func TestPollAndUpdate_TransientErrorKeepsGrab(t *testing.T) {
	q := &stubQuerier{
		activeGrabs: []dbgen.GrabHistory{activeGrab("g1", "m1")},
	}
	c := &stubClient{
		statusErr: errors.New("haul: connection refused"),
	}
	svc := queue.NewService(q, &stubDownloader{client: c}, nil, testLogger())

	if err := svc.PollAndUpdate(context.Background()); err != nil {
		t.Fatalf("PollAndUpdate returned error: %v", err)
	}

	if len(q.markRemovedCalls) != 0 {
		t.Errorf("expected NO MarkGrabRemoved calls for transient error, got %d", len(q.markRemovedCalls))
	}
	if len(q.updateStatusCalls) != 0 {
		t.Errorf("expected NO UpdateGrabStatus calls for transient error, got %d", len(q.updateStatusCalls))
	}
	if c.statusCalls != 1 {
		t.Errorf("expected exactly one Status call, got %d", c.statusCalls)
	}
}

// TestPollAndUpdate_NormalStatusUpdate asserts the happy path: when the
// client returns a valid QueueItem with a new status, the grab's
// download_status and downloaded_bytes are persisted via UpdateGrabStatus.
// No MarkGrabRemoved in this path.
func TestPollAndUpdate_NormalStatusUpdate(t *testing.T) {
	q := &stubQuerier{
		activeGrabs: []dbgen.GrabHistory{activeGrab("g1", "m1")},
	}
	c := &stubClient{
		statusResult: plugin.QueueItem{
			ClientItemID: "abc123infohash",
			Status:       plugin.StatusCompleted,
			Downloaded:   5000,
			ContentPath:  "/downloads/arrival/file.mkv",
		},
	}
	svc := queue.NewService(q, &stubDownloader{client: c}, nil, testLogger())

	if err := svc.PollAndUpdate(context.Background()); err != nil {
		t.Fatalf("PollAndUpdate returned error: %v", err)
	}

	if len(q.markRemovedCalls) != 0 {
		t.Errorf("expected NO MarkGrabRemoved calls for normal update, got %d", len(q.markRemovedCalls))
	}
	if len(q.updateStatusCalls) != 1 {
		t.Fatalf("expected exactly one UpdateGrabStatus call, got %d", len(q.updateStatusCalls))
	}
	call := q.updateStatusCalls[0]
	if call.ID != "g1" {
		t.Errorf("UpdateGrabStatus ID = %q, want g1", call.ID)
	}
	if call.DownloadStatus != "completed" {
		t.Errorf("UpdateGrabStatus DownloadStatus = %q, want completed", call.DownloadStatus)
	}
	if call.DownloadedBytes != 5000 {
		t.Errorf("UpdateGrabStatus DownloadedBytes = %d, want 5000", call.DownloadedBytes)
	}
}

// TestPollAndUpdate_NoChangeSkipsUpdate asserts the "status unchanged" fast
// path: if the client reports the same status and downloaded bytes as the
// cached grab, no UpdateGrabStatus is issued (we avoid unnecessary DB writes).
func TestPollAndUpdate_NoChangeSkipsUpdate(t *testing.T) {
	g := activeGrab("g1", "m1")
	q := &stubQuerier{activeGrabs: []dbgen.GrabHistory{g}}
	c := &stubClient{
		statusResult: plugin.QueueItem{
			ClientItemID: "abc123infohash",
			Status:       plugin.DownloadStatus(g.DownloadStatus), // same "downloading"
			Downloaded:   g.DownloadedBytes,                       // same 1000
		},
	}
	svc := queue.NewService(q, &stubDownloader{client: c}, nil, testLogger())

	if err := svc.PollAndUpdate(context.Background()); err != nil {
		t.Fatalf("PollAndUpdate returned error: %v", err)
	}

	if len(q.updateStatusCalls) != 0 {
		t.Errorf("expected NO UpdateGrabStatus calls when nothing changed, got %d",
			len(q.updateStatusCalls))
	}
}

// TestPollAndUpdate_MultipleGrabsIndependentFates exercises the case where
// one grab is gone from the client (should be marked removed) and another
// grab for a different movie is still healthy (should be updated normally).
// The two should be handled independently — one bad grab must not abort the
// sync for the rest.
func TestPollAndUpdate_MultipleGrabsIndependentFates(t *testing.T) {
	// Two grabs on the same client. The stubClient only lets us return one
	// result for both, so we use a client that branches on item ID.
	c := &branchingClient{
		results: map[string]plugin.QueueItem{
			"hash-ok": {
				ClientItemID: "hash-ok",
				Status:       plugin.StatusCompleted,
				Downloaded:   9999,
			},
		},
		errors: map[string]error{
			"hash-gone": plugin.ErrItemNotFound,
		},
	}

	grabOK := activeGrab("g-ok", "movie-ok")
	grabOK.ClientItemID = sql.NullString{String: "hash-ok", Valid: true}

	grabGone := activeGrab("g-gone", "movie-gone")
	grabGone.ClientItemID = sql.NullString{String: "hash-gone", Valid: true}

	q := &stubQuerier{activeGrabs: []dbgen.GrabHistory{grabOK, grabGone}}
	svc := queue.NewService(q, &stubDownloader{client: c}, nil, testLogger())

	if err := svc.PollAndUpdate(context.Background()); err != nil {
		t.Fatalf("PollAndUpdate returned error: %v", err)
	}

	if len(q.updateStatusCalls) != 1 || q.updateStatusCalls[0].ID != "g-ok" {
		t.Errorf("expected UpdateGrabStatus for g-ok, got %+v", q.updateStatusCalls)
	}
	if len(q.markRemovedCalls) != 1 || q.markRemovedCalls[0] != "g-gone" {
		t.Errorf("expected MarkGrabRemoved for g-gone, got %v", q.markRemovedCalls)
	}
}

// branchingClient returns different results per item ID so we can exercise
// the "one good, one not-found" case.
type branchingClient struct {
	stubClient
	results map[string]plugin.QueueItem
	errors  map[string]error
}

func (c *branchingClient) Status(_ context.Context, id string) (plugin.QueueItem, error) {
	if err, ok := c.errors[id]; ok {
		return plugin.QueueItem{}, err
	}
	return c.results[id], nil
}
