package trakt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"log/slog"
)

// newTestClient returns a Client pointed at the given test server URL.
func newTestClient(serverURL string) *Client {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return New("test-api-key", logger).
		WithBaseURL(serverURL).
		WithHTTPClient(&http.Client{})
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

// checkTraktHeaders verifies the three mandatory Trakt API headers are present.
func checkTraktHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if v := r.Header.Get("Content-Type"); v != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", v)
	}
	if v := r.Header.Get("trakt-api-version"); v != apiVersion {
		t.Errorf("trakt-api-version = %q, want %s", v, apiVersion)
	}
	if v := r.Header.Get("trakt-api-key"); v != "test-api-key" {
		t.Errorf("trakt-api-key = %q, want test-api-key", v)
	}
}

// ---------------------------------------------------------------------------
// GetWatchlist
// ---------------------------------------------------------------------------

func TestGetWatchlist_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/jdoe/watchlist/movies" {
			t.Errorf("path = %q, want /users/jdoe/watchlist/movies", r.URL.Path)
		}
		checkTraktHeaders(t, r)

		resp := []map[string]any{
			{
				"movie": map[string]any{
					"title": "Inception",
					"year":  2010,
					"ids": map[string]any{
						"trakt": 1,
						"tmdb":  27205,
						"imdb":  "tt1375666",
					},
				},
			},
			{
				"movie": map[string]any{
					"title": "Interstellar",
					"year":  2014,
					"ids": map[string]any{
						"trakt": 2,
						"tmdb":  157336,
						"imdb":  "tt0816692",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	movies, err := c.GetWatchlist(context.Background(), "jdoe")
	if err != nil {
		t.Fatalf("GetWatchlist() error = %v", err)
	}
	if len(movies) != 2 {
		t.Fatalf("len(movies) = %d, want 2", len(movies))
	}

	got := movies[0]
	if got.Title != "Inception" {
		t.Errorf("Title = %q, want Inception", got.Title)
	}
	if got.Year != 2010 {
		t.Errorf("Year = %d, want 2010", got.Year)
	}
	if got.IDs.TMDB != 27205 {
		t.Errorf("IDs.TMDB = %d, want 27205", got.IDs.TMDB)
	}
	if got.IDs.IMDB != "tt1375666" {
		t.Errorf("IDs.IMDB = %q, want tt1375666", got.IDs.IMDB)
	}
}

func TestGetWatchlist_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	movies, err := c.GetWatchlist(context.Background(), "nobody")
	if err != nil {
		t.Fatalf("GetWatchlist() error = %v", err)
	}
	if len(movies) != 0 {
		t.Errorf("len(movies) = %d, want 0", len(movies))
	}
}

func TestGetWatchlist_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetWatchlist(context.Background(), "jdoe")
	if err == nil {
		t.Fatal("GetWatchlist() expected error for 401, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetCustomList
// ---------------------------------------------------------------------------

func TestGetCustomList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/users/jdoe/lists/sci-fi/items/movies"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %s", r.URL.Path, wantPath)
		}
		checkTraktHeaders(t, r)

		resp := []map[string]any{
			{
				"movie": map[string]any{
					"title": "Dune",
					"year":  2021,
					"ids": map[string]any{
						"trakt": 3,
						"tmdb":  438631,
						"imdb":  "tt1160419",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	movies, err := c.GetCustomList(context.Background(), "jdoe", "sci-fi")
	if err != nil {
		t.Fatalf("GetCustomList() error = %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}
	if movies[0].Title != "Dune" {
		t.Errorf("Title = %q, want Dune", movies[0].Title)
	}
	if movies[0].IDs.TMDB != 438631 {
		t.Errorf("IDs.TMDB = %d, want 438631", movies[0].IDs.TMDB)
	}
}

func TestGetCustomList_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetCustomList(context.Background(), "jdoe", "missing-list")
	if err == nil {
		t.Fatal("GetCustomList() expected error for 404, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetPopular
// ---------------------------------------------------------------------------

func TestGetPopular_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movies/popular" {
			t.Errorf("path = %q, want /movies/popular", r.URL.Path)
		}
		checkTraktHeaders(t, r)

		resp := []map[string]any{
			{
				"title": "The Dark Knight",
				"year":  2008,
				"ids": map[string]any{
					"trakt": 10,
					"tmdb":  155,
					"imdb":  "tt0468569",
				},
			},
			{
				"title": "Oppenheimer",
				"year":  2023,
				"ids": map[string]any{
					"trakt": 11,
					"tmdb":  872585,
					"imdb":  "tt15398776",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustMarshal(t, resp))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	movies, err := c.GetPopular(context.Background())
	if err != nil {
		t.Fatalf("GetPopular() error = %v", err)
	}
	if len(movies) != 2 {
		t.Fatalf("len(movies) = %d, want 2", len(movies))
	}
	if movies[0].Title != "The Dark Knight" {
		t.Errorf("Title = %q, want The Dark Knight", movies[0].Title)
	}
	if movies[0].IDs.TMDB != 155 {
		t.Errorf("IDs.TMDB = %d, want 155", movies[0].IDs.TMDB)
	}
}

func TestGetPopular_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetPopular(context.Background())
	if err == nil {
		t.Fatal("GetPopular() expected error for 500, got nil")
	}
}

func TestGetPopular_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetPopular(context.Background())
	if err == nil {
		t.Fatal("GetPopular() expected error for malformed JSON, got nil")
	}
}
