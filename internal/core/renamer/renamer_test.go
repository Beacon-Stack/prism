package renamer_test

import (
	"testing"

	"github.com/beacon-stack/prism/internal/core/renamer"
	"github.com/beacon-stack/prism/pkg/plugin"
)

var testMovie = renamer.Movie{
	Title:         "Inception",
	OriginalTitle: "Inception",
	Year:          2010,
}

var testQuality = plugin.Quality{
	Resolution: plugin.Resolution1080p,
	Source:     plugin.SourceBluRay,
	Codec:      plugin.CodecX264,
	Name:       "Bluray-1080p",
}

func TestApply_DefaultFormat(t *testing.T) {
	got := renamer.Apply(renamer.DefaultFileFormat, testMovie, testQuality)
	want := "Inception (2010) Bluray-1080p"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_CustomFormat(t *testing.T) {
	got := renamer.Apply("{Movie Title} [{Release Year}] - {Quality Full}", testMovie, testQuality)
	want := "Inception [2010] - Bluray-1080p"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_VideoCodec(t *testing.T) {
	got := renamer.Apply("{Movie Title} ({Release Year}) {Quality Full} {MediaInfo VideoCodec}", testMovie, testQuality)
	want := "Inception (2010) Bluray-1080p x264"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_SlashInTitle(t *testing.T) {
	m := renamer.Movie{Title: "AC/DC: Let There Be Rock", Year: 1980}
	got := renamer.Apply(renamer.DefaultFileFormat, m, testQuality)
	// Slash must be stripped; colon is fine on Linux
	if len(got) == 0 {
		t.Fatal("got empty string")
	}
	for _, ch := range got {
		if ch == '/' {
			t.Errorf("output contains forward slash: %q", got)
		}
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Inception", "Inception"},
		{"Batman: Begins", "Batman - Begins"},
		{"AC/DC: Let There Be Rock", "ACDC - Let There Be Rock"},
		{"Movie  With  Spaces", "Movie With Spaces"},
	}
	for _, tc := range tests {
		got := renamer.CleanTitle(tc.input)
		if got != tc.want {
			t.Errorf("CleanTitle(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFolderName(t *testing.T) {
	got := renamer.FolderName(renamer.DefaultFolderFormat, testMovie)
	want := "Inception (2010)"
	if got != want {
		t.Errorf("FolderName = %q, want %q", got, want)
	}
}

func TestDestPath(t *testing.T) {
	got := renamer.DestPath("/mnt/movies", renamer.DefaultFileFormat, renamer.DefaultFolderFormat, testMovie, testQuality, renamer.ColonSpaceDash, ".mkv")
	want := "/mnt/movies/Inception (2010)/Inception (2010) Bluray-1080p.mkv"
	if got != want {
		t.Errorf("DestPath = %q, want %q", got, want)
	}
}

func TestApply_ZeroYear(t *testing.T) {
	m := renamer.Movie{Title: "Unknown", Year: 0}
	got := renamer.Apply("{Movie Title} ({Release Year})", m, plugin.Quality{})
	// Year should be empty string, not "0"
	want := "Unknown ()"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── Comprehensive table-driven tests (inspired by Sonarr/Radarr) ───────────

func TestApplyWithOptions_ColonStrategies(t *testing.T) {
	movie := renamer.Movie{Title: "Batman: The Dark Knight", Year: 2008}
	format := "{Movie CleanTitle} ({Release Year})"
	qual := plugin.Quality{}

	tests := []struct {
		name  string
		colon renamer.ColonReplacement
		want  string
	}{
		{"delete", renamer.ColonDelete, "Batman The Dark Knight (2008)"},
		{"dash", renamer.ColonDash, "Batman- The Dark Knight (2008)"},
		{"space-dash", renamer.ColonSpaceDash, "Batman - The Dark Knight (2008)"},
		{"smart", renamer.ColonSmart, "Batman - The Dark Knight (2008)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renamer.ApplyWithOptions(format, movie, qual, tt.colon)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApply_Edition(t *testing.T) {
	tests := []struct {
		name   string
		format string
		movie  renamer.Movie
		want   string
	}{
		{
			name:   "edition present",
			format: "{Movie Title} ({Release Year}) {Edition}",
			movie:  renamer.Movie{Title: "Blade Runner", Year: 1982, Edition: "The Final Cut"},
			want:   "Blade Runner (1982) The Final Cut",
		},
		{
			name:   "edition empty",
			format: "{Movie Title} ({Release Year}) {Edition}",
			movie:  renamer.Movie{Title: "Blade Runner", Year: 1982},
			want:   "Blade Runner (1982)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renamer.Apply(tt.format, tt.movie, plugin.Quality{})
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanTitleColon_Comprehensive(t *testing.T) {
	tests := []struct {
		name  string
		title string
		colon renamer.ColonReplacement
		want  string
	}{
		{"basic", "Normal Title", renamer.ColonDelete, "Normal Title"},
		{"colon delete", "CSI: Vegas", renamer.ColonDelete, "CSI Vegas"},
		{"colon dash", "CSI: Vegas", renamer.ColonDash, "CSI- Vegas"},
		{"colon space-dash", "CSI: Vegas", renamer.ColonSpaceDash, "CSI - Vegas"},
		{"smart colon-space", "Batman: Begins", renamer.ColonSmart, "Batman - Begins"},
		{"smart colon-no-space", "Code:Breaker", renamer.ColonSmart, "Code-Breaker"},
		{"multiple colons", "A: B: C", renamer.ColonSpaceDash, "A - B - C"},
		{"angle brackets", "<Title>", renamer.ColonDelete, "Title"},
		{"question mark", "Who?", renamer.ColonDelete, "Who"},
		{"pipe", "A|B", renamer.ColonDelete, "AB"},
		{"quotes", `Say "Hello"`, renamer.ColonDelete, "Say Hello"},
		{"spaces collapsed", "Too   Many   Spaces", renamer.ColonDelete, "Too Many Spaces"},
		{"trimmed", "  Padded  ", renamer.ColonDelete, "Padded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renamer.CleanTitleColon(tt.title, tt.colon)
			if got != tt.want {
				t.Errorf("CleanTitleColon(%q, %q) = %q, want %q", tt.title, tt.colon, got, tt.want)
			}
		})
	}
}

func TestDestPath_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		fileFmt   string
		folderFmt string
		movie     renamer.Movie
		qual      plugin.Quality
		colon     renamer.ColonReplacement
		ext       string
		want      string
	}{
		{
			name:      "basic movie",
			root:      "/media/movies",
			fileFmt:   "{Movie Title} ({Release Year})",
			folderFmt: "{Movie Title} ({Release Year})",
			movie:     renamer.Movie{Title: "Oppenheimer", Year: 2023},
			qual:      plugin.Quality{},
			colon:     renamer.ColonDelete,
			ext:       ".mkv",
			want:      "/media/movies/Oppenheimer (2023)/Oppenheimer (2023).mkv",
		},
		{
			name:      "quality and codec",
			root:      "/data/movies",
			fileFmt:   "{Movie Title} ({Release Year}) [{Quality Full}][{MediaInfo VideoCodec}]",
			folderFmt: "{Movie Title} ({Release Year})",
			movie:     renamer.Movie{Title: "Dune", Year: 2021},
			qual:      plugin.Quality{Name: "Bluray-2160p", Codec: "x265"},
			colon:     renamer.ColonDelete,
			ext:       ".mkv",
			want:      "/data/movies/Dune (2021)/Dune (2021) [Bluray-2160p][x265].mkv",
		},
		{
			name:      "colon in title",
			root:      "/media/movies",
			fileFmt:   "{Movie CleanTitle} ({Release Year})",
			folderFmt: "{Movie CleanTitle} ({Release Year})",
			movie:     renamer.Movie{Title: "Star Wars: A New Hope", Year: 1977},
			qual:      plugin.Quality{},
			colon:     renamer.ColonSpaceDash,
			ext:       ".mkv",
			// FolderName uses Apply() which defaults to ColonDelete, so folder uses delete strategy.
			// File uses the passed colon strategy (ColonSpaceDash).
			want: "/media/movies/Star Wars A New Hope (1977)/Star Wars - A New Hope (1977).mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renamer.DestPath(tt.root, tt.fileFmt, tt.folderFmt, tt.movie, tt.qual, tt.colon, tt.ext)
			if got != tt.want {
				t.Errorf("DestPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
