package traktanticipated

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/trakt"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("trakt_anticipated", func(_ json.RawMessage) (plugin.ImportList, error) {
		return &Plugin{}, nil
	})
}

// Plugin fetches the most anticipated upcoming movies from Trakt.
type Plugin struct {
	client *trakt.Client
}

func (p *Plugin) Name() string { return "Trakt Anticipated Movies" }

func (p *Plugin) SetTraktClient(c any) {
	if tc, ok := c.(*trakt.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("Trakt client not configured — set trakt.client_id in config")
	}
	movies, err := p.client.GetAnticipated(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.ImportListItem, 0, len(movies))
	for _, m := range movies {
		if m.IDs.TMDB == 0 {
			continue
		}
		items = append(items, plugin.ImportListItem{
			TMDbID: m.IDs.TMDB,
			IMDbID: m.IDs.IMDB,
			Title:  m.Title,
			Year:   m.Year,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("Trakt client not configured — set trakt.client_id in config")
	}
	_, err := p.client.GetAnticipated(ctx)
	return err
}
