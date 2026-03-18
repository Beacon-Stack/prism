package parser

import "testing"

func TestParseRevision(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		version int
		isReal  bool
	}{
		{"plain release", "Movie.2024.1080p.BluRay.x264-GRP", 1, false},
		{"PROPER", "Movie.2024.1080p.BluRay.PROPER.x264-GRP", 2, false},
		{"PROPER2", "Movie.2024.1080p.BluRay.PROPER2.x264-GRP", 3, false},
		{"REPACK", "Movie.2024.1080p.BluRay.REPACK.x264-GRP", 2, false},
		{"REPACK2", "Movie.2024.1080p.BluRay.REPACK2.x264-GRP", 3, false},
		{"RERIP", "Movie.2024.1080p.BluRay.RERIP.x264-GRP", 2, false},
		{"REAL alone", "Movie.2024.1080p.BluRay.REAL.x264-GRP", 1, true},
		{"REAL PROPER", "Movie.2024.1080p.REAL.PROPER.BluRay.x264-GRP", 2, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Revision.Version != tc.version {
				t.Errorf("Revision.Version: got %d, want %d", got.Revision.Version, tc.version)
			}
			if got.Revision.IsReal != tc.isReal {
				t.Errorf("Revision.IsReal: got %v, want %v", got.Revision.IsReal, tc.isReal)
			}
		})
	}
}

func TestParseMarkers_AllFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		check func(ParsedRelease) bool
		desc  string
	}{
		{"PROPER", "Movie.2024.1080p.BluRay.PROPER.x264", func(p ParsedRelease) bool { return p.IsProper }, "IsProper"},
		{"REPACK", "Movie.2024.1080p.BluRay.REPACK.x264", func(p ParsedRelease) bool { return p.IsRepack }, "IsRepack"},
		{"RERIP", "Movie.2024.1080p.BluRay.RERIP.x264", func(p ParsedRelease) bool { return p.IsRepack }, "IsRepack via RERIP"},
		{"HYBRID", "Movie.2024.1080p.BluRay.Hybrid.x265", func(p ParsedRelease) bool { return p.IsHybrid }, "IsHybrid"},
		{"3D", "Movie.2024.1080p.3D.BluRay", func(p ParsedRelease) bool { return p.Is3D }, "Is3D"},
		{"SBS", "Movie.2024.1080p.SBS.BluRay", func(p ParsedRelease) bool { return p.Is3D }, "Is3D via SBS"},
		{"HSBS", "Movie.2024.1080p.HSBS.BluRay", func(p ParsedRelease) bool { return p.Is3D }, "Is3D via HSBS"},
		{"HOU", "Movie.2024.1080p.HOU.BluRay", func(p ParsedRelease) bool { return p.Is3D }, "Is3D via HOU"},
		{"HC", "Movie.2024.1080p.HC.BluRay", func(p ParsedRelease) bool { return p.IsHardcodedSub }, "IsHardcodedSub via HC"},
		{"HARDCODED", "Movie.2024.1080p.HARDCODED.BluRay", func(p ParsedRelease) bool { return p.IsHardcodedSub }, "IsHardcodedSub"},
		{"HARDSUB", "Movie.2024.1080p.HARDSUB.BluRay", func(p ParsedRelease) bool { return p.IsHardcodedSub }, "IsHardcodedSub via HARDSUB"},
		{"KORSUB", "Movie.2024.1080p.KORSUB.BluRay", func(p ParsedRelease) bool { return p.IsHardcodedSub }, "IsHardcodedSub via KORSUB"},
		{"SAMPLE", "Movie.2024.1080p.BluRay.SAMPLE", func(p ParsedRelease) bool { return p.IsSample }, "IsSample"},
		{"INTERNAL", "Movie.2024.1080p.BluRay.INTERNAL.x264", func(p ParsedRelease) bool { return p.IsInternal }, "IsInternal"},
		{"LIMITED", "Movie.2024.1080p.BluRay.LIMITED.x264", func(p ParsedRelease) bool { return p.IsLimited }, "IsLimited"},
		{"SUBBED", "Movie.2024.1080p.BluRay.SUBBED.x264", func(p ParsedRelease) bool { return p.IsSubbed }, "IsSubbed"},
		{"DUBBED", "Movie.2024.1080p.BluRay.DUBBED.x264", func(p ParsedRelease) bool { return p.IsDubbed }, "IsDubbed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if !tc.check(got) {
				t.Errorf("%s should be true for %q", tc.desc, tc.input)
			}
		})
	}
}

func TestParseMarkers_PlainRelease(t *testing.T) {
	t.Parallel()
	got := Parse("Movie.2024.1080p.BluRay.x264-GRP")
	flags := []struct {
		name string
		val  bool
	}{
		{"IsProper", got.IsProper},
		{"IsRepack", got.IsRepack},
		{"IsHybrid", got.IsHybrid},
		{"Is3D", got.Is3D},
		{"IsHardcodedSub", got.IsHardcodedSub},
		{"IsSample", got.IsSample},
		{"IsInternal", got.IsInternal},
		{"IsLimited", got.IsLimited},
		{"IsSubbed", got.IsSubbed},
		{"IsDubbed", got.IsDubbed},
	}
	for _, f := range flags {
		if f.val {
			t.Errorf("plain release: %s should be false", f.name)
		}
	}
}
