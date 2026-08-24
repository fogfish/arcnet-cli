# Quickstart: `context_retrieve`

Validates spec.md's three user stories end-to-end against a real local graph, served by the already-existing `arc serve`. `context_retrieve` never writes to the graph or its git history (spec FR-010) — every scenario below can be re-run repeatedly with identical results (spec SC-006).

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init`, with `Transport Layer Security` (an `Entity`) linked to `rescorla-2026-tls13` (a `Source` whose text mentions "TLS 1.3") — the same fixture `specs/007-arc-subgraph/quickstart.md` and `specs/008-arc-serve-mcp/quickstart.md` use
- An MCP client capable of speaking stdio — the official [MCP Inspector](https://modelcontextprotocol.io) (`npx @modelcontextprotocol/inspector arc serve`) or a short Go program using `github.com/modelcontextprotocol/go-sdk/mcp`'s `Client`

## Scenario 1 — Assemble working context from a topic in one call (spec.md User Story 1)

```sh
$ arc serve
```

Connect a client and call:

```json
{ "name": "context_retrieve", "arguments": { "query": "TLS 1.3" } }
```

**Expected outcome**: one text content block, a patch-exchange document containing the full node object for `rescorla-2026-tls13` (content match: its text mentions "TLS 1.3") plus the full node object for `Transport Layer Security` (neighbor expansion: directly linked to the match) — up to the default limit of 10, no duplicates.

Calling with a query that matches nothing:

```json
{ "name": "context_retrieve", "arguments": { "query": "quantum key distribution" } }
```

**Expected outcome**: a valid, empty patch document — no node sections, no error.

## Scenario 2 — Narrow retrieval to a relevant slice of the graph (spec.md User Story 2)

```json
{ "name": "context_retrieve", "arguments": { "query": "TLS", "filter": { "type": ["Source"] } } }
```

**Expected outcome**: only `rescorla-2026-tls13` (a `Source`) appears in the result — `Transport Layer Security` (an `Entity`, reachable only via neighbor expansion) is excluded because it does not satisfy the filter, even though it would appear in Scenario 1's unfiltered call.

## Scenario 3 — Control how much context comes back (spec.md User Story 3)

```json
{ "name": "context_retrieve", "arguments": { "query": "TLS", "limit": 1 } }
```

**Expected outcome**: exactly one node object in the result — the highest-ranked candidate (a direct content/attribute match ranks above a neighbor-only match; see contracts/mcp-contract.md's ranking rule).

```json
{ "name": "context_retrieve", "arguments": { "query": "TLS", "limit": 0 } }
```

**Expected outcome**: `isError: true`, content text `limit 0 must be a positive integer` — the server itself keeps running and answers the next call normally (spec FR-012).

## Verifying read-only behavior (spec SC-004)

```sh
$ git -C <graph-root> status --porcelain   # before
$ # ... run every scenario above ...
$ git -C <graph-root> status --porcelain   # after — must be identical (empty)
```
