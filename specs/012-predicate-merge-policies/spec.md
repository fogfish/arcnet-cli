# Feature Specification: Per-Predicate Merge Reconciliation for arc apply

**Feature Branch**: `012-predicate-merge-policies`

**Created**: 2026-07-08

**Status**: Draft

**Input**: User description: "Change how arc apply reconciles an incoming patch's contribution to a node that already exists in the graph, from arc's current \"one merge behavior per node type\" model to ARCNET-CORE v0.7's \"each predicate declares its own merge behavior\" model (§9.3)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A node's fields each reconcile by their own rule, not one rule for the whole node (Priority: P1)

A graph maintainer applies a patch that contributes to a `resource` node that already exists in the graph. That resource's `ref` (its type — a citable work vs. a tracked topic) was set when the node was first created and must never change again. Its `status` (read vs. backlog) legitimately changes over time as the maintainer works through their reading list. Its `tags` grow as new patches mention new topics. Today, arc treats the whole node under one rule, so a field like `status` that should freely update gets the same treatment as `ref`, which should never change. After this feature, each of these fields reconciles according to its own declared rule in the same single merge: `ref` stays exactly as first written, `status` takes the newest contributed value, and `tags` accumulates every distinct value contributed so far — all within one patch application, with no whole-node rule overriding any individual field.

**Why this priority**: This is the entire point of the change — without correct per-predicate reconciliation, nothing else in this feature has value. Every other capability (conflict flagging, replay safety) only matters once individual predicates are actually reconciled by their own rule.

**Independent Test**: Can be fully tested by applying a sequence of patches that each contribute different predicates to the same existing node and confirming, after each application, that every predicate's resulting value matches what its own declared rule (not the node's type) predicts.

**Acceptance Scenarios**:

1. **Given** an existing `resource` node whose `ref` predicate is already set, **When** a patch contributes a different value for `ref` to that same node, **Then** the node's `ref` value is unchanged and no other predicate on the node is affected by that rejection.
2. **Given** an existing `resource` node whose `status` predicate is `backlog`, **When** a later patch contributes `status: read` for that node, **Then** the node's `status` becomes `read`.
3. **Given** an existing `entity` node with two already-recorded `tags`, **When** a patch contributes a third, previously unseen tag for that node, **Then** the node ends up with all three tags, none dropped or duplicated.
4. **Given** a single patch that simultaneously contributes to an immutable predicate, a legitimately-changing predicate, and a growing predicate on the same existing node, **When** the patch is applied, **Then** each of the three predicates reflects its own rule's outcome in the result of that one application.

---

### User Story 2 - Human-review conflict flagging fires only where the rule calls for it (Priority: P2)

A graph maintainer applies a patch that contributes a genuinely different value to a predicate whose rule is "first writer wins" (e.g. a resource's `abstract`, already summarized once). Because a real conflict exists between what's already recorded and what the new patch claims, arc marks the divergence for human review, exactly as it does today. At the same time, another patch contributes a different value to a predicate whose rule is "latest write always wins" (e.g. `status`) or "grows without discarding" (e.g. accumulating tags or appending prose) — for these, no genuine conflict exists by definition, so nothing is ever flagged, even though the values differ.

**Why this priority**: Preserving the existing, trusted conflict-review mechanism — and confining it to exactly the cases where it belongs — is what keeps this change safe to ship; a maintainer who already relies on conflict markers must see them disappear from fields where they never made sense, without losing them where they still do.

**Independent Test**: Can be fully tested by applying diverging contributions to one predicate of each rule and confirming a conflict marker appears only for the "first writer wins" predicate (and, once set, a "stays absent until first written" predicate), never for the others.

**Acceptance Scenarios**:

1. **Given** an existing node whose `abstract` predicate already holds a value, **When** a patch contributes a different value for `abstract`, **Then** the existing value is preserved and the divergence is marked for human review using arc's existing conflict-marker format.
2. **Given** an existing node whose `status` predicate already holds a value, **When** a patch contributes a different value for `status`, **Then** the new value takes effect and no conflict is marked.
3. **Given** an existing node whose `tags` predicate already holds values, **When** a patch contributes an overlapping and a new value, **Then** the result is the deduplicated union and no conflict is marked.
4. **Given** a predicate that stays absent until first written (e.g. a resource's `url`), **When** a first patch supplies a value and a later patch supplies a different value, **Then** no conflict is marked for the first contribution, but the later, genuinely diverging contribution is marked for human review.

---

### User Story 3 - Replaying or reordering patches never changes the outcome (Priority: P3)

An engineer investigating the graph's history re-applies an old patch, or applies two patches to the same node in the reverse of their original order (for example, while rebuilding the graph from its git history). Every predicate's declared rule still produces the exact same final result as the original application, regardless of how many times a patch is replayed or in what order two patches touching the same node are applied — with one deliberate exception: a predicate whose rule always takes the latest write follows arc's own applied-order, the same way git already treats the most recent commit to a tracked file as authoritative, so reordering which of two such contributions is applied last is expected to change that predicate's outcome, not a defect.

**Why this priority**: This underwrites the graph's git-replay and rollback guarantee; without it, the per-predicate model would be an improvement in expressiveness that quietly breaks the property every other command and workflow depends on.

**Independent Test**: Can be fully tested by applying a patch twice in a row, and by applying two patches to the same node in both possible orders, and confirming the resulting graph file is identical in every case except where the rule itself is order-sensitive by design.

**Acceptance Scenarios**:

1. **Given** a patch has already been applied to a node, **When** the exact same patch is applied to that node again, **Then** the node's file is byte-for-byte unchanged by the second application.
2. **Given** two patches each contribute to the same existing node's `tags` and `entries` predicates, **When** they are applied in one order and, separately from a fresh copy of the graph, in the reverse order, **Then** both resulting node files are identical.
3. **Given** two patches contribute different values to the same "latest write always wins" predicate, **When** the patches are applied one after the other, **Then** the node's value is whichever patch was applied last, consistent regardless of which patch's content happened to be written first or second in the source material.
4. **Given** a conflict has already been marked for human review on a predicate, **When** the same conflicting patch is applied again, **Then** the marker is not duplicated or re-wrapped.

---

### Edge Cases

- A predicate whose rule is "stays absent until first written" has no existing value; a patch contributes one; a later patch contributes an identical value — no conflict is ever marked, since nothing genuinely diverges.
- A predicate whose rule keeps existing content only until a designated validation step revalidates it: an ordinary patch contribution that diverges from the already-set value is not flagged and does not overwrite it — the value only changes through that separate, out-of-scope validation step.
- Two contributions to a "latest write always wins" predicate are applied back to back — the second application's value always wins, even if it is identical to, or "earlier" in some external sense than, the first; arc apply does not compare timestamps for this rule, only application order, mirroring how git already treats the most recent commit to a tracked file as authoritative.
- A predicate that grows an ordered list receives an entry that is already present — the entry is not duplicated, so re-applying the same patch does not grow the list further.
- A predicate that grows prose receives a paragraph that is materially the same as one already present — it is not appended again.
- A predicate encountered for the first time within the very patch that introduces it (no schema document existed before this application) is reconciled using its freshly auto-registered default rule within that same application, not deferred to a later run.
- A node's type is one a graph has registered itself (not one of the four built-in kinds) — its predicates still reconcile individually by their own declared rule; no node type gets special-cased whole-node treatment.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Arc apply MUST reconcile every predicate present on an existing node, an incoming contribution, or both, according to that specific predicate's own declared merge behavior, rather than by one behavior applied to the whole node.
- **FR-002**: Arc MUST support seven distinct, individually selectable merge behaviors, each with genuinely different observable outcomes: **immutable**, **union**, **firstWriteWin**, **fillIfEmpty**, **lastWriteWin**, **append**, and **validatedOverwrite**.
- **FR-003**: Under **immutable**, a predicate's first genuinely-written value MUST remain unchanged by any later contribution, for as many further contributions as arrive, in any order, with no conflict ever flagged.
- **FR-004**: Under **union**, contributions to a predicate MUST combine into a single deduplicated collection of values, with no conflict ever flagged.
- **FR-005**: Under **firstWriteWin**, a predicate's first genuinely-written value MUST persist; a later contribution whose value genuinely differs from it MUST be flagged for human review and MUST NOT silently replace the existing value.
- **FR-006**: Under **fillIfEmpty**, a predicate with no existing value MUST accept its first contributed value without that acceptance being flagged; from that point forward the predicate MUST behave exactly as firstWriteWin.
- **FR-007**: Under **lastWriteWin**, the most recently *applied* contribution to a predicate MUST always take effect, with no conflict ever flagged. Unlike every other behavior in this menu, lastWriteWin is intentionally sensitive to arc's own application order (mirroring git's own last-commit-wins treatment of a tracked file) rather than to any timestamp declared inside the contribution itself; this is a deliberate, documented exception to FR-010's general order-independence guarantee.
- **FR-008**: Under **append**, a contribution MUST grow a predicate's existing content without discarding what is already there: for a list-shaped predicate, an already-present entry is not duplicated; for a prose predicate, already-present prose is not repeated. No conflict is ever flagged.
- **FR-009**: Under **validatedOverwrite**, a predicate's value, once set, MUST NOT be changed by an ordinary patch contribution, and a divergence MUST NOT be flagged for human review; only a dedicated validation process — outside this feature's scope — may change it.
- **FR-010**: Every merge behavior's reconciliation MUST be idempotent (applying the same patch to a node more than once produces no further change) and, with the single documented exception of lastWriteWin (FR-007), commutative (applying two patches that touch the same node in either order MUST produce the same final node).
- **FR-011**: Arc apply's existing conflict-marker mechanism and format MUST continue to be used, unchanged, and MUST fire only when a predicate's declared behavior is firstWriteWin (including a fillIfEmpty predicate once it holds its first value).
- **FR-012**: Arc apply MUST NOT flag a conflict for a predicate whose declared behavior is union, append, lastWriteWin, immutable, or validatedOverwrite, even when its existing and incoming values genuinely differ.
- **FR-013**: Arc MUST determine each present predicate's merge behavior by looking it up in the graph's schema index (built from `_schema/predicates/` documents per the existing schema-index mechanism); this feature changes only how that index's declared behaviors are consulted during reconciliation, not how the index itself is built or parsed.
- **FR-014**: A predicate encountered during patch application with no prior schema document MUST be reconciled using its automatically assigned default behavior within that same application.
- **FR-015**: The prior whole-node, single-behavior dispatch (one merge behavior chosen by a node's type) MUST no longer determine how any predicate reconciles, for every node type including graph-registered custom types.
- **FR-016**: A graph's built-in predicate vocabulary, as seeded when a graph is first initialized, MUST declare each predicate's genuinely correct behavior from the seven-behavior menu — in particular, distinguishing a predicate that must never change (e.g. a resource's `ref`) from one that legitimately changes over time (e.g. a resource's `status`) — so that a freshly initialized graph exhibits correct per-predicate reconciliation without any hand-editing of schema documents first. Per FR-018, a `text`-role predicate's seeded behavior is `append` unless explicitly overridden.
- **FR-017**: `arc apply --verbose` MUST report, for every predicate present on either side of a merged node — not only the ones flagged for conflict — that predicate's name, the `MergeOp` it resolved to (via the schema index, per FR-013), and its per-predicate outcome (e.g. unchanged/filled/overwritten/appended/flagged), in addition to the existing one-line-per-node summary. This is the direct verbose-output counterpart to FR-001: since reconciliation is now per-predicate rather than per-node, `--verbose`'s report must be too. The reported outcome MUST reflect the actual result of that predicate's reconciliation, not merely which merge behavior it dispatched to (BUG-002 — see FR-019).
- **FR-018**: Every predicate whose declared `role` is `text` MUST default to the **append** merge behavior when its schema document is seeded or auto-registered, so that `role: text` alone is sufficient for a graph maintainer to predict `append` dispatch without reading each predicate's individual assignment; a specific `text`-role predicate MAY still be hand-edited to a different behavior afterward (FR-013 already permits that), but no built-in `text`-role predicate is seeded with any behavior other than `append`.
- **FR-019**: A predicate whose declared behavior is `union` or `append` MUST be reported as `unchanged` when its reconciled value is identical to its existing value, and as `appended` (or `created`, if the predicate had no prior value) only when the reconciled value genuinely differs from the existing one — a plain restatement of FR-017's accuracy requirement as its own testable rule, since this is the one dispatch class where the distinction was previously not implemented: a `union`/`append` predicate whose incoming contribution added no genuinely new value (e.g. a fully duplicate list entry, or a prose paragraph judged a near-duplicate of one already present, per `mergeText`'s own Jaccard-similarity dedup) was previously always reported `appended` regardless of whether anything actually changed. *(Bugfix BUG-002, 2026-07-12)*

### Key Entities

- **Predicate Merge Behavior**: The single rule — one of the seven canonical behaviors — declared on a predicate's own schema document, and now the sole authority arc apply consults to reconcile that predicate wherever it appears, on any node of any type.
- **Node Reconciliation**: The act of merging an incoming contribution into an existing node, now performed by reconciling every predicate present on either side individually, rather than by choosing one behavior for the entire node.
- **Conflict Marker**: The existing human-review marker for a genuinely diverging value, unchanged in form, now scoped strictly to predicates whose declared behavior calls for it.
- **Predicate Merge Report** *(BUG-001)*: The `--verbose` record of one predicate's reconciliation within one node merge — its name, resolved `MergeOp`, and outcome — surfaced alongside the existing per-node summary line (FR-017). Its reported outcome for a `union`/`append` predicate MUST reflect whether the reconciled value actually changed, not just that a union/append dispatch ran (FR-019, BUG-002).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across a fixture graph exercising all seven merge behaviors, 100% of predicates on a merged node reconcile according to their own declared behavior, with zero cases of a predicate instead following its node's previous whole-node default.
- **SC-002**: Across the same fixture, conflict markers appear only for firstWriteWin (and post-first-write fillIfEmpty) predicates with genuinely diverging values — zero conflict markers appear for union, append, lastWriteWin, immutable, or validatedOverwrite reconciliations.
- **SC-003**: Applying the same patch twice, or applying two patches that touch the same node in either order, produces an identical resulting node file in 100% of tested cases for the six order-independent behaviors; for lastWriteWin, the result deterministically matches whichever patch was applied last, in 100% of tested cases.
- **SC-004**: A graph maintainer can determine exactly how a future contribution to any predicate will be reconciled by reading that predicate's own schema document alone, without reading arc's source code or knowing the node's type.
- **SC-005** *(BUG-001)*: Running `arc apply --verbose` against a merged node with N distinct present predicates reports N per-predicate outcome lines (name, resolved `MergeOp`, outcome), in addition to the existing per-node summary line, for 100% of tested merges.
- **SC-006** *(BUG-001)*: Across a freshly initialized graph, 100% of built-in predicates whose `role` is `text` are seeded with the `append` merge behavior.
- **SC-007** *(BUG-002)*: Re-applying a contribution to a `union`/`append` predicate that adds no genuinely new value (a full or Jaccard-near-duplicate of existing content) reports that predicate's outcome as `unchanged`, never `appended`, in 100% of tested cases.

## Assumptions

- Arc's internal representation of merge behaviors is corrected and widened as needed so all seven canonical behaviors (immutable, union, firstWriteWin, fillIfEmpty, lastWriteWin, append, validatedOverwrite) are genuinely distinct from one another; today's internal vocabulary collapses firstWriteWin and fillIfEmpty into one value and mislabels lastWriteWin, which this feature's own purpose requires fixing. This is treated as necessary groundwork for per-predicate dispatch, not as a change to the schema-document parsing mechanism itself (which stays out of scope) — only the set of behavior values that mechanism recognizes, and what already-seeded documents declare, changes.
- The whole-node `merge` field on a type's schema document (introduced as a temporary bridge so today's whole-node dispatch could keep working) is no longer consulted by arc apply once this feature ships. Removing that field from the schema-document shape itself is out of scope; it simply becomes unused by reconciliation.
- Building the "designated validation process" that may overwrite a validatedOverwrite predicate is a separate, not-yet-specified feature; this feature only guarantees that ordinary patch application never performs that overwrite itself.
- An already-existing graph whose predicate schema documents still declare the previous, coarser vocabulary is handled the same way any non-conforming schema document already is: arc fails clearly rather than guessing, and a maintainer re-initializes or hand-updates the affected documents. No automatic migration is introduced by this feature.
- lastWriteWin is resolved by arc's own application order, not by comparing a timestamp declared inside each contribution: doing the latter would require persisting, per lastWriteWin predicate, provenance of which contribution last wrote it — surviving across separate `arc apply` invocations, each a fresh process with no memory of prior merges — which in turn would require a visible on-disk shape change for those predicates. That is judged out of proportion to this feature's purpose (wiring already-declared per-predicate rules into reconciliation, not extending the node file format), so lastWriteWin instead adopts the same last-commit-wins convention git itself already applies to any tracked file; a future feature may revisit this if genuine timestamp-based provenance is ever needed.
- A predicate's already-established shape (a single value, a list, or prose) continues to determine what "combine," "grow," or "replace" concretely means for that predicate; this feature changes which behavior applies to a predicate, not how its shape is determined.
- ~~`abstract`/`definition`/`notes`/`relevance`/`description` (all `role: text`) are seeded `firstWriteWin`, distinguished from the generic `text`/`append` predicate on a per-name basis (research.md D2).~~ Superseded by FR-018 (BUG-001): every `text`-role predicate, including these five, is seeded `append` — a graph maintainer who wants one of them to freeze-and-flag instead (e.g. treating `abstract` as a fact set once) must hand-edit that predicate's own schema document, mirroring how any other non-default assignment already works (FR-013).

**Bugfix**: 2026-07-08 — BUG-001 added FR-017/FR-018, SC-005/SC-006, and the Predicate Merge Report entity: `arc apply --verbose` must report per-predicate (not just per-node) outcomes, and every `role: text` predicate must seed `append` rather than being distinguished per-name.

**Bugfix**: 2026-07-12 — BUG-002 added FR-019 and SC-007, and clarified FR-017/the Predicate Merge Report entity: a `union`/`append` predicate's reported outcome must reflect whether its reconciled value actually changed, not just which merge behavior dispatched — previously, `mergeTexts`/`mergeAttrs` always reported `appended` for this dispatch class even when the incoming contribution was a full or near-duplicate of existing content and nothing was actually added.
