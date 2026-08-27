# Phase 1 Data Model: Graph Ontology Schema Tool

No new persisted or domain type is introduced (research.md D1). This document names the existing types `schema`'s handler reads and the two new, MCP-adapter-local shapes it produces.

## Existing types read (unchanged, `internal/core/rules.go`)

- **`core.Index`** — `{Predicates map[string]PredicateDef, Types map[string]TypeDef}`, the graph's effective schema, already resolved once per `buildServer` call and already read by `node_get`/`subgraph_get`/`context_retrieve`.
- **`core.PredicateDef`** — `{Role string, Merge MergeOp, Label string, Aligned string, Description string}`. `schema`'s reply projects only `Description`; `Role`/`Merge`/`Label`/`Aligned` are read from the same value by the existing tools' renderers (`core.RenderNode`/`RenderPatch`) but are out of scope for this feature's output (research.md D2).
- **`core.TypeDef`** — `{Required []string, Optional []string, Description string}`. `schema`'s reply projects all three fields.

## New types (`cmd/arc/graph/serve.go`, MCP-adapter-local — no `internal/` package addition)

- **`schemaArgs`** — the tool's input schema: an empty struct (`struct{}`), following `mcp.AddTool`'s generic `In` parameter convention. No fields, matching spec FR-006 (no input parameters).
- **`schemaHandler`** — `func(dir string, index core.Index) func(context.Context, *mcp.CallToolRequest, schemaArgs) (*mcp.CallToolResult, any, error)`, mirroring `nodeGetHandler`/`subgraphGetHandler`'s existing factory shape (`serve.go:132`, `serve.go:182`) — closes over the already-resolved `index`, calls `renderSchema(index)`, wraps the result in one `mcp.TextContent`, and calls `logCall("schema", ..., nil)` exactly like every other handler (the operation cannot fail once `index` is already in hand, so the `error` return is always `nil` — kept only because `mcp.AddTool`'s handler signature requires it, matching Go idiom for a function whose contract requires an error return that this particular implementation never produces).
- **`renderSchema(index core.Index) string`** — pure function, no I/O, no `context.Context` — mirrors `renderMatchTable`'s existing shape (`serve.go:116-123`). Builds the markdown document described in research.md D2:
  ```text
  ## Predicates

  - **<name>**: <description>
  ...

  ## Classes

  ### <name>

  <description>

  Required: <name>, <name>, ...  (or "(none)")
  Optional: <name>, <name>, ...  (or "(none)")
  ...
  ```
  Both `Predicates` and `Types` maps are iterated in sorted-key order (`sort.Strings` over `slices.Collect(maps.Keys(...))` or equivalent) for deterministic, diffable output — the same determinism concern `renderMatchTable` does not need (it iterates a pre-ordered `[]kernel.Match` slice, not a map) but `schema` does, since `core.Index`'s fields are Go maps.

## Session-level change (no new type)

- `buildServer`'s `mcp.NewServer(&mcp.Implementation{...}, nil)` call gains a non-nil second argument: `&mcp.ServerOptions{Instructions: schemaAdvertisement}`, where `schemaAdvertisement` is a package-level `const string` (research.md D3) naming `schema` and recommending it as the first call of a session. No new Go type — `mcp.ServerOptions` already exists in the SDK.
