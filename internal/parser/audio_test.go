package parser

import (
	"testing"

	"github.com/beacon-media/prism/pkg/plugin"
)

func TestParseAudioCodec_AllVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.AudioCodec
	}{
		// ── TrueHD family ────────────────────────────────────────────────
		{"TrueHD.Atmos adjacent", "Movie.1080p.BluRay.TrueHD.Atmos.x265", plugin.AudioCodecTrueHDAtmos},
		{"TrueHD Atmos with channels between", "Movie.2160p.REMUX.TrueHD.7.1.Atmos", plugin.AudioCodecTrueHDAtmos},
		{"TrueHD bare", "Movie.1080p.BluRay.TrueHD.x264", plugin.AudioCodecTrueHD},

		// ── DTS family ───────────────────────────────────────────────────
		{"DTS-X", "Movie.2160p.BluRay.DTS-X.x265", plugin.AudioCodecDTSX},
		{"DTS-HD MA", "Movie.1080p.BluRay.DTS-HD.MA.x264", plugin.AudioCodecDTSHDMA},
		{"DTS-HD Master Audio", "Movie.1080p.BluRay.DTS-HD.Master.Audio.x264", plugin.AudioCodecDTSHDMA},
		{"DTS-HD bare", "Movie.1080p.BluRay.DTS-HD.x264", plugin.AudioCodecDTSHD},
		{"DTS bare", "Movie.1080p.BluRay.DTS.x264", plugin.AudioCodecDTS},

		// ── EAC3 / DD+ family ────────────────────────────────────────────
		{"DDP with Atmos", "Movie.1080p.WEB-DL.DDP5.1.Atmos.H.265", plugin.AudioCodecEAC3Atmos},
		{"DDP without Atmos", "Movie.1080p.WEB-DL.DDP5.1.H.264", plugin.AudioCodecEAC3},
		{"EAC3", "Movie.1080p.WEB-DL.EAC3.x264", plugin.AudioCodecEAC3},
		{"DD+", "Movie.720p.WEB-DL.DD+.5.1.x264", plugin.AudioCodecEAC3},
		{"DDPLUS", "Movie.1080p.WEB-DL.DDPLUS.5.1.x264", plugin.AudioCodecEAC3},
		{"E-AC-3", "Movie.1080p.WEB-DL.E-AC-3.x264", plugin.AudioCodecEAC3},
		{"bare Atmos implies EAC3", "Movie.1080p.WEB-DL.Atmos.H.265", plugin.AudioCodecEAC3Atmos},

		// ── AC3 / DD family ──────────────────────────────────────────────
		{"DD5.1", "Movie.1080p.WEB-DL.DD5.1.H.264", plugin.AudioCodecAC3},
		{"AC3", "Movie.1080p.BluRay.AC3.x264", plugin.AudioCodecAC3},
		{"DD bare", "Movie.1080p.BluRay.DD.x264", plugin.AudioCodecAC3},

		// ── Other codecs ─────────────────────────────────────────────────
		{"AAC", "Movie.1080p.WEB-DL.AAC.x264", plugin.AudioCodecAAC},
		{"AAC2.0 combined", "Movie.1080p.WEB-DL.AAC2.0.x264", plugin.AudioCodecAAC},
		{"FLAC", "Movie.1080p.BluRay.FLAC.x264", plugin.AudioCodecFLAC},
		{"PCM", "Movie.1080p.BluRay.PCM.x264", plugin.AudioCodecPCM},
		{"LPCM", "Movie.1080p.BluRay.LPCM.x264", plugin.AudioCodecPCM},
		{"MP3", "Movie.DVDRip.MP3.XviD", plugin.AudioCodecMP3},
		{"Opus", "Movie.1080p.WEB-DL.Opus.AV1", plugin.AudioCodecOpus},
		{"no audio", "Movie.1080p.BluRay.x264", plugin.AudioCodecUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.AudioCodec != tc.want {
				t.Errorf("AudioCodec: got %q, want %q", got.AudioCodec, tc.want)
			}
		})
	}
}

func TestParseAudioChannels_AllVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  plugin.AudioChannels
	}{
		{"7.1", "Movie.2160p.REMUX.TrueHD.7.1.HEVC", plugin.AudioChannels71},
		{"8CH", "Movie.1080p.BluRay.DTS.8CH.x264", plugin.AudioChannels71},
		{"5.1", "Movie.1080p.BluRay.DTS.5.1.x264", plugin.AudioChannels51},
		{"6CH", "Movie.1080p.BluRay.AAC.6CH.x264", plugin.AudioChannels51},
		{"2.0", "Movie.1080p.BluRay.LPCM.2.0.x264", plugin.AudioChannels20},
		{"Stereo", "Movie.720p.WEB-DL.AAC.Stereo.x264", plugin.AudioChannels20},
		{"2CH", "Movie.720p.WEB-DL.AAC.2CH.x264", plugin.AudioChannels20},
		{"1.0", "Movie.DVDRip.AC3.1.0.XviD", plugin.AudioChannels10},
		{"Mono", "Movie.DVDRip.AC3.Mono.XviD", plugin.AudioChannels10},
		{"1CH", "Movie.DVDRip.AC3.1CH.XviD", plugin.AudioChannels10},
		{"no channels", "Movie.1080p.BluRay.x264", plugin.AudioChannelsUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.AudioChannels != tc.want {
				t.Errorf("AudioChannels: got %q, want %q", got.AudioChannels, tc.want)
			}
		})
	}
}

func TestParseAudioChannels_YearFalsePositives(t *testing.T) {
	t.Parallel()
	// The [^\d] anchor prevents years from triggering channel detection.
	tests := []struct {
		name  string
		input string
		want  plugin.AudioChannels
	}{
		// "2015" ends in digit, space, then "1080p" — 5.1 should NOT be detected from "15 1".
		{"2015 not 5.1", "Movie.2015.1080p.BluRay.x264", plugin.AudioChannelsUnknown},
		// "2010" — no 1.0 from "10".
		{"2010 not 1.0", "Movie.2010.1080p.BluRay.x264", plugin.AudioChannelsUnknown},
		// "2020" — no 2.0 from "20".
		{"2020 not 2.0", "Movie.2020.1080p.BluRay.x264", plugin.AudioChannelsUnknown},
		// DD5.1 should still work (letter before 5).
		{"DD5.1 works", "Movie.2024.1080p.WEB-DL.DD5.1.H.264", plugin.AudioChannels51},
		// AAC2.0 should still work (letter before 2).
		{"AAC2.0 works", "Movie.2024.1080p.WEB-DL.AAC2.0.x264", plugin.AudioChannels20},
		// DDP5.1 should work.
		{"DDP5.1 works", "Movie.2024.1080p.WEB-DL.DDP5.1.H.265", plugin.AudioChannels51},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.AudioChannels != tc.want {
				t.Errorf("AudioChannels: got %q, want %q", got.AudioChannels, tc.want)
			}
		})
	}
}
