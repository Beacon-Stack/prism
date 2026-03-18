package tmdbcollection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/luminarr/luminarr/internal/metadata/tmdb"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("tmdb_collection", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("tmdb_collection: invalid settings: %w", err)
		}
		if cfg.CollectionID == 0 {
			return nil, fmt.Errorf("tmdb_collection: collection_id is required")
		}
		return &Plugin{cfg: cfg}, nil
	})
}

// Config holds the settings for the TMDB collection plugin.
type Config struct {
	CollectionID int `json:"collection_id"`
}

// Plugin fetches all movies in a specific TMDB collection (franchise).
type Plugin struct {
	cfg    Config
	client *tmdb.Client
}

func (p *Plugin) Name() string { return "TMDB Collection" }

func (p *Plugin) SetTMDBClient(c any) {
	if tc, ok := c.(*tmdb.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("TMDB client not configured")
	}
	franchise, err := p.client.GetFranchise(ctx, p.cfg.CollectionID)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.ImportListItem, 0, len(franchise.Parts))
	for _, part := range franchise.Parts {
		items = append(items, plugin.ImportListItem{
			TMDbID:     part.TMDBID,
			Title:      part.Title,
			Year:       part.Year,
			PosterPath: part.PosterPath,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("TMDB client not configured")
	}
	_, err := p.client.GetFranchise(ctx, p.cfg.CollectionID)
	return err
}
