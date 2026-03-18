package parser

import "testing"

func TestExtractTitle_YearEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantYear  int
	}{
		// Year in title — last year before stop token wins.
		{"2001 Space Odyssey", "2001.A.Space.Odyssey.1968.1080p.BluRay", "2001 A Space Odyssey", 1968},
		{"Blade Runner 2049", "Blade.Runner.2049.2017.1080p.BluRay", "Blade Runner 2049", 2017},
		{"1917", "1917.2019.1080p.WEBRip", "1917", 2019},

		// Multiple years.
		{"three years", "Movie.1950.And.2020.2024.1080p.BluRay", "Movie 1950 And 2020", 2024},

		// Year after stop token should be ignored.
		{"year after quality", "Movie.1080p.2025.BluRay", "Movie", 0},

		// No year.
		{"no year at all", "Alien.1080p.BluRay.x264", "Alien", 0},
		{"no year minimal", "SomeMovie", "SomeMovie", 0},

		// Year range edges.
		{"year 1900", "Movie.1900.1080p.BluRay", "Movie", 1900},
		{"year 2099", "Movie.2099.1080p.BluRay", "Movie", 2099},
		{"1899 not a year", "Movie.1899.1080p.BluRay", "Movie 1899", 0},

		// Title starting with number.
		{"10 Cloverfield Lane", "10.Cloverfield.Lane.2016.1080p.BluRay", "10 Cloverfield Lane", 2016},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
			if got.Year != tc.wantYear {
				t.Errorf("Year: got %d, want %d", got.Year, tc.wantYear)
			}
		})
	}
}

func TestExtractTitle_AllCapsNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"all caps underscored", "THE_DARK_KNIGHT_2008_1080P_BLURAY", "The Dark Knight"},
		{"all caps dotted", "THE.DARK.KNIGHT.2008.1080P.BLURAY", "The Dark Knight"},
		// Mixed case — below 60% threshold, not forced lowercase.
		{"mixed case", "The.Dark.Knight.2008.1080p.BluRay", "The Dark Knight"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestExtractTitle_DiscNoise(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"Title31 stripped", "THE_HUNGERGAMES_MOCKINGJAY_PT1_Title31", "The Hungergames Mockingjay Part 1"},
		{"Title01 stripped", "Movie.Title01.2020.1080p.BluRay", "Movie"},
		{"Chapter05 stripped", "Movie.Chapter05.2020.1080p.BluRay", "Movie"},
		{"Disc01 stripped", "Movie.Disc01.2020.1080p.BluRay", "Movie"},
		{"Track03 stripped", "Movie.Track03.2020.1080p.BluRay", "Movie"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestExtractTitle_PtNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"Pt2", "Avengers.Infinity.War.Pt2.2018.WEBRip", "Avengers Infinity War Part 2"},
		{"Pt.2", "Harry.Potter.Pt.2.2011.1080p.BluRay", "Harry Potter Part 2"},
		{"Pt3", "Movie.Pt3.2020.1080p", "Movie Part 3"},
		{"PT1 uppercase", "THE_HUNGERGAMES_MOCKINGJAY_PT1_Title31", "The Hungergames Mockingjay Part 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestExtractTitle_PathStripping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"unix path", "/media/movies/The.Matrix.1999.1080p.BluRay.x264/The.Matrix.1999.1080p.BluRay.x264.mkv", "The Matrix"},
		{"windows path", `C:\Movies\The.Dark.Knight.2008.1080p.BluRay.x264.mkv`, "The Dark Knight"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestExtractTitle_FileExtensions(t *testing.T) {
	t.Parallel()
	exts := []string{"mkv", "mp4", "avi", "mov", "wmv", "m4v", "ts", "m2ts", "flv", "webm"}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()
			input := "The.Matrix.1999.1080p.BluRay.x264." + ext
			got := Parse(input)
			if got.Title != "The Matrix" {
				t.Errorf("Title: got %q, want %q (ext: %s)", got.Title, "The Matrix", ext)
			}
			if got.Year != 1999 {
				t.Errorf("Year: got %d, want 1999 (ext: %s)", got.Year, ext)
			}
		})
	}
}

func TestExtractTitle_Separators(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"dots", "The.Dark.Knight.2008.1080p.BluRay", "The Dark Knight"},
		{"underscores", "The_Dark_Knight_2008_1080p_BluRay", "The Dark Knight"},
		{"spaces", "The Dark Knight 2008 1080p BluRay", "The Dark Knight"},
		{"hyphens in title", "WALL-E.2008.1080p.BluRay", "WALL-E"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}
