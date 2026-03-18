package parser

import (
	"strings"
	"testing"

	"github.com/luminarr/luminarr/pkg/plugin"
)

func TestParse_EmptyInput(t *testing.T) {
	t.Parallel()
	got := Parse("")
	if got.Title != "" {
		t.Errorf("Title: got %q, want empty", got.Title)
	}
	if got.Year != 0 {
		t.Errorf("Year: got %d, want 0", got.Year)
	}
	if got.Resolution != plugin.ResolutionUnknown {
		t.Errorf("Resolution: got %q, want unknown", got.Resolution)
	}
	if got.Source != plugin.SourceUnknown {
		t.Errorf("Source: got %q, want unknown", got.Source)
	}
	if got.ReleaseGroup != "" {
		t.Errorf("ReleaseGroup: got %q, want empty", got.ReleaseGroup)
	}
}

func TestParse_SingleWord(t *testing.T) {
	t.Parallel()
	got := Parse("Movie")
	if got.Title != "Movie" {
		t.Errorf("Title: got %q, want %q", got.Title, "Movie")
	}
}

func TestParse_VeryLongTitle(t *testing.T) {
	t.Parallel()
	// 50 words + quality tokens — should not panic or hang.
	words := make([]string, 50)
	for i := range words {
		words[i] = "Word"
	}
	input := strings.Join(words, ".") + ".2024.1080p.BluRay.x264-GRP"
	got := Parse(input)
	if got.Year != 2024 {
		t.Errorf("Year: got %d, want 2024", got.Year)
	}
	if got.Resolution != plugin.Resolution1080p {
		t.Errorf("Resolution: got %q", got.Resolution)
	}
	if got.ReleaseGroup != "GRP" {
		t.Errorf("ReleaseGroup: got %q", got.ReleaseGroup)
	}
}

func TestParse_UnicodeTitle(t *testing.T) {
	t.Parallel()
	got := Parse("Amélie.2001.1080p.BluRay.x264-GRP")
	if got.Year != 2001 {
		t.Errorf("Year: got %d, want 2001", got.Year)
	}
	if got.Source != plugin.SourceBluRay {
		t.Errorf("Source: got %q", got.Source)
	}
}

func TestParse_NumericTitle(t *testing.T) {
	t.Parallel()
	got := Parse("300.2006.1080p.BluRay.x264-GRP")
	if got.Year != 2006 {
		t.Errorf("Year: got %d, want 2006", got.Year)
	}
	if got.Title != "300" {
		t.Errorf("Title: got %q, want %q", got.Title, "300")
	}
}

func TestParse_SpecialCharactersInTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
	}{
		{"hyphen WALL-E", "WALL-E.2008.1080p.BluRay", "WALL-E"},
		{"colon stripped", "Movie.Name.2020.1080p.BluRay", "Movie Name"},
		{"parentheses in title", "Movie.(Special).2020.1080p", "Movie (Special)"},
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

func TestParse_Integration_FullRelease(t *testing.T) {
	t.Parallel()
	// A realistic full release name — verify ALL fields are correctly extracted.
	input := "The.Shawshank.Redemption.1994.Remastered.2160p.BluRay.REMUX.HEVC.DTS-HD.MA.5.1-FraMeSToR"
	got := Parse(input)

	if got.Title != "The Shawshank Redemption" {
		t.Errorf("Title: got %q", got.Title)
	}
	if got.Year != 1994 {
		t.Errorf("Year: got %d", got.Year)
	}
	if got.Resolution != plugin.Resolution2160p {
		t.Errorf("Resolution: got %q", got.Resolution)
	}
	if got.Source != plugin.SourceRemux {
		t.Errorf("Source: got %q", got.Source)
	}
	if got.Codec != plugin.CodecX265 {
		t.Errorf("Codec: got %q", got.Codec)
	}
	if got.AudioCodec != plugin.AudioCodecDTSHDMA {
		t.Errorf("AudioCodec: got %q", got.AudioCodec)
	}
	if got.AudioChannels != plugin.AudioChannels51 {
		t.Errorf("AudioChannels: got %q", got.AudioChannels)
	}
	if got.Edition != "Remastered" {
		t.Errorf("Edition: got %q", got.Edition)
	}
	if got.ReleaseGroup != "FraMeSToR" {
		t.Errorf("ReleaseGroup: got %q", got.ReleaseGroup)
	}
}

func TestParse_Integration_WebDLAtmos(t *testing.T) {
	t.Parallel()
	input := "Dune.Part.Two.2024.2160p.AMZN.WEB-DL.DDP5.1.Atmos.H.265.DoVi-FLUX"
	got := Parse(input)

	if got.Title != "Dune Part Two" {
		t.Errorf("Title: got %q", got.Title)
	}
	if got.Year != 2024 {
		t.Errorf("Year: got %d", got.Year)
	}
	if got.Source != plugin.SourceWEBDL {
		t.Errorf("Source: got %q", got.Source)
	}
	if got.Codec != plugin.CodecX265 {
		t.Errorf("Codec: got %q", got.Codec)
	}
	if got.HDR != plugin.HDRDolbyVision {
		t.Errorf("HDR: got %q", got.HDR)
	}
	if got.AudioCodec != plugin.AudioCodecEAC3Atmos {
		t.Errorf("AudioCodec: got %q", got.AudioCodec)
	}
	if got.AudioChannels != plugin.AudioChannels51 {
		t.Errorf("AudioChannels: got %q", got.AudioChannels)
	}
	if got.ReleaseGroup != "FLUX" {
		t.Errorf("ReleaseGroup: got %q", got.ReleaseGroup)
	}
}

func TestParse_Integration_ProperRepack(t *testing.T) {
	t.Parallel()
	input := "Movie.2024.1080p.BluRay.x264.PROPER.REAL-GRP"
	got := Parse(input)

	if !got.IsProper {
		t.Error("IsProper should be true")
	}
	if got.Revision.Version != 2 {
		t.Errorf("Revision.Version: got %d, want 2", got.Revision.Version)
	}
	if !got.Revision.IsReal {
		t.Error("Revision.IsReal should be true")
	}
}

func TestParse_Integration_InternalLimited(t *testing.T) {
	t.Parallel()
	input := "Movie.2024.1080p.BluRay.INTERNAL.LIMITED.x264-GRP"
	got := Parse(input)

	if !got.IsInternal {
		t.Error("IsInternal should be true")
	}
	if !got.IsLimited {
		t.Error("IsLimited should be true")
	}
}
