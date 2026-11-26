package movie_test

import (
	"context"
	"testing"

	"github.com/beacon-stack/prism/internal/core/movie"
)

func TestRecentlyAddedAndGenreCounts(t *testing.T) {
	meta := &mockTMDB{}
	svc, q := newTestService(t, meta)
	libraryID, profileID := seedTestFixtures(t, q)
	ctx := context.Background()

	if seeds, err := svc.RecentlyAdded(ctx, 10); err != nil || len(seeds) != 0 {
		t.Fatalf("empty library: seeds=%v err=%v", seeds, err)
	}
	if counts, err := svc.GenreCounts(ctx); err != nil || len(counts) != 0 {
		t.Fatalf("empty library: counts=%v err=%v", counts, err)
	}

	for _, m := range []struct {
		id     int
		title  string
		genres []string
	}{
		{550, "Fight Club", []string{"Drama", "Thriller"}},
		{155, "The Dark Knight", []string{"Action", "Drama"}},
		{13, "Forrest Gump", []string{"Drama", "Comedy"}},
	} {
		d := sampleDetail()
		d.ID, d.Title, d.OriginalTitle, d.Genres, d.IMDBId = m.id, m.title, m.title, m.genres, ""
		meta.movieDetail = d
		if _, err := svc.Add(ctx, movie.AddRequest{TMDBID: m.id, LibraryID: libraryID, QualityProfileID: profileID, Monitored: true}); err != nil {
			t.Fatalf("adding %s: %v", m.title, err)
		}
	}

	seeds, err := svc.RecentlyAdded(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	titles := map[int]string{}
	for _, s := range seeds {
		if s.ID == "" {
			t.Errorf("seed %d has no library ID", s.TMDBID)
		}
		titles[s.TMDBID] = s.Title
	}
	if len(seeds) != 3 || titles[550] != "Fight Club" || titles[155] != "The Dark Knight" || titles[13] != "Forrest Gump" {
		t.Errorf("seeds = %+v", seeds)
	}
	if limited, _ := svc.RecentlyAdded(ctx, 2); len(limited) != 2 {
		t.Errorf("limit 2 returned %d", len(limited))
	}

	counts, err := svc.GenreCounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["Drama"] != 3 || counts["Action"] != 1 || counts["Comedy"] != 1 || counts["Thriller"] != 1 || len(counts) != 4 {
		t.Errorf("counts = %v", counts)
	}
}
