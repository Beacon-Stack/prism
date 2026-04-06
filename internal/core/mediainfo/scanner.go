package mediainfo

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// codecMap translates ffprobe codec_name values to Prism's canonical names.
var codecMap = map[string]string{
	"hevc":       "x265",
	"h265":       "x265",
	"h264":       "x264",
	"avc":        "x264",
	"av1":        "AV1",
	"av01":       "AV1",
	"vp9":        "VP9",
	"mpeg4":      "XviD",
	"mpeg2video": "MPEG2",
}

// Scanner wraps ffprobe for video file metadata extraction.
// A Scanner with an empty ffprobePath is permanently disabled; all methods
// are safe to call on a disabled scanner.
type Scanner struct {
	ffprobePath string
	timeout     time.Duration
}

// New creates a Scanner. ffprobePath may be empty (search $PATH) or an
// absolute/relative path. If the resolved binary is not found, the scanner
// is created in disabled state (Available() returns false).
func New(ffprobePath string, timeout time.Duration) *Scanner {
	if ffprobePath == "" {
		// Search $PATH.
		if p, err := exec.LookPath("ffprobe"); err == nil {
			ffprobePath = p
		}
		// If not found, ffprobePath stays "" → disabled.
	} else {
		// Explicit path: verify it is actually resolvable.
		if p, err := exec.LookPath(ffprobePath); err == nil {
			ffprobePath = p
		} else {
			ffprobePath = "" // not found → disabled
		}
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Scanner{ffprobePath: ffprobePath, timeout: timeout}
}

// Available reports whether ffprobe was found and scanning is enabled.
func (s *Scanner) Available() bool {
	return s.ffprobePath != ""
}

// FFprobePath returns the resolved path to ffprobe, or "" if unavailable.
func (s *Scanner) FFprobePath() string {
	return s.ffprobePath
}

// Scan runs ffprobe on filePath and returns the parsed metadata.
// Returns an error if the scanner is unavailable or ffprobe fails.
func (s *Scanner) Scan(ctx context.Context, filePath string) (*Result, error) {
	if !s.Available() {
		return nil, fmt.Errorf("ffprobe not available")
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	//nolint:gosec // filePath is a trusted internal value (imported file path stored in DB)
	cmd := exec.CommandContext(ctx, s.ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		filePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	return parseOutput(out)
}

// ── ffprobe JSON structures ────────────────────────────────────────────────

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Index          int               `json:"index"`
	CodecType      string            `json:"codec_type"` // "video", "audio", "subtitle"
	CodecName      string            `json:"codec_name"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	BitDepthStr    string            `json:"bits_per_raw_sample"`
	ColorTransfer  string            `json:"color_transfer"`  // "smpte2084" → HDR10
	ColorPrimaries string            `json:"color_primaries"` // "bt2020" → wide gamut
	ColorSpace     string            `json:"color_space"`
	Channels       int               `json:"channels"`
	SideDataList   []ffprobeSideData `json:"side_data_list"`
}

type ffprobeSideData struct {
	SideDataType string `json:"side_data_type"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"` // "matroska,webm", "mov,mp4,m4a,3gp,3g2,mj2"
	Duration   string `json:"duration"`    // seconds as string
	BitRate    string `json:"bit_rate"`
}

// parseOutput converts raw ffprobe JSON into a Result.
func parseOutput(data []byte) (*Result, error) {
	var raw ffprobeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing ffprobe output: %w", err)
	}

	r := &Result{}

	// Container
	r.Container = normaliseContainer(raw.Format.FormatName)

	// Duration
	if d, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
		r.DurationSecs = d
	}

	// Format-level bitrate (fallback for video bitrate)
	if br, err := strconv.ParseInt(raw.Format.BitRate, 10, 64); err == nil {
		r.VideoBitrate = br
	}

	// Find primary video and audio streams.
	var videoStream, audioStream *ffprobeStream
	for i := range raw.Streams {
		st := &raw.Streams[i]
		switch st.CodecType {
		case "video":
			if videoStream == nil {
				videoStream = st
			}
		case "audio":
			if audioStream == nil {
				audioStream = st
			}
		}
	}

	if videoStream != nil {
		r.Codec = normaliseCodec(videoStream.CodecName)
		r.Width = videoStream.Width
		r.Height = videoStream.Height
		r.Resolution = normaliseResolution(videoStream.Height)
		r.ColorSpace = videoStream.ColorSpace
		r.HDRFormat = detectHDR(videoStream)
		if bd, err := strconv.Atoi(videoStream.BitDepthStr); err == nil && bd > 0 {
			r.BitDepth = bd
		}
	}

	if audioStream != nil {
		r.AudioCodec = audioStream.CodecName
		r.AudioChannels = audioStream.Channels
	}

	return r, nil
}

// normaliseCodec converts an ffprobe codec_name to Prism's canonical name.
func normaliseCodec(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if canon, ok := codecMap[name]; ok {
		return canon
	}
	return name
}

// normaliseResolution maps a pixel height to a display string.
func normaliseResolution(height int) string {
	switch {
	case height >= 2160:
		return "2160p"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height > 0:
		return "SD"
	default:
		return ""
	}
}

// normaliseContainer extracts the primary container name from ffprobe's
// comma-separated format_name field.
func normaliseContainer(formatName string) string {
	parts := strings.Split(formatName, ",")
	if len(parts) == 0 {
		return formatName
	}
	name := strings.TrimSpace(parts[0])
	// Rename a few verbose names to terse equivalents.
	switch name {
	case "matroska":
		return "mkv"
	case "mov":
		return "mp4"
	}
	return name
}

// detectHDR determines the HDR format from stream metadata.
func detectHDR(st *ffprobeStream) string {
	// Dolby Vision: look for RPU side data.
	for _, sd := range st.SideDataList {
		if strings.Contains(strings.ToLower(sd.SideDataType), "dovi") ||
			strings.Contains(strings.ToLower(sd.SideDataType), "dolby") {
			return "Dolby Vision"
		}
	}

	// HDR10: SMPTE ST 2084 transfer function (PQ) with BT.2020 primaries.
	if st.ColorTransfer == "smpte2084" {
		return "HDR10"
	}

	// HLG: Hybrid Log-Gamma.
	if st.ColorTransfer == "arib-std-b67" {
		return "HLG"
	}

	return "SDR"
}
