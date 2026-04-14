package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/beacon-stack/prism/internal/db/generated"
)

const testTimestamp = "2025-01-01T00:00:00Z"

// SeedQualityProfile inserts a quality profile with sensible defaults
// and returns it. The inserted profile accepts 1080p BluRay releases.
func SeedQualityProfile(t *testing.T, q *dbgen.Queries) dbgen.QualityProfile {
	t.Helper()
	ctx := context.Background()
	cutoff := `{"resolution":"1080p","source":"bluray","codec":"x264","hdr":"none","name":"Bluray-1080p"}`
	qualities := `[{"resolution":"1080p","source":"bluray","codec":"x264","hdr":"none","name":"Bluray-1080p"}]`

	row, err := q.CreateQualityProfile(ctx, dbgen.CreateQualityProfileParams{
		ID:             uuid.New().String(),
		Name:           "Test HD " + uuid.New().String()[:8],
		CutoffJson:     cutoff,
		QualitiesJson:  qualities,
		UpgradeAllowed: false,
		CreatedAt:      testTimestamp,
		UpdatedAt:      testTimestamp,
	})
	if err != nil {
		t.Fatalf("testutil.SeedQualityProfile: %v", err)
	}
	return row
}

// SeedLibrary inserts a library backed by a freshly created quality profile.
func SeedLibrary(t *testing.T, q *dbgen.Queries) dbgen.Library {
	t.Helper()
	profile := SeedQualityProfile(t, q)
	return SeedLibraryWithProfile(t, q, profile.ID)
}

// SeedLibraryWithProfile inserts a library using the given quality profile ID.
func SeedLibraryWithProfile(t *testing.T, q *dbgen.Queries, profileID string) dbgen.Library {
	t.Helper()
	ctx := context.Background()
	row, err := q.CreateLibrary(ctx, dbgen.CreateLibraryParams{
		ID:                      uuid.New().String(),
		Name:                    "Test Movies",
		RootPath:                "/movies",
		DefaultQualityProfileID: profileID,
		MinFreeSpaceGb:          5,
		TagsJson:                "[]",
		CreatedAt:               testTimestamp,
		UpdatedAt:               testTimestamp,
	})
	if err != nil {
		t.Fatalf("testutil.SeedLibrary: %v", err)
	}
	return row
}

// MovieOption is a functional option for SeedMovie.
type MovieOption func(*dbgen.CreateMovieParams)

// WithMonitored sets the monitored flag on the seeded movie.
func WithMonitored(monitored bool) MovieOption {
	return func(p *dbgen.CreateMovieParams) {
		p.Monitored = monitored
	}
}

// WithTMDBID sets the TMDB ID on the seeded movie.
func WithTMDBID(id int) MovieOption {
	return func(p *dbgen.CreateMovieParams) {
		p.TmdbID = int32(id)
	}
}

// WithMovieStatus sets the status field on the seeded movie.
func WithMovieStatus(status string) MovieOption {
	return func(p *dbgen.CreateMovieParams) {
		p.Status = status
	}
}

// SeedMovie inserts a movie into a freshly created library with sensible defaults.
func SeedMovie(t *testing.T, q *dbgen.Queries, opts ...MovieOption) dbgen.Movie {
	t.Helper()
	lib := SeedLibrary(t, q)
	return SeedMovieInLibrary(t, q, lib.ID, lib.DefaultQualityProfileID, opts...)
}

// SeedMovieInLibrary inserts a movie into the specified library.
func SeedMovieInLibrary(t *testing.T, q *dbgen.Queries, libraryID, profileID string, opts ...MovieOption) dbgen.Movie {
	t.Helper()
	ctx := context.Background()

	now := time.Now().UTC().Format(time.RFC3339)
	params := dbgen.CreateMovieParams{
		ID:               uuid.New().String(),
		TmdbID:           27205,
		Title:            "Inception",
		OriginalTitle:    "Inception",
		Year:             2010,
		Overview:         "A thief who steals corporate secrets through dreams.",
		GenresJson:       `["Action","Sci-Fi"]`,
		Status:           "released",
		Monitored:        true,
		LibraryID:        libraryID,
		QualityProfileID: profileID,
		AddedAt:          now,
		UpdatedAt:        now,
	}

	for _, opt := range opts {
		opt(&params)
	}

	row, err := q.CreateMovie(ctx, params)
	if err != nil {
		t.Fatalf("testutil.SeedMovieInLibrary: %v", err)
	}
	return row
}

// SeedGrabHistory inserts one grab_history row for movieID and returns it.
func SeedGrabHistory(t *testing.T, q *dbgen.Queries, movieID, title string) dbgen.GrabHistory {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := q.CreateGrabHistory(ctx, dbgen.CreateGrabHistoryParams{
		ID:                uuid.New().String(),
		MovieID:           movieID,
		ReleaseGuid:       uuid.New().String(),
		ReleaseTitle:      title,
		ReleaseSource:     "bluray",
		ReleaseResolution: "1080p",
		Protocol:          "torrent",
		Size:              8_000_000_000,
		GrabbedAt:         now,
		DownloadStatus:    "completed",
	})
	if err != nil {
		t.Fatalf("testutil.SeedGrabHistory: %v", err)
	}
	return row
}
