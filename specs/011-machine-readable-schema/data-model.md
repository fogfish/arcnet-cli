# Data Model: Machine-Readable Predicate & Type Schema

## `core.PredicateDef`

The decoded, in-memory shape of one `_schema/predicates/<name>.md` document (research.md D1, D3).

| Field | Type | Source | Notes |
|---|---|---|---|
| `Role` | `string` | `Attrs["role"]` | One of `meta`/`text`/`href`/`edge`/`link` (CORE §5/§9.1). Mandatory; `Resolve` fails the whole load if absent or outside this set (spec FR-014). |
| `Merge` | `core.MergeOp` | `Attrs["merge"]` | Mandatory; `Resolve` fails the whole load if absent or not a recognized value. |
| `Label` | `string` | `Attrs["label"]` | Optional; empty string means "not declared" — a consumer defaults to the capitalized predicate name (CORE §10.8). |
| `Aligned` | `string` | `Attrs["aligned"]` | Optional; empty string means "not declared." |
| `Description` | `string` | `Texts["description"]` | Mandatory per spec FR-001; `Resolve` fails the whole load if empty. |

**Validation rules**: `Role` ∈ {`meta`,`text`,`href`,`edge`,`link`}; `Merge` ∈ the recognized `core.MergeOp` value set; `Description` non-empty. Any violation aborts `Resolve` with a `faults`-annotated error naming the file and the specific field (spec FR-014).

## `core.TypeDef`

The decoded, in-memory shape of one `_schema/types/<name>.md` document (research.md D1, D3).

| Field | Type | Source | Notes |
|---|---|---|---|
| `Merge` | `core.MergeOp` | `Attrs["merge"]` | Mandatory (spec FR-015, the interactively-resolved arcnet-cli-specific bridge field beyond CORE's own `Class`-node shape) — `arc apply`'s existing whole-node merge dispatch reads this field exactly as it reads `_schema/nodes/<kind>.md`'s `merge` today. |
| `Required` | `[]string` | `Edges` where `Predicate == "required"` | Each entry's `Target` is a predicate name; order preserved from the document. |
| `Optional` | `[]string` | `Edges` where `Predicate == "optional"` | Same shape as `Required`. |
| `Description` | `string` | `Texts["description"]` | Mandatory per spec FR-002. |

**Validation rules**: `Merge` present and recognized; `Description` non-empty. `Required`/`Optional` MAY both be empty (a maximally permissive type — the shape `RegisterType` produces for an auto-discovered type, spec FR-011). No cross-check that `Required ∩ Optional == ∅` is performed by this feature (conformance-level validation of this kind is `arc lint`'s separate, future rule-change feature per spec.md Assumptions).

## `core.Index`

The in-memory structure `internal/app/schema/service.Resolve` builds once per command invocation (spec.md's Schema Index entity; research.md D1).

```go
type Index struct {
    Predicates map[string]PredicateDef
    Types      map[string]TypeDef
}
```

- `Predicates`/`Types` are keyed by the predicate/type's own name (`@id`, equal to its file's basename).
- Absence of a key means "not registered" — the same recognition signal `map[string]bool`/`MergeRuleSet.Lookup` gave today, now carrying the full declaration alongside the yes/no.
- Immutable after construction: `Resolve` returns a fully-built value, never a handle a caller mutates in place.

**Relationships**: `internal/app/graph/service.Apply` and `internal/app/lint/service.Lint` each take one `core.Index` parameter (research.md D8), replacing their previous `(core.MergeRuleSet, map[string]bool)` pair.

## Predicate Schema Node (`_schema/predicates/<name>.md`)

The on-disk `core.Node` representation of one `PredicateDef`, per CORE §9.1 (see [contracts/schema-document-contract.md](contracts/schema-document-contract.md) for the exact rendered shape):

- `ID`: the predicate's camelCase name, equal to the file's basename.
- `Type`: the literal `"Property"`.
- `Attrs`: `"role"` (mandatory), `"merge"` (mandatory), `"label"` (optional), `"aligned"` (optional) — each a single-element `[]Predicate`.
- `Texts["description"]`: mandatory prose.

## Type Schema Node (`_schema/types/<name>.md`, replacing `_schema/nodes/<kind>.md`)

The on-disk `core.Node` representation of one `TypeDef`, per CORE §9.2 plus the FR-015 bridge:

- `ID`: the type's name, equal to the file's basename.
- `Type`: the literal `"Class"`.
- `Attrs`: `"merge"` (mandatory, FR-015 bridge field — not part of CORE's own documented `Class` shape).
- `Texts["description"]`: mandatory prose.
- `Edges`: zero or more `{Predicate: "required", Target: <predicateName>}` entries followed by zero or more `{Predicate: "optional", Target: <predicateName>}` entries (rendered as one flat bulleted list per research.md D4, not yet grouped under `## Requires`/`## Optional` headings).

## Discovered Predicate / Type

A predicate or `@type` value encountered mid-`arc apply` with no existing schema document (spec.md Key Entities), auto-registered via `RegisterPredicate`/`RegisterType` (research.md D5, D7) with:

- **Discovered Predicate**: `Role: "edge"`, `Merge: core.MergeUnion`, no `Label`/`Aligned`, and a generic placeholder `Description` (e.g. "Auto-registered by arc apply; describe this predicate's meaning here.") — never left with an empty mandatory `Description` (spec FR-001).
- **Discovered Type**: `Merge: core.MergeUnion`, empty `Required`/`Optional`, and a generic placeholder `Description` (e.g. "Auto-registered by arc apply; describe this type's meaning here.").

Both are written once (never overwritten if already present, FR-012) and land in the same commit as the rest of the triggering patch application (FR-013).
