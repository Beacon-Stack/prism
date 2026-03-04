package blocklist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/davidfic/luminarr/internal/core/blocklist"
	"github.com/davidfic/luminarr/internal/testutil"
)

func newSvc(t *testing.T) (*blocklist.Service, string) {
	t.Helper()
	q := testutil.NewTestDB(t)
	movie := testutil.SeedMovie(t, q)
	return blocklist.NewService(q), movie.ID
}

func TestAdd_IsBlocklisted(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	ok, err := svc.IsBlocklisted(ctx, "guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not blocklisted before add")
	}

	if err := svc.Add(ctx, movieID, "guid-1", "Release Title", "", "torrent", 1_000_000, "grab failed"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	ok, err = svc.IsBlocklisted(ctx, "guid-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected blocklisted after add")
	}
}

func TestIsBlocklisted_Unknown(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc(t)

	ok, err := svc.IsBlocklisted(ctx, "unknown-guid")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not blocklisted for unknown GUID")
	}
}

func TestAdd_DuplicateGUID(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	if err := svc.Add(ctx, movieID, "guid-dup", "Title", "", "torrent", 0, ""); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	err := svc.Add(ctx, movieID, "guid-dup", "Title", "", "torrent", 0, "")
	if !errors.Is(err, blocklist.ErrAlreadyBlocklisted) {
		t.Fatalf("expected ErrAlreadyBlocklisted, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	if err := svc.Add(ctx, movieID, "guid-del", "Delete Me", "", "torrent", 0, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, _, err := svc.List(ctx, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if err := svc.Delete(ctx, entries[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	ok, err := svc.IsBlocklisted(ctx, "guid-del")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not blocklisted after delete")
	}
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	for i, guid := range []string{"g1", "g2", "g3"} {
		title := "Release"
		_ = i
		if err := svc.Add(ctx, movieID, guid, title, "", "torrent", 0, ""); err != nil {
			t.Fatalf("Add %s: %v", guid, err)
		}
	}

	_, total, err := svc.List(ctx, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("expected 3 entries before clear, got %d", total)
	}

	if err := svc.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	_, total, err = svc.List(ctx, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", total)
	}
}

func TestList_Pagination(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	guids := []string{"p1", "p2", "p3", "p4", "p5"}
	for _, guid := range guids {
		if err := svc.Add(ctx, movieID, guid, "Title "+guid, "", "torrent", 0, ""); err != nil {
			t.Fatalf("Add %s: %v", guid, err)
		}
	}

	page1, total, err := svc.List(ctx, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("expected total=5, got %d", total)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 items on page 1, got %d", len(page1))
	}

	page2, total2, err := svc.List(ctx, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if total2 != 5 {
		t.Fatalf("expected total=5 on page 2, got %d", total2)
	}
	if len(page2) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(page2))
	}
}

func TestList_MovieTitleJoined(t *testing.T) {
	ctx := context.Background()
	svc, movieID := newSvc(t)

	if err := svc.Add(ctx, movieID, "guid-title", "Release", "", "torrent", 0, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, _, err := svc.List(ctx, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	if entries[0].MovieTitle == "" {
		t.Error("expected MovieTitle to be joined from movies table")
	}
}
