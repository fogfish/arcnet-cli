# Feature Specification: ARCNET-CORE v0.11 — Schema Vocabulary Conformance

**Feature Branch**: `023-core-vocabulary-conformance`

**Created**: 2026-08-23

**Status**: Draft (amended 2026-08-23 during `/speckit-plan` — see [plan.md](plan.md) "Validation Findings")

**Bugfix**: 2026-08-23 — [BUG-001](bugs/BUG-001.md) CORE 0.11 → 0.12 retires or renames six of the
seven predicates this spec's Assumptions (§"Predicates the specification uses but never
registers") and Out of Scope both declared settled in the tool's favour. New User Story 6 below,
new FR-030–FR-036, and strikethroughs on the superseded Assumption/Out-of-Scope lines.

**Bugfix (round 2)**: 2026-08-23 — [BUG-001](bugs/BUG-001.md) verification found FR-037
contradicted this feature's own Compatibility Policy and plan.md's existing rejection of "lenient
reader for one release" for the merge-vocabulary closure; FR-037 rewritten as a plain breaking
change. Open Item 3 (`definition`→`text` merge collision) resolved: `text` adopts `MergeAppend`
uniformly (no per-type merge override introduced); FR-030 and the Edge Case updated accordingly.
FR-036 marked exempt from an acceptance scenario, matching FR-025's treatment.

**Input**: User description: "The schema vocabulary `arc init` seeds is not conformant with ARCNET-CORE v0.10 in five independent ways, all of them edits to the same built-in predicate and type tables. First, the tool defines a seventh merge operation, `validatedOverwrite`, used by the `scoreZ` and `scoreC` predicates; §9.3 fixes the menu at six values and §4.8 requires every merge to be commutative and idempotent, which an overwrite is not — so a freshly initialized graph is non-conformant out of the box. Second, every seeded `Class` node carries a type-level `merge` attribute, but §9.3 explicitly retired type-level merge in favour of per-predicate merge, and §10.8 registers `merge` as a predicate used by `Property` only; the tool's own code comments already concede the field is no longer consulted. Fourth, `Timeline` requires `granularity` and `period` where §11.5 requires `cites` alone, producing the same class of false failure. Fifth, three core predicates from §10.2 are missing entirely — `author`, `about`, `genre` — so any patch using them triggers auto-registration with a guessed role and merge that are wrong for all three, and three registered predicates carry the wrong merge operation: `abstract` and `description` are `append` where §10.2 and §10.8 declare `firstWriteWin`, and `cites` is `append` where §10.6 declares `union`. The `abstract` and `description` mismatches are the material ones: re-applying a patch appends the abstract to itself, breaking the idempotency guarantee §14.3 requires of every apply. All four must be corrected, existing graphs must be able to pick up the corrected vocabulary, and the false-positive lint rules must stop firing."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A document's summary holds a single, first-fixed value (Priority: P1)

An author applies a patch describing a document, then applies it again after the upstream document was lightly edited — a reworded opening, a corrected date. The document's summary holds one value: the one established by the first apply. Where the two disagree, the divergence is flagged for review rather than absorbed.

Today the summary prose is combined by appending. A byte-identical re-apply is already harmless — the tool drops paragraphs that closely resemble one already present — but a *reworded* summary falls below that resemblance threshold and is appended as a second paragraph. Re-ingest a lightly edited document a few times and the summary becomes a stack of near-synonymous paragraphs, none of them wrong, the whole no longer a summary. The specification declares these predicates single-valued and first-fixed precisely to prevent this; the tool declares them accumulating.

**Why this priority**: This is the only finding in the feature that degrades authored content rather than merely mislabelling it, and it does so silently on an operation authors are explicitly told is safe to repeat. Every other correction here is vocabulary hygiene.

**Independent Test**: Apply a patch carrying a document summary to a fresh graph, apply a second patch whose summary is a substantial rewording of the first, and confirm the stored summary is still the first one and the divergence was reported — without changing anything about type definitions, folders, or lint.

**Acceptance Scenarios**:

1. **Given** a document with an established summary, **When** the author applies a patch whose summary is a substantial rewording of it, **Then** the stored summary is unchanged, remains a single paragraph, and the divergence is reported for review.
2. **Given** a fresh graph and a patch carrying a document with summary prose, **When** the author applies the patch twice, **Then** the resulting summary is byte-identical to its state after the first apply.
3. **Given** the same graph, **When** the author applies the patch a third time, **Then** the graph reports no change to commit.
4. ~~**Given** an entity with an established definition and a reference with an established relevance note, **When** the author applies a patch carrying reworded prose for either, **Then** each stored value is unchanged and each divergence is reported.~~ **Superseded 2026-08-23 ([BUG-001](bugs/BUG-001.md) round 2, FR-030): now holds for `Reference`'s `relevance` only.** `Entity`'s leading-prose predicate is renamed `definition`→`text` (FR-030) and `text` is `MergeAppend`, not `MergeFirstWriteWin` — reworded `Entity` prose accumulates again, the same behaviour this scenario originally existed to rule out for `Entity`. Accepted as the cost of resolving the `text` merge collision without a new per-type merge mechanism; see the Edge Case above.
5. **Given** a patch carrying a predicate or type definition with descriptive prose, **When** the author applies it twice, **Then** the definition's description is byte-identical after both applies.
6. **Given** a patch carrying citation links for a document that already cites some of the same targets, **When** the author applies it, **Then** each cited target appears exactly once.

---

### User Story 2 - A newly initialized graph declares only conformant merge behaviour (Priority: P2)

An author initializes a graph and inspects the seeded vocabulary. Every predicate definition declares one of the six merge behaviours the specification fixes; none declares a seventh, tool-invented one. No type definition carries a merge declaration at all, because merge is a property of a predicate and not of a type.

Today a fresh graph is non-conformant from its very first commit: two analytics predicates declare a seventh merge behaviour that no other conformant tool can interpret, and every one of the eight seeded type definitions carries a type-level merge attribute that the specification retired and that the tool itself no longer reads.

**Why this priority**: A graph that is non-conformant at creation cannot be exchanged with any other conformant tool, and the invented merge value is the single declaration most likely to make another reader reject the graph outright. It ranks below Story 1 only because it mislabels rather than destroys.

**Independent Test**: Initialize a fresh graph, read every seeded predicate and type definition, and confirm the merge vocabulary is the closed set of six and that no type definition declares a merge — without applying any patch.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the author initializes a graph, **Then** every seeded predicate definition declares a merge drawn from exactly `immutable`, `union`, `firstWriteWin`, `fillIfEmpty`, `lastWriteWin`, `append`.
2. **Given** the same freshly initialized graph, **When** the author reads every seeded type definition, **Then** none carries a merge declaration.
3. **Given** the same freshly initialized graph, **When** the author reads the two analytics score predicates, **Then** each declares a conformant merge behaviour and neither declares the retired seventh one.
4. **Given** a hand-written predicate definition declaring a merge value outside the closed set of six, **When** the author runs any command that reads the graph's vocabulary, **Then** the command fails with an error naming the offending value, the file it appears in, and the six legal values.
5. **Given** a graph whose type definitions predate this feature and still carry a merge declaration, **When** the author runs any command that reads the graph's vocabulary, **Then** the command succeeds and the stale declaration is ignored without a diagnostic.
6. **Given** a freshly initialized graph, **When** the author lints it, **Then** the graph reports clean.

---

### User Story 3 - Conformant nodes stop being reported as violations (Priority: P3)

An author lints a graph whose nodes are shaped exactly as ARCNET-CORE describes them. The lint reports nothing. Specifically: a subject node carrying no publication date and no creation timestamp is accepted, and a timeline period node carrying only its chronological citations is accepted.

Today the tool imposes a universal base type that requires a publication date and a creation timestamp on *every* node, and it requires a granularity and a period code on every timeline node. The specification requires none of these. An author whose graph is correct by the specification therefore gets a wall of missing-required-predicate reports that they cannot act on, which trains them to ignore lint output entirely.

**Why this priority**: False positives are corrosive but not destructive — they degrade trust in the tool's own quality gate rather than the graph's content. The correction is also the one most visible to an author on day one, which is why it stays above pure vocabulary registration.

**Independent Test**: Lint a hand-built graph whose nodes carry exactly the predicates the specification requires and no more, and confirm zero missing-required-predicate reports — without applying any patch or changing merge behaviour.

**Acceptance Scenarios**:

1. **Given** a subject node carrying no publication date and no creation timestamp, **When** the author lints the graph, **Then** no missing-required-predicate diagnostic is reported for that node.
2. **Given** a timeline period node carrying only chronological citations, **When** the author lints the graph, **Then** no missing-required-predicate diagnostic is reported for that node.
3. **Given** a timeline period node that additionally carries a granularity, a period code, and a heading, **When** the author lints the graph, **Then** none of the three is reported as an undeclared predicate.
4. **Given** a document node that carries no publication date, **When** the author lints the graph, **Then** a missing-required-predicate diagnostic *is* reported — a document's publication date remains required by its own type.
5. **Given** a graph produced by applying a patch to a freshly initialized graph, **When** the author lints it, **Then** the graph reports clean.

---

### User Story 4 - The specification's own metadata predicates are registered (Priority: P4)

An author applies a patch that records who wrote a document, what it is about, and what genre it belongs to. The three predicates are already part of the graph's seeded vocabulary, so the graph accepts them the way it accepts any other core predicate — and lints clean afterwards.

Today none of the three is registered, and because all three are front-matter attributes rather than body links, nothing registers them on first use either: the tool's auto-registration only ever observes predicates that appear in a node's body. So every occurrence produces two lint violations — one for using an unregistered predicate, one for carrying a predicate the node's own type never declared — on metadata the specification itself defines. The author's only remedies are to hand-write three definitions or to stop recording authorship.

**Why this priority**: The consequence is lint noise on conformant content, the same corrosive class as Story 3, confined to three predicates. It ranks below Story 3 because it affects only graphs that record this metadata at all.

**Independent Test**: Apply a patch using all three predicates to a freshly initialized graph and confirm the graph lints clean, without touching merge behaviour or type requirements.

**Acceptance Scenarios**:

1. **Given** a freshly initialized graph, **When** the author reads the seeded vocabulary, **Then** definitions exist for the authorship, aboutness, and genre predicates, each declaring the role and merge behaviour the specification assigns it.
2. **Given** a freshly initialized graph and a patch using all three predicates, **When** the author applies it and lints the graph, **Then** no unregistered-predicate and no undeclared-predicate diagnostic is reported for any of the three.
3. **Given** the same apply, **When** the author inspects the result, **Then** zero predicates were created — the three resolve against the seeded vocabulary rather than being registered on the spot.
4. **Given** a patch that records two authors for one document across two separate applies, **When** the author applies both, **Then** the document carries both names, each exactly once.
5. **Given** the seeded vocabulary, **When** the author reads the citation predicate's definition, **Then** it declares the merge behaviour that combines contributions without duplication rather than the one that appends.

---

### User Story 5 - REMOVED

> **Removed 2026-08-23, post-implementation.** This story specified an `arc upgrade` adoption step that migrated a graph seeded by a previous release to the corrected vocabulary. It was implemented, then removed by explicit decision: `arc` is pre-1.0 and experimental, and carrying a dedicated compatibility/migration path for graphs seeded by earlier releases is tech debt the project does not want yet. The merge-vocabulary closure (FR-001/FR-004) is accepted as a plain breaking change — an existing graph seeded by a previous release simply no longer loads; there is no in-place remedy, and none is planned before 1.0. See [Out of Scope](#out-of-scope).
>
> Every requirement this story owned (FR-017–FR-024) and its acceptance scenarios are struck. US1–US4 are unaffected: none of them ever depended on this story.

---

### User Story 6 - The vocabulary catches up to ARCNET-CORE v0.12 (Priority: P5)

**Added 2026-08-23 — [BUG-001](bugs/BUG-001.md).** ARCNET-CORE 0.12 resolved, from the spec side,
the exact ambiguity this feature's Assumptions section resolved from the tool side ("predicates
the specification uses but never registers... the tool registering them is more conformant than
the specification"). Six of those seven predicates are now renamed or retired outright; the
seventh (`granularity`) is retired too. An author who initializes a graph today gets a vocabulary
that is portable to a 0.11-conformant reader and not to a 0.12-conformant one.

**Why this priority**: Same corrosive-not-destructive class as Stories 3 and 4 — vocabulary
hygiene, not data loss — but it ranks last because it corrects decisions this very feature already
shipped, rather than a gap the feature never addressed.

**Independent Test**: Initialize a fresh graph and confirm the seeded vocabulary carries `text`
(not `definition`) as `Entity`'s required leading-prose predicate, `author` (not `authors`) as the
sole authorship predicate on `Source`/`Reference`, `published` (not `year`) on `Reference`, no
`notes` on `Entity`/`Resource`, no `ref`/`status` on `Reference`, and no `granularity` anywhere —
without applying any patch.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the author initializes a graph, **Then**
   `_schema/Class/Entity.md` requires `text`, and no `_schema/Property/definition.md` exists.
2. **Given** the same graph, **When** the author inspects `Source` and `Reference`'s Optional
   predicates, **Then** each declares `author` and neither declares `authors`.
3. **Given** the same graph, **When** the author inspects `Reference`'s Optional predicates,
   **Then** `published` is present and `year` is absent.
4. **Given** the same graph, **When** the author inspects `Entity` and `Resource`'s Optional
   predicates, **Then** neither declares `notes`.
5. **Given** the same graph, **When** the author inspects `Reference`'s Optional predicates,
   **Then** neither `ref` nor `status` is present.
6. **Given** the same graph, **When** the author inspects `Timeline`'s Optional predicates,
   **Then** `granularity` is absent, and the tool derives yearly-vs-monthly bucketing from the
   `@id` period-code shape or the containing folder instead.
7. **Given** a patch that ingests an `Entity` node with leading prose, **When** the author applies
   the patch and re-reads the node, **Then** the prose is stored and recovered under `text` — the
   parser/renderer key changes in lockstep with the schema table, not just the Required/Optional
   list — and a second apply carrying reworded prose accumulates it (`text` is `MergeAppend`; see
   the Edge Case above and FR-030), rather than being flagged for review.
8. **Given** a graph seeded by a previous release whose `Entity`/`Reference` nodes carry
   `definition`/`authors`/`year`/`notes`/`ref`/`status`/`granularity` front-matter keys, **When**
   the author runs any command that reads the graph, **Then** the command's behaviour is
   unspecified by this feature beyond FR-037's plain-breaking-change treatment — no reader
   leniency is introduced, matching FR-001/FR-004's precedent for the merge-vocabulary closure.

---

### Edge Cases

- **Near-identical prose today.** The tool already drops an incoming paragraph closely resembling one present, so a byte-identical re-apply is harmless before this feature. Any test asserting the corrected behaviour must therefore use *reworded* prose, or it passes without a fix (see [plan.md](plan.md) F2).
- **Stale type-level merge declarations.** Reading tolerates a type definition that still carries a merge declaration; writing never emits one. This falls out of the data model itself (`core.TypeDef` carries no `Merge` field to populate) rather than from any dedicated compatibility code.
- **Relaxing a requirement on an existing graph.** Moving the publication-date requirement off the universal base type and onto the document type removes it from subject, resource, reference, and timeline nodes. Nodes that already carry the attribute keep it; nothing is rewritten and only the lint expectation changes.
- **The retired seventh merge value in an author's own definition.** A hand-written definition declaring it is rejected with the same error as any other illegal value; it is not silently rewritten.
- **`definition`→`text` collides with the already-registered generic `text` predicate — resolved.** ([BUG-001](bugs/BUG-001.md), round 2) `internal/core/markdown.go`'s `textPredicateFor` already returns the literal string `"text"` for `Resource`'s leading prose, and that generic `text` predicate is seeded `MergeAppend`, not `MergeFirstWriteWin`. Renaming `Entity`'s leading-prose key to `text` therefore changes `Entity`'s merge behaviour from first-fixed to accumulating. **Resolved**: `text` stays `MergeAppend` uniformly across every type that uses it — no per-type or role-qualified merge override is introduced. This is a deliberate, accepted regression of `Entity`'s prose-drift protection (the very defect User Story 1 fixed for `Source`), traded for not inventing a new merge-scoping mechanism the merge algebra does not otherwise have; an author who wants first-fixed `Entity` prose must flag divergent re-applies for review same as any other `append` predicate.
- **`notes` is a shared round-trip primitive, not a per-type list entry.** ([BUG-001](bugs/BUG-001.md)) `textPredicateFor`'s non-leading-prose branch returns the literal `"notes"` unconditionally for every node type, not just `Entity`/`Resource`. Retiring `notes` from those two types' Optional lists alone does not stop the parser/renderer from keying their trailing body sections as `notes`; the primitive itself needs a type-aware branch.

## Requirements *(mandatory)*

### Functional Requirements

#### The merge vocabulary

- **FR-001**: The set of merge behaviours the system recognizes MUST be exactly `immutable`, `union`, `firstWriteWin`, `fillIfEmpty`, `lastWriteWin`, `append`.
- **FR-002**: A vocabulary document declaring a merge value outside that set MUST be rejected, with an error naming the offending document.
- **FR-003**: The two analytics score predicates MUST declare a merge behaviour drawn from the set in FR-001.
- **FR-004**: The system MUST NOT retain any recognized merge behaviour beyond the six in FR-001, including for backwards compatibility.

#### Type-level merge

- **FR-005**: A seeded type definition MUST NOT carry a merge declaration.
- **FR-006**: Reading a graph MUST continue to succeed when an existing type definition carries a merge declaration; the declaration MUST be ignored and MUST NOT produce a diagnostic.

#### Required-predicate corrections

- **FR-007**: The universal base type MUST require no predicates.
- **FR-008**: The document type MUST require exactly `title`, `published`, `abstract`, `mentions`.
- **FR-009**: The timeline period type MUST require exactly `cites`, and ~~MUST permit `granularity`, `period`, and `heading` as optional~~ **superseded by FR-035 ([BUG-001](bugs/BUG-001.md)): `granularity` is retired, not merely optional; `period` and `heading` remain optional.**
- **FR-010**: Lint MUST NOT report a missing-required-predicate diagnostic for a node that carries every predicate its own type requires under FR-007 through FR-009, FR-028, or FR-029.
- **FR-028**: The external-work type MUST require only its title, retaining its classification, relevance, status, and notes predicates as optional.
- **FR-029**: The document and subject types MUST permit topical tags; the type-definition type MUST require its own description and permit the inheritance predicate the tool already writes onto seeded type definitions.

#### Predicate registrations and merge corrections

- **FR-011**: The seeded vocabulary MUST register the authorship, aboutness, and genre predicates named by the specification, each with role `meta` and merge `union`.
- **FR-012**: Applying a patch that uses any of those three predicates MUST resolve them against the seeded vocabulary — registering nothing on the spot — and the resulting graph MUST report no unregistered-predicate and no undeclared-predicate diagnostic for them.
- **FR-013**: The summary, description, ~~definition,~~ and relevance predicates MUST declare merge `firstWriteWin`. **Amended 2026-08-23 ([BUG-001](bugs/BUG-001.md) round 2, FR-030): `definition` is retired; `Entity`'s leading-prose predicate is now `text`, which declares `MergeAppend`, not `firstWriteWin` — see FR-030.**
- **FR-014**: The citation predicate MUST declare merge `union`.
- **FR-015**: Applying the same patch twice MUST leave every first-fixed prose value byte-identical to its state after the first apply, and MUST leave every citation target present exactly once.
- **FR-015a**: Applying a patch whose first-fixed prose diverges from an established value MUST preserve the established value and report the divergence for review, rather than accumulating both.
- **FR-016**: Applying the same patch a second time MUST produce no commit.
- **FR-026**: Every predicate the specification aligns to a standard-vocabulary term MUST declare that term — specifically the title, summary, location, identifier, alternative-name, and authorship predicates, none of which declares one today.
- **FR-027**: The publication-year predicate MUST declare merge `immutable`, not `fillIfEmpty`.

#### ARCNET-CORE v0.12 predicate retirement ([BUG-001](bugs/BUG-001.md))

- **FR-030**: `Entity`'s required leading-prose predicate MUST be renamed from `definition` to
  `text`. **Resolved 2026-08-23 (round 2)**: `text` declares `MergeAppend` uniformly (the same
  merge behaviour `Resource` already uses for its own `text` predicate) — no per-type or
  role-qualified merge override is introduced. This deliberately trades away `Entity`'s
  first-fixed prose protection (US1 Acceptance Scenario 4, now struck) to avoid a new
  merge-scoping mechanism the algebra does not otherwise have.
- **FR-031**: The authorship predicate registered on `Source` and `Reference` MUST be the singular
  `author`; the plural `authors` MUST NOT be registered or declared by either type's Optional list.
  This supersedes the Assumption "v0.11 registers both" below.
- **FR-032**: `Reference`'s Optional list MUST declare `published`, not `year`, for its publication
  date; `year` MUST NOT be registered.
- **FR-033**: `Entity` and `Resource` MUST NOT declare `notes` in their Optional lists, and the
  parser/renderer's non-leading-prose key (`textPredicateFor`, `internal/core/markdown.go`) MUST
  NOT key either type's trailing body section as `notes`.
- **FR-034**: `Reference` MUST NOT declare `ref` or `status` in its Optional list.
- **FR-035**: `Timeline` MUST NOT declare `granularity` at all — supersedes FR-009's "MUST permit
  `granularity`... as optional". Yearly-vs-monthly bucketing MUST be derived from the `@id`
  period-code shape or the containing folder, not from a stored predicate.
- **FR-036**: Every internal comment, error message, or doc link citing `§10.7` or `§10.8` from the
  pre-0.12 numbering MUST be updated to cite the single, renumbered `§10.7` ("Schema predicates").
  **Exempt from an acceptance scenario** ([BUG-001](bugs/BUG-001.md), round 2) — a doc/comment-only
  correction with no user-observable behavior, same treatment as FR-025.
- **FR-037**: ~~A graph seeded by a previous release whose nodes carry the retired
  `definition`/`authors`/`year`/`notes`/`ref`/`status`/`granularity` front-matter keys MUST
  continue to be read successfully for one release (reader leniency), while `arc init`/`arc apply`
  MUST emit only the new forms (writer strictness) — per the compatibility policy this feature's
  plan already established.~~ **Rewritten 2026-08-23 ([BUG-001](bugs/BUG-001.md), round 2)**: this
  contradicted `ARCHITECTURE.md`'s Compatibility Policy and plan.md's own Complexity Tracking
  entry, which already rejected "lenient reader for one release" as an alternative for this same
  feature's merge-vocabulary closure. Corrected: The retirement of
  `definition`/`authors`/`year`/`notes`/`ref`/`status`/`granularity` MUST be treated as a plain
  breaking change, identically to FR-001/FR-004 — a graph seeded by a previous release whose nodes
  carry these keys is not specially supported; re-initializing or re-applying content against the
  corrected vocabulary is the expected recovery, with no reader-leniency mechanism introduced.

#### Documentation

- **FR-025**: User-facing documentation describing the seeded vocabulary MUST be updated to match the corrected vocabulary, including the merge menu of six and the corrected type requirements.

### Key Entities

- **Predicate definition**: A registered predicate's own record in the graph, declaring its serialization role, its merge behaviour, an optional human-readable label, an optional aligned standard-vocabulary term, and a descriptive body. The merge behaviour is the field this feature corrects.
- **Type definition**: A registered node type's own record in the graph, declaring which predicates a conforming instance must and may carry, an optional base type, and a descriptive body. The merge declaration is the field this feature removes.
- **Merge behaviour**: The named rule by which two contributions to the same predicate on the same node are combined. A closed set of six.
- **Built-in vocabulary**: The predicate and type definitions the tool seeds into every new graph.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Applying any patch N times produces the same graph content as applying it once, for every N ≥ 1 — verified over a patch exercising summaries, definitions, relevance notes, descriptions, citations, and every combine-by-union predicate.
- **SC-008**: Every seeded predicate that the specification aligns to a standard vocabulary declares that alignment; zero declare none where one exists.
- **SC-002**: 100% of the merge declarations in a freshly initialized graph draw from the six-value set; 0% of its type definitions carry a merge declaration.
- **SC-003**: A freshly initialized graph lints clean, and remains clean after a representative patch is applied to it.
- **SC-004**: A hand-built graph whose nodes carry exactly the predicates ARCNET-CORE requires produces zero missing-required-predicate diagnostics.
- **SC-005**: A patch using the authorship, aboutness, and genre predicates reports zero predicates created when applied to a freshly initialized graph.
- **SC-007**: Every seeded predicate and type definition matches its ARCNET-CORE v0.10 declaration, verified against a checked-in snapshot of the complete seeded vocabulary so that a future change to it cannot pass review unnoticed.
- **SC-009** ([BUG-001](bugs/BUG-001.md)): A freshly initialized graph's seeded vocabulary contains zero of `definition`, `authors`, `year`, `notes` (on `Entity`/`Resource`), `ref`, `status`, `granularity`; a round-trip parse→render of an `Entity`'s leading prose and of any type's trailing body section never keys either as a retired name.

## Assumptions

- **Scope of "five ways".** The feature description enumerates its findings as *First, Second, Fourth, Fifth* and closes with "All four must be corrected". The third finding — that the universal base type requires a publication date and a creation timestamp on every node, where ARCNET-CORE requires neither — is omitted from the prose but is unambiguously the missing item, and is carried here as FR-007/FR-008. Five corrections are in scope, not four.
- **Analytics score merge behaviour.** The two analytics score predicates adopt `lastWriteWin`. No conformant merge behaviour expresses "only a designated recomputation pass may overwrite this", which is the guarantee the retired seventh value encoded; that guarantee was policy the merge algebra was never the right place to express, and belongs in whatever command performs the recomputation. `fillIfEmpty` was rejected because scores are recomputed by design.
- **Upstream revision.** This feature targets ARCNET-CORE **v0.11** (Draft, 2026-08-23), validated against the published document during planning. The description and the branch name say v0.10; v0.11 supersedes it and resolves three of the upstream ambiguities the description worked around. Section citations carried over from v0.10 are remapped in [research.md](research.md) D0.
- **Authorship predicate name.** ~~Not an assumption any more: v0.11 registers **both** — the singular in §10.2 as the core vocabulary term, the plural in §10.7 as the array form the type definitions reference. Both are seeded, both with merge `union` and the same standard alignment.~~ **Superseded 2026-08-23 ([BUG-001](bugs/BUG-001.md), FR-031): v0.12 registers only the singular `author`; `authors` is retired, not merely redundant.**
- **External-work type requirements.** v0.11's normative type definition for the external-work type requires its title alone, where the previous revision's change note required title, classification, and relevance. This feature follows v0.11 and keeps the other two optional, which also makes the specification's own worked example conform. **This supersedes a recorded clarification in the preceding feature's specification**; see [plan.md](plan.md) F3.
- **Definition and relevance merge behaviour.** v0.11 registers both predicates with a role but states no merge behaviour, while making merge mandatory on every predicate definition. ~~Both adopt `firstWriteWin`, following the specification's own stated rationale for type-specific prose predicates ("a single, first-fixed value is wanted instead"), which names these two alongside the summary predicate.~~ **`relevance` adopts `firstWriteWin`, following that same rationale ([BUG-001](bugs/BUG-001.md) round 3): `definition` is retired under CORE 0.12 (FR-030) — `Entity`'s leading-prose predicate is now `text`, which is `MergeAppend`, not `firstWriteWin`.** Filed upstream.
- **Citation predicate role.** ARCNET-CORE §10.6 declares the citation predicate's role inconsistently within a single entry. The tool's existing role is kept unchanged, because it is what every worked example in the specification actually does. Only the merge behaviour changes.
- **Prose duplication is reported, not repaired.** Where an existing graph's summary has already accumulated paragraphs under the previous appending behaviour, the original boundary is unrecoverable and any repair would be a guess. Detection is best-effort — a first-fixed prose value holding more than one paragraph — and its output is advisory.
- **Predicates the specification uses but never registers.** Under v0.10 seven predicates were used by the core types yet unregistered. v0.11's new §10.7 registers six of them, so those six are now validated against upstream rather than exempt — which is how the publication-year defect (FR-027) surfaced. ~~Only the timeline granularity predicate remains unregistered upstream; the tool keeps registering it, which is more conformant than the specification.~~ **Superseded 2026-08-23 ([BUG-001](bugs/BUG-001.md)): v0.12 resolves this ambiguity from the spec side by retiring or renaming all seven — `definition`→`text`, `authors`→`author`, `year`→`published`, `notes`/`ref`/`status`/`granularity` removed outright (FR-030–FR-035). "The tool is more conformant than the spec" no longer holds; the spec caught up.**
- **The universal base type itself stays.** Inventing a base type is not a violation — the specification permits any registered type to exist, and the inheritance predicate is itself registered. Only the base type's over-broad requirements are non-conformant. Removing the inheritance mechanism is a separate, larger decision and is out of scope.
- **No compatibility path for graphs seeded by a previous release.** `arc` is pre-1.0 and experimental; carrying a migration/adoption command for this correction was judged unnecessary tech debt and removed after being implemented (see User Story 5). An existing graph whose seeded vocabulary declares the retired seventh merge value simply stops loading — there is no in-place remedy.
- **Sequencing.** This feature builds on the merged `022-reference-type-folders` work, which established the five-type vocabulary and type-named folders.

## Out of Scope

- Removing the universal base type or the inheritance mechanism.
- ~~Registering, renaming, or re-typing the seven predicates the specification uses without registering.~~ **Superseded 2026-08-23 ([BUG-001](bugs/BUG-001.md)): now in scope as User Story 6 / FR-030–FR-036 — CORE 0.12 itself renames or retires all seven, so this is no longer a tool-side choice to defer.**
- Any change to how a node's type is classified, to folder layout, or to the exchange format's manifest.
- The lint gaps tracked separately as combinatorial category validation and identity-charset checking.
- Repairing content already damaged by the previous appending behaviour.
- **Migrating, or otherwise preserving compatibility for, a graph seeded by a previous release.** The merge-vocabulary closure (FR-001/FR-004) is a breaking change to graph data; existing graphs are not supported. Removed post-implementation per explicit decision — `arc` is pre-1.0/experimental and a dedicated migration path was judged unnecessary tech debt at this stage.

## Dependencies

- Depends on the merged `022-reference-type-folders` feature for the five-type vocabulary and folder naming this feature's type definitions are expressed against.
- Must not run concurrently with any other feature that edits the built-in type table.
