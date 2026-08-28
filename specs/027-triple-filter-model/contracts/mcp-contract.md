# MCP Contract: triple-shaped `filter` argument (replaces the type/tags/attrs/attrPatterns shape)

This is a breaking amendment to `specs/008-arc-serve-mcp/contracts/mcp-contract.md` and `specs/025-context-retrieve-tool/contracts/mcp-contract.md`'s `filter` argument shape, affecting `node_grep`, `subgraph_get`, and `context_retrieve`. Per spec.md's explicit scope, this is **not** a compatible evolution: the old `{ "type": [...], "tags": [...], "attrs": {...}, "attrPatterns": {...} }` shape is removed outright, with no alias, no dual acceptance, and no version negotiation (ADR 003 rule 1 is unaffected — the handler stays a thin wrapper; only the JSON it decodes changes).

## New `filter` shape (all three tools)

```json
{
  "statements": [
    { "predicate": "cites" },
    { "predicate": "type", "target": "Source" },
    { "predicate": "tags", "target": ["cryptography", "protocols"] },
    { "predicate": "title", "targetPattern": "^TLS 1\\.3" }
  ]
}
```

- `filter` and `statements` are both optional; an absent `filter`, or a `filter` with an empty/absent `statements` list, matches every node — unchanged posture from today's absent-filter behavior.
- Each entry in `statements` is one triple statement, all fields optional:
  - `source`, `predicate`, `target`: each accepts either a single string or a JSON array of strings (OR-of-values against that position — research.md D1/D7). Omitted = wildcard (that position is unconstrained).
  - `sourcePattern`, `predicatePattern`, `targetPattern`: each accepts either a single regexp string or an array of regexp strings (OR-of-patterns against that position), for the `--attr name~=pattern` case. A statement may combine e.g. both `target` and `targetPattern` on the same position (OR'd together, per `Matcher`'s definition) though the common case sets only one.
- Statements are ANDed together (spec FR-001) — every statement must be satisfied by *some* fact on the candidate node for flat inclusion (`node_grep`, and `subgraph_get`/`context_retrieve`'s final result narrowing), or, for a statement with only `predicate` set (`source`/`target` both omitted), the connection followed during `subgraph_get`/`context_retrieve`'s neighbor expansion (research.md D3).
- Field naming avoids RDF/set-theory jargon per spec FR-016: `source`/`predicate`/`target` mirror `core.Link`'s own field names (`Predicate`/`Target`), never `s`/`p`/`o` or `subject`/`object`.

## `node_grep`

- **Input schema** (delta only): `filter` is now `{ "statements": [ ... ] }` as above, replacing the old four-field shape. No other field of `nodeGrepArgs` changes.
- Every statement narrows which nodes are scanned exactly as `Filter.Match` already does today (spec FR-005) — `node_grep` never traverses, so every statement (including a bare `{"predicate": "cites"}`) applies to flat inclusion, unlike `subgraph_get`/`context_retrieve` below (research.md D3: `Traversal()`/`Narrowing()` only diverge where BFS exists).

## `subgraph_get`

- **Input schema** (new field — research.md D8): `subgraphGetArgs` gains `filter`, following the identical optional-pointer pattern `nodeGrepArgs.Filter`/`contextRetrieveArgs.Filter` already use. `id`/`depth` are unchanged.
  ```json
  { "id": "string (required)", "depth": "integer (optional, default 1)", "filter": { "statements": [ ... ] } }
  ```
- A statement with only `predicate` set scopes BFS neighbor expansion (both outgoing and incoming/backlink direction) to connections whose `Link.Predicate` matches — the seed's own connections included. A statement with `source` and/or `target` also set narrows the final result set instead (does not affect what gets reached). Both kinds may appear together in one `statements` list.
- Omitting `filter` entirely (or sending `{}`/`{"statements": []}`) preserves today's exact behavior: every connection followed, no narrowing — identical output to before this feature for the same `id`/`depth`.

## `context_retrieve`

- **Input schema** (delta only): `filter` is now `{ "statements": [ ... ] }`, replacing the old shape. `query`/`limit` are unchanged.
- The filter's *full, unsplit* statement list continues to narrow the upfront content-match/attribute-match universe (`filterIncluded`/`pathIncluded`) exactly as `Filter.Match` does today. Once neighbor expansion pools candidates, a statement with only `predicate` set additionally scopes which connections that expansion follows, and is excluded from re-narrowing the resulting (non-direct-match) candidates — research.md D3's `Traversal()`/`Narrowing()` split, applied the same way `subgraph_get` applies it.

## Error replies (unchanged mechanism, new trigger)

- A syntactically invalid regexp in any `*Pattern` field returns the existing `service.ErrInvalidFilterPattern` contract (`IsError: true`, human-readable message) — same error type as today's `attrPatterns` validation, now triggered by any of the six pattern-capable fields instead of just `attrPatterns`.
- An unrecognized JSON field in `filter` (e.g. a client still sending the old `"type"`/`"tags"`/`"attrs"`/`"attrPatterns"` shape) is rejected by MCP input-schema validation before the handler runs — this is the intended, unguarded break; no fallback decoding path is added.

## Session/tool-description advertisement (research.md D9)

- `schemaAdvertisement` (`InitializeResult.Instructions`) gains a clause naming predicate-scoped traversal as available.
- `node_grep`, `subgraph_get`, `context_retrieve`'s `mcp.Tool.Description` each gain a short clause pointing at the `filter.statements` shape, so a client that calls `schema()` first (already the advertised recommended first call) can construct a triple-shaped filter without trial and error.

## Operational logging (unchanged contract shape, spec FR-013 continuity)

```text
serve: node_grep pattern="handshake" ok
serve: subgraph_get id="TLS" depth=1 ok
serve: context_retrieve query="TLS handshake" filter=true limit=10 ok
```

No change to the logged-argument shape (`logCall`'s existing `fmt.Sprintf` calls do not print filter contents today and continue not to).
