# Implementation Plan: MCP `node_match` Filter Tool

**Branch**: `028-node-match-filter` | **Date**: 2026-08-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/028-node-match-filter/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a new `node_match(filter)` MCP tool that reuses the existing `core.Filter`/`Statement`/`Matcher` primitive from `specs/027-triple-filter-model` unchanged, plus one small, additive extension to it: `Filter.MatchingFacts(node)` (new `core.Fact{Property, Value}` type), which enumerates the specific attribute/edge/type facts on a node that satisfied at least one statement — the same node-fact walk `statementSatisfiedBy` already performs, but collecting instead of short-circuiting. A new `service.Match` function (`internal/app/graph/service/match.go`) enumerates every node file exactly as `service.Grep` already does (`walkNodeFiles`, `readGrepNode`, `guardIsGraph`), keeps only nodes for which `filter.Match(node)` is true, and reports `filter.MatchingFacts(node)` for each; an empty/missing filter is rejected up front with a new `service.ErrEmptyFilter`, since node_match's whole output is evidence of *why* nodes matched, and every node vacuously matches an empty filter. `internal/app/graph/component.go` gets a thin `Match` delegator (mirroring `Grep`/`Subgraph`), and `cmd/arc/graph/serve.go` registers `node_match` as a fifth read-only MCP tool, reusing `mcpFilter`/`toCoreFilter`/`inputSchemaFor` verbatim and rendering results as a new `| id | property | value |` markdown table (`renderFactTable`, mirroring `renderMatchTable`). Per the user's explicit instruction, this feature is MCP-only: no `arc` Cobra subcommand is added in this release — that is deferred to a future release, exactly as `context_retrieve`/`schema` are already MCP-only today.

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk/mcp` (already `arc serve`'s transport, ADR 003) — no new dependency added by this feature

**Storage**: N/A for the matching mechanism itself — node data continues to be read via `internal/adapter/fsys` through the same `walkNodeFiles`/`readGrepNode` enumeration `service.Grep` already uses; no new I/O path

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit tests (table-driven `Filter.MatchingFacts` cases in `internal/core/filter_test.go`, `service.Match` cases in a new `internal/app/graph/service/match_test.go`) and colocated E2E tests extending `cmd/arc/graph/serve_test.go`'s existing `connectServeSession` helper (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature touches one domain file (`internal/core/filter.go`, additive `Fact`/`MatchingFacts`), one new service file (`internal/app/graph/service/match.go`), one new kernel file (`internal/app/graph/kernel/match.go`), one delegator addition (`internal/app/graph/component.go`), and one existing command file (`cmd/arc/graph/serve.go`, new tool registration) — no new package, no new adapter, no new Cobra command

**Performance Goals**: `Filter.MatchingFacts` is an in-memory, allocation-light value operation over an already-parsed `core.Node`, structurally identical in cost to the existing `statementSatisfiedBy` walk it reuses; `service.Match`'s per-node scan cost is dominated by file I/O and node parsing, identical to `service.Grep`'s existing per-node cost minus the regexp content scan (node_match never opens node bodies for text search, only front matter/edges already parsed by `readGrepNode`)

**Constraints**: `Filter`/`Filter.Match`/`Filter.MatchingFacts` remain pure, no-I/O value operations (existing contract, unchanged); `node_match`'s `filter` argument is REQUIRED with at least one statement — unlike `node_grep`/`subgraph_get`/`context_retrieve`'s optional `filter`, this is a deliberate, documented deviation (spec FR-005) since an empty filter's vacuous match-everything semantics would make `node_match`'s fact-evidence output meaningless; no CLI/Cobra surface for this feature in this release (explicit user instruction — deferred to a future release, same posture as `context_retrieve`/`schema`)

**Scale/Scope**: One additive domain method (`internal/core/filter.go`: `Fact` type, `Filter.MatchingFacts`), one new service file (`internal/app/graph/service/match.go`: `Match` function, `ErrEmptyFilter` in `errors.go`), one new kernel file (`internal/app/graph/kernel/match.go`: `MatchEntry`/`MatchResult`), one delegator line (`internal/app/graph/component.go`), one MCP-side addition confined to `cmd/arc/graph/serve.go` (`nodeMatchArgs`, `nodeMatchHandler`, `renderFactTable`, one `mcp.AddTool` registration, `Long`/schema-advertisement text update); one glossary-entry addition to `ARCHITECTURE.md` for `Fact`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | No ADR conflict: ADR 003 rule 1 (MCP handler stays a thin wrapper) holds — `nodeMatchHandler` only decodes JSON into a domain value and calls `appgraph.Match`. `ARCHITECTURE.md`'s Glossary MUST gain a `Fact` entry in the implementing PR (tracked as a task, not a plan-time gate failure). |
| II. DDD & Glossary | PASS | `Fact` is the only new domain concept; it is named and documented here (data-model.md) before implementation. No flag/domain-name mismatch: the MCP wire shape's `property`/`value` field names match `core.Fact.Property`/`Value` exactly. |
| III. Hexagonal Architecture | PASS | `Fact`/`Filter.MatchingFacts` live in `internal/core` (no `cobra`/`mcp` import); `service.Match` lives in `internal/app/graph/service` and calls only `fsys.Mounter`/`core.Filter`, no MCP types. `cmd/arc/graph/serve.go` remains JSON decode → domain call → render only. No new adapter, no new port. |
| IV. Functional Programming Style | PASS | `Filter.MatchingFacts` is a small, pure, side-effect-free function mirroring `statementSatisfiedBy`'s existing shape; `renderFactTable` mirrors `renderMatchTable`'s existing pure-function shape exactly. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new adapter. `service.Match` reuses `walkNodeFiles`/`readGrepNode`/`guardIsGraph` verbatim rather than re-implementing node enumeration — the exact kind of reuse Principle V/research.md precedent (027) already established for `Grep`/`Subgraph`. No Cobra command is added, per explicit user instruction — matches `context_retrieve`/`schema`'s existing MCP-only precedent rather than inventing new scope. |
| VI. TDD | PASS (process gate for implementation) | `internal/core/filter_test.go`'s new table-driven `MatchingFacts` cases, `internal/app/graph/service/match_test.go`, and `cmd/arc/graph/serve_test.go`'s new `node_match` E2E scenarios MUST all be written first, per Principle VI. |
| VII. External Integration & Adapter Consistency | PASS | No new adapter, no new external system. `service.Match` touches no filesystem/network I/O beyond the existing `internal/adapter/fsys` usage `walkNodeFiles`/`readGrepNode` already have. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario across spec.md's three user stories MUST get a colocated E2E test in `cmd/arc/graph/serve_test.go`, following the existing `TestServeNodeGrep*`/`connectServeSession` pattern. |
| IX. Command & Flag Design (CLIG) | N/A this release | No Cobra command/flag surface is introduced by this feature (explicit user instruction: a future release exposes `node_match`'s capability as a command). Nothing to check against CLIG for this PR. |
| ADR 003 (MCP Server as second primary-adapter family) | PASS | Rule 1 (thin wrapper) verified above. Rule 3 (loopback-by-default bind) is untouched — this feature does not touch `--http`/`resolveHTTPAddr`. |

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/028-node-match-filter/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── mcp-contract.md  # node_match input/output wire shape
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
└── core/
    ├── filter.go            # ADDITIVE: Fact{Property, Value string}; Filter.MatchingFacts(node Node) []Fact
    └── filter_test.go       # + table-driven MatchingFacts tests (unit, fogfish/it/v2)

internal/app/graph/
├── kernel/
│   └── match.go             # NEW: MatchEntry{ID, Property, Value}, MatchResult{Root, Matches, Unreadable}
├── service/
│   ├── match.go              # NEW: Match(ctx, mounter, filter, dir) — rejects an empty filter
│   │                         #   (ErrEmptyFilter), reuses walkNodeFiles/readGrepNode/guardIsGraph
│   │                         #   from grep.go/apply.go, keeps nodes passing filter.Match, reports
│   │                         #   filter.MatchingFacts per kept node
│   ├── match_test.go         # NEW: unit tests for Match (empty filter, zero matches, multi-fact
│   │                         #   dedup, edge-fact matches) — User Stories 1-3
│   └── errors.go             # + ErrEmptyFilter sentinel
└── component.go              # + Match delegator (mirrors Grep/Subgraph/ContextRetrieve's shape)

cmd/arc/graph/
├── serve.go                  # + nodeMatchArgs{Filter mcpFilter} (required, non-pointer — unlike the
│                             #   other tools' optional *mcpFilter); nodeMatchHandler; renderFactTable;
│                             #   one mcp.AddTool("node_match", ...) registration; schemaAdvertisement
│                             #   and NewServeCmd's Long text updated to mention node_match
└── serve_test.go             # + node_match E2E scenarios (User Stories 1-3), following the existing
                              #   TestServeNodeGrep*/connectServeSession pattern

ARCHITECTURE.md               # "Fact" glossary row added (Principle I/II, implementation-phase task)
```

**Structure Decision**: This feature adds exactly one new domain concept (`core.Fact` plus `Filter.MatchingFacts`, in the existing `internal/core/filter.go`), one new service file and one new kernel file in the existing `internal/app/graph` package family, one delegator line, and one MCP tool registration in the existing `cmd/arc/graph/serve.go`. No new package, no new adapter, and — per explicit user instruction — no new Cobra command in this release; that is deferred to a future release once the MCP-side shape has proven itself, mirroring how `context_retrieve`/`schema` shipped MCP-only before any Cobra equivalent existed.

## Complexity Tracking

*No entries — Constitution Check has no violations to justify.*
