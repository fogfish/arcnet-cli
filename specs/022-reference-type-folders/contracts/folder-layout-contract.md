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

**Extended to node identities — `specs/003-apply-patch` BUG-008.** The same hazard applies one
level down, to the identity a node file is *named by*, and was reasoned about here for folder
names only. Two consequences, both now specified by `003-apply-patch` FR-026/FR-027/FR-029:

- A node identity's existence MUST NOT be decided by handing a constructed path to the
  filesystem and seeing whether it opens — on a case-insensitive volume
  `Open("Entity/Lightstep.md")` returns the bytes of `Entity/LightStep.md`, so the filesystem,
  not the tool, silently decides whether two spellings are one node. The volume's case behavior
  is probed at run time (`internal/adapter/fsys`'s `CaseFolder`) and the identity resolved
  against it.
- Retrieving a file's **real name** MUST go through `ReadDir`, never `Stat`: `os`'s
  `fs.FileInfo.Name()` is derived from the path string handed in, so `Stat` echoes the caller's
  own spelling back and hides the disagreement. (Asking whether a location *folds* case is a
  different question — an existence question, which `Stat` answers correctly. C7's rule is about
  names, not about existence.)

## C8. Compatibility

Breaking. A graph created before this feature has `_schema/predicates/` and `_schema/types/`,
neither of which this contract's schema resolution reads. Such a graph MUST fail to resolve —
loudly, naming the missing folder, before any write occurs. It MUST NOT be partially migrated,
repaired, or written into. No migration path is provided.
