# Quickstart: Validate MCP Tool Metadata Colocation

Prerequisites: repo checked out at branch `029-mcp-tool-metadata`, Go
toolchain matching `go.mod` (1.26.6).

## 1. Build and unit-test

```sh
go build ./...
go test ./cmd/arc/graph/... -run TestServe -v
```

Expected: builds clean; every existing `TestServe*` E2E scenario in
`serve_test.go` still passes unchanged (research.md D7 — behavior is
untouched, only metadata/file layout changed).

## 2. Confirm colocation (spec SC-001)

```sh
ls cmd/arc/graph/serve_tool_*.go
```

Expected: one file per tool — `serve_tool_node_get.go`,
`serve_tool_node_grep.go`, `serve_tool_subgraph_get.go`,
`serve_tool_context_retrieve.go`, `serve_tool_schema.go`,
`serve_tool_node_match.go`. Open any one file: that tool's args struct,
`mcp.Tool` var, handler, and (if any) render function are all present with
no cross-reference to another tool's registration block.

## 3. Confirm every parameter has a description and an example (spec SC-002)

Add (as part of implementation, TDD-first per constitution Principle VI) a
table-driven unit test in `cmd/arc/graph/serve_test.go` (or a new
`serve_metadata_test.go`) that, for every tool var in a `[]*mcp.Tool` list,
walks `tool.InputSchema.(*jsonschema.Schema).Properties` and asserts via
`github.com/fogfish/it/v2`:
- `prop.Description != ""`
- `len(prop.Examples) >= 1`

Run:

```sh
go test ./cmd/arc/graph/... -run TestToolMetadata -v
```

## 4. Confirm filter arguments carry ≥2 examples (spec SC-003)

Same test file, for each of `node_grep`/`subgraph_get`/`context_retrieve`/
`node_match`, assert `len(tool.InputSchema.Properties["filter"].Examples) >= 2`.

## 5. Confirm server-level guidance is client-visible without a tool call (spec SC-004)

```sh
go run ./cmd/arc serve &
# or, in a test: connect via mcp.NewInMemoryTransports() as serve_test.go's
# connectServeSession helper already does, and inspect
# session.InitializeResult().Instructions
```

Expected: `Instructions` is non-empty and names `schema` as the recommended
first call (existing E2E pattern in `serve_test.go`'s
`connectServeSession`/`TestServeSchema*` tests — extend one assertion to
also check the new content from research.md D5, e.g. that `Instructions`
mentions every one of the six tool names, per data-model.md's Validation
Rules).

## 6. Confirm overlap guidance exists per tool (spec SC-005)

Manual/documentation check: `contracts/mcp-contract.md`'s "Workflow note"
row for `node_grep`, `subgraph_get`, and `context_retrieve` each explicitly
say when to prefer that tool over the others; `node_match`'s note explains
why it exists alongside `node_grep`. Confirm each of the six
`<tool>WorkflowNote` constants appears in `sessionInstructions()`'s
composition (grep `WorkflowNote` across `cmd/arc/graph/serve*.go` and
`sessionInstructions`'s body — every note declared must be referenced).

## 7. Static analysis

```sh
staticcheck ./cmd/arc/graph/...
go vet ./cmd/arc/graph/...
```

Expected: clean — no unused `must`/`withExamples` results, no unreferenced
`WorkflowNote` constants (an unreferenced one signals a tool added to
`sessionInstructions()`'s composition list was forgotten).
