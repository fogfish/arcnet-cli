# Implementation Plan: MCP Tool Metadata Colocation

**Branch**: `029-mcp-tool-metadata` | **Date**: 2026-08-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/029-mcp-tool-metadata/spec.md`, plus an explicit follow-on `/speckit-plan` instruction: split `cmd/arc/graph/serve.go` into one file per MCP tool (`serve_tool_node_get.go`, etc.) plus a `serve.go` retaining `buildServer`/`NewServeCmd`/shared filter-conversion code; declare each tool's `mcp.Tool` colocated with its parameters/types; add `jsonschema` tags; draft descriptions an LLM client can act on.

## Summary

`cmd/arc/graph/serve.go` (615 lines, package `graph`) today registers all six MCP tools (`node_get`, `node_grep`, `subgraph_get`, `context_retrieve`, `schema`, `node_match`) in one `buildServer` function; each tool's args struct (with a single-line `jsonschema` description per field, no examples) is declared 100-250 lines away from its `mcp.AddTool(server, &mcp.Tool{Name, Description, ...}, handler)` call, and the shared filter type `mcpStatement` has no field-level documentation at all. This feature splits `serve.go` into `serve_tool_<name>.go` per tool (args struct + a package-level `var <name>Tool = &mcp.Tool{...}` + handler + any tool-only render function, all in one file) plus a slimmed `serve.go` retaining `buildServer`/`NewServeCmd`/`logCall`/`resolveHTTPAddr`/the shared filter wire-shape (`stringOrArray`/`mcpStatement`/`mcpFilter`/`toMatcher`/`toCoreFilter`)/two new small helpers (`must`, `withExamples`). Since Go's `const` cannot hold a struct with pointer fields, "colocated constant" is realized as a package `var` built once via `must(<tool>InputSchema())` (research.md D2). Per-field example values — which the `jsonschema` struct tag cannot express (research.md D3, confirmed by reading `google/jsonschema-go`'s tag-parsing source directly) — are attached by mutating the reflected schema's `Examples []any` field per property, the same post-processing pattern `stringOrArraySchema()` already uses for the filter's string-or-array fields. The existing single hand-written `schemaAdvertisement` session-instructions paragraph is replaced by `sessionInstructions()`, composed from one small `const <tool>WorkflowNote` per tool file plus one fixed purpose sentence, so server-wide workflow/tool-preference guidance (spec Story 4) stays colocated with the tool it describes rather than living in a disconnected paragraph a maintainer must remember to revisit. No tool's registered name, wire behavior, or JSON response shape changes; `serve_test.go` is left unsplit and passes unchanged (research.md D7).

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk v1.6.1` (`mcp.Server`/`mcp.AddTool`/`mcp.Tool`/`mcp.ToolAnnotations`, already `arc serve`'s transport, ADR 003) and `github.com/google/jsonschema-go v0.4.3` (`jsonschema.Schema`/`jsonschema.ForType`, already used for `inputSchemaFor`) — no new dependency added by this feature; both already in `go.mod`

**Storage**: N/A — this feature touches only in-memory MCP metadata construction (tool registration, JSON Schema values); no change to how graph data is read (`internal/adapter/fsys`, unchanged)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit and colocated E2E tests (constitution Principles VI, VIII); a new table-driven unit test asserting every tool's schema has per-property `Description`/`Examples` (quickstart.md steps 3-4), colocated in `cmd/arc/graph/serve_test.go` or a sibling `serve_metadata_test.go` in the same package — no new test infrastructure, reuses the existing `connectServeSession`/`mcp.NewInMemoryTransports()` pattern already in `serve_test.go`

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature touches only `cmd/arc/graph/` (splitting one existing file into seven; no new package, no new adapter, no new domain type, no new Cobra command)

**Performance Goals**: N/A/unchanged — `mcp.Tool` var construction happens once at package-load time (`must(...)`-initialized package vars), identical amortized cost to today's per-request-independent, once-per-`buildServer`-call schema derivation; no new I/O or per-request cost is introduced

**Constraints**: No tool's `Name` changes (a rename would be a breaking wire change for existing clients — explicitly out of scope, spec Assumptions: "not intended to change what the server actually returns"); no new `OutputSchema`/structured `Out` type is introduced (research.md D6 — output is documented via an added sentence in `Description`, not a wire-format change); `serve_test.go` is not split (research.md D7)

**Scale/Scope**: One file (`serve.go`) split into seven (`serve.go` + 6× `serve_tool_<name>.go`); two new small shared helpers in `serve.go` (`must[T any](T, error) T`, `withExamples(*jsonschema.Schema, string, ...any) *jsonschema.Schema`); `jsonschema` tags added to `mcpStatement`'s 6 fields (currently none); every existing tool's `Description` gains an output-describing sentence (research.md D6); one new `sessionInstructions()` function replacing the `schemaAdvertisement` constant, composed from 6 new `const <tool>WorkflowNote` values (one per tool file, research.md D5); one new colocated unit test verifying metadata completeness (quickstart.md)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS | No ADR conflict: ADR 003 rule 1 (MCP handler stays a thin wrapper) holds — no handler gains logic, only its surrounding `mcp.Tool`/schema declaration moves file and gains metadata. `ARCHITECTURE.md` needs no Glossary change (no new domain concept — see Principle II). |
| II. DDD & Glossary | PASS | No new domain concept is introduced; `mcp.Tool`/`jsonschema.Schema` are transport-layer (MCP wire) types, not domain types, and already live in `cmd/`, not `internal/core` — unchanged by this feature. |
| III. Hexagonal Architecture | PASS | All changes stay inside `cmd/arc/graph/`; no handler gains a new call into `internal/app/graph`/`internal/core` beyond what it already makes. `cmd/` remains decode → domain call → render only. |
| IV. Functional Programming Style | PASS | `must`/`withExamples` are small, pure/near-pure (the only "side effect" is `must`'s panic on a construction-time-only error, the accepted idiom per research.md D2) helper functions; each `<tool>InputSchema()` remains a small, single-purpose function mirroring `inputSchemaFor`'s existing shape. No inline comments are added to executable statements — only GoDoc-style comments above new types/vars/consts/funcs, consistent with existing `serve.go` style throughout this file today. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new adapter, no new abstraction beyond the two small helpers research.md D2/D3 justify by direct evidence (Go's `const` cannot hold `mcp.Tool`; the `jsonschema` tag cannot express `Examples`). `OutputSchema`/structured `Out` types are explicitly rejected (research.md D6) as scope the user did not ask for and the spec's own Assumptions rule out. |
| VI. TDD | PASS (process gate for implementation) | The new metadata-completeness unit test (quickstart.md steps 3-4) MUST be written first and MUST fail (e.g. against the *current* `serve.go`, which has zero `Examples` anywhere) before the split/enrichment is implemented, per Principle VI. |
| VII. External Integration & Adapter Consistency | PASS | No new adapter, no new external system; no `internal/adapter/fsys` usage changes. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario across spec.md's four user stories MUST get a colocated assertion in `cmd/arc/graph/serve_test.go` (existing `connectServeSession`/`TestServe*` pattern) or the new metadata-test file in the same package — Story 1→SC-001 (file layout, checked structurally not via `go test`, quickstart.md step 2), Story 2→SC-002, Story 3→SC-003, Story 4→SC-004/SC-005, all via quickstart.md's steps. |
| IX. Command & Flag Design (CLIG) | N/A this release | No Cobra command/flag surface changes — `arc serve`'s existing `--http` flag and `Long`/`Example` text are untouched (only the MCP-protocol-level `Instructions` string changes, which is not a CLI flag/help-text surface). |
| ADR 003 (MCP Server as second primary-adapter family) | PASS | Rule 1 (thin wrapper) verified above. Rule 3 (loopback-by-default bind) is untouched — `resolveHTTPAddr` moves file (stays in `serve.go`) with no logic change. |

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/029-mcp-tool-metadata/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — D1-D7 design decisions with evidence
├── data-model.md        # Phase 1 output — spec's Key Entities mapped to Go realization
├── quickstart.md        # Phase 1 output — 7-step validation guide
├── contracts/
│   └── mcp-contract.md  # Phase 1 output — drafted Description/example text for all 6 tools + shared filter fields + sessionInstructions() composition
└── tasks.md              # Phase 2 output (/speckit-tasks command — NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/arc/graph/
├── serve.go                       # buildServer, NewServeCmd, logCall, resolveHTTPAddr,
│                                   #   serveImplName/serveImplVersion, sessionInstructions(),
│                                   #   stringOrArray/mcpStatement (+ new jsonschema tags,
│                                   #   research.md D4)/mcpFilter/toMatcher/toCoreFilter,
│                                   #   stringOrArraySchema, inputSchemaFor,
│                                   #   must[T any](T, error) T (NEW), withExamples (NEW)
├── serve_tool_node_get.go         # NEW: nodeGetArgs, nodeGetTool var, nodeGetInputSchema,
│                                   #   nodeGetHandler, nodeGetWorkflowNote
├── serve_tool_node_grep.go        # NEW: nodeGrepArgs, nodeGrepTool var, nodeGrepInputSchema,
│                                   #   nodeGrepHandler, renderMatchTable, nodeGrepWorkflowNote
├── serve_tool_subgraph_get.go     # NEW: subgraphGetArgs, subgraphGetTool var,
│                                   #   subgraphGetInputSchema, subgraphGetHandler,
│                                   #   subgraphGetWorkflowNote
├── serve_tool_context_retrieve.go # NEW: contextRetrieveArgs, contextRetrieveTool var,
│                                   #   contextRetrieveInputSchema, contextRetrieveHandler,
│                                   #   contextRetrieveWorkflowNote
├── serve_tool_schema.go           # NEW: schemaArgs, schemaTool var, schemaHandler,
│                                   #   renderSchema, joinOrNone, schemaWorkflowNote
├── serve_tool_node_match.go       # NEW: nodeMatchArgs, nodeMatchTool var,
│                                   #   nodeMatchInputSchema, nodeMatchHandler,
│                                   #   renderFactTable, nodeMatchWorkflowNote
└── serve_test.go                  # UNCHANGED (research.md D7) — same package, same
                                    #   exported test surface, existing TestServe* pass as-is
```

**Structure Decision**: Single existing command package, `cmd/arc/graph` (no new package, per Principle V). `serve.go`'s 615 lines become 8 files: a slimmed shared `serve.go` plus one `serve_tool_<name>.go` per MCP tool (6 files), `serve_test.go` unchanged. No `internal/` package is touched — this feature is entirely a `cmd/`-layer metadata/organization change (Constitution Check row III/V above).
