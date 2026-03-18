package parser

import "testing"

// FuzzParse ensures Parse never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	// Seed corpus with representative inputs.
	seeds := []string{
		"",
		"Movie",
		"The.Dark.Knight.2008.1080p.BluRay.x264-GROUP",
		"Movie.2024.2160p.BluRay.REMUX.HEVC.DTS-HD.MA.7.1.DoVi-FraMeSToR",
		"Movie.2024.1080p.WEB-DL.DDP5.1.Atmos.H.265-FLUX",
		"THE_DARK_KNIGHT_2008_1080P_BLURAY",
		"Movie.2024.Extended.Cut.1080p.BluRay.x264-GROUP",
		"Movie.2024.1080p.BluRay.x264.[D-Z0N3]",
		"/path/to/Movie.2024.1080p.BluRay.x264-GROUP.mkv",
		"Movie.2024.French.German.1080p.BluRay.PROPER.x264-GROUP",
		"Movie.2024.1080p.3D.SBS.BluRay.Hybrid.INTERNAL.x265-GRP",
		"WALL-E.2008.1080p.BluRay",
		"1917.2019.1080p.WEBRip",
		"2001.A.Space.Odyssey.1968.1080p.BluRay",
		"Movie.2024.CAM.x264-GROUP",
		"Movie.2024.RAW-HD-GROUP",
		"Movie.2024.BR-DISK-GROUP",
		"Movie.2024.1080p.WEB-DL.DD5.1.H.264-NTb.mp4",
		// Adversarial inputs.
		"----",
		"...",
		"[[[",
		".mkv",
		"Movie-",
		"Movie.2024.1080p.BluRay.x264-",
		string(make([]byte, 500)), // long null-ish string
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Parse must never panic.
		p := Parse(input)

		// Basic invariants.
		if p.Year < 0 {
			t.Errorf("Year is negative: %d", p.Year)
		}
		if p.Revision.Version < 1 {
			t.Errorf("Revision.Version < 1: %d", p.Revision.Version)
		}
		// Quality() must not panic.
		_ = p.Quality()
	})
}

// FuzzParseReleaseGroup ensures release group extraction never panics.
func FuzzParseReleaseGroup(f *testing.F) {
	seeds := []string{
		"",
		"Movie-GROUP",
		"Movie.WEB-DL.DTS-HD.MA-GROUP",
		"Movie.[D-Z0N3]",
		"Movie.(GROUP)",
		"Movie-",
		"----",
		"Movie.mkv",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = parseReleaseGroup(input)
	})
}
