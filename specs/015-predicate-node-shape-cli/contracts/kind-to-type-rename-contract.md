# `--kind` → `--type` / MCP `kind` → `type` Contract Delta

This is the concrete before/after for the four user/tool-visible surfaces this feature renames. No `--json`
output payload changes shape (research.md D4) — everything here is an *input* surface (a CLI flag or an MCP
tool's filter argument) plus two output-side labels (a warning string, a table column header).

**This is a breaking, non-additive change** (plan.md Constitution Check, Complexity Tracking) — accepted
pre-1.0, called out explicitly per constitution Principle XIV rather than hidden, consistent with how specs
010/012/013 already treated their own breaking `--json` schema changes.

## 1. `arc grep` / `arc subgraph` — `--kind` flag

**Before**:

```text
arc grep --kind source TLS
arc subgraph "Transport Layer Security" --kind source
```

**After**:

```text
arc grep --type source TLS
arc subgraph "Transport Layer Security" --type source
```

Semantics unchanged: repeatable, OR'd across repeated uses, AND'd against `--tag`/`--attr`. Invoking `--kind`
after this change ships fails with the standard Cobra "unknown flag: --kind" error — no alias, no warning.

## 2. `arc apply` — unrecognized-type warning

**Before**: `"<type> is not a recognized node kind for this graph — auto-registered with a default schema document"`

**After**: `"<type> is not a recognized node type for this graph — auto-registered with a default schema document"`

## 3. `arc serve`'s `node_grep` MCP tool — filter wire field

**Before**:

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "kind": ["source"] } } }
```

**After**:

```json
{ "name": "node_grep", "arguments": { "pattern": "TLS", "filter": { "type": ["source"] } } }
```

Semantics unchanged. A client still sending `"kind"` after this change ships receives a tool-call error
identifying `kind` as an unrecognized filter property: `node_grep`'s argument schema is generated from
`mcpFilter` by the MCP SDK's JSON Schema inference, which sets `additionalProperties: false` on every
struct-derived object schema — this is the SDK's existing, unconditional posture for every tool argument
object in this server, not something introduced by this rename. The rename therefore surfaces the same
no-alias treatment on the MCP side that FR-010 already mandates for the CLI's retired `--kind` flag: a clear,
immediate error, not a silent no-op.

## 4. `arc serve`'s `node_grep` MCP tool — result table header

**Before**: `| id | kind | line | snippet |`

**After**: `| id | type | line | snippet |`

The table's data rows are unaffected — the column already carried `kernel.Match.Type`'s value; only the
header label changes.
