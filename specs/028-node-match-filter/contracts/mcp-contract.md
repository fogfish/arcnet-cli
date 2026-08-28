# MCP Contract: `node_match`

## Tool registration

```go
mcp.AddTool(server, &mcp.Tool{
	Name:        "node_match",
	Description: "List every distinct fact ({id, property, value}) on nodes that fully satisfy a required filter.statements triple filter (see schema) — evidence of why each node matched, not the node's full content.",
	InputSchema: nodeMatchSchema,
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}, nodeMatchHandler(dir, cfgFile.Grep))
```

`node_match` is read-only (`ReadOnlyHint: true`) and included in `arc serve --readonly`'s tool set alongside `node_get`/`node_grep`/`subgraph_get`/`context_retrieve`.

## Input

```json
{
  "filter": {
    "statements": [
      { "predicate": "type", "target": "Source" },
      { "predicate": "tags", "target": ["cryptography", "protocols"] }
    ]
  }
}
```

- `filter` is **required** (unlike `node_grep`/`subgraph_get`/`context_retrieve`'s optional `filter`) and MUST contain at least one entry in `statements`.
- Each statement uses the exact wire shape already defined in `specs/VISION.md`'s "MCP filter object" section: `source`/`predicate`/`target` (plus `sourcePattern`/`predicatePattern`/`targetPattern`), each a single string or an array of strings (OR-of-values), any omitted position a wildcard. Statements are ANDed together — a node must satisfy every statement to appear in the result at all.
- A predicate-only statement (source/target both omitted) behaves as an ordinary narrowing statement here — `node_match` performs no relation traversal, so there is no traversal/narrowing split to apply (research.md D4, spec FR-006).

### Error responses

| Condition | Response |
|---|---|
| `filter` omitted, or `filter.statements` empty | Tool error: `filter must contain at least one statement` (`service.ErrEmptyFilter`) |
| A `*Pattern` field is not a valid regexp | Tool error: `service.ErrInvalidFilterPattern` (identical to `node_grep`/`subgraph_get`/`context_retrieve`'s existing behavior) |
| Graph root is not an initialized graph | Tool error: `service.ErrNotAGraph` (identical to every other tool) |

## Output

A markdown table, one row per matching fact, header always present:

```text
| id | property | value |
|---|---|---|
| rescorla-tls13 | type | Source |
| rescorla-tls13 | tags | cryptography |
```

- Zero matching nodes → header row only (never an error) — spec FR-007/SC-002.
- A node satisfying the filter via two distinct facts produces two rows, both carrying the same `id` — spec User Story 2.
- Row order: by `id`, then `property`, then `value` (research.md D3) — deterministic, not meaningful ranking.
- Unlike `node_get`/`subgraph_get`/`context_retrieve`, no node body/front-matter/edges beyond the matching fact(s) are ever included in the reply (spec FR-008).

## Session-level advertisement

`schemaAdvertisement` (`cmd/arc/graph/serve.go`) gains one clause naming `node_match` alongside `node_grep`/`subgraph_get`/`context_retrieve` as a `filter.statements`-accepting tool, noting that — unlike the other three — its filter is required.
