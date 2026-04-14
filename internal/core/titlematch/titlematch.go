// Package titlematch reports whether a release title plausibly belongs to
// a given movie. It guards auto-search, manual search, and RSS sync against
// indexers that return unrelated results for short or generic queries —
// for example, an indexer returning "The Firm 1993" when asked for "Big"
// (1988) will be rejected here even if the indexer's scoring would
// otherwise rank it as the best candidate.
package titlematch

import (
	"strconv"
	"strings"
	"unicode"
)

// Matches reports whether releaseTitle is a plausible match for
// (movieTitle, year). A release matches when:
//
//   - the normalized movie title appears as a word-boundary-aligned
//     substring of the normalized release title, and
//   - the release year (as a 4-digit string) also appears word-aligned,
//     when year > 0. Movies without a known year accept title-only.
//
// The year requirement is strict: a release that does not contain the
// correct year is rejected. Scene and p2p releases almost always carry
// the year in the filename, and this strictness is what catches indexers
// that return unrelated movies.
func Matches(releaseTitle, movieTitle string, year int) bool {
	normRelease := Normalize(releaseTitle)
	normMovie := Normalize(movieTitle)
	if normMovie == "" {
		return false
	}
	if !containsWordAligned(normRelease, normMovie) {
		return false
	}
	if year <= 0 {
		return true
	}
	return containsWordAligned(normRelease, strconv.Itoa(year))
}

// Normalize lowercases a string, converts common separators (dots,
// underscores, hyphens) to spaces, strips other non-alphanumeric
// characters, and collapses whitespace.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(' ')
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ':
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// containsWordAligned reports whether haystack contains needle aligned on
// space boundaries. This prevents a movie titled "it" from matching every
// release that incidentally contains the substring "it".
func containsWordAligned(haystack, needle string) bool {
	idx := 0
	for {
		pos := strings.Index(haystack[idx:], needle)
		if pos < 0 {
			return false
		}
		abs := idx + pos
		atStart := abs == 0 || haystack[abs-1] == ' '
		end := abs + len(needle)
		atEnd := end == len(haystack) || haystack[end] == ' '
		if atStart && atEnd {
			return true
		}
		idx = abs + 1
	}
}
