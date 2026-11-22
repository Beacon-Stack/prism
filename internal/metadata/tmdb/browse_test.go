package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

var today = time.Date(2026, 11, 22, 15, 0, 0, 0, time.UTC)

func TestBrowseQuery_Categories(t *testing.T) {
	cases := []struct {
		cat, path string
		want      map[string]string
	}{
		{"trending", "/trending/movie/week", map[string]string{"page": "1"}},
		{"popular", "/discover/movie", map[string]string{"sort_by": "popularity.desc"}},
		{"top-rated", "/discover/movie", map[string]string{"sort_by": "vote_average.desc", "vote_count.gte": "300"}},
		{"new", "/discover/movie", map[string]string{
			"with_release_type": "4", "region": "US",
			"release_date.gte": "2026-09-23", "release_date.lte": "2026-11-22",
		}},
		{"coming-soon", "/discover/movie", map[string]string{
			"primary_release_date.gte": "2026-11-23", "primary_release_date.lte": "2027-03-22",
		}},
	}
	for _, c := range cases {
		path, p, err := BrowseQuery(BrowseFilter{Category: c.cat, Today: today})
		if err != nil {
			t.Fatalf("%s: %v", c.cat, err)
		}
		if path != c.path {
			t.Errorf("%s: path %s, want %s", c.cat, path, c.path)
		}
		for k, v := range c.want {
			if p.Get(k) != v {
				t.Errorf("%s: %s=%q, want %q", c.cat, k, p.Get(k), v)
			}
		}
	}
}

func TestBrowseQuery_Filters(t *testing.T) {
	_, p, err := BrowseQuery(BrowseFilter{Category: "new", GenreID: 27, ProviderID: 8, MinRating: 7, Region: "gb", Page: 3, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"page": "3", "with_genres": "27", "with_watch_providers": "8", "watch_region": "GB", "region": "GB",
		"with_watch_monetization_types": "flatrate|free|ads", "vote_average.gte": "7.0", "vote_count.gte": "50",
	}
	for k, v := range want {
		if p.Get(k) != v {
			t.Errorf("%s=%q, want %q", k, p.Get(k), v)
		}
	}
	// Top rated keeps its own, higher vote floor.
	_, p, _ = BrowseQuery(BrowseFilter{Category: "top-rated", MinRating: 8, Today: today})
	if p.Get("vote_count.gte") != "300" {
		t.Errorf("top-rated vote floor = %s", p.Get("vote_count.gte"))
	}
	// Trending is a fixed list: filters don't apply.
	path, p, _ := BrowseQuery(BrowseFilter{Category: "trending", GenreID: 27, Today: today})
	if path != "/trending/movie/week" || p.Get("with_genres") != "" {
		t.Errorf("trending should ignore filters: %s %v", path, p)
	}
	if _, _, err := BrowseQuery(BrowseFilter{Category: "bogus"}); err == nil {
		t.Error("unknown category should error")
	}
}

func TestNormalizeRegion(t *testing.T) {
	for in, want := range map[string]string{"us": "US", "GB": "GB", "": "US", "usa": "US", "1x": "US"} {
		if got := NormalizeRegion(in); got != want {
			t.Errorf("NormalizeRegion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSortProviders(t *testing.T) {
	type raw = struct {
		ID                int            `json:"provider_id"`
		Name              string         `json:"provider_name"`
		LogoPath          string         `json:"logo_path"`
		DisplayPriorities map[string]int `json:"display_priorities"`
	}
	got := sortProviders("US", []raw{
		{ID: 1, Name: "Zed", DisplayPriorities: map[string]int{"GB": 0}},
		{ID: 8, Name: "Netflix", DisplayPriorities: map[string]int{"US": 0}},
		{ID: 9, Name: "Prime", DisplayPriorities: map[string]int{"US": 2}},
	})
	if got[0].ID != 8 || got[1].ID != 9 || got[2].ID != 1 {
		t.Errorf("order = %+v", got)
	}
}

func TestBrowse_CallsDiscoverAndParses(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		_, _ = w.Write([]byte(`{"page":2,"total_pages":9,"results":[{"id":11,"title":"New One","release_date":"2026-11-01","poster_path":"/p.jpg","vote_average":7.4}]}`))
	}))
	defer srv.Close()

	pr, err := newTestClient(srv.URL).Browse(context.Background(), BrowseFilter{Category: "new", Page: 2, ProviderID: 8, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/discover/movie" || gotQuery.Get("with_watch_providers") != "8" || gotQuery.Get("with_release_type") != "4" {
		t.Errorf("request %s %v", gotPath, gotQuery)
	}
	if pr.Page != 2 || pr.TotalPages != 9 || len(pr.Results) != 1 || pr.Results[0].Title != "New One" || pr.Results[0].Year != 2026 || pr.Results[0].Popularity != 7.4 {
		t.Errorf("parsed %+v", pr)
	}
}

func TestWatchProviders_SortedForRegion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/watch/providers/movie" || r.URL.Query().Get("watch_region") != "US" {
			t.Errorf("request %s %v", r.URL.Path, r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"results":[{"provider_id":9,"provider_name":"Prime Video","display_priorities":{"US":3}},{"provider_id":8,"provider_name":"Netflix","logo_path":"/n.jpg","display_priorities":{"US":1}}]}`))
	}))
	defer srv.Close()
	ps, err := newTestClient(srv.URL).WatchProviders(context.Background(), "us")
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "Netflix" || ps[0].LogoPath != "/n.jpg" {
		t.Errorf("providers %+v", ps)
	}
}

func TestSearchPaginated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/movie" || r.URL.Query().Get("query") != "dune" || r.URL.Query().Get("page") != "1" {
			t.Errorf("request %s %v", r.URL.Path, r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"results":[{"id":438631,"title":"Dune","release_date":"2021-09-15","vote_average":7.8}]}`))
	}))
	defer srv.Close()
	pr, err := newTestClient(srv.URL).SearchPaginated(context.Background(), "dune", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Results) != 1 || pr.Results[0].Popularity != 7.8 {
		t.Errorf("parsed %+v", pr)
	}
}
