package v1

import (
	"context"
	"net/http"
	"sort"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/metadata/tmdb"
)

// ── Types ────────────────────────────────────────────────────────────────────

type personInput struct {
	ID int `path:"id" doc:"TMDB person ID"`
}

type personFilmBody struct {
	TMDBID     int    `json:"tmdb_id"      doc:"TMDB movie ID"`
	Title      string `json:"title"        doc:"Movie title"`
	Year       int    `json:"year"         doc:"Release year"`
	PosterPath string `json:"poster_path"  doc:"TMDB poster path"`
	InLibrary  bool   `json:"in_library"   doc:"Whether this movie is in the Prism library"`
	MovieID    string `json:"movie_id,omitempty" doc:"Prism movie ID if in library"`
}

type personDetailBody struct {
	ID                 int              `json:"id"                   doc:"TMDB person ID"`
	Name               string           `json:"name"                 doc:"Person name"`
	ProfilePath        string           `json:"profile_path"         doc:"TMDB profile image path"`
	KnownForDepartment string           `json:"known_for_department" doc:"Primary department (Acting, Directing, etc.)"`
	Films              []personFilmBody `json:"films"                doc:"Filmography sorted by year descending"`
}

type personDetailOutput struct {
	Body *personDetailBody
}

// ── Registration ─────────────────────────────────────────────────────────────

// RegisterPeopleRoutes registers the /api/v1/people/{id} endpoint.
func RegisterPeopleRoutes(api huma.API, movieSvc *movie.Service, tmdbClient *tmdb.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-person",
		Method:      http.MethodGet,
		Path:        "/api/v1/people/{id}",
		Summary:     "Get person details and filmography",
		Description: "Returns person info and their filmography from TMDB, cross-referenced with the Prism library.",
		Tags:        []string{"People"},
	}, func(ctx context.Context, input *personInput) (*personDetailOutput, error) {
		if tmdbClient == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "TMDB is not configured")
		}

		person, err := tmdbClient.GetPerson(ctx, input.ID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}

		// Determine filmography type based on department.
		personType := "actor"
		if person.KnownForDepartment == "Directing" {
			personType = "director"
		}

		items, err := tmdbClient.GetPersonFilmography(ctx, input.ID, personType)
		if err != nil {
			return nil, huma.NewError(http.StatusBadGateway, "TMDB request failed", err)
		}

		// Sort by year descending (most recent first).
		sort.Slice(items, func(i, j int) bool {
			return items[i].Year > items[j].Year
		})

		films := make([]personFilmBody, 0, len(items))
		for _, f := range items {
			film := personFilmBody{
				TMDBID:     f.TMDBID,
				Title:      f.Title,
				Year:       f.Year,
				PosterPath: f.PosterPath,
			}
			if movieSvc != nil {
				if m, lookupErr := movieSvc.GetByTMDBID(ctx, f.TMDBID); lookupErr == nil {
					film.InLibrary = true
					film.MovieID = m.ID
				}
			}
			films = append(films, film)
		}

		return &personDetailOutput{Body: &personDetailBody{
			ID:                 person.ID,
			Name:               person.Name,
			ProfilePath:        person.ProfilePath,
			KnownForDepartment: person.KnownForDepartment,
			Films:              films,
		}}, nil
	})
}
