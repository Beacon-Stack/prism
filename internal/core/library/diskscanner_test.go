package library

import (
	"testing"

	"github.com/beacon-stack/prism/pkg/plugin"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantYear  int
	}{
		{"standard scene", "Movie.Title.2020.1080p.BluRay.x265.mkv", "Movie Title", 2020},
		{"underscores", "Movie_Title_2010.mkv", "Movie Title 2010", 0},
		{"hyphens", "Movie-Title-2015.mkv", "Movie Title", 2015},
		{"no year", "SomeMovie.mkv", "SomeMovie", 0},
		{"year in title", "2001.A.Space.Odyssey.1968.720p.mkv", "2001 A Space Odyssey", 1968},
		{"title is a year", "1917.2019.1080p.BluRay.mkv", "1917", 2019},
		{"blade runner 2049", "Blade.Runner.2049.2017.UHD.Remux.mkv", "Blade Runner 2049", 2017},
		{"no extension dots", "Inception 2010", "Inception", 2010},
		{"trailing parens", "Movie (2020).mkv", "Movie", 2020},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, year := parseFilename(tt.input)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if year != tt.wantYear {
				t.Errorf("year = %d, want %d", year, tt.wantYear)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"123", true},
		{"1 2 3", true},
		{"abc", false},
		{"12a", false},
		{"  ", true},
	}
	for _, tt := range tests {
		got := isNumeric(tt.input)
		if got != tt.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseQualityFromPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantRes    plugin.Resolution
		wantSource plugin.Source
		wantCodec  plugin.Codec
		wantHDR    plugin.HDRFormat
	}{
		{
			"1080p BluRay x265 HDR10",
			"/movies/Title.2020.1080p.BluRay.x265.HDR.mkv",
			plugin.Resolution1080p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRHDR10,
		},
		{
			"2160p Remux HEVC DolbyVision",
			"/movies/Title.2020.2160p.Remux.HEVC.Dolby.Vision.mkv",
			plugin.Resolution2160p, plugin.SourceRemux, plugin.CodecX265, plugin.HDRDolbyVision,
		},
		{
			"720p WEB-DL x264",
			"/movies/Title.720p.WEB-DL.x264.mkv",
			plugin.Resolution720p, plugin.SourceWEBDL, plugin.CodecX264, plugin.HDRNone,
		},
		{
			"4K UHD HDR10+",
			"/movies/Title.4K.UHD.HDR10PLUS.mkv",
			plugin.Resolution2160p, plugin.SourceUnknown, plugin.CodecUnknown, plugin.HDRHDR10Plus,
		},
		{
			"WEBRip AV1",
			"/movies/Title.WEBRip.AV1.mkv",
			plugin.ResolutionUnknown, plugin.SourceWEBRip, plugin.CodecAV1, plugin.HDRNone,
		},
		{
			"HDTV",
			"/movies/Title.HDTV.mkv",
			plugin.ResolutionUnknown, plugin.SourceHDTV, plugin.CodecUnknown, plugin.HDRNone,
		},
		{
			"DVD",
			"/movies/Title.DVDRip.mkv",
			plugin.ResolutionUnknown, plugin.SourceDVD, plugin.CodecUnknown, plugin.HDRNone,
		},
		{
			"HLG",
			"/movies/Title.2160p.HLG.mkv",
			plugin.Resolution2160p, plugin.SourceUnknown, plugin.CodecUnknown, plugin.HDRHLG,
		},
		{
			"nothing detectable",
			"/movies/random-file.mkv",
			plugin.ResolutionUnknown, plugin.SourceUnknown, plugin.CodecUnknown, plugin.HDRNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := ParseQualityFromPath(tt.path)
			if q.Resolution != tt.wantRes {
				t.Errorf("resolution = %q, want %q", q.Resolution, tt.wantRes)
			}
			if q.Source != tt.wantSource {
				t.Errorf("source = %q, want %q", q.Source, tt.wantSource)
			}
			if q.Codec != tt.wantCodec {
				t.Errorf("codec = %q, want %q", q.Codec, tt.wantCodec)
			}
			if q.HDR != tt.wantHDR {
				t.Errorf("hdr = %q, want %q", q.HDR, tt.wantHDR)
			}
		})
	}
}

func TestBuildQualityName(t *testing.T) {
	tests := []struct {
		name  string
		input plugin.Quality
		want  string
	}{
		{
			"all unknown",
			plugin.Quality{
				Resolution: plugin.ResolutionUnknown,
				Source:     plugin.SourceUnknown,
				Codec:      plugin.CodecUnknown,
				HDR:        plugin.HDRNone,
			},
			"Unknown",
		},
		{
			"full quality",
			plugin.Quality{
				Resolution: plugin.Resolution1080p,
				Source:     plugin.SourceBluRay,
				Codec:      plugin.CodecX265,
				HDR:        plugin.HDRHDR10,
			},
			"bluray 1080p x265 hdr10",
		},
		{
			"partial quality",
			plugin.Quality{
				Resolution: plugin.Resolution720p,
				Source:     plugin.SourceUnknown,
				Codec:      plugin.CodecX264,
				HDR:        plugin.HDRNone,
			},
			"720p x264",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQualityName(tt.input)
			if got != tt.want {
				t.Errorf("buildQualityName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeCandidateTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"The Matrix", "the matrix"},
		{"Spider-Man: No Way Home", "spiderman no way home"},
		{"  Extra   Spaces  ", "extra spaces"},
		{"UPPER CASE", "upper case"},
		{"Title's Apostrophe", "titles apostrophe"},
	}
	for _, tt := range tests {
		got := normalizeCandidateTitle(tt.input)
		if got != tt.want {
			t.Errorf("normalizeCandidateTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarshalTags(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"nil", nil, "[]"},
		{"empty", []string{}, "[]"},
		{"single", []string{"action"}, `["action"]`},
		{"multi", []string{"action", "sci-fi"}, `["action","sci-fi"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalTags(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("marshalTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiskFree_TempDir(t *testing.T) {
	dir := t.TempDir()
	free := diskFree(dir)
	if free <= 0 {
		t.Errorf("diskFree(%q) = %d, want > 0", dir, free)
	}
}

func TestDiskFree_NonExistent(t *testing.T) {
	free := diskFree("/definitely-does-not-exist-xyz")
	if free != -1 {
		t.Errorf("diskFree(nonexistent) = %d, want -1", free)
	}
}
