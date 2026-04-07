package tmdbtrending

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-stack/prism/internal/metadata/tmdb"
	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("tmdb_trending", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if len(settings) > 0 && string(settings) != "{}" {
			if err := json.Unmarshal(settings, &cfg); err != nil {
				return nil, fmt.Errorf("tmdb_trending: invalid settings: %w", err)
			}
		}
		if cfg.Window == "" {
			cfg.Window = "week"
		}
		if cfg.Window != "day" && cfg.Window != "week" {
			return nil, fmt.Errorf("tmdb_trending: window must be 'day' or 'week'")
		}
		return &Plugin{cfg: cfg}, nil
	})
}

// Config holds the settings for the TMDB trending plugin.
type Config struct {
	Window string `json:"window"` // "day" or "week"
}

// Plugin fetches trending movies from TMDB.
type Plugin struct {
	cfg    Config
	client *tmdb.Client
}

func (p *Plugin) Name() string { return "TMDB Trending" }

func (p *Plugin) SetTMDBClient(c any) {
	if tc, ok := c.(*tmdb.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("TMDB client not configured")
	}
	results, err := p.client.GetTrendingMovies(ctx, p.cfg.Window, 1)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.ImportListItem, 0, len(results))
	for _, r := range results {
		items = append(items, plugin.ImportListItem{
			TMDbID:     r.ID,
			Title:      r.Title,
			Year:       r.Year,
			PosterPath: r.PosterPath,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("TMDB client not configured")
	}
	_, err := p.client.GetTrendingMovies(ctx, p.cfg.Window, 1)
	return err
}
