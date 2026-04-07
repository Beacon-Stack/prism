package plexwatchlist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beacon-stack/prism/internal/registry"
)

// mustMarshal panics if json.Marshal fails — acceptable in test helpers.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

// newTestPlugin builds a Plugin pointing at the given test server URL.
// The Plugin.client field lets us bypass safedialer for local test servers.
func newTestPlugin(accessToken, serverURL string) *Plugin {
	return &Plugin{
		cfg:    Config{AccessToken: accessToken},
		client: newTestHTTPClient(serverURL),
	}
}

// newTestHTTPClient returns an *http.Client that redirects all requests to
// serverURL, replacing the scheme+host. This is the same trick the Plex media
// server plugin tests use to intercept calls to a fixed base URL.
func newTestHTTPClient(serverURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{baseURL: serverURL},
	}
}

// redirectTransport rewrites the host/scheme of every outgoing request to point
// at the test server. This lets us test plugins that construct their own URLs
// from a hardcoded constant (like plexBaseURL) without modifying the plugin.
type redirectTransport struct {
	baseURL string
	inner   http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	// Keep path+query; replace scheme+host.
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(rt.baseURL, "http://")
	if rt.inner != nil {
		return rt.inner.RoundTrip(cloned)
	}
	return http.DefaultTransport.RoundTrip(cloned)
}

// ---------------------------------------------------------------------------
// Registry factory
// ---------------------------------------------------------------------------

func TestFactory_Valid(t *testing.T) {
	settings := json.RawMessage(`{"access_token":"tok"}`)
	p, err := registry.Default.NewImportList("plex_watchlist", settings)
	if err != nil {
		t.Fatalf("NewImportList() error = %v", err)
	}
	if p.Name() != "Plex Watchlist" {
		t.Errorf("Name() = %q, want Plex Watchlist", p.Name())
	}
}

func TestFactory_MissingAccessToken(t *testing.T) {
	settings := json.RawMessage(`{}`)
	_, err := registry.Default.NewImportList("plex_watchlist", settings)
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
}

func TestFactory_InvalidJSON(t *testing.T) {
	_, err := registry.Default.NewImportList("plex_watchlist", json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// Name
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	p := &Plugin{cfg: Config{AccessToken: "tok"}}
	if got := p.Name(); got != "Plex Watchlist" {
		t.Errorf("Name() = %q, want Plex Watchlist", got)
	}
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path contains watchlist endpoint.
		if !strings.HasPrefix(r.URL.Path, "/library/sections/watchlist/all") {
			t.Errorf("path = %q, want /library/sections/watchlist/all...", r.URL.Path)
		}
		// Verify Plex-specific headers.
		if tok := r.Header.Get("X-Plex-Token"); tok != "my-plex-token" {
			t.Errorf("X-Plex-Token = %q, want my-plex-token", tok)
		}
		if ci := r.Header.Get("X-Plex-Client-Identifier"); ci != plexClientID {
			t.Errorf("X-Plex-Client-Identifier = %q, want %s", ci, plexClientID)
		}
		if prod := r.Header.Get("X-Plex-Product"); prod != plexProduct {
			t.Errorf("X-Plex-Product = %q, want %s", prod, plexProduct)
		}

		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []map[string]any{
					{
						"title": "Inception",
						"year":  2010,
						"type":  "movie",
						"guid":  "plex://movie/abc123",
						"Guid": []map[string]any{
							{"id": "tmdb://27205"},
							{"id": "imdb://tt1375666"},
						},
					},
					{
						// Non-movie type — should be filtered out.
						"title": "Breaking Bad",
						"year":  2008,
						"type":  "show",
						"guid":  "plex://show/xyz",
						"Guid":  []map[string]any{},
					},
					{
						// Has no TMDb ID — should be filtered out.
						"title": "Unknown",
						"year":  2020,
						"type":  "movie",
						"guid":  "plex://movie/zzz",
						"Guid":  []map[string]any{{"id": "imdb://tt9999999"}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("my-plex-token", srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	// Non-movie type and missing TMDb ID must be filtered.
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TMDbID != 27205 {
		t.Errorf("TMDbID = %d, want 27205", items[0].TMDbID)
	}
	if items[0].IMDbID != "tt1375666" {
		t.Errorf("IMDbID = %q, want tt1375666", items[0].IMDbID)
	}
	if items[0].Title != "Inception" {
		t.Errorf("Title = %q, want Inception", items[0].Title)
	}
	if items[0].Year != 2010 {
		t.Errorf("Year = %d, want 2010", items[0].Year)
	}
}

func TestFetch_EmptyWatchlist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestFetch_OnlyTMDbID(t *testing.T) {
	// A movie that has a tmdb GUID but no imdb GUID — still importable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []map[string]any{
					{
						"title": "Arthouse Film",
						"year":  2019,
						"type":  "movie",
						"guid":  "plex://movie/art",
						"Guid":  []map[string]any{{"id": "tmdb://99999"}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TMDbID != 99999 {
		t.Errorf("TMDbID = %d, want 99999", items[0].TMDbID)
	}
	if items[0].IMDbID != "" {
		t.Errorf("IMDbID = %q, want empty", items[0].IMDbID)
	}
}

func TestFetch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestPlugin("bad-tok", srv.URL)
	_, err := p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() expected error for 401, got nil")
	}
}

func TestFetch_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	_, err := p.Fetch(context.Background())
	if err == nil {
		t.Fatal("Fetch() expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------------

func TestTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	if err := p.Test(context.Background()); err != nil {
		t.Fatalf("Test() = %v", err)
	}
}

func TestTest_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	if err := p.Test(context.Background()); err == nil {
		t.Fatal("Test() expected error for 403, got nil")
	}
}

// ---------------------------------------------------------------------------
// Sanitizer
// ---------------------------------------------------------------------------

func TestSanitizer_RedactsAccessToken(t *testing.T) {
	raw := json.RawMessage(`{"access_token":"super-secret"}`)
	out := registry.Default.SanitizeImportListSettings("plex_watchlist", raw)

	var result map[string]string
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("unmarshal sanitized output: %v", err)
	}
	if result["access_token"] != "***" {
		t.Errorf("access_token = %q, want ***", result["access_token"])
	}
}

func TestSanitizer_MalformedJSON(t *testing.T) {
	out := registry.Default.SanitizeImportListSettings("plex_watchlist", json.RawMessage(`bad`))
	if string(out) != "{}" {
		t.Errorf("sanitizer with bad JSON = %s, want {}", string(out))
	}
}

// ---------------------------------------------------------------------------
// TMDb lookup fallback via SetTMDBClient
// ---------------------------------------------------------------------------

func TestFetch_TMDBFallback_ResolvesViaIMDbID(t *testing.T) {
	// Movie has only an imdb GUID; plugin should call tmdbLookup and get a TMDb ID back.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []map[string]any{
					{
						"title": "Obscure Film",
						"year":  2015,
						"type":  "movie",
						"guid":  "plex://movie/obs",
						"Guid":  []map[string]any{{"id": "imdb://tt5555555"}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)

	// Inject a mock TMDb finder that maps tt5555555 → TMDb ID 111111.
	p.tmdbLookup = func(_ context.Context, imdbID string) (int, string, error) {
		if imdbID == "tt5555555" {
			return 111111, "", nil
		}
		return 0, "", nil
	}

	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].TMDbID != 111111 {
		t.Errorf("TMDbID = %d, want 111111 (resolved via TMDb fallback)", items[0].TMDbID)
	}
	if items[0].IMDbID != "tt5555555" {
		t.Errorf("IMDbID = %q, want tt5555555", items[0].IMDbID)
	}
}

func TestFetch_TMDBFallback_SkipsIfLookupFails(t *testing.T) {
	// Movie has only an imdb GUID; tmdbLookup returns 0 → item must be skipped.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"MediaContainer": map[string]any{
				"Metadata": []map[string]any{
					{
						"title": "Unresolvable",
						"year":  2018,
						"type":  "movie",
						"guid":  "plex://movie/unr",
						"Guid":  []map[string]any{{"id": "imdb://tt0000000"}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	p := newTestPlugin("tok", srv.URL)
	p.tmdbLookup = func(_ context.Context, _ string) (int, string, error) {
		return 0, "", nil // not found
	}

	items, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0 (unresolvable movie must be skipped)", len(items))
	}
}
