package parser

import "testing"

func TestParseEdition_AllRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// ── Multi-word editions (must match before bare fallbacks) ────────
		{"Directors Cut", "Movie.1982.Directors.Cut.1080p.BluRay", "Director's Cut"},
		{"Director's Cut apostrophe", "Movie.1982.Director's.Cut.1080p.BluRay", "Director's Cut"},
		{"Directors Edition", "Movie.1979.Directors.Edition.1080p.BluRay", "Director's Cut"},
		{"Extended Cut", "Movie.2005.Extended.Cut.1080p.BluRay", "Extended"},
		{"Extended Edition", "Movie.2001.Extended.Edition.2160p.BluRay", "Extended"},
		{"Extended Version", "Movie.1986.Extended.Version.1080p.BluRay", "Extended"},
		{"Theatrical Cut", "Movie.2001.Theatrical.Cut.1080p.BluRay", "Theatrical"},
		{"Theatrical Edition", "Movie.2020.Theatrical.Edition.1080p.BluRay", "Theatrical"},
		{"Unrated", "Movie.2003.Unrated.1080p.BluRay", "Unrated"},
		{"Unrated Cut", "Movie.2007.Unrated.Cut.1080p.BluRay", "Unrated"},
		{"Uncensored", "Movie.2020.Uncensored.1080p.WEB-DL", "Unrated"},
		{"Ultimate Cut", "Movie.2009.Ultimate.Cut.1080p.BluRay", "Ultimate"},
		{"Ultimate Edition", "Movie.2016.Ultimate.Edition.2160p.BluRay", "Ultimate"},
		{"Special Edition", "Movie.1986.Special.Edition.1080p.BluRay", "Special Edition"},
		{"Criterion", "Movie.1979.Criterion.1080p.BluRay", "Criterion"},
		{"Criterion Collection", "Movie.1954.Criterion.Collection.1080p.BluRay", "Criterion"},
		{"IMAX", "Movie.2021.IMAX.2160p.WEB-DL", "IMAX"},
		{"IMAX Edition", "Movie.2005.IMAX.Edition.2160p.WEB-DL", "IMAX"},
		{"Final Cut", "Movie.1982.Final.Cut.2160p.BluRay", "Final Cut"},
		{"Open Matte", "Movie.1980.Open.Matte.1080p.BluRay", "Open Matte"},
		{"Rogue Cut", "Movie.2014.Rogue.Cut.1080p.BluRay", "Rogue Cut"},
		{"Black and Chrome", "Movie.2015.Black.and.Chrome.1080p.BluRay", "Black and Chrome"},
		{"Remastered", "Movie.1975.Remastered.1080p.BluRay", "Remastered"},
		{"4K Remastered", "Movie.1975.4K.Remastered.2160p.BluRay", "Remastered"},
		{"Digitally Remastered", "Movie.1990.Digitally.Remastered.1080p.BluRay", "Remastered"},
		{"Anniversary", "Movie.1982.Anniversary.Edition.1080p.BluRay", "Anniversary"},
		{"25th Anniversary", "Movie.1982.25th.Anniversary.1080p.BluRay", "Anniversary"},
		{"40th Anniversary Edition", "Movie.1979.40th.Anniversary.Edition.2160p.BluRay", "Anniversary"},

		// ── Single-word fallbacks ────────────────────────────────────────
		{"bare Extended", "Movie.2001.Extended.2160p.BluRay", "Extended"},
		{"bare Theatrical", "Movie.2021.Theatrical.2160p.WEB-DL", "Theatrical"},
		{"Redux", "Movie.1979.Redux.1080p.BluRay", "Redux"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != tc.want {
				t.Errorf("Edition: got %q, want %q", got.Edition, tc.want)
			}
		})
	}
}

func TestParseEdition_FalsePositives(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		// Abbreviations that should NOT match.
		{"DC not Directors Cut", "Movie.2020.DC.1080p.BluRay.x264"},
		{"DC in title", "DC.League.of.Super-Pets.2022.1080p.BluRay"},
		{"SE not Special Edition", "Movie.2020.SE.1080p.BluRay"},
		{"CC not Criterion", "Movie.2020.CC.1080p.BluRay"},
		// Standard release with no edition.
		{"no edition standard", "Movie.2020.1080p.BluRay.x264"},
		{"no edition 4K", "Movie.2021.2160p.BluRay.x265.HDR"},
		// Remaster without -ed suffix.
		{"Remaster no ed", "Movie.1990.Remaster.1080p.BluRay"},
		// Dir abbreviation.
		{"Dir not Directors", "Movie.2020.Dir.Cut.1080p.BluRay"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != "" {
				t.Errorf("Edition: got %q, want empty (false positive)", got.Edition)
			}
		})
	}
}

func TestParseEdition_SeparatorVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"dots", "Movie.1982.Directors.Cut.1080p", "Director's Cut"},
		{"underscores", "Movie_1982_Directors_Cut_1080p", "Director's Cut"},
		{"spaces", "Movie 1982 Directors Cut 1080p", "Director's Cut"},
		{"mixed", "Movie.1982_Directors-Cut.1080p", "Director's Cut"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != tc.want {
				t.Errorf("Edition: got %q, want %q", got.Edition, tc.want)
			}
		})
	}
}

func TestParseEdition_PositionVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"after quality", "Movie.2020.2160p.Extended.BluRay.x265", "Extended"},
		{"before year", "Movie.Redux.1979.1080p.BluRay", "Redux"},
		{"between year and quality", "Movie.2020.Extended.1080p.BluRay", "Extended"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != tc.want {
				t.Errorf("Edition: got %q, want %q", got.Edition, tc.want)
			}
		})
	}
}
