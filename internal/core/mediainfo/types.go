// Package mediainfo provides ffprobe-based video file metadata scanning.
package mediainfo

// Result holds the actual technical metadata extracted from a video file by
// ffprobe. All fields are best-effort — missing streams leave fields at zero
// values.
type Result struct {
	Container    string  `json:"container"`     // e.g. "mkv", "mp4"
	DurationSecs float64 `json:"duration_secs"` // total duration
	VideoBitrate int64   `json:"video_bitrate"` // bits/sec (from format or stream)

	// Video stream (primary)
	Codec      string `json:"codec"` // normalised: "x265", "x264", "AV1"
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Resolution string `json:"resolution"`  // normalised: "2160p", "1080p", "720p", "SD"
	ColorSpace string `json:"color_space"` // e.g. "bt2020nc", "bt709"
	HDRFormat  string `json:"hdr_format"`  // "HDR10", "Dolby Vision", "SDR"
	BitDepth   int    `json:"bit_depth"`   // 8, 10, 12

	// Audio (primary stream)
	AudioCodec    string `json:"audio_codec"`    // e.g. "eac3", "dts", "aac"
	AudioChannels int    `json:"audio_channels"` // 2, 6, 8
}
