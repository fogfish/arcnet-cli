# Feature Specification: Batch Apply a Directory of Patches (`arc apply batch`)

**Feature Branch**: `020-apply-batch`

**Created**: 2026-08-06

**Status**: Draft

**Input**: User description: "`arc apply batch <dir>` — apply every `*.md` patch in a directory recursively in published-date order, skipping already-ingested sources; each patch is still exactly one commit"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ingest a whole corpus in one command (Priority: P1)

A user has a directory holding the extracted patches for many documents — the output of an extraction pipeline run over a reading backlog, an archive export, or a colleague's shared bundle — and wants every one of them in the graph without invoking the single-patch command once per file.

**Why this priority**: This is the feature's entire reason to exist. Today the only way to ingest fifty documents is fifty invocations, hand-ordered by the user so that history accumulates chronologically; getting that order wrong silently produces a graph whose timeline and first-writer-wins merges reflect the order files happened to be listed in, not the order the documents were published.

**Independent Test**: Can be fully tested by pointing the command at a directory of patches for documents not yet in the graph, then inspecting the resulting graph and git history — the graph has grown by every document's contribution and history carries one commit per document, oldest publication first.

**Acceptance Scenarios**:

1. **Given** an initialized graph and a directory containing several well-formed patches for documents not yet tracked, **When** the user runs the batch command against that directory, **Then** every patch is applied and the graph carries each document's contribution, exactly as if each patch had been applied individually.
2. **Given** a directory whose patches carry differing publication dates, **When** the batch is applied, **Then** the patches are applied in ascending publication-date order (oldest first), regardless of filename or filesystem enumeration order.
3. **Given** a batch of N patches that all apply successfully, **When** the user inspects the git history afterward, **Then** exactly N new commits exist — one per patch, each carrying the same subject, stats, and document-identity trailer a single-patch application would have produced — and no commit spans two documents.
4. **Given** patches nested in subdirectories beneath the target directory, **When** the batch is applied, **Then** those patches are discovered and applied too, ordered by publication date alongside the top-level ones rather than grouped per directory.
5. **Given** a completed batch, **When** the command finishes, **Then** it reports how many patches were applied, how many were skipped, and how many failed, so the user can confirm the outcome without a separate inspection step.

---

### User Story 2 - Re-run a batch safely over a directory that is partly ingested (Priority: P1)

A user re-runs the batch against the same directory — because new patches were added to it, because an earlier run was interrupted, or simply because they are unsure how far the last run got.

**Why this priority**: A batch over a large corpus is exactly the operation most likely to be interrupted or repeated, and the directory it reads is a growing working folder, not a one-shot input. Without a safe re-run, the user must track by hand which files were already ingested — reintroducing the bookkeeping this command exists to remove.

**Independent Test**: Can be fully tested by running the batch twice over the same directory and confirming the second run creates no commits and reports every patch as already tracked.

**Acceptance Scenarios**:

1. **Given** a directory whose patches have all already been ingested, **When** the user runs the batch again, **Then** no filesystem or git change occurs, every patch is reported as already tracked, and the command succeeds.
2. **Given** a directory of patches of which some are already ingested and some are new, **When** the batch is run, **Then** only the not-yet-tracked documents are applied — each as its own commit — and the already-tracked ones are skipped without a commit.
3. **Given** a batch run that was interrupted partway through, **When** the user re-runs the same batch, **Then** the documents committed before the interruption are skipped and the run continues from the first document that had not yet been ingested.
4. **Given** two patches in the directory that describe the same document, **When** the batch is applied, **Then** the first one applied ingests the document and the second is skipped as already tracked, leaving one commit for that document rather than two.

---

### User Story 3 - One bad patch does not abandon the rest of the corpus (Priority: P2)

A user's directory contains one patch the tool cannot apply — a truncated file, a manifest missing its publication date, a body that does not follow the expected node-section structure — among dozens of good ones.

**Why this priority**: In a real corpus, extraction failures are individually rare and collectively certain. If a single malformed file abandoned the run, batching a large directory would become a slow, manual bisect — apply, hit the failure, remove the file, re-run — which is worse than the per-file invocation it replaces.

**Independent Test**: Can be fully tested by placing one malformed patch among several valid ones and confirming that every valid patch is applied, the malformed one is reported by path with the reason, and the command exits with a failure status.

**Acceptance Scenarios**:

1. **Given** a directory containing one malformed patch among several valid ones, **When** the batch is applied, **Then** every valid patch is applied and committed, and the malformed one is skipped without leaving any partial graph state or dangling commit.
2. **Given** a batch run in which at least one patch failed, **When** the command finishes, **Then** it reports each failure by file path with the reason it could not be applied, alongside the counts of applied and skipped patches.
3. **Given** a batch run in which at least one patch failed, **When** the command exits, **Then** it exits with a non-zero status distinguishable from success, so a script or CI job does not mistake a partly-failed run for a clean one.
4. **Given** a batch run in which every patch failed, **When** the command finishes, **Then** it reports every failure and exits non-zero, with no commits produced.

---

### User Story 4 - Halt the batch at the first failure (Priority: P3)

A user running a batch as part of an automated pipeline — or over a corpus where a failure signals something systematically wrong with the extraction run — wants the batch to stop the moment a patch fails, rather than working through the remainder.

**Why this priority**: Valuable, but a refinement of the default: continuing and reporting at the end already surfaces every failure, so this only changes how early the user learns of it and how much of the corpus is committed by then. It is the smaller, opt-in half of the failure story and can ship after User Story 3.

**Independent Test**: Can be fully tested by running the batch in strict mode over a directory with a malformed patch positioned in the middle of the publication order, and confirming the run stops there: patches ordered before it are committed, patches ordered after it are untouched and reported as unprocessed.

**Acceptance Scenarios**:

1. **Given** a directory with a malformed patch among valid ones, **When** the user runs the batch in strict (halt-on-first-failure) mode, **Then** the run stops at that patch: every patch ordered before it is committed, and no patch ordered after it is applied.
2. **Given** a strict-mode run that halted, **When** the command finishes, **Then** it names the patch that failed and reports how many patches were left unprocessed, and exits with a non-zero status.
3. **Given** a strict-mode run that halted, **When** the user fixes the offending patch and re-runs the batch, **Then** the already-committed documents are skipped and the run resumes from the repaired patch onward.

---

### Edge Cases

- What happens when the target directory contains `.md` files that are not graph patches (a README, hand-written notes, a schema-only patch meant for the schema import command)? They are passed over and counted as "not a patch" in the summary — never applied, and never treated as a failure. Pointing the batch at a mixed content directory is safe.
- What happens when the target directory contains files that are not Markdown at all? They are ignored entirely and do not appear in the summary.
- What happens when the target directory contains no applicable patches at all (empty, or nothing but non-patch files)? The command succeeds, makes no changes, and says plainly that it found nothing to apply.
- What happens when the target path does not exist, or is a file rather than a directory? The command refuses with a clear explanation and makes no changes.
- What happens when the current directory is not an initialized graph? The command refuses and makes no changes, the same way every other graph-mutating command does — before reading any patch.
- What happens when two patches carry the same publication date? Their relative order is deterministic and stable across runs and across machines, resolved by their path, so repeated runs over the same directory produce the same history.
- What happens when a patch's manifest carries no publication date, or one that cannot be interpreted? It cannot be placed in publication order, so it is treated as a failed patch under the run's failure-handling rule rather than being applied at an arbitrary position.
- What happens when the directory tree contains the graph's own version-control metadata, or other hidden directories? They are not descended into; batch discovery never treats version-control internals as patch input.
- What happens when applying one patch flags a merge conflict? That patch still counts as applied — a flagged conflict is a recorded outcome, not a failure — and the run continues, with the conflicted files surfaced in the final summary so they are not lost among the other output.
- What happens when the batch is interrupted (the user presses Ctrl-C, the process is killed) partway through a patch? Documents committed before the interruption remain committed; the in-flight patch leaves no partial graph state and no dangling commit, exactly as the single-patch command guarantees.
- What happens when a patch introduces an unregistered node kind? The batch behaves exactly as the single-patch command does — the node is applied with the default merge behavior and a warning is surfaced — the run is not refused on that basis.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a single directory path as input and discover every Markdown file beneath it, recursively through subdirectories.
- **FR-002**: The tool MUST refuse, and MUST make no changes, when the given path does not exist, is not a directory, or when the current graph is not initialized — checked before any patch is read.
- **FR-003**: The tool MUST classify each discovered Markdown file as an applicable graph patch or not, and MUST pass over the non-applicable ones without applying them and without counting them as failures; the count of passed-over files MUST appear in the run's summary.
- **FR-004**: The tool MUST determine each applicable patch's publication date from its manifest and MUST apply the patches in ascending publication-date order, oldest first, independent of filename or filesystem enumeration order.
- **FR-005**: When two or more patches share the same publication date, the tool MUST order them deterministically by their path relative to the target directory, so that repeated runs over unchanged input produce identical history.
- **FR-006**: The tool MUST apply each patch with exactly the same creation, merging, timeline-derivation, conflict-flagging, and unregistered-kind behavior as a single-patch application — the batch command introduces no separate application path or altered semantics.
- **FR-007**: The tool MUST produce exactly one commit per successfully applied patch, carrying the same subject, stats, and document-identity trailer a single-patch application produces; the tool MUST NOT combine two documents into one commit, nor split one document across two.
- **FR-008**: The tool MUST skip any patch whose document is already tracked in the graph, making no filesystem or git change for it, and MUST count it as skipped in the run's summary.
- **FR-009**: The tool MUST evaluate the already-tracked check against the graph's state at the moment each patch is reached, so that a document ingested earlier in the same run is recognized as tracked by a later, duplicate patch in that run.
- **FR-010**: By default, when a patch fails to apply, the tool MUST record the failure and continue with the remaining patches in order.
- **FR-011**: The tool MUST offer an opt-in strict mode that halts the run at the first failing patch, leaving every later patch unprocessed, and MUST report how many patches were left unprocessed.
- **FR-012**: A failing patch MUST leave no partial graph state and no dangling commit, whether the run continues or halts; commits produced for previously applied patches MUST remain intact and valid.
- **FR-013**: On completion, the tool MUST report a summary covering: the number of patches applied, skipped as already tracked, passed over as not applicable, and failed — plus, for each failure, the file path and the reason it could not be applied.
- **FR-014**: The tool MUST exit with a success status when no patch failed (including a run in which every patch was skipped or nothing applicable was found), and with a non-zero status distinguishable from success when at least one patch failed.
- **FR-015**: While the run is in progress, the tool MUST report per-patch outcomes as they happen — the document applied, skipped, or failed — so the user is not left without output during a long batch; this per-patch reporting MUST be suppressed in the tool's quiet mode.
- **FR-016**: The tool MUST surface, in the final summary, every merge conflict flagged across the whole run, identifying the files that need manual resolution, so conflicts raised early in a long batch are not lost in the scrollback.
- **FR-017**: The tool MUST surface unregistered-kind warnings raised by individual patches without aborting the run, consistent with single-patch behavior.
- **FR-018**: The tool MUST provide a machine-readable output mode whose result carries the per-patch outcomes (document identity, source file path, outcome, and reason on failure) and the run-level counts, so a pipeline can consume the batch result without parsing human-readable text.
- **FR-019**: The tool MUST NOT descend into version-control metadata directories or other hidden directories when discovering patches.
- **FR-020**: A patch whose publication date is absent or uninterpretable MUST be treated as a failed patch under FR-010/FR-011, never applied at an arbitrary position in the order.
- **FR-021**: The tool MUST NOT modify, move, delete, or rename any file in the target directory; the patch directory is read-only input to this command.
- **FR-022**: When the target directory yields no applicable patches, the tool MUST succeed, make no changes, and state plainly that there was nothing to apply.

### Out of Scope

- **Schema patches**: Importing `Property`/`Class` definitions is the schema import command's responsibility. Schema-only patches encountered in the directory are passed over per FR-003; batch schema import is separate, later work.
- **Fetching patches from a URL or catalog**: This feature reads a local directory only. Downloading a bundle of patches for batch application is separate, later work.
- **Reordering or rewriting history**: The batch produces commits in publication order as it goes; it never rewrites, squashes, or reorders commits already in the graph.
- **Conflict resolution**: As with single-patch application, this feature's responsibility ends at surfacing flagged conflicts (FR-016). Resolving them is separate, later work.
- **Retrying failed patches**: A failed patch is reported, not retried. Re-running the batch after a fix is the supported recovery path (FR-008 makes that safe).
- **Parallel application**: Commits are produced one per patch in a defined order; applying patches concurrently is not part of this feature.

### Key Entities

- **Patch Directory**: The user-supplied root of the recursive search; read-only input, never modified by the run.
- **Discovered File**: One Markdown file found beneath the patch directory, classified as either an applicable graph patch or passed over.
- **Batch Plan**: The applicable patches ordered by publication date (ties resolved by relative path) — the exact sequence in which they will be applied, fixed before the first commit.
- **Patch Outcome**: The result recorded for one patch in the plan — applied (with its resulting commit and node counts), skipped as already tracked, or failed (with the reason) — plus, in strict mode, unprocessed for patches never reached.
- **Batch Summary**: The run-level result: counts by outcome, the failures with their paths and reasons, and the conflicts flagged across the whole run.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can ingest a directory of documents with a single command invocation, with no manual ordering step and no per-file invocation.
- **SC-002**: 100% of successful batch runs produce exactly one commit per applied patch — never fewer (documents combined) and never more (a document split or re-committed).
- **SC-003**: 100% of batch runs apply patches in ascending publication-date order; running the same directory into two fresh graphs produces the same commit sequence both times.
- **SC-004**: 100% of re-runs over an unchanged, fully-ingested directory produce zero graph changes and zero commits, and exit successfully.
- **SC-005**: A batch containing one unapplicable patch among valid ones applies 100% of the valid patches in default mode, and reports the unapplicable one by path with a reason.
- **SC-006**: 100% of runs with at least one failed patch exit with a non-zero status; 100% of runs with no failed patch exit successfully.
- **SC-007**: 100% of failed or interrupted patches leave zero partial node writes and zero dangling commits; commits produced earlier in the same run remain intact.
- **SC-008**: A user can determine, from the command's own output alone, which documents were applied, which were already tracked, which were passed over, and which failed and why — without a follow-up inspection command.
- **SC-009**: Pointing the batch at a directory that mixes patches with unrelated Markdown files applies 100% of the patches and 0% of the unrelated files, and reports zero failures on account of those files.
- **SC-010**: In strict mode, 100% of runs halt at the first failing patch, with zero patches ordered after it applied.
- **SC-011**: A batch of 100 typical single-document patches completes without user intervention, and the user sees per-patch progress throughout rather than a silent wait until the end.

## Assumptions

- The graph the batch is applied to already exists and was created by the graph's initialization command; this feature does not initialize a graph.
- The batch command is a subcommand of the existing apply command surface, and defers entirely to the existing single-patch application for the semantics of applying one patch — this feature adds discovery, ordering, iteration, and reporting around it, not new merge or commit behavior.
- Strict (halt-on-first-failure) mode is opt-in via an explicit flag; continuing through failures and reporting at the end is the default (User Story 3 / FR-010).
- "Applicable graph patch" means a Markdown file whose front matter declares the patch manifest the single-patch command already recognizes. A file with no such manifest is passed over (FR-003); a file that declares the manifest but is otherwise malformed is a failure (FR-010/FR-020).
- Publication dates are compared as calendar dates; a date with no time component orders by date alone, and patches sharing a date fall back to path ordering (FR-005).
- Applying a batch is fully local and offline — no network access is required or attempted.
- Patch files are trusted input produced by a compatible extraction pipeline; this feature validates structure but does not authenticate origin.
- Symbolic links encountered during discovery are not followed into directories outside the target tree, so a link cycle cannot make discovery run forever.
- The existing quiet, verbose, and machine-readable output modes apply to this command with their established meanings.
