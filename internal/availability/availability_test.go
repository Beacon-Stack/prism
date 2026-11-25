package availability

import (
	"context"
	"errors"
	"testing"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/queue"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func TestDecide(t *testing.T) {
	cases := []struct {
		name string
		f    Facts
		want State
	}{
		{"in the media server", Facts{Status: "downloaded", InMediaServer: true}, Available},
		{"in the media server, no file Prism can see", Facts{Status: "released", Monitored: true, InMediaServer: true}, Available},
		{"in the media server while an upgrade downloads", Facts{Status: "downloaded", InMediaServer: true, Downloading: true}, Available},
		{"downloading", Facts{Status: "released", Monitored: true, Downloading: true}, Downloading},
		{"on disk, not in the media server yet", Facts{Status: "downloaded", Monitored: true}, Downloaded},
		{"file gone", Facts{Status: "missing", Monitored: true}, Missing},
		{"not out yet", Facts{Status: "announced", Monitored: true}, Upcoming},
		{"released, nothing yet", Facts{Status: "released", Monitored: true}, Wanted},
		{"released, not monitored", Facts{Status: "released"}, Unmonitored},
	}
	for _, c := range cases {
		if got := Decide(c.f); got != c.want {
			t.Errorf("%s: Decide = %q, want %q", c.name, got, c.want)
		}
	}
}

type fakeLister struct {
	items []plugin.MediaItem
	err   error
}

func (f *fakeLister) ListMovieItems(context.Context) ([]plugin.MediaItem, error) {
	return f.items, f.err
}

type fakeServers struct{ listers []plugin.MovieLister }

func (f *fakeServers) MovieListers(context.Context) ([]plugin.MovieLister, error) {
	return f.listers, nil
}

type fakeQueue struct{ items []queue.Item }

func (f *fakeQueue) GetQueue(context.Context) ([]queue.Item, error) { return f.items, nil }

func TestFor(t *testing.T) {
	plex := &fakeLister{items: []plugin.MediaItem{
		{TMDBID: 100, Title: "In Plex", Year: 2020},
		{Title: "Crouching Tiger, Hidden Dragon", Year: 2000}, // no TMDB ID
	}}
	q := &fakeQueue{items: []queue.Item{
		{MovieID: "dl", Status: "downloading", Size: 1000, DownloadedBytes: 250},
		{MovieID: "failed", Status: "failed", Size: 1000, DownloadedBytes: 900},
	}}
	svc := NewService(&fakeServers{listers: []plugin.MovieLister{plex}}, q, nil)
	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	movies := []movie.Movie{
		{ID: "plex", TMDBID: 100, Title: "In Plex", Year: 2020, Status: "downloaded"},
		{ID: "bytitle", TMDBID: 146, Title: "Crouching Tiger Hidden Dragon", Year: 2000, Status: "downloaded"},
		{ID: "disk", TMDBID: 200, Title: "Only On Disk", Year: 2021, Status: "downloaded"},
		{ID: "dl", TMDBID: 300, Title: "On Its Way", Year: 2022, Status: "released", Monitored: true},
		{ID: "failed", TMDBID: 400, Title: "Failed Grab", Year: 2023, Status: "released", Monitored: true},
	}
	got := svc.For(context.Background(), movies)

	want := map[string]State{"plex": Available, "bytitle": Available, "disk": Downloaded, "dl": Downloading, "failed": Wanted}
	for id, state := range want {
		if got[id].State != state {
			t.Errorf("%s: state = %q, want %q", id, got[id].State, state)
		}
	}
	if got["dl"].Progress != 0.25 {
		t.Errorf("download progress = %v, want 0.25", got["dl"].Progress)
	}
}

// With no media server set up, a file on disk is as far as a movie goes.
func TestFor_NoMediaServer(t *testing.T) {
	svc := NewService(&fakeServers{}, nil, nil)
	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	got := svc.For(context.Background(), []movie.Movie{{ID: "a", TMDBID: 1, Status: "downloaded"}})
	if got["a"].State != Downloaded {
		t.Errorf("state = %q, want downloaded", got["a"].State)
	}
}

// A media server that is briefly unreachable must not turn every movie
// back to "downloaded".
func TestRefresh_KeepsLastKnownContentsOnFailure(t *testing.T) {
	plex := &fakeLister{items: []plugin.MediaItem{{TMDBID: 100}}}
	svc := NewService(&fakeServers{listers: []plugin.MovieLister{plex}}, nil, nil)
	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	plex.items, plex.err = nil, errors.New("connection refused")
	if err := svc.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh with a failing server = nil, want an error")
	}
	if !svc.InMediaServer(movie.Movie{TMDBID: 100}) {
		t.Error("movie no longer reported as in the media server after a failed refresh")
	}
}
