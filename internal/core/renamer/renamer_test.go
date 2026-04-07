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
