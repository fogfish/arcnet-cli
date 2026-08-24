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
| ~~`year`~~ | `fillIfEmpty` | ~~**`immutable`**~~ | §10.7 — **retired outright under CORE 0.12; see §8** |
| ~~`definition`~~ | `append` | ~~**`firstWriteWin`**~~ | §10.2 rationale (research D8) — **renamed to `text`/`MergeAppend` under CORE 0.12; see §8** |
| `relevance` | `append` | **`firstWriteWin`** | §10.2 rationale (research D8) — unaffected by CORE 0.12 (BUG-001 F7) |
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
| ~~`authors`~~ | ~~`schema:author`~~ | §10.7 — **`authors` retired under CORE 0.12; the alignment moves to the surviving singular `author` (already aligned, §3.1). See §8.** |

### 3.4 Deliberate, documented divergences

| `@id` | Tool | Upstream | Why |
| --- | --- | --- | --- |
| `cites` | `role: link` | `role: edge` (§10.6) | §10.6's own entry says "recorded under its `## Cites` block" and §11.2's example writes that heading — both `link` behaviour (§5). Merge corrected, role kept. File upstream. |
| ~~`granularity`~~, `heading`, `period`, `indexed`, `scoreZ`, `scoreC`, `subClassOf` | registered | unregistered | `arc:`-native extensions, or (for `granularity`) used by §11.5's example with no §10 entry. Registering them is more conformant, not less. **`granularity` retired outright under CORE 0.12 — no longer "more conformant than the spec," the spec caught up. See §8.** |
| ~~`definition`~~, `relevance` | `firstWriteWin` | *merge unstated* | §9.1 makes `merge` mandatory; the tool must pick. See research D8. **`definition` renamed to `text` under CORE 0.12, which is `MergeAppend`, not `firstWriteWin` — see §8.** |

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

---

## 8. ARCNET-CORE v0.12 predicate retirement ([BUG-001](bugs/BUG-001.md), added 2026-08-23)

CORE 0.12 resolves, from the spec side, the same seven-predicate ambiguity §3.4 above resolved
from the tool side ("registering them is more conformant, not less"). Six of the seven are now
renamed or retired; `relevance` (the seventh) is unaffected — see §3.2/§3.4 strikethroughs above.
**Round 2** (bugfix-verify) resolved the one open design question — the `definition`→`text` merge
collision — and corrected an over-broad compatibility claim; both are reflected below.

### 8.1 `CorePredicateDefs` corrections

| `@id` | Change |
| --- | --- |
| `definition` | **Removed.** `Entity`'s leading-prose predicate becomes `text` (already registered, `MergeAppend` — the same entry `Resource` uses). No new predicate is added. |
| `authors` | **Removed.** `author` (§3.1) is the sole authorship predicate; its existing `schema:author` alignment already covers this. |
| `year` | **Removed.** `Reference`'s Optional list points at the already-registered `published` (`MergeImmutable`) instead. |
| `notes` | **Removed** from `Entity`/`Resource`'s Optional lists only — the `@id` itself is not deleted from `CorePredicateDefs` if any other type still legitimately uses it; audit at implementation time (T068). |
| `ref` | **Removed** from `CorePredicateDefs` and from `Reference`'s Optional list. |
| `status` | **Removed** from `CorePredicateDefs` and from `Reference`'s Optional list. |
| `granularity` | **Removed** from `CorePredicateDefs` and from `Timeline`'s Optional list. |

### 8.2 The `definition`→`text` merge collision — resolved

`textPredicateFor` (`internal/core/markdown.go`) already returns the literal `"text"` for
`Resource`'s leading prose, and that predicate is `MergeAppend` — not the `MergeFirstWriteWin` §3.2
originally gave `definition`. **Decision (round 2): `text` stays `MergeAppend` uniformly.** No
per-type or role-qualified merge override is introduced. This is a deliberate regression of
`Entity`'s first-fixed prose protection (§5's effective-contract table, and spec.md US1 Acceptance
Scenario 4, now struck) — accepted rather than adding merge-scoping complexity the six-value,
predicate-name-keyed algebra (§1) does not otherwise have. `Reference`'s `relevance` is unaffected
and keeps `MergeFirstWriteWin`.

### 8.3 `CoreTypeDefs` corrections (supersedes the relevant §4 rows)

```go
"Entity": {
	Required:    []string{"category", "text", "mentionedIn"},   // was "definition"
	Optional:    []string{"aliases", "tags", "indexed", "mentions", /* §10.5 unchanged */},
	                      // "notes" removed
},
"Resource": {
	Required:    []string{"text", "tags", "mentionedIn"},        // unchanged
	Optional:    []string{"indexed"},                            // "notes" removed
},
"Timeline": {
	Required:    []string{"cites"},                               // unchanged
	Optional:    []string{"period", "heading", "indexed", "mentions", "mentionedIn"},
	                      // "granularity" removed
},
"Reference": {
	Required:    []string{"title"},                               // unchanged
	Optional:    []string{"url", "author", "published", "doi", "isCitedBy", "relevance", "indexed"},
	                      // "authors"→"author", "year"→"published", "ref"/"status"/"notes" removed
},
```

### 8.4 Compatibility — corrected (round 2)

The retirement of `definition`/`authors`/`year`/`notes`/`ref`/`status`/`granularity` is a **plain
breaking change**, identical in kind to §1's `MergeValidatedOverwrite` deletion — no
reader-leniency-for-one-release mechanism is introduced. A first draft of this section proposed
reader leniency; that was corrected during bugfix-verify because it directly contradicted
`ARCHITECTURE.md`'s Compatibility Policy and this feature's own §6 decision not to carry migration
machinery for a pre-1.0 tool. Consistency with §6 was the deciding factor.

### 8.5 Effective-contract consequences (supersedes relevant §5 rows)

| Type | Effective Required (round 2) | Effective Required (§5, pre-BUG-001) |
| --- | --- | --- |
| `Entity` | `category, text, mentionedIn` | `category, definition, mentionedIn` |
| `Reference` | `title` | `title` (unchanged — `ref`/`relevance` were already Optional per §4) |

`Timeline`'s effective Required (`cites`) and `Resource`'s (`text, tags, mentionedIn`) are
unchanged by §8 — only their Optional lists lose `granularity`/`notes` respectively.
