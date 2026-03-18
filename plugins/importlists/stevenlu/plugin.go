package stevenlu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/pkg/plugin"
)

const dataURL = "https://popular-movies-data.stevenlu.com/movies.json"

func init() {
	registry.Default.RegisterImportList("stevenlu", func(_ json.RawMessage) (plugin.ImportList, error) {
		return &Plugin{
			http: &http.Client{
				Transport: safedialer.Transport(),
				Timeout:   30 * time.Second,
			},
		}, nil
	})
}

// Plugin fetches the StevenLu popular movies list — a curated, daily-updated
// JSON file of currently popular movies. No API key or configuration required.
type Plugin struct {
	http *http.Client
}

func (p *Plugin) Name() string { return "StevenLu Popular Movies" }

type movie struct {
	Title  string `json:"title"`
	TmdbID int    `json:"tmdb_id"`
	ImdbID string `json:"imdb_id"`
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	movies, err := p.fetch(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.ImportListItem, 0, len(movies))
	for _, m := range movies {
		if m.TmdbID == 0 {
			continue
		}
		items = append(items, plugin.ImportListItem{
			TMDbID: m.TmdbID,
			IMDbID: m.ImdbID,
			Title:  m.Title,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	movies, err := p.fetch(ctx)
	if err != nil {
		return err
	}
	if len(movies) == 0 {
		return fmt.Errorf("stevenlu: returned empty list")
	}
	return nil
}

func (p *Plugin) fetch(ctx context.Context) ([]movie, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("stevenlu: %w", err)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stevenlu: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stevenlu: HTTP %d", resp.StatusCode)
	}

	var movies []movie
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, fmt.Errorf("stevenlu: decode: %w", err)
	}
	return movies, nil
}
