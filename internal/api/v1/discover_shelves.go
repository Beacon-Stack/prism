package v1

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/collection"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

// Discover's home view: rows of movies the library doesn't have yet, most
// of them picked from what the library does have. Which movies, genres and
// collections the rows start from is drawn at random from a seed, so the
// same seed gives the same rows and a new one reshuffles them.

const (
	shelfSeedPool   = 40 // newest library movies a "Because you have" row can start from
	shelfSeedRows   = 3
	shelfGenrePool  = 5 // the library's most common genres a genre row can use
	shelfGenreRows  = 2
	shelfGenrePages = 3 // a genre row shows one of the first pages of its list
	shelfCollRows   = 2
	shelfSize       = 20
	// A row with fewer movies than this isn't worth showing. Collection
	// rows are exempt: one missing movie is the point.
	shelfMinSize = 4
)

type discoverShelvesInput struct {
	Region string `query:"region" default:"US" doc:"Country for release dates (ISO 3166-1)"`
	Seed   int64  `query:"seed" doc:"Picks which library movies, genres and collections the rows start from; the same seed gives the same rows. 0 picks at random."`
}

type discoverShelfBody struct {
	ID      string               `json:"id"                doc:"Stable within one response"`
	Kind    string               `json:"kind"              enum:"because,genre,collection,list" doc:"What the row is built from"`
	Title   string               `json:"title"             doc:"Row heading"`
	SeeAll  string               `json:"see_all,omitempty" doc:"App path that shows the whole list"`
	Results []discoverResultBody `json:"results"           doc:"Movies not in the library"`
}

type discoverShelvesBody struct {
	Shelves []discoverShelfBody `json:"shelves"`
}

type discoverShelvesOutput struct {
	Body *discoverShelvesBody
}

// shelfPlan is one row before its movies are fetched.
type shelfPlan struct {
	id, kind, title, seeAll string
	minSize                 int
	fetch                   func(context.Context) ([]tmdb.SearchResult, error)
}

// pickN returns up to n items chosen at random, in random order.
func pickN[T any](rng *rand.Rand, items []T, n int) []T {
	if n > len(items) {
		n = len(items)
	}
	out := make([]T, 0, n)
	for _, i := range rng.Perm(len(items))[:n] {
		out = append(out, items[i])
	}
	return out
}

// topGenres returns the library's most common genres that TMDB has an ID
// for, most common first, at most n. Ties go alphabetically so the result
// doesn't depend on map order.
func topGenres(counts map[string]int, known []tmdb.Genre, n int) []tmdb.Genre {
	out := make([]tmdb.Genre, 0, len(known))
	for _, g := range known {
		if counts[g.Name] > 0 {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if counts[out[i].Name] != counts[out[j].Name] {
			return counts[out[i].Name] > counts[out[j].Name]
		}
		return out[i].Name < out[j].Name
	})
	return firstN(out, n)
}

// interleave merges the groups round-robin: the first of each, then the
// second of each, and so on.
func interleave(groups ...[]shelfPlan) []shelfPlan {
	var out []shelfPlan
	for i := 0; ; i++ {
		added := false
		for _, g := range groups {
			if i < len(g) {
				out = append(out, g[i])
				added = true
			}
		}
		if !added {
			return out
		}
	}
}

// fillShelves turns each plan's fetched movies into a row: only movies
// with a poster that aren't in the library and haven't appeared in an
// earlier row. Rows left too short are dropped.
func fillShelves(plans []shelfPlan, fetched [][]discoverResultBody) []discoverShelfBody {
	shelves := make([]discoverShelfBody, 0, len(plans))
	seen := make(map[int]bool)
	for i, p := range plans {
		results := make([]discoverResultBody, 0, shelfSize)
		for _, r := range fetched[i] {
			if r.InLibrary || r.PosterPath == "" || seen[r.TMDBID] || len(results) == shelfSize {
				continue
			}
			results = append(results, r)
		}
		if len(results) == 0 || len(results) < p.minSize {
			continue
		}
		for _, r := range results {
			seen[r.TMDBID] = true
		}
		shelves = append(shelves, discoverShelfBody{ID: p.id, Kind: p.kind, Title: p.title, SeeAll: p.seeAll, Results: results})
	}
	return shelves
}

func filmographyResults(items []tmdb.FilmographyItem) []tmdb.SearchResult {
	// Newest first; undated entries (year 0) fall to the end.
	sort.SliceStable(items, func(i, j int) bool { return items[i].Year > items[j].Year })
	out := make([]tmdb.SearchResult, 0, len(items))
	for _, it := range items {
		out = append(out, tmdb.SearchResult{ID: it.TMDBID, Title: it.Title, Year: it.Year, PosterPath: it.PosterPath})
	}
	return out
}

func registerDiscoverShelves(api huma.API, movieSvc *movie.Service, collectionSvc *collection.Service, tmdbClient *tmdb.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "discover-shelves",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/shelves",
		Summary:     "Rows of movies to add, picked from what the library has",
		Description: "Rows: recommendations from recently added movies, top-rated movies in the library's most common genres, what's missing from its collections, plus trending, new on streaming and coming soon. Every row leaves out movies already in the library, and no movie appears twice. A row TMDB fails to answer for is left out.",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverShelvesInput) (*discoverShelvesOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		// Shuffling which rows to show needs to be repeatable, not secret.
		seed := uint64(input.Seed)
		if seed == 0 {
			seed = rand.Uint64() //nolint:gosec // G404: picks rows, guards nothing
		}
		rng := rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // G404: as above
		region := tmdb.NormalizeRegion(input.Region)
		now := time.Now()

		var because, genres, colls []shelfPlan

		if movieSvc != nil {
			seeds, _ := movieSvc.RecentlyAdded(ctx, shelfSeedPool)
			for _, s := range pickN(rng, seeds, shelfSeedRows) {
				because = append(because, shelfPlan{
					id:      fmt.Sprintf("because-%d", s.TMDBID),
					kind:    "because",
					title:   "Because you have " + s.Title,
					minSize: shelfMinSize,
					fetch: func(ctx context.Context) ([]tmdb.SearchResult, error) {
						pr, err := tmdbClient.Recommendations(ctx, s.TMDBID)
						if err != nil {
							return nil, err
						}
						return pr.Results, nil
					},
				})
			}

			counts, _ := movieSvc.GenreCounts(ctx)
			if len(counts) > 0 {
				known, _ := tmdbClient.GenreList(ctx)
				for _, g := range pickN(rng, topGenres(counts, known, shelfGenrePool), shelfGenreRows) {
					page := 1 + rng.IntN(shelfGenrePages)
					seeAll := url.Values{"category": {tmdb.CategoryTopRated}, "genre": {strconv.Itoa(g.ID)}, "hide_owned": {"1"}}
					genres = append(genres, shelfPlan{
						id:      fmt.Sprintf("genre-%d", g.ID),
						kind:    "genre",
						title:   "Top-rated " + g.Name + " you don't have",
						seeAll:  "/discover?" + seeAll.Encode(),
						minSize: shelfMinSize,
						fetch: func(ctx context.Context) ([]tmdb.SearchResult, error) {
							pr, err := tmdbClient.Browse(ctx, tmdb.BrowseFilter{Category: tmdb.CategoryTopRated, GenreIDs: []int{g.ID}, Page: page, Today: now})
							if err != nil {
								return nil, err
							}
							return pr.Results, nil
						},
					})
				}
			}
		}

		if collectionSvc != nil {
			all, _ := collectionSvc.List(ctx)
			incomplete := make([]collection.Collection, 0, len(all))
			for _, c := range all {
				if c.Missing > 0 {
					incomplete = append(incomplete, c)
				}
			}
			for _, c := range pickN(rng, incomplete, shelfCollRows) {
				plan := shelfPlan{
					id:     "collection-" + c.ID,
					kind:   "collection",
					seeAll: "/collections/" + c.ID,
				}
				if c.PersonType == "franchise" {
					plan.title = "Missing from " + c.Name
					plan.fetch = func(ctx context.Context) ([]tmdb.SearchResult, error) {
						pr, err := tmdbClient.FranchiseMovies(ctx, c.PersonID)
						if err != nil {
							return nil, err
						}
						return pr.Results, nil
					}
				} else {
					plan.title = "More from " + c.Name
					plan.fetch = func(ctx context.Context) ([]tmdb.SearchResult, error) {
						items, err := tmdbClient.GetPersonFilmography(ctx, c.PersonID, c.PersonType)
						if err != nil {
							return nil, err
						}
						return filmographyResults(items), nil
					}
				}
				colls = append(colls, plan)
			}
		}

		list := func(category, title string) shelfPlan {
			return shelfPlan{
				id:      "list-" + category,
				kind:    "list",
				title:   title,
				seeAll:  "/discover?category=" + category,
				minSize: shelfMinSize,
				fetch: func(ctx context.Context) ([]tmdb.SearchResult, error) {
					pr, err := tmdbClient.Browse(ctx, tmdb.BrowseFilter{Category: category, Region: region, Today: now})
					if err != nil {
						return nil, err
					}
					return pr.Results, nil
				},
			}
		}
		lists := []shelfPlan{
			list(tmdb.CategoryTrending, "Trending this week"),
			list(tmdb.CategoryNewOnline, "New on streaming"),
			list(tmdb.CategoryComingSoon, "Coming soon"),
		}

		plans := interleave(because, lists, genres, colls)
		fetched := make([][]discoverResultBody, len(plans))
		var wg sync.WaitGroup
		for i, p := range plans {
			wg.Go(func() {
				results, err := p.fetch(ctx)
				if err != nil {
					return // the row is left out
				}
				fetched[i] = enrichDiscover(ctx, movieSvc, results)
			})
		}
		wg.Wait()

		return &discoverShelvesOutput{Body: &discoverShelvesBody{Shelves: fillShelves(plans, fetched)}}, nil
	})
}
