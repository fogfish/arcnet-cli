# Contract: Graph Folder Layout (CORE §6, v0.11)

**Feature**: `022-reference-type-folders` · **Status**: normative for this feature

## C1. Type-folder naming

A **type folder** holds nodes of exactly one `@type`. Its name MUST equal that type name
**character for character** — same case, no pluralization, no abbreviation, no synonym.

```
folder(T) = T          for every type T except Timeline
```

The derivation MUST be the identity function. It MUST NOT consult a lookup table, append a
pluralizing suffix, change case, or otherwise transform the type name. This holds for core
types and for types the tool does not recognize, identically.

## C2. Exempt (non-type) folders

| Folder | Why exempt | Rule for its contents |
|---|---|---|
| `_schema/` | A namespace prefix, not a type. | Its children ARE type folders and MUST follow C1: `Property/` (CORE §9.1), `Class/` (CORE §9.2). |
| `timeline/` | An index. Its `Timeline` nodes are bucketed by granularity, not filed flat. | `timeline/yearly/<YYYY>.md`, `timeline/monthly/<YYYY-MM>.md`. A `Timeline` node MUST NEVER be filed at `Timeline/<id>.md`. |
| `.arc/` | Tool state, not graph content. | Unchanged. |

## C3. Canonical layout produced by `arc init`

Exactly these eight folders, and no others:

```
Source/                 Entity/                Resource/              Reference/
timeline/yearly/        timeline/monthly/
_schema/Property/       _schema/Class/
```

`arc init` MUST NOT create `sources/`, `entities/`, `resources/`, `references/`,
`_schema/predicates/`, or `_schema/types/`.

## C4. Node path

```
path(node) = folder(node.Type) + "/" + node.ID + ".md"
```

except `@type: Timeline`, which uses the C2 bucketed form.

## C5. Type is read, never inferred

A consumer MUST read `@type` from the node document. It MUST NOT infer a node's type from the
folder the node sits in. C1 makes the folder a reliable **mirror** of the type, not a
substitute for it. A node an author has hand-moved out of its type folder MUST still be
handled according to its declared `@type`.

## C6. Layout agreement invariant

The static layout `arc init` creates and the derivation `arc apply` writes through are separate
mechanisms and MUST be kept in agreement:

```
∀ content type T ∈ CoreTypeDefs, T ≠ Timeline :  folder(T) ∈ DefaultLayout.Folders
∀ f ∈ DefaultLayout.Folders, f ∉ {timeline/*, _schema/*} : ∃ T . folder(T) = f
```

This MUST be enforced by a test. Violating it does not raise an error at runtime — the write
path creates missing parent directories silently — so the divergence would surface only as a
graph with two parallel folder sets.

## C7. Path assertions

Any test asserting a node's location MUST compare the **exact path string** the store was
asked for. It MUST NOT rely on a filesystem stat. macOS/APFS is case-insensitive and would
report `Source/`, `source/`, and `SOURCE/` as the same path; a case defect would pass locally
and fail on a case-sensitive CI filesystem.

## C8. Compatibility

Breaking. A graph created before this feature has `_schema/predicates/` and `_schema/types/`,
neither of which this contract's schema resolution reads. Such a graph MUST fail to resolve —
loudly, naming the missing folder, before any write occurs. It MUST NOT be partially migrated,
repaired, or written into. No migration path is provided.
