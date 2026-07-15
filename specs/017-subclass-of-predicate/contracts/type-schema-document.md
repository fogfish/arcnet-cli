# Contract: `_schema/types/<name>.md` — `rdfs:subClassOf` extension

This extends the existing machine-readable type-schema document contract (spec 011) that `_schema/types/<name>.md` documents, `arc apply`'s auto-registration, `arc lint`, and any external tool reading `_schema/types/` already rely on. Everything spec 011 already guarantees (`## Requires`/`## Optional` sections, `merge` front-matter, mandatory description body) is unchanged; this document covers only the addition.

## New: zero or more `subClassOf` edges (`Aligned: "rdfs:subClassOf"`)

A type document MAY carry any number of flat bullet edges of the form:

```
- subClassOf:: [[<base-type-name>]]
```

The predicate's registered name is the plain camelCase `subClassOf` — not the colon-bearing `rdfs:subClassOf` term itself, which lives in the predicate's own `Aligned` field — because `internal/core`'s bullet-parsing regex only accepts `\w+` predicate names (matching every other RDF-aligned predicate already registered: `isPartOf`, `broader`, …).

- Each `<base-type-name>` MUST name another document in `_schema/types/` (an unresolved reference fails schema loading — see Errors below).
- A type may declare more than one such edge (multiple inheritance).
- Placement/order relative to the `## Requires`/`## Optional` sections is not significant; `subClassOf` edges render as flat bullets (predicate role `edge`), never grouped under a heading.
- Declaring `subClassOf` toward a type that is, directly or transitively, an ancestor of the declaring type (a cycle) fails schema loading — see Errors below.

## New: the implicit `Node` base

Every type in `_schema/types/` **except** `Node`, `Property`, and `Class` is treated as if it carried `subClassOf → Node`, whether or not that edge is present in its document. A graph maintainer authoring a new custom type does not need to write this edge for it to take effect — but `arc init`'s seeded `source`/`entity`/`resource`/`timeline` documents write it explicitly anyway, for the document's own self-description.

`Node`'s own contract (`_schema/types/Node.md`, seeded by `arc init`):

| Required | Optional |
|---|---|
| `published`, `created` | `tags`, `text`, `updated`, `scoreZ`, `scoreC` |

## Effective contract (what every consumer sees)

Any command or tool that reads a type's `Required`/`Optional` predicate contract — via `internal/app/schema/service.Resolve`'s returned `core.Index.Types` — sees the fully flattened, inherited result: this type's own directly declared `Required`/`Optional`, unioned with every `rdfs:subClassOf` ancestor's own (transitively resolved) `Required`/`Optional`, deduplicated, with "required" always winning over "optional" for the same predicate. The raw `rdfs:subClassOf` edges themselves are not exposed through `core.TypeDef` — a consumer reading `core.Index.Types["source"].Required` gets the complete list directly, with no separate hierarchy-walking step of its own required.

## Errors

Schema loading (`Resolve`, and therefore every command that depends on it — `arc lint`, `arc apply`, `arc serve`, …) fails, before any other schema-dependent work proceeds, when:

- A type's `rdfs:subClassOf` edge names a type with no corresponding `_schema/types/<name>.md` document (`ErrSchemaUnresolvedBase`).
- A type's `rdfs:subClassOf` edges form a cycle, directly or through a longer chain, including a type naming itself (`ErrSchemaCycle`).

Both follow the same reporting shape as the existing `ErrSchemaInvalid`/`ErrSchemaMissing` errors — naming the offending type so a maintainer can locate and fix the document.

## Compatibility

A type document with zero `rdfs:subClassOf` edges behaves exactly as before this feature shipped, aside from now implicitly inheriting `Node`'s contract if it is not itself `Node`/`Property`/`Class`. `Property` and `Class` documents are entirely unaffected.
