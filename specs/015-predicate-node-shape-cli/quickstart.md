# Quickstart: Validating CLI/MCP "Type" Terminology Consistency

## Prerequisites

- Build `arc` from this branch: `go build ./cmd/arc`
- A fixture graph with nodes of at least two distinct types (e.g. the existing `testdata` fixtures used by
  `cmd/arc/graph/grep_test.go` / `subgraph_test.go`)

## 1. `--type` flag replaces `--kind` (spec US1)

```sh
./arc grep --type source TLS
./arc subgraph "Transport Layer Security" --type source
```

**Expected**: identical results to what `--kind source` produced before this change (same matches, same
subgraph). Then confirm the old flag is gone:

```sh
./arc grep --kind source TLS
```

**Expected**: fails with `unknown flag: --kind` (standard Cobra error, non-zero exit), not a silent no-op.

```sh
./arc grep --help
./arc subgraph --help
```

**Expected**: flag descriptions, `Long` help text, and `Example` lines all say "type", not "kind".

## 2. `arc apply`'s warning text (spec US2)

Apply a patch introducing a node whose type is not yet in the graph's schema index:

```sh
./arc apply patch-with-new-type.md
```

**Expected**: the printed warning reads `"... is not a recognized node type for this graph ..."` (not
"kind").

## 3. MCP `node_grep`'s `type` field and table header (spec US3)

Start `arc serve` and call `node_grep` with a `type`-keyed filter (via any MCP client, or the project's
existing MCP test harness):

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "type": ["source"] } } }
```

**Expected**: results are restricted to nodes of type `source`, and the returned table's header row reads
`| id | type | line | snippet |`.

Then confirm the old field name is rejected, not silently honored:

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "kind": ["source"] } } }
```

**Expected**: the tool call fails with an error identifying `kind` as an unrecognized filter property —
`node_grep`'s argument schema disallows unlisted properties (the MCP SDK's default posture for every tool's
struct-derived argument schema), so a stale `kind` key is rejected, not silently dropped, mirroring the CLI's
own no-alias treatment of the retired `--kind` flag (documented edge-case behavior).

## Automated verification

```sh
go test ./cmd/arc/graph/... ./internal/core/... ./internal/app/graph/...
```

**Expected**: all pass, including the updated assertions for the renamed flag/field/warning-text/table-header
surfaces (spec Acceptance Scenarios 1-6, Edge Cases).
