# MCP Contract: `arc serve`

## Command usage

```text
arc serve [--http <addr>]
```

- No positional arguments.
- `--http <addr>` (string, default `""` — stdio). `<addr>` is a `[host]:port` address spec (research.md D5): a bare port or `:port` (no host) binds `127.0.0.1` only; an explicit host (`0.0.0.0:8080`, `192.168.1.10:8080`) binds exactly that host.
- All other DS-03 persistent root flags apply as usual (`--quiet`/`--verbose`/`--json`/`--color`), but are not meaningful to this command's own output: `arc serve` produces no `stdout` document of its own (its "output" is MCP tool-call content) and no color-styled text.

## Startup behavior

| Condition | Result |
|---|---|
| Target directory is not an initialized graph | `RunE` returns `service.ErrNotAGraph` immediately (spec FR-004); no transport is started |
| `--http`'s address is not a syntactically valid `[host]:port`, or its port is already in use | `RunE` returns a clear error immediately (spec FR-005); no fallback to stdio |
| Preflight passes | The three tools are registered on an `mcp.Server`, then `server.Run(ctx, &mcp.StdioTransport{})` (no `--http`) or an HTTP listener wrapping `mcp.NewStreamableHTTPHandler` (with `--http`) blocks until `ctx` is canceled (SIGINT/SIGTERM) |

## Tools

### `node_get`

- **Description**: Fetch a node's full content by id.
- **Input schema**:
  ```json
  { "id": "string (required) — node basename" }
  ```
- **Annotations**: `readOnlyHint: true`.
- **Success reply**: one `TextContent`, `Text` = `string(core.RenderNode(node))` — front-matter (fenced YAML) then body, byte-identical to the node's own on-disk rendering.
- **Error reply** (`IsError: true`, content = human-readable message): `id` does not identify any existing node (spec FR-007).

### `node_grep`

- **Description**: Search node content for lines matching a regexp pattern, optionally narrowed by a filter.
- **Input schema**:
  ```json
  {
    "pattern": "string (required) — regexp",
    "filter": {
      "kind": ["string", "..."],
      "tags": ["string", "..."],
      "attrs": { "<name>": "<value>" },
      "attrPatterns": { "<name>": "<regexp>" }
    }
  }
  ```
  `filter` and every one of its fields are optional; an absent/empty filter matches every node (VISION.md Filtering — MCP filter object).
- **Annotations**: `readOnlyHint: true`.
- **Success reply**: one `TextContent`, `Text` = a markdown table:
  ```markdown
  | id | kind | line | snippet |
  |---|---|---|---|
  | TLS-1-3 | source | 42 | ...negotiates **TLS 1.3** during... |
  ```
  Zero matches → the table header only (no rows), not an error (spec FR-009).
- **Error reply**: `pattern` is not a syntactically valid regexp (spec FR-010); a `filter.attrPatterns` value is not a syntactically valid regexp.

### `subgraph_get`

- **Description**: Return the fully-resolved subgraph rooted at a node, to a given hop depth.
- **Input schema**:
  ```json
  { "id": "string (required) — seed node basename", "depth": "integer (optional, default 1)" }
  ```
- **Annotations**: `readOnlyHint: true`.
- **Success reply**: one `TextContent`, `Text` = `string(core.RenderPatch(result.Patch))` — the identical bytes `arc subgraph`'s own human-mode stdout produces for the same seed/depth/graph state, including the synthesized document manifest.
- **Error reply**: `id` does not identify any existing node (spec FR-013); `depth`, once present, is negative (spec FR-013) — a non-integer `depth` is rejected by MCP input-schema validation before the handler runs.

## Operational logging (spec FR-019)

Every tool call, regardless of outcome, produces exactly one line on the server process's `stderr`:

```text
serve: node_get id=TLS-1-3 ok
serve: node_grep pattern="TLS \d\.\d" ok (7 matches)
serve: subgraph_get id=missing-node error: no node found with basename missing-node
```

## Transport equivalence (spec SC-007)

The same tool call, same arguments, same graph state, returns byte-identical `TextContent` whether made over the default stdio transport or over `--http`'s Streamable HTTP/SSE transport — both transports front the identical registered `mcp.Server` and tool handlers; only the wire framing differs.
