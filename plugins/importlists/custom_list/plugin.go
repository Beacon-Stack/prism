package customlist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/safedialer"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("custom_list", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("custom_list: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("custom_list: url is required")
		}
		return &Plugin{
			cfg: cfg,
			http: &http.Client{
				Transport: safedialer.Transport(),
				Timeout:   30 * time.Second,
			},
		}, nil
	})
}

// Config holds the settings for the Custom List plugin.
type Config struct {
	URL string `json:"url"`
}

// Plugin fetches movies from a user-provided JSON URL.
// The URL must return a JSON array of objects, each with at least a "tmdb" or
// "tmdb_id" field (integer). Optional fields: "title", "year", "imdb"/"imdb_id".
type Plugin struct {
	cfg  Config
	http *http.Client
}

func (p *Plugin) Name() string { return "Custom JSON List" }

// rawItem is a flexible representation of a single list item.
// We accept both "tmdb" and "tmdb_id" keys, same for imdb.
type rawItem struct {
	TMDB   int    `json:"tmdb"`
	TMDbID int    `json:"tmdb_id"`
	IMDB   string `json:"imdb"`
	IMDbID string `json:"imdb_id"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
}

func (r rawItem) tmdbID() int {
	if r.TMDB != 0 {
		return r.TMDB
	}
	return r.TMDbID
}

func (r rawItem) imdbID() string {
	if r.IMDB != "" {
		return r.IMDB
	}
	return r.IMDbID
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	items, err := p.fetch(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]plugin.ImportListItem, 0, len(items))
	for _, it := range items {
		tmdbID := it.tmdbID()
		if tmdbID == 0 {
			continue
		}
		out = append(out, plugin.ImportListItem{
			TMDbID: tmdbID,
			IMDbID: it.imdbID(),
			Title:  it.Title,
			Year:   it.Year,
		})
	}
	return out, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	items, err := p.fetch(ctx)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("custom_list: URL returned empty list")
	}
	if items[0].tmdbID() == 0 {
		return fmt.Errorf("custom_list: first item has no tmdb or tmdb_id field")
	}
	return nil
}

func (p *Plugin) fetch(ctx context.Context) ([]rawItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("custom_list: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("custom_list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("custom_list: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Limit to 10 MB to prevent abuse.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("custom_list: read: %w", err)
	}

	var items []rawItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("custom_list: expected JSON array: %w", err)
	}
	return items, nil
}
