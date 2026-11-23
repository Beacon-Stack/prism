// Package torznab implements the Torznab indexer plugin for Prism.
// Torznab is an RSS/Atom-style XML protocol used by Prowlarr and Jackett
// to expose torrent indexers over a common API.
package torznab

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/safedialer"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterIndexer("torznab", func(settings json.RawMessage) (plugin.Indexer, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("torznab: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("torznab: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterIndexerSanitizer("torznab", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["api_key"]; ok {
			m["api_key"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for a Torznab indexer instance.
type Config struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key,omitempty"`
}

// Indexer is a Torznab indexer plugin instance.
type Indexer struct {
	cfg    Config
	client *http.Client

	// Cached movie-search caps (see movieCaps). capsCheckAt is when to
	// ask the indexer again.
	capsMu      sync.Mutex
	caps        *movieCaps
	capsCheckAt time.Time
}

// movieCaps is the part of an indexer's caps document a movie search needs.
type movieCaps struct {
	// params are the movie-search supportedParams, lowercased ("q",
	// "imdbid", "tmdbid"). Empty when the indexer didn't list them.
	params map[string]bool
	// unavailable: the caps say movie-search is off, so an ID search could
	// only cost a request.
	unavailable bool
}

const (
	// capsTTL is how long a caps document is trusted. Caps change only
	// when the indexer's definition does.
	capsTTL = 6 * time.Hour
	// capsRetryAfter is how long to search without caps after the caps
	// request failed, before asking again.
	capsRetryAfter = 10 * time.Minute
	// capsTimeout bounds the caps request made ahead of a search.
	capsTimeout = 5 * time.Second
)

// New creates a new Torznab Indexer from the given config.
func New(cfg Config) *Indexer {
	return &Indexer{
		cfg: cfg,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: safedialer.LANTransport(),
		},
	}
}

// Name returns the human-readable plugin name.
func (idx *Indexer) Name() string { return "Torznab" }

// Protocol returns the protocol this indexer serves.
func (idx *Indexer) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

// Capabilities fetches and parses the indexer's capabilities document.
func (idx *Indexer) Capabilities(ctx context.Context) (plugin.Capabilities, error) {
	caps, err := idx.fetchCaps(ctx)
	if err != nil {
		return plugin.Capabilities{}, err
	}

	categories := make([]int, 0, len(caps.Categories.Items))
	for _, c := range caps.Categories.Items {
		categories = append(categories, c.ID)
	}

	return plugin.Capabilities{
		SearchAvailable:   caps.Searching.Search.Available == "yes",
		TVSearchAvailable: caps.Searching.TVSearch.Available == "yes",
		MovieSearch:       caps.Searching.MovieSearch.Available == "yes",
		Categories:        categories,
	}, nil
}

func (idx *Indexer) fetchCaps(ctx context.Context) (capsResponse, error) {
	u := idx.buildURL("caps", url.Values{})
	resp, err := idx.get(ctx, u)
	if err != nil {
		return capsResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return capsResponse{}, fmt.Errorf("torznab: capabilities returned HTTP %d", resp.StatusCode)
	}

	var caps capsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&caps); err != nil {
		return capsResponse{}, fmt.Errorf("torznab: decoding capabilities: %w", err)
	}
	return caps, nil
}

// movieSearchCaps returns the cached movie-search caps, fetching them when
// there are none or they are old. Nil when the indexer can't say: the ID
// search then goes out as it always did, with the TMDB ID.
func (idx *Indexer) movieSearchCaps(ctx context.Context) *movieCaps {
	idx.capsMu.Lock()
	defer idx.capsMu.Unlock()
	if time.Now().Before(idx.capsCheckAt) {
		return idx.caps
	}
	cctx, cancel := context.WithTimeout(ctx, capsTimeout)
	defer cancel()
	doc, err := idx.fetchCaps(cctx)
	if err != nil {
		// Keep whatever we had; a caps hiccup must not change searches.
		idx.capsCheckAt = time.Now().Add(capsRetryAfter)
		return idx.caps
	}
	mc := &movieCaps{
		params:      map[string]bool{},
		unavailable: strings.EqualFold(strings.TrimSpace(doc.Searching.MovieSearch.Available), "no"),
	}
	for _, p := range strings.Split(doc.Searching.MovieSearch.SupportedParams, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			mc.params[p] = true
		}
	}
	idx.caps, idx.capsCheckAt = mc, time.Now().Add(capsTTL)
	return idx.caps
}

// idParams are the IDs to send in a movie search, or nil to skip it.
// Only IDs the indexer lists are sent: one that doesn't know an ID can
// answer with its recent-uploads feed or nothing. Without caps, the TMDB
// ID goes out as before.
func idParams(caps *movieCaps, q plugin.SearchQuery) url.Values {
	params := url.Values{}
	if caps == nil {
		if q.TMDBID != 0 {
			params.Set("tmdbid", strconv.Itoa(q.TMDBID))
			return params
		}
		return nil
	}
	if caps.unavailable {
		return nil
	}
	if q.IMDBID != "" && caps.params["imdbid"] {
		params.Set("imdbid", q.IMDBID)
	}
	if q.TMDBID != 0 && caps.params["tmdbid"] {
		params.Set("tmdbid", strconv.Itoa(q.TMDBID))
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

// Search queries the indexer for releases matching q.
//
// Strategy: run BOTH the tmdbid lookup and the free-text search, then dedupe
// by GUID. Doing both is necessary because some Torznab/Prowlarr indexers
// silently ignore the `tmdbid=` parameter and return the recent-uploads feed
// instead (since `?t=movie` with no params IS the recent feed). The old
// "try tmdbid first, fall back if zero results" strategy was fooled by
// these indexers — the recent feed is non-empty so the fall-through never
// triggered, and search results were polluted with random recent uploads.
//
// Indexers that properly honour `tmdbid` contribute precise hits; indexers
// that return the recent feed contribute noise that's filtered downstream
// by the title-match guard in autosearch/releases. The text search is the
// safety net that always runs.
func (idx *Indexer) Search(ctx context.Context, q plugin.SearchQuery) ([]plugin.Release, error) {
	seen := make(map[string]bool)
	var all []plugin.Release
	var firstErr error

	addAll := func(rels []plugin.Release) {
		for _, r := range rels {
			key := r.GUID
			if key == "" {
				key = r.Title // fall back when GUID missing
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, r)
		}
	}

	// The ID search and the title search go out together: one after the
	// other doubled each indexer's time, and for a Cloudflare-walled
	// indexer each request is a full challenge solve.
	type answer struct {
		releases []plugin.Release
		err      error
	}
	var idAns, textAns answer
	var wg sync.WaitGroup

	// 1. ID search, with the IDs this indexer says it understands.
	if params := idParams(idx.movieSearchCaps(ctx), q); params != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idAns.releases, idAns.err = idx.fetchReleases(ctx, idx.buildURL("movie", params))
		}()
	}

	// 2. Free-text search — always runs, both as the primary path for
	// indexers that don't support IDs and as a safety net for indexers
	// that return the recent feed instead. No category filter: many indexers
	// return empty when cat=2000 is specified, and the title+year query is
	// specific enough without it.
	if q.Query != "" {
		query := q.Query
		if q.Year != 0 {
			query = fmt.Sprintf("%s %d", query, q.Year)
		}
		params := url.Values{}
		params.Set("q", query)
		wg.Add(1)
		go func() {
			defer wg.Done()
			textAns.releases, textAns.err = idx.fetchReleases(ctx, idx.buildURL("search", params))
		}()
	}
	wg.Wait()

	// ID hits first, so they win the dedupe.
	for _, a := range []answer{idAns, textAns} {
		if a.err != nil {
			if firstErr == nil {
				firstErr = a.err
			}
			continue
		}
		addAll(a.releases)
	}

	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

// GetRecent returns the most recent releases from the indexer's RSS feed.
func (idx *Indexer) GetRecent(ctx context.Context) ([]plugin.Release, error) {
	u := idx.buildURL("movie", url.Values{})
	return idx.fetchReleases(ctx, u)
}

// Test checks that the indexer is reachable and returns a valid capabilities
// response. Returns a non-nil error if connectivity or configuration fails.
func (idx *Indexer) Test(ctx context.Context) error {
	u := idx.buildURL("caps", url.Values{})
	resp, err := idx.get(ctx, u)
	if err != nil {
		return fmt.Errorf("torznab: test request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torznab: test returned HTTP %d", resp.StatusCode)
	}

	var caps capsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&caps); err != nil {
		// EOF means the body was empty — the indexer answered HTTP 200 but
		// returned no caps document. This is acceptable: some Prowlarr-backed
		// indexers omit caps while still serving search results. Only reject
		// responses that are clearly not XML at all (e.g. a JSON REST API).
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("torznab: invalid caps response (wrong URL?): %w", err)
		}
	}
	return nil
}

// buildURL constructs the full API URL for the given Torznab function and
// additional query parameters. The apikey is appended when configured.
// Trailing slashes on the configured URL are stripped so callers can enter
// either "http://host/5" or "http://host/5/" and both produce valid URLs.
func (idx *Indexer) buildURL(t string, params url.Values) string {
	params.Set("t", t)
	if idx.cfg.APIKey != "" {
		params.Set("apikey", idx.cfg.APIKey)
	}
	base := strings.TrimRight(idx.cfg.URL, "/")
	return base + "/api?" + params.Encode()
}

// get executes an HTTP GET against the given URL using the context for
// cancellation and deadline propagation.
func (idx *Indexer) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("torznab: building request: %w", err)
	}
	return idx.client.Do(req)
}

// fetchReleases performs a GET to rawURL, decodes the Torznab RSS response,
// and returns the parsed releases.
func (idx *Indexer) fetchReleases(ctx context.Context, rawURL string) ([]plugin.Release, error) {
	resp, err := idx.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torznab: search returned HTTP %d", resp.StatusCode)
	}

	var feed rssResponse
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("torznab: empty response body — check the API key and indexer URL")
		}
		return nil, fmt.Errorf("torznab: decoding response: %w", err)
	}

	releases := make([]plugin.Release, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		releases = append(releases, idx.toRelease(item))
	}
	return releases, nil
}

// toRelease converts a parsed RSS item into a plugin.Release.
func (idx *Indexer) toRelease(item rssItem) plugin.Release {
	guid := item.GUID.Value
	if guid == "" {
		guid = item.Link
	}

	r := plugin.Release{
		GUID:        guid,
		Title:       item.Title,
		Indexer:     item.JackettIndexer, // Prowlarr fills this; empty for standalone indexers.
		Protocol:    plugin.ProtocolTorrent,
		DownloadURL: item.Enclosure.URL,
		InfoURL:     item.Link,
		Size:        item.Enclosure.Length,
	}

	// Extract torznab-specific attributes.
	var flags []plugin.IndexerFlag
	for _, attr := range item.Attrs {
		switch attr.Name {
		case "seeders":
			r.Seeds, _ = strconv.Atoi(attr.Value)
		case "peers":
			r.Peers, _ = strconv.Atoi(attr.Value)
		case "infohash":
			// BitTorrent v1 info hash. Most torznab indexers (Prowlarr,
			// Jackett) populate this even when the download URL is an HTTP
			// redirect that hides the underlying magnet — without it, our
			// cross-indexer dedup falls back to URL parsing and misses
			// most duplicates.
			r.InfoHash = strings.ToLower(strings.TrimSpace(attr.Value))
		case "size":
			// Prefer the torznab size attribute over enclosure length when present.
			if sz, err := strconv.ParseInt(attr.Value, 10, 64); err == nil && sz > 0 {
				r.Size = sz
			}
		case "downloadvolumefactor":
			switch attr.Value {
			case "0":
				flags = append(flags, plugin.FlagFreeleech)
			case "0.5":
				flags = append(flags, plugin.FlagHalfleech)
			case "0.75":
				flags = append(flags, plugin.FlagFreeleech25)
			case "0.25":
				flags = append(flags, plugin.FlagFreeleech75)
			}
		case "uploadvolumefactor":
			if attr.Value == "2" {
				flags = append(flags, plugin.FlagDoubleUpload)
			}
		case "tag":
			switch strings.ToLower(attr.Value) {
			case "freeleech":
				flags = append(flags, plugin.FlagFreeleech)
			case "halfleech":
				flags = append(flags, plugin.FlagHalfleech)
			case "internal":
				flags = append(flags, plugin.FlagInternal)
			case "scene":
				flags = append(flags, plugin.FlagScene)
			case "nuked":
				flags = append(flags, plugin.FlagNuked)
			}
		}
	}
	r.IndexerFlags = dedupeFlags(flags)

	r.AgeDays = parseAgeDays(item.PubDate)
	return r
}

// dedupeFlags removes duplicate flags while preserving order.
func dedupeFlags(flags []plugin.IndexerFlag) []plugin.IndexerFlag {
	if len(flags) == 0 {
		return nil
	}
	seen := make(map[plugin.IndexerFlag]struct{}, len(flags))
	out := make([]plugin.IndexerFlag, 0, len(flags))
	for _, f := range flags {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			out = append(out, f)
		}
	}
	return out
}

// parseAgeDays parses an RFC1123Z pubDate string and returns the number of
// calendar days elapsed since that time. Returns 0 on parse failure.
func parseAgeDays(pubDate string) float64 {
	if pubDate == "" {
		return 0
	}
	t, err := time.Parse(time.RFC1123Z, pubDate)
	if err != nil {
		// Try RFC1123 as a fallback (some indexers omit the timezone offset).
		t, err = time.Parse(time.RFC1123, pubDate)
		if err != nil {
			return 0
		}
	}
	return time.Since(t).Hours() / 24
}

// ---------------------------------------------------------------------------
// XML types
// ---------------------------------------------------------------------------

// rssResponse is the top-level RSS document returned by Torznab.
type rssResponse struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

// rssItem represents a single release entry in the Torznab RSS feed.
type rssItem struct {
	Title     string    `xml:"title"`
	GUID      rssGUID   `xml:"guid"`
	Link      string    `xml:"link"`
	PubDate   string    `xml:"pubDate"`
	Enclosure enclosure `xml:"enclosure"`
	// Prowlarr/Jackett include the upstream indexer name per item.
	JackettIndexer string `xml:"jackettindexer"`
	// Torznab attributes use the namespace http://torznab.com/schemas/2015/feed.
	Attrs []torznabAttr `xml:"http://torznab.com/schemas/2015/feed attr"`
}

// rssGUID holds the guid element, which may carry an isPermaLink attribute.
type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// enclosure is the RSS enclosure element that carries the download URL and size.
type enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// torznabAttr is a <torznab:attr name="..." value="..."/> element.
type torznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// capsResponse is the top-level capabilities document.
type capsResponse struct {
	XMLName    xml.Name       `xml:"caps"`
	Searching  capsSearching  `xml:"searching"`
	Categories capsCategories `xml:"categories"`
}

type capsSearching struct {
	Search      capsSearchType `xml:"search"`
	TVSearch    capsSearchType `xml:"tv-search"`
	MovieSearch capsSearchType `xml:"movie-search"`
}

type capsSearchType struct {
	Available       string `xml:"available,attr"`
	SupportedParams string `xml:"supportedParams,attr"`
}

type capsCategories struct {
	Items []capsCategory `xml:"category"`
}

type capsCategory struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}
