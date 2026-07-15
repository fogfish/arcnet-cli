# Phase 1 Data Model: Type Inheritance via `rdfs:subClassOf`

## Entities

### `subClassOf` (predicate, `Aligned: "rdfs:subClassOf"`)

New entry in `kernel.CorePredicateDefs` (`internal/app/schema/kernel/schema.go`):

| Field | Value |
|---|---|
| Role | `"edge"` |
| Merge | `core.MergeUnion` |
| Label | *(none — edge-role predicates render flat, no heading label needed)* |
| Aligned | `"rdfs:subClassOf"` |
| Description | Declares that the subject type inherits every predicate the target type requires or permits. |

The predicate's own name is the plain camelCase `subClassOf`, matching every other RDF-aligned predicate already registered (`isPartOf` → `Aligned: "dcterms:isPartOf"`, `broader` → `Aligned: "skos:broader"`, …) — `internal/core`'s bullet-parsing regex only accepts `\w+` predicate names, so the colon-bearing RDFS term lives in `Aligned`, never in the bullet key itself.

Renders on a Class node as a flat bullet per base type, e.g. `- subClassOf:: [[Node]]`. A type may carry zero, one, or many such edges (multiple inheritance, spec FR-002).

### `Node` (type)

New entry in `kernel.CoreTypeDefs`:

| Field | Value |
|---|---|
| Merge | `core.MergeUnion` (inert — see research.md D6) |
| Required | `["published", "created"]` |
| Optional | `["tags", "text", "updated", "scoreZ", "scoreC"]` |
| Description | Documents `Node` as the graph's implicit universal base for content types. |

`Node` is never itself a node's `@type` in practice; it exists to be inherited from.

### Reshaped `source`/`entity`/`resource`/`timeline` (types)

Each gains an explicit `rdfs:subClassOf → Node` edge (redundant with the implicit rule, written for self-description) and drops every `Required`/`Optional` entry now supplied transitively by `Node`:

| Type | Removed from Required | Removed from Optional | Net Required after change | Net Optional after change |
|---|---|---|---|---|
| `source` | — | `tags`, `created`, `updated`, `scoreZ`, `scoreC` | `title`, `published`\*, `abstract`, `mentions` | `authors`, `url`, `cites`, `doi`, `indexed` |
| `entity` | — | `tags`, `published`, `created`, `updated`, `scoreZ`, `scoreC` | `category`, `definition`, `mentionedIn`\* | `aliases`, `notes`, `indexed`, `mentions`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`, `referencedBy` |
| `resource` | — | `tags`, `text`, `published`, `created`, `updated`, `scoreZ`, `scoreC` | `ref`, `relevance`\* | `url`, `isCitedBy`, `authors`, `year`, `doi`, `status`, `notes`, `indexed`, `mentions`, `mentionedIn`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`, `referencedBy` |
| `timeline` | — | `tags`, `text`, `created`, `updated`, `scoreZ`, `scoreC` | `granularity`, `cites`, `period`\* | `heading`, `indexed`, `mentions`, `mentionedIn` |

\* `published`/`created` are no longer listed directly on any of these four types — they arrive via `Node`'s `Required`, and are reflected in the *effective* contract `Resolve`/`Seed` compute, not in each type's own literal `Required` list. The "Net Required after change" column above is each type's own *direct* declaration; the *effective* contract (what `arc lint` actually checks) is this list plus `Node`'s `Required`/`Optional` for every one of these four types.

`Property`/`Class` are unchanged (research.md D5 — excluded from the `Node` relationship entirely).

### Raw type record (internal to `internal/app/schema/service`, not exported)

Package-private intermediate used only during resolution — never part of `core.TypeDef` or `core.Index`:

```
rawType {
    merge       core.MergeOp
    required    []string   // this type's own directly declared required predicates
    optional    []string   // this type's own directly declared optional predicates
    subClassOf  []string   // this type's own directly declared base-type names
    description string
}
```

### Effective (inherited) contract — computation

Given the map of every type's `rawType` (built in one pass over `_schema/types/*.md`), the effective `core.TypeDef` for type `T` is computed by memoized recursion:

```
resolve(T):
    if T has a memoized effective TypeDef → return it
    if T is on the current recursion stack → cycle error, naming T
    push T onto the recursion stack

    required = copy(raw[T].required)
    optional = copy(raw[T].optional)
    bases = raw[T].subClassOf
    if T not in {"Node", "Property", "Class"} → bases = bases + ["Node"]   // implicit universal base

    for each base in bases:
        if base not in raw → unresolved-base-type error, naming T and base
        baseEffective = resolve(base)
        required = required ∪ baseEffective.Required
        optional = (optional ∪ baseEffective.Optional) \ required

    pop T from the recursion stack
    effective = TypeDef{Merge: raw[T].merge, Required: required, Optional: optional, Description: raw[T].description}
    memoize T → effective
    return effective
```

This produces, for every type name, a fully flattened `core.TypeDef` — the only representation ever stored in `core.Index.Types`. No downstream consumer (lint, or any future schema-contract consumer per FR-009) sees `rawType`, `subClassOf`, or any notion of hierarchy — only the final union.

## Validation Rules

- FR-006/FR-007 (dedup, required-wins): enforced structurally by the `required ∪ …` / `(optional ∪ …) \ required` set operations above — a predicate can never appear in both `Required` and `Optional` of an effective contract, and duplicate contributions collapse naturally (set union).
- FR-010 (cycle detection): the active recursion stack check above fires the moment a type is revisited before its own resolution completes, regardless of cycle length (including direct self-reference, the smallest case per the spec's Edge Cases).
- FR-011 (unresolved base type): the `base not in raw` check above fires for any `rdfs:subClassOf` target with no corresponding `_schema/types/<name>.md` document, treating it as contributing no predicates (the resolution stops for that reference; `Resolve` still fails overall — research.md D4 — since a dangling reference makes the schema invalid, not silently partial).
- FR-012 (merge behavior not inherited): `resolve(T)`'s `Merge` always comes from `raw[T].merge`, never from any base's — the recursion only ever propagates `Required`/`Optional`.

## State Transitions

None — types and their `rdfs:subClassOf` relationships are declarative schema documents, not stateful entities with a lifecycle beyond create/read (identical to today's `Required`/`Optional` handling).
