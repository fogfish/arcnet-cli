# Phase 0 Research: Context Retrieve Tool (`context_retrieve`)

## D1: No new external dependency

**Decision**: This feature adds zero new third-party dependencies. It reuses `github.com/modelcontextprotocol/go-sdk/mcp` (already the codebase's MCP transport, ADR 003) and `internal/pkg/grep` (already `arc grep`/`node_grep`'s content-scan engine).

**Rationale**: Every capability this feature needs — MCP tool registration, regexp-backed concurrent file scanning, graph traversal — already exists in the codebase for `node_grep`/`subgraph_get`. Introducing a second scanning or traversal library would be exactly the "second, divergent client for the same capability" constitution Principle VII forbids.

## D2: Extend `internal/app/graph`, not a new domain package

**Decision**: `ContextRetrieve` is added to the existing `internal/app/graph` domain package (`kernel/context.go` for its result type, `service/context.go` for its implementation, one new delegator in `component.go`) — the same package `Grep`, `Subgraph`, and `NodeGet` already live in. `cmd/arc/graph/serve.go` gains one new tool registration and handler; no new Cobra command, no new `cmd/` package.

**Rationale**: The user directed this explicitly — "do not implement any new use-cases... it is a functional extension of existing components" — and it is what ADR 003 already requires structurally: an MCP tool handler MUST be a thin wrapper calling an `internal/app/<domain>.Component` primary-port function, never a parallel implementation. `context_retrieve`'s three-pass retrieval is a graph-read operation exactly like `arc grep`/`arc subgraph`'s own operations, so it belongs in the same domain package rather than a new one.

**Alternatives considered**: A new `internal/app/context` (or `internal/app/rag`) package was considered and rejected — it would duplicate `walkNodeFiles`/`enumerateNodes`/`buildReverseIndex`/`bfs`/`capPool`/`degree`, all already private to `internal/app/graph/service`, or force them to be exported across a package boundary for a single new caller. Neither serves this feature better than calling them directly from a same-package file.

## D3: Content-match pass reuses `internal/pkg/grep.Search`, escaped and case-insensitive

**Decision**: The content-match pass calls the same `grep.Search` `node_grep`/`arc grep` already use, with `pattern = "(?i)" + regexp.QuoteMeta(query)` — never the raw `query` string compiled directly.

**Rationale**: `regexp.QuoteMeta` guarantees every character in `query` is treated literally, so a natural-language query (parentheses, question marks, etc.) can never fail as an invalid regexp (spec FR-003) — without a second, bespoke substring-scanning loop duplicating what `grep.Search` already does concurrently and correctly for the identical file set (`walkNodeFiles`'s `.md`-only, `.arc`/`_schema`-excluding walk).

**Alternatives considered**: A hand-written `strings.Contains` scan over each node's raw file content was considered and rejected — it would need to re-derive line numbers and re-implement the concurrent worker pool `grep.Search` already provides, for no behavioral gain over the escaped-pattern approach.

## D4: Attribute-match pass is one new, small pure function

**Decision**: A new unexported function in `service/context.go`, `matchesAttrs(node core.Node, query string) bool`, checks every `core.Predicate.Value` across `node.Attrs` for case-insensitive substring containment of `query`.

**Rationale**: This is the one genuinely new piece of matching logic this feature introduces — `core.Filter`'s existing matching (`matchAttrs`/`matchAttrPatterns`) is exact-value or caller-supplied-regexp based, not substring-based, so it cannot be reused directly for a free-text query. The function is intentionally small and pure (no I/O), consistent with Principle IV.

## D5: Neighbor expansion reuses `subgraph.go`'s existing traversal primitives verbatim

**Decision**: For every node id found by the content-match or attribute-match pass, the neighbor-expansion pass calls the same `bfs`, `nodeTargets`, `buildReverseIndex`, and `capPool` helpers `Subgraph` already defines in the same package, each invoked at depth 1 in both directions (direct via `nodeTargets`, backlink via the reverse index) exactly as `Subgraph` already does for its own seed. The direct-reachable and backlink-reachable neighbor ids across every direct-match seed are pooled before `capPool` is applied once per direction, using `cfg.Subgraph.DirectCap`/`BacklinkCap` (already loaded by `buildServer` for `subgraph_get`).

**Rationale**: Clarifications session 2026-08-24 confirmed neighbor expansion must apply "the same reachable-set size safeguards... as `subgraph_get`/`arc subgraph`" (spec FR-009) — reusing the identical, already-tested functions is the only way to guarantee that literally, and avoids a second traversal/capping implementation that could silently drift from `subgraph_get`'s behavior (Principle V).

**Alternatives considered**: A single pooled BFS seeded from all direct/attribute matches at once (rather than one BFS per seed, unioned) was considered — rejected because `bfs`'s existing signature takes one seed id, and per-seed depth-1 lookups (`nodeTargets(index[id])` / `rev[id]`) are already O(1) direct map access, so the "per-seed BFS" framing adds no real cost while keeping the exact function signature `Subgraph` already validates.

## D6: Ranking — two-tier (direct vs. neighbor-only), then connectivity, then id

**Decision**: The final candidate pool is sorted by: (1) tier — direct matches (content or attribute pass) before neighbor-only matches; (2) within a tier, `degree(index, rev, id)` descending — the exact function `capPool` already uses to prioritize truncation; (3) id ascending, as the final deterministic tiebreak. The pool is then truncated to `limit`.

**Rationale**: This is the literal rule confirmed in Clarifications session 2026-08-24 ("Direct matches... rank above neighbor-only matches; within a tier, rank by hop-distance and connectivity" — hop-distance collapses to the two-tier split since neighbor expansion is fixed at one hop, per D5/spec Assumptions). Reusing `degree` (already defined for `capPool`'s own tie-break) keeps "more connected ranks higher" consistent across both this feature and `subgraph_get`'s truncation behavior, rather than inventing a second connectivity metric.

## D7: Reply rendering reuses `core.Patch` + `core.RenderPatch`, exactly like `subgraph_get`

**Decision**: `ContextRetrieve`'s result carries a synthesized `core.Patch` (document id `"context:" + slugify(query) + "@" + timestamp`, title `"Context: " + query`, `Nodes` = the ranked, truncated candidate list) — the identical value shape `Subgraph` already produces, rendered by the MCP handler via `string(core.RenderPatch(result.Patch, index))`, identical to `subgraph_get`'s own reply construction.

**Rationale**: `context_retrieve` returns "the top-N node objects with full content" (spec FR-001) — semantically the same "many independent full node objects" shape `subgraph_get` already renders, so no new rendering function is needed (ADR 003 D2 precedent: reuse existing renderers over inventing a parallel reply format). `slugify` is already defined, private, in the same package (`subgraph.go`).

**Alternatives considered**: A new markdown table (one row per node, like `node_grep`'s reply) was rejected — a table row cannot carry a node's full prose text, edges, and links, which spec FR-001 requires the reply to include.

## D8: New error sentinel `ErrInvalidLimit`, validated inside `service.ContextRetrieve`

**Decision**: A new `faults.Safe1[string]` sentinel, `ErrInvalidLimit`, added to `service/errors.go` alongside `ErrInvalidDepth`/`ErrInvalidPattern`. `service.ContextRetrieve` returns it when `limit <= 0`, before any file is read — mirroring `Grep`'s own up-front pattern validation (spec FR-007).

**Rationale**: Validating inside the service function (not the MCP handler) keeps the precondition unit-testable without spinning up an MCP session, consistent with `Grep.ErrInvalidPattern`'s placement; `subgraph_get`'s `depth` validation lives in the handler only because `depth`'s *default resolution* (nil → 1) is itself a handler-level concern before any domain value exists — `limit`'s default (10) is resolved the same way, in the handler, before the already-non-nil `int` reaches the service call.

## D9: No new configuration knob

**Decision**: No `ContextConfig` type is added to `internal/app/config/kernel`. Neighbor-expansion caps reuse the already-loaded `cfgFile.Subgraph` (`DirectCap`/`BacklinkCap`); content-match concurrency reuses the already-loaded `cfgFile.Grep.Workers`.

**Rationale**: Every configurable knob this feature needs already has a home; adding a third, overlapping config struct would violate Principle V's YAGNI rule for no behavioral gain — `context_retrieve`'s neighbor expansion is explicitly specified (Clarifications, D5) to behave exactly like `subgraph_get`'s, so sharing its cap configuration is the correct, not merely convenient, choice.

## D10: No `arc context` CLI command in this increment

**Decision**: `context_retrieve` is MCP-only in this increment — no Cobra command is added or planned as part of this feature. This is the first `internal/app/graph` primary-port function reached by exactly one primary-adapter family (MCP) instead of two (Cobra + MCP, as `NodeGet`/`Grep`/`Subgraph` already are).

**Rationale**: Confirmed in spec.md's Clarifications/Assumptions: VISION.md lists `arc context` as a separate, not-yet-scoped CLI command with its own open research question ("the specific retrieval algorithm... is not specified... and must be developed through experimentation") — conflating the two would smuggle that unresolved CLI-scope question into this feature. This does not violate ADR 003's binding rule (an MCP tool handler must be a thin wrapper over a domain `Component` function, never a parallel reimplementation) — that rule is about *how* the MCP tool calls the domain layer, not a requirement that every domain function also have a Cobra twin. Noted explicitly so it reads as a deliberate scope decision on review, not an oversight.
