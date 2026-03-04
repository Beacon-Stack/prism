package movie

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ParsedFilename holds the result of parsing a release-style filename.
type ParsedFilename struct {
	Title string
	Year  int
}

// stopTokens are release-group markers and technical tokens that signal the
// end of a title.  They are matched case-insensitively as whole words.
var stopTokens = []string{
	// resolutions / scan type
	"2160p", "1080p", "1080i", "720p", "720i", "576p", "480p", "4k", "uhd",
	// source
	"bluray", "blu-ray", "bdrip", "brrip", "dvdrip", "dvdscr", "dvd",
	"webrip", "web-rip", "web-dl", "webdl", "web", "hdtv", "hdcam", "cam",
	"hdrip", "vhsrip", "ts", "hc", "r5",
	// video codec
	"x264", "x265", "h264", "h265", "hevc", "avc", "xvid", "divx",
	"10bit", "10-bit", "hdr", "hdr10", "dolbyvision", "dv",
	// audio codec
	"dts", "dts-hd", "dtshd", "truehd", "atmos", "ddplus", "dd", "ac3",
	"aac", "mp3", "flac", "eac3",
	// channels
	"7.1", "5.1", "2.0",
	// misc
	"repack", "proper", "extended", "theatrical", "directors", "dc",
	"unrated", "retail", "readnfo", "nfofix", "limited", "complete",
	"remux", "imax", "hybrid",
}

// yearRe matches a 4-digit year in the range 1900–2099.
var yearRe = regexp.MustCompile(`\b((?:19|20)\d{2})\b`)

// stopRe is built from stopTokens and matches any of them as whole words,
// case-insensitively.
var stopRe = func() *regexp.Regexp {
	escaped := make([]string, len(stopTokens))
	for i, t := range stopTokens {
		escaped[i] = regexp.QuoteMeta(t)
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(escaped, "|") + `)\b`)
}()

// ParseFilename extracts a clean title and year from a release-style filename.
//
// Strategy:
//  1. Strip the file extension (if any) and normalise separators (dots,
//     underscores, dashes-between-words) to spaces.
//  2. Locate the earliest position that is either:
//     a. a 4-digit year, or
//     b. a known stop-token (codec, resolution, source, …).
//  3. Everything before that position is the title.
//  4. Trim and title-case the result.
func ParseFilename(name string) ParsedFilename {
	// Strip leading path components.
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = name[idx+1:]
	}

	// Strip extension.
	if dot := strings.LastIndex(name, "."); dot > 0 {
		ext := strings.ToLower(name[dot+1:])
		videoExts := map[string]bool{
			"mkv": true, "mp4": true, "avi": true, "mov": true,
			"wmv": true, "m4v": true, "ts": true, "m2ts": true,
		}
		if videoExts[ext] {
			name = name[:dot]
		}
	}

	// Normalise separators to spaces.
	// Replace dots and underscores with spaces, but only when they are acting
	// as word separators — not within numeric tokens like "7.1" or "10-bit".
	// Simple approach: replace all dots and underscores with spaces, then
	// collapse multiple spaces.
	normalized := strings.Map(func(r rune) rune {
		if r == '.' || r == '_' {
			return ' '
		}
		return r
	}, name)
	// Collapse runs of spaces.
	normalized = strings.Join(strings.Fields(normalized), " ")

	// Find the earliest stop-token (codec, resolution, source, …).
	stopIdx := len(normalized)
	if m := stopRe.FindStringIndex(normalized); m != nil {
		stopIdx = m[0]
	}

	// Find all year matches.  We want the LAST year that appears at or before
	// the stop token — that is the release year.  This handles titles that
	// begin with a year-like number (e.g. "2001 A Space Odyssey 1968 ...").
	year := 0
	yearCutAt := stopIdx
	for _, m := range yearRe.FindAllStringIndex(normalized, -1) {
		if m[0] > stopIdx {
			break
		}
		if y, err := strconv.Atoi(normalized[m[0]:m[1]]); err == nil {
			year = y
			yearCutAt = m[0]
		}
	}

	rawTitle := strings.TrimSpace(normalized[:yearCutAt])
	if rawTitle == "" {
		// The title itself is a year-like number (e.g. "1917 2019 1080p").
		// Fall back: use everything before the first stop token as the title
		// and drop the release year guess (TMDB will still find it).
		rawTitle = strings.TrimSpace(normalized[:stopIdx])
		year = 0
	}

	// Remove a trailing dash or parenthesis that may be left over.
	rawTitle = strings.TrimRight(rawTitle, " -([{")

	title := toTitleCase(rawTitle)

	return ParsedFilename{Title: title, Year: year}
}

// toTitleCase capitalises the first letter of each word, leaving the rest as-is
// (preserves existing uppercase like "II", "IV", "NYC").
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
