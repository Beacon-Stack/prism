package mock

import (
	"context"
	"fmt"

	"github.com/beacon-media/prism/internal/metadata/tmdb"
)

// TMDBClient is a configurable mock of movie.MetadataProvider.
type TMDBClient struct {
	SearchMoviesFunc func(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error)
	GetMovieFunc     func(ctx context.Context, tmdbID int) (*tmdb.MovieDetail, error)

	Calls []string
}

func (m *TMDBClient) SearchMovies(ctx context.Context, query string, year int) ([]tmdb.SearchResult, error) {
	m.Calls = append(m.Calls, "SearchMovies")
	if m.SearchMoviesFunc != nil {
		return m.SearchMoviesFunc(ctx, query, year)
	}
	return []tmdb.SearchResult{
		{
			ID:    27205,
			Title: "Inception",
			Year:  2010,
		},
	}, nil
}

func (m *TMDBClient) GetMovie(ctx context.Context, tmdbID int) (*tmdb.MovieDetail, error) {
	m.Calls = append(m.Calls, "GetMovie")
	if m.GetMovieFunc != nil {
		return m.GetMovieFunc(ctx, tmdbID)
	}
	return &tmdb.MovieDetail{
		ID:             tmdbID,
		Title:          fmt.Sprintf("Mock Movie %d", tmdbID),
		OriginalTitle:  fmt.Sprintf("Mock Movie %d", tmdbID),
		Year:           2010,
		Overview:       "A test movie.",
		RuntimeMinutes: 148,
		Genres:         []string{"Action", "Sci-Fi"},
		Status:         "released",
	}, nil
}
