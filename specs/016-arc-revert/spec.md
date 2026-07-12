# Feature Specification: Retract a Patch's Contribution from the Graph (`arc revert`)

**Feature Branch**: `016-arc-revert`

**Created**: 2026-07-12

**Status**: Draft

**Input**: User description: "`arc revert <source-id>` removes a previously-applied patch document's contribution from the graph by tracing it through the graph's own git history: locate its original ingest commit via its recorded `Source-Id`; when nothing has since touched what that commit changed (it is the latest change, or no other patch touched the same files), undo exactly that commit; when a node the patch introduced has since been modified by other patches, do not blanket-revert — instead remove a node only if this patch was its sole author and nothing since touched it (also removing all links to it), and for a node modified by others since, leave its metadata, predicates, edges, and links untouched and remove only the specific text content this patch itself contributed, identified via the graph's own commit history."

## Clarifications

### Session 2026-07-12

- Q: How should `arc revert` detect, on a later invocation, that a given patch has already been retracted — especially after a merge-aware (per-node) reconciliation, which doesn't produce a literal whole-commit undo the way the simple path does? → A: By checking whether the document's own source node file no longer exists in the graph — the same signal `arc apply` already uses for its own idempotency check (spec 003 FR-003), just inverted. No new commit-trailer convention is introduced for detection. This is independent of, and does not replace, the existing requirement that every revert — including the merge-aware path — still produces its own commit (FR-015).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Undo the patch just applied (Priority: P1)

A user applies a patch, immediately realizes it was the wrong document (wrong test fixture, mistaken source, bad extraction run), and wants the graph returned to exactly how it was a moment ago, before anything else happens to touch the graph.

**Why this priority**: This is the most common and lowest-risk use of a revert — undoing the very last thing done to the graph, with nothing else in between to reconcile against. If this fails to be a clean, complete undo, no more elaborate revert scenario can be trusted either.

**Independent Test**: Can be fully tested by applying one patch, immediately reverting it by its identifier, and confirming the graph's files and history are identical to their state before that patch was applied (aside from the revert's own new commit).

**Acceptance Scenarios**:

1. **Given** a patch was just applied and no other change has happened since, **When** the user reverts it by the identifier reported at apply time, **Then** every node file the patch created is removed, every node file it merely contributed to is restored to its pre-patch content, and no content from any earlier patch is affected.
2. **Given** a successful revert of the most recent patch, **When** the user inspects the graph afterward, **Then** exactly one new commit exists recording the retraction, and no file in the graph differs from its state immediately before the reverted patch was applied.
3. **Given** a revert completes, **When** the command finishes, **Then** the tool reports what was removed (nodes removed, nodes restored, links removed) so the user can confirm the outcome without a separate inspection step.

---

### User Story 2 - Retract an old patch that nothing has touched since (Priority: P2)

A user discovers, well after the fact, that a patch applied some time ago should never have been ingested — but every node and file that patch touched has been left alone by every patch applied afterward.

**Why this priority**: Mistakes are rarely caught the instant they happen. A revert that only works on the single most recent commit would be far less useful than one that recognizes when an older patch is still cleanly separable from everything applied since, even though it isn't literally the latest change.

**Independent Test**: Can be fully tested by applying an old patch, applying one or more unrelated patches afterward that touch none of the same files, then reverting the old patch and confirming the result is identical to a graph where that old patch was never applied, with the later, unrelated patches' content fully intact.

**Acceptance Scenarios**:

1. **Given** a patch applied earlier whose files have not been touched by any patch applied since, **When** the user reverts it, **Then** every file it touched is restored to its pre-patch content, exactly as if it had been the most recent change.
2. **Given** other, unrelated patches were applied after the one being reverted, **When** the revert completes, **Then** none of those later patches' contributions are altered or removed.

---

### User Story 3 - Retract a patch whose nodes were later enriched by other patches (Priority: P3)

A user wants to retract a patch that introduced a node later touched by one or more subsequent patches — for example, a node created by an early patch that a later patch went on to add new predicates or relations to. A blanket undo of the original commit would also destroy what the later patches contributed, since the graph's own merge step does not separately record which patch wrote which piece of an already-merged node.

**Why this priority**: This is the scenario that makes a naive revert unsafe and is the entire reason this feature needs more than a plain commit-undo. Without correct behavior here, retracting any patch that touched shared, frequently-mentioned content (a common entity, a heavily-cited resource) would risk silently destroying other patches' legitimate contributions — an unacceptable form of data loss for a graph whose value is its accumulated history.

**Independent Test**: Can be fully tested by applying two patches that both touch the same node — the first creating it, the second adding further content to it — then reverting the first patch and confirming the node still exists, still carries everything the second patch contributed, and no longer carries anything that only the first patch contributed.

**Acceptance Scenarios**:

1. **Given** a node created exclusively by the patch being reverted, with no other patch ever having modified it, **When** the revert runs, **Then** the node's file is removed entirely, and every link elsewhere in the graph that referenced that node is also removed, leaving no dangling reference.
2. **Given** a node created by the patch being reverted and later modified by a different patch, **When** the revert runs, **Then** the node file is not removed, its predicates, edges, and links contributed by the later patch remain exactly as they were, and only the body text the reverted patch itself contributed is taken out.
3. **Given** the reverted patch's own contribution to a shared node was, at the time it was written, flagged as a conflict against a still-later patch's differing value, **When** the revert runs, **Then** the reverted patch's recorded value is correctly identified and removed from within that flagged record without corrupting or losing the later patch's own value.
4. **Given** a single revert touches several nodes, some exclusively owned and some shared with other patches, **When** the revert completes, **Then** each node is reconciled according to its own case (removed outright, or reduced to just the other patches' contributions) within that one operation.

---

### Edge Cases

- What happens when the given identifier does not match any ingest commit in the graph's history? The tool must refuse with a clear explanation and make no changes.
- What happens when the identified patch has already been reverted? The tool must make no changes and report clearly that there is nothing to retract.
- What happens when the target directory is not an initialized graph? The tool must refuse and make no changes, the same way other graph-mutating commands do.
- What happens when a node the patch introduced is referenced by links from nodes elsewhere in the graph that the patch itself never touched? Every such reference must still be found and removed when that node is removed, regardless of which file it lives in.
- What happens when text the reverted patch contributed was, at the time it was written, judged a near-duplicate of content already present and therefore never distinctly recorded? There is nothing distinctly attributable to remove for that content, and the revert proceeds without error on that basis.
- What happens when reverting is interrupted partway (process killed, disk full, permission error)? The graph and its git history must be left exactly as they were before the attempt — no partially removed nodes, no partially stripped text, no dangling commit.
- What happens when every node the patch touched turns out to be exclusively owned by it (no shared nodes at all)? The revert behaves the same as reverting the most recent commit — a clean, whole removal of everything the patch contributed.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a patch's identifier (the `Source-Id` recorded at the time it was applied) and locate its original ingest commit from the graph's git history before making any change.
- **FR-002**: The tool MUST refuse to proceed, and MUST make no changes, when the given identifier does not correspond to any ingest commit in the graph's history.
- **FR-003**: The tool MUST detect that a patch has already been retracted by checking whether the document's own source node file no longer exists in the graph — the same signal `arc apply` already uses for its idempotency check, inverted (Clarifications, Session 2026-07-12). When detected, the tool MUST refuse to proceed, MUST make no changes, and MUST report clearly that there is nothing to do.
- **FR-004**: The tool MUST refuse to proceed, and MUST make no changes, when the target directory is not an initialized graph.
- **FR-005**: When the ingest commit being reverted is the most recent commit affecting the graph, the tool MUST undo exactly and only the changes that commit introduced.
- **FR-006**: When the ingest commit being reverted is not the most recent commit, but no other commit has since touched any file that ingest commit changed, the tool MUST undo exactly and only the changes that commit introduced, the same as FR-005.
- **FR-007**: When a file the ingest commit changed has since been changed again by another patch, the tool MUST NOT perform a blanket undo of the whole ingest commit, and MUST instead reconcile the affected node(s) individually as described in FR-008 through FR-011.
- **FR-008**: For each node the reverted patch introduced or modified, the tool MUST determine whether any other patch has modified that same node since.
- **FR-009**: A node that the reverted patch exclusively defined, with no other patch ever having modified it, MUST be removed from the graph entirely as part of the revert.
- **FR-010**: When a node is removed under FR-009, the tool MUST also remove every link anywhere else in the graph that references that node's identity, so no dangling reference to a removed node remains.
- **FR-011**: A node that has been modified by at least one patch other than the one being reverted MUST NOT be removed, and its predicates, attributes, edges, and links contributed by any other patch MUST remain exactly as they were before the revert.
- **FR-012**: For a node covered by FR-011, the tool MUST identify, using the graph's own commit history, the specific body text the reverted patch itself contributed to that node, and MUST remove only that text, leaving every other predicate, attribute, edge, and link on the node untouched.
- **FR-013**: When the text the reverted patch contributed to a shared node was, at the time it was written, recorded together with a later, differing contribution in a single flagged conflict record, the tool MUST correctly recover and remove only the reverted patch's own recorded value, MUST NOT remove or alter the other patch's recorded value, and MUST NOT leave a corrupted or malformed record behind.
- **FR-014**: When no distinctly attributable trace of the reverted patch's contribution to a node remains to be found (e.g. it was judged a near-duplicate of other content and never distinctly recorded), the tool MUST proceed without treating this as an error or leaving any part of the revert incomplete.
- **FR-015**: On success, the tool MUST produce exactly one commit capturing every change the revert made — removed node files, restored or reconciled node files, and removed links — leaving nothing uncommitted and nothing committed separately.
- **FR-016**: The tool MUST leave no partially-removed node, no partially-stripped text content, and no dangling commit when the revert fails or is interrupted at any point before its commit completes.
- **FR-017**: On success, the tool MUST report to the user what happened — nodes removed, nodes reconciled (content stripped) versus left untouched, and links removed — without requiring a separate inspection command to confirm the outcome.
- **FR-018**: The tool MUST report, as part of its output, which of the two reconciliation approaches (whole-commit undo, or per-node reconciliation) was used for the revert, so the user can tell which case they were in.
- **FR-019**: The tool MUST offer an opt-in, more detailed report that, for each node the revert touched, identifies the node and states how it was reconciled (removed, content stripped, or left untouched because another patch's contribution required it); this detail MUST NOT appear in the tool's default output.

### Out of Scope

- **Retracting more than one patch per invocation**: This feature covers retracting exactly one previously-applied patch, identified by its own `Source-Id`, per command invocation. Batch retraction of several patches at once is separate, later work.
- **Reviewing or resolving conflict records left by `arc apply`**: This feature only guarantees that its own removal of a reverted patch's contribution from a flagged conflict record does not corrupt or discard the other side's value (FR-013). Presenting, listing, or resolving conflict records in general remains the existing, separate mechanism.
- **Re-applying a retracted patch**: Restoring a patch's contribution after it has been retracted is a separate capability from retracting it in the first place.

### Key Entities

- **Patch / Source-Id**: The identifier recorded when a patch document was originally applied, used to locate that patch's ingest commit in the graph's history without the user needing to know or supply a commit hash.
- **Ingest Commit**: The single git commit `arc apply` produced when the patch was originally applied, carrying the trailer that records its `Source-Id`.
- **Exclusively-Owned Node**: A node introduced by the patch being reverted that no other patch has modified since — safe to remove in full.
- **Shared Node**: A node introduced or touched by the patch being reverted that at least one other patch has also modified since — never removed outright; only the reverted patch's own text contribution is taken out of it.
- **Node Contribution**: The specific piece of a node's content (a body-text block) attributable to one particular patch, as distinguished from what other patches contributed to the same node, traced via the graph's own commit history.
- **Reconciliation Approach**: Which of the two behaviors — undoing a whole commit, or reconciling affected nodes individually — a given revert used, reported to the user as part of its outcome.
- **Link**: A reference from one node to another; must never be left dangling (pointing at a node that no longer exists) once that node is removed by a revert.
- **Revert Commit**: The single commit a successful revert produces, capturing every node removed, every node's stripped content, and every link removed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Reverting the most recently applied patch removes 100% of what that patch added and nothing else, restoring the graph to exactly its state immediately before that patch was applied except for the revert's own new commit.
- **SC-002**: Reverting an older patch that no other patch has since touched produces a graph identical, aside from history bookkeeping, to one where that patch was never applied — verified across 100% of tested non-overlapping cases.
- **SC-003**: 0% of a later patch's contributions are lost when an earlier patch touching the same node is reverted — every predicate, edge, and text contribution any other patch added remains fully intact in 100% of tested cases.
- **SC-004**: A user can initiate and complete a revert using only the identifier reported when the patch was originally applied, with no need to look up or supply a git commit hash.
- **SC-005**: 100% of reverts that fail or are interrupted partway leave the graph and its history exactly as they were before the attempt — zero observed cases of partially removed nodes or dangling commits.
- **SC-006**: After a node is fully removed by a revert, a full-graph search finds 0 remaining references to that node's identity.
- **SC-007**: A user can determine, from the revert command's own output, which reconciliation approach was used, without inspecting the graph or its history manually.
- **SC-008**: 100% of attempts to revert an already-retracted patch result in zero graph or history changes, with a clear "nothing to do" outcome.

## Assumptions

- The graph being reverted was created and grown by `arc init`/`arc apply`, and each patch application corresponds to exactly one ingest commit, as already guaranteed by `arc apply` — this feature relies on that one-commit-per-patch invariant to identify which commits touched which files.
- Node rendering is deterministic and stable, so a commit's diff for a given node reflects only genuinely new or changed content, never an incidental rewrite of unrelated content — an existing precondition this feature depends on rather than introduces.
- "Text content" a patch contributed refers to body-prose content on a node; scalar and list-valued predicates, edges, and links on a shared node are left untouched by a revert regardless of which patch originally wrote them (FR-011), consistent with the user's own instruction to never blanket-modify a shared node's metadata.
- Reverting is fully local and offline, consistent with `arc apply` — no network access is required or attempted.
- A patch's already-flagged conflict record (from a prior `arc apply`) self-documents which value belongs to which side, making it possible to remove one side's value without additional stored provenance.
- A document's own source node is always exclusively created by its own patch — no other patch ever writes to another document's source record — so the source node's continued existence reliably indicates the patch has not been retracted, and its absence reliably indicates it already has been (Clarifications, Session 2026-07-12).
