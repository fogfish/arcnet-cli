# Implementation Plan: MCP `node_links` and `node_backlinks` Tools

**Branch**: `030-mcp-node-links-backlinks` | **Date**: 2026-08-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/030-mcp-node-links-backlinks/spec.md`, plus an explicit follow-on `/speckit-plan` instruction: add `node_links(id)` and `node_backlinks(id)` to the existing MCP server using the existing enumeration/backlink primitives, implementing the link/backlink matching service within `internal/app/graph`.

## Summary

Add two new read-only MCP tools, `node_links(id)` and `node_backlinks(id)`, following the exact `node_get`/`node_match` pattern already established in this codebase: a new `internal/app/graph/service/links.go` adds `NodeLinks`/`NodeBacklinks`, both reusing `enumerateNodes`/`guardIsGraph`/`ErrSeedNotFound` unchanged; `internal/app/graph/component.go` gets two thin delegators; `cmd/arc/graph/serve_tool_node_links.go`/`serve_tool_node_backlinks.go` register the tools and render results as markdown tables, mirroring `node_match`'s `renderFactTable` convention. Per spec Assumptions, "outgoing hrefs/edges/links" is taken literally — both tools report a node's structural `Edges` **and** its inline prose `HRefs` through the same entry shape. Since `Node.HRefs` are documented elsewhere in this codebase as "never navigable" (`internal/core/ast.go`, and `service/subgraph.go`'s `nodeTargets`/`buildReverseIndex`, which `Subgraph`'s BFS traversal depends on staying Edges-only), this feature does **not** modify those existing functions. It instead adds two small, separately-named helpers — `nodeRelations(n) []core.Link` (Edges + HRefs) and `buildRelationReverseIndex(index) reverseIndex` (built from `nodeRelations`, not `n.Edges` alone) — so the new Edges+HRefs behavior is isolated to `node_links`/`node_backlinks` and `Subgraph`'s existing Edges-only navigability invariant is left untouched. No new domain type is introduced for outgoing entries (`core.Link{Predicate, Target}` is reused as-is); one new kernel type, `kernel.BacklinkEntry{Source, Predicate}`, is added for incoming entries, mirroring `service/subgraph.go`'s already-existing private `backlinkEdge` shape. No Cobra command is added in this release, matching `node_match`/`context_retrieve`/`schema`'s existing MCP-only precedent.

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` (already `arc serve`'s transport, ADR 003) — no new dependency added by this feature

**Storage**: N/A for the lookup mechanism itself — node data continues to be read via `internal/adapter/fsys` through the same `enumerateNodes`/`guardIsGraph` enumeration `NodeGet`/`Subgraph`/`Match` already use; no new I/O path

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit tests (a new `internal/app/graph/service/links_test.go`, table-driven, covering `NodeLinks`/`NodeBacklinks`) and colocated E2E tests extending `cmd/arc/graph/serve_test.go`'s existing `connectServeSession` helper (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature adds one new kernel file, one new service file, one delegator addition, and two new MCP tool files, plus an existing-file update to `cmd/arc/graph/serve.go` — no new package, no new adapter, no new Cobra command

**Performance Goals**: `NodeLinks` is O(len(Edges)+len(HRefs)) for the single looked-up node, identical in shape to `NodeGet`'s existing single-node lookup cost; `NodeBacklinks` requires one full-graph scan (`enumerateNodes` + the new `buildRelationReverseIndex`), structurally identical in cost to `Subgraph`'s existing `buildReverseIndex` call — dominated by file I/O and node parsing, not new algorithmic complexity

**Constraints**: Both tools accept only `id` — no filter/scope parameter, matching the literal `node_links(id)`/`node_backlinks(id)` signatures the spec fixes (spec Assumptions); responses are restricted to the relation list only, no node content beyond `{predicate, target}`/`{source, predicate}` (spec FR-009); `nodeRelations`/`buildRelationReverseIndex` MUST NOT replace or alter `nodeTargets`/`admittedEdges`/`buildReverseIndex` — those remain Edges-only for `Subgraph`'s BFS traversal, which depends on that invariant; no CLI/Cobra surface for this feature in this release, matching `node_match`/`context_retrieve`/`schema`'s existing MCP-only precedent

**Scale/Scope**: One new kernel file (`internal/app/graph/kernel/links.go`: `BacklinkEntry`), one new service file (`internal/app/graph/service/links.go`: `NodeLinks`, `NodeBacklinks`, `nodeRelations`, `buildRelationReverseIndex`), one delegator addition (`internal/app/graph/component.go`), two new MCP tool files (`cmd/arc/graph/serve_tool_node_links.go`, `serve_tool_node_backlinks.go`), one existing-file update (`cmd/arc/graph/serve.go`: two `mcp.AddTool` registrations, `sessionInstructions()`, `NewServeCmd`'s `Long` text); one short `ARCHITECTURE.md` note documenting why `node_links`/`node_backlinks` deliberately diverge from `Subgraph`'s Edges-only HRefs treatment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | ADR 003 rule 1 (MCP handler stays a thin wrapper) holds — both handlers only decode JSON, call the identical `appgraph.NodeLinks`/`appgraph.NodeBacklinks` function, and render. `ARCHITECTURE.md` MUST gain a short note (tracked as a task, not a plan-time gate failure) explaining that `node_links`/`node_backlinks` deliberately include `HRefs` alongside `Edges` while `Subgraph`'s traversal deliberately does not — so a future contributor reading `nodeTargets`'s "HRefs are never navigable" comment does not assume that holds everywhere. |
| II. DDD & Glossary | PASS | No new `core`-level domain type is introduced — `core.Link` (existing) is reused verbatim for outgoing entries. `kernel.BacklinkEntry` is a `kernel`-layer DTO, same category as `kernel.MatchEntry` (spec 028), which required no Glossary entry of its own. |
| III. Hexagonal Architecture | PASS | `NodeLinks`/`NodeBacklinks` live in `internal/app/graph/service` (no `cobra`/`mcp` import), calling only `fsys.Mounter`/`core.Node` types. `cmd/arc/graph/serve_tool_node_links.go`/`serve_tool_node_backlinks.go` remain JSON decode → domain call → render only. No new adapter, no new port. |
| IV. Functional Programming Style | PASS | `nodeRelations`/`buildRelationReverseIndex` are small, pure, side-effect-free functions mirroring `nodeTargets`/`buildReverseIndex`'s existing shape exactly; `renderLinkTable`/`renderBacklinkTable` mirror `renderFactTable`'s existing pure-function shape. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new adapter, no new error sentinel (`ErrSeedNotFound` is reused unchanged). `nodeRelations`/`buildRelationReverseIndex` are added as new, narrowly-scoped functions rather than parameterizing the existing Edges-only `nodeTargets`/`buildReverseIndex` with a "include HRefs?" flag threaded through `Subgraph`'s unrelated call sites — the smaller, single-purpose addition Principle V/Interface Segregation favors. |
| VI. TDD | PASS (process gate for implementation) | `internal/app/graph/service/links_test.go` and `cmd/arc/graph/serve_test.go`'s new `node_links`/`node_backlinks` E2E scenarios MUST be written first, per Principle VI. |
| VII. External Integration & Adapter Consistency | PASS | No new adapter, no new external system. `NodeLinks`/`NodeBacklinks` touch no I/O beyond the existing `internal/adapter/fsys` usage `enumerateNodes` already has. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario across spec.md's three user stories MUST get a colocated E2E test in `cmd/arc/graph/serve_test.go`, following the existing `TestServeNodeMatch*`/`connectServeSession` pattern. |
| IX. Command & Flag Design (CLIG) | N/A this release | No Cobra command/flag surface is introduced by this feature — matches `node_match`/`context_retrieve`/`schema`'s existing MCP-only precedent. `specs/VISION.md` Phase 8 already names `arc edges`/`arc backlinks` as separate, not-yet-scheduled future roadmap items; this feature does not commit to or preclude them. |
| ADR 003 (MCP Server as second primary-adapter family) | PASS | Rule 1 (thin wrapper) verified above. Rule 3 (loopback-by-default bind) is untouched — this feature does not touch `--http`/`resolveHTTPAddr`. |

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/030-mcp-node-links-backlinks/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── mcp-contract.md  # node_links/node_backlinks input/output wire shape
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/app/graph/
├── kernel/
│   └── links.go              # NEW: BacklinkEntry{Source, Predicate string}
├── service/
│   ├── links.go               # NEW: NodeLinks(ctx, mounter, dir, id) ([]core.Link, error);
│   │                          #   NodeBacklinks(ctx, mounter, dir, id) ([]kernel.BacklinkEntry, error);
│   │                          #   nodeRelations(n core.Node) []core.Link (Edges + HRefs, new — NOT
│   │                          #   nodeTargets, which stays Edges-only for Subgraph);
│   │                          #   buildRelationReverseIndex(index nodeIndex) reverseIndex (new,
│   │                          #   built from nodeRelations — NOT buildReverseIndex, which stays
│   │                          #   Edges-only for Subgraph); both reuse enumerateNodes/guardIsGraph/
│   │                          #   ErrSeedNotFound from subgraph.go/apply.go/errors.go unchanged
│   └── links_test.go          # NEW: unit tests for NodeLinks/NodeBacklinks (zero-relation, unknown
│                              #   id, self-reference, duplicate predicate/target, href-with-no-
│                              #   -predicate cases) — User Stories 1-3, Edge Cases
└── component.go               # + NodeLinks/NodeBacklinks delegators (mirror NodeGet/Match's shape)

cmd/arc/graph/
├── serve_tool_node_links.go     # NEW: nodeLinksTool, nodeLinksArgs{ID string}, nodeLinksHandler,
│                                #   renderLinkTable, nodeLinksWorkflowNote — mirrors
│                                #   serve_tool_node_get.go's shape
├── serve_tool_node_backlinks.go # NEW: nodeBacklinksTool, nodeBacklinksArgs{ID string},
│                                #   nodeBacklinksHandler, renderBacklinkTable,
│                                #   nodeBacklinksWorkflowNote
├── serve.go                     # + two mcp.AddTool registrations; sessionInstructionsPurpose text
│                                #   updated ("six tools" → "eight tools", names added);
│                                #   sessionInstructions() appends the two new WorkflowNote consts;
│                                #   NewServeCmd's Long text updated to mention node_links/
│                                #   node_backlinks
└── serve_test.go                # + node_links/node_backlinks E2E scenarios (User Stories 1-3),
                                 #   following the existing TestServeNodeMatch*/connectServeSession
                                 #   pattern

ARCHITECTURE.md                 # short note: node_links/node_backlinks deliberately include HRefs
                                 #   alongside Edges; Subgraph's traversal deliberately does not
                                 #   (Principle I, implementation-phase task)
```

**Structure Decision**: This feature adds one new kernel file (`kernel.BacklinkEntry`), one new service file (`NodeLinks`/`NodeBacklinks` plus two small, deliberately new — not shared with `Subgraph` — relation-combining helpers), one delegator addition, and two new MCP tool files in the existing `cmd/arc/graph/` per-tool-file convention (established by spec 029), plus updates to `serve.go`'s registration/instructions/help text. No new package, no new adapter, and — matching `node_match`/`context_retrieve`/`schema` precedent — no new Cobra command in this release.

## Complexity Tracking

*No entries — Constitution Check has no violations to justify.*
