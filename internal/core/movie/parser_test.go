package movie

import (
	"testing"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		input     string
		wantTitle string
		wantYear  int
	}{
		// --- classic dot-separated releases ---
		{
			input:     "The.Dark.Knight.2008.1080p.BluRay.x264-GROUP",
			wantTitle: "The Dark Knight",
			wantYear:  2008,
		},
		{
			input:     "Inception.2010.2160p.UHD.BluRay.x265.HEVC-GROUP",
			wantTitle: "Inception",
			wantYear:  2010,
		},
		{
			input:     "The.Shawshank.Redemption.1994.REPACK.1080p.BluRay.x264",
			wantTitle: "The Shawshank Redemption",
			wantYear:  1994,
		},
		{
			input:     "Avengers.Endgame.2019.WEB-DL.1080p.DTS-HD.MA.7.1-GROUP",
			wantTitle: "Avengers Endgame",
			wantYear:  2019,
		},
		// --- underscore separated ---
		{
			input:     "Interstellar_2014_BluRay_1080p_x264",
			wantTitle: "Interstellar",
			wantYear:  2014,
		},
		// --- year in title (tricky) ---
		{
			input:     "2001.A.Space.Odyssey.1968.1080p.BluRay",
			wantTitle: "2001 A Space Odyssey",
			wantYear:  1968,
		},
		{
			input:     "1917.2019.1080p.WEBRip",
			wantTitle: "1917",
			wantYear:  2019,
		},
		// --- no year ---
		{
			input:     "Alien.1080p.BluRay.x264",
			wantTitle: "Alien",
			wantYear:  0,
		},
		// --- with file extension ---
		{
			input:     "The.Godfather.1972.1080p.BluRay.mkv",
			wantTitle: "The Godfather",
			wantYear:  1972,
		},
		// --- WEBRip / HDTV ---
		{
			input:     "Breaking.Bad.S01E01.2008.HDTV.x264",
			wantTitle: "Breaking Bad S01E01",
			wantYear:  2008,
		},
		// --- 4K / UHD label before year ---
		{
			input:     "Dune.Part.Two.2024.2160p.UHD.BluRay.HDR.x265",
			wantTitle: "Dune Part Two",
			wantYear:  2024,
		},
		// --- mixed case in title ---
		{
			// All-caps title — toTitleCase preserves existing uppercase.
			input:     "WALL-E.2008.1080p.BluRay",
			wantTitle: "WALL-E",
			wantYear:  2008,
		},
		// --- full path ---
		{
			input:     "/media/movies/The.Matrix.1999.1080p.BluRay.x264/The.Matrix.1999.1080p.BluRay.x264.mkv",
			wantTitle: "The Matrix",
			wantYear:  1999,
		},
		// --- plain title no separators ---
		{
			input:     "Joker 2019 1080p BluRay",
			wantTitle: "Joker",
			wantYear:  2019,
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseFilename(tc.input)
			if got.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", got.Title, tc.wantTitle)
			}
			if got.Year != tc.wantYear {
				t.Errorf("Year: got %d, want %d", got.Year, tc.wantYear)
			}
		})
	}
}
