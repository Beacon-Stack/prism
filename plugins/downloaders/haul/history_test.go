package haul

// Tests for the history-lookup client wired in Phase 3. Mirrors
// pilot/plugins/downloaders/haul/history_test.go but uses Prism's
// narrower HistoryFilter (no series_id / episode_id / season /
// episode — those are TV-only).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── buildHistoryQueryString ──────────────────────────────────────────────────

func TestBuildHistoryQueryString_AllFields(t *testing.T) {
	q := buildHistoryQueryString(HistoryFilter{
		Service:        "prism",
		InfoHash:       "abc",
		MovieID:        "mov-1",
		TMDBID:         550,
		IncludeRemoved: true,
		Limit:          50,
	})
	for _, want := range []string{
		"service=prism",
		"info_hash=abc",
		"movie_id=mov-1",
		"tmdb_id=550",
		"include_removed=true",
		"limit=50",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("expected query to contain %q, got: %s", want, q)
		}
	}
}

func TestBuildHistoryQueryString_EmptyFilter(t *testing.T) {
	q := buildHistoryQueryString(HistoryFilter{})
	if q != "" {
		t.Errorf("empty filter should yield empty query; got %q", q)
	}
}

// IncludeRemoved=false (default) must NOT add the param so the server
// uses its own default. Pinning so a future "always send it"
// refactor doesn't accidentally send `include_removed=false`.
func TestBuildHistoryQueryString_IncludeRemovedFalseOmitted(t *testing.T) {
	q := buildHistoryQueryString(HistoryFilter{Service: "prism"})
	if strings.Contains(q, "include_removed") {
		t.Errorf("IncludeRemoved=false should NOT appear in query; got %s", q)
	}
}

// ── LookupHistory (integration-ish via httptest.Server) ─────────────────────

// Headline behaviour — happy-path lookup decodes the items array.
func TestLookupHistory_DecodesItemsArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/history" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []HistoryRecord{
				{InfoHash: "abc", Name: "Fight Club", Requester: "prism", TMDBID: 550, MovieID: "mov-1"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL})
	out, err := c.LookupHistory(context.Background(), HistoryFilter{
		Service: "prism",
		MovieID: "mov-1",
	})
	if err != nil {
		t.Fatalf("LookupHistory: %v", err)
	}
	if len(out) != 1 || out[0].InfoHash != "abc" || out[0].MovieID != "mov-1" {
		t.Errorf("got %+v", out)
	}
}

// Empty array returns a non-nil slice. Same UX rationale as Pilot:
// callers can `for _, x := range` without nil checks.
func TestLookupHistory_EmptyResultIsNonNilSlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []HistoryRecord{}})
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL})
	out, err := c.LookupHistory(context.Background(), HistoryFilter{Service: "prism"})
	if err != nil {
		t.Fatalf("LookupHistory: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(out) != 0 {
		t.Errorf("expected empty slice; got %d", len(out))
	}
}

// 404 from /by-hash/:hash → nil + nil error. UI doesn't toast.
func TestLookupHistoryByHash_404IsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL})
	rec, err := c.LookupHistoryByHash(context.Background(), "abc")
	if err != nil {
		t.Fatalf("404 must not be an error; got %v", err)
	}
	if rec != nil {
		t.Errorf("404 must return nil record; got %+v", rec)
	}
}

func TestLookupHistoryByHash_DecodesRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(HistoryRecord{InfoHash: "abc", Name: "Fight Club"})
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL})
	rec, err := c.LookupHistoryByHash(context.Background(), "abc")
	if err != nil {
		t.Fatalf("LookupHistoryByHash: %v", err)
	}
	if rec == nil || rec.InfoHash != "abc" {
		t.Errorf("got %+v", rec)
	}
}
