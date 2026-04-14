package indexer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/beacon-stack/prism/internal/core/indexer"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
)

// activeGrabStub implements just ListGrabHistoryByMovie so we can test
// ActiveGrabForMovie's filter logic in isolation, without requiring a real
// Postgres instance. Embeds dbgen.Querier — any other method call panics,
// which is fine because the function under test only calls the one query.
type activeGrabStub struct {
	dbgen.Querier
	rows []dbgen.GrabHistory
	err  error
}

func (s *activeGrabStub) ListGrabHistoryByMovie(_ context.Context, _ string) ([]dbgen.GrabHistory, error) {
	return s.rows, s.err
}

func makeGrab(id, status string) dbgen.GrabHistory {
	return dbgen.GrabHistory{ID: id, MovieID: "m1", DownloadStatus: status}
}

// TestActiveGrabForMovie pins the contract that ActiveGrabForMovie returns
// exactly the grab that the partial unique index idx_grab_history_active_movie
// permits per movie — i.e. any row whose download_status is NOT one of the
// terminal states (completed, failed, removed).
//
// This is the regression guard for the manual-grab 500 → 409 fix in
// internal/api/v1/releases.go. If the active-grab definition here drifts
// away from the WHERE clause on the unique index, the precheck will stop
// matching what the constraint enforces, and grab failures will start
// surfacing as raw SQL errors again.
func TestActiveGrabForMovie(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		rows   []dbgen.GrabHistory
		wantID string // "" means expect nil
	}{
		{
			name:   "no history",
			rows:   nil,
			wantID: "",
		},
		{
			name: "only completed",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "completed"),
			},
			wantID: "",
		},
		{
			name: "only terminal mix",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "completed"),
				makeGrab("g2", "failed"),
				makeGrab("g3", "removed"),
			},
			wantID: "",
		},
		{
			name: "single queued grab",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "queued"),
			},
			wantID: "g1",
		},
		{
			name: "single downloading grab",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "downloading"),
			},
			wantID: "g1",
		},
		{
			name: "single seeding grab is still active",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "seeding"),
			},
			wantID: "g1",
		},
		{
			name: "active grab returned even when terminal entries exist",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "completed"),
				makeGrab("g2", "failed"),
				makeGrab("g3", "queued"),
			},
			wantID: "g3",
		},
		{
			name: "first non-terminal wins when multiple are present",
			rows: []dbgen.GrabHistory{
				makeGrab("g1", "queued"),
				makeGrab("g2", "downloading"),
			},
			wantID: "g1",
		},
		{
			name: "unknown status is treated as active",
			rows: []dbgen.GrabHistory{
				// Defensive: anything we don't recognise as terminal is
				// active. Matches the WHERE clause's NOT IN semantics.
				makeGrab("g1", "weird-future-state"),
			},
			wantID: "g1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := indexer.NewService(&activeGrabStub{rows: tc.rows}, nil, nil, nil)
			got, err := svc.ActiveGrabForMovie(ctx, "m1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantID == "" {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected grab %q, got nil", tc.wantID)
			}
			if got.ID != tc.wantID {
				t.Errorf("expected ID %q, got %q", tc.wantID, got.ID)
			}
		})
	}
}

func TestActiveGrabForMovie_QuerierError(t *testing.T) {
	ctx := context.Background()
	want := errors.New("boom")
	svc := indexer.NewService(&activeGrabStub{err: want}, nil, nil, nil)
	_, err := svc.ActiveGrabForMovie(ctx, "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped %v, got %v", want, err)
	}
}
