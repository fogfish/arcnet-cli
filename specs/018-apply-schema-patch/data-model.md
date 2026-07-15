# Data Model: Import Schema Definitions via `arc apply schema`

No new persisted document shape is introduced — a schema patch's `Property`/
`Class` node sections are written using the exact same
`_schema/predicates/<name>.md` / `_schema/types/<name>.md` document contract
`internal/app/schema` already defines (spec 011, extended by spec 017). This
feature adds only in-memory domain values scoped to the new operation.

## `kernel.ApplySchemaResult` (`internal/app/schema/kernel/apply.go`)

The value `service.ApplyPatch` returns, rendered by
`bios.Registry[kernel.ApplySchemaResult]` in `cmd/arc/ctrl`.

| Field | Type | Meaning |
|---|---|---|
| `Source` | `string` | The resolved local path or URL the patch was read from. |
| `Created` | `map[string]int` | Counts of newly created definitions, keyed `"predicate"`/`"type"`. |
| `Merged` | `map[string]int` | Counts of definitions merged into an existing document, same keys. |
| `CommitHash` | `string` | Short hash of the resulting commit; empty when nothing changed (D7 — no-op re-apply). |

No `Skipped` field (unlike `graph.kernel.ApplyResult`): there is no
source-tracking idempotency gate (research.md D7) — a no-op is expressed as
zero-valued `Created`/`Merged` and an empty `CommitHash`, not a distinct
boolean state.

## `port.VCS` (`internal/app/schema/port/vcs.go`)

```go
type VCS interface {
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

Satisfied structurally by the existing `internal/adapter/git.VCS` concrete
type (research.md D3) — no new adapter code.

## `port.Fetcher` (`internal/app/schema/port/fetcher.go`)

```go
type Fetcher interface {
    Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}
```

Satisfied by the new `internal/adapter/http.Client` (research.md D2). The
returned `io.ReadCloser` is passed directly to `core.ParsePatch(io.Reader)`
— identical downstream handling to the local-file path's `fsys.Store.Open`
result, so `service.ApplyPatch`'s parse step is source-agnostic once past a
short `readPatchSource` branch.

## Errors (`internal/app/schema/service/errors.go` additions)

| Constant | Shape | Meaning |
|---|---|---|
| `ErrDisallowedNodeType` | `faults.Safe2[string, string]` | A patch node's `@id`/`@type` is not `Property`/`Class`; names both. Entire operation fails, zero writes (spec FR-005/FR-006). |
| `ErrPatchRead` | `faults.Safe1[string]` | The patch source (local path, URL, or resolved `arcnet:` reference) could not be read/fetched or failed to parse; names the source. |
| `ErrEmptyArcnetReference` | `faults.Type` | The input was the bare prefix `"arcnet:"` with nothing after it (spec FR-002a edge case) — rejected before any fetch attempt. |

`internal/app/schema/kernel` additions:

| Value | Shape | Meaning |
|---|---|---|
| `ArcnetCatalogBaseURL` | package-level `var string` | `"https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/"` — the base an `arcnet:<suffix>` input resolves against (research.md D1a). A `var`, not a `const`, purely so an E2E test can point it at an `httptest.Server` for the duration of one test (mirroring `internal/app/ctrl/service.resolveLocalRoot`'s existing package-var-indirection precedent) — production code never reassigns it. |

`internal/adapter/http` additions:

| Constant | Shape | Meaning |
|---|---|---|
| `ErrFetch` | `faults.Safe1[string]` | The HTTP request failed outright (network error, timeout); names the URL. |
| `ErrFetchStatus` | `faults.Safe2[string, int]` | The server responded with a non-2xx status; names the URL and status code. |

## Existing types reused unchanged

- `core.Patch`, `core.Node` (`internal/core/ast.go`) — the parsed patch and its node sections; `Nodes[i].Type` is inspected for the `Property`/`Class` classification (research.md D4) and otherwise treated exactly as `graph.Apply` already treats a node.
- `core.PredicateDef`, `core.TypeDef`, `core.Index` (`internal/core`) — the resolved schema shape `decodePredicateDef`/`decodeTypeDef`/`core.Merge` already operate on (research.md D5/D6).
- `core.MergeOp` and its `validMergeOps`/`validRoles` tables (`internal/app/schema/service/schema.go`) — reused verbatim to validate a patch-carried `Property`/`Class` node before it is written.
