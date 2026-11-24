package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/parser"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// The search core, shared with Pilot's design: results stream in as each
// indexer answers, an indexer that keeps missing gets a short wait when a
// person is watching, and that memory survives restarts.

// errSearchTimedOut replaces a raw context.DeadlineExceeded in user-facing
// per-indexer outcomes. Slow (Cloudflare-protected) indexers finish their
// scrape server-side after we disconnect and the result is cached, so
// searching again shortly is the fix, and the message says so.
var errSearchTimedOut = errors.New("timed out — slow (Cloudflare-protected) indexers finish in the background; search again shortly for cached results")

// struggleTimeout is how long an interactive search waits on an indexer
// that keeps missing (see indexerHealth): typically a Cloudflare-walled
// site whose challenge FlareSolverr often can't solve. Its request still
// goes out and Pulse caches what it finds; everyone else's results just
// don't sit behind it for 25s. A var so tests can shorten it.
var struggleTimeout = 6 * time.Second

// indexerHealth remembers each indexer's last few search outcomes. One
// that missed (timed out or failed) struggleMisses of its last
// healthWindow searches is "struggling". A window rather than a streak:
// a struggling indexer's cached answers come back fast and would reset a
// streak between every two cold misses.
type indexerHealth struct {
	mu     sync.Mutex
	recent map[string][]bool // true = miss, oldest first
}

const (
	healthWindow   = 3
	struggleMisses = 2
	// healthSettingKey is where indexer health is saved in the settings
	// table, so a restart doesn't give struggling indexers the full wait
	// again.
	healthSettingKey = "indexer_health"
)

func (h *indexerHealth) record(id string, miss bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.recent == nil {
		h.recent = make(map[string][]bool)
	}
	r := append(h.recent[id], miss)
	if len(r) > healthWindow {
		r = r[len(r)-healthWindow:]
	}
	h.recent[id] = r
}

func (h *indexerHealth) struggling(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	misses := 0
	for _, miss := range h.recent[id] {
		if miss {
			misses++
		}
	}
	return misses >= struggleMisses
}

func (h *indexerHealth) snapshot() map[string][]bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string][]bool, len(h.recent))
	for id, r := range h.recent {
		out[id] = append([]bool(nil), r...)
	}
	return out
}

func (h *indexerHealth) restore(saved map[string][]bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recent = make(map[string][]bool, len(saved))
	for id, r := range saved {
		if len(r) > healthWindow {
			r = r[len(r)-healthWindow:]
		}
		h.recent[id] = append([]bool(nil), r...)
	}
}

// LoadHealth restores indexer health saved by an earlier run. Nothing
// saved, or something unreadable, starts fresh.
func (s *Service) LoadHealth(ctx context.Context) error {
	if s.q == nil {
		return nil
	}
	raw, err := s.q.GetSetting(ctx, healthSettingKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("loading indexer health: %w", err)
	}
	var saved map[string][]bool
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		return fmt.Errorf("reading saved indexer health: %w", err)
	}
	s.health.restore(saved)
	return nil
}

// saveHealth writes the current indexer health, in the background after
// each search; saves are serialized so an older snapshot can't land after
// a newer one.
func (s *Service) saveHealth() {
	if s.q == nil {
		return
	}
	s.healthSaveMu.Lock()
	defer s.healthSaveMu.Unlock()
	b, err := json.Marshal(s.health.snapshot())
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.q.SetSetting(ctx, dbgen.SetSettingParams{
		Key:       healthSettingKey,
		Value:     string(b),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// Outcome is how one indexer did in a search.
type Outcome struct {
	IndexerID   string
	IndexerName string
	Results     int
	Error       string // empty on success
	// Pending: the indexer hit our deadline but its scrape continues
	// server-side and will be cached; not a failure.
	Pending bool
	// Searching: no answer yet. Only in the snapshots of a running search.
	Searching bool
}

// Snapshot is a search's results so far, ranked and tagged exactly as the
// finished search will be, with where each indexer stands.
type Snapshot struct {
	Results  []SearchResult
	Outcomes []Outcome
	Done     bool
}

// SearchInteractive runs a search for a person watching the results:
// onUpdate is called each time an indexer answers, with everything found
// so far, and an indexer that keeps missing is waited on for
// struggleTimeout rather than the full timeout. onUpdate calls are
// serialized and arrive in order; keep them quick.
func (s *Service) SearchInteractive(ctx context.Context, query plugin.SearchQuery, onUpdate func(Snapshot)) (Snapshot, error) {
	return s.search(ctx, query, nil, onUpdate, true)
}

func (s *Service) search(ctx context.Context, query plugin.SearchQuery, allowedIDs []string, onUpdate func(Snapshot), interactive bool) (Snapshot, error) {
	rows, err := s.q.ListEnabledIndexers(ctx)
	if err != nil {
		return Snapshot{Done: true}, fmt.Errorf("listing enabled indexers: %w", err)
	}
	if allowedIDs != nil {
		allowed := make(map[string]struct{}, len(allowedIDs))
		for _, id := range allowedIDs {
			allowed[id] = struct{}{}
		}
		kept := rows[:0:0]
		for _, r := range rows {
			if _, ok := allowed[r.ID]; ok {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	if len(rows) == 0 {
		return Snapshot{Done: true}, nil
	}

	minByIndexer := make(map[string]int, len(rows))
	for _, r := range rows {
		minByIndexer[r.ID] = int(r.MinSeeders)
	}

	var (
		mu       sync.Mutex
		raw      []SearchResult
		outcomes = make([]Outcome, len(rows))
		pos      = make(map[string]int, len(rows))
		errs     []error
	)
	for i, r := range rows {
		outcomes[i] = Outcome{IndexerID: r.ID, IndexerName: r.Name, Searching: true}
		pos[r.ID] = i
	}
	snapshot := func(done bool) Snapshot {
		results := make([]SearchResult, len(raw))
		for i, r := range raw {
			r.FilterReasons = append([]string(nil), r.FilterReasons...)
			results[i] = r
		}
		return Snapshot{Results: finish(results, minByIndexer), Outcomes: append([]Outcome(nil), outcomes...), Done: done}
	}

	// First update before anyone answers: which indexers are searching.
	if onUpdate != nil {
		onUpdate(snapshot(false))
	}

	var wg sync.WaitGroup
	for _, row := range rows {
		wg.Add(1)
		go func(row dbgen.IndexerConfig) {
			defer wg.Done()
			cfg, _ := rowToConfig(row)
			releases, err := s.searchSlotted(ctx, cfg, query, interactive)

			mu.Lock()
			defer mu.Unlock()
			o := &outcomes[pos[cfg.ID]]
			o.Searching = false
			switch {
			case err == nil:
				for _, r := range releases {
					raw = append(raw, newSearchResult(r, cfg.ID, cfg.Name))
				}
				o.Results = len(releases)
			case errors.Is(err, errSearchTimedOut):
				o.Pending = true
			default:
				o.Error = err.Error()
				errs = append(errs, fmt.Errorf("indexer %q: %w", cfg.Name, err))
			}
			if onUpdate != nil {
				onUpdate(snapshot(false))
			}
		}(row)
	}
	wg.Wait()
	go s.saveHealth()

	mu.Lock()
	defer mu.Unlock()
	var combined error
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		combined = fmt.Errorf("%d indexer(s) failed: %v", len(errs), msgs)
	}
	return snapshot(true), combined
}

// searchSlotted waits for the indexer's rate limit, then takes a global
// concurrency slot for the request alone. The wait comes first and holds
// nothing, so an indexer waiting out its own rate limit doesn't keep
// faster indexers queued behind it.
func (s *Service) searchSlotted(ctx context.Context, cfg Config, query plugin.SearchQuery, interactive bool) ([]plugin.Release, error) {
	idx, err := s.cachedIndexer(cfg.Kind, cfg.ID, cfg.Settings)
	if err != nil {
		return nil, err
	}
	if err := s.rl.Wait(ctx, cfg.ID, extractRateLimit(cfg.Settings)); err != nil {
		return nil, fmt.Errorf("waiting for this indexer's rate limit: %w", err)
	}
	if s.sem != nil {
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	timeout := perIndexerSearchTimeout
	if interactive && s.health.struggling(cfg.ID) {
		timeout = struggleTimeout
	}
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	releases, err := idx.Search(sctx, query)
	// A search the caller ended (window closed) says nothing about the
	// indexer; anything else does.
	if ctx.Err() == nil {
		s.health.record(cfg.ID, err != nil)
	}
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		err = errSearchTimedOut
	}
	return releases, err
}

// newSearchResult fills in what the indexer didn't: its name, and the
// quality, edition and group parsed from the title.
func newSearchResult(r plugin.Release, indexerID, indexerName string) SearchResult {
	// Prefer the per-item indexer name from Prowlarr/Jackett; fall back to
	// the user-configured name.
	if r.Indexer == "" {
		r.Indexer = indexerName
	}
	parsed := parser.Parse(r.Title)
	if r.Quality.Source == "" || r.Quality.Source == plugin.SourceUnknown {
		r.Quality = parsed.Quality()
	}
	if r.Edition == "" && parsed.Edition != "" {
		r.Edition = parsed.Edition
	}
	if r.ReleaseGroup == "" {
		r.ReleaseGroup = parsed.ReleaseGroup
	}
	return SearchResult{Release: r, IndexerID: indexerID, QualityScore: r.Quality.Score()}
}

// finish tags, ranks and merges a result set, the same for every search.
func finish(results []SearchResult, minByIndexer map[string]int) []SearchResult {
	// Tag low-seed releases per the per-indexer min_seeders threshold.
	// Tagging (not dropping) preserves user agency — see filter.go. MUST
	// run before dedup so the merged "winning" row inherits any
	// FilterReasons accumulated by its constituents.
	applyMinSeedersFilter(results, minByIndexer)

	// Sort by quality score descending, then by seeds descending.
	sort.SliceStable(results, func(i, j int) bool {
		si, sj := results[i].QualityScore, results[j].QualityScore
		if si != sj {
			return si > sj
		}
		return results[i].Seeds > results[j].Seeds
	})

	// Collapse cross-indexer duplicates of the same torrent (by info_hash)
	// into a single row showing the MEDIAN seed count.
	return dedupeByInfoHash(results)
}
