---

description: "Task list for 023-core-vocabulary-conformance"
---

# Tasks: ARCNET-CORE v0.11 — Schema Vocabulary Conformance

**Input**: Design documents from `/specs/023-core-vocabulary-conformance/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [constitution.md](../../.specify/memory/constitution.md) (governs Phase 2 and Phase N)

**Tests**: NOT optional. Constitution Principles VI and VIII require every one of the acceptance
scenarios in `spec.md` to map 1:1 to a colocated E2E test, written before implementation
(red-green-refactor). Originally 29, spanning US1–US5; **US5 (7 scenarios) was removed
post-implementation** (see Phase 7 below) — 22 remain, all passing.

**Organization**: Grouped by user story. US1–US4 deliver value to new graphs and are independently shippable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete work)
- **[Story]**: US1–US5, mapping to `spec.md`

## Path Conventions

Per [plan.md](plan.md) "Project Structure". This feature touches `internal/core`, `internal/app/schema`, `internal/app/ctrl`, `internal/app/lint`, and adds one command under `cmd/arc/ctrl/`.

---

## ⚠️ Read First: Two Ordering Constraints That Are Not Negotiable

1. **The golden baseline (T001–T003) must be captured BEFORE any table edit.** Built after the
   change it certifies nothing. `CORE-FIX.md` §7 records that two of this feature's findings
   survived four spec revisions precisely because the seeded vocabulary was only ever reviewed as
   diffs of Go map literals.

2. ~~Deleting `MergeValidatedOverwrite` (Phase 8) must come AFTER `arc upgrade` ships (Phase 7).~~
   **No longer applicable.** This constraint existed to sequence Phase 8's breaking change after
   a remedy command, `arc upgrade`, shipped. That command was implemented, then removed
   post-implementation — `arc` is pre-1.0/experimental and the compatibility/migration machinery
   was judged unnecessary tech debt. Phase 8 now stands alone; see its note in place of the old
   Phase 7.

---

## Phase 1: Setup — Golden Baseline

**Purpose**: Build the regression net before changing anything it protects. No new module
dependency is required by this feature.

- [X] T001 Add golden-snapshot test with an `-update` flag in `internal/app/schema/service/seed_golden_test.go`, writing one file per `Seed()` map key under `testdata/golden/schema/` (contract [C2.1](contracts/seeded-schema-contract.md))
- [X] T002 Generate the baseline by running `go test ./internal/app/schema/service -run TestSeedGolden -update` against **unmodified** `Seed()` output and commit `testdata/golden/schema/` as the pre-change baseline
- [X] T003 [P] Confirm `go test ./...` and `staticcheck ./...` are clean at the baseline commit

---

## Phase 2: Design Preconditions

**Purpose**: Constitution Compliance Checklist preconditions. The deliverable of each task is a
recorded design decision, not working code.

**⚠️ CRITICAL**: No implementation (Phase 3+) begins until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add glossary rows for *Built-in Vocabulary* and *Merge Vocabulary* to `ARCHITECTURE.md`, and update the existing *Predicate Schema Node* / *Type Schema Node* rows to drop the retired type-level merge (a third row, *Schema Upgrade*, was added for `arc upgrade` and later removed along with it — see Phase 7)
- [X] ~~T005~~ **Removed along with Phase 7/`kernel.UpgradeResult`** — verified `kernel.UpgradeResult` no name clash at the time; the type no longer exists

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] ~~T006~~ **Removed along with Phase 7** — confirmed the (now-deleted) `arc upgrade` verb/flag/exit-code/`--json` design against `contracts/upgrade-command-contract.md` (also deleted)
- [X] T007 [P] Confirm the `ErrSchemaInvalid` message contract in [contracts/merge-vocabulary-contract.md](contracts/merge-vocabulary-contract.md) C1.2 names the offending document (the "six legal values + `arc upgrade` remedy" extension this originally checked for was implemented, then reverted along with Phase 7)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] ~~T008~~ **Removed along with Phase 7** — confirmed the (now-deleted) `arc upgrade` introduced no new external system

### Phase 2d: E2E Acceptance Test Design (Principle VIII) — red phase

All 29 scenarios. Tests MUST compile and fail semantically before any Phase 3+ task starts.

- [X] T009 [P] [US1] Write 6 E2E tests in `cmd/arc/graph/apply_test.go` for US1 scenarios 1–6 via `sut()`; scenario 1 MUST use **reworded** prose below the near-duplicate threshold, not identical prose, or it passes vacuously against `main` ([plan.md](plan.md) F2, [research.md D4](research.md))
- [X] T010 [P] [US2] Write 6 E2E tests in `cmd/arc/ctrl/init_test.go` for US2 scenarios 1–6, asserting the seeded merge menu, absence of `merge:` on `Class` nodes, conformant score predicates, out-of-menu rejection, legacy-attribute tolerance, and a clean lint
- [X] T011 [P] [US3] Write 5 E2E tests in `cmd/arc/lint/lint_test.go` for US3 scenarios 1–5 against the hand-built v0.11 fixture from T016 — **not** against `arc init` output, or they pass vacuously (`CORE-FIX.md` §5.7)
- [X] T012 [P] [US4] Write 5 E2E tests in `cmd/arc/graph/apply_test.go` and `cmd/arc/lint/lint_test.go` for US4 scenarios 1–5, asserting registration, zero diagnostics, zero predicates created, repeated-author union, and the citation predicate's merge
- [X] ~~T013~~ **Removed along with Phase 7** — the 7 US5 E2E tests in `cmd/arc/ctrl/upgrade_test.go` no longer exist; the file was deleted

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T014 Confirm this feature introduces no configuration value, environment variable, or secret

**Checkpoint**: Phase 2 complete — implementation may begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Shared scaffolding US3 and US4 depend on.

- [X] ~~T015~~ **Removed along with Phase 7** — added `kernel.UpgradeResult`, which no longer exists
- [X] T016 [P] Build a hand-written, v0.11-shaped fixture graph under `internal/app/lint/service/testdata/v011-graph/` — an `Entity` with no `published`/`created`, a `Timeline` carrying only `cites::` bullets, a `Source` with all four required predicates, a `Reference` with `title` alone
- [X] ~~T017~~ **Removed along with Phase 7** — registered `ctrl.NewUpgradeCmd()`, which no longer exists

**Checkpoint**: Foundation ready.

---

## Phase 3: User Story 1 — A document's summary holds a single, first-fixed value (Priority: P1) 🎯 MVP

**Goal**: Type-specific prose predicates stop accumulating paragraphs; a divergent contribution
preserves the established value and is flagged for review.

**Independent Test**: Apply a patch with a summary, apply a second whose summary is a substantial
rewording, confirm the stored value is the first one and the divergence was reported. Requires no
change to type contracts, folders, or lint.

> E2E tests written in T009 and currently failing (red).

- [X] T018 [US1] Change `abstract` and `description` from `MergeAppend` to `MergeFirstWriteWin` in `internal/app/schema/kernel/schema.go` `CorePredicateDefs` (FR-013; §10.2, §10.8)
- [X] T019 [US1] Change `definition` and `relevance` from `MergeAppend` to `MergeFirstWriteWin` in the same map (FR-013; [research.md D8](research.md) — these are `Entity` and `Reference` leading-prose keys, so omitting them fixes drift only for `Source`)
- [X] T020 [US1] Change `cites` from `MergeAppend` to `MergeUnion` in the same map, leaving `role: link` unchanged as a documented divergence (FR-014; §10.6, [research.md D6](research.md) note 1)
- [X] T021 [US1] Regenerate `testdata/golden/schema/` and **review the diff as a content change** — it is the reviewable artifact for every predicate this phase touches
- [X] T022 [US1] Add unit tests in `internal/core/merge_test.go` with `github.com/fogfish/it/v2` covering `firstWriteWin` over reworded prose, identical prose, and absent-then-present, asserting the `OutcomeFlagged` / `OutcomeUnchanged` / `OutcomeFilled` labels
- [X] T023 [US1] Update the stale comment in `internal/core/merge.go` `mergeTexts` that claims `"notes"` declares `firstWriteWin` — `notes` is `append` per §10.7 and stays that way

**Checkpoint**: US1's 6 E2E tests pass; prose drift is fixed for all four first-fixed predicates.

---

## Phase 4: User Story 2 — A newly initialized graph declares only conformant merge behaviour (Priority: P2)

**Goal**: A fresh graph seeds only conformant merge values and carries no type-level merge.

**Independent Test**: `arc init` into an empty directory; confirm the union of `merge:` values
across `_schema/Property/` is exactly the six, no `_schema/Class/` file carries `merge:`, and the
graph lints clean.

> Scenario 4 (out-of-menu rejection) is delivered by **Phase 8**, not here — see the ⚠️ note. The
> other 5 scenarios complete in this phase.

> E2E tests written in T010 and currently failing (red).

- [X] T024 [US2] Change `scoreZ` and `scoreC` from `MergeValidatedOverwrite` to `MergeLastWriteWin` in `internal/app/schema/kernel/schema.go`, and rewrite both descriptions to drop the "overwritten only by that designated pass" claim the merge algebra never enforced (FR-003; [research.md D7](research.md))
- [X] T025 [US2] Delete the `Merge` field from `core.TypeDef` in `internal/core/rules.go` (FR-005; [data-model.md §2](data-model.md) records the live-consumer audit — nothing reads it)
- [X] T026 [US2] Remove the `Merge:` entry from all 8 `CoreTypeDefs` literals in `internal/app/schema/kernel/schema.go`, and update the map's doc comment which still describes the field as "kept for continuity"
- [X] T027 [US2] Stop emitting the `merge` attribute in `typeNode` in `internal/app/schema/service/schema.go` (FR-005)
- [X] T028 [US2] Keep `decodeTypeDef`'s tolerance for a legacy `merge` attribute on a `Class` node — drop `rawType.merge` and the `validMergeOps` lookup, but continue succeeding on its presence, and add **no lint rule** (FR-006; contract [C1.3](contracts/merge-vocabulary-contract.md))
- [X] T029 [US2] Remove the `merge` attribute from `RegisterType`'s auto-registration node in `internal/app/schema/service/schema.go`, which still writes `merge: union` onto every auto-created `Class` document
- [X] T030 [US2] Regenerate `testdata/golden/schema/` and review the diff — all 8 type documents lose a front-matter line
- [X] T031 [US2] Add unit tests in `internal/app/schema/service/schema_test.go` asserting that a `Class` document carrying `merge: union`, `merge: nonsense`, and no `merge` all resolve identically

**Checkpoint**: 5 of US2's 6 E2E tests pass; a fresh graph is conformant. Scenario 4 awaits Phase 8.

---

## Phase 5: User Story 3 — Conformant nodes stop being reported as violations (Priority: P3)

**Goal**: A graph shaped exactly as ARCNET-CORE v0.11 describes lints clean.

**Independent Test**: Lint the T016 fixture graph and confirm zero missing-required-predicate
diagnostics, while a `Source` missing `published` still reports one.

**Every change in this phase is a strict relaxation** ([data-model.md §5](data-model.md)) — no node
that lints clean today can start failing.

> E2E tests written in T011 and currently failing (red).

- [X] T032 [US3] Empty `Node`'s `Required` and move `published`/`created` into its `Optional` in `internal/app/schema/kernel/schema.go` (FR-007; §11.1)
- [X] T033 [US3] Add `published` to `Source`'s own `Required` and `tags` to its `Optional` (FR-008, FR-029; §11.2)
- [X] T034 [US3] Reduce `Timeline`'s `Required` to `cites` alone, demoting `granularity` and `period` to `Optional` alongside `heading` (FR-009; §11.5)
- [X] T035 [US3] Reduce `Reference`'s `Required` to `title` alone, retaining `ref`, `relevance`, `status`, `notes` in `Optional` (FR-028; §11.6, [research.md D3](research.md)) — **this supersedes spec 022's recorded Clarification**; note the supersession in `specs/022-reference-type-folders/spec.md`'s Clarifications section rather than leaving it silently contradicted
- [X] T036 [US3] Add `tags` to `Entity`'s `Optional`; add `description` to `Class`'s `Required` and `subClassOf` to its `Optional` (FR-029; §9.2, §11.3 — the tool writes `subClassOf::` onto five seeded `Class` documents while `Class` never declared it, a violation the seeded tree inflicts on itself)
- [X] T037 [US3] Regenerate `testdata/golden/schema/` and review the diff
- [X] T038 [US3] Update `internal/app/lint/service/rules_type_conformance_test.go` to assert against the T016 fixture; delete any expectation that a node requires `published` or `created` by inheritance
- [X] T039 [US3] Verify contract [C2.2c](contracts/seeded-schema-contract.md) still holds — every predicate named in any `Class` document's bullets has its own `Property` document — and add it as an assertion in `seed_golden_test.go`

**Checkpoint**: US3's 5 E2E tests pass; the seeded tree is closed under predicate reference.

---

## Phase 6: User Story 4 — The specification's own metadata predicates are registered (Priority: P4)

**Goal**: `author`, `about`, and `genre` resolve against the seeded vocabulary, and every
predicate the spec aligns to a standard term declares it.

**Independent Test**: Apply a patch using all three predicates to a fresh graph and confirm it
lints clean, without touching merge behaviour or type requirements.

> E2E tests written in T012 and currently failing (red).

- [X] T040 [US4] Add `author` (`meta`/`union`/`schema:author`), `about` (`meta`/`union`), and `genre` (`meta`/`union`) to `CorePredicateDefs` in `internal/app/schema/kernel/schema.go`, with descriptions drawn from §10.2 including its enumerated `about` and `genre` value sets (FR-011)
- [X] T041 [P] [US4] Add the six missing `Aligned` terms in the same map — `title`→`schema:title`, `abstract`→`schema:abstract`, `url`→`schema:url`, `doi`→`schema:doi`, `aliases`→`skos:altLabel`, `authors`→`schema:author` (FR-026; §10.1, §10.2, §10.7)
- [X] T042 [P] [US4] Change `year` from `MergeFillIfEmpty` to `MergeImmutable` in the same map (FR-027; §10.7)
- [X] T043 [US4] Regenerate `testdata/golden/schema/` and review the diff — three new files plus six `aligned:` lines
- [X] T044 [US4] Add an assertion in `seed_golden_test.go` for contracts [C2.2d](contracts/seeded-schema-contract.md) and SC-008: the three predicates exist, and no seeded predicate omits an `aligned` term the spec assigns it

**Checkpoint**: US4's 5 E2E tests pass. Note that no change to `distinctPredicates` is made or needed — front-matter predicates never reached auto-registration in the first place ([research.md D5](research.md)).

---

## Phase 7: User Story 5 — REMOVED

> **Removed 2026-08-23, post-implementation.** T045–T052 implemented `arc upgrade` — the plan and
> Phase 8's ordering below assumed it. It was then removed by explicit decision: `arc` is pre-1.0
> and experimental, and the compatibility/migration machinery (`planUpgrade`/`applyUpgrade`, the
> prose-drift scan, the write-before-resolve ordering, `ctrl.Upgrade`, `cmd/arc/ctrl/upgrade.go`,
> `kernel.UpgradeResult`, `ctrl/port.SchemaResolver`) was judged unnecessary tech debt at this
> stage. `US5` and its 7 E2E tests (`cmd/arc/ctrl/upgrade_test.go`) are deleted along with it.
> Phase 8 no longer depends on this phase — see its note below.

---

## Phase 8: Merge Vocabulary Enforcement — completes US2

**Purpose**: The breaking half of US2. Originally sequenced after a Phase 7 that no longer
exists — see the note above. Tightening `validMergeOps` makes every graph seeded by a previous
release fail to load; this is accepted as a plain breaking change (FR-001/FR-002/FR-004), with
no remedy command, because the tool has no compatibility guarantee yet. See `ARCHITECTURE.md`'s
Compatibility Policy.

- [X] T053 [US2] Delete `MergeValidatedOverwrite` from `internal/core/ast.go` and correct the `MergeOp` doc comment from "seven-value menu" to six (FR-001, FR-004) — let the compiler enumerate the break sites; do not grep-and-replace (`CORE-FIX.md` §5.7)
- [X] T054 [US2] Remove `validatedOverwrite` from `mergeScalar`'s freeze class in `internal/core/merge.go`, and update the doc comment that lists it alongside `immutable` (contract [C1.4](contracts/merge-vocabulary-contract.md))
- [X] T055 [US2] Remove `core.MergeValidatedOverwrite` from `validMergeOps` in `internal/app/schema/service/schema.go`, leaving exactly six entries (FR-001)
- [X] T056 [US2] `ErrSchemaInvalid` in `internal/app/schema/service/errors.go` names the document path and field (FR-002); the "six legal values + `arc upgrade` remedy" extension was implemented, then reverted along with Phase 7 — the error is the plain, pre-existing generic form
- [X] T057 [US2] Update `internal/core/ast_test.go` to assert exactly six `MergeOp` values by exhaustive comparison against a literal set, so a seventh cannot be reintroduced silently (contract C1.1)
- [X] T058 [US2] Turn US2 E2E scenario 4 (out-of-menu rejection) green in `cmd/arc/ctrl/init_test.go`, asserting the document path is named (no remedy to assert — see T056)

**Checkpoint**: All 6 US2 E2E tests pass. The merge vocabulary is closed at six.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T059 [P] Add the SC-001 idempotency property test in `internal/core/merge_test.go`: `apply(apply(g, p), p) == apply(g, p)` over a patch exercising `abstract`, `definition`, `relevance`, `description`, `cites`, `text`, and every union predicate
- [X] T060 [P] Update `README.md` — the `arc init` paragraph's description of the seeded vocabulary (FR-025); no `arc upgrade` section — see Phase 7
- [X] T061 [P] Update `ARCHITECTURE.md` glossary rows written in T004 to their final form, and record the breaking-change/no-compatibility-path decision in a Compatibility Policy section (Principle I)
- [X] T062 [P] Update `specs/VISION.md`, which still described the pre-0.5 `kind` model and a `_meta/predicates.md` path that no longer exists (FR-025)
- [X] T063 Run every remaining scenario in [quickstart.md](quickstart.md) end to end against a real binary (US5's walkthrough is removed along with the command)
- [X] T064 Confirm `go test ./... -cover` are clean (`staticcheck` unusable in this environment — pre-existing Go-toolchain export-data mismatch, not a regression; `go vet` used instead)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). Retained verbatim.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I)
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV)

**Note for TN04**: no new ADR was needed for either the removed `arc upgrade` command (which
would have followed ADR 001's existing `cmd → component → service → kernel` path) or its removal.
**TN15 is not a formality here**: Phase 8 is a breaking change to graphs in the field, now with no
remedy command at all.

### Verification record (2026-08-23, revised same day — `arc upgrade` removed)

`arc upgrade` was implemented, verified (first pass below, superseded), and then removed by
explicit decision: `arc` is pre-1.0/experimental, and the compatibility/migration machinery it
required was judged unnecessary tech debt. This record reflects the codebase as it stands after
removal.

| # | Evidence |
| --- | --- |
| TN01 | `ARCHITECTURE.md` Directory Structure no longer lists `cmd/arc/ctrl/upgrade.go`. A **Compatibility Policy** section (replacing the earlier Compatibility Record) states the current, load-bearing position: breaking changes to the built-in vocabulary are accepted outright, with no migration path, because the tool is pre-1.0. |
| TN02 | Glossary carries *Built-in Vocabulary* and *Merge Vocabulary* (the *Schema Upgrade* row, added for `arc upgrade`, was removed with it); *Predicate Schema Node*, *Type Schema Node*, *Merge Behavior*, *`Node`*, and *Reference Node* remain corrected — none of those five depended on the removed command. |
| TN03 | N/A — the command this verified no longer exists. |
| TN04 | No new ADR was needed for the removal either; deleting a command and its supporting port/service files is a pure subtraction along ADR 001's existing layering. |
| TN05 | N/A — `cmd/arc/ctrl/upgrade.go` and `internal/app/ctrl/service/upgrade.go` are both deleted. |
| TN06 | **Unaffected by the removal.** The E2E layer was strictly red-first for all 29 original scenarios (7 of which, US5's, are now deleted along with the code they tested); the remaining 22 were unaffected by this change. |
| TN07 | Unaffected: `github.com/fogfish/it/v2` throughout; zero `testify` references. |
| TN08 | Unaffected. |
| TN09 | N/A — the one new port this introduced, `ctrl/port.SchemaResolver`, is deleted along with everything that used it. |
| TN10 | N/A — `bios.Registry[kernel.UpgradeResult]` no longer exists. Every remaining command's TTY/`NO_COLOR`/`--quiet`/`--verbose` behaviour is unaffected. |
| TN11 | Unaffected: no configuration value, environment variable, or secret was ever introduced, by the removed command or otherwise. |
| TN12 | N/A — the command whose help text this verified no longer exists. |
| TN13 | The 22 surviving scenarios remain green; US5's 7 are deleted along with `cmd/arc/ctrl/upgrade_test.go`. |
| TN14 | All 22 remaining spec.md scenarios map 1:1 to a passing, colocated E2E test — verified mechanically after removal. `TestApplySchemaTwiceLeavesDescriptionByteIdentical` (US1 scenario 5) is unaffected by the removal and still stands. |
| TN15 | **Still a breaking change, now with no migration path at all.** Deleting `MergeValidatedOverwrite` breaks every graph seeded by a previous release; there is no `arc upgrade` or any other remedy. This is accepted per `ARCHITECTURE.md`'s Compatibility Policy — `arc` is pre-1.0 and offers no compatibility guarantee yet, so no version-bump signal beyond the ordinary release note is required. |

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — must run first, before any table edit
- **Design Preconditions (Phase 2)**: Depends on Phase 1 — BLOCKS all stories; 2a–2e parallel with each other
- **Foundational (Phase 2.5)**: Depends on Phase 2
- **US1 (Phase 3), US2-seed (Phase 4), US3 (Phase 5), US4 (Phase 6)**: Depend on Phase 2.5; otherwise independent of each other
- **Phase 7**: Removed post-implementation (was US5, `arc upgrade`) — no longer a dependency of anything
- **Phase 8**: Depends only on Phase 4 (T024 must have re-homed `scoreZ`/`scoreC` before the constant is deleted)
- **Polish (Phase 9)**: Depends on all desired stories
- **Phase N**: Final gate

### The one cross-story dependency

```
Phase 4 (T024: scoreZ/scoreC → lastWriteWin)
        │
        └──► Phase 8 (delete MergeValidatedOverwrite, tighten the gate)
```

The `Phase 4 → Phase 7 → Phase 8` chain this section originally described no longer applies —
Phase 7 is removed, and Phase 8 depends directly on Phase 4.

Every other story pair is independent.

### File-contention note

`internal/app/schema/kernel/schema.go` is edited by Phases 3, 4, 5, and 6. Tasks within a phase
that touch it are **not** marked [P] relative to each other; tasks in different phases must be
sequenced or rebased. This is the file `CORE-FIX.md` §2 warns against editing concurrently.

### Within each user story

- E2E tests (Phase 2d) already red before implementation starts
- Vocabulary table edits → golden regeneration → unit tests
- Golden diff reviewed as a content change, never regenerated reflexively

---

## Parallel Opportunities

- **Phase 1**: T003 after T002
- **Phase 2**: T009–T012 parallel — different test files, all red (T013, the fifth, is removed along with Phase 7)
- **Phase 2.5**: T016 stands alone (T015/T017, upgrade-specific, are removed along with Phase 7)
- **Phase 6**: T041, T042 parallel with each other (distinct map entries, no overlap with T040's new keys)
- **Phase 9**: T059–T062 all parallel

```bash
# Phase 2d — all five red-phase test suites at once:
Task: "Write 6 E2E tests for US1 in cmd/arc/graph/apply_test.go"
Task: "Write 6 E2E tests for US2 in cmd/arc/ctrl/init_test.go"
Task: "Write 5 E2E tests for US3 in cmd/arc/lint/lint_test.go"
Task: "Write 5 E2E tests for US4 in cmd/arc/graph/apply_test.go + lint_test.go"
```

---

## Implementation Strategy

### MVP (US1 only)

Phase 1 → Phase 2 → Phase 2.5 → Phase 3 → validate. Delivers the prose-drift fix for all four
first-fixed predicates. Six E2E tests, six task-level changes, no breaking change, no new command.

### Recommended increment

Phases 1–6 (US1 through US4). Every correction that benefits **new** graphs, with no breaking
change — the whole feature except the gate. Shippable as one PR; `arc init` output becomes
v0.11-conformant and `arc lint` stops reporting false positives.

### Full delivery

Add Phase 8. There is no Phase 7 to sequence it after any more — the boundary it crosses is "no
existing graph is affected" to "every existing graph seeded by a previous release fails to load,
with no remedy provided," which is why it stays its own reviewable step.

### Task count

| Phase | Tasks | Story |
| --- | --- | --- |
| 1 Setup | 3 | — |
| 2 Design Preconditions | 7 | 2d maps to the four surviving stories (4 tasks removed with Phase 7) |
| 2.5 Foundational | 1 | — (2 tasks removed with Phase 7) |
| 3 | 6 | US1 |
| 4 | 8 | US2 (seed half) |
| 5 | 8 | US3 |
| 6 | 5 | US4 |
| 7 | 0 (removed) | was US5 |
| 8 | 6 | US2 (gate half) |
| 9 Polish | 6 | — |
| N Compliance | 15 | — |
| **Total** | **65** | (79 originally, 14 removed with Phase 7 and its dependents) |
