package parser

import "testing"

func TestParseReleaseGroup_CompoundSuffixSkipping(t *testing.T) {
	t.Parallel()
	// Each compound suffix (DL, HD, X, Rip, DISK, R, Ray) must be skipped.
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"WEB-DL no group", "Movie.2024.1080p.WEB-DL.x264", ""},
		{"WEB-DL with group", "Movie.2024.1080p.WEB-DL.x264-NTb", "NTb"},
		{"WEB-Rip no group", "Movie.2024.1080p.WEB-Rip.x264", ""},
		{"RAW-HD no group", "Concert.2022.RAW-HD", ""},
		{"RAW-HD with group", "Concert.2022.RAW-HD-FraMeSToR", "FraMeSToR"},
		{"DTS-HD no group", "Movie.2024.1080p.BluRay.DTS-HD", ""},
		{"DTS-HD with group", "Movie.2024.1080p.BluRay.DTS-HD.MA.x264-DON", "DON"},
		{"DTS-X no group", "Movie.2024.2160p.BluRay.DTS-X", ""},
		{"DTS-X with group", "Movie.2024.2160p.BluRay.DTS-X.x265-GRP", "GRP"},
		{"BR-DISK no group", "Movie.2023.BR-DISK", ""},
		{"BR-DISK with group", "Movie.2023.BR-DISK-GROUP", "GROUP"},
		{"DVD-R no group", "Movie.2024.DVD-R", ""},
		{"DVD-R with group", "Movie.2024.DVD-R-GROUP", "GROUP"},
		{"Blu-Ray no group", "Movie.2024.1080p.Blu-Ray", ""},
		{"Blu-Ray with group", "Movie.2024.1080p.Blu-Ray.x264-YIFY", "YIFY"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseReleaseGroup_BracketVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"square brackets", "Movie.2024.1080p.BluRay.x264.[D-Z0N3]", "D-Z0N3"},
		{"parens", "Movie.2024.1080p.BluRay.x264.(GROUP)", "GROUP"},
		{"bracket with internal hyphen", "Movie.2024.1080p.[BHD-Studio]", "BHD-Studio"},
		{"bracket with dots", "Movie.2024.1080p.[Some.Group]", "Some.Group"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseReleaseGroup_FileExtensions(t *testing.T) {
	t.Parallel()
	exts := []string{"mkv", "mp4", "avi", "m4v", "ts", "wmv", "mov", "flv", "webm"}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()
			input := "Movie.2024.1080p.BluRay.x264-GROUP." + ext
			got := parseReleaseGroup(input)
			if got != "GROUP" {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", input, got, "GROUP")
			}
		})
	}
}

func TestParseReleaseGroup_NoGroup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{"no hyphen", "Movie.2024.1080p.BluRay.x264"},
		{"empty string", ""},
		{"only extension", "movie.mkv"},
		{"trailing hyphen", "Movie.2024.1080p.BluRay.x264-"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != "" {
				t.Errorf("parseReleaseGroup(%q) = %q, want empty", tc.input, got)
			}
		})
	}
}

func TestParseReleaseGroup_NonAlphanumericRejected(t *testing.T) {
	t.Parallel()
	// Candidates with non-alphanumeric chars (dots, spaces) should be rejected
	// as invalid group names.
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"dots in candidate", "Movie.2024.DTS-HD.MA.x264", ""},
		{"trailing period and space", "Movie.2024.1080p.BluRay.x264-GROUP. ", "GROUP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
