package tmdbperson

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-media/prism/internal/metadata/tmdb"
	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("tmdb_person", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("tmdb_person: invalid settings: %w", err)
		}
		if cfg.PersonID == 0 {
			return nil, fmt.Errorf("tmdb_person: person_id is required")
		}
		if cfg.PersonType == "" {
			cfg.PersonType = "director"
		}
		if cfg.PersonType != "actor" && cfg.PersonType != "director" {
			return nil, fmt.Errorf("tmdb_person: person_type must be 'actor' or 'director'")
		}
		return &Plugin{cfg: cfg}, nil
	})
}

// Config holds the settings for the TMDB person plugin.
type Config struct {
	PersonID   int    `json:"person_id"`
	PersonType string `json:"person_type"` // "actor" or "director"
}

// Plugin fetches the filmography of a specific TMDB person.
type Plugin struct {
	cfg    Config
	client *tmdb.Client
}

func (p *Plugin) Name() string { return "TMDB Person" }

func (p *Plugin) SetTMDBClient(c any) {
	if tc, ok := c.(*tmdb.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("TMDB client not configured")
	}
	filmography, err := p.client.GetPersonFilmography(ctx, p.cfg.PersonID, p.cfg.PersonType)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.ImportListItem, 0, len(filmography))
	for _, f := range filmography {
		items = append(items, plugin.ImportListItem{
			TMDbID:     f.TMDBID,
			Title:      f.Title,
			Year:       f.Year,
			PosterPath: f.PosterPath,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("TMDB client not configured")
	}
	_, err := p.client.GetPersonFilmography(ctx, p.cfg.PersonID, p.cfg.PersonType)
	return err
}
