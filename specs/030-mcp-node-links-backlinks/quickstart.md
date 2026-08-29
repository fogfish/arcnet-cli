# Quickstart: Validating `node_links` and `node_backlinks`

## Prerequisites

- A built `arc` binary (`go build ./...`) or `go run ./cmd/arc`.
- An initialized graph fixture with a known mix of outgoing structural edges and inline prose (`[[Target]]`/`[[Target|alias]]`) references — reuse an existing `cmd/arc/graph/testdata/` fixture already used by `serve_test.go`'s `TestServeNodeGet*`/`TestServeSubgraphGet*` tests, so expected relation counts are already established by that fixture's content.

## Automated validation (primary path)

Run the full suite, including the new `node_links`/`node_backlinks` unit and E2E tests this feature adds:

```bash
go test ./internal/app/graph/... -run 'NodeLinks|NodeBacklinks' -v
go test ./cmd/arc/graph/... -run 'TestServeNodeLinks|TestServeNodeBacklinks' -v
go test ./...
```

Each acceptance scenario in `spec.md` MUST have a passing E2E test in `cmd/arc/graph/serve_test.go`, following the existing `TestServeNodeMatch*`/`connectServeSession` pattern (constitution Principle VIII):

- `TestServeNodeLinksReturnsOneRowPerRelation` — User Story 1, Scenario 1.
- `TestServeNodeLinksEmptyResultForNoOutgoingRelations` — User Story 1, Scenario 2.
- `TestServeNodeBacklinksReturnsOneRowPerReferencingRelation` — User Story 2, Scenario 1.
- `TestServeNodeBacklinksEmptyResultForNoIncomingRelations` — User Story 2, Scenario 2.
- `TestServeNodeLinksAndBacklinksAccountForEveryRelation` — User Story 3, Scenario 1.
- `TestServeNodeLinksUnknownIDReturnsNotFoundError` / `TestServeNodeBacklinksUnknownIDReturnsNotFoundError` — Edge Case: unknown id.
- `TestServeNodeLinksIncludesInlineHRefWithEmptyPredicate` — Edge Case: bare inline reference.

## Manual validation (exploratory)

1. Start the server: `arc serve` (stdio transport) from the fixture graph's root.
2. From an MCP-capable client (or the same in-memory harness `connectServeSession` uses), call:
   ```json
   { "name": "node_links", "arguments": { "id": "tls-1-3" } }
   ```
3. Confirm the reply is a markdown table with one row per outgoing relation on `tls-1-3` — both its structural edges and any inline prose references.
4. Call `node_backlinks` with the id of a node known to be referenced by `tls-1-3` (or another fixture node), and confirm one row appears with `source = tls-1-3`, using the same `predicate` value as the matching `node_links` row from step 3.
5. Call either tool with an id that does not exist in the fixture graph, and confirm a tool error naming `no node found with basename <id>` — not an empty table.

## Expected outcome

`node_links`/`node_backlinks` behave exactly as `contracts/mcp-contract.md` specifies: an id-only lookup, one row per relation (structural or inline) in the requested direction, empty-but-headered table when a valid node has none, a clear not-found error for an unknown id, and full agreement between a relation's `node_links` row on its source and its `node_backlinks` row on its target.
