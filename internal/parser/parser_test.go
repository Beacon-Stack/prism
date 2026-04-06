package parser

import (
	"testing"

	"github.com/beacon-media/prism/pkg/plugin"
)

// ── Video quality tests (ported from quality/parser_test.go) ─────────────────

func TestParse_VideoQuality(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantRes    plugin.Resolution
		wantSource plugin.Source
		wantCodec  plugin.Codec
		wantHDR    plugin.HDRFormat
		wantName   string
	}{
		{"bluray 1080p x264", "The.Dark.Knight.2008.1080p.BluRay.x264-GROUP", plugin.Resolution1080p, plugin.SourceBluRay, plugin.CodecX264, plugin.HDRNone, "Bluray-1080p x264"},
		{"bluray 1080p x265 dashed", "Interstellar.2014.1080p.Blu-Ray.x265-YIFY", plugin.Resolution1080p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRNone, "Bluray-1080p x265"},
		{"bluray 1080p HEVC", "Inception.2010.1080p.BLURAY.HEVC-GROUP", plugin.Resolution1080p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRNone, "Bluray-1080p x265"},
		{"bluray 1080p no codec", "Pulp.Fiction.1994.1080p.BluRay-SPARKS", plugin.Resolution1080p, plugin.SourceBluRay, plugin.CodecUnknown, plugin.HDRNone, "Bluray-1080p"},
		{"bluray 2160p x265", "Dune.2021.2160p.BluRay.x265.10bit-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRNone, "Bluray-2160p x265"},
		{"UHD maps to 2160p", "Avatar.2009.UHD.BluRay.x265-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRNone, ""},
		{"4K maps to 2160p", "Top.Gun.Maverick.2022.4K.BluRay.x265-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRNone, ""},
		{"remux 1080p", "The.Godfather.1972.1080p.BluRay.REMUX.AVC-GROUP", plugin.Resolution1080p, plugin.SourceRemux, plugin.CodecX264, plugin.HDRNone, "Bluray Remux-1080p x264"},
		{"remux 2160p", "Mad.Max.Fury.Road.2015.2160p.BluRay.REMUX.HEVC-FGT", plugin.Resolution2160p, plugin.SourceRemux, plugin.CodecX265, plugin.HDRNone, "Bluray Remux-2160p x265"},
		{"BDREMUX", "Parasite.2019.1080p.BDREMUX.x265-GROUP", plugin.Resolution1080p, plugin.SourceRemux, plugin.CodecX265, plugin.HDRNone, ""},
		{"WEB-DL 1080p x264", "The.Crown.S01E01.2016.1080p.WEB-DL.DD5.1.H.264-GROUP", plugin.Resolution1080p, plugin.SourceWEBDL, plugin.CodecX264, plugin.HDRNone, "WEBDL-1080p x264"},
		{"WEB.DL dot-separated", "Oppenheimer.2023.1080p.WEB.DL.x265-GROUP", plugin.Resolution1080p, plugin.SourceWEBDL, plugin.CodecX265, plugin.HDRNone, ""},
		{"WEBRip 1080p", "The.Mandalorian.S01E01.2019.1080p.WEBRip.x264-GROUP", plugin.Resolution1080p, plugin.SourceWEBRip, plugin.CodecX264, plugin.HDRNone, "WEBRip-1080p x264"},
		{"HDTV 720p", "Game.of.Thrones.S08E06.2019.720p.HDTV.x264-GROUP", plugin.Resolution720p, plugin.SourceHDTV, plugin.CodecX264, plugin.HDRNone, "HDTV-720p x264"},
		{"HDR10", "Blade.Runner.2049.2017.2160p.BluRay.x265.HDR10-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRHDR10, "Bluray-2160p x265 HDR10"},
		{"Dolby Vision DV", "The.Batman.2022.2160p.BluRay.x265.DV-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRDolbyVision, "Bluray-2160p x265 Dolby Vision"},
		{"Dolby Vision DoVi", "Severance.S01E01.2022.2160p.ATVP.WEB-DL.DoVi.x265-GROUP", plugin.Resolution2160p, plugin.SourceWEBDL, plugin.CodecX265, plugin.HDRDolbyVision, ""},
		{"HDR10Plus", "Dune.Part.Two.2024.2160p.BluRay.x265.HDR10Plus-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRHDR10Plus, "Bluray-2160p x265 HDR10+"},
		{"HLG", "One.Piece.Film.Red.2022.2160p.BluRay.x265.HLG-GROUP", plugin.Resolution2160p, plugin.SourceBluRay, plugin.CodecX265, plugin.HDRHLG, "Bluray-2160p x265 HLG"},
		{"AV1", "Killers.of.the.Flower.Moon.2023.1080p.WEB-DL.AV1-GROUP", plugin.Resolution1080p, plugin.SourceWEBDL, plugin.CodecAV1, plugin.HDRNone, "WEBDL-1080p AV1"},
		{"DVDRip XviD", "The.Matrix.1999.DVDRip.XviD-GROUP", plugin.ResolutionSD, plugin.SourceDVD, plugin.CodecXVID, plugin.HDRNone, "DVD-SD XviD"},
		{"CAM", "Avengers.Endgame.2019.CAM.x264-GROUP", plugin.ResolutionUnknown, plugin.SourceCAM, plugin.CodecX264, plugin.HDRNone, "CAM x264"},
		{"TS telesync", "Fast.X.2023.TS.x264-GROUP", plugin.ResolutionUnknown, plugin.SourceTelesync, plugin.CodecX264, plugin.HDRNone, "Telesync x264"},
		{"WORKPRINT", "Some.Movie.2024.WORKPRINT-GROUP", plugin.ResolutionUnknown, plugin.SourceWorkprint, plugin.CodecUnknown, plugin.HDRNone, "Workprint"},
		{"DVDSCR", "Awards.Movie.2024.DVDSCR.x264-GROUP", plugin.ResolutionSD, plugin.SourceDVDSCR, plugin.CodecX264, plugin.HDRNone, "DVDSCR-SD x264"},
		{"BDMV BR-DISK", "Movie.2023.BDMV-GROUP", plugin.ResolutionUnknown, plugin.SourceBRDisk, plugin.CodecUnknown, plugin.HDRNone, "BR-DISK"},
		{"RAW-HD", "Concert.2022.RAW-HD-GROUP", plugin.ResolutionUnknown, plugin.SourceRawHD, plugin.CodecUnknown, plugin.HDRNone, "Raw-HD"},
		{"no quality info", "SomeOldMovie-GROUP", plugin.ResolutionUnknown, plugin.SourceUnknown, plugin.CodecUnknown, plugin.HDRNone, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Resolution != tc.wantRes {
				t.Errorf("Resolution: got %q, want %q", got.Resolution, tc.wantRes)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source: got %q, want %q", got.Source, tc.wantSource)
			}
			if got.Codec != tc.wantCodec {
				t.Errorf("Codec: got %q, want %q", got.Codec, tc.wantCodec)
			}
			if got.HDR != tc.wantHDR {
				t.Errorf("HDR: got %q, want %q", got.HDR, tc.wantHDR)
			}
			if tc.wantName != "" && got.QualityName != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.QualityName, tc.wantName)
			}
		})
	}
}

// ── Audio tests (ported from quality/parser_test.go TestParseAudio) ──────────

func TestParse_Audio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		input        string
		wantAudio    plugin.AudioCodec
		wantChannels plugin.AudioChannels
	}{
		{"TrueHD Atmos", "Movie.2024.1080p.BluRay.TrueHD.Atmos.x265-GRP", plugin.AudioCodecTrueHDAtmos, plugin.AudioChannelsUnknown},
		{"TrueHD Atmos 7.1", "Movie.2024.2160p.BluRay.REMUX.HEVC.TrueHD.7.1.Atmos-GRP", plugin.AudioCodecTrueHDAtmos, plugin.AudioChannels71},
		{"TrueHD bare", "Movie.2024.1080p.BluRay.TrueHD.5.1.x264-GRP", plugin.AudioCodecTrueHD, plugin.AudioChannels51},
		{"DTS-X", "Movie.2024.2160p.BluRay.DTS-X.x265-GRP", plugin.AudioCodecDTSX, plugin.AudioChannelsUnknown},
		{"DTS-HD MA 7.1", "Movie.2024.1080p.BluRay.DTS-HD.MA.7.1.x264-DON", plugin.AudioCodecDTSHDMA, plugin.AudioChannels71},
		{"DTS-HD bare", "Movie.2024.1080p.BluRay.DTS-HD.5.1.x264-GRP", plugin.AudioCodecDTSHD, plugin.AudioChannels51},
		{"DTS bare", "Movie.2024.1080p.BluRay.DTS.x264-GRP", plugin.AudioCodecDTS, plugin.AudioChannelsUnknown},
		{"DDP5.1 Atmos", "Movie.2024.1080p.WEB-DL.DDP5.1.Atmos.H.265-GRP", plugin.AudioCodecEAC3Atmos, plugin.AudioChannels51},
		{"DDP5.1", "Movie.2024.1080p.WEB-DL.DDP5.1.H.264-GRP", plugin.AudioCodecEAC3, plugin.AudioChannels51},
		{"bare Atmos", "Movie.2024.1080p.WEB-DL.Atmos.H.265-GRP", plugin.AudioCodecEAC3Atmos, plugin.AudioChannelsUnknown},
		{"DD5.1", "Movie.2024.1080p.WEB-DL.DD5.1.H.264-GRP", plugin.AudioCodecAC3, plugin.AudioChannels51},
		{"AAC 2.0", "Movie.2024.1080p.WEB-DL.AAC2.0.x264-GRP", plugin.AudioCodecAAC, plugin.AudioChannels20},
		{"FLAC", "Movie.2024.1080p.BluRay.FLAC.x264-GRP", plugin.AudioCodecFLAC, plugin.AudioChannelsUnknown},
		{"PCM", "Movie.2024.1080p.BluRay.PCM.x264-GRP", plugin.AudioCodecPCM, plugin.AudioChannelsUnknown},
		{"LPCM", "Movie.2024.1080p.BluRay.LPCM.2.0.x264-GRP", plugin.AudioCodecPCM, plugin.AudioChannels20},
		{"Opus", "Movie.2024.1080p.WEB-DL.Opus.5.1.AV1-GRP", plugin.AudioCodecOpus, plugin.AudioChannels51},
		{"no audio", "Movie.2024.1080p.BluRay.x264-GRP", plugin.AudioCodecUnknown, plugin.AudioChannelsUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.AudioCodec != tc.wantAudio {
				t.Errorf("AudioCodec: got %q, want %q", got.AudioCodec, tc.wantAudio)
			}
			if got.AudioChannels != tc.wantChannels {
				t.Errorf("AudioChannels: got %q, want %q", got.AudioChannels, tc.wantChannels)
			}
		})
	}
}

// ── Edition tests (ported from edition/parser_test.go) ───────────────────────

func TestParse_Edition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Directors Cut", "Blade.Runner.1982.Directors.Cut.1080p.BluRay.x264-GROUP", "Director's Cut"},
		{"Director's Cut apostrophe", "Blade.Runner.1982.Director's.Cut.1080p.BluRay.x265-GROUP", "Director's Cut"},
		{"Extended Cut", "Kingdom.of.Heaven.2005.Extended.Cut.1080p.BluRay.x264-GROUP", "Extended"},
		{"bare Extended", "The.Lord.of.the.Rings.2001.Extended.2160p.UHD.BluRay.REMUX.HDR.HEVC-GROUP", "Extended"},
		{"Theatrical Cut", "Donnie.Darko.2001.Theatrical.Cut.1080p.BluRay.x264-GROUP", "Theatrical"},
		{"bare Theatrical", "Zack.Snyders.Justice.League.2021.Theatrical.2160p.WEB-DL.x265-GROUP", "Theatrical"},
		{"Unrated", "Bad.Santa.2003.Unrated.1080p.BluRay.x264-GROUP", "Unrated"},
		{"Uncensored", "Movie.2020.Uncensored.1080p.WEB-DL.x264-GROUP", "Unrated"},
		{"IMAX", "Justice.League.2021.IMAX.2160p.WEB-DL.DDP5.1.HDR.HEVC-GROUP", "IMAX"},
		{"Final Cut", "Blade.Runner.1982.The.Final.Cut.2160p.UHD.BluRay.x265-GROUP", "Final Cut"},
		{"Redux", "Apocalypse.Now.1979.Redux.1080p.BluRay.x264-GROUP", "Redux"},
		{"Remastered", "Jaws.1975.Remastered.1080p.BluRay.x265-GROUP", "Remastered"},
		{"4K Remastered", "Jaws.1975.4K.Remastered.2160p.BluRay.x265-GROUP", "Remastered"},
		{"Special Edition", "Aliens.1986.Special.Edition.1080p.BluRay.x265-GROUP", "Special Edition"},
		{"Criterion", "Stalker.1979.Criterion.1080p.BluRay.x265-GROUP", "Criterion"},
		{"Ultimate Cut", "Watchmen.2009.Ultimate.Cut.1080p.BluRay.x264-GROUP", "Ultimate"},
		{"Anniversary", "E.T.1982.Anniversary.Edition.1080p.BluRay.x264-GROUP", "Anniversary"},
		{"40th Anniversary", "Alien.1979.40th.Anniversary.Edition.2160p.BluRay.x265-GROUP", "Anniversary"},
		{"Rogue Cut", "X-Men.Days.of.Future.Past.2014.Rogue.Cut.1080p.BluRay.x264-GROUP", "Rogue Cut"},
		{"Black and Chrome", "Mad.Max.Fury.Road.2015.Black.and.Chrome.1080p.BluRay.x264-GROUP", "Black and Chrome"},
		{"Open Matte", "The.Shining.1980.Open.Matte.1080p.BluRay.x264-GROUP", "Open Matte"},
		{"no edition", "Movie.2020.1080p.BluRay.x264-GROUP", ""},
		{"DC abbreviation not matched", "Movie.2020.DC.1080p.BluRay.x264-GROUP", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != tc.want {
				t.Errorf("Edition: got %q, want %q", got.Edition, tc.want)
			}
		})
	}
}

// ── Title + year tests (ported from movie/parser_test.go) ────────────────────

func TestParse_Title(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantYear  int
	}{
		{"standard", "The.Dark.Knight.2008.1080p.BluRay.x264-GROUP", "The Dark Knight", 2008},
		{"2160p UHD", "Inception.2010.2160p.UHD.BluRay.x265.HEVC-GROUP", "Inception", 2010},
		{"REPACK", "The.Shawshank.Redemption.1994.REPACK.1080p.BluRay.x264", "The Shawshank Redemption", 1994},
		{"underscores", "Interstellar_2014_BluRay_1080p_x264", "Interstellar", 2014},
		{"year in title", "2001.A.Space.Odyssey.1968.1080p.BluRay", "2001 A Space Odyssey", 1968},
		{"year-titled movie", "1917.2019.1080p.WEBRip", "1917", 2019},
		{"no year", "Alien.1080p.BluRay.x264", "Alien", 0},
		{"mkv extension", "The.Godfather.1972.1080p.BluRay.mkv", "The Godfather", 1972},
		{"full path", "/media/movies/The.Matrix.1999.1080p.BluRay.x264/The.Matrix.1999.1080p.BluRay.x264.mkv", "The Matrix", 1999},
		{"spaces", "Joker 2019 1080p BluRay", "Joker", 2019},
		{"all caps disc rip", "THE_HUNGERGAMES_MOCKINGJAY_PT1_Title31", "The Hungergames Mockingjay Part 1", 0},
		{"Pt2 normalization", "Avengers.Infinity.War.Pt2.2018.WEBRip", "Avengers Infinity War Part 2", 2018},
		{"Pt.2 normalization", "Harry.Potter.And.The.Deathly.Hallows.Pt.2.2011.1080p.BluRay", "Harry Potter And The Deathly Hallows Part 2", 2011},
		{"disc noise Title01", "The.Dark.Knight.Rises.2012.Title01.1080p", "The Dark Knight Rises", 2012},
		{"all caps underscores", "THE_DARK_KNIGHT_2008_1080P_BLURAY", "The Dark Knight", 2008},
		{"WALL-E hyphen", "WALL-E.2008.1080p.BluRay", "WALL-E", 2008},
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

// ── Release group tests (ported from quality/parser_test.go) ─────────────────

func TestParse_ReleaseGroup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard", "The.Dark.Knight.2008.1080p.BluRay.x264-FraMeSToR", "FraMeSToR"},
		{"WEB-DL with group", "Movie.2024.1080p.WEB-DL.DD5.1.H.264-NTb", "NTb"},
		{"DTS-HD MA with group", "Movie.2024.1080p.BluRay.DTS-HD.MA.x264-DON", "DON"},
		{"DTS-X with group", "Movie.2024.2160p.BluRay.REMUX.DTS-X.x265-GRP", "GRP"},
		{"no group", "Movie.2024.1080p.BluRay.x264", ""},
		{"RAW-HD no group", "Concert.2022.RAW-HD", ""},
		{"bracket group", "Movie.2024.1080p.BluRay.x264.[D-Z0N3]", "D-Z0N3"},
		{"mkv stripped", "The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv", "GROUP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.ReleaseGroup != tc.want {
				t.Errorf("ReleaseGroup: got %q, want %q", got.ReleaseGroup, tc.want)
			}
		})
	}
}

// ── Markers tests ────────────────────────────────────────────────────────────

func TestParse_Markers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		isProper bool
		isRepack bool
		isHybrid bool
		revision int
	}{
		{"PROPER", "Movie.2024.1080p.BluRay.x264.PROPER-GRP", true, false, false, 2},
		{"REPACK", "Movie.2024.1080p.WEB-DL.x265.REPACK-GRP", false, true, false, 2},
		{"PROPER2", "Movie.2024.1080p.BluRay.PROPER2.x264-GRP", true, false, false, 3},
		{"HYBRID", "Movie.2024.1080p.BluRay.Hybrid.x265-GRP", false, false, true, 1},
		{"plain release", "Movie.2024.1080p.BluRay.x264-GRP", false, false, false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.IsProper != tc.isProper {
				t.Errorf("IsProper: got %v, want %v", got.IsProper, tc.isProper)
			}
			if got.IsRepack != tc.isRepack {
				t.Errorf("IsRepack: got %v, want %v", got.IsRepack, tc.isRepack)
			}
			if got.IsHybrid != tc.isHybrid {
				t.Errorf("IsHybrid: got %v, want %v", got.IsHybrid, tc.isHybrid)
			}
			if got.Revision.Version != tc.revision {
				t.Errorf("Revision.Version: got %d, want %d", got.Revision.Version, tc.revision)
			}
		})
	}
}

// ── Quality() compatibility test ─────────────────────────────────────────────

func TestParse_QualityCompat(t *testing.T) {
	p := Parse("The.Dark.Knight.2008.1080p.BluRay.DTS-HD.MA.7.1.x264-FraMeSToR")
	q := p.Quality()
	if q.Resolution != plugin.Resolution1080p {
		t.Errorf("Resolution: got %q", q.Resolution)
	}
	if q.Source != plugin.SourceBluRay {
		t.Errorf("Source: got %q", q.Source)
	}
	if q.Codec != plugin.CodecX264 {
		t.Errorf("Codec: got %q", q.Codec)
	}
	if q.AudioCodec != plugin.AudioCodecDTSHDMA {
		t.Errorf("AudioCodec: got %q", q.AudioCodec)
	}
	if q.AudioChannels != plugin.AudioChannels71 {
		t.Errorf("AudioChannels: got %q", q.AudioChannels)
	}
	if q.Name != "Bluray-1080p x264" {
		t.Errorf("Name: got %q", q.Name)
	}
}
