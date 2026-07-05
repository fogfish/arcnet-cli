# Quickstart: `arc serve`

Validates spec.md's four user stories end-to-end against a real local graph. `arc serve` needs no network access when run without `--http`, and none of its three tools ever write to the graph or its git history (spec FR-015) — every scenario below can be re-run repeatedly with identical results.

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`), or `go run ./cmd/arc` throughout
- A graph created by `arc init` with at least one document ingested via `arc apply`, containing a seed node (`Transport Layer Security`) linked to a source (`rescorla-2026-tls13`) — the same fixture `specs/007-arc-subgraph/quickstart.md` uses
- An MCP client capable of speaking stdio or Streamable HTTP — either the official [MCP Inspector](https://modelcontextprotocol.io) (`npx @modelcontextprotocol/inspector`) or a short Go program using `github.com/modelcontextprotocol/go-sdk/mcp`'s `Client`

## Scenario 1 — Fetch a node's full context by id (spec.md User Story 1)

```sh
$ arc serve
```

The server starts and blocks, serving over stdio. Connect a client (e.g. MCP Inspector: `npx @modelcontextprotocol/inspector arc serve`) and call:

```json
{ "name": "node_get", "arguments": { "id": "Transport Layer Security" } }
```

**Expected outcome**: one text content block, markdown:

```markdown
```yaml
id: Transport Layer Security
category: form structure attribute process
```

TLS is the successor to SSL.
```

Calling `node_get` with an id that doesn't exist:

```json
{ "name": "node_get", "arguments": { "id": "No Such Node" } }
```

**Expected outcome**: `isError: true`, content text `no node found with basename No Such Node` — the server itself keeps running and answers the next call normally.

## Scenario 2 — Search node content to find relevant nodes (spec.md User Story 2)

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS 1\\.3" } }
```

**Expected outcome**: a markdown table with one row per matching line, e.g.:

```markdown
| id | kind | line | snippet |
|---|---|---|---|
| rescorla-2026-tls13 | source | 1 | TLS 1.3 is the latest version of the Transport Layer Security protocol. |
```

Narrowed by a filter:

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "kind": ["source"] } } }
```

**Expected outcome**: only matches within `source`-kind nodes appear; a pattern matching nothing returns the table header with zero rows, not an error.

## Scenario 3 — Expand a node into its neighborhood in one call (spec.md User Story 3)

```json
{ "name": "subgraph_get", "arguments": { "id": "Transport Layer Security" } }
```

**Expected outcome**: markdown identical to `arc subgraph "Transport Layer Security"`'s own stdout (see `specs/007-arc-subgraph/quickstart.md` Scenario 1) — the seed plus its direct neighbors, one patch-exchange document.

```json
{ "name": "subgraph_get", "arguments": { "id": "Transport Layer Security", "depth": 2 } }
```

**Expected outcome**: the wider, 2-hop neighborhood, matching `arc subgraph ... --depth 2`'s own output for the same graph state.

## Scenario 4 — Reach the server over a network connection (spec.md User Story 4)

```sh
$ arc serve --http :8080
```

**Expected outcome**: binds `127.0.0.1:8080` only (no host given — spec Clarifications); connect an MCP client configured for Streamable HTTP at `http://127.0.0.1:8080` and confirm the same three tool calls above return byte-identical results.

```sh
$ arc serve --http 0.0.0.0:8080
```

**Expected outcome**: binds all interfaces — reachable from another host on the network, per the explicit host given.

```sh
$ arc serve --http :8080 &
$ arc serve --http :8080
❌ address 127.0.0.1:8080 already in use. Run `arc help serve` for guidance.
$ echo $?
1
```

**Expected outcome**: the second invocation refuses to start rather than silently falling back to stdio.

## Verifying read-only behavior (spec SC-006)

```sh
$ git status --short > /tmp/before.txt
$ arc serve &
# ... exercise all three tools via a client ...
$ kill %1
$ git status --short > /tmp/after.txt
$ diff /tmp/before.txt /tmp/after.txt
# empty — arc serve changed nothing, no matter how many tool calls were made
```

## Verifying operational logging (spec FR-019/SC-008)

```sh
$ arc serve 2> /tmp/serve.log &
# ... call node_get, node_grep, subgraph_get, including one unknown id ...
$ kill %1
$ cat /tmp/serve.log
serve: node_get id="Transport Layer Security" ok
serve: node_grep pattern="TLS 1\.3" ok (1 matches)
serve: subgraph_get id="Transport Layer Security" ok
serve: node_get id="No Such Node" error: no node found with basename No Such Node
```

**Expected outcome**: every call, successful or not, produced exactly one line; an operator can reconstruct the whole session from `stderr` alone.
