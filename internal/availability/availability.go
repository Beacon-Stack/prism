// Package availability works out where each movie stands from the user's
// side: ready to watch on the media server, downloaded but not there yet,
// downloading, or still wanted.
package availability

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/queue"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// State is where a movie stands.
type State string

const (
	// Available: the media server has it, so it can be watched.
	Available State = "available"
	// Downloading: a download for it is in the queue.
	Downloading State = "downloading"
	// Downloaded: the file is on disk but the media server doesn't have
	// it (yet), or there is no media server to ask.
	Downloaded State = "downloaded"
	// Missing: there was a file and it is gone.
	Missing State = "missing"
	// Upcoming: not released yet.
	Upcoming State = "upcoming"
	// Wanted: released and monitored, with nothing downloaded or on its
	// way.
	Wanted State = "wanted"
	// Unmonitored: nothing downloaded and Prism isn't looking for it.
	Unmonitored State = "unmonitored"
)

// Facts are the things State is decided from.
type Facts struct {
	Status        string // the movie's stored status
	Monitored     bool
	InMediaServer bool
	Downloading   bool
}

// Decide gives the state for a set of facts. The media server wins: a
// movie that can be watched is available however it got there, including
// from a folder Prism can't see.
func Decide(f Facts) State {
	switch {
	case f.InMediaServer:
		return Available
	case f.Downloading:
		return Downloading
	case f.Status == "downloaded":
		return Downloaded
	case f.Status == "missing":
		return Missing
	case f.Status == "announced":
		return Upcoming
	case f.Monitored:
		return Wanted
	default:
		return Unmonitored
	}
}

// Info is a movie's state, with how far along a download is.
type Info struct {
	State    State
	Progress float64 // 0..1, meaningful while Downloading
}

// ListerSource supplies the media servers that can list their movies.
type ListerSource interface {
	MovieListers(ctx context.Context) ([]plugin.MovieLister, error)
}

// QueueSource supplies the downloads in progress.
type QueueSource interface {
	GetQueue(ctx context.Context) ([]queue.Item, error)
}

// Service keeps a copy of what the media servers hold, refreshed on a
// schedule, and answers from it without calling out.
type Service struct {
	servers ListerSource
	queue   QueueSource
	logger  *slog.Logger

	mu      sync.RWMutex
	tmdbIDs map[int]struct{}
	titles  map[string]struct{} // "normalised title|year", for items with no TMDB ID
}

// NewService creates a Service. Until the first Refresh it knows of no
// media server contents, so nothing is reported as available.
func NewService(servers ListerSource, q QueueSource, logger *slog.Logger) *Service {
	return &Service{servers: servers, queue: q, logger: logger}
}

// Refresh re-reads what every media server holds. If a server can't be
// reached its last known contents are kept: a server that is down for a
// minute shouldn't turn the whole library back to "downloaded".
func (s *Service) Refresh(ctx context.Context) error {
	listers, err := s.servers.MovieListers(ctx)
	if err != nil {
		return err
	}
	tmdbIDs := make(map[int]struct{})
	titles := make(map[string]struct{})
	for _, l := range listers {
		items, err := l.ListMovieItems(ctx)
		if err != nil {
			return err
		}
		for _, it := range items {
			if it.TMDBID > 0 {
				tmdbIDs[it.TMDBID] = struct{}{}
			} else if it.Title != "" && it.Year > 0 {
				titles[titleKey(it.Title, it.Year)] = struct{}{}
			}
		}
	}
	s.mu.Lock()
	s.tmdbIDs, s.titles = tmdbIDs, titles
	s.mu.Unlock()
	return nil
}

// InMediaServer reports whether a media server holds the movie. It is
// matched by TMDB ID; an item the server has no ID for is matched by
// exact title and year.
func (s *Service) InMediaServer(m movie.Movie) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.tmdbIDs[m.TMDBID]; ok && m.TMDBID > 0 {
		return true
	}
	if len(s.titles) == 0 || m.Year == 0 {
		return false
	}
	if _, ok := s.titles[titleKey(m.Title, m.Year)]; ok {
		return true
	}
	_, ok := s.titles[titleKey(m.OriginalTitle, m.Year)]
	return ok
}

// For returns the state of each movie, keyed by movie ID.
func (s *Service) For(ctx context.Context, movies []movie.Movie) map[string]Info {
	progress := make(map[string]float64)
	if s.queue != nil {
		items, err := s.queue.GetQueue(ctx)
		if err != nil && s.logger != nil {
			s.logger.Warn("availability: reading queue", "error", err)
		}
		for _, it := range items {
			if it.Status == "failed" || it.Status == "removed" || it.Status == "completed" {
				continue
			}
			p := 0.0
			if it.Size > 0 {
				p = min(1, float64(it.DownloadedBytes)/float64(it.Size))
			}
			if cur, ok := progress[it.MovieID]; !ok || p > cur {
				progress[it.MovieID] = p
			}
		}
	}
	out := make(map[string]Info, len(movies))
	for _, m := range movies {
		p, downloading := progress[m.ID]
		state := Decide(Facts{
			Status:        m.Status,
			Monitored:     m.Monitored,
			InMediaServer: s.InMediaServer(m),
			Downloading:   downloading,
		})
		info := Info{State: state}
		if state == Downloading {
			info.Progress = p
		}
		out[m.ID] = info
	}
	return out
}

var notTitleChar = regexp.MustCompile(`[^a-z0-9\s]`)

func titleKey(title string, year int) string {
	t := notTitleChar.ReplaceAllString(strings.ToLower(title), "")
	return strings.Join(strings.Fields(t), " ") + "|" + strconv.Itoa(year)
}
