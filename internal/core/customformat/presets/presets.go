// Package presets provides built-in TRaSH-compatible custom format definitions
// that users can import with a single click. Preset data is embedded at compile
// time using //go:embed.
package presets

import (
	"embed"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed data/*.json
var presetsFS embed.FS

// Preset describes a built-in custom format that can be imported.
type Preset struct {
	ID          string `json:"id"`            // stable identifier (filename without extension)
	Name        string `json:"name"`          // display name from the JSON
	Category    string `json:"category"`      // grouping: "HD Bluray", "WEB", "Audio", "HDR", "Unwanted"
	Description string `json:"description"`   // one-line explanation
	Score       int    `json:"default_score"` // recommended score from trash_scores.default
	Data        []byte `json:"-"`             // raw TRaSH-format JSON (omitted from API list)
}

// presetMeta defines the category and description for each preset ID.
// This is separate from the JSON so we can provide richer metadata.
var presetMeta = map[string]struct {
	Category    string
	Description string
}{
	"hd-bluray-tier-01": {"HD Bluray", "Top-tier HD Bluray release groups (FraMeSToR, BHDStudio, DON, etc.)"},
	"hd-bluray-tier-02": {"HD Bluray", "Second-tier HD Bluray release groups (decibeL, EA, iFT, etc.)"},
	"hd-bluray-tier-03": {"HD Bluray", "Third-tier HD Bluray release groups (BeyondHD, CiNEPHiLES, etc.)"},
	"web-tier-01":       {"WEB", "Top-tier WEB release groups (FLUX, HONE, NTb, CMRG, etc.)"},
	"web-tier-02":       {"WEB", "Second-tier WEB release groups (DRACULA, SMURF, TOMMY, etc.)"},
	"web-tier-03":       {"WEB", "Third-tier WEB release groups (Cakes4free, KATT, VERMIN, etc.)"},
	"x265-hd":           {"Unwanted", "Penalize x265/HEVC re-encodes at HD resolution (not 4K, not Remux)"},
	"bad-dual-groups":   {"Unwanted", "Known low-quality release groups (YIFY, RARBG, PSA, etc.)"},
	"truehd-atmos":      {"Audio", "Boost for TrueHD Atmos audio"},
	"dts-hd-ma":         {"Audio", "Boost for DTS-HD Master Audio"},
	"dts-x":             {"Audio", "Boost for DTS:X immersive audio"},
	"ddplus-atmos":      {"Audio", "Boost for DD+ Atmos (Dolby Digital Plus with Atmos)"},
	"flac":              {"Audio", "Boost for FLAC lossless audio"},
	"hdr10plus-boost":   {"HDR", "Boost for HDR10+ dynamic metadata"},
	"dv-webdl":          {"HDR", "Dolby Vision from WEBDL sources"},
}

// trashJSON is the minimal struct needed to extract name and default score.
type trashJSON struct {
	Name        string         `json:"name"`
	TrashScores map[string]int `json:"trash_scores"`
}

var presetCache []Preset

func init() {
	entries, err := presetsFS.ReadDir("data")
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := presetsFS.ReadFile(filepath.Join("data", e.Name()))
		if err != nil {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".json")

		var tj trashJSON
		if err := json.Unmarshal(data, &tj); err != nil {
			continue
		}

		meta, ok := presetMeta[id]
		if !ok {
			meta = struct {
				Category    string
				Description string
			}{"Other", tj.Name}
		}

		presetCache = append(presetCache, Preset{
			ID:          id,
			Name:        tj.Name,
			Category:    meta.Category,
			Description: meta.Description,
			Score:       tj.TrashScores["default"],
			Data:        data,
		})
	}

	sort.Slice(presetCache, func(i, j int) bool {
		if presetCache[i].Category != presetCache[j].Category {
			return presetCache[i].Category < presetCache[j].Category
		}
		return presetCache[i].Name < presetCache[j].Name
	})
}

// List returns all available presets sorted by category then name.
func List() []Preset {
	out := make([]Preset, len(presetCache))
	copy(out, presetCache)
	return out
}

// Get returns a preset by its stable ID. Returns false if not found.
func Get(id string) (Preset, bool) {
	for _, p := range presetCache {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}
