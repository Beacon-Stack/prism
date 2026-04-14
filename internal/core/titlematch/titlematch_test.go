package titlematch

import "testing"

func TestNormalize(t *testing.T) {
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
		got := Normalize(tc.input)
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		name    string
		release string
		title   string
		year    int
		want    bool
	}{
		{
			name:    "exact title and year",
			release: "The.Dark.Knight.2008.BluRay.1080p.x264",
			title:   "The Dark Knight",
			year:    2008,
			want:    true,
		},
		{
			name:    "4k release",
			release: "Interstellar.2014.2160p.UHD.BluRay.x265",
			title:   "Interstellar",
			year:    2014,
			want:    true,
		},
		{
			name:    "title with colon",
			release: "Avatar.The.Way.of.Water.2022.1080p.WEBRip.x264",
			title:   "Avatar: The Way of Water",
			year:    2022,
			want:    true,
		},
		{
			name:    "wrong year rejected",
			release: "The.Dark.Knight.2008.BluRay.1080p",
			title:   "The Dark Knight",
			year:    2009,
			want:    false,
		},
		{
			name:    "different movie rejected",
			release: "Inception.2010.BluRay.1080p",
			title:   "Interstellar",
			year:    2010,
			want:    false,
		},
		{
			name:    "empty movie title rejected",
			release: "Inception.2010.BluRay.1080p",
			title:   "",
			year:    2010,
			want:    false,
		},
		{
			name:    "year present but title absent",
			release: "2008.Some.Other.Movie.BluRay",
			title:   "The Dark Knight",
			year:    2008,
			want:    false,
		},
		{
			name:    "short title It should not match Godzilla",
			release: "Godzilla.Minus.One.Limited.Edition.2024.BluRay.1080p",
			title:   "It",
			year:    2024,
			want:    false,
		},
		{
			name:    "short title It matches actual It release",
			release: "It.2017.1080p.BluRay.x264",
			title:   "It",
			year:    2017,
			want:    true,
		},
		{
			// ── Regression: the original user-reported bug ──────────────
			// Searching for "Big" (1988) must never grab a "The Firm" (1993)
			// release. Before titlematch was wired into auto-search, an
			// indexer returning "The.Firm.1993.1080p.BluRay.x264-GROUP" for
			// a query of "Big" passed the quality profile gate and was
			// grabbed. This test pins the fix.
			name:    "Big 1988 does not match The Firm 1993 regression",
			release: "The.Firm.1993.1080p.BluRay.x264-GROUP",
			title:   "Big",
			year:    1988,
			want:    false,
		},
		{
			name:    "Big 1988 matches actual Big release",
			release: "Big.1988.1080p.BluRay.x264-GROUP",
			title:   "Big",
			year:    1988,
			want:    true,
		},
		{
			// Year=0 means the movie has no known year (TMDB lookup failed
			// or the movie is pre-release). Accept title-only match.
			name:    "no year accepts title-only",
			release: "Interstellar.2014.BluRay.1080p",
			title:   "Interstellar",
			year:    0,
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Matches(tc.release, tc.title, tc.year)
			if got != tc.want {
				t.Errorf("Matches(%q, %q, %d) = %v; want %v",
					tc.release, tc.title, tc.year, got, tc.want)
			}
		})
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
