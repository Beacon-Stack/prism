package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Browse categories shared with Pilot's Discover page.
const (
	CategoryTrending   = "trending"
	CategoryPopular    = "popular"
	CategoryTopRated   = "top-rated"
	CategoryNewOnline  = "new" // newly released digitally (streaming, rent, buy)
	CategoryComingSoon = "coming-soon"
)

// Sort orders that replace a category's own order.
const (
	SortRating  = "rating"  // highest rated first
	SortNewest  = "newest"  // latest release first
	SortRevenue = "revenue" // biggest box office first
)

// Runtime buckets.
const (
	RuntimeShort  = "short"  // under 90 minutes
	RuntimeMedium = "medium" // 90 to 120 minutes
	RuntimeLong   = "long"   // over two hours
)

// BrowseFilter is one Discover page request: a category plus optional
// filters. Trending is a fixed TMDB list and ignores the filters.
type BrowseFilter struct {
	Category   string
	Page       int
	GenreIDs   []int   // movies in every one of these genres
	ProviderID int     // TMDB watch provider (e.g. 8 = Netflix); needs Region
	MinRating  float64 // TMDB vote average, 0-10
	Region     string  // ISO 3166-1 country, e.g. "US"
	Sort       string  // SortRating, SortNewest or SortRevenue; "" keeps the category's order
	Decade     int     // 1990 = released 1990-1999; ignored by new and coming-soon, which set their own dates
	Runtime    string  // RuntimeShort, RuntimeMedium or RuntimeLong
	Language   string  // original language, ISO 639-1 (e.g. "ko")
	KeywordID  int     // TMDB keyword (e.g. 10051 = heist)
	CompanyID  int     // TMDB production company (e.g. 41077 = A24)
	Today      time.Time
}

var (
	regionRE   = regexp.MustCompile(`^[A-Z]{2}$`)
	languageRE = regexp.MustCompile(`^[a-z]{2}$`)
)

// NormalizeRegion returns a valid upper-case country code, defaulting to US.
func NormalizeRegion(r string) string {
	if len(r) == 2 {
		r = string([]byte{upper(r[0]), upper(r[1])})
	}
	if !regionRE.MatchString(r) {
		return "US"
	}
	return r
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

const day = 24 * time.Hour

// BrowseQuery turns a filter into the TMDB path and query parameters.
// Kept free of I/O so the date windows and filters are easy to test.
func BrowseQuery(f BrowseFilter) (string, url.Values, error) {
	p := url.Values{}
	page := f.Page
	if page < 1 {
		page = 1
	}
	p.Set("page", strconv.Itoa(page))
	region := NormalizeRegion(f.Region)
	today := f.Today
	if today.IsZero() {
		today = time.Now()
	}
	date := func(t time.Time) string { return t.UTC().Format("2006-01-02") }

	switch f.Category {
	case "", CategoryTrending:
		return "/trending/movie/week", p, nil
	case CategoryPopular:
		p.Set("sort_by", "popularity.desc")
	case CategoryTopRated:
		p.Set("sort_by", "vote_average.desc")
		p.Set("vote_count.gte", "300")
	case CategoryNewOnline:
		// Digital release (streaming, rent or buy) in the last 60 days in
		// this country; theatrical-only releases don't count.
		p.Set("sort_by", "popularity.desc")
		p.Set("region", region)
		p.Set("with_release_type", "4")
		p.Set("release_date.gte", date(today.Add(-60*day)))
		p.Set("release_date.lte", date(today))
	case CategoryComingSoon:
		p.Set("sort_by", "popularity.desc")
		p.Set("region", region)
		p.Set("primary_release_date.gte", date(today.Add(day)))
		p.Set("primary_release_date.lte", date(today.Add(120*day)))
	default:
		return "", nil, fmt.Errorf("unknown category %q", f.Category)
	}

	upcoming := f.Category == CategoryComingSoon
	ownDates := upcoming || f.Category == CategoryNewOnline

	if ids := positive(f.GenreIDs); len(ids) > 0 {
		p.Set("with_genres", strings.Join(ids, ","))
	}
	if f.KeywordID > 0 {
		p.Set("with_keywords", strconv.Itoa(f.KeywordID))
	}
	if f.CompanyID > 0 {
		p.Set("with_companies", strconv.Itoa(f.CompanyID))
	}
	if languageRE.MatchString(f.Language) {
		p.Set("with_original_language", f.Language)
	}
	switch f.Runtime {
	case RuntimeShort:
		p.Set("with_runtime.gte", "40") // leaves out short films
		p.Set("with_runtime.lte", "89")
	case RuntimeMedium:
		p.Set("with_runtime.gte", "90")
		p.Set("with_runtime.lte", "120")
	case RuntimeLong:
		p.Set("with_runtime.gte", "121")
	}
	if f.Decade >= 1900 && f.Decade <= 2100 && !ownDates {
		start := f.Decade - f.Decade%10
		p.Set("primary_release_date.gte", fmt.Sprintf("%d-01-01", start))
		p.Set("primary_release_date.lte", fmt.Sprintf("%d-12-31", start+9))
	}
	// Unreleased movies have no votes or box office to sort by, so
	// coming-soon keeps its own order for those two.
	switch {
	case f.Sort == SortRating && !upcoming:
		p.Set("sort_by", "vote_average.desc")
		if p.Get("vote_count.gte") == "" {
			p.Set("vote_count.gte", "200")
		}
	case f.Sort == SortRevenue && !upcoming:
		p.Set("sort_by", "revenue.desc")
	case f.Sort == SortNewest:
		p.Set("sort_by", "primary_release_date.desc")
		if !ownDates {
			// Newest released, not furthest in the future; the vote floor
			// keeps out the stream of entries nobody has heard of.
			if p.Get("primary_release_date.lte") == "" || p.Get("primary_release_date.lte") > date(today) {
				p.Set("primary_release_date.lte", date(today))
			}
			if p.Get("vote_count.gte") == "" {
				p.Set("vote_count.gte", "20")
			}
		}
	}
	if f.ProviderID > 0 {
		// "On Netflix" means included with a subscription (or free), not
		// available to rent there.
		p.Set("with_watch_providers", strconv.Itoa(f.ProviderID))
		p.Set("watch_region", region)
		p.Set("with_watch_monetization_types", "flatrate|free|ads")
	}
	if f.MinRating > 0 {
		p.Set("vote_average.gte", strconv.FormatFloat(f.MinRating, 'f', 1, 64))
		// A 9.0 from four votes isn't a rating; top-rated sets its own floor.
		if p.Get("vote_count.gte") == "" {
			p.Set("vote_count.gte", "50")
		}
	}
	return "/discover/movie", p, nil
}

// positive returns the IDs above zero, as strings, without repeats.
func positive(ids []int) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, strconv.Itoa(id))
		}
	}
	return out
}

// Browse fetches one page of a Discover category with filters applied.
func (c *Client) Browse(ctx context.Context, f BrowseFilter) (*PaginatedResults, error) {
	path, params, err := BrowseQuery(f)
	if err != nil {
		return nil, err
	}
	return c.fetchMovieListPaginated(ctx, path, params)
}

// WatchProvider is a streaming service TMDB knows about in a country.
type WatchProvider struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	LogoPath string `json:"logo_path"`
}

// WatchProviders lists the streaming services available for movies in a
// country, most prominent first.
func (c *Client) WatchProviders(ctx context.Context, region string) ([]WatchProvider, error) {
	region = NormalizeRegion(region)
	var envelope struct {
		Results []struct {
			ID                int            `json:"provider_id"`
			Name              string         `json:"provider_name"`
			LogoPath          string         `json:"logo_path"`
			DisplayPriorities map[string]int `json:"display_priorities"`
		} `json:"results"`
	}
	params := url.Values{}
	params.Set("watch_region", region)
	if err := c.get(ctx, "/watch/providers/movie", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb watch providers: %w", err)
	}
	return sortProviders(region, envelope.Results), nil
}

func sortProviders(region string, in []struct {
	ID                int            `json:"provider_id"`
	Name              string         `json:"provider_name"`
	LogoPath          string         `json:"logo_path"`
	DisplayPriorities map[string]int `json:"display_priorities"`
}) []WatchProvider {
	type ranked struct {
		WatchProvider
		prio int
	}
	rs := make([]ranked, 0, len(in))
	for _, r := range in {
		prio, ok := r.DisplayPriorities[region]
		if !ok {
			prio = 1 << 20
		}
		rs = append(rs, ranked{WatchProvider{ID: r.ID, Name: r.Name, LogoPath: r.LogoPath}, prio})
	}
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].prio != rs[j].prio {
			return rs[i].prio < rs[j].prio
		}
		return rs[i].Name < rs[j].Name
	})
	out := make([]WatchProvider, len(rs))
	for i, r := range rs {
		out[i] = r.WatchProvider
	}
	return out
}

// SearchPaginated searches movies by title for the Discover page, with
// the vote average in the result (like the browse lists) and paging.
func (c *Client) SearchPaginated(ctx context.Context, query string, page int) (*PaginatedResults, error) {
	if page < 1 {
		page = 1
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", strconv.Itoa(page))
	return c.fetchMovieListPaginated(ctx, "/search/movie", params)
}

// Recommendations lists the movies TMDB recommends to someone who liked
// the given one.
func (c *Client) Recommendations(ctx context.Context, tmdbID int) (*PaginatedResults, error) {
	return c.fetchMovieListPaginated(ctx, fmt.Sprintf("/movie/%d/recommendations", tmdbID), nil)
}

// FranchiseMovies lists the movies in a TMDB collection, oldest first,
// as one page of results.
func (c *Client) FranchiseMovies(ctx context.Context, collectionID int) (*PaginatedResults, error) {
	var raw struct {
		Parts []struct {
			ID            int     `json:"id"`
			Title         string  `json:"title"`
			OriginalTitle string  `json:"original_title"`
			Overview      string  `json:"overview"`
			ReleaseDate   string  `json:"release_date"`
			PosterPath    string  `json:"poster_path"`
			BackdropPath  string  `json:"backdrop_path"`
			VoteAverage   float64 `json:"vote_average"`
		} `json:"parts"`
	}
	if err := c.get(ctx, fmt.Sprintf("/collection/%d", collectionID), nil, &raw); err != nil {
		return nil, fmt.Errorf("tmdb collection %d: %w", collectionID, err)
	}
	results := make([]SearchResult, 0, len(raw.Parts))
	for _, r := range raw.Parts {
		results = append(results, SearchResult{
			ID:            r.ID,
			Title:         r.Title,
			OriginalTitle: r.OriginalTitle,
			Overview:      r.Overview,
			ReleaseDate:   r.ReleaseDate,
			Year:          parseYear(r.ReleaseDate),
			PosterPath:    r.PosterPath,
			BackdropPath:  r.BackdropPath,
			Popularity:    r.VoteAverage, // vote average, as in the browse lists
		})
	}
	// Unreleased parts have no date; they go last.
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i].ReleaseDate, results[j].ReleaseDate
		if (a == "") != (b == "") {
			return b == ""
		}
		return a < b
	})
	return &PaginatedResults{Results: results, Page: 1, TotalPages: 1}, nil
}

// SearchKeywords searches TMDB keywords by name.
func (c *Client) SearchKeywords(ctx context.Context, query string) ([]Keyword, error) {
	params := url.Values{}
	params.Set("query", query)
	var envelope struct {
		Results []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/search/keyword", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb search keywords: %w", err)
	}
	out := make([]Keyword, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		out = append(out, Keyword{ID: r.ID, Name: r.Name})
	}
	return out, nil
}

// SearchCompanies searches TMDB production companies by name.
func (c *Client) SearchCompanies(ctx context.Context, query string) ([]Company, error) {
	params := url.Values{}
	params.Set("query", query)
	var envelope struct {
		Results []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			LogoPath string `json:"logo_path"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/search/company", params, &envelope); err != nil {
		return nil, fmt.Errorf("tmdb search companies: %w", err)
	}
	out := make([]Company, 0, len(envelope.Results))
	for _, r := range envelope.Results {
		out = append(out, Company{ID: r.ID, Name: r.Name, LogoPath: r.LogoPath})
	}
	return out, nil
}
