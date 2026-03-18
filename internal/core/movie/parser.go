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
	"dts", "dts-hd", "dtshd", "dts-x", "dtsx", "truehd", "atmos",
	"ddplus", "ddp", "dd", "ac3", "aac", "mp3", "flac", "eac3",
	"lpcm", "pcm", "opus",
	// channels
	"7.1", "5.1", "2.0",
	// misc release flags
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

// discNoiseRe matches MakeMKV / disc-ripper chapter/title labels that appear
// at the end of a filename and carry no title information.
// Examples: Title31, Title_01, Chapter03, Disc2, Track07
var discNoiseRe = regexp.MustCompile(`(?i)\b(?:Title|Chapter|Disc|Track)\d+\b`)

// ptRe normalises "Part" abbreviations that appear in many release names.
// Matches: PT1, PT2, Pt.1, pt 2, Part.1, Part 2 (already correct form kept).
// The replacement is done before stop-token scanning so that "Part 1" is
// preserved in the title rather than being cut by a digit-based stop.
var ptRe = regexp.MustCompile(`(?i)\bPt\.?\s*(\d+)\b`)

// allCapsThreshold is the fraction of alphabetic characters that must be
// uppercase for us to consider the whole string "all-caps" and lowercase it
// before title-casing.  A value of 0.6 means: if 60 %+ of letters are
// uppercase, normalise the case.
const allCapsThreshold = 0.6

// ParseFilename extracts a clean title and year from a release-style filename.
//
// Processing pipeline:
//  1. Strip path prefix and video file extension.
//  2. Remove disc-ripper noise labels (Title31, Chapter02, …).
//  3. Normalise "Pt1" / "Pt.2" → "Part 1" / "Part 2".
//  4. Replace dots and underscores with spaces; collapse whitespace.
//  5. All-caps detection: if ≥60 % of letters are uppercase, lowercase first.
//  6. Find the last 4-digit year at or before the first stop token → release year.
//  7. Everything before that position is the title; trim and title-case.
func ParseFilename(name string) ParsedFilename {
	// 1. Strip leading path components.
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = name[idx+1:]
	}

	// Strip video file extension.
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

	// 2. Normalise separators to spaces first so that subsequent regexes
	// can rely on \b word boundaries (underscores are \w, so _PT1_ has no
	// boundary before P until the underscore is replaced with a space).
	normalized := strings.Map(func(r rune) rune {
		if r == '.' || r == '_' {
			return ' '
		}
		return r
	}, name)
	normalized = strings.Join(strings.Fields(normalized), " ")

	// 3. Remove disc-ripper noise labels (Title31, Chapter02, …).
	normalized = discNoiseRe.ReplaceAllString(normalized, " ")

	// 4. Normalise Pt/PT abbreviations → "Part N".
	normalized = ptRe.ReplaceAllStringFunc(normalized, func(m string) string {
		digits := ptRe.FindStringSubmatch(m)
		if len(digits) < 2 {
			return m
		}
		return "Part " + digits[1]
	})

	// Re-collapse spaces after substitutions.
	normalized = strings.Join(strings.Fields(normalized), " ")

	// 5. All-caps detection.
	// Count uppercase vs total alphabetic characters in the normalised string.
	// If the majority are uppercase, the source used no mixed-case separators
	// (e.g. a raw disc rip), so we lowercase everything before title-casing.
	upper, total := 0, 0
	for _, r := range normalized {
		if unicode.IsLetter(r) {
			total++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if total > 0 && float64(upper)/float64(total) >= allCapsThreshold {
		normalized = strings.ToLower(normalized)
	}

	// 6. Find earliest stop token and last year before it.
	stopIdx := len(normalized)
	if m := stopRe.FindStringIndex(normalized); m != nil {
		stopIdx = m[0]
	}

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

	// 7. Extract and clean title.
	rawTitle := strings.TrimSpace(normalized[:yearCutAt])
	if rawTitle == "" {
		// Year-titled movie (e.g. "1917 2019 1080p"): use everything before
		// the first stop token as the title and drop the year guess.
		rawTitle = strings.TrimSpace(normalized[:stopIdx])
		year = 0
	}

	// Remove trailing punctuation left over from separator stripping.
	rawTitle = strings.TrimRight(rawTitle, " -([{")

	title := toTitleCase(rawTitle)

	return ParsedFilename{Title: title, Year: year}
}

// toTitleCase capitalises the first letter of each word, leaving the rest
// as-is.  This preserves intentional uppercase like "II", "IV", "NYC" while
// still capitalising the first character of each word after lowercasing.
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
