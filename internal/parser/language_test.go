package parser

import "testing"

func TestParseLanguages_Detection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []Language
	}{
		{"French", "Movie.2020.French.1080p.BluRay", []Language{LangFrench}},
		{"MULTI", "Movie.2020.MULTI.1080p.BluRay", []Language{LangMulti}},
		{"German", "Movie.2020.German.1080p.BluRay", []Language{LangGerman}},
		{"Spanish", "Movie.2020.Spanish.1080p.BluRay", []Language{LangSpanish}},
		{"Italian", "Movie.2020.Italian.1080p.BluRay", []Language{LangItalian}},
		{"Russian", "Movie.2020.Russian.1080p.BluRay", []Language{LangRussian}},
		{"Japanese", "Movie.2020.Japanese.1080p.BluRay", []Language{LangJapanese}},
		{"Korean", "Movie.2020.Korean.1080p.BluRay", []Language{LangKorean}},
		{"Chinese", "Movie.2020.Chinese.1080p.BluRay", []Language{LangChinese}},
		{"Hindi", "Movie.2020.Hindi.1080p.BluRay", []Language{LangHindi}},
		{"TrueFrench", "Movie.2020.TrueFrench.1080p.BluRay", []Language{LangFrench}},
		{"ENG abbreviation", "Movie.2020.ENG.1080p.BluRay", []Language{LangEnglish}},
		{"RUS abbreviation", "Movie.2020.RUS.1080p.BluRay", []Language{LangRussian}},
		{"no language", "Movie.2020.1080p.BluRay.x264", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if len(got.Languages) != len(tc.want) {
				t.Fatalf("Languages: got %v, want %v", got.Languages, tc.want)
			}
			for i, lang := range tc.want {
				if got.Languages[i] != lang {
					t.Errorf("Languages[%d]: got %q, want %q", i, got.Languages[i], lang)
				}
			}
		})
	}
}

func TestParseLanguages_MultipleLanguages(t *testing.T) {
	t.Parallel()
	got := Parse("Movie.2020.French.German.1080p.BluRay")
	if len(got.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d: %v", len(got.Languages), got.Languages)
	}
}

func TestParseLanguages_Deduplication(t *testing.T) {
	t.Parallel()
	got := Parse("Movie.2020.French.French.1080p.BluRay")
	if len(got.Languages) != 1 {
		t.Fatalf("expected 1 language (deduplicated), got %d: %v", len(got.Languages), got.Languages)
	}
}

func TestParseLanguages_WordBoundaryProtection(t *testing.T) {
	t.Parallel()
	// These words CONTAIN language codes but should NOT trigger detection
	// because the regex uses \b word boundaries.
	tests := []struct {
		name  string
		input string
	}{
		{"NORMANDY contains NOR", "The.Normandy.Chronicles.2020.1080p.BluRay"},
		{"FINDING contains FIN", "Finding.Nemo.2003.1080p.BluRay"},
		{"SPARKS contains SPA (group)", "Movie.2020.1080p.BluRay-SPARKS"},
		{"ENGLAND contains ENG", "England.Pride.2020.1080p.BluRay"},
		{"DANGER contains DAN", "Danger.Zone.2020.1080p.BluRay"},
		{"POLICE contains POL", "Police.Academy.2020.1080p.BluRay"},
		{"TURKEY contains TUR", "Turkey.Holiday.2020.1080p.BluRay"},
		{"FRAGILE contains FRA", "Fragile.Dreams.2020.1080p.BluRay"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if len(got.Languages) > 0 {
				t.Errorf("false positive: detected %v in %q", got.Languages, tc.input)
			}
		})
	}
}
