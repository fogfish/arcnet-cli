# Phase 0 Research: Schema Vocabulary Conformance

**Feature**: `023-core-vocabulary-conformance` · **Date**: 2026-08-23

**Upstream authority**: [ARCNET-CORE.md](https://github.com/fogfish/arcnet-spec/blob/main/ARCNET-CORE.md)
— fetched 2026-08-23, **Status: Draft · Version: 0.11 · Date: 2026-08-23**.

**Tool baseline**: `internal/app/schema/kernel/schema.go` @ `db21b39` (post-022).

---

## D0. The upstream document is v0.11, not v0.10

`spec.md`, `CORE-FIX.md`, and this feature's branch name all say **v0.10**. The document
published at the URL the user supplied is **v0.11, dated 2026-08-23** — the same day. Feature
`022-reference-type-folders` already targeted v0.11 (its §6 folder rule), so the repository is
mid-way between the two revisions in its own documentation.

**Decision**: validate against **v0.11** and retitle this feature accordingly. Every finding in
`spec.md` survives the revision; three of the workarounds `CORE-FIX.md` proposed do not,
because v0.11 fixed the upstream defects they were working around (D1, D2, D3 below).

**Rationale**: the user's plan input names the live URL as the thing to validate against. A
conformance feature that targets a superseded revision of its own reference document is
self-defeating.

**Section-number remapping** (v0.10 → v0.11), for every citation carried forward from `spec.md`:

| Concept | v0.10 cite | v0.11 cite |
| --- | --- | --- |
| Merge vocabulary (menu of six) | §9.3 | §9.3 *(unchanged)* |
| Predicate node format | §9.1 | §9.1 *(unchanged)* |
| Type node format | §9.2 | §9.2 *(unchanged)* |
| Content predicates | §10.2 | §10.2 + **§10.7 (new)** |
| Citation predicates | §10.6 | §10.6 *(unchanged)* |
| Schema predicates | §10.8 | §10.8 *(unchanged)* |
| Core types | §11.x | §11.x *(unchanged)* |
| Idempotency of apply | §14.3 | **§13.2** (ingestion idempotency check) |
| Domain profiles | §15 | **§14** |

---

## D1. `CORE-FIX.md` §8.1 is obsolete — §10.7 now exists

`CORE-FIX.md` recorded that "§10.7 does not exist" and that eight cross-references dangle.
v0.11 adds **§10.7 "(Content predicates continued)"**, registering six of the seven predicates
that `CORE-FIX.md` §8.5 flagged as used-but-unregistered: `definition`, `relevance`, `authors`,
`year`, `status`, `ref`, `notes`.

**Decision**: drop the §8.5 "the tool is more conformant than the spec" carve-out for those six
and validate them against their now-normative registrations. One consequence is a real defect:
`year` is `fillIfEmpty` in the tool and **`immutable`** upstream (D6).

**Still unregistered upstream**: `granularity` — used by §11.5's own worked example, registered
by no §10 entry. The tool registers it; that remains more conformant than the spec, unchanged.

---

## D2. `CORE-FIX.md` §8.2 is obsolete — upstream registers *both* `author` and `authors`

`spec.md`'s Assumption "register both, revisit when the spec settles" was a hedge against an
upstream inconsistency. v0.11 settles it explicitly:

- **§10.2 `author`** — role `meta`, merge `union`, aligned `schema:author`. "The author of the content."
- **§10.7 `authors`** — role `meta`, merge `union`, aligned `schema:author`. "Document authors (array); `author` is the core vocabulary term."

**Decision**: register both, as the spec now does — but promote this from an Assumption to a
requirement traceable to §10.2/§10.7. `authors` is the predicate §11.2/§11.6 actually reference
in their `## Optional` lists; `author` is the core vocabulary term. Neither is an alias of the
other in any mechanical sense; both are ordinary registrations.

---

## D3. `CORE-FIX.md` §8.3 is obsolete, and reverses spec 022's clarification

`CORE-FIX.md` §8.3 tabulated three inconsistent statements about `Reference` in v0.10 and
recommended following the 0.9→0.10 revision note. Spec 022 adopted exactly that and recorded
it as a Clarification: *"Reference requires `title`, `ref`, `relevance`."*

**v0.11 §11.6's normative `Class` node says:**

```
## Requires
- required:: [[title]]

## Optional
- optional:: [[url]]
- optional:: [[authors]]
- optional:: [[year]]
- optional:: [[doi]]
- optional:: [[isCitedBy]]
```

The revision note that spec 022 relied on is gone; the `Class` block is now the only normative
statement. It requires `title` **alone**, and it lists neither `ref`, `relevance`, `status`, nor
`notes` — even though §11.6's own worked example carries `ref` and `status`, and its prose says
a `Reference` "records only enough to identify, locate, and **justify** keeping the pointer"
(justification being what `relevance` holds).

**Decision**: adopt v0.11's `Requires: title` and keep `ref`, `relevance`, `status`, `notes`,
`indexed` as **Optional**, alongside the five §11.6 lists.

**Rationale**:
- The retraction is a pure relaxation. No graph that lints clean today stops linting clean:
  requirements are removed, and everything removed from `Requires` reappears in `Optional`, so
  nothing becomes undeclared.
- Keeping the four as Optional is what makes §11.6's own example conform to §11.6's own type —
  the same reading spec 022 applied to a different inconsistency in the same section.
- Spec 022's prose keying (a `Reference`'s leading prose is stored under `relevance`) is
  unaffected: `relevance` stays declared, it simply stops being mandatory.

**This supersedes spec 022's recorded Clarification** and is called out as such in `plan.md`
rather than applied quietly — 022's decision was correct for the document that existed when it
was made.

**Alternative rejected**: keep `title`/`ref`/`relevance` required. This preserves 022's decision
but leaves the tool requiring two predicates the current spec does not, which is precisely the
false-positive class this feature exists to eliminate (spec US3).

---

## D4. `spec.md`'s US1 premise overstates the current damage

`spec.md` User Story 1 asserts that re-applying a patch "concatenates the summary onto itself"
and that "a third apply triples it". **That is not what the code does today.**

`internal/core/merge.go` `mergeText` (BUG-004) splits both sides into paragraphs and drops any
incoming paragraph whose Jaccard similarity over 3-word shingles against an existing paragraph
exceeds `0.8`. A byte-identical re-apply is therefore **already** a no-op for every `append`
text predicate, and `Merge`'s own doc comment already claims idempotency for every op except
`lastWriteWin`.

**What is actually broken**, and what changing `abstract`/`description` to `firstWriteWin` fixes:

| Scenario | Today (`append`) | After (`firstWriteWin`) |
| --- | --- | --- |
| Identical prose re-applied | no-op (near-duplicate guard) | no-op |
| **Reworded prose** (Jaccard ≤ 0.8) | **silently appended as a second paragraph** | first value kept, divergence flagged `needsReview` |
| Genuinely new prose from a second source | appended | first value kept, divergence flagged |

**Decision**: keep the requirement (§10.2 and §10.8 declare `firstWriteWin`; conformance is the
reason, and it is sufficient) but **correct `spec.md`'s rationale**. The real defect is that a
regenerated or lightly-edited abstract accumulates paragraphs instead of holding a single
first-fixed value — a slow drift, not a doubling on every run — and that the near-duplicate
threshold is a heuristic standing in for a merge behaviour the spec declares outright.

**Rationale for correcting rather than quietly proceeding**: US1's acceptance scenarios 1 and 2
pass against `main` today. Left as written they would be green before a line is changed, and
Principle VIII's 1:1 scenario-to-test mapping would certify a fix that was never made.

---

## D5. `spec.md`'s US4 premise is factually wrong — nothing is auto-registered

`spec.md` User Story 4 asserts that first use of `author`/`about`/`genre` "silently creates a
definition with a role and a merge behaviour the tool guessed".

`internal/app/graph/service/apply.go:113` `distinctPredicates` enumerates **`node.Edges` and
`node.Texts` only** — it never walks `node.Attrs`. `author`, `about`, and `genre` are all
`role: meta`, i.e. front-matter attributes. They are therefore **never passed to
`RegisterPredicate` at all**. No placeholder file is written; nothing is guessed.

**What actually happens today**: the three stay unregistered, `resolveMergeOp` falls through to
its `MergeUnion` default — which is coincidentally the correct merge for all three — and
`arc lint` reports each occurrence twice: once as `predicateRegistered` (unregistered predicate)
and once as `typeOptional` (undeclared on its type).

**Decision**: keep FR-011 (register all three; §10.2 requires it) and **rewrite FR-012 and US4's
scenarios** around the real symptom — lint noise on conformant nodes — instead of a placeholder
file that is never created. US4 merges naturally into US3's false-positive theme.

**Explicitly out of scope**: teaching `distinctPredicates` to walk `Attrs`. That would make
every unregistered front-matter key auto-register, a behaviour change far beyond this feature
and one with its own design questions (which role would it guess for a scalar?).

---

## D6. Full predicate delta, tool vs. upstream v0.11

Validated entry by entry against §10.1–§10.8. **▲** = correction, **+** = addition,
**✓** = already conformant (44 entries, elided).

| Predicate | Tool today | Upstream v0.11 | Action |
| --- | --- | --- | --- |
| `abstract` | `text` / **`append`** / — | `text` / **`firstWriteWin`** / `schema:abstract` | ▲ merge + aligned |
| `description` | `text` / **`append`** / — | `text` / **`firstWriteWin`** | ▲ merge |
| `cites` | `link` / **`append`** / `cito:cites` | `edge`¹ / **`union`** / `cito:cites` | ▲ merge only |
| `year` | `meta` / **`fillIfEmpty`** | `meta` / **`immutable`** (§10.7) | ▲ merge |
| `scoreZ`, `scoreC` | `meta` / **`validatedOverwrite`** | *(not upstream — `arc:` native)* | ▲ → `lastWriteWin` (D7) |
| `title` | `meta` / `immutable` / — | + `schema:title` | ▲ aligned |
| `url` | `meta` / `fillIfEmpty` / — | + `schema:url` | ▲ aligned |
| `doi` | `meta` / `fillIfEmpty` / — | + `schema:doi` | ▲ aligned |
| `aliases` | `meta` / `union` / — | + `skos:altLabel` (§10.1) | ▲ aligned |
| `authors` | `meta` / `union` / — | + `schema:author` (§10.7) | ▲ aligned |
| `author` | **absent** | `meta` / `union` / `schema:author` | + |
| `about` | **absent** | `meta` / `union` | + |
| `genre` | **absent** | `meta` / `union` | + |
| `definition` | `text` / `append` | `text` / **merge unstated** (§10.7) | ▲ → `firstWriteWin` (D8) |
| `relevance` | `text` / `append` | `text` / **merge unstated** (§10.7) | ▲ → `firstWriteWin` (D8) |
| `notes` | `text` / `append` | `text` / `append` (§10.7) | ✓ — but see D8 note |
| `granularity`, `heading`, `period`, `indexed`, `subClassOf` | `arc:`-native | unregistered upstream | ✓ keep |

¹ **`cites` role stays `link`.** §10.6 declares role `edge` and, in the same entry, says the
predicate is "recorded under its `## Cites` block" — which §5 defines as `link` behaviour.
§11.2's `Source` example writes a `## Cites` heading, confirming `link`. Only the merge changes.
Unchanged from `spec.md`'s Assumption; file the contradiction upstream.

**Aligned-term additions are not cosmetic.** `aligned` is a `recommended` front-matter field in
§9.1 and it is the only machine-readable bridge from a graph's own predicate to the standard
vocabulary term it maps to. Six seeded predicates carry a standard alignment upstream and none
in the tool.

---

## D7. `scoreZ` / `scoreC` — resolving `spec.md`'s open decision

`spec.md` assumed `lastWriteWin`. Reading the implementation confirms it and adds a reason the
assumption did not know about.

`mergeScalar` groups `validatedOverwrite` with `immutable` in its **freeze** class: existing,
once non-empty, is permanent. So the two score predicates are *today* effectively immutable —
they can never be recomputed, which directly contradicts their own registered description
("recomputed by a validation/ingest pass"). The seventh merge op is not merely non-conformant,
it does not do what it says.

**Decision**: `lastWriteWin`. **Rationale**: conformant (§9.3), and it is the only op in the
menu under which a recomputation pass can actually recompute. The "only a designated pass may
overwrite this" guard is policy; it belongs in whichever command performs the recomputation, not
in the merge algebra. **Alternatives rejected**: `fillIfEmpty` (makes scores write-once, the same
defect under a conformant name); dropping both predicates (they are `arc:`-native and harmless);
keeping `validatedOverwrite` as an extension (§9.3's menu is closed and §14 requires profile
predicates to draw from it).

---

## D8. `definition` and `relevance` — an upstream gap this feature must fill

§10.7 registers both with a role and an aligned term but **states no `merge`**. §9.1 makes
`merge` mandatory on every `Property` node, so the tool must choose one.

**Decision**: `firstWriteWin` for both. **Rationale**: §10.2's `text` entry states the rule
directly — "A type MAY instead declare its own, more specific text predicate (e.g. `abstract`,
`definition`, `relevance`, §10.7) when a precise name aids reading and **a single, first-fixed
value is wanted instead**." `definition` and `relevance` are named in that sentence, and
`abstract`, the third one named, is registered `firstWriteWin` in §10.2. The generic `text` and
the explicitly-`append` `notes` are the accumulating pair; the three named text predicates are
the first-fixed trio.

**Consequence**: this extends the idempotency correction beyond `abstract`/`description` to
`Entity.definition` and `Reference.relevance` — the leading-prose predicates for two of the five
core types (spec 022 body-prose keying). Without it, this feature would fix prose drift for
`Source` and leave the identical defect in place for `Entity` and `Reference`.

**Alternative rejected**: leave both `append` and file the gap upstream. That preserves a known
drift defect in two core types for one release, for no benefit.

---

## D9. Type-table delta, tool vs. upstream v0.11

| Type | Requires — tool | Requires — v0.11 | Optional delta |
| --- | --- | --- | --- |
| `Source` §11.2 | `title, abstract, mentions` | `title, `**`published`**`, abstract, mentions` | **+ `tags`** |
| `Entity` §11.3 | `category, definition, mentionedIn` ✓ | same | **+ `tags`** |
| `Resource` §11.4 | `text, tags, mentionedIn` ✓ | same | ✓ (`+ indexed`, arc) |
| `Timeline` §11.5 | **`granularity, cites, period`** | **`cites`** | ▲ demote `granularity`, `period` |
| `Reference` §11.6 | `title, `**`ref, relevance`** | **`title`** | ▲ + `ref, relevance, status, notes` (D3) |
| `Node` *(arc-native)* | **`published, created`** | *(§11.1: `@id`/`@type` only)* | ▲ **empty Requires** |
| `Property` §9.1 | `role, merge, description` ✓ | same | ✓ |
| `Class` §9.2 | *(none)* | **`description`** mandatory | ▲ promote `description` |

Every one of the eight loses its `merge:` attribute (§9.3 retired type-level merge; §10.8 scopes
`merge` to `Property`). `indexed` stays in every content type's Optional — it is `arc:`-native
and stamped by `arc apply` on every node it creates; dropping it would make the tool's own
output fail the tool's own check.

`Class` gaining `Requires: description` is safe: `decodeTypeDef` already rejects a `Class`
document with an empty description, so no loadable graph can violate it.

---

## D10. The migration deadlock, and the ordering that resolves it

Tightening `validMergeOps` to six (spec FR-002) makes **every graph seeded by a previous
release fail to load** — its own `_schema/Property/scoreZ.md` declares `validatedOverwrite`, and
`decodePredicateDef` rejects an unrecognized merge before anything else runs. The remedy is a
command; the command cannot run if it resolves the schema first.

**Decision**: `arc upgrade` writes the corrected seed **before** it resolves anything.

```
1. Locate graph root, verify .arc/ exists          — no schema read
2. Compute seed via schema.Seed()                  — pure, no I/O, no Resolve
3. Diff against on-disk built-in documents         — byte compare, no decode
4. Write replacements; leave non-built-in files    — no Resolve
5. NOW Resolve the corrected schema                — guaranteed to validate
6. Scan content nodes for prose-drift candidates   — needs the Index from 5
7. Stage + one commit, or exit clean if 3 found nothing
```

Steps 1–4 never decode a `Property` document, so an unrecognized merge value on disk cannot
block the command that exists to remove it. This satisfies FR-022 and FR-024 structurally rather
than by a special case.

**Second-order requirement**: every *other* command hard-fails on an un-upgraded graph, with an
error that today would read "schema document invalid: merge". `ErrSchemaInvalid` MUST name
`arc upgrade` as the remedy, or the failure mode is a dead end.

**Alternative rejected**: `arc apply schema arcnet:core`. `required`/`optional` merge by `union`
and `merge`/`role` are `immutable` — the merge path can express none of this feature's changes,
all of which are retractions or overwrites. This is the same wall spec 022 hit.

**Alternative rejected**: tolerate `validatedOverwrite` on read for one release, per the
program's "readers are lenient" policy. It contradicts FR-001/FR-004 (the menu is exactly six)
and leaves the non-conformant value live in graphs indefinitely, since nothing would force the
upgrade.

---

## D11. Command placement for the migration

**Decision**: a new top-level `arc upgrade`, in the existing `ctrl` (control-plane) domain
alongside `init`, wired at `cmd/arc/ctrl/upgrade.go`.

**Rationale**: `init` and `upgrade` are the same operation over a graph's built-in vocabulary —
one for a new graph, one for an existing one — and both are control-plane, not graph-content.
`ctrl/service.Init` already owns `schema.Seed()` output; `upgrade` reuses it directly.

**Alternatives rejected**: `arc init --upgrade` (overloads a command whose entire contract is
"the target directory must be empty" — two mutually exclusive guards behind one verb);
`arc apply schema --replace` (a destructive replace flag on a command documented as additive
merge, and it sits under `apply`, whose subject is patches, not releases).

**Flags**: `--dry-run` (report, write nothing, exit 0), plus the inherited `--json`/`--verbose`/
`--quiet` from `bios`. No `--force`: replacing built-in documents *is* the contract, and a
confirmation prompt would break non-interactive use (Principle IX).

---

## D12. Prose-drift detection is advisory and best-effort

FR-023 requires reporting nodes whose `abstract`/`description` show evidence of prior
duplication, without repairing them.

**Decision**: report a node when a `firstWriteWin`-declared text predicate holds **more than one
paragraph**. Rationale: under the corrected vocabulary these predicates hold a single first-fixed
value, so a multi-paragraph value is precisely the shape only the old `append` path could have
produced. Reuses `splitParagraphs` — no new heuristic, no threshold to tune.

**Explicitly not repaired**: the boundary between original text and accumulated text is
unrecoverable, and `mergeText`'s near-duplicate guard means the paragraphs are not even
necessarily similar. Output is a list of node paths under a "review" heading; exit code
unaffected.

**Alternative rejected**: reuse `jaccardSimilarity` to find near-duplicate paragraph pairs. It
finds only the doubling case, missing the reworded-append case (D4) that is the more common one.

---

## D13. Golden-file snapshot of the seeded tree

`CORE-FIX.md` §7 recommends this and notes that findings B3 and B4 "survived unnoticed across
four spec revisions" precisely because the seeded vocabulary is reviewed as diffs of Go map
literals.

**Decision**: build it here (spec SC-007). A `testdata/golden/schema/` tree holding every file
`Seed()` produces, asserted byte-for-byte by a test in `internal/app/schema/service`, refreshed
by `go test ./internal/app/schema/service -update`.

**Rationale**: this feature changes 15 predicate entries and all 8 type entries at once. Without
a snapshot, review is 23 map-literal diffs; with one, it is a readable diff of rendered Markdown
that shows exactly what a user's graph will contain.
