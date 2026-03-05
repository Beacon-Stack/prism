// Package plex implements a Luminarr media server plugin for Plex.
// On import_complete it triggers a library section refresh so the
// new movie appears immediately.
package plex

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/safedialer"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterMediaServer("plex", func(settings json.RawMessage) (plugin.MediaServer, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("plex: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("plex: url is required")
		}
		if cfg.Token == "" {
			return nil, fmt.Errorf("plex: token is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterMediaServerSanitizer("plex", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["token"]; ok {
			m["token"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for a Plex media server.
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Server is a Plex media server plugin instance.
type Server struct {
	cfg    Config
	client *http.Client
}

// New creates a new Server from the given config.
func New(cfg Config) *Server {
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Server{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
	}
}

func (s *Server) Name() string { return "Plex" }

// plexSections represents the XML response from /library/sections.
type plexSections struct {
	XMLName     xml.Name      `xml:"MediaContainer"`
	Directories []plexSection `xml:"Directory"`
}

type plexSection struct {
	Key       string         `xml:"key,attr"`
	Title     string         `xml:"title,attr"`
	Type      string         `xml:"type,attr"`
	Locations []plexLocation `xml:"Location"`
}

type plexLocation struct {
	Path string `xml:"path,attr"`
}

// RefreshLibrary triggers a refresh of the Plex library section that contains
// moviePath. If no matching section is found, it falls back to refreshing all
// movie sections.
func (s *Server) RefreshLibrary(ctx context.Context, moviePath string) error {
	sections, err := s.getSections(ctx)
	if err != nil {
		return fmt.Errorf("plex: listing sections: %w", err)
	}

	// Find sections whose location path is a prefix of moviePath.
	var matched []string
	for _, sec := range sections.Directories {
		for _, loc := range sec.Locations {
			if strings.HasPrefix(moviePath, loc.Path) {
				matched = append(matched, sec.Key)
				break
			}
		}
	}

	// Fall back: refresh all movie sections if no path match.
	if len(matched) == 0 {
		for _, sec := range sections.Directories {
			if sec.Type == "movie" {
				matched = append(matched, sec.Key)
			}
		}
	}

	if len(matched) == 0 {
		return fmt.Errorf("plex: no movie library sections found")
	}

	for _, key := range matched {
		if err := s.refreshSection(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getSections(ctx context.Context) (plexSections, error) {
	url := fmt.Sprintf("%s/library/sections", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return plexSections{}, err
	}
	req.Header.Set("X-Plex-Token", s.cfg.Token)
	req.Header.Set("Accept", "application/xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return plexSections{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return plexSections{}, fmt.Errorf("plex: sections returned %d: %s", resp.StatusCode, body)
	}

	var sections plexSections
	if err := xml.NewDecoder(resp.Body).Decode(&sections); err != nil {
		return plexSections{}, fmt.Errorf("plex: decoding sections: %w", err)
	}
	return sections, nil
}

func (s *Server) refreshSection(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/library/sections/%s/refresh", s.cfg.URL, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("plex: building refresh request: %w", err)
	}
	req.Header.Set("X-Plex-Token", s.cfg.Token)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("plex: refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("plex: refresh returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// Test verifies that the Plex server is reachable with the configured token.
func (s *Server) Test(ctx context.Context) error {
	url := s.cfg.URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("plex: building test request: %w", err)
	}
	req.Header.Set("X-Plex-Token", s.cfg.Token)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("plex: test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("plex: test returned %d: %s", resp.StatusCode, body)
	}
	return nil
}
