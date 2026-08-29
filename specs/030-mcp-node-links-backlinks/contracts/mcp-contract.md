# MCP Contract: `node_links` and `node_backlinks`

## Tool registration

```go
mcp.AddTool(server, nodeLinksTool, nodeLinksHandler(dir))
mcp.AddTool(server, nodeBacklinksTool, nodeBacklinksHandler(dir))
```

Both are read-only (`ReadOnlyHint: true`) and included in `arc serve`'s fixed tool set alongside `node_get`/`node_grep`/`subgraph_get`/`context_retrieve`/`schema`/`node_match`. `arc serve` exposes only read-only tools today — there is no `--readonly` flag or a separate mutating tool set to opt out of.

## `node_links`

### Input

```json
{ "id": "tls-1-3" }
```

- `id` is **required**: the target node's basename, using the identical convention `node_get`'s `id` argument already uses.

### Output

A markdown table, one row per outgoing relation on that node — both its structural `Edges` and any inline prose `HRefs` (spec 030 Assumptions) — header always present:

```text
| predicate | target |
|---|---|
| cites | rescorla-tls13 |
|  | some-other-node |
```

- A relation with no explicit predicate (a bare inline reference) renders with an empty `predicate` cell — it is still one row, never dropped (spec 030 Edge Cases).
- Zero outgoing relations → header row only, never an error (spec FR-006).
- A node with a self-referencing relation (target equals the queried node's own id) → one ordinary row, `target` equal to `id`.
- Duplicate relations (same predicate/target repeated, or several different predicates between the same pair) → one row per occurrence, never deduplicated (spec FR-007).

## `node_backlinks`

### Input

```json
{ "id": "rescorla-tls13" }
```

- `id` is **required**: the target node's basename, identical convention to `node_links`/`node_get`.

### Output

A markdown table, one row per incoming relation whose target is that node — from either the referencing node's `Edges` or its `HRefs` — header always present:

```text
| source | predicate |
|---|---|
| tls-1-3 | cites |
| some-other-node |  |
```

- Same empty-predicate, self-reference, and no-deduplication rules as `node_links`, applied to the reverse direction.
- Zero incoming relations → header row only, never an error (spec FR-006).

### Error responses (both tools)

| Condition | Response |
|---|---|
| `id` matches no node in the graph | Tool error: `no node found with basename <id>` (`service.ErrSeedNotFound`, identical to `node_get`'s existing behavior) |
| Graph root is not an initialized graph | Tool error: `service.ErrNotAGraph` (identical to every other tool) |

## Cross-tool consistency (spec SC-003)

For any relation in the graph, the source node's `node_links` result and the target node's `node_backlinks` result agree: the relation appears in exactly one row of each, with the same `predicate` value. This holds because both tools are built from the same `nodeRelations` helper (data-model.md) — `node_links` calls it directly on the queried node, and `node_backlinks`'s reverse index is built by calling it on every node in the graph.

## Session-level advertisement

`sessionInstructionsPurpose` (`cmd/arc/graph/serve.go`) is updated from "six tools" to "eight tools," naming `node_links`/`node_backlinks` alongside the existing six. Each tool's own `nodeLinksWorkflowNote`/`nodeBacklinksWorkflowNote` constant (colocated in its own `serve_tool_*.go` file, per spec 029's convention) is appended to `sessionInstructions()`, describing when to prefer these tools over `node_get` (full content) or `node_match` (filter-driven fact search).
