# Data Model: CLI/MCP "Type" Terminology Consistency

This feature introduces no new entity and no new data shape. It renames one existing internal field and one
existing external wire field, both already carrying "the node's type" as their value — the concept and its
underlying representation are unchanged; only the name used to refer to it changes.

## Filter (existing, `internal/core/filter.go`)

The optional, composable node-selection criteria shared by `arc grep`, `arc subgraph`, and `arc serve`'s
`node_grep` MCP tool (unchanged: a zero-value `Filter{}` matches every node; every populated group is AND'd
together; values within `Types` are OR'd, matching today's `Kinds` semantics exactly).

| Field (before) | Field (after) | Semantics (unchanged) |
|---|---|---|
| `Kinds []string` | `Types []string` | Empty matches every node; otherwise OR'd — a node matches if its `Type` is any listed value |
| `Tags []string` | `Tags []string` (unchanged) | Not touched by this feature |
| `Attrs map[string]string` | `Attrs map[string]string` (unchanged) | Not touched by this feature |
| `AttrPatterns map[string]*regexp.Regexp` | `AttrPatterns map[string]*regexp.Regexp` (unchanged) | Not touched by this feature |

## mcpFilter (existing, `cmd/arc/graph/serve.go`)

The `node_grep` MCP tool's JSON-native filter argument shape.

| Wire field (before) | Wire field (after) | Semantics (unchanged) |
|---|---|---|
| `"kind"` (`Kind []string`) | `"type"` (`Type []string`) | Same OR'd type-restriction criterion as `Filter.Types` above |
| `"tags"` (`Tags []string`) | `"tags"` (unchanged) | Not touched by this feature |
| `"attrs"` (`Attrs map[string]string`) | `"attrs"` (unchanged) | Not touched by this feature |
| `"attrPatterns"` (`AttrPatterns map[string]string`) | `"attrPatterns"` (unchanged) | Not touched by this feature |

No `--json` output payload (`ApplyResult`, `GrepResult`, `SubgraphResult`) changes shape — their field names
already say `type` where relevant (research.md D4).
