# Data Model: CamelCase Node Class Names

This feature adds no new persisted entity or file format — it changes the
**casing convention** of an existing identifier (`core.Node.Type` / a
`Class` schema document's `@id`) and adds one new validation outcome. No
`core.Node`, `core.Patch`, `core.Index`, `core.TypeDef`, or
`core.PredicateDef` field is added, removed, or retyped.

## Class Name (Type Identifier)

Represents: the identifier a node's `@type` field or a `_schema/types/*.md`
document's `@id` carries (e.g. `Entity`, `Source`, a user-defined
`Hypothesis`).

**Validity rule (new, this feature)**: the identifier's first rune MUST
satisfy `unicode.IsUpper` (research.md D1). No constraint on subsequent
characters.

**Where enforced**:
| Producer | Enforcement point | Requirement |
|---|---|---|
| `arc apply` (patch H1 heading) | `internal/core.patchNodeIdentity` | FR-004/FR-005 |
| `arc apply` (explicit `@type` in yaml fence) | `internal/core.patchNodeIdentity` | FR-008 |
| Built-in schema (`CoreTypeDefs` keys) | `internal/app/schema/kernel/schema.go` (static data, not runtime-checked) | FR-002 |
| `arc init` (seeded `_schema/types/*.md`) | Inherits from `CoreTypeDefs` via `Seed()` — no separate check | FR-003 |
| `arc lint` (schema type definitions) | `checkSchemaTypeCase` over `core.Index.Types` | FR-006 |
| `arc lint` (node's own `@type` reference) | `checkNodeTypeCase` over each parsed `core.Node` | FR-007 |

**Not enforced by this feature**: `internal/core.ParseNode`/`identityFields`
(reading a standalone on-disk node file) does not itself reject a
lowercase `@type` — that content is still readable (so `arc lint`,
`arc grep`, `arc subgraph`, etc. keep working against a repository that
has pre-existing lowercase-typed content); `arc lint` is the mechanism
that surfaces it as a violation (FR-009), not a parse-time hard failure.

## Rejection Outcome (`arc apply`)

Represents: the observable result of `arc apply` refusing a patch document
whose H1 heading or explicit `@type` value violates the Class Name rule
above.

| Field | Value |
|---|---|
| Exit status | non-zero |
| Graph mutation | none (no file created/modified, no commit) |
| stderr | human-readable message naming the offending heading/value and stating the CamelCase requirement (`internal.core.ErrTypeCasing`, research.md D3) |

This is not a new persisted type — it is `internal/core.ParsePatch`'s
existing error-return path (already consumed by
`internal/app/graph/service.Apply`'s `readPatch` failure branch,
`internal/app/graph/service/apply.go:171-175`) carrying a new error value.

## Lint Violation (`RuleTypeCase`)

Represents: one `kernel.Violation` (existing type,
`internal/app/lint/kernel/lint.go`) reported for a Class Name that fails
the rule above, at either enforcement point in the table.

| Field | Node-level occurrence (FR-007) | Schema-level occurrence (FR-006) |
|---|---|---|
| `Rule` | `kernel.RuleTypeCase` | `kernel.RuleTypeCase` |
| `Path` | the offending node's file path | `_schema/types/<name>.md` |
| `Line` | `0` (no specific line — the whole node's `@type` is at fault, matching `RuleUnrecognizedKind`'s own convention) | `0` (graph-spanning, matching `RuleUniqueBasename`'s convention) |
| `Message` | `type %q is not CamelCase` | `type %q is not CamelCase` |

No new field is added to `kernel.Violation` itself — this reuses the
existing struct exactly as `RulePredicateCase`/`RuleUnrecognizedKind`
already do.
