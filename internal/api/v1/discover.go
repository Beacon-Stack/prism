package v1

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/collection"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

// Types

type discoverResultBody struct {
	TMDBID         int     `json:"tmdb_id"         doc:"TMDB movie ID"`
	Title          string  `json:"title"           doc:"Movie title"`
	Year           int     `json:"year"            doc:"Release year"`
	Overview       string  `json:"overview"        doc:"Plot summary"`
	PosterPath     string  `json:"poster_path"     doc:"TMDB poster path"`
	Rating         float64 `json:"rating"          doc:"TMDB vote average"`
	InLibrary      bool    `json:"in_library"      doc:"Already in Prism library"`
	Excluded       bool    `json:"excluded"        doc:"On the import exclusion list"`
	LibraryMovieID string  `json:"library_movie_id,omitempty" doc:"Prism movie ID if in library"`
}

type discoverListBody struct {
	Results    []discoverResultBody `json:"results"`
	Page       int                  `json:"page"`
	TotalPages int                  `json:"total_pages"`
}

type discoverListOutput struct {
	Body *discoverListBody
}

type genreBody struct {
	ID   int    `json:"id"   doc:"TMDB genre ID"`
	Name string `json:"name" doc:"Genre name"`
}

type genreListOutput struct {
	Body []genreBody
}

// Inputs

type discoverPageInput struct {
	Page int `query:"page" default:"1" doc:"Page number (1-based)"`
}

type discoverGenreInput struct {
	ID   string `path:"id"   doc:"TMDB genre ID"`
	Page int    `query:"page" default:"1" doc:"Page number (1-based)"`
}

type discoverBrowseInput struct {
	Category   string  `query:"category" default:"trending" enum:"trending,popular,top-rated,new,coming-soon" doc:"Which list"`
	Page       int     `query:"page" default:"1" minimum:"1" maximum:"500" doc:"Page number (1-based)"`
	Genre      []int   `query:"genre" doc:"TMDB genre IDs, comma-separated; movies must be in every one"`
	Provider   int     `query:"provider" doc:"TMDB streaming service ID; 0 for any"`
	MinRating  float64 `query:"min_rating" minimum:"0" maximum:"10" doc:"Minimum TMDB rating (0-10)"`
	Region     string  `query:"region" default:"US" doc:"Country for release dates and streaming services (ISO 3166-1)"`
	Sort       string  `query:"sort" enum:"rating,newest,revenue" doc:"Order that replaces the list's own: highest rated, newest or biggest box office"`
	Decade     int     `query:"decade" doc:"First year of a decade (1990 = 1990-1999); 0 for any"`
	Runtime    string  `query:"runtime" enum:"short,medium,long" doc:"Under 90 minutes, 90-120 minutes, or over two hours"`
	Language   string  `query:"language" maxLength:"2" doc:"Original language (ISO 639-1)"`
	Keyword    int     `query:"keyword" doc:"TMDB keyword ID; 0 for any"`
	Company    int     `query:"company" doc:"TMDB production company ID; 0 for any"`
	Collection int     `query:"collection" doc:"TMDB collection ID: lists that collection's movies and ignores everything else"`
}

type discoverRegionInput struct {
	Region string `query:"region" default:"US" doc:"Country (ISO 3166-1)"`
}

type discoverSearchInput struct {
	Query string `query:"q" maxLength:"200" doc:"Title to search for"`
	Page  int    `query:"page" default:"1" minimum:"1" maximum:"500" doc:"Page number (1-based)"`
}

type providerBody struct {
	ID       int    `json:"id"        doc:"TMDB watch provider ID"`
	Name     string `json:"name"      doc:"Service name"`
	LogoPath string `json:"logo_path" doc:"TMDB logo path"`
}

type providerListOutput struct {
	Body []providerBody
}

type discoverMovieInput struct {
	TmdbID int `path:"tmdbId" doc:"TMDB movie ID"`
}

type discoverMovieDetailBody struct {
	TMDBID          int                          `json:"tmdb_id"          doc:"TMDB movie ID"`
	IMDBId          string                       `json:"imdb_id,omitempty" doc:"IMDB ID"`
	Title           string                       `json:"title"            doc:"Movie title"`
	OriginalTitle   string                       `json:"original_title"   doc:"Original title"`
	Year            int                          `json:"year"             doc:"Release year"`
	Overview        string                       `json:"overview"         doc:"Plot summary"`
	ReleaseDate     string                       `json:"release_date"     doc:"Release date (YYYY-MM-DD)"`
	RuntimeMinutes  int                          `json:"runtime_minutes"  doc:"Runtime in minutes"`
	Genres          []string                     `json:"genres"           doc:"Genre names"`
	PosterPath      string                       `json:"poster_path"      doc:"TMDB poster path"`
	BackdropPath    string                       `json:"backdrop_path"    doc:"TMDB backdrop path"`
	Status          string                       `json:"status"           doc:"Release status"`
	Rating          float64                      `json:"rating"           doc:"TMDB vote average"`
	InLibrary       bool                         `json:"in_library"       doc:"Already in Prism library"`
	Excluded        bool                         `json:"excluded"         doc:"On the import exclusion list"`
	LibraryMovieID  string                       `json:"library_movie_id,omitempty" doc:"Prism movie ID if in library"`
	Cast            []discoverCastBody           `json:"cast,omitempty"            doc:"Top billed cast"`
	Crew            []discoverCrewBody           `json:"crew,omitempty"            doc:"Key crew members"`
	Recommendations []discoverRecommendationBody `json:"recommendations,omitempty" doc:"Recommended movies"`
	GenreRefs       []genreBody                  `json:"genre_refs"       doc:"Genres with their TMDB IDs"`
	Keywords        []discoverRefBody            `json:"keywords"         doc:"TMDB keywords"`
	Companies       []discoverRefBody            `json:"companies"        doc:"Production companies"`
	Collection      *discoverRefBody             `json:"collection,omitempty" doc:"The collection this movie is part of"`
}

// discoverRefBody names something a list of movies can be built from.
type discoverRefBody struct {
	ID        int    `json:"id"                   doc:"TMDB ID"`
	Name      string `json:"name"                 doc:"Name"`
	ImagePath string `json:"image_path,omitempty" doc:"TMDB image path, if it has one"`
	Detail    string `json:"detail,omitempty"     doc:"What a person is known for"`
}

type discoverSuggestInput struct {
	Query string `query:"q" maxLength:"200" doc:"Text to match"`
}

type discoverSuggestBody struct {
	People      []discoverRefBody `json:"people"      doc:"Matching people"`
	Collections []discoverRefBody `json:"collections" doc:"Matching movie collections"`
	Keywords    []discoverRefBody `json:"keywords"    doc:"Matching keywords"`
	Companies   []discoverRefBody `json:"companies"   doc:"Matching production companies"`
}

type discoverSuggestOutput struct {
	Body *discoverSuggestBody
}

type discoverCastBody struct {
	ID          int    `json:"id"           doc:"TMDB person ID"`
	Name        string `json:"name"         doc:"Actor name"`
	Character   string `json:"character"    doc:"Character name"`
	ProfilePath string `json:"profile_path" doc:"TMDB profile image path"`
}

type discoverCrewBody struct {
	ID          int    `json:"id"           doc:"TMDB person ID"`
	Name        string `json:"name"         doc:"Crew member name"`
	Job         string `json:"job"          doc:"Job title"`
	ProfilePath string `json:"profile_path" doc:"TMDB profile image path"`
}

type discoverRecommendationBody struct {
	TMDBID     int    `json:"tmdb_id"      doc:"TMDB movie ID"`
	Title      string `json:"title"        doc:"Movie title"`
	Year       int    `json:"year"         doc:"Release year"`
	PosterPath string `json:"poster_path"  doc:"TMDB poster path"`
}

type discoverMovieDetailOutput struct {
	Body *discoverMovieDetailBody
}

// enrichDiscover turns TMDB results into response items, marking the ones
// already in the library and dropping those with no usable title.
func enrichDiscover(ctx context.Context, movieSvc *movie.Service, results []tmdb.SearchResult) []discoverResultBody {
	// Resolve library membership for the whole page in one query rather
	// than one GetByTMDBID per result.
	var inLibrary map[int]string
	if movieSvc != nil && len(results) > 0 {
		ids := make([]int, len(results))
		for i, r := range results {
			ids[i] = r.ID
		}
		inLibrary, _ = movieSvc.LibraryIDsByTMDBIDs(ctx, ids)
	}

	out := make([]discoverResultBody, 0, len(results))
	for _, r := range results {
		title, ok := displayTitle(r.Title, r.OriginalTitle)
		if !ok {
			continue
		}
		item := discoverResultBody{
			TMDBID:     r.ID,
			Title:      title,
			Year:       r.Year,
			Overview:   r.Overview,
			PosterPath: r.PosterPath,
			Rating:     r.Popularity, // PaginatedResults stores vote_average here
		}
		if movieID, ok := inLibrary[r.ID]; ok {
			item.InLibrary = true
			item.LibraryMovieID = movieID
		}
		out = append(out, item)
	}
	return out
}

// maxSuggestions is how many of each kind a search suggests.
const maxSuggestions = 6

// firstN returns at most the first n items.
func firstN[T any](items []T, n int) []T {
	if len(items) > n {
		return items[:n]
	}
	return items
}

// Registration

// RegisterDiscoverRoutes registers the /api/v1/discover/* endpoints.
// collectionSvc may be nil; the shelves then leave out collection rows.
func RegisterDiscoverRoutes(api huma.API, movieSvc *movie.Service, collectionSvc *collection.Service, tmdbClient *tmdb.Client) {
	enrich := func(ctx context.Context, results []tmdb.SearchResult) []discoverResultBody {
		return enrichDiscover(ctx, movieSvc, results)
	}

	registerList := func(opID, path, summary string, fetch func(context.Context, int) (*tmdb.PaginatedResults, error)) {
		huma.Register(api, huma.Operation{
			OperationID: opID,
			Method:      http.MethodGet,
			Path:        path,
			Summary:     summary,
			Tags:        []string{"Discover"},
		}, func(ctx context.Context, input *discoverPageInput) (*discoverListOutput, error) {
			if tmdbClient == nil {
				return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
			}
			page := input.Page
			if page < 1 {
				page = 1
			}
			pr, err := fetch(ctx, page)
			if err != nil {
				return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
			}
			return &discoverListOutput{Body: &discoverListBody{
				Results:    enrich(ctx, pr.Results),
				Page:       pr.Page,
				TotalPages: pr.TotalPages,
			}}, nil
		})
	}

	registerList("discover-trending", "/api/v1/discover/trending", "Trending movies this week",
		func(ctx context.Context, page int) (*tmdb.PaginatedResults, error) {
			return tmdbClient.FetchTrending(ctx, page)
		})

	registerList("discover-popular", "/api/v1/discover/popular", "Popular movies",
		func(ctx context.Context, page int) (*tmdb.PaginatedResults, error) {
			return tmdbClient.FetchPopular(ctx, page)
		})

	registerList("discover-top-rated", "/api/v1/discover/top-rated", "Top rated movies",
		func(ctx context.Context, page int) (*tmdb.PaginatedResults, error) {
			return tmdbClient.FetchTopRated(ctx, page)
		})

	registerList("discover-upcoming", "/api/v1/discover/upcoming", "Upcoming movies",
		func(ctx context.Context, page int) (*tmdb.PaginatedResults, error) {
			return tmdbClient.FetchUpcoming(ctx, page)
		})

	// Browse with filters: one endpoint for every category, so genre,
	// streaming service and rating can be combined with any of them.
	huma.Register(api, huma.Operation{
		OperationID: "discover-browse",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/browse",
		Summary:     "Browse movies by category with optional filters",
		Description: "Categories: trending, popular, top-rated, new (released to streaming, rent or buy in the last 60 days), coming-soon. Filters are ignored for trending. decade is ignored for new and coming-soon, which set their own dates; sort by rating or revenue is ignored for coming-soon.",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverBrowseInput) (*discoverListOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		var pr *tmdb.PaginatedResults
		var err error
		if input.Collection > 0 {
			pr, err = tmdbClient.FranchiseMovies(ctx, input.Collection)
		} else {
			pr, err = tmdbClient.Browse(ctx, tmdb.BrowseFilter{
				Category:   input.Category,
				Page:       input.Page,
				GenreIDs:   input.Genre,
				ProviderID: input.Provider,
				MinRating:  input.MinRating,
				Region:     input.Region,
				Sort:       input.Sort,
				Decade:     input.Decade,
				Runtime:    input.Runtime,
				Language:   input.Language,
				KeywordID:  input.Keyword,
				CompanyID:  input.Company,
				Today:      time.Now(),
			})
		}
		if err != nil {
			if strings.HasPrefix(err.Error(), "unknown category") {
				return nil, huma.NewError(http.StatusBadRequest, err.Error())
			}
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}
		return &discoverListOutput{Body: &discoverListBody{
			Results:    enrich(ctx, pr.Results),
			Page:       pr.Page,
			TotalPages: pr.TotalPages,
		}}, nil
	})

	// Streaming services for the service filter.
	huma.Register(api, huma.Operation{
		OperationID: "discover-providers",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/providers",
		Summary:     "List streaming services in a country",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverRegionInput) (*providerListOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		ps, err := tmdbClient.WatchProviders(ctx, input.Region)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}
		out := make([]providerBody, 0, len(ps))
		for _, p := range ps {
			out = append(out, providerBody{ID: p.ID, Name: p.Name, LogoPath: p.LogoPath})
		}
		return &providerListOutput{Body: out}, nil
	})

	// Title search, marked with what's already in the library.
	huma.Register(api, huma.Operation{
		OperationID: "discover-search",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/search",
		Summary:     "Search movies by title",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverSearchInput) (*discoverListOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		q := strings.TrimSpace(input.Query)
		if q == "" {
			return &discoverListOutput{Body: &discoverListBody{Results: []discoverResultBody{}, Page: 1, TotalPages: 0}}, nil
		}
		pr, err := tmdbClient.SearchPaginated(ctx, q, input.Page)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}
		return &discoverListOutput{Body: &discoverListBody{
			Results:    enrich(ctx, pr.Results),
			Page:       pr.Page,
			TotalPages: pr.TotalPages,
		}}, nil
	})

	// People, collections, keywords and studios matching a search, so a
	// search can lead somewhere other than a title.
	huma.Register(api, huma.Operation{
		OperationID: "discover-suggest",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/suggest",
		Summary:     "Find people, collections, keywords and studios by name",
		Description: "A kind that TMDB fails to answer for comes back empty rather than failing the request.",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverSuggestInput) (*discoverSuggestOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		body := &discoverSuggestBody{
			People:      []discoverRefBody{},
			Collections: []discoverRefBody{},
			Keywords:    []discoverRefBody{},
			Companies:   []discoverRefBody{},
		}
		q := strings.TrimSpace(input.Query)
		if q == "" {
			return &discoverSuggestOutput{Body: body}, nil
		}

		var wg sync.WaitGroup
		wg.Go(func() {
			people, err := tmdbClient.SearchPeople(ctx, q)
			if err != nil {
				return
			}
			for _, p := range firstN(people, maxSuggestions) {
				body.People = append(body.People, discoverRefBody{ID: p.ID, Name: p.Name, ImagePath: p.ProfilePath, Detail: p.KnownForDepartment})
			}
		})
		wg.Go(func() {
			franchises, err := tmdbClient.SearchFranchises(ctx, q)
			if err != nil {
				return
			}
			for _, f := range firstN(franchises, maxSuggestions) {
				body.Collections = append(body.Collections, discoverRefBody{ID: f.ID, Name: f.Name, ImagePath: f.PosterPath})
			}
		})
		wg.Go(func() {
			keywords, err := tmdbClient.SearchKeywords(ctx, q)
			if err != nil {
				return
			}
			for _, k := range firstN(keywords, maxSuggestions) {
				body.Keywords = append(body.Keywords, discoverRefBody{ID: k.ID, Name: k.Name})
			}
		})
		wg.Go(func() {
			companies, err := tmdbClient.SearchCompanies(ctx, q)
			if err != nil {
				return
			}
			for _, c := range firstN(companies, maxSuggestions) {
				body.Companies = append(body.Companies, discoverRefBody{ID: c.ID, Name: c.Name, ImagePath: c.LogoPath})
			}
		})
		wg.Wait()
		return &discoverSuggestOutput{Body: body}, nil
	})

	registerDiscoverShelves(api, movieSvc, collectionSvc, tmdbClient)

	// Genre browse
	huma.Register(api, huma.Operation{
		OperationID: "discover-genre",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/genre/{id}",
		Summary:     "Discover movies by genre",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverGenreInput) (*discoverListOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		genreID, err := strconv.Atoi(input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid genre ID")
		}
		page := input.Page
		if page < 1 {
			page = 1
		}
		pr, err := tmdbClient.DiscoverByGenre(ctx, genreID, page)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}
		return &discoverListOutput{Body: &discoverListBody{
			Results:    enrich(ctx, pr.Results),
			Page:       pr.Page,
			TotalPages: pr.TotalPages,
		}}, nil
	})

	// Genre list
	huma.Register(api, huma.Operation{
		OperationID: "discover-genres",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/genres",
		Summary:     "List TMDB movie genres",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, _ *struct{}) (*genreListOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		genres, err := tmdbClient.GenreList(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}
		bodies := make([]genreBody, 0, len(genres))
		for _, g := range genres {
			bodies = append(bodies, genreBody{ID: g.ID, Name: g.Name})
		}
		return &genreListOutput{Body: bodies}, nil
	})

	// Single movie detail by TMDB ID
	huma.Register(api, huma.Operation{
		OperationID: "discover-movie",
		Method:      http.MethodGet,
		Path:        "/api/v1/discover/{tmdbId}",
		Summary:     "Get movie details from TMDB by ID",
		Tags:        []string{"Discover"},
	}, func(ctx context.Context, input *discoverMovieInput) (*discoverMovieDetailOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}
		detail, err := tmdbClient.GetMovieExtended(ctx, input.TmdbID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}

		body := &discoverMovieDetailBody{
			TMDBID:         detail.ID,
			IMDBId:         detail.IMDBId,
			Title:          detail.Title,
			OriginalTitle:  detail.OriginalTitle,
			Year:           detail.Year,
			Overview:       detail.Overview,
			ReleaseDate:    detail.ReleaseDate,
			RuntimeMinutes: detail.RuntimeMinutes,
			Genres:         detail.Genres,
			PosterPath:     detail.PosterPath,
			BackdropPath:   detail.BackdropPath,
			Status:         detail.Status,
			Rating:         detail.Rating,
			GenreRefs:      make([]genreBody, 0, len(detail.GenreRefs)),
			Keywords:       make([]discoverRefBody, 0, len(detail.Keywords)),
			Companies:      make([]discoverRefBody, 0, len(detail.Companies)),
		}
		for _, g := range detail.GenreRefs {
			body.GenreRefs = append(body.GenreRefs, genreBody{ID: g.ID, Name: g.Name})
		}
		for _, k := range detail.Keywords {
			body.Keywords = append(body.Keywords, discoverRefBody{ID: k.ID, Name: k.Name})
		}
		for _, c := range detail.Companies {
			body.Companies = append(body.Companies, discoverRefBody{ID: c.ID, Name: c.Name, ImagePath: c.LogoPath})
		}
		if f := detail.Franchise; f != nil {
			body.Collection = &discoverRefBody{ID: f.ID, Name: f.Name, ImagePath: f.PosterPath}
		}

		// Check library status.
		if movieSvc != nil {
			if m, err := movieSvc.GetByTMDBID(ctx, detail.ID); err == nil {
				body.InLibrary = true
				body.LibraryMovieID = m.ID
			}
		}

		// Map cast.
		cast := make([]discoverCastBody, 0, len(detail.Cast))
		for _, c := range detail.Cast {
			cast = append(cast, discoverCastBody{
				ID:          c.ID,
				Name:        c.Name,
				Character:   c.Character,
				ProfilePath: c.ProfilePath,
			})
		}
		body.Cast = cast

		// Map crew.
		crew := make([]discoverCrewBody, 0, len(detail.Crew))
		for _, c := range detail.Crew {
			crew = append(crew, discoverCrewBody{
				ID:          c.ID,
				Name:        c.Name,
				Job:         c.Job,
				ProfilePath: c.ProfilePath,
			})
		}
		body.Crew = crew

		// Map recommendations.
		recs := make([]discoverRecommendationBody, 0, len(detail.Recommendations))
		for _, r := range detail.Recommendations {
			recs = append(recs, discoverRecommendationBody{
				TMDBID:     r.TMDBID,
				Title:      r.Title,
				Year:       r.Year,
				PosterPath: r.PosterPath,
			})
		}
		body.Recommendations = recs

		return &discoverMovieDetailOutput{Body: body}, nil
	})
}
