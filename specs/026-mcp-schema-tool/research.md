# Phase 0 Research: Graph Ontology Schema Tool

No `[NEEDS CLARIFICATION]` markers remain in spec.md's Technical Context inputs — the user's plan-phase instruction ("using existing schema management functionality available at application," explicit output shape for properties vs. classes, "advertise usage of schema as part of server description") resolved every open question spec.md's own Assumptions section had left. This document records the resulting decisions and the alternatives considered, in the Decision/Rationale/Alternatives format.

## D1: Data source — reuse `buildServer`'s already-resolved `index`, no new domain call

**Decision**: The `schema` tool handler takes a closure over the `index core.Index` value `buildServer` already computes once via `resolveIndexOrDefault(store)` (`cmd/arc/graph/subgraph.go:39`, itself a thin wrapper over `internal/app/schema.Resolve(store)`), exactly the same value `node_get`/`subgraph_get`/`context_retrieve` already close over for rendering (`serve.go:266`, `serve.go:283-305`). No new call into `internal/app/schema` or any other domain package is added.

**Rationale**: The user's plan input is explicit: "using existing schema management functionality available at application." `internal/app/schema.Resolve` already returns exactly the ontology the spec asks for — `core.Index{Predicates map[string]core.PredicateDef, Types map[string]core.TypeDef}` (`internal/core/rules.go:47-52`) — and `buildServer` already resolves it once per server construction. Adding a second call would duplicate work the server already does on every connection/restart for no behavioral difference, since `schema` (like the other three tools) is read against the same `index` snapshot for the lifetime of one `buildServer` invocation.

**Alternatives considered**: Calling `internal/app/schema.Resolve(store)` fresh inside the handler on every invocation, so a schema change made via `arc apply schema` mid-session is picked up without restarting `arc serve`. Rejected: none of the other three tools re-resolve `index` per call either (`node_get`/`subgraph_get`/`context_retrieve` all close over the same `buildServer`-time value), so doing so only for `schema` would introduce an inconsistency the spec's own Assumptions section already ruled out ("reflects the graph's current schema... including any project-specific classes or predicates added earlier in the same session" — satisfied because `buildServer` resolves once per server lifetime, and `arc apply schema` changes made *before* that resolution are already visible; changes made after start are out of scope for every existing read tool, not just this one).

## D2: Output shape — markdown, two sections, per-entity field set fixed by the plan input

**Decision**: `schema`'s reply is one `mcp.TextContent` markdown document with two sections:
- **Predicates**: one line per predicate, name and description only (`- **name**: description`), sorted by name for deterministic output.
- **Classes**: one subsection per class (`### name`), its description as prose, followed by two lines listing required and optional predicate names (`Required: a, b, c` / `Optional: d, e` — `(none)` when a list is empty), sorted by name.

**Rationale**: The plan input is explicit and narrower than spec.md's Key Entities section implied: "For properties return only property name and textual description. For classes, return its description and all required and optional properties/predicates." `core.PredicateDef`'s `Role`/`Merge`/`Label`/`Aligned` fields (`internal/core/rules.go:16-22`) are deliberately omitted from `schema`'s reply even though they exist on the resolved value — this is a narrower projection than the full `PredicateDef`, not a limitation of the data source.

**Alternatives considered**: Reusing `renderMatchTable`'s markdown-table shape (`serve.go:118`) for predicates. Rejected for classes, since a table cell cannot hold a variable-length required/optional list cleanly; kept a table for predicates alone was considered but rejected too, to keep one consistent document shape across both sections rather than mixing a table and a heading list. JSON output (a second `mcp.Content` structured payload) was considered and rejected: every existing tool (`node_get`, `node_grep`, `subgraph_get`, `context_retrieve`) replies with a single markdown `TextContent`, and `schema` following the same convention keeps the reply shape uniform across the whole tool surface (research.md precedent, `serve.go:144,169,206,247`).

## D3: Session advertisement — `mcp.ServerOptions.Instructions`, not enforcement

**Decision**: `buildServer`'s `mcp.NewServer(&mcp.Implementation{...}, nil)` call (`serve.go:281`) changes its second argument from `nil` to `&mcp.ServerOptions{Instructions: "..."}`, naming `schema` and recommending it as the first call. The SDK sends this string to every connecting client as `InitializeResult.Instructions` (`go-sdk@v1.6.1/mcp/server.go:1125`, `mcp/protocol.go:398-403`: "Instructions describing how to use the server and its features") — the exact, currently-unused mechanism this feature needs, requiring no new protocol surface.

**Rationale**: The plan input says "Advertise usage of schema as part of server description" — `Instructions` is literally the MCP server's self-description field, sent once at the start of every session before any tool is called, matching spec.md FR-008/FR-009 (guidance presented at connection time, consistently across sessions) without requiring session-state tracking to enforce a hard call order (spec.md's own Assumptions section already ruled out enforcement as out of scope).

**Alternatives considered**: Enforcing call order by tracking per-session state and rejecting other tool calls until `schema` has been invoked. Rejected per spec.md's documented Assumption — no reasonable default supports a hard-blocking interpretation of "advertise," and it would need new session-lifecycle code with its own failure-handling behavior, out of scope for this feature. Advertising only through `schema`'s own `mcp.Tool.Description` (discoverable via `tools/list`) was considered and rejected as insufficient on its own: `tools/list` requires a client to already think to call it, and to read all four tools' descriptions to notice `schema` is meant to go first, where `Instructions` reaches every client unconditionally and immediately upon connecting.

## D4: Cobra help text

**Decision**: `NewServeCmd`'s `Long` field (`serve.go:317-329`) is updated to name `schema` alongside the existing four tools and note that it is the recommended first call, mirroring how the field already documents each existing tool in one sentence.

**Rationale**: Constitution Principle XII requires help text not to drift from actual behavior; the `Long` field already enumerates every registered tool by name, so a fifth tool must be added there for `arc serve --help` to stay accurate for a human operator, independent of the MCP-level `Instructions` string an LLM client sees (D3).

**Alternatives considered**: None — this is a direct, mechanical consequence of Principle XII, not a design choice with real alternatives.
