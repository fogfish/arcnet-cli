# MCP Contract: `context_retrieve` (addition to `arc serve`)

This feature adds one tool, `context_retrieve`, to the existing `arc serve` MCP server (specs/008-arc-serve-mcp) alongside `node_get`, `node_grep`, and `subgraph_get`. `arc serve`'s own startup behavior, transport selection (stdio default, `--http <addr>` opt-in), and per-call stderr logging format are unchanged — see specs/008-arc-serve-mcp/contracts/mcp-contract.md for those.

## `context_retrieve`

- **Description**: Assemble the full content of every node relevant to a free-text query in one call — content match, attribute match, and neighbor expansion combined, ranked, deduplicated, and truncated to `limit`.
- **Input schema**:
  ```json
  {
    "query": "string (required) — free text, matched literally and case-insensitively",
    "filter": {
      "type":         ["string", "..."],
      "tags":         ["string", "..."],
      "attrs":        { "<name>": "<value>" },
      "attrPatterns": { "<name>": "<regexp>" }
    },
    "limit": "integer (optional, default 10)"
  }
  ```
  `filter` and every one of its fields are optional; an absent/empty filter matches every node (VISION.md Filtering — MCP filter object). `limit`, once resolved, must be a positive integer.
- **Annotations**: `readOnlyHint: true`.
- **Success reply**: one `TextContent`, `Text` = `string(core.RenderPatch(result.Patch))` — a synthesized patch-exchange document (document id `context:<slugified-query>@<timestamp>`, title `Context: <query>`) containing the ranked, truncated candidate nodes, byte-shape-identical to `subgraph_get`'s own reply construction. Zero candidates found → a valid, empty patch document (no node sections), not an error (spec FR-008).
- **Error reply** (`IsError: true`, content = human-readable message):
  - `limit`, once resolved, is not a positive integer (spec FR-007) — a non-integer JSON value for `limit` is rejected by MCP input-schema validation before the handler runs.
  - `filter.attrPatterns` value is not a syntactically valid regexp (existing `ErrInvalidFilterPattern` contract, unchanged from `node_grep`/`subgraph_get`).

## Ranking and truncation (spec FR-002 through FR-006, Clarifications 2026-08-24)

1. Run three passes over the candidate universe (narrowed to `filter`-matching nodes throughout): content match (literal, case-insensitive, via the same engine `node_grep` uses), attribute match (literal, case-insensitive, against every front-matter attribute value), neighbor expansion (one hop, both directions, from every node found by the first two passes — same traversal and size caps as `subgraph_get`).
2. Deduplicate by node id.
3. Rank: direct matches (content or attribute) before neighbor-only matches; within a tier, by connectivity (total edge/link count) descending; remaining ties by id ascending.
4. Truncate to `limit`.

## Operational logging (unchanged contract, spec FR-013)

```text
serve: context_retrieve query="TLS handshake" filter=true limit=10 ok
serve: context_retrieve query="no such topic" filter=false limit=10 ok (0 nodes)
serve: context_retrieve query="TLS" filter=false limit=0 error: limit 0 must be a positive integer
```
