package tmdbtoprated

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-media/prism/internal/metadata/tmdb"
	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("tmdb_top_rated", func(_ json.RawMessage) (plugin.ImportList, error) {
		return &Plugin{}, nil
	})
}

// Plugin fetches the all-time highest rated movies from TMDB.
type Plugin struct {
	client *tmdb.Client
}

func (p *Plugin) Name() string { return "TMDB Top Rated" }

func (p *Plugin) SetTMDBClient(c any) {
	if tc, ok := c.(*tmdb.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("TMDB client not configured")
	}
	results, err := p.client.GetTopRatedMovies(ctx, 1)
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
	_, err := p.client.GetTopRatedMovies(ctx, 1)
	return err
}
