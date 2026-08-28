# Quickstart: Validating `node_match`

## Prerequisites

- A built `arc` binary (`go build ./...`) or `go run ./cmd/arc`.
- An initialized graph fixture with a known mix of node types/tags/edges — reuse an existing `cmd/arc/graph/testdata/` fixture already used by `serve_test.go`'s `TestServeNodeGrep*` tests, so expected match counts are already established by that fixture's content.

## Automated validation (primary path)

Run the full suite, including the new `node_match` unit and E2E tests this feature adds:

```bash
go test ./internal/core/... ./internal/app/graph/... ./cmd/arc/graph/... -run 'MatchingFacts|Match' -v
go test ./cmd/arc/graph/... -run TestServeNodeMatch -v
go test ./...
```

Each acceptance scenario in `spec.md` MUST have a passing E2E test in `cmd/arc/graph/serve_test.go`, following the existing `TestServeNodeGrep*`/`connectServeSession` pattern (constitution Principle VIII):

- `TestServeNodeMatchReturnsOneRowPerMatchingFact` — User Story 1, Scenario 1.
- `TestServeNodeMatchEmptyResultForNoMatches` — User Story 1, Scenario 2.
- `TestServeNodeMatchMultiStatementReportsBothFacts` — User Story 2, Scenario 1.
- `TestServeNodeMatchArrayAttributeReportsEachElement` — User Story 2, Scenario 2.
- `TestServeNodeMatchPredicateOnlyStatementMatchesEdgeFact` — User Story 3, Scenario 1.
- `TestServeNodeMatchEmptyFilterReturnsValidationError` — Edge Case: missing/empty filter.

## Manual validation (exploratory)

1. Start the server: `arc serve` (stdio transport) from the fixture graph's root.
2. From an MCP-capable client (or the same in-memory harness `connectServeSession` uses), call:
   ```json
   { "name": "node_match", "arguments": { "filter": { "statements": [ { "predicate": "type", "target": "Source" } ] } } }
   ```
3. Confirm the reply is a markdown table with one row per `Source` node, `property` = `type`, `value` = `Source`.
4. Call again with `{ "filter": { "statements": [] } }` and confirm a tool error naming `filter must contain at least one statement` — not an empty table and not a crash.

## Expected outcome

`node_match` behaves exactly as `contracts/mcp-contract.md` specifies: a required filter, one row per distinct matching fact, empty-but-headered table for zero matches, and a clear validation error for an empty/missing filter — all without ever including a matched node's unrelated content in the reply.
