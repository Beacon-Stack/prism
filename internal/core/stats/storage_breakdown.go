package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"

	"github.com/beacon-stack/prism/pkg/plugin"
)

// StorageSlice is the space taken by one group of movie files.
type StorageSlice struct {
	Label  string `json:"label"`
	Bytes  int64  `json:"bytes"`
	Movies int64  `json:"movies"`
	// TimedBytes and RuntimeMinutes cover only the files whose movie has a
	// known runtime, so bytes per hour isn't skewed by the ones without.
	TimedBytes     int64 `json:"timed_bytes"`
	RuntimeMinutes int64 `json:"runtime_minutes"`
}

// LargestMovie is one movie and the space all its files take.
type LargestMovie struct {
	MovieID        string `json:"movie_id"`
	Title          string `json:"title"`
	Year           int64  `json:"year"`
	Bytes          int64  `json:"bytes"`
	Resolution     string `json:"resolution"` // of its largest file
	RuntimeMinutes int64  `json:"runtime_minutes"`
}

// StorageForecast projects when the library disks fill up.
type StorageForecast struct {
	BytesPerDay int64 `json:"bytes_per_day"` // average over the snapshots; zero or less when not growing
	FreeBytes   int64 `json:"free_bytes"`    // free space across the library disks
	// DaysUntilFull and FullOn are unset when the library isn't growing.
	DaysUntilFull *int64     `json:"days_until_full,omitempty"`
	FullOn        *time.Time `json:"full_on,omitempty"`
}

// StorageBreakdown is where the library's disk space goes.
type StorageBreakdown struct {
	Total        StorageSlice     `json:"total"`
	ByResolution []StorageSlice   `json:"by_resolution"` // largest first
	ByGenre      []StorageSlice   `json:"by_genre"`      // largest first; a movie counts toward each of its genres
	ByDecade     []StorageSlice   `json:"by_decade"`     // oldest first
	Largest      []LargestMovie   `json:"largest"`
	Forecast     *StorageForecast `json:"forecast,omitempty"` // unset until there is a week of snapshots and a library disk to measure
}

const (
	maxStorageGenres  = 10
	maxLargestMovies  = 10
	forecastSnapshots = 90
	// Fewer days of history than this says little about the rate.
	forecastMinSpan = 7 * 24 * time.Hour
)

// sliceSet accumulates StorageSlices by label, counting each movie once
// per label however many files it has.
type sliceSet struct {
	slices map[string]*StorageSlice
	seen   map[string]map[string]bool
}

func newSliceSet() *sliceSet {
	return &sliceSet{slices: map[string]*StorageSlice{}, seen: map[string]map[string]bool{}}
}

func (s *sliceSet) add(label, movieID string, bytes, runtime int64) {
	sl := s.slices[label]
	if sl == nil {
		sl = &StorageSlice{Label: label}
		s.slices[label] = sl
		s.seen[label] = map[string]bool{}
	}
	sl.Bytes += bytes
	if runtime > 0 {
		sl.TimedBytes += bytes
		sl.RuntimeMinutes += runtime
	}
	if !s.seen[label][movieID] {
		s.seen[label][movieID] = true
		sl.Movies++
	}
}

// sorted returns the slices ordered by less, ties by label.
func (s *sliceSet) sorted(less func(a, b StorageSlice) bool) []StorageSlice {
	out := make([]StorageSlice, 0, len(s.slices))
	for _, sl := range s.slices {
		out = append(out, *sl)
	}
	sort.Slice(out, func(i, j int) bool {
		if less(out[i], out[j]) != less(out[j], out[i]) {
			return less(out[i], out[j])
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func largestFirst(a, b StorageSlice) bool { return a.Bytes > b.Bytes }

// StorageBreakdown groups the space movie files take by resolution, genre
// and decade, lists the largest movies and projects when the disks fill.
func (s *Service) StorageBreakdown(ctx context.Context) (StorageBreakdown, error) {
	rows, err := s.q.ListMovieFileStorage(ctx)
	if err != nil {
		return StorageBreakdown{}, fmt.Errorf("listing movie file storage: %w", err)
	}

	total, byRes, byGenre, byDecade := newSliceSet(), newSliceSet(), newSliceSet(), newSliceSet()
	movies := map[string]*LargestMovie{}
	biggestFile := map[string]int64{}

	for _, r := range rows {
		var runtime int64
		if r.RuntimeMinutes != nil {
			runtime = *r.RuntimeMinutes
		}
		var q plugin.Quality
		_ = json.Unmarshal([]byte(r.QualityJson), &q)
		resolution := string(q.Resolution)
		if resolution == "" {
			resolution = "unknown"
		}

		total.add("", r.MovieID, r.SizeBytes, runtime)
		byRes.add(resolution, r.MovieID, r.SizeBytes, runtime)
		if r.Year > 0 {
			byDecade.add(fmt.Sprintf("%ds", r.Year/10*10), r.MovieID, r.SizeBytes, runtime)
		}
		var genres []string
		_ = json.Unmarshal([]byte(r.GenresJson), &genres)
		for _, g := range genres {
			if g != "" {
				byGenre.add(g, r.MovieID, r.SizeBytes, runtime)
			}
		}

		m := movies[r.MovieID]
		if m == nil {
			m = &LargestMovie{MovieID: r.MovieID, Title: r.Title, Year: r.Year, RuntimeMinutes: runtime}
			movies[r.MovieID] = m
		}
		m.Bytes += r.SizeBytes
		if r.SizeBytes >= biggestFile[r.MovieID] {
			biggestFile[r.MovieID] = r.SizeBytes
			m.Resolution = resolution
		}
	}

	out := StorageBreakdown{
		ByResolution: byRes.sorted(largestFirst),
		ByGenre:      byGenre.sorted(largestFirst),
		ByDecade:     byDecade.sorted(func(a, b StorageSlice) bool { return a.Label < b.Label }),
		Largest:      make([]LargestMovie, 0, len(movies)),
	}
	if t := total.slices[""]; t != nil {
		out.Total = *t
	}
	if len(out.ByGenre) > maxStorageGenres {
		out.ByGenre = out.ByGenre[:maxStorageGenres]
	}
	for _, m := range movies {
		out.Largest = append(out.Largest, *m)
	}
	sort.Slice(out.Largest, func(i, j int) bool {
		if out.Largest[i].Bytes != out.Largest[j].Bytes {
			return out.Largest[i].Bytes > out.Largest[j].Bytes
		}
		return out.Largest[i].Title < out.Largest[j].Title
	})
	if len(out.Largest) > maxLargestMovies {
		out.Largest = out.Largest[:maxLargestMovies]
	}

	// The forecast is a bonus: without it the breakdown still stands.
	if free, ok := s.libraryFreeBytes(ctx); ok {
		if trend, err := s.StorageTrend(ctx, forecastSnapshots); err == nil {
			out.Forecast = Forecast(trend, free, time.Now())
		}
	}
	return out, nil
}

// Forecast projects when freeBytes runs out at the average rate between
// the first and last of points (oldest first). It returns nil when the
// points span less than a week.
func Forecast(points []StoragePoint, freeBytes int64, now time.Time) *StorageForecast {
	if len(points) < 2 {
		return nil
	}
	first, last := points[0], points[len(points)-1]
	span := last.CapturedAt.Sub(first.CapturedAt)
	if span < forecastMinSpan {
		return nil
	}
	perDay := int64(float64(last.TotalBytes-first.TotalBytes) / (span.Hours() / 24))
	f := &StorageForecast{BytesPerDay: perDay, FreeBytes: freeBytes}
	if perDay > 0 {
		days := freeBytes / perDay
		fullOn := now.Add(time.Duration(days) * 24 * time.Hour)
		f.DaysUntilFull, f.FullOn = &days, &fullOn
	}
	return f
}

// libraryFreeBytes adds up the free space on the disks the libraries live
// on, counting a disk shared by several libraries once. ok is false when
// no library disk could be measured.
func (s *Service) libraryFreeBytes(ctx context.Context) (free int64, ok bool) {
	libs, err := s.q.ListLibraries(ctx)
	if err != nil {
		return 0, false
	}
	devices := map[uint64]bool{}
	for _, lib := range libs {
		info, err := os.Stat(lib.RootPath)
		if err != nil {
			continue
		}
		st, isStat := info.Sys().(*syscall.Stat_t)
		if !isStat || devices[uint64(st.Dev)] {
			continue
		}
		var fs syscall.Statfs_t
		if err := syscall.Statfs(lib.RootPath, &fs); err != nil {
			continue
		}
		devices[uint64(st.Dev)] = true
		free += int64(fs.Bavail) * int64(fs.Bsize)
		ok = true
	}
	return free, ok
}
