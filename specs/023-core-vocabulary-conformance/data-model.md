# Phase 1 Data Model: Schema Vocabulary Conformance

**Feature**: `023-core-vocabulary-conformance` · **Date**: 2026-08-23

Authority: ARCNET-CORE **v0.11** (2026-08-23). Deltas derived in [research.md](research.md) D6/D9.
The byte-exact rendered form of every entry below is fixed by the golden snapshot
(`testdata/golden/schema/`, research D13); this document is the reviewable source of truth for
*what changes and why*.

---

## 1. `core.MergeOp` — closed set of six

`internal/core/ast.go`

```go
const (
	MergeImmutable     MergeOp = "immutable"
	MergeUnion         MergeOp = "union"
	MergeFirstWriteWin MergeOp = "firstWriteWin"
	MergeFillIfEmpty   MergeOp = "fillIfEmpty"
	MergeLastWriteWin  MergeOp = "lastWriteWin"
	MergeAppend        MergeOp = "append"
)
```

`MergeValidatedOverwrite` is **deleted**, not deprecated (FR-004). Deletion is a compile-time
break at every dispatch site; the compiler enumerates them (`mergeScalar`'s freeze class in
`internal/core/merge.go`, `validMergeOps` in `internal/app/schema/service/schema.go`, and the
`ast_test.go` vocabulary assertion). Do not grep-and-replace.

The doc comment on `MergeOp` currently reads "fixed, seven-value menu" — it becomes six.

## 2. `core.TypeDef.Merge` — removed

`internal/core/rules.go`

```go
type TypeDef struct {
	Required    []string
	Optional    []string
	Description string
}
```

Live-consumer audit (research D9): the field is written by `service.typeNode`, read by
`decodeTypeDef` into `rawType.merge`, carried through `resolveEffectiveTypes`, and read by
nothing else — `core.Merge` dispatches exclusively per-predicate via `resolveMergeOp`. The
existing doc comments on both `decodeTypeDef` and `CoreTypeDefs` already record it as inert.
Removal is therefore a pure deletion, not a behaviour change.

**Asymmetry to preserve (FR-005/FR-006)**: `decodeTypeDef` keeps *reading past* a `merge`
attribute on an existing `Class` document without failing; `typeNode` stops *writing* one. No
lint rule reports the stale attribute — an un-upgraded graph would otherwise report eight
violations for a field with no effect.

---

## 3. `CorePredicateDefs` — 15 changed, 3 added

`internal/app/schema/kernel/schema.go`. Unlisted entries (39 of 57) are unchanged.

### 3.1 Additions (§10.2)

| `@id` | `role` | `merge` | `aligned` | Description source |
| --- | --- | --- | --- | --- |
| `author` | `meta` | `union` | `schema:author` | §10.2 — "The author of the content." |
| `about` | `meta` | `union` | — | §10.2 — subject matter: technique/theory/platform/system/technology/language/framework/field |
| `genre` | `meta` | `union` | — | §10.2 — genre: paper/standard/tool/dataset/post |

### 3.2 Merge corrections

| `@id` | Was | Becomes | Authority |
| --- | --- | --- | --- |
| `abstract` | `append` | **`firstWriteWin`** | §10.2 |
| `description` | `append` | **`firstWriteWin`** | §10.8 |
| `cites` | `append` | **`union`** | §10.6 |
| `year` | `fillIfEmpty` | **`immutable`** | §10.7 |
| `definition` | `append` | **`firstWriteWin`** | §10.2 rationale (research D8) |
| `relevance` | `append` | **`firstWriteWin`** | §10.2 rationale (research D8) |
| `scoreZ` | `validatedOverwrite` | **`lastWriteWin`** | §9.3 menu (research D7) |
| `scoreC` | `validatedOverwrite` | **`lastWriteWin`** | §9.3 menu (research D7) |

### 3.3 Standard-vocabulary alignments added

| `@id` | `aligned` | Authority |
| --- | --- | --- |
| `title` | `schema:title` | §10.2 |
| `abstract` | `schema:abstract` | §10.2 |
| `url` | `schema:url` | §10.2 |
| `doi` | `schema:doi` | §10.2 |
| `aliases` | `skos:altLabel` | §10.1 |
| `authors` | `schema:author` | §10.7 |

### 3.4 Deliberate, documented divergences

| `@id` | Tool | Upstream | Why |
| --- | --- | --- | --- |
| `cites` | `role: link` | `role: edge` (§10.6) | §10.6's own entry says "recorded under its `## Cites` block" and §11.2's example writes that heading — both `link` behaviour (§5). Merge corrected, role kept. File upstream. |
| `granularity`, `heading`, `period`, `indexed`, `scoreZ`, `scoreC`, `subClassOf` | registered | unregistered | `arc:`-native extensions, or (for `granularity`) used by §11.5's example with no §10 entry. Registering them is more conformant, not less. |
| `definition`, `relevance` | `firstWriteWin` | *merge unstated* | §9.1 makes `merge` mandatory; the tool must pick. See research D8. |

`@id` and `@type` remain deliberately unregistered: `internal/core.identityFields` strips them
before a `Node` is constructed, so nothing ever resolves them through `core.Index.Predicates`.

---

## 4. `CoreTypeDefs` — all 8 lose `Merge`, 5 change contract

```go
"Source": {
	Required:    []string{"title", "published", "abstract", "mentions"},
	Optional:    []string{"authors", "url", "cites", "tags", "doi", "indexed"},
	Description: "A node for one ingested document — the provenance origin other nodes derive from.",
},
"Entity": {
	Required:    []string{"category", "definition", "mentionedIn"},
	Optional:    []string{"aliases", "tags", "notes", "indexed", "mentions",
	                      /* §10.5 semantic predicates, unchanged */},
	Description: "A node for a subject occurring in sources, typed by Sowa category.",
},
"Resource": {
	Required:    []string{"text", "tags", "mentionedIn"},
	Optional:    []string{"notes", "indexed"},
},
"Timeline": {
	Required:    []string{"cites"},
	Optional:    []string{"granularity", "period", "heading", "indexed", "mentions", "mentionedIn"},
},
"Reference": {
	Required:    []string{"title"},
	Optional:    []string{"url", "authors", "year", "doi", "isCitedBy",
	                      "ref", "relevance", "status", "notes", "indexed"},
},
"Node": {
	Required:    []string{},
	Optional:    []string{"published", "created", "tags", "text", "updated", "scoreZ", "scoreC"},
},
"Property": {
	Required:    []string{"role", "merge", "description"},
	Optional:    []string{"label", "aligned"},
},
"Class": {
	Required:    []string{"description"},
	Optional:    []string{"required", "optional", "subClassOf"},
},
```

### Change rationale, per type

- **`Source`** — `published` promoted from the `Node` base into `Source`'s own `Requires`
  (§11.2); `tags` added to Optional (§11.2). Net effect for `Source` is neutral: it required
  `published` by inheritance before and requires it directly now.
- **`Entity`** — `tags` added to Optional (§11.3). Requires unchanged.
- **`Resource`** — unchanged (spec 022 already landed §11.4).
- **`Timeline`** — `granularity` and `period` demoted to Optional (§11.5 requires `cites`
  alone). §11.5's own example carries `granularity` and no `period`, so both must stay declared.
- **`Reference`** — `ref` and `relevance` demoted to Optional per v0.11 §11.6's `Class` block;
  `ref`, `relevance`, `status`, `notes` retained as Optional so §11.6's own example conforms.
  **Supersedes spec 022's Clarification** (research D3).
- **`Node`** — Requires emptied. §11.1 states every node carries `@id`/`@type` and nothing else
  universally. `published`/`created` move to Optional, where they remain declarable on every
  type by inheritance.
- **`Property`** — unchanged (§9.1).
- **`Class`** — `description` promoted to Requires (§9.2 "description (mandatory)"). Safe:
  `decodeTypeDef` already rejects a `Class` document with an empty description, so no loadable
  graph can violate the new requirement. `subClassOf` added to Optional — the tool writes it on
  five seeded types (`CoreTypeBases`) while `Class` never declared it, which is itself a
  self-inflicted `typeOptional` violation on the seeded tree.

`CoreTypeBases` is unchanged: the five content types keep their explicit `subClassOf: Node`.
`Node` surviving with an empty `Requires` keeps the inheritance mechanism intact while removing
the over-broad contract — removing `Node` itself is out of scope (spec Out of Scope).

---

## 5. Effective-contract consequences

`resolveEffectiveTypes` unions each type's own lists with its bases', then subtracts `Required`
from `Optional`. With `Node.Required` empty:

| Type | Effective Required (after) | Effective Required (before) |
| --- | --- | --- |
| `Source` | `title, published, abstract, mentions` | `title, abstract, mentions, published, created` |
| `Entity` | `category, definition, mentionedIn` | + `published, created` |
| `Resource` | `text, tags, mentionedIn` | + `published, created` |
| `Timeline` | `cites` | `granularity, cites, period, published, created` |
| `Reference` | `title` | `title, ref, relevance, published, created` |

Every row is a strict relaxation. **No node that lints clean today can start failing** — which
is what makes User Story 3 safe to ship independently of the migration.

---

## 6. `kernel.UpgradeResult` — REMOVED

> **Removed 2026-08-23, post-implementation.** This section described the domain value for an
> `arc upgrade` migration command. The command was implemented, then removed: `arc` is pre-1.0
> and experimental, and a dedicated compatibility/migration path for graphs seeded by a
> previous release was judged unnecessary tech debt. No replacement value exists — the
> merge-vocabulary closure this feature makes (§1) is a plain breaking change with no remedy
> command.

---

## 7. Validation rules

| Rule | Where | Behaviour |
| --- | --- | --- |
| Merge value ∈ menu of six | `decodePredicateDef` via `validMergeOps` | reject; error names the offending document |
| Merge value on a `Class` node | `decodeTypeDef` | ignored, never rejected, never linted (FR-006) |
| Role ∈ meta/text/href/edge/link | `validRoles` | unchanged |
| `Class` description non-empty | `decodeTypeDef` | unchanged — now also expressed as `Class.Requires` |
