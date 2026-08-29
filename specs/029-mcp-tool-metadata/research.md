# Phase 0 Research: MCP Tool Metadata Colocation

**Input**: [spec.md](spec.md), the user's `/speckit-plan` instruction (split `cmd/arc/graph/serve.go` into one file per tool plus a shared `serve.go`; declare each tool's `mcp.Tool` as a constant colocated with its parameters/types; add `jsonschema` tags; draft descriptions an LLM client can act on).

All decisions below were made by reading the actual current implementation (`cmd/arc/graph/serve.go`, 615 lines, package `graph`) rather than assuming SDK behavior; evidence is cited per decision.

## D1 — File split boundary

**Decision**: Split `cmd/arc/graph/serve.go` into:

- `serve_tool_node_get.go`, `serve_tool_node_grep.go`, `serve_tool_subgraph_get.go`, `serve_tool_context_retrieve.go`, `serve_tool_schema.go`, `serve_tool_node_match.go` — one file per tool, each holding that tool's args struct, its `mcp.Tool` value, its handler factory, and any render function/workflow note that exists only for that tool (e.g. `renderMatchTable` moves into `serve_tool_node_grep.go`, `renderFactTable`/`renderSchema`/`joinOrNone` move into their respective tool files).
- `serve.go` keeps everything genuinely cross-tool: `buildServer`, `NewServeCmd`, `logCall`, `resolveHTTPAddr`, `serveImplName`/`serveImplVersion`, the filter wire-shape (`stringOrArray`, `mcpStatement`, `mcpFilter`, `toMatcher`, `toCoreFilter`, `stringOrArraySchema`), the schema-generation helpers (`inputSchemaFor`, `must`, `withExamples`), and the session-instructions composition (`sessionInstructions`, replacing the current single `schemaAdvertisement` constant — see D3).

**Rationale**: This is what the user explicitly asked for, and it matches spec FR-001/FR-011 (one colocated block per tool) exactly — today a tool's `Description` (in `buildServer`, line ~473+) and its args struct (defined 90-250 lines earlier, e.g. `nodeGrepArgs` at line 246 vs. its registration at line 483) are already separated by hundreds of lines in one file; per-tool files make that separation impossible by construction.

**Alternatives considered**: Keep one file but reorder it so each tool's struct/handler/registration are adjacent (rejected — `buildServer` fundamentally needs one function that mounts every tool onto one `*mcp.Server`, so registration itself can't be colocated with a single tool without either duplicating server setup per file or splitting registration out, which is what per-file `mcp.AddTool` calls in each tool's own `init`-time var effectively does anyway).

## D2 — "`mcp.Tool` as a constant"

**Decision**: Use a package-level `var`, not Go's `const` keyword, for each tool's `mcp.Tool` value, e.g. in `serve_tool_node_get.go`:

```go
var nodeGetTool = &mcp.Tool{
	Name:        "node_get",
	Description: "...",
	InputSchema: must(nodeGetInputSchema()),
	Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
}
```

`buildServer` then becomes `mcp.AddTool(server, nodeGetTool, nodeGetHandler(dir, index))` for each tool — no schema-derivation call left inline in `buildServer`.

**Rationale**: `mcp.Tool` (`protocol.go:1295`) contains pointer and interface-typed fields (`Annotations *ToolAnnotations`, `InputSchema any`) that are not constant expressions under the Go spec — `const nodeGetTool = mcp.Tool{...}` does not compile. A package-level `var` initialized once at package load is the closest available equivalent to the user's intent ("one immutable, colocated definition per tool") and is the idiomatic Go substitute (the same pattern the stdlib uses for `regexp.MustCompile`-style package vars). `must[T any](v T, err error) T` (new, in `serve.go`) panics on a schema-derivation error, which is appropriate here: such an error can only come from a malformed struct tag on the maintainer's own args type, i.e. a programming error caught the first time the package is loaded (including in every test run), never a runtime/user-input condition.

**Evidence**: `mcp.Tool` struct definition confirms `InputSchema any` / `Annotations *ToolAnnotations` (`go-sdk@v1.6.1/mcp/protocol.go:1295-1339`).

**Alternatives considered**: Keep `InputSchema` derivation inline inside `buildServer` as today (rejected — this is exactly the split D1 exists to remove: the tool's contract would again be defined half in the tool file, half in `buildServer`). Return an `error` from a per-tool `newXTool() (*mcp.Tool, error)` constructor called by `buildServer` instead of panicking (considered and rejected as unnecessary complexity per Principle V/YAGNI — the only possible error is a malformed static struct tag, not a runtime condition, so every existing caller would just do `must(newXTool())` anyway; panicking once at package-var-initialization time surfaces the same problem earlier, at `go build`/`go vet` composite-literal-eval time via a table-driven "every tool var is non-nil" unit test, without threading an error return through `buildServer` for a condition that cannot occur at runtue).

## D3 — Where per-parameter examples come from

**Decision**: Add a `withExamples(s *jsonschema.Schema, field string, examples ...any) *jsonschema.Schema` helper in `serve.go`. Each tool file's own `<tool>InputSchema() (*jsonschema.Schema, error)` function calls `inputSchemaFor[<tool>Args]()` and then chains `withExamples` per field before returning, e.g.:

```go
func nodeGrepInputSchema() (*jsonschema.Schema, error) {
	s, err := inputSchemaFor[nodeGrepArgs]()
	if err != nil {
		return nil, err
	}
	withExamples(s, "pattern", "TODO", `func\s+\w+\(`)
	withExamples(s, "filter", filterExampleContentNarrow, filterExamplePredicateOnly)
	return s, nil
}
```

**Rationale**: The `jsonschema` struct tag is parsed by `github.com/google/jsonschema-go` (`jsonschema/infer.go:329`, confirmed by reading the source directly) as `fs.Description = tag` — the *entire* tag string becomes the field's `Description`; there is no tag syntax for `Examples`/`Enum`/`Default`. `jsonschema.Schema` (`jsonschema/schema.go:62-74`) does have an `Examples []any` field (and `Enum []any`), but it can only be populated by mutating the `*Schema` after reflection, exactly the way `stringOrArraySchema()` already overrides the schema for `stringOrArray`-typed fields today (`serve.go:149-156`, precedent already in the codebase, wired through `inputSchemaFor`'s `TypeSchemas` option). `withExamples` is the same category of post-processing, applied per named property rather than per type.

**Evidence**: `jsonschema/doc.go:72` ("Use the 'jsonschema' struct tag to provide a description for the property") and `jsonschema/infer.go:329` (`fs.Description = tag`, no other field set from the tag) — read directly from `google/jsonschema-go@v0.4.3` in the module cache.

**Alternatives considered**: Encode an example inline in the tag string itself, e.g. `jsonschema:"regexp pattern to search node content for, e.g. \"TODO\""` (rejected as the sole mechanism — it satisfies a human reading the description but does not populate the schema's own `examples` keyword, which is what spec FR-003/SC-002 asks for as a distinct, structured field many MCP clients read separately from `description`; nothing stops using a *first-example-mentioned-in-prose* as a convenience in addition, but it is not a substitute for a real value in the `examples` array of the field's JSON Schema).

## D4 — Filter's own field-level documentation

**Decision**: Add `jsonschema` tags to every `mcpStatement` field (currently none — `serve.go:83-90`), describing each of `Source`/`SourcePattern`/`Predicate`/`PredicatePattern`/`Target`/`TargetPattern` individually. Keep this on the shared `mcpStatement`/`mcpFilter` types in `serve.go`, since they are reused verbatim by four tools (`node_grep`, `subgraph_get`, `context_retrieve`, `node_match`). Attach whole-`filter`-object example values (not sub-field examples) per tool, via D3's `withExamples(s, "filter", ...)`, since a realistic usage pattern is a complete filter object (e.g. `{"statements":[{"predicate":"knows"}]}`), not an isolated field value.

**Rationale**: Satisfies spec FR-004 (≥2 distinct usage-pattern examples on the filter argument) while keeping the syntax description itself (what `sourcePattern` means, that it's a regexp, that fields accept a single string or an array) written exactly once and shared, per spec FR-011 ("no separate list left stale") — today that description exists nowhere at the field level, only as prose buried in `schemaAdvertisement` and in `specs/VISION.md`.

**Evidence**: `mcpStatement` struct fields have no `jsonschema` tag today (`serve.go:83-90`), confirmed by direct read.

## D5 — Server workflow guidance and tool-preference notes (spec Story 4)

**Decision**: Replace the single `schemaAdvertisement` constant (`serve.go:57`) with a `sessionInstructions()` function in `serve.go` that joins one shared "call `schema` first" sentence with one short workflow/preference `const` note declared in each tool file, next to that tool's `mcp.Tool` var, e.g. in `serve_tool_node_grep.go`:

```go
// nodeGrepWorkflowNote documents node_grep's place in a session and when to
// prefer it over context_retrieve/subgraph_get — composed into the server's
// session-start Instructions by serve.go's sessionInstructions().
const nodeGrepWorkflowNote = "Prefer node_grep when you need the exact matching line(s) of a node's raw content; prefer context_retrieve when you want whole ranked node objects instead of line matches."
```

`mcp.ServerOptions.Instructions` remains a single string on the wire (an SDK/protocol constraint, not a scope choice — `NewServer(..., &mcp.ServerOptions{Instructions: sessionInstructions()})`), but its *source* is now one short constant per tool, colocated with that tool's own definition, instead of one 900-character paragraph a maintainer has to remember to revisit by hand whenever any tool changes.

**Rationale**: Directly satisfies spec FR-007/FR-008/FR-009 and Story 4's edge case ("preference guidance recorded as part of adding the tool, not deferred to a later, separate edit of an unrelated block") — adding or changing a tool's overlap guidance is now a one-file diff, and `sessionInstructions()`'s composition makes the *absence* of a tool's note visible (a reviewer sees `sessionInstructions()`'s call list and a new tool file not represented in it, rather than having to remember to open an unrelated 900-character string and hand-edit it).

**Alternatives considered**: Leave `schemaAdvertisement` as one hand-authored paragraph but move it into `serve.go` unchanged, only fixing D1-D4 (rejected — this leaves spec Story 4 / FR-007-009 and the corresponding edge case unaddressed, and was explicitly requested in the original feature spec, not just the follow-on refactor instruction). Give each `mcp.Tool` an `_meta` field carrying preference metadata read by the client directly instead of the server's `Instructions` string (rejected — `_meta` is free-form per the MCP spec and not a documented, client-interoperable channel for this; `Instructions` is the one session-level channel every MCP client is expected to surface to its model, which is what spec FR-009/SC-004 requires be true "without making any tool call").

## D6 — Tool output description (spec FR-006)

**Decision**: Do not introduce a typed `Out` / `OutputSchema` for any handler. Instead, each tool's `Description` string is authored as two sentences by convention: what the tool does and how its arguments affect it, then a final sentence stating the shape of what it returns (e.g. node_grep's existing description already ends "...(see schema)."; the output half — "Returns one markdown table row per matching line: id, type, line number, snippet." — is new).

**Rationale**: `mcp.Tool.OutputSchema` is only populated by `AddTool[In, Out]` when `Out` is a concrete (non-`any`) type (confirmed: `server.go:307`, `reflect.TypeFor[Out]() != reflect.TypeFor[any]()`); every handler in this file returns `(*mcp.CallToolResult, any, error)` — `Out` is `any` — because responses are pre-rendered markdown text in `CallToolResult.Content`, not structured `StructuredContent`. Changing that shape (introducing real `Out` structs, threading `StructuredContent` through six handlers, updating every render function and E2E assertion) is a materially larger, unrelated change the user's instruction did not ask for and the feature spec's Assumptions section explicitly rules out ("not intended to change what the server actually returns for existing tools"). A one-more-sentence addition to `Description` satisfies FR-006 ("describe a tool's output ... as part of the tool's specification") without it.

**Evidence**: `toolForErr` (`go-sdk@v1.6.1/mcp/server.go:285-313`) only calls `setSchema[Out]` when `t.OutputSchema != nil || reflect.TypeFor[Out]() != reflect.TypeFor[any]()`; every existing handler signature in `serve.go` uses `any` for `Out`.

**Alternatives considered**: Introduce `OutputSchema` for at least `schema`/`node_match` (rejected for this feature — inconsistent to add it for two tools and not the other four, and out of the scope the user described; a future feature can revisit once/if a client-visible need for structured output (not just prose description) is identified).

## D7 — Test files

**Decision**: `cmd/arc/graph/serve_test.go` is left as a single file, unsplit, for this feature.

**Rationale**: The user's instruction named `serve.go`'s split explicitly and did not mention `serve_test.go`; every test continues to compile and pass unchanged because D1-D6 only move/rename unexported package-private symbols within the same package (`graph`) and do not change any handler's behavior, any `mcp.Tool.Name`, or any JSON wire shape a test asserts against. Splitting ~1491 lines of existing tests into six files to mirror the new source layout is a legitimate follow-up but is additional scope beyond what was asked, and the existing single-file-per-scenario-suite already satisfies constitution Principle VIII (colocated with the command package, one scenario per test function) regardless of how many files the production code lives in.

**Alternatives considered**: Split tests 1:1 with the new tool files to mirror `grep.go`/`grep_test.go`-style pairing used elsewhere in `cmd/arc/graph/` (flagged as a reasonable follow-up in the Completion Report, not done here to avoid silently expanding a scope the user stated precisely).
