package stats_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/stats"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/internal/testutil"
	"github.com/beacon-stack/prism/pkg/plugin"
)

type fixture struct {
	t        *testing.T
	q        *dbgen.Queries
	svc      *stats.Service
	movieSvc *movie.Service
	lib      dbgen.Library
	nextTMDB int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	movieSvc := movie.NewService(q, nil, events.New(logger), logger)
	return &fixture{t: t, q: q, svc: stats.NewService(q, movieSvc), movieSvc: movieSvc, lib: testutil.SeedLibrary(t, q), nextTMDB: 1000}
}

// movie seeds a movie; runtime 0 leaves the runtime unknown.
func (f *fixture) movie(title string, year, runtime int64, genres, addedAt string) dbgen.Movie {
	f.t.Helper()
	f.nextTMDB++
	return testutil.SeedMovieInLibrary(f.t, f.q, f.lib.ID, f.lib.DefaultQualityProfileID, testutil.WithTMDBID(f.nextTMDB), func(p *dbgen.CreateMovieParams) {
		p.Title, p.Year, p.GenresJson = title, year, genres
		if runtime > 0 {
			p.RuntimeMinutes = &runtime
		}
		if addedAt != "" {
			p.AddedAt = addedAt
		}
	})
}

func (f *fixture) file(m dbgen.Movie, name string, bytes int64, resolution string) {
	f.t.Helper()
	if err := f.movieSvc.AttachFile(context.Background(), m.ID, "/movies/"+name, bytes, plugin.Quality{Resolution: plugin.Resolution(resolution)}); err != nil {
		f.t.Fatalf("AttachFile %s: %v", name, err)
	}
}

func (f *fixture) watch(m dbgen.Movie, at time.Time, user string) {
	f.t.Helper()
	if err := f.q.InsertWatchEvent(context.Background(), dbgen.InsertWatchEventParams{
		ID: uuid.New().String(), MovieID: m.ID, TmdbID: m.TmdbID, WatchedAt: at.UTC().Format(time.RFC3339), UserName: user, Source: "plex",
	}); err != nil {
		f.t.Fatalf("InsertWatchEvent: %v", err)
	}
}

const gb = int64(1_000_000_000)

func TestStorageBreakdown(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if b, err := f.svc.StorageBreakdown(ctx); err != nil || b.Total.Bytes != 0 || len(b.Largest) != 0 || len(b.ByResolution) != 0 {
		t.Fatalf("empty library: %+v err=%v", b, err)
	}

	heat := f.movie("Heat", 1995, 170, `["Crime","Drama"]`, "")
	f.file(heat, "heat-4k.mkv", 60*gb, "2160p")
	f.file(heat, "heat-hd.mkv", 10*gb, "1080p") // a second copy of the same movie
	dune := f.movie("Dune", 2021, 155, `["Sci-Fi","Drama"]`, "")
	f.file(dune, "dune.mkv", 40*gb, "2160p")
	clerks := f.movie("Clerks", 1994, 0, `["Comedy"]`, "") // runtime unknown
	f.file(clerks, "clerks.mkv", 5*gb, "1080p")
	f.movie("No File Yet", 2024, 100, `["Drama"]`, "")

	b, err := f.svc.StorageBreakdown(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if b.Total.Bytes != 115*gb || b.Total.Movies != 3 {
		t.Errorf("total = %+v", b.Total)
	}
	// Clerks has no runtime, so it is left out of the bytes-per-hour figures.
	if b.Total.TimedBytes != 110*gb || b.Total.RuntimeMinutes != 170+170+155 {
		t.Errorf("timed total = %d bytes over %d min", b.Total.TimedBytes, b.Total.RuntimeMinutes)
	}

	find := func(slices []stats.StorageSlice, label string) stats.StorageSlice {
		for _, s := range slices {
			if s.Label == label {
				return s
			}
		}
		t.Errorf("no slice %q in %+v", label, slices)
		return stats.StorageSlice{}
	}
	if b.ByResolution[0].Label != "2160p" || b.ByResolution[1].Label != "1080p" {
		t.Errorf("resolutions should be largest first: %+v", b.ByResolution)
	}
	if s := find(b.ByResolution, "2160p"); s.Bytes != 100*gb || s.Movies != 2 {
		t.Errorf("2160p = %+v", s)
	}
	if s := find(b.ByResolution, "1080p"); s.Bytes != 15*gb || s.Movies != 2 || s.TimedBytes != 10*gb {
		t.Errorf("1080p = %+v", s)
	}
	// A movie counts toward each of its genres, once however many files it has.
	if s := find(b.ByGenre, "Drama"); s.Bytes != 110*gb || s.Movies != 2 {
		t.Errorf("Drama = %+v", s)
	}
	if b.ByGenre[0].Label != "Drama" {
		t.Errorf("genres should be largest first: %+v", b.ByGenre)
	}
	if len(b.ByDecade) != 2 || b.ByDecade[0].Label != "1990s" || b.ByDecade[0].Bytes != 75*gb || b.ByDecade[1].Label != "2020s" {
		t.Errorf("decades should be oldest first: %+v", b.ByDecade)
	}

	if len(b.Largest) != 3 || b.Largest[0].Title != "Heat" || b.Largest[0].Bytes != 70*gb || b.Largest[0].Resolution != "2160p" {
		t.Errorf("largest[0] should be Heat, both files, labelled by its bigger one: %+v", b.Largest)
	}
	if b.Largest[1].Title != "Dune" || b.Largest[2].Title != "Clerks" {
		t.Errorf("largest order = %+v", b.Largest)
	}
	// The test library's root (/movies) doesn't exist, so there is no disk to measure.
	if b.Forecast != nil {
		t.Errorf("forecast without a measurable disk: %+v", b.Forecast)
	}
}

// scan records what ffprobe found in the movie's one file.
func (f *fixture) scan(m dbgen.Movie, mediainfoJSON string) {
	f.t.Helper()
	ctx := context.Background()
	files, err := f.q.ListMovieFiles(ctx, m.ID)
	if err != nil || len(files) != 1 {
		f.t.Fatalf("ListMovieFiles: %d files, err=%v", len(files), err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := f.q.UpdateMovieFileMediainfo(ctx, dbgen.UpdateMovieFileMediainfoParams{MediainfoJson: mediainfoJSON, MediainfoScannedAt: &now, ID: files[0].ID}); err != nil {
		f.t.Fatalf("UpdateMovieFileMediainfo: %v", err)
	}
}

// A file renamed to "Title (Year).mkv" has no quality from its name; the
// statistics take what the ffprobe scan found instead.
func TestStats_UnknownQualityFilledFromScan(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	renamed := f.movie("The Thing", 1982, 109, `["Horror"]`, "")
	f.file(renamed, "The Thing (1982).mkv", 80*gb, "")
	f.scan(renamed, `{"codec":"x265","resolution":"2160p","hdr_format":"Dolby Vision"}`)

	named := f.movie("Heat", 1995, 170, `["Crime"]`, "")
	f.file(named, "Heat.1995.1080p.BluRay.mkv", 20*gb, "1080p")
	f.scan(named, `{"codec":"x264","resolution":"720p","hdr_format":"SDR"}`) // the name's 1080p stands

	lowRes := f.movie("Clerks", 1994, 92, `["Comedy"]`, "")
	f.file(lowRes, "Clerks (1994).avi", 1*gb, "")
	f.scan(lowRes, `{"codec":"XviD","resolution":"SD","hdr_format":"SDR"}`)

	unscanned := f.movie("Dune", 2021, 155, `["Sci-Fi"]`, "")
	f.file(unscanned, "Dune (2021).mkv", 40*gb, "")

	b, err := f.svc.StorageBreakdown(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range b.Largest {
		got[m.Title] = m.Resolution
	}
	want := map[string]string{"The Thing": "2160p", "Heat": "1080p", "Clerks": "sd", "Dune": "unknown"}
	for title, res := range want {
		if got[title] != res {
			t.Errorf("largest: %s is %q, want %q", title, got[title], res)
		}
	}
	bytes := map[string]int64{}
	for _, s := range b.ByResolution {
		bytes[s.Label] = s.Bytes
	}
	if bytes["2160p"] != 80*gb || bytes["1080p"] != 20*gb || bytes["sd"] != 1*gb || bytes["unknown"] != 40*gb {
		t.Errorf("by resolution = %v", bytes)
	}

	// The Quality Distribution card and its tier filter agree.
	buckets, err := f.svc.QualityDistribution(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, q := range buckets {
		if q.Resolution == "2160p" {
			found = true
			if q.Codec != "x265" || q.HDR != "dolby_vision" || q.Source != "unknown" || q.Count != 1 {
				t.Errorf("2160p bucket = %+v: codec and HDR come from the scan, the source can't", q)
			}
		}
		// Heat's name gave only the resolution; the scan supplies the rest.
		if q.Resolution == "1080p" && (q.Codec != "x264" || q.HDR != "none") {
			t.Errorf("1080p bucket = %+v, want codec x264 and no HDR from the scan", q)
		}
	}
	if !found {
		t.Errorf("no 2160p bucket in %+v", buckets)
	}

	ids, err := f.svc.MovieIDsByQualityTier(ctx, "2160p", "")
	if err != nil || len(ids) != 1 || ids[0] != renamed.ID {
		t.Errorf("2160p tier movies = %v err=%v", ids, err)
	}
	tiers, err := f.svc.QualityTiers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tierCount := map[string]int64{}
	for _, tr := range tiers {
		tierCount[tr.Resolution] += tr.Count
	}
	if tierCount["2160p"] != 1 || tierCount["unknown"] != 1 {
		t.Errorf("tiers = %+v", tiers)
	}
}

func TestStorageBreakdown_ListsAreCapped(t *testing.T) {
	f := newFixture(t)
	for i := range 14 {
		m := f.movie(fmt.Sprintf("Movie %02d", i), 2000, 100, fmt.Sprintf(`["Genre %02d"]`, i), "")
		f.file(m, fmt.Sprintf("m%d.mkv", i), int64(i+1)*gb, "1080p")
	}
	b, err := f.svc.StorageBreakdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Largest) != 10 || b.Largest[0].Title != "Movie 13" {
		t.Errorf("largest: %d entries, first %q", len(b.Largest), b.Largest[0].Title)
	}
	if len(b.ByGenre) != 10 || b.ByGenre[0].Label != "Genre 13" {
		t.Errorf("genres: %d entries, first %q", len(b.ByGenre), b.ByGenre[0].Label)
	}
}

func TestForecast(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	at := func(daysAgo int, bytes int64) stats.StoragePoint {
		return stats.StoragePoint{CapturedAt: now.AddDate(0, 0, -daysAgo), TotalBytes: bytes}
	}

	// 30 GB in 30 days, 100 GB free: full in 100 days.
	f := stats.Forecast([]stats.StoragePoint{at(30, 500*gb), at(15, 520*gb), at(0, 530*gb)}, 100*gb, now)
	if f == nil || f.BytesPerDay != gb || f.FreeBytes != 100*gb || f.DaysUntilFull == nil || *f.DaysUntilFull != 100 {
		t.Fatalf("growing: %+v", f)
	}
	if want := now.AddDate(0, 0, 100); !f.FullOn.Equal(want) {
		t.Errorf("full on %v, want %v", f.FullOn, want)
	}

	// Shrinking or flat: a rate, but no date.
	for name, points := range map[string][]stats.StoragePoint{
		"shrinking": {at(30, 500*gb), at(0, 470*gb)},
		"flat":      {at(30, 500*gb), at(0, 500*gb)},
	} {
		f := stats.Forecast(points, 100*gb, now)
		if f == nil || f.BytesPerDay > 0 || f.DaysUntilFull != nil || f.FullOn != nil {
			t.Errorf("%s: %+v", name, f)
		}
	}

	// Too little history to say anything.
	for name, points := range map[string][]stats.StoragePoint{
		"none":       nil,
		"one point":  {at(0, 500*gb)},
		"three days": {at(3, 500*gb), at(0, 530*gb)},
	} {
		if f := stats.Forecast(points, 100*gb, now); f != nil {
			t.Errorf("%s: want no forecast, got %+v", name, f)
		}
	}
}

func TestWatch(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, time.UTC)

	w, err := f.svc.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if w.Owned != 0 || w.Plays != 0 || len(w.MostWatched) != 0 || len(w.LongestUnwatched) != 0 || len(w.ByMonth) != 12 {
		t.Fatalf("empty library: %+v", w)
	}

	heat := f.movie("Heat", 1995, 170, `["Crime"]`, "2021-03-01T00:00:00Z")
	f.file(heat, "heat.mkv", 60*gb, "2160p")
	dune := f.movie("Dune", 2021, 155, `["Sci-Fi"]`, "2022-06-01T00:00:00Z")
	f.file(dune, "dune.mkv", 40*gb, "2160p")
	alien := f.movie("Alien", 1979, 117, `["Horror"]`, "2020-01-01T00:00:00Z")
	f.file(alien, "alien.mkv", 30*gb, "1080p")
	brazil := f.movie("Brazil", 1985, 132, `["Sci-Fi"]`, "2023-01-01T00:00:00Z")
	f.file(brazil, "brazil.mkv", 20*gb, "1080p")
	wanted := f.movie("Wanted, No File", 2024, 100, `["Drama"]`, "2019-01-01T00:00:00Z")

	f.watch(heat, thisMonth, "ana")
	f.watch(heat, thisMonth.Add(time.Hour), "ben")
	f.watch(heat, thisMonth.AddDate(0, -2, 0), "ana")
	f.watch(dune, thisMonth.AddDate(0, -2, 0), "ana")
	f.watch(dune, thisMonth.AddDate(0, -30, 0), "ana") // older than the twelve months charted
	f.watch(wanted, thisMonth, "ana")                  // watched elsewhere; there's no file here

	w, err = f.svc.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if w.Owned != 4 || w.Watched != 2 {
		t.Errorf("owned/watched = %d/%d, want 4/2: only movies with a file count", w.Owned, w.Watched)
	}
	if w.Plays != 6 || w.MinutesWatched != 3*170+2*155+100 {
		t.Errorf("plays = %d, minutes = %d", w.Plays, w.MinutesWatched)
	}

	if len(w.ByMonth) != 12 || w.ByMonth[11].Month != now.Format("2006-01") {
		t.Fatalf("by month should be twelve months ending now: %+v", w.ByMonth)
	}
	if w.ByMonth[11].Plays != 3 || w.ByMonth[9].Plays != 2 || w.ByMonth[10].Plays != 0 {
		t.Errorf("by month = %+v", w.ByMonth)
	}

	if len(w.MostWatched) != 3 || w.MostWatched[0].Title != "Heat" || w.MostWatched[0].Plays != 3 || w.MostWatched[1].Title != "Dune" {
		t.Errorf("most watched = %+v", w.MostWatched)
	}
	if !w.MostWatched[0].LastWatchedAt.Equal(thisMonth.Add(time.Hour)) {
		t.Errorf("Heat last watched %v", w.MostWatched[0].LastWatchedAt)
	}

	if len(w.LongestUnwatched) != 2 || w.LongestUnwatched[0].Title != "Alien" || w.LongestUnwatched[1].Title != "Brazil" {
		t.Fatalf("unwatched should be Alien then Brazil (longest in the library first, files only): %+v", w.LongestUnwatched)
	}
	if got := w.LongestUnwatched[0].AddedAt; got.Year() != 2020 {
		t.Errorf("Alien added %v", got)
	}
}
