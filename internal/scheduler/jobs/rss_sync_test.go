package jobs

import (
	"encoding/json"
	"testing"

	dbsqlite "github.com/beacon-media/prism/internal/db/generated/sqlite"
	"github.com/beacon-media/prism/pkg/plugin"
)

func TestNormalizeTitle(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"The.Dark.Knight", "the dark knight"},
		{"Interstellar_2014", "interstellar 2014"},
		{"Avatar: The Way of Water", "avatar the way of water"},
		{"WALL-E", "wall e"},
		{"A.I. Artificial.Intelligence", "a i artificial intelligence"},
		{"", ""},
		{"  multiple   spaces  ", "multiple spaces"},
	}
	for _, tc := range cases {
		got := normalizeTitle(tc.input)
		if got != tc.want {
			t.Errorf("normalizeTitle(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestReleaseMatchesMovie(t *testing.T) {
	cases := []struct {
		release string
		title   string
		year    int
		want    bool
	}{
		{
			release: "The.Dark.Knight.2008.BluRay.1080p.x264",
			title:   "The Dark Knight",
			year:    2008,
			want:    true,
		},
		{
			release: "Interstellar.2014.2160p.UHD.BluRay.x265",
			title:   "Interstellar",
			year:    2014,
			want:    true,
		},
		{
			release: "Avatar.The.Way.of.Water.2022.1080p.WEBRip.x264",
			title:   "Avatar: The Way of Water",
			year:    2022,
			want:    true,
		},
		{
			// Wrong year — should not match.
			release: "The.Dark.Knight.2008.BluRay.1080p",
			title:   "The Dark Knight",
			year:    2009,
			want:    false,
		},
		{
			// Different movie — title doesn't appear.
			release: "Inception.2010.BluRay.1080p",
			title:   "Interstellar",
			year:    2010,
			want:    false,
		},
		{
			// Empty movie title — must not match anything.
			release: "Inception.2010.BluRay.1080p",
			title:   "",
			year:    2010,
			want:    false,
		},
		{
			// Year present but title absent.
			release: "2008.Some.Other.Movie.BluRay",
			title:   "The Dark Knight",
			year:    2008,
			want:    false,
		},
		{
			// Short title "It" must not match "Godzilla" releases containing "it" as substring.
			release: "Godzilla.Minus.One.Limited.Edition.2024.BluRay.1080p",
			title:   "It",
			year:    2024,
			want:    false,
		},
		{
			// Short title "It" should match an actual "It" release.
			release: "It.2017.1080p.BluRay.x264",
			title:   "It",
			year:    2017,
			want:    true,
		},
	}

	for _, tc := range cases {
		got := releaseMatchesMovie(tc.release, tc.title, tc.year)
		if got != tc.want {
			t.Errorf("releaseMatchesMovie(%q, %q, %d) = %v; want %v",
				tc.release, tc.title, tc.year, got, tc.want)
		}
	}
}

func TestContainsWordAligned(t *testing.T) {
	cases := []struct {
		haystack string
		needle   string
		want     bool
	}{
		{"the dark knight 2008", "the dark knight", true},
		{"it 2017 1080p", "it", true},
		{"godzilla minus one limited edition 2024", "it", false},
		{"hermit 2024 bluray", "it", false},
		{"item 2024 bluray", "it", false},
		{"", "it", false},
		{"it", "it", true},
		{"the it crowd 2006", "it", true},
	}
	for _, tc := range cases {
		got := containsWordAligned(tc.haystack, tc.needle)
		if got != tc.want {
			t.Errorf("containsWordAligned(%q, %q) = %v; want %v",
				tc.haystack, tc.needle, got, tc.want)
		}
	}
}

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
	makeFile := func(q plugin.Quality) dbsqlite.MovieFile {
		b, _ := json.Marshal(q)
		return dbsqlite.MovieFile{QualityJson: string(b)}
	}

	t.Run("empty files", func(t *testing.T) {
		best := bestFileQuality(nil)
		if best.Resolution != "" {
			t.Errorf("expected zero quality, got resolution=%q", best.Resolution)
		}
	})

	t.Run("single file", func(t *testing.T) {
		q := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay}
		best := bestFileQuality([]dbsqlite.MovieFile{makeFile(q)})
		if best.Resolution != plugin.Resolution1080p {
			t.Errorf("resolution = %q, want 1080p", best.Resolution)
		}
	})

	t.Run("picks best quality", func(t *testing.T) {
		low := plugin.Quality{Resolution: plugin.Resolution720p, Source: plugin.SourceHDTV}
		high := plugin.Quality{Resolution: plugin.Resolution2160p, Source: plugin.SourceBluRay}
		best := bestFileQuality([]dbsqlite.MovieFile{makeFile(low), makeFile(high)})
		if best.Resolution != plugin.Resolution2160p {
			t.Errorf("resolution = %q, want 2160p", best.Resolution)
		}
	})

	t.Run("invalid json skipped", func(t *testing.T) {
		good := plugin.Quality{Resolution: plugin.Resolution1080p}
		files := []dbsqlite.MovieFile{
			{QualityJson: "not json"},
			makeFile(good),
		}
		best := bestFileQuality(files)
		if best.Resolution != plugin.Resolution1080p {
			t.Errorf("resolution = %q, want 1080p", best.Resolution)
		}
	})
}
