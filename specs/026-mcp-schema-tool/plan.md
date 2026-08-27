# Implementation Plan: Graph Ontology Schema Tool

**Branch**: `026-mcp-schema-tool` | **Date**: 2026-08-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/026-mcp-schema-tool/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add `schema()` as a new, argument-free tool on the already-running `arc serve` MCP server: it renders the graph's already-resolved `core.Index` (built once per `buildServer` call via the existing `resolveIndexOrDefault`/`internal/app/schema.Resolve`, the same index `node_get`/`subgraph_get`/`context_retrieve` already render with) as markdown — for each predicate, name and description only; for each class, description plus its required and optional predicate names. Per explicit user direction, this reuses the schema index `buildServer` already resolves rather than adding any new schema-management logic, and reaches for no new domain package. The server additionally advertises `schema` as the recommended first call via `mcp.ServerOptions.Instructions`, sent to every connecting client in `InitializeResult` — the one currently-unset MCP mechanism built for exactly this purpose — and the Cobra `Long` help text is updated to mention it for human operators.

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/modelcontextprotocol/go-sdk/mcp` (already `arc serve`'s transport, ADR 003) — no new dependency added by this feature

**Storage**: Local graph files under the git-tracked working tree, read via `internal/adapter/fsys` through the existing `internal/app/schema.Resolve(store)` call already made once per `buildServer` invocation; no new I/O path

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit tests and colocated E2E tests extending `cmd/arc/graph/serve_test.go`'s existing `connectServeSession`/`textOf` helpers (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature extends one existing package (`cmd/arc/graph`) only; no new domain package

**Performance Goals**: Renders the in-memory `core.Index` already held by `buildServer` — no filesystem I/O or traversal per call, so response time is bounded by string formatting only (sub-millisecond for the built-in vocabulary's ~20 predicates/8 classes plus any project-specific additions)

**Constraints**: Strictly read-only (spec FR-007); MUST take no input parameters (spec FR-006); MUST reflect the schema resolved at server-start/connection time, consistent with how `node_get`/`subgraph_get`/`context_retrieve` already read the same `index` value (spec FR-005, Assumptions)

**Scale/Scope**: One new MCP tool (`schema`) plus its handler and registration in the existing `cmd/arc/graph/serve.go`; one new rendering function; a `mcp.ServerOptions.Instructions` string added to the existing `mcp.NewServer` call (currently `nil` options); no new Cobra command, no new domain package, no new configuration knob, no new error sentinel (the operation cannot fail given an already-resolved `core.Index`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | ADR 003 already covers MCP tools as a second primary-adapter family and is followed exactly; no new ADR needed. `ARCHITECTURE.md`'s MCP Tool glossary entry and its `cmd/arc/graph/serve.go` file-tree comment (currently listing `node_get`/`node_grep`/`subgraph_get`/`context_retrieve`) MUST be updated in the implementing PR — tracked as a task, not a plan-time gate failure. |
| II. DDD & Glossary | PASS | No new domain concept: `schema` renders the already-named `core.Index`/`core.PredicateDef`/`core.TypeDef` (CORE §9.1/§9.2), already documented in `internal/core/rules.go`'s GoDoc and referenced by the existing MCP Tool glossary entry. |
| III. Hexagonal Architecture | PASS | The handler calls no new domain function at all — it renders the `index core.Index` value `buildServer` already holds. `cmd/arc/graph/serve.go` remains argument-decode → domain-call (already made) → render only, the same shape as `node_get`/`subgraph_get`. |
| IV. Functional Programming Style | PASS | The new rendering function is small, pure, side-effect-free (input `core.Index`, output `string`), mirroring `renderMatchTable`'s existing shape. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new config type, no new domain call — reuses the `index` value `buildServer` already resolves via `resolveIndexOrDefault`/`internal/app/schema.Resolve`, per explicit user direction to use "existing schema management functionality." |
| VI. TDD | PASS (process gate for implementation) | Unit test for the new rendering function (empty index, built-in-only index, index with project-specific additions) MUST be written first, per Principle VI. |
| VII. External Integration & Adapter Consistency | PASS | No new adapter; no I/O of its own — the handler reads the `index` value already produced by the existing `fsys.Local{}`-backed `internal/app/schema.Resolve` call in `buildServer`. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario in spec.md's three user stories MUST get a colocated E2E test in `cmd/arc/graph/serve_test.go`, using the existing `connectServeSession`/`textOf` helpers; the session-advertisement scenarios (User Story 2) assert against `session.InitializeResult().Instructions`. |
| IX. Command & Flag Design (CLIG) | N/A | This feature adds no Cobra command or flag — `arc serve`'s own flag surface (`--http`) is unchanged; only its `Long` help text gains a mention of `schema`. |
| ADR 003 (MCP Server as second primary-adapter family) | PASS | Rule 1 (thin wrapper over an `internal/app/<domain>` function) is trivially satisfied since the handler makes no new domain call at all, only renders an already-resolved value. |

No violations requiring Complexity Tracking.

## Post-Design Re-Check

*Re-evaluated after Phase 1 (data-model.md, contracts/mcp-contract.md, quickstart.md).*

All gates above still hold after design: `schemaArgs`/`schemaHandler` (data-model.md) follow the exact factory shape already accepted for `nodeGetHandler`/`subgraphGetHandler`; `renderSchema` (data-model.md) follows `renderMatchTable`'s existing pure-function shape; the contract (contracts/mcp-contract.md) confirms `schema` has no error reply, since it renders an already-resolved, in-memory value with no I/O and no arguments to reject. No new Constitution Check risk was surfaced during design.

## Project Structure

### Documentation (this feature)

```text
specs/026-mcp-schema-tool/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── mcp-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

Every file below already exists (modified); no new package boundary is introduced (per explicit user direction: reuse existing schema management functionality, no new use-case).

```text
cmd/arc/graph/
├── serve.go                # MODIFIED: + schema tool's zero-argument input type, renderSchema,
│                             #   schemaHandler, mcp.AddTool registration; buildServer's
│                             #   mcp.NewServer call gains a non-nil &mcp.ServerOptions{Instructions: ...};
│                             #   NewServeCmd's Long help text mentions schema
└── serve_test.go           # MODIFIED: + one E2E test per spec.md acceptance scenario (8 scenarios
                              #   across 3 stories), via connectServeSession/textOf, plus assertions
                              #   against session.InitializeResult().Instructions (Principle VIII)

ARCHITECTURE.md                # MODIFIED: MCP Tool glossary entry + cmd/arc/graph file-tree comment
                              #   gain schema (Principle I, Constitution Check)
```

**Structure Decision**: This feature touches exactly one existing package — `cmd/arc/graph` (MCP wiring: one new tool handler + registration, plus a server-options change, in the existing `serve.go`). No new Cobra command, no new domain package, no new adapter, no new configuration type, no new domain function (the handler renders the `index core.Index` value `buildServer` already resolves).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No entries — Constitution Check above reports no violations.
