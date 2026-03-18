# Plan: Comprehensive Parser Overhaul (Radarr-Inspired)

**Status**: Draft
**Scope**: New unified parser package replacing the three existing parsers
**Depends on**: Plan 1 (release group + audio + presets) should land first -- this plan subsumes and replaces those incremental parsers with a proper architecture

---

## Summary

Build a comprehensive, Radarr-inspired release name parser using a **tokenizer-first architecture**. The current approach of running flat regex scans against normalized strings works for simple cases but breaks down on complex real-world release names. A tokenizer that classifies tokens in priority order -- where each token is consumed exactly once -- eliminates cross-contamination between patterns.

The new parser lives in `internal/parser/`, returns a single rich `ParsedRelease` struct, and the old parsers become thin wrappers for backward compatibility.

---

## Architecture Decisions

### Decision 1: New package at `internal/parser/`

**Why not modify existing parsers?**
- Current parsing is split across `quality/parser.go`, `edition/parser.go`, and `movie/parser.go`, each with its own normalization pass
- A unified parser eliminates redundant normalization and ensures tokens are consumed only once
- A separate package avoids circular dependencies -- depends only on `pkg/plugin` for type constants

### Decision 2: Single `ParsedRelease` result struct

One call, one struct. Consumers that currently call three separate parsers will call one function. The struct composes all current output plus audio, language, release group, revision, and marker fields.

### Decision 3: Tokenizer-first pipeline

```
Raw input
  -> Normalize (strip path/extension, normalize separators)
  -> Tokenize (split into []Token at separator boundaries)
  -> Classify tokens in priority order (each consumed exactly once)
  -> Remaining unclassified tokens before first quality token = title
  -> Year extraction uses positional awareness from token stream
```

This is the Radarr approach. It handles edge cases far better than flat regex scanning because there's no cross-contamination between patterns.

### Decision 4: Backward-compatible migration via wrappers

`quality.Parse()`, `edition.Parse()`, and `movie.ParseFilename()` continue to work but delegate to `parser.Parse()` internally. Zero breakage in existing call sites.

---

## New Data Structures

### `internal/parser/types.go`

```go
type AudioCodec string
const (
    AudioUnknown     AudioCodec = ""
    AudioTrueHD      AudioCodec = "truehd"
    AudioTrueHDAtmos AudioCodec = "truehd_atmos"
    AudioDTSX        AudioCodec = "dts_x"
    AudioDTSHDMA     AudioCodec = "dts_hd_ma"
    AudioDTSHD       AudioCodec = "dts_hd"
    AudioDTS         AudioCodec = "dts"
    AudioEAC3Atmos   AudioCodec = "eac3_atmos"
    AudioEAC3        AudioCodec = "eac3"
    AudioAC3         AudioCodec = "ac3"
    AudioAAC         AudioCodec = "aac"
    AudioFLAC        AudioCodec = "flac"
    AudioPCM         AudioCodec = "pcm"
    AudioMP3         AudioCodec = "mp3"
    AudioOpus        AudioCodec = "opus"
)

type AudioChannels string
const (
    ChannelsUnknown AudioChannels = ""
    Channels71      AudioChannels = "7.1"
    Channels51      AudioChannels = "5.1"
    Channels20      AudioChannels = "2.0"
    Channels10      AudioChannels = "1.0"
)

type Language string
// English, French, German, Spanish, Italian, Portuguese, Russian,
// Japanese, Korean, Chinese, Hindi, Arabic, Multi, etc.

type Revision struct {
    Version int  // 1 = original, 2 = PROPER/REPACK, 3 = PROPER2, etc.
    IsReal  bool // REAL tag present
}

type ParsedRelease struct {
    // Identity
    Title        string
    Year         int
    ReleaseGroup string

    // Video quality
    Resolution plugin.Resolution
    Source     plugin.Source
    Codec      plugin.Codec
    HDR        plugin.HDRFormat
    Is10Bit    bool

    // Audio
    AudioCodec    AudioCodec
    AudioChannels AudioChannels
    HasAtmos      bool // layered flag, can coexist with TrueHD or EAC3

    // Edition
    Edition    string // canonical name, e.g. "Director's Cut"
    EditionRaw string // matched text from input

    // Languages
    Languages []Language

    // Revision
    Revision Revision

    // Markers / Flags
    IsHybrid       bool
    Is3D           bool
    IsHardcodedSub bool
    IsScene        bool
    IsSample       bool
    IsInternal     bool
    IsLimited      bool
    IsSubbed       bool
    IsDubbed       bool
    IsUncut        bool

    // Source metadata
    RawTitle    string // original input before normalization
    QualityName string // human-readable label
}

// Convenience methods for backward compatibility
func (p ParsedRelease) Quality() plugin.Quality { ... }
func (p ParsedRelease) ParsedFilename() movie.ParsedFilename { ... }
```

### Token type (internal)

```go
type Token struct {
    Text     string
    Index    int
    Consumed bool
}
```

---

## Implementation Phases

### Phase 1: Core Tokenizer and Title/Year Extraction

**Goal**: New package with tokenizer, title extraction, and year extraction that matches or exceeds `movie.ParseFilename` accuracy.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/types.go` | All new type definitions |
| `internal/parser/normalize.go` | Strip path, extension, disc noise, separator normalization, all-caps detection |
| `internal/parser/tokenizer.go` | Split normalized input into `[]Token` with position tracking |
| `internal/parser/title.go` | Title and year extraction from token stream |
| `internal/parser/parser.go` | Main `Parse(input string) ParsedRelease` entry point |
| `internal/parser/parser_test.go` | Integration tests, porting existing cases |

**Normalization pipeline** (from `normalize.go`):
1. Strip full file path, keep only filename (or release name)
2. Strip file extension (`.mkv`, `.mp4`, `.avi`, `.ts`, `.m4v`)
3. Remove disc ripper noise (`Title01`, `Chapter02`, `Disc1`, `Track3`)
4. Normalize `Pt.N` -> `Part N`
5. Replace dots and underscores with spaces
6. Detect all-caps and apply title casing if >= 60% uppercase

**Tokenizer** (from `tokenizer.go`):
1. Split on spaces (after normalization)
2. Each token gets an index and `Consumed = false`
3. Tokens are the unit of classification for all subsequent phases

**Title extraction** (from `title.go`):
1. Scan left-to-right for the year pattern `(19|20)\d{2}`
2. Everything before the year (that isn't consumed) = title tokens
3. If no year found, stop at the first quality/marker token
4. Handle multi-year titles ("Blade Runner 2049 2017") by preferring the last plausible year

**Test baseline**: Port all 20 cases from `movie/parser_test.go` plus all 9 from `library/diskscanner_test.go`. Add 20+ edge cases:
- Titles with colons: "Star Wars Episode IV A New Hope"
- Multiple years: "Blade Runner 2049 2017 1080p"
- Roman numerals: "Rocky III 1982"
- Year-only titles: "1917 2019", "2001 A Space Odyssey 1968"
- Hyphenated: "Spider-Man Into the Spider-Verse 2018"

---

### Phase 2: Video Quality Extraction

**Goal**: Replace flat regex scanning with token-based classification for resolution, source, codec, HDR. Add 10-bit detection.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/video.go` | Resolution, source, codec, HDR, 10-bit classifiers |
| `internal/parser/video_test.go` | Port 67 quality test cases, add 10-bit cases |

**Classification approach**: Each classifier takes `[]Token`, matches against pre-compiled regexes, and sets `Consumed = true` on matched tokens. Classification order:
1. Source (first, because some sources imply resolution)
2. Resolution
3. Codec
4. HDR
5. 10-bit (`10bit`, `10-bit`, `Hi10`, `Hi10P`)

This order mirrors the existing priority in `quality/parser.go` where source runs first so SD resolution can be inferred from DVD sources.

**Token-based advantage**: The pattern `WEB DL` (after dot normalization) becomes two tokens. The classifier recognizes the pair as `WEB-DL` by checking adjacent tokens. This is more robust than regex `WEB[\s._-]?DL` which can false-match across unrelated boundaries.

---

### Phase 3: Audio Extraction

**Goal**: Parse audio codec and channels from release names.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/audio.go` | Audio codec and channel classifiers |
| `internal/parser/audio_test.go` | 30+ test cases |

**Codec recognition order** (most specific first):
1. `TrueHD Atmos` / `TrueHD.Atmos` (multi-token: consumes both)
2. `TrueHD`
3. `DTS-X` / `DTSX`
4. `DTS-HD MA` / `DTS-HD.MA` / `DTS-HD Master Audio` (multi-token)
5. `DTS-HD` (bare)
6. `DTS` (bare)
7. `Atmos` (bare -- implies EAC3 Atmos when TrueHD not present)
8. `EAC3` / `DDP` / `DD+` / `DDPlus`
9. `AC3` / `DD` (bare, not followed by P/+)
10. `AAC`
11. `FLAC`
12. `PCM` / `LPCM`
13. `MP3`
14. `Opus`

**Channel recognition**: `7.1`, `5.1`, `2.0`, `1.0`, `8CH`, `6CH`, `2CH`, `Stereo`, `Mono`

**Atmos handling**: Atmos is a spatial audio layer on top of a base codec. Set `HasAtmos = true` independently. If `TrueHD Atmos` matched, codec = TrueHDAtmos and HasAtmos = true. If bare `Atmos` matched without TrueHD, codec = EAC3Atmos and HasAtmos = true.

---

### Phase 4: Edition, Language, and Release Group

**Goal**: Integrate edition detection into token stream, add language and release group parsing.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/edition.go` | Port 17 edition rules to token-based matching |
| `internal/parser/language.go` | Language tag detection via token lookup table |
| `internal/parser/group.go` | Release group extraction |
| `internal/parser/edition_test.go` | Port 30+ existing cases |
| `internal/parser/language_test.go` | 20+ cases |
| `internal/parser/group_test.go` | 15+ cases |

**Release group extraction**:
1. Work on the **raw** (un-normalized) input to preserve hyphens
2. Strip file extension
3. Find the last hyphen
4. Candidate = everything after it
5. Reject if candidate is a known compound suffix (`DL`, `HD`, `MA`, `X`, `Rip`, `DISK`)
6. Reject if candidate contains spaces (not a valid group name)
7. Strip brackets if present

**Language detection**: Lookup table from token text to `Language` constant:
```
"english"/"eng" -> English
"french"/"fre"/"fra"/"vff"/"vfq"/"truefrench" -> French
"german"/"ger"/"deu" -> German
"spanish"/"spa"/"esp" -> Spanish
"italian"/"ita" -> Italian
"multi" -> Multi
"dual" -> Dual (implies multi-audio)
"nordic"/"nor"/"swe"/"dan"/"fin" -> respective languages
"japanese"/"jpn" -> Japanese
"korean"/"kor" -> Korean
"chinese"/"chi"/"mandarin"/"cantonese" -> Chinese
"hindi"/"hin" -> Hindi
"russian"/"rus" -> Russian
"portuguese"/"por"/"ptbr" -> Portuguese
"vostfr" -> French (subtitled)
```

Languages appear after quality tokens, so they're classified after video/audio.

---

### Phase 5: Markers and Special Detection

**Goal**: PROPER/REPACK, hybrid, 3D, hardcoded subs, scene validation, sample detection, misc markers.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/markers.go` | All marker/flag classifiers |
| `internal/parser/scene.go` | Scene naming format heuristic |
| `internal/parser/markers_test.go` | 25+ cases |
| `internal/parser/scene_test.go` | 10+ cases |

**Revision detection**:
- `PROPER` -> Version 2
- `PROPER2` -> Version 3
- `REPACK` -> Version 2 (equivalent to PROPER)
- `REPACK2` -> Version 3
- `RERIP` -> Version 2
- `REAL` -> orthogonal `IsReal` flag (fixes naming error, not a re-encode)

**3D detection**: `3D`, `SBS` (side-by-side), `HOU` (half over/under), `OU` (over/under), `HSBS`, `HOU`

**Hardcoded subs**: `HC`, `HARDCODED`, `HARDSUB`, `KORSUB`

**Hybrid detection**: `HYBRID` token

**Sample detection**: `SAMPLE` token in filename, or path contains `/Sample/` or `/Samples/`

**Scene validation heuristic**: A release follows scene conventions if:
1. Original (pre-normalization) input uses dots as separators
2. No spaces in the original input
3. Exactly one hyphen (before the group name)
4. Group name is present
5. Follows `Title.Year.Quality.Source.Codec-GROUP` structure

**Other markers**: `INTERNAL`, `LIMITED`, `SUBBED`, `DUBBED`, `UNCUT`, `CENSORED`, `UNCENSORED`, `EXTENDED`, `THEATRICAL` (these overlap with editions -- editions take priority; standalone markers set boolean flags)

---

### Phase 6: Integration and Migration

**Goal**: Wire new parser into all existing call sites. Old parsers become wrappers.

**New files**:
| File | Purpose |
|------|---------|
| `internal/parser/compat.go` | Adapter functions: `ToQuality()`, `ToEdition()`, `ToParsedFilename()` |

**Files to modify**:

| File | Change |
|------|--------|
| `internal/core/quality/parser.go` | `Parse()` calls `parser.Parse()` then `result.Quality()` |
| `internal/core/edition/parser.go` | `Parse()` calls `parser.Parse()` then extracts edition |
| `internal/core/movie/parser.go` | `ParseFilename()` calls `parser.Parse()` then `result.ParsedFilename()` |
| `internal/core/library/diskscanner.go` | `ParseQualityFromPath()` and `parseFilename()` use `parser.Parse()` |
| `internal/core/indexer/service.go` | One `parser.Parse(r.Title)` call replaces separate quality + edition parsing (~lines 308-323, 406-419) |
| `internal/api/v1/parse.go` | Optionally return full `ParsedRelease` for richer API responses |
| `internal/core/customformat/matcher.go` | Add `ReleaseInfoFromParsed(parser.ParsedRelease) ReleaseInfo` adapter |

**Key integration in indexer service (before)**:
```go
r.Quality = quality.Parse(r.Title)
r.Edition = edition.Parse(r.Title).Name
```

**After**:
```go
parsed := parser.Parse(r.Title)
r.Quality = parsed.Quality()
r.Edition = parsed.Edition
r.ReleaseGroup = parsed.ReleaseGroup
// Audio, languages, revision, markers now available for CF matching
```

---

## Migration Strategy

### Step 1: Build in isolation (Phases 1-5)

Build and test the new parser in `internal/parser/` with zero modifications to existing code. All tests run against the new package only. During development, run both old and new parsers on the same inputs and compare (golden-file approach).

### Step 2: Wrapper migration (Phase 6a)

Old parser files delegate to `parser.Parse()`. Run the full existing test suite -- old tests now exercise the new parser through wrappers. Any failure = regression to fix before proceeding.

### Step 3: Direct integration (Phase 6b)

Update consumers (indexer service, library scanner, API) to call `parser.Parse()` directly and use the richer result. This is where new fields (audio, language, group) start flowing through the system.

### Step 4: Custom format adapter (Phase 6c)

Add `customformat.ReleaseInfoFromParsed()` so custom format matching can leverage all parsed fields instead of requiring manual regex workarounds.

### Backward compatibility guarantees

- `plugin.Quality` struct: unchanged
- `edition.Edition` struct: unchanged
- `movie.ParsedFilename` struct: unchanged
- `quality.BuildName()`: unchanged
- `customformat.ReleaseInfo`: gains new fields, none removed
- All existing DB columns: unchanged
- All existing test cases: must pass without modification

---

## Test Strategy

### Test data sources

1. **Existing tests (baseline)**: 67 quality, 30 edition, 20 movie parser, 9 library scanner = 126 cases ported as regression tests
2. **Radarr test fixtures**: Radarr (GPLv3) has hundreds of real-world test cases in `QualityParserFixture.cs`, `ParsingServiceFixture.cs`, `LanguageParserFixture.cs`. Translate to Go table-driven tests.
3. **Scene databases**: predb.org and similar publish release names usable as test fixtures (names only, no copyrighted content)
4. **Manual edge cases**: Construct cases for each known limitation

### Test organization

```
internal/parser/
  parser_test.go       # Full integration tests (parse and check all fields)
  video_test.go        # Video classifier unit tests
  audio_test.go        # Audio classifier unit tests
  edition_test.go      # Edition classifier unit tests
  language_test.go     # Language classifier unit tests
  group_test.go        # Release group extraction tests
  markers_test.go      # Marker/flag detection tests
  scene_test.go        # Scene naming validation tests
  testdata/
    releases.json      # Bulk regression: [{input, expected_fields}, ...]
```

### Coverage target

95%+ line coverage in `internal/parser/`. Every codec, resolution, source, HDR format, language, and marker has at least one positive and one negative test.

### Benchmark target

< 5 microseconds per `Parse()` call. The current combined quality+edition+title parsing takes ~2 microseconds. The token-based approach should be comparable since it uses pre-compiled regexes with fewer passes over the string.

---

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Tokenizer behavioral differences from flat regex | Some edge cases produce different results | Medium | Port every existing test. Run old+new in parallel during development. Only deprecate old parsers at 100% parity. |
| Performance regression from tokenization | Slower search results (parsing is per-candidate) | Low | Benchmark from day one. Pre-compiled regexes, single-pass split. Target < 5us. |
| Scene naming diversity exceeds patterns | Uncommon naming styles not handled | Medium | Extensible classifier system. Bulk regression tests from real-world data catch gaps early. |
| Custom format compatibility | `ImplReleaseTitle` regex matching must still work | Low | Raw title is preserved. `ReleaseInfo` adapter uses same `plugin.*` string constants. Mechanical mapping. |
| `TS` ambiguity (telesync vs `.ts` extension) | False quality classification | Low | Extension stripping in normalization phase, before tokenization. `TS` token only matches standalone. |
| Scope creep | Parser becomes a never-ending project | Medium | Strict phase boundaries. Each phase is independently shippable. Phase 1-2 alone provide value. |

---

## File Summary

### New files (16 source + 8 test)

```
internal/parser/
  types.go           # ParsedRelease, AudioCodec, AudioChannels, Language, Revision
  normalize.go       # Input normalization pipeline
  tokenizer.go       # []Token creation from normalized input
  title.go           # Title + year extraction
  video.go           # Resolution, source, codec, HDR, 10-bit
  audio.go           # Audio codec, channels, Atmos
  edition.go         # Edition detection (ported from core/edition)
  language.go        # Language tag detection
  group.go           # Release group extraction
  markers.go         # PROPER/REPACK, hybrid, 3D, HC subs, misc flags
  scene.go           # Scene naming validation heuristic
  compat.go          # Adapter methods for backward compatibility
  parser.go          # Main Parse() entry point, orchestrates all classifiers
  parser_test.go     # Integration tests
  video_test.go
  audio_test.go
  edition_test.go
  language_test.go
  group_test.go
  markers_test.go
  scene_test.go
  testdata/
    releases.json    # Bulk regression fixture
```

### Modified files (7)

```
internal/core/quality/parser.go      # Becomes wrapper
internal/core/edition/parser.go      # Becomes wrapper
internal/core/movie/parser.go        # Becomes wrapper
internal/core/library/diskscanner.go # Uses parser.Parse()
internal/core/indexer/service.go     # One Parse() call replaces three
internal/api/v1/parse.go             # Richer API response
internal/core/customformat/matcher.go # ReleaseInfoFromParsed() adapter
```

---

## Estimated Effort by Phase

| Phase | Description | Complexity |
|-------|-------------|------------|
| 1 | Tokenizer + Title/Year | Medium -- core architecture, sets the pattern |
| 2 | Video Quality | Low-Medium -- well-understood, porting existing regexes |
| 3 | Audio | Medium -- new capability, many codec variants |
| 4 | Edition + Language + Group | Medium -- three subsystems, language has a long tail |
| 5 | Markers + Scene | Low -- simple boolean flags |
| 6 | Integration + Migration | Medium -- many call sites, regression risk |

Phases 1-3 provide the most immediate value. Phases 4-5 can be done incrementally. Phase 6 is the final cutover.
