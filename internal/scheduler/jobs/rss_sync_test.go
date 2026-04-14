package jobs

import (
	"encoding/json"
	"testing"

	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// Title match + normalization tests live in the titlematch package; see
// internal/core/titlematch/titlematch_test.go.

func TestMovieEligibleForGrab(t *testing.T) {
	cases := []struct {
		name     string
		minAvail string
		status   string
		want     bool
	}{
		{"tba always eligible", "tba", "Rumored", true},
		{"announced always eligible", "announced", "Rumored", true},
		{"empty minAvail always eligible", "", "Rumored", true},
		{"in_cinemas allows In Production", "in_cinemas", "In Production", true},
		{"in_cinemas allows Post Production", "in_cinemas", "Post Production", true},
		{"in_cinemas allows Released", "in_cinemas", "Released", true},
		{"in_cinemas blocks Rumored", "in_cinemas", "Rumored", false},
		{"in_cinemas blocks Planned", "in_cinemas", "Planned", false},
		{"released only allows Released", "released", "Released", true},
		{"released blocks In Production", "released", "In Production", false},
		{"released blocks Post Production", "released", "Post Production", false},
		{"unknown minAvail defaults to eligible", "custom", "anything", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := movieEligibleForGrab(tc.minAvail, tc.status)
			if got != tc.want {
				t.Errorf("movieEligibleForGrab(%q, %q) = %v, want %v",
					tc.minAvail, tc.status, got, tc.want)
			}
		})
	}
}

func TestBestFileQuality(t *testing.T) {
	makeFile := func(q plugin.Quality) dbgen.MovieFile {
		b, _ := json.Marshal(q)
		return dbgen.MovieFile{QualityJson: string(b)}
	}

	t.Run("empty files", func(t *testing.T) {
		best := bestFileQuality(nil)
		if best.Resolution != "" {
			t.Errorf("expected zero quality, got resolution=%q", best.Resolution)
		}
	})

	t.Run("single file", func(t *testing.T) {
		q := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay}
		best := bestFileQuality([]dbgen.MovieFile{makeFile(q)})
		if best.Resolution != plugin.Resolution1080p {
			t.Errorf("resolution = %q, want 1080p", best.Resolution)
		}
	})

	t.Run("picks best quality", func(t *testing.T) {
		low := plugin.Quality{Resolution: plugin.Resolution720p, Source: plugin.SourceHDTV}
		high := plugin.Quality{Resolution: plugin.Resolution2160p, Source: plugin.SourceBluRay}
		best := bestFileQuality([]dbgen.MovieFile{makeFile(low), makeFile(high)})
		if best.Resolution != plugin.Resolution2160p {
			t.Errorf("resolution = %q, want 2160p", best.Resolution)
		}
	})

	t.Run("invalid json skipped", func(t *testing.T) {
		good := plugin.Quality{Resolution: plugin.Resolution1080p}
		files := []dbgen.MovieFile{
			{QualityJson: "not json"},
			makeFile(good),
		}
		best := bestFileQuality(files)
		if best.Resolution != plugin.Resolution1080p {
			t.Errorf("resolution = %q, want 1080p", best.Resolution)
		}
	})
}
