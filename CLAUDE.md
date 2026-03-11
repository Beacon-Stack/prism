# Luminarr — Claude Code Rules

## Releases

**Before creating any GitHub release**, you MUST query GitHub for the current latest release:

```sh
gh release list --repo luminarr/luminarr --limit 5
```

Use the actual latest version from GitHub to determine the next version number. **Never rely on local `git tag`** — local tags can be out of sync with GitHub.

## GitHub

All `gh` commands MUST target `luminarr/luminarr`:

```sh
gh <command> --repo luminarr/luminarr
```

## Code Quality

- Run `make check` before every push (golangci-lint + tsc --noEmit).
- One logical unit per commit.
- Frontend tests: `cd web/ui && npm test` must pass before pushing frontend changes.
