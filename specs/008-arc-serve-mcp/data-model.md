# Phase 1 Data Model: MCP Server (`arc serve`)

No new domain entity is introduced by this feature. Every type below either already exists (reused as-is) or is a small, private, presentation-tier shape confined to `cmd/arc/graph/serve.go`.

## Reused, unchanged domain types

| Type | Source | Role in this feature |
|---|---|---|
| `core.Node` | `internal/core/ast.go` | `node_get`'s return value and `subgraph_get`'s per-entry shape (id, kind, attrs, text, notes, hrefs, edges, links) — matches spec.md's Node Object key entity exactly |
| `core.Patch` | `internal/core/ast.go` | The value `core.RenderPatch` serializes for `subgraph_get`'s reply — identical to `kernel.SubgraphResult.Patch`, already produced by `service.Subgraph` |
| `core.Filter` | `internal/core/filter.go` | The value `node_grep`'s decoded MCP filter object is converted into before calling `service.Grep` |
| `kernel.GrepResult` / `kernel.Match` | `internal/app/graph/kernel/grep.go` | `service.Grep`'s return value; `node_grep`'s markdown table is built directly from its `Matches []Match` |
| `kernel.SubgraphResult` | `internal/app/graph/kernel/subgraph.go` | `service.Subgraph`'s return value; only `.Patch` is used by `subgraph_get`'s reply (the truncation/count fields are not surfaced through this tool — see spec Edge Cases: bounded, not reported per-field, to the MCP client) |

## New, small, private types (`cmd/arc/graph/serve.go`)

**`mcpFilter`** — the wire shape `node_grep`'s optional `filter` argument decodes into (research.md D4):

| Field | JSON key | Maps to `core.Filter` |
|---|---|---|
| `Kind` | `kind` | `Filter.Kinds` (`[]core.Kind`, OR semantics — unchanged from existing Filtering rules) |
| `Tags` | `tags` | `Filter.Tags` (AND semantics) |
| `Attrs` | `attrs` | `Filter.Attrs` (exact-match, AND semantics) |
| `AttrPatterns` | `attrPatterns` | `Filter.AttrPatterns`, each value `regexp.Compile`d (AND semantics) |

An absent or all-empty `mcpFilter` converts to a zero-value `core.Filter{}` (matches every node), identical to the existing CLI behavior with no `--kind`/`--tag`/`--attr` flags.

**`nodeGetArgs`** — `node_get`'s input schema (auto-derived by `mcp.AddTool` from struct tags):

| Field | JSON key | Required | Notes |
|---|---|---|---|
| `ID` | `id` | yes | Node basename (spec Assumptions: id = basename) |

**`nodeGrepArgs`** — `node_grep`'s input schema:

| Field | JSON key | Required | Notes |
|---|---|---|---|
| `Pattern` | `pattern` | yes | Regexp, same syntax `arc grep` accepts |
| `Filter` | `filter` | no | `*mcpFilter`, omitted/null matches every node |

**`subgraphGetArgs`** — `subgraph_get`'s input schema:

| Field | JSON key | Required | Notes |
|---|---|---|---|
| `ID` | `id` | yes | Seed node basename |
| `Depth` | `depth` | no | `*int`; nil resolves to default `1` (spec FR-011) |

No output-schema type is declared for any of the three tools (research.md D2: `AddTool`'s `Out` type parameter is `any`, so no `outputSchema` is advertised and no `StructuredContent` is populated) — each tool's entire reply is one `mcp.TextContent` whose `Text` is markdown, per the user's explicit instruction.

## Validation rules (carried into Functional Requirements, unchanged from spec.md)

- `nodeGetArgs.ID` / `subgraphGetArgs.ID` unresolved against the graph → `service.ErrSeedNotFound` (existing sentinel, reused verbatim) → tool call returns `IsError: true` with the error text as content (spec FR-007/FR-013).
- `nodeGrepArgs.Pattern` not a valid regexp → `service.ErrInvalidPattern` (existing sentinel) → same error-content contract (spec FR-010).
- `subgraphGetArgs.Depth`, once dereferenced, not a non-negative integer → `service.ErrInvalidDepth` (existing sentinel) → same contract (spec FR-013). Note: since the field is a Go `*int`, "not an integer" is rejected by JSON-schema-driven input validation before the handler runs at all; only "negative" reaches `ErrInvalidDepth`.
- `mcpFilter.AttrPatterns` value not a valid regexp → a new, small `faults.Safe1` sentinel (`ErrInvalidFilterPattern`), same error-content contract.
