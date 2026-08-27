# Quickstart: `schema`

Validates spec.md's three user stories end-to-end against a real local graph, served by the already-existing `arc serve`. `schema` never writes to the graph or its git history (spec FR-007) — every scenario below can be re-run repeatedly with identical results.

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init` (built-in vocabulary only is enough for Scenario 1; Scenario 2 also needs a project-specific class/predicate added, e.g. via `arc apply schema`)
- An MCP client capable of speaking stdio — the official [MCP Inspector](https://modelcontextprotocol.io) (`npx @modelcontextprotocol/inspector arc serve`) or a short Go program using `github.com/modelcontextprotocol/go-sdk/mcp`'s `Client`

## Scenario 1 — Discover the built-in ontology in one call (spec.md User Story 1)

```sh
$ arc serve
```

Connect a client and call:

```json
{ "name": "schema", "arguments": {} }
```

**Expected outcome**: one text content block, listing every built-in predicate (name + description only) under `## Predicates` and every built-in class (description, then required/optional predicate names) under `## Classes` — see contracts/mcp-contract.md for the exact document shape. No `arguments` beyond `{}` are accepted or required.

## Scenario 2 — Ontology reflects project-specific additions (spec.md User Story 1, Acceptance Scenario 2)

Add a project-specific class or predicate to the graph (e.g. `arc apply schema <patch.md>`), then, on a *newly connected* session:

```json
{ "name": "schema", "arguments": {} }
```

**Expected outcome**: the reply includes the built-in vocabulary from Scenario 1 plus the newly added class/predicate, since `arc serve`'s server construction resolves the schema index once per server lifetime (research.md D1) — a session started after the addition sees it; a session already connected before the addition does not, without reconnecting.

## Scenario 3 — Session advertises `schema` as the first call (spec.md User Story 2)

Inspect the client's initialize response, before calling any tool:

```text
InitializeResult.instructions
```

**Expected outcome**: a non-empty string naming `schema` and recommending it as the first call of the session — present on every connection, not only the first one a given client ever makes (spec FR-009). This is separate from `schema`'s own tool description, visible via `tools/list`.

## Verifying read-only behavior

```sh
$ git -C <graph-root> status --porcelain   # before
$ # ... run every scenario above ...
$ git -C <graph-root> status --porcelain   # after — must be identical (empty)
```
