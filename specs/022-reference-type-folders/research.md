# Phase 0 Research: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Feature**: `022-reference-type-folders` · **Date**: 2026-08-23
**Baseline**: `f64863b`, `go1.27.0`, `go build ./... && go test ./...` green before any change.

This document does two things: it records the **deep impact analysis of the new folder
structure** requested with `/speckit-plan`, and it resolves every design decision the plan
depends on. Every claim below was verified against the source at the baseline commit, not
inferred from documentation.

---

## 0. Provenance of the planning input — read this first

The `/speckit-plan` input was pasted from [`CORE-FIX.md`](../../CORE-FIX.md) §4.2–§4.7, a
program brief whose own header declares it an *"analysis pass over ARCNET-CORE.md **v0.10**
(Draft, 2026-08-21)"*. `CORE-FIX.md` §4.1 was later updated with v0.11's §6 folder rule, but
§4.2–§4.7 were **not**. The pasted material is therefore internally inconsistent with its own
§4.1, and three of its instructions are superseded:

| Pasted instruction | Status | Superseded by |
|---|---|---|
| `coreKindFolders` — *add `Reference` → `references`* | **Wrong twice.** Wrong case, and the map itself becomes dead code (§2.1). | D1 |
| `DefaultLayout.Folders` — *add `references`* | Wrong case; and five existing entries also rename. | D1, §1.1 |
| Unit test: `nodeFolder("Reference") == "references"` | Wrong case; and it is no longer a special case worth a bespoke assertion. | D1, §6 |
| `arc migrate` with a versioned migration registry — *RECOMMENDED* | Out of scope. | D6 |
| Commit subject `graph(migrate): core-v0.10` | No migration command exists, so no such commit. | D6 |
| E2E: *migration on a fixture graph covering all four §4.4 classification rows* | No migration; §4.4's classification table has no consumer. | D6 |
| `FR-008` = *node re-typing* | `spec.md`'s FR-008 is the `Node` description. The pasted FR numbering is `CORE-FIX.md` §4.3's, not this feature's. | — |

Everything else the input supplied — the touch-point list, the union-cannot-retract warning,
the `TextPredicateFor` constraint, the render round-trip constraint, the string-not-stat test
rule — is correct, was verified, and is carried into the plan. Two of the constraints resolve
to a *smaller* problem than stated (§3), and the verification turned up **three hazards the
input did not name** (§2.3, §3.3, §4.2).

---

## 1. Impact analysis: the new folder structure

### 1.1 The rename set

| # | Current | v0.11 | Kind |
|---|---|---|---|
| 1 | `sources/` | `Source/` | type folder |
| 2 | `entities/` | `Entity/` | type folder |
| 3 | `resources/` | `Resource/` | type folder |
| 4 | — | `Reference/` | type folder (**new**) |
| 5 | `_schema/predicates/` | `_schema/Property/` | type folder under a namespace prefix |
| 6 | `_schema/types/` | `_schema/Class/` | type folder under a namespace prefix |
| 7 | `timeline/yearly/`, `timeline/monthly/` | *unchanged* | exempt — bucketed index |
| 8 | `.arc/`, `_schema/` itself | *unchanged* | not type folders |

Six renames, one addition. Rows 5 and 6 are the load-bearing ones: they are what makes the
break fail closed (§5).

### 1.2 Every derivation site, verified

There are exactly **four** places a folder name is produced, and they are not equivalent:

| Site | Role | Change |
|---|---|---|
| [`apply.go:32-56`](../../internal/app/graph/service/apply.go#L32-L56) — `coreKindFolders` + `nodeFolder`/`nodePath` | **The** node-path derivation. Everything that reads or writes a node file routes through it. | Rewritten (D1) |
| [`ctrl/kernel/graph.go:31-42`](../../internal/app/ctrl/kernel/graph.go#L31-L42) — `DefaultLayout.Folders` | The static list `arc init` creates. Independent of `nodeFolder` — **the two can silently disagree**. | Rewritten (§2.3) |
| [`schema/kernel/schema.go:18-21`](../../internal/app/schema/kernel/schema.go#L18-L21) — `TypesDir`/`PredicatesDir` | The two `_schema/` paths. Already constants, already the single source for both seeding and resolution. | Two string literals (D2) |
| [`revert.go:317-323`](../../internal/app/graph/service/revert.go#L317-L323) — `referrerPath` | Delegates to `nodePath`, diverting only `Timeline`. | **No change** (§2.2) |

### 1.3 What is *not* affected — and why that is the important finding

The folder rename's blast radius is far smaller than the 20-plus files that mention a folder
literal, because of three structural properties that hold at the baseline:

**(a) Nothing infers `@type` from a folder.** Swept every `filepath.Dir` / `path.Dir` /
`strings.Split` call in `internal/` and `cmd/`: not one derives a node's type from its path.
`lint`'s `kindIndex` is built from parsed `@type` values. This is CORE §6's "a consumer MUST
read `@type` from the node, never infer it from the folder" — already satisfied, by accident
of good design. **Consequence: renaming folders cannot change any node's type, and cannot
break any type-dependent behaviour.**

**(b) Traversal is layout-agnostic.** `lint` and `grep` walk every `*.md` under the root and
skip exactly two names — `.arc` and `_schema`
([`lint.go:219`](../../internal/app/lint/service/lint.go#L219),
[`grep.go:49`](../../internal/app/graph/service/grep.go#L49)) — matching on the folder *name*,
not on a whitelist of content folders. A folder called `Source/` is discovered on exactly the
same terms as one called `sources/`. **Consequence: no traversal, indexing, or search code
changes at all.** `serve.go` (MCP) contains no folder literal whatsoever.

**(c) Links resolve by basename.** CORE §4.2, and the code agrees: `@id` is the link target;
the path is never part of edge resolution. **Consequence: no edge breaks, and no backlink
sweep changes.**

Taken together: the folder rename is **four literal-string edits plus one function rewrite**.
The remaining work is test-fixture churn (§6), not production logic.

---

## 2. Decisions — folder derivation

### D1. `nodeFolder` becomes the identity function; `coreKindFolders` is deleted

**Decision.** Delete the `coreKindFolders` map. `nodeFolder(kind)` returns `kind` unchanged.

```
func nodeFolder(kind string) string { return kind }
```

**Rationale.** Under v0.10 the map existed to translate `Source`→`sources`, and the
`strings.HasSuffix(kind, "s")` / `kind + "s"` fallback existed to pluralize everything else.
v0.11 §6 makes the folder name *equal* the type name, so both the map and the fallback compute
the identity. Adding `"Reference": "references"` to the map — as the pasted input directs —
would reintroduce exactly the special-casing v0.11 exists to remove, and would make `Reference`
the one core type whose folder is not its name.

**This is why the pasted `References/` vs `references/` case-sensitivity warning dissolves.**
That hazard was a property of the `kind + "s"` fallback. With no fallback, there is no
transform, no case change, and nothing for a case-insensitive filesystem to hide. The
`--- ` `nodeFolder("Reference") == "references"` test the input proposes would assert a bug.

**Alternatives rejected.**
- *Keep the map, add `Reference`.* Rejected: preserves a table whose every entry is now
  `k → k`, and leaves the pluralizing fallback live to misfile every domain-profile type.
- *Lowercase the fallback.* Rejected, and `CORE-FIX.md` §4.6 already flags why: it would
  silently re-file every existing domain-profile type's folder. It also contradicts §6.

**Retained behaviour.** `nodeFolder` is still never called with `Timeline` — `Apply`'s per-node
loop intercepts the timeline kind upstream, and `referrerPath` diverts a `Timeline` referrer.
The existing comment documenting that invariant stays accurate and must be preserved.

### D2. `TypesDir`/`PredicatesDir` keep their Go names; only their values change

**Decision.** `PredicatesDir = "_schema/Property"`, `TypesDir = "_schema/Class"`. Do not rename
the Go identifiers.

**Rationale.** The constants are already the single source of truth for both `Seed()` and
`Resolve()`, so two string edits propagate everywhere correctly. Renaming the identifiers to
`PropertyDir`/`ClassDir` would touch a dozen call sites for zero behavioural gain and inflate
the diff around the genuinely risky changes. The identifiers name the *concept* (the folder of
type nodes); the values name the *path*. Their GoDoc must be updated to state the new paths.

### D3 (hazard the input did not name). `DefaultLayout.Folders` and `nodeFolder` are independent and can drift

`arc init` creates folders from a static list; `arc apply` writes to a path computed by
`nodeFolder`. Nothing ties them together — today they agree only because a human kept them in
sync. A rename that updates one and not the other yields a graph whose `arc init` creates
`Resource/` while `arc apply` writes to `resources/`, with **no error**: `fsys` creates parent
directories on write ([`local.go:41`](../../internal/adapter/fsys/local.go#L41)), so the wrong
folder is created silently on first use.

**Decision.** Add a test asserting that every content type in `CoreTypeDefs` with a base in
`CoreTypeBases` has `nodeFolder(type)` present in `DefaultLayout.Folders`, and vice versa. This
converts a convention into a checked invariant and is the single highest-value new test in the
feature. It also fails loudly if a future type is added to one list and not the other.

### D4. `pluralizeKind` is display-only and MUST NOT be touched

[`cmd/arc/graph/apply.go:31`](../../cmd/arc/graph/apply.go#L31) defines `pluralizeKind(kind,
count)`, which special-cases `Entity → entities` and appends `s` otherwise. It **looks** exactly
like a folder deriver, sits in the file named `apply.go`, and is 40 lines from real path code.
Its only call site is [line 74](../../cmd/arc/graph/apply.go#L74):
`fmt.Sprintf("+%d %s", created, pluralizeKind(k, created))` — the human-readable ingest summary
("`+3 entities`").

**Decision.** Leave it exactly as it is, and add a GoDoc line saying it formats a count for
display and is unrelated to folder derivation. Changing it to identity would print "`+3
Entity`". This is the most likely wrong edit in the whole feature; the touch-point list does
not mention it, which is precisely why it needs naming here.

---

## 3. Decisions — body-prose predicate

### D5. The table changes in one place; the parse/render round trip is safe by construction

**Decision.** In [`markdown.go:384`](../../internal/core/markdown.go#L384), `textPredicateFor`'s
leading switch becomes `Source→abstract`, `Entity→definition`, `Resource→text`,
`Reference→relevance`, default `text`. Trailing stays unconditionally `notes`.

**Verified: the render round-trip constraint is weaker than stated.** The input asks for a
parse→render→parse fixture per affected type because "a mismatch between parse and render keys
silently drops body prose". Both sides call the *same private function* —
parse at [`markdown.go:844,847`](../../internal/core/markdown.go#L844), render at
[`markdown.go:1201,1202`](../../internal/core/markdown.go#L1201). They **cannot** disagree with
each other; one edit moves both. The fixtures are still worth adding (they pin the *observable*
key, which is the actual requirement), but they are regression cover, not a mismatch hunt.

**Also verified: existing on-disk `Resource` files round-trip byte-identically.** An old
`Resource` file stores its leading prose as bare paragraph text, not as a `## Relevance` block.
Re-parsed under the new table it lands in `Texts["text"]`; rendered back, `text` is the leading
key, so it is written bare again. The stored *bytes* are unchanged; only the in-memory key —
and therefore what type-conformance lint sees — changes. That is exactly the intended effect
and it carries no file-rewrite risk.

### D6a (constraint resolved). The `TextPredicateFor` auto-discovery concern is a non-issue on any seeded graph

[`apply.go:131-132`](../../internal/app/graph/service/apply.go#L131-L132) uses
`TextPredicateFor` to skip the two reserved prose keys before auto-registering discovered
predicates. Changing the `Resource` case does change which key is skipped: `relevance` is no
longer skipped for a `Resource`, and `text` now is.

**Why it does not matter.** Auto-registration goes through `RegisterPredicate` →
`registerIfAbsent`, which **never overwrites an existing document**
([`schema.go:386-395`](../../internal/app/schema/service/schema.go#L386-L395)). Both `text` and
`relevance` are members of `CorePredicateDefs` and are therefore seeded into every graph by
`arc init`. So the newly-unskipped `relevance` resolves to "already present, no write". Net
behaviour change on any graph `arc init` produced: **none**.

The one graph where it would matter is one whose `_schema/Property/` was hand-trimmed to remove
`relevance`. There, a `Resource` carrying `relevance` would now auto-register it with
`role: text, merge: append` — which is the correct definition anyway. No action needed; recorded
so the question is not re-litigated.

### D6b (hazard the input did not name). `revertLeadingKey` is a hand-copied duplicate that is **already out of sync**

[`revert.go:398`](../../internal/app/graph/service/revert.go#L398) carries `revertLeadingKey`, a
second copy of the table, explicitly documented as a duplicate to be kept in sync. Comparing
the two at baseline:

| `@type` | `core.textPredicateFor` | `revert.revertLeadingKey` |
|---|---|---|
| `Source` | `abstract` | `abstract` |
| `Entity` | `definition` | `definition` |
| `Resource` | `relevance` | `relevance` |
| `hypothesis` | `text` *(default)* | **`claim`** |
| `aporia` | `text` *(default)* | **`tension`** |
| `thought` | `text` *(default)* | **`claim`** |

**The tables already disagree on three types.** A domain-profile `hypothesis` node has its
prose written under `text` by apply and looked for under `claim` by revert. This is a live,
pre-existing defect of exactly the class the input's constraint warns about — the constraint is
right, it is just already violated, and not by `Resource`.

**Decision.** Update `revertLeadingKey`'s `Resource` case and add its `Reference` case, keeping
this feature's scope. Do **not** fix the three stale domain entries here — that is an unrelated
pre-existing bug touching domain profiles this feature does not otherwise address. Instead:
(1) note it in the plan's follow-ups, and (2) add a test that asserts the two tables agree for
all five *core* types, so this feature's half is locked down and the divergence cannot spread to
core. Deleting the duplicate entirely (having `revert` call `core.TextPredicateFor`) is the real
fix and is the recommended follow-up; it is out of scope here only because the three domain
entries would change behaviour for domain-profile graphs.

---

## 4. Decisions — type vocabulary

### D7. `Resource` and `Reference` definitions

Per `spec.md` FR-001/FR-004, with the clarification resolved in favour of the upstream revision
note (see `spec.md` § Clarifications):

| | `Resource` | `Reference` |
|---|---|---|
| Required | `text`, `tags`, `mentionedIn` | `title`, `ref`, `relevance` |
| Optional | `notes` | `url`, `authors`, `year`, `doi`, `status`, `isCitedBy`, `notes` |
| `subClassOf` | `Node` | `Node` |
| Type merge | unchanged (`firstWriteWin`) | `firstWriteWin` (inherits `Resource`'s former op — the op belonged to the external-work semantics) |

`Resource` sheds the structural/semantic optionals it carried under the old definition
(`indexed`, `mentions`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`,
`isReplacedBy`, `conformsTo`, `related`, `referencedBy`) — §11.4 lists `notes` alone. `Reference`
does not inherit them either; §11.6 lists no semantic predicates.

### D8. Predicate description rewording

`ref`, `status`, `relevance` describe external-work semantics while naming "resource". Reword to
name `Reference`. `mentionedIn` is already worded generically and needs no change beyond
confirming it covers `Resource` as well as `Entity`. `cites`/`isCitedBy` mention "cited
resource" — reword to "cited reference". `Node`'s description enumerates the four inheriting
content types and must name five.

### D9 (hazard the input did not name). Seeding is pure render — the union problem does not reach `arc init`

The input directs: *"Verify this before designing anything else."* Verified, and the answer
splits in two.

- **`Seed()` does not merge.** [`schema.go:57-77`](../../internal/app/schema/service/schema.go#L57-L77)
  renders each `CoreTypeDefs` entry straight to bytes; `arc init` writes those bytes to fresh
  paths. **A new graph receives the exact new definitions.** The union problem is structurally
  incapable of reaching `arc init`.
- **`ApplySchema` does merge.** [`apply.go:156`](../../internal/app/schema/service/apply.go#L156)
  calls `core.Merge(existing, node, metaIndex, sourceID)` with `metaIndex.Predicates =
  CorePredicateDefs`, in which `required` and `optional` are `merge: union`. Union has no
  inverse. So re-applying a corrected `Resource` onto an existing one yields a type requiring
  `ref + relevance + text + tags + mentionedIn` — the input's claim is **confirmed exactly**.

**Consequence for this feature:** the union defect is real but unreachable, because the only
path that installs the new definitions is `Seed()`, and the only graphs that get them are new
ones. `spec.md` FR-021 states this; no code guards against it because no code can reach it.

---

## 5. The breaking change: verified to fail closed

`spec.md` originally assumed a pre-feature graph would silently grow a second folder set. **That
assumption was wrong and has been corrected in the spec.** Verified behaviour:

`Resolve()` ([`schema.go:127`](../../internal/app/schema/service/schema.go#L127)) stats `.arc`,
then calls `resolvePredicates` and `resolveTypes`, each of which calls `readSchemaDir`
([`schema.go:468`](../../internal/app/schema/service/schema.go#L468)):

```
entries, err := store.ReadDir(dir)
if err != nil { return nil, ErrSchemaMissing.With(err, dir) }
```

A pre-feature graph has `_schema/predicates/` and `_schema/types/`, not `_schema/Property/` and
`_schema/Class/`. `ReadDir` fails → `ErrSchemaMissing` naming the exact missing folder → the
whole `Resolve` fails → **every command that resolves the schema refuses before writing
anything**. `Resolve`'s own contract is explicit: *"A missing schema folder … fails the entire
load — never skipped … Never returns a partially-populated Index."*

This is the strongest possible outcome for a no-migration break: loud, immediate, path-naming,
and with a zero-width window for partial writes. It is why D6 (no migration) is safe.

### D6. No migration command — and why the pasted recommendation is declined

**Decision.** No `arc migrate`, no `--replace` flag, no repair path. Confirmed by the user in
the `/speckit-specify` clarification: *"No migration! We are breaking compatibility. It is
fine."*

The pasted input recommends option (a), an `arc migrate` with a versioned migration registry,
on the argument that CORE is Draft with four breaking revisions in six releases and the
mechanism will be needed again. That argument is sound in the abstract and is recorded here so
it is not lost — but it is a *future* mechanism argument, not a requirement of this feature, and
the user has ruled on scope. Two facts make the ruling comfortable rather than merely binding:

1. §5 — the break fails closed and names the missing folder. Nobody loses data to it.
2. §D9 — a fresh graph gets exactly-correct definitions with no union residue.

The only concession retained is error-message quality: `ErrSchemaMissing`'s text should read
well when the cause is "this graph predates the folder rename". That is wording, not migration.

---

## 6. Test strategy

The pasted strategy is adopted with three corrections and two additions.

| Test | Level | Notes |
|---|---|---|
| Schema seed golden files for `Resource` + `Reference` | unit | as proposed |
| `textPredicateFor` table, five core types × {leading, trailing} | unit | as proposed (10 cases) |
| ~~`nodeFolder("Reference") == "references"`~~ → `nodeFolder(t) == t` for every core type **and** a domain type | unit | **corrected** (D1): assert identity, not a lowercase special case |
| **`DefaultLayout.Folders` ↔ `nodeFolder` agreement** | unit | **new** (D3) — the drift this feature is most likely to introduce |
| **`core.TextPredicateFor` ↔ `revertLeadingKey` agreement for the five core types** | unit | **new** (D6b) — locks down an already-divergent duplicate |
| parse→render→parse fixture per affected type | unit | keep; pins the observable key (D5 explains it cannot catch a parse/render mismatch) |
| Node lands at `<Type>/<id>.md` for all five types + a domain type | E2E | assert the **exact path string returned by the store**, never a filesystem stat — APFS is case-insensitive and would hide a case error |
| `arc init` creates exactly the eight folders | E2E | string set equality |
| `arc lint` clean on a freshly initialized graph | E2E | replaces the proposed "clean on a *migrated* graph" |
| `arc apply` → `arc revert` round trip for `Resource` and `Reference` | E2E | covers `revertLeadingKey` and `referrerPath` together |
| ~~migration over §4.4's four classification rows~~ | — | **dropped** (D6) |

**The string-not-stat rule is retained and generalized.** Even though D1 removes the specific
`References/`-vs-`references/` hazard, every path assertion in this feature must compare the
exact string the store was asked for. On the developer's APFS machine a stat-based assertion
passes for `Source/` *and* `source/` *and* `SOURCE/`; on CI's ext4 it would not. This is the
single rule most likely to let a real defect through unnoticed.

### Test-fixture blast radius

28 test files contain a folder literal. Ranked by occurrence count, the concentration is:

```
74  cmd/arc/graph/apply_test.go          22  internal/app/graph/service/apply_test.go
52  cmd/arc/lint/lint_test.go            16  cmd/arc/graph/revert_test.go
37  internal/app/graph/service/subgraph_test.go   14  internal/app/graph/service/grep_test.go
35  internal/app/graph/service/revert_test.go     10  internal/app/lint/service/rules_links_test.go
26  cmd/arc/graph/subgraph_test.go               10  cmd/arc/ctrl/init_test.go
26  cmd/arc/graph/batch_test.go                   … 18 more files, ≤9 each
```

This is mechanical churn, not design risk: the literals are fixture paths, and the rename is a
straight substitution. It is nonetheless the bulk of the feature's diff, and `tasks.md` should
treat it as its own tranche rather than smearing it across the behavioural tasks. The three
`testdata/` patch fixtures under `cmd/arc/graph/testdata/` were checked and contain patch
manifests only — no folder paths — so none needs editing.

---

## 7. Follow-ups deliberately not taken here

1. **Delete `revertLeadingKey`**, replacing it with a call to `core.TextPredicateFor`. Fixes the
   three-type divergence in D6b. Needs its own spec: it changes behaviour for domain-profile
   graphs.
2. **A folder-conformance lint rule** (`RuleFolderMirrorsType`). v0.11 §6 exists to make the rule
   machine-checkable; making it *true* is this feature, making it *checked* is the next.
3. **The remaining v0.11 drift** — `published` required on `Source`, `tags` optional on
   `Source`/`Entity`, `Timeline` requiring `cites` alone. `CORE-FIX.md` scopes this as its
   Feature 023.
4. **`ErrSchemaMissing` wording** for the predates-the-rename case, if the plain message proves
   confusing in practice.
