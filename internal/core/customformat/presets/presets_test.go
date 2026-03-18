package presets

import (
	"encoding/json"
	"testing"
)

func TestListReturnsAllPresets(t *testing.T) {
	all := List()
	if len(all) == 0 {
		t.Fatal("List() returned no presets")
	}

	// Verify expected count matches the number of JSON files we ship.
	const expectedCount = 15
	if len(all) != expectedCount {
		t.Errorf("List() returned %d presets, expected %d", len(all), expectedCount)
		for _, p := range all {
			t.Logf("  - %s (%s)", p.ID, p.Name)
		}
	}
}

func TestGetKnownPreset(t *testing.T) {
	p, ok := Get("hd-bluray-tier-01")
	if !ok {
		t.Fatal("Get(hd-bluray-tier-01) returned false")
	}
	if p.Name != "HD Bluray Tier 01" {
		t.Errorf("Name = %q, want %q", p.Name, "HD Bluray Tier 01")
	}
	if p.Category != "HD Bluray" {
		t.Errorf("Category = %q, want %q", p.Category, "HD Bluray")
	}
	if p.Score != 1800 {
		t.Errorf("Score = %d, want %d", p.Score, 1800)
	}
	if len(p.Data) == 0 {
		t.Error("Data is empty")
	}
}

func TestGetUnknownPreset(t *testing.T) {
	_, ok := Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestAllPresetsHaveValidJSON(t *testing.T) {
	for _, p := range List() {
		var raw map[string]any
		if err := json.Unmarshal(p.Data, &raw); err != nil {
			t.Errorf("preset %q has invalid JSON: %v", p.ID, err)
		}
		if p.Name == "" {
			t.Errorf("preset %q has empty Name", p.ID)
		}
		if p.Category == "" {
			t.Errorf("preset %q has empty Category", p.ID)
		}
		if p.Description == "" {
			t.Errorf("preset %q has empty Description", p.ID)
		}
	}
}

func TestAllPresetsHaveSpecs(t *testing.T) {
	type trashJSON struct {
		Specifications []any `json:"specifications"`
	}
	for _, p := range List() {
		var tj trashJSON
		if err := json.Unmarshal(p.Data, &tj); err != nil {
			t.Errorf("preset %q: %v", p.ID, err)
			continue
		}
		if len(tj.Specifications) == 0 {
			t.Errorf("preset %q has no specifications", p.ID)
		}
	}
}

func TestPresetsSortedByCategoryThenName(t *testing.T) {
	all := List()
	for i := 1; i < len(all); i++ {
		prev, curr := all[i-1], all[i]
		if prev.Category > curr.Category {
			t.Errorf("presets not sorted by category: %q (%s) before %q (%s)",
				prev.ID, prev.Category, curr.ID, curr.Category)
		}
		if prev.Category == curr.Category && prev.Name > curr.Name {
			t.Errorf("presets not sorted by name within category %q: %q before %q",
				prev.Category, prev.Name, curr.Name)
		}
	}
}
