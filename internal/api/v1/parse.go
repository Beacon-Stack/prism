package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-media/prism/internal/core/movie"
)

type parseInput struct {
	Filename string `query:"filename" required:"true" doc:"Filename or path to parse"`
}

type parseOutput struct {
	Body *parsedFilenameBody
}

type parsedFilenameBody struct {
	Title string `json:"title" doc:"Extracted movie title"`
	Year  int    `json:"year"  doc:"Extracted year (0 if not found)"`
}

// RegisterParseRoutes registers the filename-parsing utility endpoint.
func RegisterParseRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "parse-filename",
		Method:      http.MethodGet,
		Path:        "/api/v1/parse",
		Summary:     "Parse a release filename into title and year",
		Tags:        []string{"Utility"},
	}, func(_ context.Context, input *parseInput) (*parseOutput, error) {
		parsed := movie.ParseFilename(input.Filename)
		return &parseOutput{Body: &parsedFilenameBody{
			Title: parsed.Title,
			Year:  parsed.Year,
		}}, nil
	})
}
