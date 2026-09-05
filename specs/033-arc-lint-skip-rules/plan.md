# Implementation Plan: Lint Rule Skipping (`arc lint --skip`, `arc lint rules`)

**Branch**: `033-arc-lint-skip-rules` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/033-arc-lint-skip-rules/spec.md`

## Summary

`arc lint` gains a `--skip` flag (comma-separated rule names) that removes
those rules' violations from the report and recomputes the pass/fail
outcome accordingly, and a new sibling `arc lint rules` command that lists
every rule the tool implements with a human-readable description.

Per the user's explicit direction, the entire feature is implemented at the
`cmd/arc/lint` level: `internal/app/lint/service.Lint` is untouched and
always computes every violation; `cmd/arc/lint` post-filters the
`kernel.LintResult` it already gets back, rebuilding a consistent result via
the existing exported `kernel.NewLintResultWithForeign` constructor
([research D1/D2](./research.md)). The rule catalog — canonical identifier
plus human-readable description, the single source both `--skip`'s
validation and `arc lint rules`' listing read — is one new ordered slice,
`kernel.RuleDefinitions`, added to `internal/app/lint/kernel`
([research D3](./research.md)).

## Technical Context

**Language/Version**: Go 1.26.6 (per `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/fogfish/faults`,
`github.com/fogfish/it/v2`. **No new dependency** — this feature adds one
flag, one subcommand, and one data slice to code paths that already exist.

**Storage**: N/A for the new behavior itself — `arc lint --skip` reads
exactly what `arc lint` already reads through `internal/adapter/fsys`;
`arc lint rules` reads nothing (static in-process data).

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`. E2E tests
colocated in `cmd/arc/lint/lint_test.go` (extended) and possibly a new
colocated file, driven through `RunE` via the existing `sut()` helper, flags
set via `cmd.Flags().Set(...)` (Principles VI, VIII).

**Target Platform**: linux/darwin/windows, amd64+arm64 (windows/arm64
excluded) per `.goreleaser.yaml` — unchanged, no platform-specific code.

**Project Type**: Single Cobra CLI binary (Principle III).

**Performance Goals**: Negligible — filtering a slice already held in memory
and matching against an 18-entry catalog. No new pass over the graph.

**Constraints**: `arc lint` remains strictly read-only; `--skip` and
`arc lint rules` introduce no write path. `--skip`'s validation MUST run
before the graph is resolved (FR-008), which is an ordering change to
`RunE`, not a new I/O operation.

**Scale/Scope**: One new kernel value (`RuleDefinitions` + its type), one new
`service/errors.go` constant, one new command file (`arc lint rules`), edits
to the existing `arc lint` command file for `--skip`. ~10 acceptance
scenarios across 3 user stories plus cross-cutting edge cases.

## Constitution Check

*GATE: evaluated before Phase 0, re-evaluated after Phase 1 design.*

**ADRs read before planning** (Principle I): [ADR 001 system architecture](../../adrs/001-system-architecture.md),
[ADR 002 UX design system](../../adrs/002-ux-design-system.md). ADR 003 (MCP
server adapter) is not touched — this feature adds no MCP tool.

| Principle | Gate | Status |
|---|---|---|
| I — ADRs binding | No plan decision contradicts ADR 001/002 | **PASS** — component/kernel/service layout and the `bios.Registry` output schema are both followed as-is; the one new Cobra-nesting pattern (`lint rules`) mirrors ADR 002's existing `apply`/`apply schema-patch` precedent rather than inventing a new one |
| II — DDD & glossary | Domain terms modeled, glossary updated | **PASS with action** — `ARCHITECTURE.md` does not exist yet (pre-existing, constitution-acknowledged gap, same as spec 032). The one new term ("lint rule catalog") is staged in [data-model.md](./data-model.md), ready to land when that file is created |
| III — Hexagonal | `cmd/` holds no logic; domain imports no cobra | **PASS with note** — `cmd/arc/lint` gains a filtering/reconstruction step (D1/D2) at the user's explicit direction; it is scoped to input validation (`--skip` parsing) and reuse of an already-exported kernel constructor, never new counting logic. Recorded in Complexity Tracking below since it is a deliberate, narrow exception worth naming rather than silently allowing |
| IV — Functional style | No inline comments, ≤25-line functions, immutability | **PASS by design** — `parseSkip`, the node/violation filter, and the rules printer are each small pure functions |
| V — SOLID / YAGNI | No catch-all packages, acyclic imports, narrowest interface | **PASS** — no new package; `RuleDefinitions` lives beside the `Rule` type it describes; no new interface introduced |
| VI — TDD | Tests first, compile, fail semantically | **PASS** — task ordering in `/speckit-tasks` must place every test task before its implementation task |
| VII — Adapters | All file I/O via `internal/adapter/fsys` | **N/A change** — no new file I/O; `arc lint rules` reads no file at all |
| VIII — E2E traceability | 1:1 scenario→test, colocated, via `RunE` | **PASS** — every spec scenario maps to a named test in [quickstart.md](./quickstart.md) |
| IX — CLIG | No new flags without convention, `NoArgs`, meaningful exit codes | **PASS** — `--skip` has a long form only (no reserved shorthand collides with its first letter's convention); `arc lint rules` is `NoArgs` |
| X — Output | stdout/stderr split, `--json`, quiet/verbose | **PASS** — reuses `bios.Registry`/`bios.ResolveMode` exactly; no change to `internal/bios` itself |
| XI — Config | XDG, precedence, secrets | **N/A** — no configuration, no secrets |
| XII — Docs & errors | `Short`/`Long`/`Example` populated, `faults` for errors | **PASS** — new error constant added to `internal/app/lint/service/errors.go` following the existing `ErrInvalidAttrFlag`/`ErrInvalidDepth` precedent (research D4), not a new pattern |
| XIII — Release | GoReleaser | **N/A** — no packaging change |
| XIV — Versioning | `--json` schema is a stable contract | **PASS with note** — `arc lint --json`'s existing schema is unchanged in shape (additive behavior only, via omission of filtered entries); `arc lint rules --json` is a *new* schema, recorded as the v1 baseline in [contracts/cli-contract.md](./contracts/cli-contract.md) |

**Gate result: PASS.** One item (Principle III) is a deliberate, user-directed
scoped exception rather than an oversight; it is named explicitly in
Complexity Tracking rather than silently allowed.

### Post-Phase-1 re-evaluation

Re-checked after `data-model.md` and `contracts/` were written. No gate
changed state. One design choice was specifically re-examined:

- **Principle III, the `cmd/`-level filter/reconstruct step** — confirmed
  narrow: the only kernel logic invoked is the already-exported
  `NewLintResultWithForeign`; `cmd/arc/lint` contributes no new rule for
  *what counts as passing*, only *which violations are visible*. If a
  second consumer of filtered lint results ever appears, this logic should
  move into `service` at that point — noted as a future trigger, not a
  present requirement (YAGNI).

## Project Structure

### Documentation (this feature)

```text
specs/033-arc-lint-skip-rules/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 — D1..D7 decisions
├── data-model.md         # Phase 1 — RuleDefinition, filtered-result rules, glossary
├── quickstart.md        # Phase 1 — scenario→test map and validation guide
├── contracts/
│   └── cli-contract.md  # command surface, exit codes, output modes
├── checklists/
│   └── requirements.md  # spec quality checklist (16/16)
└── tasks.md             # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
cmd/arc/
├── root.go                 # MODIFIED: nest lint.NewLintRulesCmd() under the lint command
└── lint/
    ├── lint.go              # MODIFIED: --skip flag, parseSkip, result filtering/reconstruction
    ├── lint_test.go         # MODIFIED: new --skip scenario tests
    ├── rules.go             # NEW: `arc lint rules` Cobra command + human printer
    └── rules_test.go        # NEW: `arc lint rules` E2E tests

internal/app/lint/
├── kernel/
│   ├── lint.go              # MODIFIED: add RuleDefinition type + RuleDefinitions catalog
│   └── lint_test.go         # MODIFIED: catalog-completeness unit test
└── service/
    └── errors.go            # MODIFIED: add ErrUnknownSkipRule (faults.Safe1[string])
```

**Structure Decision**: No new package. `--skip` and its filtering logic stay
in `cmd/arc/lint` (the command's own package), matching the user's direction
and D1/D2. The rule catalog joins `Rule`'s existing home in
`internal/app/lint/kernel`, matching the user's direction and D3. The one new
file, `cmd/arc/lint/rules.go`, exists because `arc lint rules` is a distinct
`*cobra.Command` with its own printer — folding it into `lint.go` would make
that already-sizable file mix two commands' concerns.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `cmd/arc/lint` performs result filtering and reconstruction (Principle III) | Explicit user direction for this feature; also the smallest change, since `service.Lint` has exactly one caller today and gains no reuse from a new parameter | Threading a skip set through `service.Lint` and all eight `rules_*.go` checks is a materially larger change for a feature whose entire value is presentational (which already-correct violations to show), not a new correctness rule |
| `ARCHITECTURE.md` glossary addition deferred | The file does not exist yet — a pre-existing, constitution-acknowledged gap (same handling as spec 032) | Creating a stub file to host one glossary entry would satisfy the letter of Principle II while making the file's absence harder to notice; the term is staged in `data-model.md` instead |
