package mediainfo_test

import (
	"testing"

	"github.com/davidfic/luminarr/internal/core/mediainfo"
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
