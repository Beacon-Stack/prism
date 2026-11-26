package stats

import (
	"encoding/json"
	"strings"

	"github.com/beacon-stack/prism/internal/core/mediainfo"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// fileQuality returns a file's quality for the statistics: what was read
// from the release name, with resolution, codec and HDR taken from the
// ffprobe scan where the name gave nothing. A file renamed to
// "Title (Year).mkv" has no release name to read, but the scan still
// knows what the video is. Where it came from (Blu-ray, web) can't be
// read off the file, so the source stays as it was.
//
// ok is false when qualityJSON can't be decoded.
func fileQuality(qualityJSON, mediainfoJSON string) (q plugin.Quality, ok bool) {
	if err := json.Unmarshal([]byte(qualityJSON), &q); err != nil {
		return q, false
	}
	if mediainfoJSON == "" {
		return q, true
	}
	var mi mediainfo.Result
	if err := json.Unmarshal([]byte(mediainfoJSON), &mi); err != nil {
		return q, true
	}

	if q.Resolution == "" || q.Resolution == plugin.ResolutionUnknown {
		// The scan says "SD" for anything under 720 lines.
		if r := strings.ToLower(mi.Resolution); r != "" {
			q.Resolution = plugin.Resolution(r)
		}
	}
	if q.Codec == "" || q.Codec == plugin.CodecUnknown {
		if c := strings.ToLower(mi.Codec); c != "" {
			q.Codec = plugin.Codec(c)
		}
	}
	if q.HDR == "" || q.HDR == plugin.HDRUnknown {
		switch mi.HDRFormat {
		case "Dolby Vision":
			q.HDR = plugin.HDRDolbyVision
		case "HDR10":
			q.HDR = plugin.HDRHDR10
		case "HLG":
			q.HDR = plugin.HDRHLG
		case "SDR":
			q.HDR = plugin.HDRNone
		}
	}
	return q, true
}
