package parser

import (
	"testing"

	"github.com/beacon-media/prism/pkg/plugin"
)

func TestParseSource_AllVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.Source
	}{
		// ── BluRay variants ──────────────────────────────────────────────
		{"BluRay", "Movie.2024.1080p.BluRay.x264", plugin.SourceBluRay},
		{"Blu-Ray dashed", "Movie.2024.1080p.Blu-Ray.x264", plugin.SourceBluRay},
		{"BLURAY uppercase", "Movie.2024.1080p.BLURAY.x264", plugin.SourceBluRay},
		{"Blu Ray space", "Movie 2024 1080p Blu Ray x264", plugin.SourceBluRay},

		// ── Remux must beat BluRay ───────────────────────────────────────
		{"REMUX with BluRay", "Movie.2024.1080p.BluRay.Remux.x265", plugin.SourceRemux},
		{"BDREMUX", "Movie.2024.1080p.BDREMUX.x265", plugin.SourceRemux},

		// ── WEB-DL variants ─────────────────────────────────────────────
		{"WEB-DL", "Movie.2024.1080p.WEB-DL.x264", plugin.SourceWEBDL},
		{"WEBDL", "Movie.2024.1080p.WEBDL.x264", plugin.SourceWEBDL},
		{"WEB.DL", "Movie.2024.1080p.WEB.DL.x264", plugin.SourceWEBDL},
		{"WEB DL spaced", "Movie 2024 1080p WEB DL x264", plugin.SourceWEBDL},

		// ── WEBRip variants ─────────────────────────────────────────────
		{"WEBRip", "Movie.2024.1080p.WEBRip.x264", plugin.SourceWEBRip},
		{"WEB-Rip", "Movie.2024.1080p.WEB-Rip.x264", plugin.SourceWEBRip},
		{"WEB.Rip", "Movie.2024.1080p.WEB.Rip.x264", plugin.SourceWEBRip},

		// ── DVDSCR must beat DVD ────────────────────────────────────────
		{"DVDSCR", "Movie.2024.DVDSCR.x264", plugin.SourceDVDSCR},
		{"SCREENER", "Movie.2024.SCREENER.x264", plugin.SourceDVDSCR},
		{"SCR", "Movie.2024.SCR.x264", plugin.SourceDVDSCR},

		// ── DVDR variants ───────────────────────────────────────────────
		{"DVDR", "Movie.2024.DVDR", plugin.SourceDVDR},
		{"DVD-R", "Movie.2024.DVD-R", plugin.SourceDVDR},
		{"DVD9", "Movie.2024.DVD9", plugin.SourceDVDR},
		{"DVD5", "Movie.2024.DVD5", plugin.SourceDVDR},

		// ── DVD ──────────────────────────────────────────────────────────
		{"DVDRip", "Movie.2024.DVDRip.XviD", plugin.SourceDVD},
		{"DVD.Rip", "Movie.2024.DVD.Rip.x264", plugin.SourceDVD},

		// ── BR-DISK variants ────────────────────────────────────────────
		{"BDMV", "Movie.2024.BDMV", plugin.SourceBRDisk},
		{"BD25", "Movie.2024.BD25", plugin.SourceBRDisk},
		{"BD50", "Movie.2024.BD50", plugin.SourceBRDisk},
		{"BR-DISK", "Movie.2024.BR-DISK", plugin.SourceBRDisk},

		// ── Pre-release sources ─────────────────────────────────────────
		{"CAM", "Movie.2024.CAM.x264", plugin.SourceCAM},
		{"HDCAM", "Movie.2024.HDCAM", plugin.SourceCAM},
		{"CAMRIP", "Movie.2024.CAMRIP.x264", plugin.SourceCAM},
		{"TS telesync", "Movie.2024.TS.x264", plugin.SourceTelesync},
		{"TELESYNC", "Movie.2024.TELESYNC", plugin.SourceTelesync},
		{"HDTS", "Movie.2024.HDTS.x264", plugin.SourceTelesync},
		{"PDVD", "Movie.2024.PDVD.x264", plugin.SourceTelesync},
		{"TC telecine", "Movie.2024.TC", plugin.SourceTELECINE},
		{"TELECINE", "Movie.2024.TELECINE", plugin.SourceTELECINE},
		{"HDTC", "Movie.2024.HDTC.x264", plugin.SourceTELECINE},
		{"WORKPRINT", "Movie.2024.WORKPRINT", plugin.SourceWorkprint},
		{"WP", "Movie.2024.WP.x264", plugin.SourceWorkprint},
		{"R5 regional", "Movie.2024.R5.x264", plugin.SourceRegional},
		{"REGIONAL", "Movie.2024.REGIONAL", plugin.SourceRegional},
		{"RAW-HD", "Movie.2024.RAW-HD", plugin.SourceRawHD},
		{"HDTV", "Movie.2024.720p.HDTV.x264", plugin.SourceHDTV},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Source != tc.want {
				t.Errorf("Source: got %q, want %q", got.Source, tc.want)
			}
		})
	}
}

func TestParseResolution_SDInference(t *testing.T) {
	t.Parallel()
	// DVD-family sources without explicit resolution should imply SD.
	tests := []struct {
		name  string
		input string
		want  plugin.Resolution
	}{
		{"DVDRip implies SD", "Movie.2024.DVDRip.XviD", plugin.ResolutionSD},
		{"DVDR implies SD", "Movie.2024.DVDR", plugin.ResolutionSD},
		{"DVDSCR implies SD", "Movie.2024.DVDSCR.x264", plugin.ResolutionSD},
		{"Regional implies SD", "Movie.2024.R5.x264", plugin.ResolutionSD},
		{"480p explicit", "Movie.2024.480p.DVDRip.x264", plugin.Resolution480p},
		{"576p explicit", "Movie.2024.576p.DVDRip.XviD", plugin.Resolution576p},
		{"BluRay no res is unknown", "Movie.2024.BluRay.x264", plugin.ResolutionUnknown},
		{"CAM no res is unknown", "Movie.2024.CAM.x264", plugin.ResolutionUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Resolution != tc.want {
				t.Errorf("Resolution: got %q, want %q", got.Resolution, tc.want)
			}
		})
	}
}

func TestParseHDR_Ordering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.HDRFormat
	}{
		// DolbyVision must beat HDR10.
		{"DV beats HDR10", "Movie.2024.2160p.BluRay.x265.DV.HDR10", plugin.HDRDolbyVision},
		{"DoVi", "Movie.2024.2160p.BluRay.DoVi.x265", plugin.HDRDolbyVision},
		{"Dolby Vision full", "Movie.2024.2160p.BluRay.Dolby.Vision.x265", plugin.HDRDolbyVision},
		{"Dolby-Vision dashed", "Movie.2024.2160p.Dolby-Vision.x265", plugin.HDRDolbyVision},
		// HDR10+ must beat HDR10.
		{"HDR10Plus beats HDR10", "Movie.2024.2160p.BluRay.x265.HDR10Plus", plugin.HDRHDR10Plus},
		{"HDR10+ symbol", "Movie.2024.2160p.BluRay.x265.HDR10+", plugin.HDRHDR10Plus},
		// Bare HDR maps to HDR10.
		{"bare HDR", "Movie.2024.2160p.BluRay.HDR", plugin.HDRHDR10},
		{"HDR10 explicit", "Movie.2024.2160p.BluRay.HDR10", plugin.HDRHDR10},
		// HLG.
		{"HLG", "Movie.2024.2160p.BluRay.HLG", plugin.HDRHLG},
		{"HLG with DV present", "Movie.2024.2160p.HLG.Dolby.Vision", plugin.HDRDolbyVision},
		// No HDR.
		{"no HDR", "Movie.2024.1080p.BluRay.x264", plugin.HDRNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.HDR != tc.want {
				t.Errorf("HDR: got %q, want %q", got.HDR, tc.want)
			}
		})
	}
}

func TestParseCodec_HVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.Codec
	}{
		{"x265", "Movie.2024.1080p.BluRay.x265", plugin.CodecX265},
		{"H.265 dotted", "Movie.2024.1080p.BluRay.H.265", plugin.CodecX265},
		{"H-265 dashed", "Movie.2024.1080p.BluRay.H-265", plugin.CodecX265},
		{"H 265 spaced", "Movie 2024 1080p BluRay H 265", plugin.CodecX265},
		{"HEVC", "Movie.2024.1080p.BluRay.HEVC", plugin.CodecX265},
		{"x264", "Movie.2024.1080p.BluRay.x264", plugin.CodecX264},
		{"H.264 dotted", "Movie.2024.1080p.BluRay.H.264", plugin.CodecX264},
		{"H-264 dashed", "Movie.2024.1080p.BluRay.H-264", plugin.CodecX264},
		{"AVC", "Movie.2024.1080p.BluRay.AVC", plugin.CodecX264},
		{"AV1", "Movie.2024.1080p.BluRay.AV1", plugin.CodecAV1},
		{"XviD", "Movie.2024.DVDRip.XviD", plugin.CodecXVID},
		{"DivX", "Movie.2024.DVDRip.DivX", plugin.CodecXVID},
		// x265 must beat x264 even if both present.
		{"x265 beats x264", "Movie.2024.1080p.x265.x264", plugin.CodecX265},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Codec != tc.want {
				t.Errorf("Codec: got %q, want %q", got.Codec, tc.want)
			}
		})
	}
}

func TestBuildQualityName_Combinations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"resolution only", "Movie.1080p", "1080p"},
		{"source only", "Movie.BluRay", "Bluray"},
		{"no quality", "SomeMovie", ""},
		{"source+res+codec+hdr", "Movie.2160p.BluRay.x265.HDR10", "Bluray-2160p x265 HDR10"},
		{"source+res no codec", "Movie.1080p.BluRay", "Bluray-1080p"},
		{"codec only", "Movie.x264", "x264"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.QualityName != tc.want {
				t.Errorf("QualityName: got %q, want %q", got.QualityName, tc.want)
			}
		})
	}
}

func TestParseSource_FalsePositives(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.Source
	}{
		// "SCAM" contains "CAM" but word boundary should prevent match.
		{"SCAM not CAM", "The.Great.Scam.2020.1080p.BluRay.x264", plugin.SourceBluRay},
		// "BECAME" contains "CAM" but should not match.
		{"BECAME not CAM", "She.Became.2020.1080p.BluRay.x264", plugin.SourceBluRay},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Source != tc.want {
				t.Errorf("Source: got %q, want %q", got.Source, tc.want)
			}
		})
	}
}
