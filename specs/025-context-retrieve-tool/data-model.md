# Phase 1 Data Model: Context Retrieve Tool (`context_retrieve`)

This feature adds one small result type and one error sentinel to the existing `internal/app/graph` package, and one small input-schema type to `cmd/arc/graph/serve.go`. Every other type is reused, unchanged.

## Reused, unchanged domain types

| Type | Source | Role in this feature |
|---|---|---|
| `core.Node` | `internal/core/ast.go` | Every retrieval candidate, before and after ranking; the per-entry shape inside the synthesized `core.Patch` |
| `core.Patch` | `internal/core/ast.go` | The value `core.RenderPatch` serializes for `context_retrieve`'s reply — same shape `kernel.SubgraphResult.Patch` already uses |
| `core.Filter` | `internal/core/filter.go` | The value the decoded MCP `filter` argument converts into before calling `service.ContextRetrieve`; restricts every one of the three passes (spec FR-004) |
| `mcpFilter` | `cmd/arc/graph/serve.go` | Reused verbatim (research.md D2 in specs/008-arc-serve-mcp) — `context_retrieve`'s `filter` argument decodes into the identical type `node_grep`/`subgraph_get` already use |

## New domain type: `kernel.ContextRetrieveResult`

`internal/app/graph/kernel/context.go`, mirroring `kernel.SubgraphResult`'s shape:

| Field | Type | Meaning |
|---|---|---|
| `Root` | `string` | The graph root that was searched |
| `Query` | `string` | The query text, as given |
| `Limit` | `int` | The resolved limit (post-default) |
| `Patch` | `core.Patch` | The ranked, truncated candidates as a synthesized patch-exchange document, ready for `core.RenderPatch` |
| `ContentMatched` | `int` | Count of nodes found by the content-match pass, before ranking/truncation |
| `AttrMatched` | `int` | Count of nodes found by the attribute-match pass, before ranking/truncation |
| `NeighborReachable` | `int` | Count of nodes found by neighbor expansion, before capping |
| `NeighborIncluded` | `int` | Count of neighbor-expansion nodes retained after `capPool` |
| `NeighborTruncated` | `bool` | True when `NeighborReachable > NeighborIncluded` |
| `Truncated` | `bool` | True when the combined, deduplicated candidate pool exceeded `Limit` before truncation |

## New error sentinel

`internal/app/graph/service/errors.go` gains one entry, alongside `ErrInvalidDepth`/`ErrInvalidPattern`:

```go
ErrInvalidLimit = faults.Safe1[string]("limit %s must be a positive integer")
```

## New, small, private type (`cmd/arc/graph/serve.go`)

**`contextRetrieveArgs`** — `context_retrieve`'s input schema (auto-derived by `mcp.AddTool` from struct tags):

| Field | JSON key | Required | Notes |
|---|---|---|---|
| `Query` | `query` | yes | Free-text query, matched literally and case-insensitively (spec FR-003) |
| `Filter` | `filter` | no | `*mcpFilter`, omitted/null matches every node |
| `Limit` | `limit` | no | `*int`; nil resolves to default `10` (spec FR-006), mirroring `subgraphGetArgs.Depth`'s nil-resolution pattern |

## Validation rules (carried into Functional Requirements, unchanged from spec.md)

- `contextRetrieveArgs.Limit`, once resolved (default 10 when nil), not a positive integer → `service.ErrInvalidLimit` → tool call returns `IsError: true` with the error text as content (spec FR-007). A non-integer JSON value for `limit` is rejected by MCP input-schema validation before the handler runs at all, matching `subgraphGetArgs.Depth`'s existing precedent.
- `contextRetrieveArgs.Filter.AttrPatterns` value not a valid regexp → `service.ErrInvalidFilterPattern` (existing sentinel, reused verbatim) → same error-content contract already established for `node_grep`/`subgraph_get`.
- No candidate found by any of the three passes, or `filter` excludes every candidate → empty `Patch.Nodes`, not an error (spec FR-008) — same "empty result, not an error" contract `node_grep`'s zero-match case already establishes.
- `query` is never validated as a regexp — it cannot fail that way by construction (research.md D3: always escaped via `regexp.QuoteMeta` before reaching `grep.Search`).
