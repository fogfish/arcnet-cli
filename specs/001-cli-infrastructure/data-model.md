# Phase 1 Data Model: CLI Development Infrastructure Bootstrap

This feature is infrastructure/tooling, not a data-bearing feature — the spec's Key Entities section was intentionally omitted (no persisted domain data exists). The "entities" below are **configuration artifacts** the feature creates, documented here because later phases (tasks, implementation) need a shared vocabulary for them.

## Root Command (`cmd/arc`)

- **Fields**: `Use` (`arc`), `Short`, `Long`, `Version` (injected at build time), no `RunE` business logic beyond printing help/usage.
- **Relationships**: None yet — the sole node in the (currently empty) Cobra command tree. Future features attach subcommands here.
- **Validation rules**: Must satisfy CLIG — `--help` on invocation with no args or unrecognized args exits non-zero with usage; `--version` exits 0 with a version string.

## CI Workflow (`.github/workflows/*.yml`)

- **Fields**: `name`, trigger (`on.pull_request` or `on.push.branches`), `jobs[].steps[]`, required Go version pin.
- **Instances**: `check-test.yml` (`test`), `check-code.yml` (`check`), `build.yml` (`build`).
- **Relationships**: `check-test` and `check-code` gate merges of a pull request; `build` runs after merge to `main` and produces the release tag consumed by GoReleaser.
- **Validation rules**: Go version must be an explicit pinned string, never `latest`/floating.

## Release Configuration (`.goreleaser.yaml`)

- **Fields**: `builds[]` (goos/goarch matrix, `CGO_ENABLED`), `archives[]`, `checksum`, `changelog.filters.exclude[]`, `brews[]`.
- **Relationships**: Invoked by the `build` workflow via `goreleaser/goreleaser-action`, consuming the version tag that workflow just pushed.
- **Validation rules**: Changelog exclusion patterns must match FR-010 (`^docs:`, `^test:`). Build matrix must exclude unsupported `windows/arm64` per the reference project.

## Version Tag

- **Fields**: SemVer string (`vMAJOR.MINOR.PATCH`), derived from `reecetech/version-increment` given the prior tag and the triggering commit message's increment marker.
- **Relationships**: Produced by `build.yml`, consumed by `.goreleaser.yaml`'s release step.
- **Validation rules**: Must strictly increment from the previous tag; major/minor bump requires an explicit `[X.y.z]`/`[x.Y.z]` marker in the triggering commit message (default: patch).

## Release Artifact

- **Fields**: per-platform binary archive, `checksums.txt`, changelog body, (optionally) Homebrew formula.
- **Relationships**: Published to GitHub Releases against the Version Tag; the Homebrew formula references the same release's checksums.
- **Validation rules**: Changelog body must exclude non-user-facing commit categories (FR-010); formula smoke test must invoke `arc --version` (constitution Principle XIII).
