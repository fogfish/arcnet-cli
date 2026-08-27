# MCP Contract: `schema` (addition to `arc serve`)

This feature adds one tool, `schema`, to the existing `arc serve` MCP server (specs/008-arc-serve-mcp, specs/025-context-retrieve-tool) alongside `node_get`, `node_grep`, `subgraph_get`, and `context_retrieve`. `arc serve`'s own startup behavior, transport selection (stdio default, `--http <addr>` opt-in), and per-call stderr logging format are unchanged.

## `schema`

- **Description**: Return the graph's full ontology — every currently defined class and predicate, with descriptions — so a client knows what vocabulary is available before reading or writing the graph. Recommended as the first tool call of a session.
- **Input schema**: none — the tool takes no arguments (spec FR-006).
- **Annotations**: `readOnlyHint: true`.
- **Success reply**: one `TextContent`, `Text` = `renderSchema(index)` (data-model.md), a markdown document with two sections:
  - `## Predicates` — one bullet per predicate, name and description only, sorted by name: `- **name**: description`.
  - `## Classes` — one `### name` subsection per class, sorted by name, its description as prose, followed by `Required: ...` and `Optional: ...` lines naming its required/optional predicates (`(none)` when a list is empty).

  An empty `index` (no `_schema/` present — the same fallback `resolveIndexOrDefault` already applies for `node_get`/`subgraph_get`) renders both sections present with no entries under them, never an error — consistent with spec's edge case "must still return the full built-in vocabulary, not an empty result" for the ordinary case, and with the other three tools' existing behavior against a bare, non-schema-populated directory.
- **Error reply**: none — `schema` cannot fail. It renders an already-resolved, in-memory `core.Index` value with no I/O, no argument decoding, and no user-supplied input to reject.

## Example reply

```markdown
## Predicates

- **category**: Free-text classification tags for a node.
- **linkTo**: A directed reference from one node to another.

## Classes

### Entity

A first-class thing the graph describes.

Required: category
Optional: linkTo
```

## Session advertisement (spec FR-008, FR-009)

`buildServer`'s `mcp.NewServer` call passes a non-nil `&mcp.ServerOptions{Instructions: ...}` naming `schema` and recommending it as the first call. Every connecting client receives this string in `InitializeResult.Instructions`, before any tool is called — this is the *session-level* contract; it is independent of `schema`'s own `mcp.Tool.Description`, which remains separately visible via `tools/list`.

## Operational logging (unchanged contract, spec FR-013 of specs/008-arc-serve-mcp)

```text
serve: schema  ok
```

`schema` has no arguments to report, so `logCall`'s `args` parameter is an empty string, matching the format `logCall` already produces for any tool with nothing argument-specific to log.
