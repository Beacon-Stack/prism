# Plan: Release Group Extraction, Audio Parsing, and TRaSH Presets

**Status**: Draft
**Scope**: Incremental enhancements to the existing parser and custom format systems
**Depends on**: Nothing (can start immediately)

---

## Summary

Three tightly-coupled enhancements that together address community feedback about quality scoring:

1. **Parse release groups** from release names so custom formats can match on them
2. **Parse audio codec/channels** from release names (DTS-HD MA, TrueHD Atmos, etc.)
3. **Ship built-in TRaSH-compatible custom format presets** for release group tiers and audio scoring

---

## Current State

- `quality.Parse()` extracts resolution, source, codec, HDR -- no audio, no release group
- `plugin.Release` struct has no `ReleaseGroup` field
- `customformat.ReleaseInfo` already has a `ReleaseGroup` field and `ImplReleaseGroup` matcher -- but it's never populated from parsing
- Custom format scoring (`MatchRelease`, `ScoreRelease`) exists and is tested but is **not wired into the autosearch pipeline**
- TRaSH JSON import (`trash.go`) is fully functional
- The project already uses `//go:embed` for migrations and static assets

---

## Step 1: Release Group Extraction

### What

Add `ParseReleaseGroup(title string) string` to `internal/core/quality/parser.go`.

### Algorithm

1. Strip file extension if present (`.mkv`, `.mp4`, `.avi`)
2. Find the last hyphen in the title
3. Extract the candidate group name (everything after the last hyphen)
4. Reject if the candidate matches a known compound-token suffix (`DL`, `HD`, `MA`, `Rip`, `DISK`, `X`) -- these appear in `WEB-DL`, `RAW-HD`, `DTS-HD`, `BR-DISK`, `DTS-X`, `WEB-Rip`
5. If rejected, walk backwards to the previous hyphen and retry (handles `DTS-HD.MA.x264-GROUP`)
6. Handle bracket-enclosed groups: `[GROUP]` at end of string
7. Return empty string if no valid group found

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/quality/parser.go` | Add `ParseReleaseGroup()` function with pre-compiled regex and exclusion set |
| `internal/core/quality/parser_test.go` | Add `TestParseReleaseGroup` with 15+ cases |
| `pkg/plugin/types.go` | Add `ReleaseGroup string` field to `Release` struct |

### Test Cases

```
Movie.2024.1080p.BluRay.x264-FraMeSToR       -> "FraMeSToR"
Movie.2024.1080p.WEB-DL.DD5.1.H.264-NTb      -> "NTb"
Movie.2024.1080p.BluRay.DTS-HD.MA.x264-DON    -> "DON"
Movie.2024.2160p.BluRay.REMUX.DTS-X.x265-GRP  -> "GRP"
Movie.2024.1080p.BluRay.x264                   -> ""
Movie.2024.RAW-HD                               -> ""
Movie.2024.1080p.BluRay.x264.[D-Z0N3]          -> "D-Z0N3"
Movie.2024.1080p.WEB-DL.x264                   -> ""  (DL is compound suffix)
```

### Risks

- **Hyphen ambiguity** with audio tokens (`DTS-HD`, `DTS-X`). Mitigated by the explicit exclusion set.
- **Groups with hyphens in their name** (e.g., `D-Z0N3`). Handled by the bracket pattern and by recognizing that internal hyphens in group names are rare in the `-GROUP` suffix convention.

---

## Step 2: Audio Codec and Channels Parsing

### What

Add audio codec and audio channels types to `pkg/plugin/types.go` and parsing to `quality/parser.go`.

### New Types (in `pkg/plugin/types.go`)

```go
type AudioCodec string
const (
    AudioUnknown   AudioCodec = ""
    AudioTrueHD    AudioCodec = "truehd"
    AudioTrueHDAtmos AudioCodec = "truehd_atmos"
    AudioDTSX      AudioCodec = "dts_x"
    AudioDTSHDMA   AudioCodec = "dts_hd_ma"
    AudioDTSHD     AudioCodec = "dts_hd"
    AudioDTS       AudioCodec = "dts"
    AudioEAC3Atmos AudioCodec = "eac3_atmos"
    AudioEAC3      AudioCodec = "eac3"
    AudioAC3       AudioCodec = "ac3"
    AudioAAC       AudioCodec = "aac"
    AudioFLAC      AudioCodec = "flac"
    AudioPCM       AudioCodec = "pcm"
    AudioMP3       AudioCodec = "mp3"
    AudioOpus      AudioCodec = "opus"
)

type AudioChannels string
const (
    AudioChannelsUnknown AudioChannels = ""
    AudioChannels71      AudioChannels = "7.1"
    AudioChannels51      AudioChannels = "5.1"
    AudioChannels20      AudioChannels = "2.0"
    AudioChannels10      AudioChannels = "1.0"
)
```

### Parsing Regexes (order matters -- most specific first)

```
TrueHD Atmos:  (?i)truehd[\s._-]?atmos
TrueHD:        (?i)\btruehd\b
DTS-X:         (?i)\bdts[\s._-]?x\b
DTS-HD MA:     (?i)\bdts[\s._-]?hd[\s._-]?(?:ma|master[\s._-]?audio)\b
DTS-HD:        (?i)\bdts[\s._-]?hd\b
DTS:           (?i)\bdts\b
Atmos (bare):  (?i)\batmos\b                     -> EAC3 Atmos (when no TrueHD)
EAC3/DD+:      (?i)(?:\bdd[p+]\b|\bddplus\b|\beac[\s.-]?3\b)
AC3/DD:        (?i)(?:\bdd\b(?![\s._-]?[p+])|\bac[\s.-]?3\b)
AAC:           (?i)\baac\b
FLAC:          (?i)\bflac\b
PCM/LPCM:      (?i)\bl?pcm\b
MP3:           (?i)\bmp3\b
Opus:          (?i)\bopus\b

Channels 7.1:  (?i)(?:\b7[\s.]1\b|\b8ch\b)
Channels 5.1:  (?i)(?:\b5[\s.]1\b|\b6ch\b)
Channels 2.0:  (?i)(?:\b2[\s.]0\b|\bstereo\b|\b2ch\b)
Channels 1.0:  (?i)(?:\b1[\s.]0\b|\bmono\b|\b1ch\b)
```

### Files to Modify

| File | Change |
|------|--------|
| `pkg/plugin/types.go` | Add `AudioCodec`, `AudioChannels` types; add both fields to `Quality` struct |
| `internal/core/quality/parser.go` | Add `parseAudioCodec()`, `parseAudioChannels()` functions; call from `Parse()` |
| `internal/core/quality/parser_test.go` | Add `wantAudioCodec`, `wantAudioChannels` to test struct; add 20+ audio test cases |

### Impact on Quality.Score()

Audio is **deliberately excluded** from the basic `Score()` method. Audio quality is better handled through custom formats where users assign their own weights. This avoids disrupting existing quality comparisons.

### Impact on Quality.BuildName()

Update `BuildName()` to optionally include audio info in the display name (e.g., "Bluray-1080p x265 HDR10 TrueHD Atmos 7.1").

### Backward Compatibility

Adding fields to `Quality` affects JSON serialization. Existing stored JSON (in grab_history, quality profiles) will be missing the new fields. Go's `json.Unmarshal` defaults missing fields to zero values (empty strings), which is correct. No DB migration needed.

---

## Step 3: Wire Parsing into the Pipeline

### What

Connect release group and audio parsing to the indexer service and custom format matcher.

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/indexer/service.go` | After existing `quality.Parse()` call (~line 310), add `r.ReleaseGroup = quality.ParseReleaseGroup(r.Title)` |
| `internal/core/customformat/matcher.go` | Add `AudioCodec`, `AudioChannels` fields to `ReleaseInfo` struct |
| `internal/core/customformat/service.go` | Add `ImplAudioCodec = "audio_codec"` and `ImplAudioChannels = "audio_channels"` constants |
| `internal/core/customformat/matcher.go` | Add `case ImplAudioCodec` and `case ImplAudioChannels` to `evalCondition()` |
| `internal/core/customformat/trash.go` | Add `"AudioCodecSpecification": ImplAudioCodec` and `"AudioChannelsSpecification": ImplAudioChannels` to mapping |
| `internal/core/customformat/trash_test.go` | Update expected mapping count |
| `internal/core/customformat/matcher_test.go` | Add test cases for audio matching |
| `internal/api/v1/customformats.go` | Add audio_codec and audio_channels to the schema endpoint |

---

## Step 4: Wire Custom Format Scoring into Autosearch

### What

The custom format infrastructure exists but is **not connected** to the autosearch pipeline. This step makes custom format scores actually influence which releases get grabbed.

### Critical Gap

`autosearch/service.go` computes `ScoreWithBreakdown` for quality dimensions but never calls `customformat.MatchRelease()` or `customformat.ScoreRelease()`. The `ScoreBreakdown.CustomFormatScore` and `ScoreBreakdown.MatchedFormats` fields exist but are always zero/empty.

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/autosearch/service.go` | Add `cfSvc *customformat.Service` dependency. In `SearchMovie()`: load all custom formats + profile CF scores, build `ReleaseInfo` from each release candidate, call `MatchRelease` + `ScoreRelease`, apply `MinCustomFormatScore` threshold, populate `breakdown.CustomFormatScore` and `breakdown.MatchedFormats` |
| `internal/registry/registry.go` or `cmd/luminarr/main.go` | Pass `customformat.Service` to autosearch constructor |

### Scoring Flow After This Step

```
Release candidate
  -> quality.Parse()          -> Resolution, Source, Codec, HDR, Audio
  -> quality.ParseReleaseGroup() -> ReleaseGroup
  -> edition.Parse()          -> Edition
  -> profile.ScoreWithBreakdown() -> Quality score (0-100)
  -> customformat.MatchRelease()  -> Matched format IDs    [NEW]
  -> customformat.ScoreRelease()  -> CF score              [NEW]
  -> edition.Bonus()          -> Edition bonus (+30)
  -> Total = quality + CF + edition bonus
  -> Check MinCustomFormatScore threshold                   [NEW]
  -> Check UpgradeUntilCFScore ceiling                      [NEW]
```

### Risk

This is the most impactful step -- it changes release selection behavior. Custom formats that exist in the DB will start influencing grabs. Users who previously imported TRaSH formats "for later" will see them take effect.

**Mitigation**: Log CF match results at debug level. If no custom formats exist, the scoring path adds zero overhead (empty format list = skip matching).

---

## Step 5: Embedded TRaSH-Compatible Presets

### What

Ship built-in custom format preset definitions that users can enable with one click.

### New Files

```
internal/core/customformat/presets/
  presets.go          # //go:embed, List(), Get() functions
  data/
    hd-bluray-tier-01.json
    hd-bluray-tier-02.json
    hd-bluray-tier-03.json
    web-tier-01.json
    web-tier-02.json
    web-tier-03.json
    uhd-bluray-tier-01.json
    uhd-bluray-tier-02.json
    uhd-bluray-tier-03.json
    bad-dual-groups.json
    x265-hd.json           # Negative score for x265 at 1080p (re-encodes)
    hdr10plus-boost.json
    dv-webdl.json
    truehd-atmos.json
    dts-hd-ma.json
    dts-x.json
    flac.json
    dd-plus-atmos.json
```

Each JSON file uses the TRaSH format:
```json
{
  "trash_id": "hd-bluray-tier-01",
  "trash_scores": {"default": 1800},
  "name": "HD Bluray Tier 01",
  "includeCustomFormatWhenRenaming": false,
  "specifications": [{
    "name": "HD Bluray Tier 01 Groups",
    "implementation": "ReleaseGroupSpecification",
    "negate": false,
    "required": true,
    "fields": {
      "value": "(?i)^(BHDStudio|hallowed|FraMeSToR)$"
    }
  }]
}
```

### Preset Metadata

Each preset has:
- `id` -- stable identifier (filename without extension)
- `name` -- display name
- `category` -- grouping: "HD Bluray", "WEB", "UHD Bluray", "Audio", "HDR", "Unwanted"
- `description` -- one-line explanation
- `default_score` -- recommended score value

### API Endpoints

```
GET  /api/v1/custom-formats/presets       -> List available presets (metadata only)
POST /api/v1/custom-formats/presets/{id}  -> Import a preset into the DB
```

Import uses the existing `trash.Import()` pathway -- no new import logic needed.

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/customformat/service.go` | Add `ListPresets()` and `ImportPreset()` methods |
| `internal/api/v1/customformats.go` | Add preset list and import endpoints |

### Risk

Presets are embedded at compile time -- updating group lists requires a new release. Acceptable because: group lists change slowly, and users can always import updated TRaSH JSON manually.

---

## Step 6: Update Stop Tokens and Frontend

### Files to Modify

| File | Change |
|------|--------|
| `internal/core/movie/parser.go` | Add missing audio tokens to the stop token list: `dts-x`, `dtsx`, `ddp`, `ddplus`, `lpcm`, `pcm`, `opus`, `eac3` |
| `web/ui/src/types/index.ts` | Add `audio_codec`, `audio_channels` to Quality type; add Preset type |
| `web/ui/src/api/customformats.ts` (or similar) | Add `listPresets()` and `importPreset()` API calls |
| Custom formats settings page | Add "Presets" section with one-click import buttons (lower priority, can be separate PR) |

---

## Implementation Order

```
Step 1: Release Group Extraction          [independent, pure functions + tests]
Step 2: Audio Codec/Channels Parsing      [independent, pure functions + tests]
Step 3: Wire into Pipeline                [depends on 1+2]
Step 4: Autosearch CF Scoring             [depends on 3, biggest behavioral change]
Step 5: TRaSH Presets                     [depends on 3, independent of 4]
Step 6: Stop Tokens + Frontend            [depends on 2+5]
```

**PR Strategy**:
- PR 1: Steps 1-3 (parsing + wiring, no behavior change for existing users)
- PR 2: Step 4 (custom format scoring in autosearch -- behavior change, needs testing)
- PR 3: Steps 5-6 (presets + frontend)

---

## Risks Summary

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hyphen ambiguity in group extraction | False positives/negatives | Explicit compound-token exclusion set, extensive tests |
| Quality JSON schema change | Stored JSON gains new empty fields | `json.Unmarshal` handles missing fields as zero values |
| CF scoring changes release selection | Users may see different grabs | Non-fatal if no CFs configured; debug logging |
| Embedded presets go stale | Group lists drift from TRaSH guides | Users can import fresh TRaSH JSON manually |
| Performance of CF matching per search | Latency on large candidate sets | Small datasets (< 50 formats), negligible overhead |
