# Plan: Custom Formats & Presets Frontend UI

**Status**: Draft
**Scope**: New settings page for browsing/importing built-in presets and managing custom formats
**Depends on**: Steps 1-6 of release-group-audio-presets plan (complete)

---

## Summary

Add a new Settings > Custom Formats page that lets users:
1. Browse built-in presets grouped by category and one-click import them
2. View and delete existing custom formats

The backend API, React hooks, and TypeScript types are already wired up.

---

## Architecture Decision

**New dedicated settings page at `/settings/custom-formats`.**

There is currently no Custom Formats page anywhere in the UI. Custom Formats are a peer concept to Quality Profiles and Quality Definitions — they deserve their own settings page, not a hidden sub-section.

The page serves two purposes:
1. **Browse and import built-in presets** (primary)
2. **List currently imported custom formats** (necessary to show which presets are already imported)

---

## Step 1: Types and API Hooks

### Files to Modify

**`web/ui/src/types/index.ts`**

Add types for the custom format list endpoint:

```typescript
export interface CustomFormatSpecification {
  name: string;
  implementation: string;
  negate: boolean;
  required: boolean;
  fields: Record<string, string>;
}

export interface CustomFormat {
  id: string;
  name: string;
  include_when_renaming: boolean;
  specifications: CustomFormatSpecification[];
  created_at: string;
  updated_at: string;
}
```

(`CustomFormatPreset` is already defined.)

**`web/ui/src/api/custom-formats.ts`**

Add hooks:

```typescript
// Fetch all existing custom formats
useCustomFormats()        -> GET /api/v1/custom-formats

// Delete a custom format
useDeleteCustomFormat()   -> DELETE /api/v1/custom-formats/{id}
```

Update the existing `useImportPreset()` hook to add toast notifications on success/error (currently has none, unlike every other mutation in the codebase).

---

## Step 2: Create the Page

### New File

**`web/ui/src/pages/settings/custom-formats/CustomFormatsPage.tsx`**

Single-file page component following the pattern used by `IndexerList.tsx`, `QualityProfileList.tsx`, etc.

### Page Structure

```
CustomFormatsPage
  +-- Page header: "Custom Formats" + subtitle
  +-- Section 1: "Imported Custom Formats"
  |     +-- Table: Name | Specs count | Created | Delete button
  |     +-- Empty state: "No custom formats yet. Import a preset below."
  +-- Section 2: "Built-in Presets"
        +-- Category groups (Audio, HD Bluray, HDR, Unwanted, WEB)
              +-- Grid of PresetCards
                    +-- Name, description, score badge, Import/Imported button
```

### Internal Helper Components (inline in the same file)

| Component | Purpose |
|-----------|---------|
| `ScoreBadge` | Inline badge showing default score. Green for positive, red for negative. |
| `PresetCard` | Card per preset: name, description, score, import button or "Imported" indicator |
| `CategorySection` | Groups presets under a category header using `sectionHeader` style |

### "Already Imported" Detection

Cross-reference `useCustomFormats()` data with `useCustomFormatPresets()` data by **name matching**. When a preset's `name` matches an existing custom format's `name`, show "Imported" instead of the import button.

This handles presets imported through this UI, via JSON import, or via the API.

### Import Flow

1. User clicks "Import" on a preset card
2. Button shows loading state ("Importing...") using `isPending`
3. `POST /api/v1/custom-formats/presets/{id}` fires
4. Success: toast "Preset imported", queries invalidated, card shows "Imported", format appears in top table
5. 409 Conflict: toast "A custom format with this name already exists"

### Delete Flow

Each row in the imported formats table has a delete button. Follows the confirm-then-delete pattern from `IndexerList`:
- Click Delete → shows "Delete? Yes / No" inline
- On confirm → `DELETE /api/v1/custom-formats/{id}`
- On success → query invalidated, corresponding preset returns to "Import" state

---

## Step 3: Routing and Navigation

### Files to Modify

**`web/ui/src/App.tsx`**

Add route under the settings group:
```tsx
<Route path="custom-formats" element={<RouteEB><CustomFormatsPage /></RouteEB>} />
```

**`web/ui/src/layouts/Shell.tsx`**

Add sidebar entry to `settingsNav` array, positioned after "Quality Definitions" and before "Indexers":
```typescript
{ to: "/settings/custom-formats", icon: Layers, label: "Custom Formats" }
```

Use the `Layers` icon from lucide-react.

---

## Step 4: Build and Verify

```bash
cd web/ui && npm run build    # TypeScript compiles
cd web/ui && npm test          # Existing tests pass
make check                     # golangci-lint + tsc --noEmit
```

---

## Layout and Design Details

**Page layout**: `padding: 24, maxWidth: 900` (standard settings page pattern)

**Imported formats table**:
- Standard card/table with `card` style from `lib/styles.ts`
- Columns: Name, Specs (count), Created, Actions
- Skeleton loading: 3 placeholder rows

**Presets grid**:
- `display: grid, gridTemplateColumns: repeat(auto-fill, minmax(280px, 1fr)), gap: 12`
- Naturally responsive: 3 columns → 2 → 1

**Score badge format**:
- Positive: `+1800` in success color (green)
- Negative: `-10000` in danger color (red)

**Category ordering**: Preserved from backend (alphabetical): Audio, HD Bluray, HDR, Unwanted, WEB

---

## Available Presets (15 total)

| Category | Name | Score |
|----------|------|-------|
| Audio | DD+ Atmos | +300 |
| Audio | DTS-HD MA | +400 |
| Audio | DTS:X | +450 |
| Audio | FLAC | +300 |
| Audio | TrueHD Atmos | +500 |
| HD Bluray | Tier 01 | +1800 |
| HD Bluray | Tier 02 | +1750 |
| HD Bluray | Tier 03 | +1700 |
| HDR | DV (WEBDL) | +500 |
| HDR | HDR10+ Boost | +500 |
| Unwanted | Bad Dual Groups | -10000 |
| Unwanted | x265 (HD) | -10000 |
| WEB | Tier 01 | +1700 |
| WEB | Tier 02 | +1650 |
| WEB | Tier 03 | +1600 |

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| User renames an imported CF | Name match fails, preset shows "Import". Clicking import gets 409. Toast explains. |
| User imports via JSON then views presets | Name match detects it, shows "Imported" correctly. |
| All presets already imported | All cards show "Imported". No action needed. |
| API errors loading presets/formats | Standard React Query error state. Show error message. |
| Slow import (network delay) | Button shows "Importing..." with pending state. |
