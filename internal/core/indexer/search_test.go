package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/ratelimit"
	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// searchQuerier answers the queries a search makes and keeps settings in
// memory; anything else panics through the nil embedded interface.
type searchQuerier struct {
	dbgen.Querier
	rows     []dbgen.IndexerConfig
	settings *settingsStore
}

// settingsStore is a settings table shared by services in a test (as one
// database outlives a restart).
type settingsStore struct {
	mu sync.Mutex
	m  map[string]string
}

func newSettings() *settingsStore { return &settingsStore{m: map[string]string{}} }

func (s *settingsStore) set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}

func (q searchQuerier) ListEnabledIndexers(context.Context) ([]dbgen.IndexerConfig, error) {
	return q.rows, nil
}

func (q searchQuerier) GetSetting(_ context.Context, key string) (string, error) {
	q.settings.mu.Lock()
	defer q.settings.mu.Unlock()
	v, ok := q.settings.m[key]
	if !ok {
		return "", sql.ErrNoRows
	}
	return v, nil
}

func (q searchQuerier) SetSetting(_ context.Context, arg dbgen.SetSettingParams) error {
	q.settings.mu.Lock()
	defer q.settings.mu.Unlock()
	q.settings.m[arg.Key] = arg.Value
	return nil
}

// scripted answers every search with fixed titles, after its gate opens
// (or fails with err).
type scripted struct {
	titles []string
	gate   chan struct{}
	err    error
}

func (f *scripted) Name() string               { return "scripted" }
func (f *scripted) Protocol() plugin.Protocol  { return plugin.ProtocolTorrent }
func (f *scripted) Test(context.Context) error { return nil }
func (f *scripted) GetRecent(context.Context) ([]plugin.Release, error) {
	return nil, nil
}
func (f *scripted) Capabilities(context.Context) (plugin.Capabilities, error) {
	return plugin.Capabilities{}, nil
}
func (f *scripted) Search(ctx context.Context, _ plugin.SearchQuery) ([]plugin.Release, error) {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	var out []plugin.Release
	for _, t := range f.titles {
		out = append(out, plugin.Release{GUID: t, Title: t, Seeds: 10})
	}
	return out, nil
}

// newSearchService builds a service over the given indexers, by ID.
func newSearchService(t *testing.T, indexers map[string]*scripted, settings *settingsStore) *Service {
	t.Helper()
	reg := registry.New()
	var rows []dbgen.IndexerConfig
	for id, f := range indexers {
		f := f
		kind := "scripted-" + id
		reg.RegisterIndexer(kind, func(json.RawMessage) (plugin.Indexer, error) { return f, nil })
		rows = append(rows, dbgen.IndexerConfig{ID: id, Name: id, Kind: kind, Enabled: true, Settings: "{}"})
	}
	if settings == nil {
		settings = newSettings()
	}
	return NewService(searchQuerier{rows: rows, settings: settings}, reg, nil, ratelimit.New())
}

func shortStruggle(t *testing.T) {
	old := struggleTimeout
	struggleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { struggleTimeout = old })
}

var dune = plugin.SearchQuery{Query: "Dune", Year: 2021}

// Results show up as each indexer answers, not when the slowest does.
func TestSearchInteractive_StreamsAsIndexersAnswer(t *testing.T) {
	slow := &scripted{titles: []string{"Dune.2021.2160p.WEB-DL"}, gate: make(chan struct{})}
	fast := &scripted{titles: []string{"Dune.2021.1080p.BluRay.x264"}}
	svc := newSearchService(t, map[string]*scripted{"fast": fast, "slow": slow}, nil)

	var mu sync.Mutex
	var updates []Snapshot
	sawFastFirst := make(chan struct{})
	once := sync.Once{}
	go func() {
		<-sawFastFirst
		close(slow.gate)
	}()
	final, err := svc.SearchInteractive(context.Background(), dune, func(s Snapshot) {
		mu.Lock()
		updates = append(updates, s)
		mu.Unlock()
		if len(s.Results) == 1 {
			once.Do(func() { close(sawFastFirst) })
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 3 {
		t.Fatalf("updates: %d (want: start, fast, slow)", len(updates))
	}
	if n := len(updates[0].Results); n != 0 || !updates[0].Outcomes[0].Searching {
		t.Errorf("first update should be everyone searching, got %d results", n)
	}
	if len(updates[1].Results) != 1 || updates[1].Done {
		t.Errorf("second update should hold the fast indexer's result: %+v", updates[1])
	}
	if !final.Done || len(final.Results) != 2 {
		t.Errorf("final %+v", final)
	}
	// Ranked: 2160p above 1080p, as the finished search always was.
	if final.Results[0].Title != "Dune.2021.2160p.WEB-DL" {
		t.Errorf("ranking: %s first", final.Results[0].Title)
	}
}

// An indexer that keeps missing gets a short wait when a person is
// watching, and is reported pending, not failed.
func TestSearchInteractive_CutsStrugglingIndexersShort(t *testing.T) {
	shortStruggle(t)
	stuck := &scripted{gate: make(chan struct{})}
	defer close(stuck.gate)
	ok := &scripted{titles: []string{"Dune.2021.1080p.BluRay.x264"}}
	svc := newSearchService(t, map[string]*scripted{"stuck": stuck, "ok": ok}, nil)
	svc.health.record("stuck", true)
	svc.health.record("stuck", true)

	start := time.Now()
	snap, err := svc.SearchInteractive(context.Background(), dune, nil)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("took %v; the struggling indexer should be cut at %v", took, struggleTimeout)
	}
	for _, o := range snap.Outcomes {
		if o.IndexerID == "stuck" && (!o.Pending || o.Error != "") {
			t.Errorf("stuck indexer should be pending: %+v", o)
		}
	}
	if len(snap.Results) != 1 {
		t.Errorf("results %+v", snap.Results)
	}
}

// Automatic searches still wait the full time for a struggling indexer,
// so they don't lose its results.
func TestSearch_AutomaticWaitsForStrugglingIndexers(t *testing.T) {
	shortStruggle(t)
	late := &scripted{titles: []string{"Dune.2021.2160p.WEB-DL"}, gate: make(chan struct{})}
	svc := newSearchService(t, map[string]*scripted{"late": late}, nil)
	svc.health.record("late", true)
	svc.health.record("late", true)
	time.AfterFunc(300*time.Millisecond, func() { close(late.gate) })

	results, err := svc.Search(context.Background(), dune, nil)
	if err != nil || len(results) != 1 {
		t.Errorf("automatic search lost the slow indexer's result: %v %v", results, err)
	}
}

// Indexer health is saved after a search and restored by a new service,
// as after a restart.
func TestHealth_SurvivesRestart(t *testing.T) {
	settings := newSettings()
	failing := &scripted{err: errors.New("cloudflare")}
	svc := newSearchService(t, map[string]*scripted{"cf": failing}, settings)
	for i := 0; i < 2; i++ {
		_, _ = svc.SearchInteractive(context.Background(), dune, nil)
	}
	svc.saveHealth() // the search saves in the background; make it certain
	if !svc.health.struggling("cf") {
		t.Fatal("two misses should mark it struggling")
	}

	restarted := newSearchService(t, map[string]*scripted{"cf": failing}, settings)
	if err := restarted.LoadHealth(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !restarted.health.struggling("cf") {
		t.Error("a restarted service forgot the indexer keeps missing")
	}
	settings.set(healthSettingKey, "not json")
	if err := newSearchService(t, nil, settings).LoadHealth(context.Background()); err == nil {
		t.Error("junk should be reported so the caller can log it and start fresh")
	}
}

// Tag-restricted searches still only ask the allowed indexers.
func TestSearch_AllowedIndexersOnly(t *testing.T) {
	a := &scripted{titles: []string{"Dune.2021.1080p.BluRay.x264"}}
	b := &scripted{titles: []string{"Dune.2021.2160p.WEB-DL"}}
	svc := newSearchService(t, map[string]*scripted{"a": a, "b": b}, nil)
	results, err := svc.Search(context.Background(), dune, []string{"b"})
	if err != nil || len(results) != 1 || results[0].IndexerID != "b" {
		t.Errorf("results %+v err %v", results, err)
	}
}
