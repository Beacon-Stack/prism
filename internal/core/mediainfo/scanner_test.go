package mediainfo_test

import (
	"testing"

	"github.com/beacon-stack/prism/internal/core/mediainfo"
)

// TestScanner_unavailable verifies that a scanner with no binary path is
// permanently disabled and does not panic.
func TestScanner_unavailable(t *testing.T) {
	s := mediainfo.New("/nonexistent/ffprobe", 0)
	if s.Available() {
		t.Fatal("expected Available()=false for non-existent binary path")
	}
}

// TestScanner_emptyPath checks that New() searches $PATH when ffprobePath is empty.
// If ffprobe is not in $PATH this should return available=false, not panic.
func TestScanner_emptyPath(t *testing.T) {
	// Just verify it doesn't panic regardless of whether ffprobe is installed.
	s := mediainfo.New("", 0)
	_ = s.Available()
}

// TestNormaliseCodec verifies the codec mapping table.
func TestNormaliseCodec(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hevc", "x265"},
		{"h265", "x265"},
		{"h264", "x264"},
		{"avc", "x264"},
		{"av1", "AV1"},
		{"av01", "AV1"},
		{"vp9", "VP9"},
		{"mpeg4", "XviD"},
		{"mpeg2video", "MPEG2"},
		{"unknown_codec", "unknown_codec"}, // passthrough
	}
	for _, tc := range cases {
		got := mediainfo.NormaliseCodec(tc.input)
		if got != tc.want {
			t.Errorf("NormaliseCodec(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestNormaliseResolution validates height-to-label mapping.
func TestNormaliseResolution(t *testing.T) {
	cases := []struct {
		height int
		want   string
	}{
		{2160, "2160p"},
		{4320, "2160p"}, // 8K: exceeds 2160 threshold
		{1080, "1080p"},
		{1088, "1080p"},
		{720, "720p"},
		{576, "SD"},
		{480, "SD"},
		{0, ""},
	}
	for _, tc := range cases {
		got := mediainfo.NormaliseResolution(tc.height)
		if got != tc.want {
			t.Errorf("NormaliseResolution(%d) = %q, want %q", tc.height, got, tc.want)
		}
	}
}

// TestDetectHDR validates HDR format detection from stream colour metadata.
func TestDetectHDR(t *testing.T) {
	cases := []struct {
		name          string
		colorTransfer string
		sideDataType  string
		want          string
	}{
		{"SDR", "bt709", "", "SDR"},
		{"HDR10", "smpte2084", "", "HDR10"},
		{"HLG", "arib-std-b67", "", "HLG"},
		{"DolbyVision_sidedata", "", "DOVI configuration record", "Dolby Vision"},
	}
	for _, tc := range cases {
		got := mediainfo.DetectHDRTest(tc.colorTransfer, tc.sideDataType)
		if got != tc.want {
			t.Errorf("[%s] DetectHDR = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestNormaliseContainer(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"matroska,webm", "mkv"},
		{"mov,mp4,m4a,3gp,3g2,mj2", "mp4"},
		{"avi", "avi"},
		{"", ""},
	}
	for _, tc := range cases {
		got := mediainfo.NormaliseContainer(tc.input)
		if got != tc.want {
			t.Errorf("NormaliseContainer(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseOutput_FullJSON(t *testing.T) {
	// Minimal ffprobe JSON with one video and one audio stream.
	input := []byte(`{
		"streams": [
			{
				"index": 0,
				"codec_type": "video",
				"codec_name": "hevc",
				"width": 3840,
				"height": 2160,
				"bits_per_raw_sample": "10",
				"color_transfer": "smpte2084",
				"color_primaries": "bt2020",
				"color_space": "bt2020nc",
				"channels": 0
			},
			{
				"index": 1,
				"codec_type": "audio",
				"codec_name": "eac3",
				"channels": 6
			}
		],
		"format": {
			"format_name": "matroska,webm",
			"duration": "7200.123",
			"bit_rate": "25000000"
		}
	}`)

	r, err := mediainfo.ParseOutputTest(input)
	if err != nil {
		t.Fatalf("parseOutput: %v", err)
	}
	if r.Container != "mkv" {
		t.Errorf("Container = %q, want mkv", r.Container)
	}
	if r.Codec != "x265" {
		t.Errorf("Codec = %q, want x265", r.Codec)
	}
	if r.Width != 3840 || r.Height != 2160 {
		t.Errorf("dimensions = %dx%d, want 3840x2160", r.Width, r.Height)
	}
	if r.Resolution != "2160p" {
		t.Errorf("Resolution = %q, want 2160p", r.Resolution)
	}
	if r.HDRFormat != "HDR10" {
		t.Errorf("HDRFormat = %q, want HDR10", r.HDRFormat)
	}
	if r.BitDepth != 10 {
		t.Errorf("BitDepth = %d, want 10", r.BitDepth)
	}
	if r.AudioCodec != "eac3" {
		t.Errorf("AudioCodec = %q, want eac3", r.AudioCodec)
	}
	if r.AudioChannels != 6 {
		t.Errorf("AudioChannels = %d, want 6", r.AudioChannels)
	}
	if r.DurationSecs < 7200 {
		t.Errorf("DurationSecs = %f, want ~7200", r.DurationSecs)
	}
	if r.VideoBitrate != 25000000 {
		t.Errorf("VideoBitrate = %d, want 25000000", r.VideoBitrate)
	}
	if r.ColorSpace != "bt2020nc" {
		t.Errorf("ColorSpace = %q, want bt2020nc", r.ColorSpace)
	}
}

func TestParseOutput_NoStreams(t *testing.T) {
	input := []byte(`{"streams":[],"format":{"format_name":"mp4","duration":"0","bit_rate":"0"}}`)
	r, err := mediainfo.ParseOutputTest(input)
	if err != nil {
		t.Fatalf("parseOutput: %v", err)
	}
	if r.Codec != "" {
		t.Errorf("Codec = %q, want empty", r.Codec)
	}
	if r.AudioCodec != "" {
		t.Errorf("AudioCodec = %q, want empty", r.AudioCodec)
	}
}

func TestParseOutput_InvalidJSON(t *testing.T) {
	_, err := mediainfo.ParseOutputTest([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestScanner_ScanUnavailable(t *testing.T) {
	s := mediainfo.New("/nonexistent/ffprobe", 0)
	_, err := s.Scan(t.Context(), "/some/file.mkv")
	if err == nil {
		t.Fatal("expected error from disabled scanner")
	}
}

func TestScanner_FFprobePath(t *testing.T) {
	s := mediainfo.New("/nonexistent/ffprobe", 0)
	if s.FFprobePath() != "" {
		t.Errorf("FFprobePath() = %q, want empty for nonexistent", s.FFprobePath())
	}
}
