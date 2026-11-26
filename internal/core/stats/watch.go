package stats

import (
	"context"
	"fmt"
	"time"
)

// WatchedMovie is a movie and how often it has been played.
type WatchedMovie struct {
	MovieID       string    `json:"movie_id"`
	Title         string    `json:"title"`
	Year          int64     `json:"year"`
	Plays         int64     `json:"plays"`
	LastWatchedAt time.Time `json:"last_watched_at"`
}

// UnwatchedMovie is a movie with a file that nobody has played.
type UnwatchedMovie struct {
	MovieID string    `json:"movie_id"`
	Title   string    `json:"title"`
	Year    int64     `json:"year"`
	AddedAt time.Time `json:"added_at"`
}

// WatchMonth is the number of plays in one calendar month.
type WatchMonth struct {
	Month string `json:"month"` // YYYY-MM
	Plays int64  `json:"plays"`
}

// Watching summarises the watch history synced from media servers.
type Watching struct {
	Owned            int64            `json:"owned"`             // movies with a file
	Watched          int64            `json:"watched"`           // of those, played at least once
	Plays            int64            `json:"plays"`             // every play by every user
	MinutesWatched   int64            `json:"minutes_watched"`   // plays times runtime
	ByMonth          []WatchMonth     `json:"by_month"`          // the last twelve months, oldest first, empty months included
	MostWatched      []WatchedMovie   `json:"most_watched"`      // most plays first
	LongestUnwatched []UnwatchedMovie `json:"longest_unwatched"` // longest in the library first
}

const (
	watchMonths    = 12
	watchListLimit = 8
)

// lastMonths returns the n calendar months ending with now's, oldest
// first, as YYYY-MM.
func lastMonths(now time.Time, n int) []string {
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]string, n)
	for i := range n {
		out[i] = first.AddDate(0, i-(n-1), 0).Format("2006-01")
	}
	return out
}

func parseTime(v any) time.Time {
	s, _ := v.(string)
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

// Watch returns what has been watched, how much, and what hasn't.
func (s *Service) Watch(ctx context.Context) (Watching, error) {
	totals, err := s.q.GetWatchTotals(ctx)
	if err != nil {
		return Watching{}, fmt.Errorf("getting watch totals: %w", err)
	}
	out := Watching{
		Owned:          totals.Owned,
		Watched:        totals.Watched,
		Plays:          totals.Plays,
		MinutesWatched: toInt64(totals.Minutes),
	}

	months, err := s.q.GetWatchesByMonth(ctx)
	if err != nil {
		return Watching{}, fmt.Errorf("getting watches by month: %w", err)
	}
	plays := make(map[string]int64, len(months))
	for _, m := range months {
		month, _ := m.Month.(string)
		plays[month] = m.Plays
	}
	for _, month := range lastMonths(time.Now().UTC(), watchMonths) {
		out.ByMonth = append(out.ByMonth, WatchMonth{Month: month, Plays: plays[month]})
	}

	most, err := s.q.ListMostWatchedMovies(ctx, watchListLimit)
	if err != nil {
		return Watching{}, fmt.Errorf("listing most watched movies: %w", err)
	}
	out.MostWatched = make([]WatchedMovie, 0, len(most))
	for _, m := range most {
		out.MostWatched = append(out.MostWatched, WatchedMovie{
			MovieID: m.ID, Title: m.Title, Year: m.Year, Plays: m.Plays, LastWatchedAt: parseTime(m.LastWatchedAt),
		})
	}

	unwatched, err := s.q.ListLongestUnwatchedMovies(ctx, watchListLimit)
	if err != nil {
		return Watching{}, fmt.Errorf("listing unwatched movies: %w", err)
	}
	out.LongestUnwatched = make([]UnwatchedMovie, 0, len(unwatched))
	for _, m := range unwatched {
		out.LongestUnwatched = append(out.LongestUnwatched, UnwatchedMovie{
			MovieID: m.ID, Title: m.Title, Year: m.Year, AddedAt: parseTime(m.AddedAt),
		})
	}
	return out, nil
}
