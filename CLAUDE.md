# Prism: notes for Claude Code

## Releases

**Before creating any GitHub release**, you MUST query GitHub for the current latest release:

```sh
gh release list --repo Beacon-Stack/prism --limit 5
```

Use the actual latest version from GitHub to determine the next version number. **Never rely on local `git tag`** — local tags can be out of sync with GitHub.

## GitHub

All `gh` commands MUST target `Beacon-Stack/prism`:

```sh
gh <command> --repo Beacon-Stack/prism
```

## Branching

**Always work on a feature branch** — never commit directly to `main`. Create a branch before starting new work:

```sh
git checkout -b feat/my-feature
```

Merge to `main` via PR or fast-forward after work is complete and tests pass.

## Code Quality

- Run `make check` before every push: lint, dead-code checks, type-check and frontend tests.
- Comments say what the code does and why, not how it got that way; history belongs in the commit message.
- `web/ui/src/shared/` is a copy of the `web-shared` repo. Change it there, not here.
- SQL lives in `internal/db/queries/sqlite/`; run `make generate` after changing it.
- The Docker image builds with the Go version in `docker/Dockerfile`. After `go get`, check the `go` line in `go.mod` didn't move past it.
- One logical unit per commit.
- Frontend tests: `cd web/ui && npm test` must pass before pushing frontend changes.
