package v1

import (
	"math/rand/v2"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

func TestPickN(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8}
	pick := func(seed uint64, n int) []int { return pickN(rand.New(rand.NewPCG(seed, seed)), items, n) }

	a, b := pick(7, 3), pick(7, 3)
	if len(a) != 3 || a[0] != b[0] || a[1] != b[1] || a[2] != b[2] {
		t.Errorf("same seed should pick the same: %v vs %v", a, b)
	}
	if a[0] == a[1] || a[1] == a[2] || a[0] == a[2] {
		t.Errorf("picked an item twice: %v", a)
	}
	if got := pick(7, 20); len(got) != len(items) {
		t.Errorf("asking for more than there is should return all, got %v", got)
	}
	// Different seeds reshuffle. Any one pair could match by chance, so
	// look across several.
	differs := false
	for seed := uint64(1); seed <= 10; seed++ {
		c := pick(seed, 3)
		if c[0] != a[0] || c[1] != a[1] || c[2] != a[2] {
			differs = true
		}
	}
	if !differs {
		t.Error("ten other seeds all picked the same three")
	}
}

func TestTopGenres(t *testing.T) {
	known := []tmdb.Genre{{ID: 28, Name: "Action"}, {ID: 35, Name: "Comedy"}, {ID: 18, Name: "Drama"}, {ID: 27, Name: "Horror"}}
	counts := map[string]int{"Drama": 9, "Action": 4, "Comedy": 4, "Home Video": 30}
	got := topGenres(counts, known, 3)
	if len(got) != 3 || got[0].ID != 18 || got[1].ID != 28 || got[2].ID != 35 {
		t.Errorf("want Drama, Action, Comedy (ties alphabetical; unknown and unused genres left out), got %+v", got)
	}
	if got := topGenres(nil, known, 3); len(got) != 0 {
		t.Errorf("empty library should have no top genres, got %+v", got)
	}
}

func TestInterleave(t *testing.T) {
	p := func(ids ...string) []shelfPlan {
		out := make([]shelfPlan, len(ids))
		for i, id := range ids {
			out[i] = shelfPlan{id: id}
		}
		return out
	}
	got := interleave(p("a1", "a2", "a3"), p("b1"), nil, p("c1", "c2"))
	want := []string{"a1", "b1", "c1", "a2", "c2", "a3"}
	if len(got) != len(want) {
		t.Fatalf("got %d plans, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].id != want[i] {
			t.Errorf("position %d = %s, want %s", i, got[i].id, want[i])
		}
	}
}

func TestFillShelves(t *testing.T) {
	m := func(id int) discoverResultBody { return discoverResultBody{TMDBID: id, PosterPath: "/p.jpg"} }
	owned, noPoster := m(90), discoverResultBody{TMDBID: 91}
	owned.InLibrary = true

	plans := []shelfPlan{
		{id: "first", minSize: 2},
		{id: "repeats", minSize: 2},
		{id: "failed", minSize: 2},
		{id: "collection"},
	}
	shelves := fillShelves(plans, [][]discoverResultBody{
		{m(1), owned, noPoster, m(2), m(3)},
		{m(1), m(2), m(4)}, // only 4 is new: too short for minSize 2
		nil,                // fetch failed
		{m(4)},             // 4 is still free, since the short row was dropped
	})
	if len(shelves) != 2 || shelves[0].ID != "first" || shelves[1].ID != "collection" {
		t.Fatalf("shelves = %+v", shelves)
	}
	if got := shelves[0].Results; len(got) != 3 || got[0].TMDBID != 1 || got[1].TMDBID != 2 || got[2].TMDBID != 3 {
		t.Errorf("first row should drop owned and posterless movies, got %+v", got)
	}
	if got := shelves[1].Results; len(got) != 1 || got[0].TMDBID != 4 {
		t.Errorf("collection row = %+v", got)
	}

	// A row never runs past shelfSize.
	long := make([]discoverResultBody, 0, shelfSize+5)
	for i := 1; i <= shelfSize+5; i++ {
		long = append(long, m(i))
	}
	if got := fillShelves([]shelfPlan{{id: "long"}}, [][]discoverResultBody{long}); len(got[0].Results) != shelfSize {
		t.Errorf("row length = %d, want %d", len(got[0].Results), shelfSize)
	}
}

func TestFilmographyResults_NewestFirst(t *testing.T) {
	got := filmographyResults([]tmdb.FilmographyItem{{TMDBID: 1, Year: 1999}, {TMDBID: 2, Year: 0}, {TMDBID: 3, Year: 2014}})
	if got[0].ID != 3 || got[1].ID != 1 || got[2].ID != 2 {
		t.Errorf("order = %+v", got)
	}
}

// Without TMDB every Discover endpoint answers 503, which is enough to
// check what the query parameters accept: a rejected one is a 422 instead.
func TestDiscoverBrowse_QueryParameters(t *testing.T) {
	_, api := humatest.New(t)
	RegisterDiscoverRoutes(api, nil, nil, nil)

	for path, want := range map[string]int{
		"/api/v1/discover/browse?category=popular&genre=28,35&sort=newest&decade=1990&runtime=long&language=ko&keyword=1&company=2": http.StatusServiceUnavailable,
		"/api/v1/discover/browse?genre=28":           http.StatusServiceUnavailable,
		"/api/v1/discover/browse?genre=action":       http.StatusUnprocessableEntity,
		"/api/v1/discover/browse?sort=alphabetical":  http.StatusUnprocessableEntity,
		"/api/v1/discover/browse?runtime=epic":       http.StatusUnprocessableEntity,
		"/api/v1/discover/browse?language=korean":    http.StatusUnprocessableEntity,
		"/api/v1/discover/shelves?seed=42&region=GB": http.StatusServiceUnavailable,
		"/api/v1/discover/suggest?q=nolan":           http.StatusServiceUnavailable,
	} {
		if got := api.Get(path).Code; got != want {
			t.Errorf("GET %s = %d, want %d", path, got, want)
		}
	}
}
