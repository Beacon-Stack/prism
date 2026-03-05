# Plan 25 — App Settings Page

## Goal
Dedicated settings page for app-level "plumbing": appearance (light/dark/system mode + full theme presets), UI preferences (tooltips, etc.), AI configuration (opt-in), and config items migrated from System.

## Scope

### Backend
- `internal/api/v1/system.go`: extend `PUT /api/v1/system/config` to accept `ai_api_key` in addition to `tmdb_api_key`. The guard changes from "must have TMDB key" to "must have at least one key". Saves `ai.api_key` to config file via `WriteConfigKey`.

### Frontend
1. `web/ui/src/theme.ts` — pure module:
   - 13 theme presets (dark + light, see below)
   - `applyTheme()` — reads localStorage, sets CSS vars on `document.documentElement`
   - `setMode(mode)` / `setPreset(mode, id)` helpers

2. `web/ui/src/pages/settings/app/AppSettingsPage.tsx`:
   - **Appearance**: Dark/Light/System mode toggle, grid of theme preset swatches
   - **UI Preferences**: Tooltips on/off toggle
   - **AI**: enable pill (from system status), Anthropic API key field, restart-required note
   - **Configuration**: TMDB key (moved from System)
   - **Backup & Restore**: (moved from System)

3. `web/ui/src/App.tsx`: add `/settings/app` route
4. `web/ui/src/layouts/Shell.tsx`:
   - Add "App Settings" nav item (icon: `Paintbrush` or `AppWindow`)
   - Call `applyTheme()` once on mount
5. `web/ui/src/api/system.ts`: `useSaveConfig` accepts `{ tmdb_api_key?, ai_api_key? }`
6. `web/ui/src/pages/settings/system/SystemPage.tsx`: remove ConfigSection and BackupSection

## Theme Presets

### Dark
| ID | Label |
|---|---|
| `luminarr` | Luminarr (default) |
| `catppuccin-mocha` | Catppuccin Mocha |
| `catppuccin-macchiato` | Catppuccin Macchiato |
| `dracula` | Dracula |
| `nord` | Nord |
| `gruvbox-dark` | Gruvbox Dark |
| `tokyo-night` | Tokyo Night |
| `one-dark` | One Dark |
| `rose-pine` | Rosé Pine |
| `kanagawa` | Kanagawa |

### Light
| ID | Label |
|---|---|
| `catppuccin-latte` | Catppuccin Latte |
| `gruvbox-light` | Gruvbox Light |
| `solarized-light` | Solarized Light |

## localStorage Keys
- `luminarr-theme-mode`: `"dark"` | `"light"` | `"system"` (default: `"dark"`)
- `luminarr-theme-dark`: preset ID for dark mode (default: `"luminarr"`)
- `luminarr-theme-light`: preset ID for light mode (default: `"catppuccin-latte"`)
- `luminarr-ui-tooltips`: `"true"` | `"false"` (default: `"true"`)

## CSS Vars Controlled by Theme
- `--color-bg-base`, `--color-bg-surface`, `--color-bg-elevated`, `--color-bg-subtle`
- `--color-border-subtle`, `--color-border-default`, `--color-border-strong`
- `--color-accent`, `--color-accent-hover`, `--color-accent-muted`, `--color-accent-fg`
- `--color-text-primary`, `--color-text-secondary`, `--color-text-muted`
- `--color-success`, `--color-warning`, `--color-danger`
