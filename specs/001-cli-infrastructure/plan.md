# Implementation Plan: CLI Development Infrastructure Bootstrap

**Branch**: `001-cli-infrastructure` | **Date**: 2026-07-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-cli-infrastructure/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Bootstrap `arc`, an empty Cobra CLI skeleton (`cmd/arc`), plus the three mandatory GitHub Actions workflows and the GoReleaser configuration required by the project constitution, modeled directly on the reference implementation at [`fogfish/iq`](https://github.com/fogfish/iq/tree/main): a `check-test` workflow (build/test/coverage on PR), a `check-code` workflow (`staticcheck` on PR), and a `build` workflow (build/test on push to `main`, automatic SemVer tag, GoReleaser release) — with a constitution-mandated `govulncheck` gate added before release since it is absent from the reference example but binding here.

## Technical Context

**Language/Version**: Go 1.26, pinned in `go.mod` and in every `actions/setup-go` CI step (research.md: Go toolchain version)

**Primary Dependencies**: `github.com/spf13/cobra` for the root command. `github.com/charmbracelet/lipgloss` intentionally deferred — no styled output exists yet (research.md: terminal styling dependency)

**Storage**: N/A — no persisted application data in this feature

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (`it.Then(t).Should(...)`); one colocated E2E test `cmd/arc/root_test.go` using a shared `sut()` helper invoking the root command's `RunE` directly (constitution Principles VI, VIII)

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — matches `.goreleaser.yaml` build matrix

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` (constitution Principle III)

**Performance Goals**: N/A — infrastructure/tooling feature, no runtime performance target beyond "builds and runs"

**Constraints**: CI checks and release pipeline MUST require no live credentials/cloud access beyond what CI provisions (spec FR-013); Go version MUST be pinned, never floating (spec FR-011); `goreleaser/goreleaser-action`'s `version:` input MUST be pinned to a GoReleaser major version compatible with `.goreleaser.yaml`'s declared `version:` config schema — the action's own default resolution MUST NOT be relied upon (BUG-001)

**Scale/Scope**: One root command, zero subcommands, three GitHub Actions workflows, one `.goreleaser.yaml`

**Bugfix**: 2026-07-01 — BUG-001 Updated from bugfix patch (added the GoReleaser Action/config schema version-pinning constraint; the `build` workflow's release step was failing on every push to `main`)

**Bugfix**: 2026-07-01 — BUG-002 Updated from bugfix patch: third-party Actions pinned in `.github/workflows/*` (`actions/checkout`, `actions/setup-go`, `shogo82148/actions-goveralls`, `dominikh/staticcheck-action`, `reecetech/version-increment`, `goreleaser/goreleaser-action`) SHOULD be periodically checked against GitHub's currently supported Actions runtime (Node version); a runner deprecation warning without a job failure is non-blocking and does not require an immediate fix if no newer, runtime-current version is yet published upstream

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| III — Hexagonal Architecture | Yes | PASS — only `cmd/arc` exists (no domain logic yet to misplace); ADR 001 explicitly permits this for small/bootstrap tools |
| VI — TDD | Yes | PASS — E2E test written before/alongside the stub `RunE`, using `it/v2` exclusively |
| VII — Ports & Adapters | No | N/A — no external system integration in this feature |
| VIII — E2E Acceptance Testing | Yes | PASS — colocated `cmd/arc/root_test.go`, `sut()` pattern, 1:1 with spec scenarios |
| IX — CLIG/Cobra | Yes | PASS — Cobra is the sole framework; root command supports `--help`/`--version` |
| X — Styled Output (lipgloss) | Yes, conditionally | PASS — no styled output produced yet, so no lipgloss usage to violate; documented in research.md rather than skipped silently |
| XIII — Release Pipeline (GoReleaser + CI) | Yes | PASS — `.goreleaser.yaml` + `build`/`check-test`/`check-code` workflows, pinned Go version |
| XIV — Versioning/Security | Yes | PASS — automatic SemVer tagging, `govulncheck` gate added before release, no telemetry introduced |

No violations requiring justification — Complexity Tracking section is empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-cli-infrastructure/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md         # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/            # Phase 1 output
│   ├── cli-contract.md
│   └── ci-release-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── main.go          # package main: calls arc.Execute() / root command's Execute()
    ├── root.go          # Cobra root command: Use/Short/Long/Example, --version wiring, RunE (help-only stub)
    └── root_test.go     # E2E test(s) for --help/--version, one per spec.md Story 1 scenario, via sut() (Principle VIII)

.github/
└── workflows/
    ├── check-test.yml   # PR: go build + go test -coverprofile + Coveralls upload
    ├── check-code.yml   # PR: staticcheck
    └── build.yml        # push to main: build/test, SemVer tag, govulncheck, GoReleaser release

.goreleaser.yaml          # release build matrix, archives, checksum, changelog filters, brews tap

go.mod                    # module github.com/fogfish/arcnet-cli, go 1.26
go.sum
```

**Structure Decision**: This feature adds exactly one command package, `cmd/arc` (root command only, no subcommand packages), plus repository-root CI/release configuration. No `internal/<domain>` package is created — there is no domain logic yet (ADR 001's explicit small-tool exception applies); the first feature that adds real behavior introduces the first `internal/` package.

## Complexity Tracking

*No entries — Constitution Check has no unresolved violations.*
