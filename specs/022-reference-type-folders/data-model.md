# Phase 1 Data Model: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Feature**: `022-reference-type-folders` · **Date**: 2026-08-23

This feature introduces **no new Go types**. It changes the *values* carried by three existing
value tables and one pure function. That is the whole domain change; everything else in the
plan is propagation.

---

## 1. Changed values

### 1.1 `kernel.CoreTypeDefs["Resource"]` — redefined

| Field | Before | After |
|---|---|---|
| `Merge` | `MergeFirstWriteWin` | *unchanged* |
| `Required` | `ref`, `relevance` | `text`, `tags`, `mentionedIn` |
| `Optional` | `url`, `isCitedBy`, `authors`, `year`, `doi`, `status`, `notes`, `indexed`, `mentions`, `mentionedIn`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`, `referencedBy` | `notes` |
| `Description` | "an external work the graph points to but has not ingested, or a topic/area tracked for reading or research" | "A fragment of an ingested document's content that is relevant to the graph but does not warrant its own dedicated type; tag-classified so a recurring pattern can later be promoted into a proper domain type." |

CORE §11.4 lists `notes` as the sole optional. The semantic and structural optionals are not
carried over.

### 1.2 `kernel.CoreTypeDefs["Reference"]` — added

| Field | Value |
|---|---|
| `Merge` | `MergeFirstWriteWin` |
| `Required` | `title`, `ref`, `relevance` |
| `Optional` | `url`, `authors`, `year`, `doi`, `status`, `isCitedBy`, `notes` |
| `Description` | "A node for an external work the graph points to but has not ingested, or a topic/area tracked for reading or research." |

Predicate lists follow the upstream 0.9→0.10 revision note, per `spec.md` § Clarifications.
`MergeFirstWriteWin` is inherited from `Resource`'s former definition: the merge op belonged to
the external-work semantics, which moved wholesale to `Reference`.

**Every one of these ten predicates already exists in `CorePredicateDefs`.** No predicate is
added by this feature.

### 1.3 `kernel.CoreTypeBases` — `Reference` added

```
"Source": {"Node"}, "Entity": {"Node"}, "Resource": {"Node"},
"Timeline": {"Node"}, "Reference": {"Node"},
```

### 1.4 `kernel.CorePredicateDefs` — description rewording only

No role, merge, alignment, or label changes. Five descriptions are reworded to name the type
that now owns the semantics:

| Predicate | Change |
|---|---|
| `ref` | "Resource type: a citable work…" → "Reference type: a citable work…" |
| `status` | "a backlog **resource** is a research target" → "a backlog **reference** is a research target" |
| `relevance` | "why the **resource** matters" → "why the **reference** matters" |
| `cites` | "a cited **resource**" → "a cited **reference**" |
| `isCitedBy` | confirm wording targets `Reference` |
| `Node` (in `CoreTypeDefs`) | description enumerates four content types → five |

`mentionedIn` is already worded generically ("the inverse of mentions — recorded as a backlink
under the entity's own mentionedIn block") and is generalized in *meaning* to cover `Resource`
without a text change being strictly required; reword only if it reads as `Entity`-exclusive.

### 1.5 `ctrl/kernel.DefaultLayout.Folders` — rewritten

```
Source/    Entity/    Resource/    Reference/
timeline/yearly/      timeline/monthly/
_schema/Property/     _schema/Class/
```

Eight entries: four type folders, two timeline buckets, two schema type folders. `hasStub`
suppresses the `.gitkeep` for the two `_schema/` folders (they receive seed files); the other
six get one, as today.

### 1.6 `schema/kernel` path constants — values only

```
PredicatesDir = "_schema/Property"   // was "_schema/predicates"
TypesDir      = "_schema/Class"      // was "_schema/types"
```

Go identifiers unchanged (research.md D2). GoDoc updated to state the new paths.

---

## 2. Changed functions

### 2.1 `graph/service.nodeFolder` — becomes identity

```
func nodeFolder(kind string) string { return kind }
```

`coreKindFolders` is **deleted**. The `strings.HasSuffix(kind, "s")` / `kind + "s"` fallback is
**deleted**. The `strings` import may become unused in `apply.go` — check.

Invariants preserved: `nodeFolder` is still never called with `Timeline`; `nodePath` is still
`nodeFolder(node.Type) + "/" + node.ID + ".md"`; `referrerPath` still diverts `Timeline` to
`periodGranularity`.

### 2.2 `core.textPredicateFor` — two cases

| `@type` | Leading (before) | Leading (after) | Trailing |
|---|---|---|---|
| `Source` | `abstract` | `abstract` | `notes` |
| `Entity` | `definition` | `definition` | `notes` |
| `Resource` | `relevance` | **`text`** | `notes` |
| `Reference` | *(default)* `text` | **`relevance`** | `notes` |
| *any other* | `text` | `text` | `notes` |

Parse and render both route through this one function, so the two sides move together
(research.md D5).

### 2.3 `graph/service.revertLeadingKey` — two cases, mirroring 2.2

`Resource` → `text`; add `Reference` → `relevance`. The three stale domain entries
(`hypothesis`→`claim`, `aporia`→`tension`, `thought`→`claim`) are **left as-is** — they are a
pre-existing divergence from `core`'s table, out of scope here, recorded as follow-up 1 in
research.md §7.

---

## 3. Explicitly unchanged

| Thing | Why it looks affected but is not |
|---|---|
| `cmd/arc/graph/apply.go:pluralizeKind` | Display-only ("`+3 entities`"). Changing it prints "`+3 Entity`". research.md D4. |
| `lint`/`grep` traversal | Skips `.arc` and `_schema` by name; discovers content folders generically. |
| Edge/backlink resolution | Basename-keyed; path-independent. |
| `serve.go` (MCP) | Contains no folder literal. |
| `RegisterType`/`RegisterPredicate` | Path-derived from the constants in 1.6; no logic change. |
| Any `@type` derivation | Nothing derives a type from a folder anywhere in the codebase. |

---

## 4. Validation rules

Derived from `spec.md`, each mapping to a test in research.md §6:

1. `nodeFolder(t) == t` for every `t` in `CoreTypeDefs` except `Timeline`, and for an
   unrecognized domain type.
2. Every content type's `nodeFolder` value appears in `DefaultLayout.Folders`, and every
   non-timeline, non-`_schema` entry in `DefaultLayout.Folders` is some type's `nodeFolder`
   value. (Guards the D3 drift hazard.)
3. `core.TextPredicateFor(t, leading)` equals `revertLeadingKey(t)` for all five core types at
   `leading == true`. (Guards the D6b divergence.)
4. `Seed()` emits `_schema/Class/Reference.md` and a `_schema/Class/Resource.md` in which
   `ref` and `relevance` appear nowhere.
5. Every predicate named in `Reference.Required ∪ Reference.Optional` is a key of
   `CorePredicateDefs`.
6. A parsed-then-rendered node preserves its leading-prose key for each of the five core types.
