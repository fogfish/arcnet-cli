# Phase 0 Research: MCP `node_links` and `node_backlinks` Tools

## D1 — Reuse `core.Link` for `node_links`' output; no new domain type

**Problem**: `node_links`' entry shape (`{predicate, target}`) is structurally identical to the existing `core.Link{Predicate, Target, Alias}` type already used for both `Node.Edges` and `Node.HRefs`.

**Decision**: `NodeLinks` returns `[]core.Link` directly — no new type is introduced. `Alias` is simply not rendered by the MCP handler's table (two columns only), rather than defining a stripped-down duplicate struct.

**Rationale**: Mirrors `NodeGet`'s existing precedent of returning a `core.Node` (a domain type) directly rather than a `kernel`-wrapped DTO, and avoids a redundant type whose fields would be a strict subset of `core.Link`'s (Principle V, YAGNI).

**Alternatives considered**: A new `kernel.LinkEntry{Predicate, Target string}` was considered for symmetry with `kernel.BacklinkEntry` (D2), but rejected — `core.Link` already exists, is already the type both `Node.Edges` and `Node.HRefs` are stored as, and duplicating it buys no benefit `omitempty`/table-rendering can't already handle.

## D2 — New `kernel.BacklinkEntry` for `node_backlinks`' output

**Problem**: `node_backlinks`' entry shape (`{source, predicate}`) has no existing public type. `internal/app/graph/service/subgraph.go` already has a structurally identical *private* type, `backlinkEdge{Source, Predicate string}`, but it is unexported and scoped to `Subgraph`'s own reverse-index bookkeeping.

**Decision**: Add `kernel.BacklinkEntry{Source, Predicate string}` in a new `internal/app/graph/kernel/links.go`. `service.NodeBacklinks` converts each matched `backlinkEdge`-shaped result into a `kernel.BacklinkEntry` at the service/kernel boundary, the same seam `service.Match` already uses to convert its internal fact walk into `kernel.MatchEntry`.

**Rationale**: `kernel` types are what `cmd/` renders (component.go's established convention — `Match`/`Subgraph`/`Grep` all return `kernel.*Result` types); reusing the private `backlinkEdge` shape's fields keeps the new type trivial to produce.

**Alternatives considered**: Exporting `service.backlinkEdge` itself (renaming it `BacklinkEdge`) was rejected — it would pull a `service`-internal bookkeeping type across the service/kernel boundary that `component.go`'s delegators are supposed to own, and would tie `Subgraph`'s internal representation to `node_backlinks`' public output shape for no reason.

## D3 — New, separate relation-combining helpers — `Subgraph`'s Edges-only invariant is not touched

**Problem**: Per spec 030's Assumptions, `node_links`/`node_backlinks` must report **both** a node's structural `Edges` and its inline-prose `HRefs`. But `internal/core/ast.go` documents `HRefs` as "never a source of navigable edges" (AST invariant 3), and `internal/app/graph/service/subgraph.go`'s `nodeTargets`/`admittedEdges`/`buildReverseIndex` — all consumed by `Subgraph`'s BFS traversal and degree ranking — are explicitly, deliberately Edges-only (`nodeTargets`'s own doc comment: "HRefs are never navigable structural connections"). Changing those functions' behavior to include HRefs would silently change `Subgraph`'s traversal semantics, an unrelated feature this plan must not touch.

**Decision**: Add two new, separately-named functions in the new `service/links.go`, not in `subgraph.go`:
- `nodeRelations(n core.Node) []core.Link` — returns `n.Edges` followed by `n.HRefs`, unlike `nodeTargets` (Edges only).
- `buildRelationReverseIndex(index nodeIndex) reverseIndex` — identical shape to `buildReverseIndex`, but iterates `nodeRelations(n)` instead of `n.Edges` alone.

Both reuse the existing unexported `nodeIndex`/`reverseIndex`/`backlinkEdge` types from `subgraph.go` (same package, `service`) — only the iteration source changes; the existing `nodeTargets`/`admittedEdges`/`buildReverseIndex`/`admittedBacklinks` functions are left completely unmodified and continue to back `Subgraph` exactly as before.

**Rationale**: This is the smallest change that satisfies both constraints simultaneously — `node_links`/`node_backlinks` get the Edges+HRefs behavior the spec requires, and `Subgraph`'s existing, documented, load-bearing Edges-only invariant is provably untouched (no shared call site changes behavior). Threading a boolean "include HRefs?" parameter through the existing shared functions was considered and rejected as the more complex option (see Alternatives).

**Alternatives considered**: Adding an `includeHRefs bool` parameter to `nodeTargets`/`buildReverseIndex` and updating `Subgraph`'s two call sites to pass `false` was rejected — it makes every future reader of `Subgraph`'s code additionally reason about a parameter that is always `false` there, for the sole benefit of two unrelated new call sites; two small, clearly-named, single-purpose functions are more in keeping with Principle IV/V (self-explanatory naming, single responsibility) than one shared function branching on a caller-supplied flag.

## D4 — Output rendered as a markdown table, not raw JSON

**Problem**: This server's established convention for a list-shaped MCP tool result (`node_match`) is a markdown table (`renderFactTable`), not a raw JSON payload, even though the underlying domain shape is flat (`{id, property, value}`).

**Decision**: `node_links`/`node_backlinks` follow the same convention: `renderLinkTable([]core.Link) string` renders `| predicate | target |`, and `renderBacklinkTable([]kernel.BacklinkEntry) string` renders `| source | predicate |` — header row always present, header-only when the result is empty (mirrors `renderFactTable`'s FR-007-equivalent behavior, spec 030 FR-006).

**Rationale**: Consistency with every other list-shaped MCP tool response in this server; an LLM client already knows how to read this server's table-shaped replies.

**Alternatives considered**: Returning structured JSON via `mcp.CallToolResult`'s output-schema mechanism was considered but rejected as out of scope — no existing tool in this server uses it yet (spec 029's research.md D6 explicitly deferred introducing a structured `Out`/`OutputSchema` type), and introducing it here for two tools while every sibling tool stays markdown-only would be an inconsistent, unscoped expansion.

## D5 — Not-found handling reuses `ErrSeedNotFound`

**Problem**: Spec FR-005 requires both tools to reject an unknown `id` with a clear not-found error, consistent with `NodeGet`'s existing behavior for the same condition.

**Decision**: Both `NodeLinks` and `NodeBacklinks` look up `id` in the `enumerateNodes` result exactly as `NodeGet` already does, returning `ErrSeedNotFound.With(errNoCause, id)` on a miss. No new error sentinel is added.

**Rationale**: `ErrSeedNotFound` is already the single, established "id doesn't exist" sentinel shared by `NodeGet` and `Subgraph` (`service/errors.go:29`); spec 030's Edge Cases explicitly requires the same behavior, so reusing it is direct compliance with Principle V (no duplicate error for an identical condition).

**Alternatives considered**: None — this is the only reasonable choice given the existing precedent and the spec's explicit requirement to match it.

## D6 — No Cobra command in this release

**Problem**: `specs/VISION.md` Phase 8 names `arc edges <basename>`/`arc backlinks <basename>` as future, separate roadmap items (both still `[ ]` unimplemented), distinct from this feature's MCP-only scope.

**Decision**: This feature adds `node_links`/`node_backlinks` to `arc serve` only. No new Cobra subcommand is introduced.

**Rationale**: Matches the existing, established precedent already set by `context_retrieve`, `schema`, and `node_match` (spec 028 D6) — each shipped MCP-only before any Cobra equivalent existed. The feature's own explicit instruction ("Add ... functions to existing server ... implement the ... service within `internal/app/graph`") names only the MCP/service surface, not a CLI command.

**Alternatives considered**: Adding `arc edges`/`arc backlinks` in the same PR was rejected — VISION.md already tracks them as separate, not-yet-scheduled roadmap items; bundling them here would expand this feature's scope beyond what was requested.
