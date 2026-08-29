# Phase 1 Data Model: MCP Tool Metadata Colocation

This feature adds no persisted domain data and no new `internal/core` type — it
restructures existing, already-in-memory MCP wire-protocol values
(`mcp.Tool`, `jsonschema.Schema`) and where their Go source lives. The
entities below are the spec's Key Entities (spec.md) mapped onto their
concrete Go realization (research.md D1-D6).

## Tool Specification

**Realized as**: one package-level `var <tool>Tool = &mcp.Tool{...}` per tool,
declared in that tool's own `serve_tool_<name>.go` file, directly above or
below its args struct and handler.

| Field | Source |
|---|---|
| Name | `mcp.Tool.Name` — unchanged tool names (`node_get`, `node_grep`, `subgraph_get`, `context_retrieve`, `schema`, `node_match`) |
| Description | `mcp.Tool.Description` — rewritten per research.md D6 to end with an explicit output sentence |
| Parameters | `mcp.Tool.InputSchema`, built by `<tool>InputSchema() (*jsonschema.Schema, error)` (research.md D3) |
| Output | The final sentence of `Description` (research.md D6) — no separate field exists on `mcp.Tool` for this |
| Annotations | `mcp.Tool.Annotations` — unchanged (`ReadOnlyHint: true` on every tool; this feature adds no write tool) |

## Parameter Specification

**Realized as**: one field of a tool's args struct (e.g. `nodeGrepArgs.Pattern`), carrying:

| Attribute | Source |
|---|---|
| Name | Go field name + `json:"..."` tag |
| Required | Absence of `omitempty` (existing convention, unchanged) |
| Description | `jsonschema:"..."` struct tag (existing mechanism, description text improved) |
| Default behavior (if optional) | Stated as the trailing clause of the same `jsonschema` tag, e.g. `jsonschema:"number of hops to traverse from the seed, default 1"` (existing convention on `subgraphGetArgs.Depth`/`contextRetrieveArgs.Limit`, extended to every optional field that has one) |
| Example value(s) | `withExamples(schema, "<field>", ...)` applied in `<tool>InputSchema()` (research.md D3) — NOT expressible via the struct tag itself |

## Filter Usage Example

**Realized as**: one or more `any` values (JSON-shaped Go literals, typically `map[string]any`/`mcpFilter`-shaped) passed to `withExamples(s, "filter", ...)` in a tool's `<tool>InputSchema()` function. Each example is a complete `filter` object (`{"statements": [...]}`), not an isolated sub-field value, since a usage pattern only makes sense as a whole statement (research.md D4).

Each filter-accepting tool (`node_grep`, `subgraph_get`, `context_retrieve`, `node_match`) supplies its own example set, chosen to demonstrate that tool's own most relevant usage patterns:

| Tool | Example patterns to show |
|---|---|
| `node_grep` | (a) an exact-value statement narrowing which nodes are scanned; (b) a pattern (`sourcePattern`/`targetPattern`) statement |
| `subgraph_get` | (a) a predicate-only statement (scopes neighbor traversal to one relation); (b) a statement also naming source/target (narrows which nodes are returned) — the two semantics `subgraphGetArgs.Filter`'s description already distinguishes in prose |
| `context_retrieve` | (a) a single-predicate scoping statement; (b) a multi-value (`OR`-of-values) statement using the string-or-array form |
| `node_match` (required filter) | (a) a single exact-match statement; (b) a pattern-based statement — both must be non-trivial since an empty filter is rejected (`service.ErrEmptyFilter`) |

The `mcpStatement` field-level `jsonschema` tags themselves (source/predicate/target and their `*Pattern` siblings) are added once, shared, on the type declaration in `serve.go` (research.md D4) — not duplicated per tool.

## Server Workflow Guidance

**Realized as**: `sessionInstructions() string` in `serve.go` (research.md D5), sent as `mcp.ServerOptions.Instructions` at connect time. Composed from:
1. One fixed sentence stating the server's purpose and recommending `schema` as the first call.
2. One `const <tool>WorkflowNote string` per tool, declared in that tool's own file next to its `mcp.Tool` var, joined in tool-registration order.

## Tool Preference Guidance

**Realized as**: the tool-preference half of each `<tool>WorkflowNote` (research.md D5) — e.g. `nodeGrepWorkflowNote`/`contextRetrieveWorkflowNote`/`subgraphGetWorkflowNote` each state when to prefer that tool over the others in the same overlapping cluster (all three can each be used to learn something about a node's neighborhood/content); `nodeMatchWorkflowNote` states why it exists alongside `node_grep` (facts-as-evidence vs. raw content). Not a separate type — folded into Server Workflow Guidance's composed string, per tool, since the MCP protocol has exactly one server-level guidance channel (`Instructions`) to deliver it through (research.md D5's alternatives-considered).

## Validation Rules

- Every `<tool>Tool` var's `InputSchema` MUST be non-nil for every tool that takes at least one parameter (i.e. every tool except `node_get`... — `node_get` still takes `id`, so in practice every tool but the parameterless `schema`).
- Every parameter present in an args struct MUST have a non-empty `jsonschema` description tag (no parameter ships with type-only documentation).
- Every parameter's generated schema MUST carry a non-empty `Examples` slice (spec SC-002) — verified by a table-driven unit test walking each `<tool>Tool.InputSchema.Properties` map (see quickstart.md).
- The `filter` property on every filter-accepting tool's schema MUST carry `Examples` of length ≥ 2 (spec SC-003).
- `sessionInstructions()`'s composed string MUST contain every registered tool's name at least once (a cheap proxy asserting no tool was forgotten when `sessionInstructions` was last edited).
