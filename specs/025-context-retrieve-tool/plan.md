# Implementation Plan: Context Retrieve Tool (`context_retrieve`)

**Branch**: `025-context-retrieve-tool` | **Date**: 2026-08-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/025-context-retrieve-tool/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add `context_retrieve(query, filter?, limit?)` as a new tool on the already-running `arc serve` MCP server: a three-pass retrieval (content match, attribute match, neighbor expansion) over the graph, deduplicated, ranked (direct matches before neighbor-only, then connectivity, then id), and truncated to `limit` (default 10), returned as full node objects. Per explicit user direction, this is a functional extension of existing components, not a new use-case: the retrieval logic lives in the existing `internal/app/graph` domain package (reusing `Grep`'s content-scan engine and `Subgraph`'s traversal/capping primitives verbatim), and the MCP wiring is one new tool registration in the existing `cmd/arc/graph/serve.go` — no new Cobra command, no new domain package, no new external dependency (research.md D1-D2).

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/modelcontextprotocol/go-sdk/mcp` (already `arc serve`'s transport, ADR 003), `github.com/fogfish/faults` — no new dependency added by this feature (research.md D1)

**Storage**: Local graph files under the git-tracked working tree, read via `internal/adapter/fsys` — identical storage model to `node_grep`/`subgraph_get`; no caching or index layer (Clarifications 2026-08-24, spec Assumptions)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit tests (`internal/app/graph/service/context_test.go`) and colocated E2E tests extending `cmd/arc/graph/serve_test.go`'s existing `connectServeSession` helper (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature, no platform-specific code

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature extends two existing packages (`internal/app/graph`, `cmd/arc/graph`) rather than adding a new package boundary

**Performance Goals**: Retrieval against a graph of several thousand nodes completes in under 10 seconds (spec SC-003), matching `subgraph_get`'s own bar since neighbor expansion reuses its traversal and no index accelerates either tool in this increment

**Constraints**: Strictly read-only (spec FR-010); every call MUST reflect current on-disk state, no restart required (spec FR-011); a per-call failure MUST NOT crash the server (spec FR-012); `query` MUST never be compiled as a caller-supplied regexp (spec FR-003, research.md D3)

**Scale/Scope**: One new MCP tool, one new kernel result type (`kernel.ContextRetrieveResult`), one new service function (`service.ContextRetrieve`), one new `component.go` delegator, one new error sentinel (`ErrInvalidLimit`), one new tool handler + registration in `cmd/arc/graph/serve.go` — no new Cobra command (research.md D10), no new configuration knob (research.md D9)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | ADR 003 already covers MCP tools as a second primary-adapter family and is followed exactly (research.md D2); no new ADR needed. `ARCHITECTURE.md`'s MCP Tool glossary entry and its `cmd/arc/graph/serve.go` file-tree comment (currently listing only `node_get`/`node_grep`/`subgraph_get`) MUST be updated in the implementing PR per Principle I's rule — tracked as a task, not a plan-time gate failure. |
| II. DDD & Glossary | PASS | No new domain concept beyond what spec.md's Key Entities already name (Retrieval Candidate, Query) — both are implementation-internal, not user-facing glossary terms distinct from existing Node/Filter. |
| III. Hexagonal Architecture | PASS | Retrieval logic lives in `internal/app/graph/service` (no `cobra` or `mcp` import); `cmd/arc/graph/serve.go` remains argument-decode → domain-call → render only, unchanged shape from `node_grep`/`subgraph_get`. |
| IV. Functional Programming Style | PASS | New functions (`matchesAttrs`, ranking comparator) are small, pure, side-effect-free, reusing `degree`/`bfs`/`capPool` rather than duplicating their logic. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new config type, no new rendering function — every reusable primitive (`walkNodeFiles`, `enumerateNodes`, `buildReverseIndex`, `bfs`, `capPool`, `degree`, `slugify`, `core.RenderPatch`) is called, not re-implemented (research.md D3, D5-D7, D9). |
| VI. TDD | PASS (process gate for implementation) | Unit tests for `service.ContextRetrieve` (three-pass merge, ranking, dedup, limit/filter validation) MUST be written first, per Principle VI. |
| VII. External Integration & Adapter Consistency | PASS | No new adapter; reuses the existing `fsys.Mounter`/`fsys.Local` port already used by `Grep`/`Subgraph`/`NodeGet`. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario in spec.md's three user stories MUST get a colocated E2E test in `cmd/arc/graph/serve_test.go`, using the existing `connectServeSession`/`textOf` helpers. |
| IX. Command & Flag Design (CLIG) | N/A | This feature adds no Cobra command or flag (research.md D10) — `arc serve`'s own flag surface (`--http`) is unchanged. |
| ADR 003 (MCP Server as second primary-adapter family) | PASS, with one noted asymmetry | Rule 1 (thin wrapper over an `internal/app/<domain>.Component` function) is followed exactly. `context_retrieve` has no Cobra-command twin in this increment (research.md D10) — a deliberate scope decision (VISION.md's `arc context` CLI command remains a separate, unscoped item with its own open ranking-algorithm question), not a violation of ADR 003's binding rule, which governs *how* an MCP tool calls the domain layer rather than requiring a Cobra mirror for every domain function. |

No violations requiring Complexity Tracking.

## Post-Design Re-Check

*Re-evaluated after Phase 1 (data-model.md, contracts/mcp-contract.md, quickstart.md).*

All gates above still hold after design: `kernel.ContextRetrieveResult` and `contextRetrieveArgs` (data-model.md) follow the exact shape of the already-accepted `kernel.SubgraphResult`/`subgraphGetArgs`; the contract (contracts/mcp-contract.md) reuses `core.RenderPatch` and the existing filter/error contracts verbatim. No new Constitution Check risk was surfaced during design.

## Project Structure

### Documentation (this feature)

```text
specs/025-context-retrieve-tool/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── mcp-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

Every file below either already exists (modified) or is new; no new package boundary is introduced (research.md D2, per explicit user direction: this is a functional extension of `internal/app/graph` and `cmd`, not a new use-case).

```text
cmd/arc/graph/
├── serve.go                # MODIFIED: + contextRetrieveArgs, contextRetrieveHandler, mcp.AddTool registration
└── serve_test.go           # MODIFIED: + one E2E test per spec.md acceptance scenario (12 scenarios across 3 stories), via connectServeSession/textOf (Principle VIII)

internal/app/graph/
├── component.go             # MODIFIED: + ContextRetrieve delegator, mirroring Grep/Subgraph/NodeGet
├── kernel/
│   └── context.go           # NEW: kernel.ContextRetrieveResult
└── service/
    ├── context.go            # NEW: ContextRetrieve, matchesAttrs, ranking comparator — reuses walkNodeFiles/enumerateNodes/buildReverseIndex/bfs/capPool/degree/slugify from grep.go/subgraph.go (same package)
    ├── context_test.go       # NEW: unit tests — three-pass merge, ranking, dedup, filter scoping, limit validation (Principle VI)
    └── errors.go              # MODIFIED: + ErrInvalidLimit

internal/app/config/kernel/
└── config.go                 # UNCHANGED: reuses existing SubgraphConfig (caps) and GrepConfig (workers) — research.md D9

ARCHITECTURE.md                # MODIFIED: MCP Tool glossary entry + cmd/arc/graph file-tree comment gain context_retrieve (Principle I, Constitution Check)
```

**Structure Decision**: This feature touches exactly two existing packages — `internal/app/graph` (domain logic: new `kernel/context.go`, new `service/context.go`, one new `component.go` delegator, one new error) and `cmd/arc/graph` (MCP wiring: one new tool handler + registration in the existing `serve.go`). No new Cobra command, no new domain package, no new adapter, no new configuration type (research.md D2, D9, D10).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No entries — Constitution Check above reports no violations.
