# Implementation Plan: CLI/MCP "Type" Terminology Consistency

**Branch**: `015-predicate-node-shape-cli` | **Date**: 2026-07-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/015-predicate-node-shape-cli/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Rename the remaining "kind" vocabulary to "type" across `arc grep`, `arc subgraph`, `arc apply`, and `arc
serve` — the last surfaces in the codebase that still say "kind" even though the node-shape migration (specs
010/012/013) already made `@type`/`Node.Type` the sole concept everywhere else. Concretely: `optsFilter`'s
shared `--kind` flag (used by both `grep.go` and `subgraph.go`) becomes `--type`, with `internal/core.Filter`'s
`Kinds []string` field renamed to `Types []string` (and `matchKinds` to `matchTypes`) to keep the Go
identifier consistent with the flag it backs; `arc apply`'s unrecognized-type warning string drops "kind" for
"type"; `arc serve`'s `node_grep` MCP tool's `mcpFilter.Kind` wire field becomes `Type`/`"type"` and its result
table's "kind" column header becomes "type"; help text, usage examples, and the `--stubs` flag description are
updated to match. No filtering/matching logic changes (spec FR-009) — `matchTypes`'s body is `matchKinds`'s
body unchanged, only renamed. This is a pre-1.0 breaking rename with no deprecation alias (spec FR-010,
Assumptions), consistent with how specs 010/012/013 already treated the node-shape `--json` schema break.

## Technical Context

**Language/Version**: Go 1.26.5 (`go.mod`)

**Primary Dependencies**: No new dependency. Touches only already-imported `github.com/spf13/cobra` (flag
definitions in `cmd/arc/graph/grep.go`/`subgraph.go`/`serve.go`) and stdlib; `internal/core.Filter`'s matching
logic (`internal/core/filter.go`) is renamed in place, not rewritten.

**Storage**: N/A — no new filesystem, git, or external I/O; reuses the existing `fsys.Store`/`fsys.Mounter`
ports already wired into `grep`/`subgraph`/`apply`/`serve`, untouched by this feature.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). Existing
E2E/unit tests that assert on `--kind`, `Filter.Kinds`, the apply warning string, or the MCP `kind` field are
updated to the new names/wording; no new test *behavior* is introduced beyond that (this is a rename, not new
functionality) except one assertion per surface confirming the old name is rejected/ignored (spec Edge Cases).

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; unaffected by this feature)

**Project Type**: Single Cobra CLI binary (constitution Principle III) — no new command, this feature only
renames existing flags/fields/labels on four already-existing subcommands (`grep`, `subgraph`, `apply`,
`serve`)

**Performance Goals**: No new performance budget — a rename has no runtime cost distinct from today's
equivalent code path.

**Constraints**: Pre-1.0 breaking rename, no backward-compatible alias for `--kind` or the MCP `kind` field
(spec FR-010, Assumptions) — old flag/field usage fails/no-ops exactly as any other unrecognized
flag/field would, not with a special deprecation message. No new CLI flags, MCP fields, or commands (spec
Assumptions, matching the user's explicit out-of-scope boundary). Underlying `--json` field names (`type`,
etc.) and `core.Node`/`core.Filter` matching semantics are already correct and MUST NOT change (spec FR-009).

**Scale/Scope**: Four existing commands (`grep`, `subgraph`, `apply`, `serve`), one shared internal type
(`core.Filter`); no change to graph scale or on-disk format.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment |
|---|---|
| I. Architecture Documentation & ADRs | PASS — no new architectural component, no new ADR needed. [ADR 002](../../adrs/002-ux-design-system.md) (UX design system) governs flag-naming convention; this feature *corrects* an existing flag to match the convention already established for every other "type" surface, it does not introduce a new convention. |
| II. DDD & Glossary | PASS — no new domain term. "Type" is already the established `core.Node`/schema vocabulary (specs 010/011); this feature retires the last "kind" synonym rather than introducing new terminology. |
| III. Hexagonal Architecture | PASS — changes are confined to `cmd/arc/graph` (Cobra flag/help text) and `internal/core/filter.go` (already-domain-layer matching logic, renamed not restructured); no new port/adapter. |
| IV. Functional Style | PASS — `matchKinds`→`matchTypes` is a pure rename of an existing pure function; no new mutable state. |
| V. Code Quality & Simplicity (SOLID/YAGNI) | PASS — no new abstraction; this is strictly subtractive/renaming (one flag name, one struct field, one warning string, one wire field, one table header) with no added indirection. |
| VI. TDD | Applies — existing table-driven tests in `internal/core/filter_test.go`, `cmd/arc/graph/grep_opts_test.go` are updated for the new names before the rename lands, per existing repo convention. |
| VII. External Integration & Adapters | PASS — no new external integration. The MCP `node_grep` wire field rename is a contract change to an already-existing adapter (`cmd/arc/graph/serve.go`, [ADR 003](../../adrs/003-mcp-server-adapter.md)), not a new one. |
| VIII. E2E Acceptance Testing | Applies — one E2E assertion update per renamed surface in `cmd/arc/graph/{grep,subgraph,apply,serve}_test.go`, mirroring spec.md's acceptance scenarios; new assertions added for the "old name now rejected/ignored" edge cases (spec Edge Cases). |
| IX. CLI/CLIG | Applies directly — this feature *is* a flag rename. CLIG's own guidance (consistent, predictable flag naming) is exactly what motivates dropping the "kind"/"type" synonym pair; no new flag is added, an existing one is renamed in place (spec Assumptions: no new flags). |
| X. Terminal Output | PASS — no new output mode; `arc grep`'s per-line row and `arc subgraph`'s table already print the *value*, not a "kind:"/"type:" label, so no human-readable output shape changes beyond the MCP table's column header (which is explicitly required by spec FR-007) and help/usage text. |
| XI. Configuration | N/A — no new config surface. |
| XII. Documentation & Errors | Applies — help text (`Long`, flag descriptions, `Example` blocks) and the apply warning string are corrected as part of this feature (spec FR-003/FR-004/FR-005); this is the feature's primary deliverable, not a side effect. |
| XIII. Distribution & Release | PASS — no change to build/release pipeline. |
| XIV. Versioning, Security & Compatibility | **PARTIAL, explicitly justified** — this is a breaking change to a flag name (`--kind`→`--type`) and an MCP wire field name (`kind`→`type`), which Principle XIV's letter says MUST bump the major version and SHOULD be preceded by a deprecation warning in a prior minor release. Neither precedes this change. **Accepted here** for the same reason specs 010/012/013 accepted their own breaking `--json` schema changes without prior warning: the project is pre-1.0 (`0.1.x` release train), and no release has documented `--kind`/the MCP `kind` field as a stability-guaranteed contract. Recorded here for visibility, not silently absorbed — no other Principle XIV rule (SemVer tagging, `govulncheck`, telemetry) is affected. |

**Result**: No blocking violations. One flagged, pre-1.0-accepted trade-off (Principle XIV), consistent with
existing project precedent (specs 010/012/013's own Constitution Check entries for the same class of change).
No Complexity Tracking entry beyond documenting that precedent below.

## Project Structure

### Documentation (this feature)

```text
specs/015-predicate-node-shape-cli/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── kind-to-type-rename-contract.md
├── checklists/
│   └── requirements.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/arc/graph/
├── grep.go              # Existing — optsFilter.kind→type, --kind→--type flag, Long/Example help text (FR-001/FR-003)
├── grep_test.go          # Existing — --kind assertions → --type (FR-001), new "old --kind flag rejected" case
├── grep_opts_test.go     # Existing — TestOptsFilterBuildComposesKindTagAttr updated for f.Types
├── subgraph.go           # Existing — reuses optsFilter (same rename), --stubs help text "kind and id" → "type and id" (FR-002/FR-004)
├── subgraph_test.go      # Existing — --kind assertions → --type (FR-002)
├── serve.go              # Existing — mcpFilter.Kind→Type ("type" wire field), result table header "kind"→"type" (FR-006/FR-007)
└── serve_test.go          # Existing — MCP filter payload "kind"→"type", table-header assertion updated

internal/core/
├── filter.go             # Existing — Filter.Kinds→Types, matchKinds→matchTypes (rename only, FR-009: body unchanged)
└── filter_test.go        # Existing — Kinds: → Types: across table-driven cases

internal/app/graph/service/
├── apply.go               # Existing — unrecognized-type warning string "node kind" → "node type" (FR-005)
└── apply_test.go           # Existing — no wording assertions today; unaffected beyond compiling

cmd/arc/graph/apply_test.go # Existing — stderr NOT-contain assertions ("not a recognized node kind") updated to new wording
```

**Structure Decision**: No new command or package. This feature touches four existing `cmd/arc/graph`
command files (`grep.go`, `subgraph.go`, `serve.go`, plus `apply.go`'s existing warning-string call site in
`internal/app/graph/service/apply.go`) and one existing shared domain type (`internal/core.Filter`) — a
rename-only change confined to the same files/packages the four commands already live in, with matching
test-file updates in place.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|
| Breaking `--kind`→`--type` flag rename and MCP `kind`→`type` wire field rename (Principle XIV) with no prior deprecation warning | The old and new names describe the same single concept (a node's type); keeping both as permanent aliases would mean the CLI/MCP surface teaches two names for one thing forever, which is exactly the inconsistency this feature exists to remove (spec FR-008/FR-009) | A deprecation-warning-first rollout (keep `--kind` working with a stderr warning for one minor release, then remove it) was considered and rejected: this project is pre-1.0 with no released version that has ever documented `--kind`/MCP `kind` as a stable contract, so there is no external consumer this project owes a warning period to — the same reasoning specs 010/012/013 already applied to their own breaking `--json` schema changes (see those specs' own Constitution Check / Complexity Tracking entries) |
