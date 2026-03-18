package parser

import "github.com/luminarr/luminarr/pkg/plugin"

// ParseQuality is a convenience function that returns only the plugin.Quality
// portion of a full parse. Drop-in replacement for quality.Parse().
func ParseQuality(title string) plugin.Quality {
	return Parse(title).Quality()
}

// ParseReleaseGroup is a convenience function that returns only the release
// group. Drop-in replacement for quality.ParseReleaseGroup().
func ParseReleaseGroup(title string) string {
	return parseReleaseGroup(title)
}

// ParseTitle is a convenience function that returns only the title and year.
// Drop-in replacement for movie.ParseFilename().
func ParseTitle(name string) (title string, year int) {
	p := Parse(name)
	return p.Title, p.Year
}

// ParseEdition is a convenience function that returns only the edition name.
// Returns empty string if no edition is detected.
func ParseEdition(title string) string {
	return Parse(title).Edition
}
