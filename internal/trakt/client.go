// Package trakt provides a minimal HTTP client for the Trakt API v2.
// Only the endpoints needed for import lists are implemented.
package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/beacon-media/prism/internal/safedialer"
)

const (
	baseURL     = "https://api.trakt.tv"
	apiVersion  = "2"
	httpTimeout = 30 * time.Second
)

// Client is a Trakt API v2 HTTP client.
type Client struct {
	clientID string
	baseURL  string // overridable in tests; defaults to baseURL constant
	http     *http.Client
	logger   *slog.Logger
}

// New creates a new Trakt client. clientID is the Trakt API key
// (obtained from trakt.tv/oauth/applications). logger may be nil.
func New(clientID string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Client{
		clientID: clientID,
		baseURL:  baseURL,
		http: &http.Client{
			Timeout:   httpTimeout,
			Transport: safedialer.Transport(),
		},
		logger: logger,
	}
}

// WithBaseURL overrides the API base URL. Intended for tests only.
func (c *Client) WithBaseURL(u string) *Client {
	c.baseURL = u
	return c
}

// WithHTTPClient overrides the underlying HTTP client. Intended for tests only.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	c.http = hc
	return c
}

// Movie represents a movie from the Trakt API.
type Movie struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   struct {
		Trakt int    `json:"trakt"`
		TMDB  int    `json:"tmdb"`
		IMDB  string `json:"imdb"`
	} `json:"ids"`
}

// GetWatchlist fetches a user's movie watchlist.
func (c *Client) GetWatchlist(ctx context.Context, username string) ([]Movie, error) {
	path := fmt.Sprintf("/users/%s/watchlist/movies", username)
	var items []struct {
		Movie Movie `json:"movie"`
	}
	if err := c.get(ctx, path, &items); err != nil {
		return nil, fmt.Errorf("trakt watchlist: %w", err)
	}
	movies := make([]Movie, len(items))
	for i, item := range items {
		movies[i] = item.Movie
	}
	return movies, nil
}

// GetCustomList fetches a user's custom list items (movies only).
func (c *Client) GetCustomList(ctx context.Context, username, listSlug string) ([]Movie, error) {
	path := fmt.Sprintf("/users/%s/lists/%s/items/movies", username, listSlug)
	var items []struct {
		Movie Movie `json:"movie"`
	}
	if err := c.get(ctx, path, &items); err != nil {
		return nil, fmt.Errorf("trakt custom list: %w", err)
	}
	movies := make([]Movie, len(items))
	for i, item := range items {
		movies[i] = item.Movie
	}
	return movies, nil
}

// GetPopular fetches Trakt's popular movies list.
func (c *Client) GetPopular(ctx context.Context) ([]Movie, error) {
	var movies []Movie
	if err := c.get(ctx, "/movies/popular", &movies); err != nil {
		return nil, fmt.Errorf("trakt popular: %w", err)
	}
	return movies, nil
}

// GetTrending fetches Trakt's trending movies (most watched right now).
func (c *Client) GetTrending(ctx context.Context) ([]Movie, error) {
	var items []struct {
		Movie Movie `json:"movie"`
	}
	if err := c.get(ctx, "/movies/trending", &items); err != nil {
		return nil, fmt.Errorf("trakt trending: %w", err)
	}
	movies := make([]Movie, len(items))
	for i, item := range items {
		movies[i] = item.Movie
	}
	return movies, nil
}

// GetAnticipated fetches Trakt's most anticipated movies.
func (c *Client) GetAnticipated(ctx context.Context) ([]Movie, error) {
	var items []struct {
		Movie Movie `json:"movie"`
	}
	if err := c.get(ctx, "/movies/anticipated", &items); err != nil {
		return nil, fmt.Errorf("trakt anticipated: %w", err)
	}
	movies := make([]Movie, len(items))
	for i, item := range items {
		movies[i] = item.Movie
	}
	return movies, nil
}

// GetBoxOffice fetches Trakt's current box office movies.
func (c *Client) GetBoxOffice(ctx context.Context) ([]Movie, error) {
	var items []struct {
		Movie Movie `json:"movie"`
	}
	if err := c.get(ctx, "/movies/boxoffice", &items); err != nil {
		return nil, fmt.Errorf("trakt box office: %w", err)
	}
	movies := make([]Movie, len(items))
	for i, item := range items {
		movies[i] = item.Movie
	}
	return movies, nil
}

func (c *Client) get(ctx context.Context, path string, dst any) error {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trakt-api-version", apiVersion)
	req.Header.Set("trakt-api-key", c.clientID)

	c.logger.InfoContext(ctx, "trakt request",
		slog.String("method", http.MethodGet),
		slog.String("path", path),
	)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}
