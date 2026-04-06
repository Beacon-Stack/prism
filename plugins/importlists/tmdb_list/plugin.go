package tmdblist

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-media/prism/internal/metadata/tmdb"
	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("tmdb_list", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("tmdb_list: invalid settings: %w", err)
		}
		if cfg.ListID == "" {
			return nil, fmt.Errorf("tmdb_list: list_id is required")
		}
		return &Plugin{cfg: cfg}, nil
	})
}

// Config holds the settings for the TMDB list plugin.
type Config struct {
	ListID string `json:"list_id"`
}

// Plugin fetches movies from a specific TMDB user list.
type Plugin struct {
	cfg    Config
	client *tmdb.Client
}

func (p *Plugin) Name() string { return "TMDB List" }

func (p *Plugin) SetTMDBClient(c any) {
	if tc, ok := c.(*tmdb.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("TMDB client not configured")
	}
	results, err := p.client.GetList(ctx, p.cfg.ListID, 1)
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
	_, err := p.client.GetList(ctx, p.cfg.ListID, 1)
	return err
}
