package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

// ── Types ────────────────────────────────────────────────────────────────────

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

// ── Inputs ───────────────────────────────────────────────────────────────────

type discoverPageInput struct {
	Page int `query:"page" default:"1" doc:"Page number (1-based)"`
}

type discoverGenreInput struct {
	ID   string `path:"id"   doc:"TMDB genre ID"`
	Page int    `query:"page" default:"1" doc:"Page number (1-based)"`
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

// ── Registration ─────────────────────────────────────────────────────────────

// RegisterDiscoverRoutes registers the /api/v1/discover/* endpoints.
func RegisterDiscoverRoutes(api huma.API, movieSvc *movie.Service, tmdbClient *tmdb.Client) {
	enrich := func(ctx context.Context, results []tmdb.SearchResult) []discoverResultBody {
		out := make([]discoverResultBody, 0, len(results))
		for _, r := range results {
			item := discoverResultBody{
				TMDBID:     r.ID,
				Title:      r.Title,
				Year:       r.Year,
				Overview:   r.Overview,
				PosterPath: r.PosterPath,
				Rating:     r.Popularity, // PaginatedResults stores vote_average here
			}
			if movieSvc != nil {
				if m, err := movieSvc.GetByTMDBID(ctx, r.ID); err == nil {
					item.InLibrary = true
					item.LibraryMovieID = m.ID
				}
			}
			out = append(out, item)
		}
		return out
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
