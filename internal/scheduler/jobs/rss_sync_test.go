package jobs

import "testing"

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
