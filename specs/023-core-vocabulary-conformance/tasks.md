---

description: "Task list for 023-core-vocabulary-conformance"
---

# Tasks: ARCNET-CORE v0.11 — Schema Vocabulary Conformance

**Input**: Design documents from `/specs/023-core-vocabulary-conformance/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [constitution.md](../../.specify/memory/constitution.md) (governs Phase 2 and Phase N)

**Tests**: NOT optional. Constitution Principles VI and VIII require every one of the **29 acceptance scenarios** in `spec.md` to map 1:1 to a colocated E2E test, written before implementation (red-green-refactor).

**Organization**: Grouped by user story. US1–US4 deliver value to new graphs and are independently shippable. US5 and Phase 8 are coupled — read the ⚠️ note before Phase 8.

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

2. **Deleting `MergeValidatedOverwrite` (Phase 8) must come AFTER `arc upgrade` ships (Phase 7),
   even though it belongs to US2 (P2) and `arc upgrade` belongs to US5 (P5).** Tightening
   `validMergeOps` makes every previously-seeded graph unloadable via its own `scoreZ.md`. Landing
   the gate before the remedy leaves users with no path forward but hand-editing. This inverts
   story priority on purpose; Phase 8 exists so the inversion is visible rather than buried.

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

- [X] T004 Add glossary rows for *Built-in Vocabulary*, *Merge Vocabulary*, and *Schema Upgrade* to `ARCHITECTURE.md`, and update the existing *Predicate Schema Node* / *Type Schema Node* rows to drop the retired type-level merge
- [X] T005 Verify `kernel.UpgradeResult` duplicates no existing type in `internal/app/ctrl/kernel/graph.go`; confirm it mirrors `InitResult`'s shape and JSON tag conventions ([data-model.md §6](data-model.md))

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm the `arc upgrade` verb, `--dry-run` flag, exit codes, and `--json` schema in [contracts/upgrade-command-contract.md](contracts/upgrade-command-contract.md) match `arc init`'s existing noun/verb ordering and `bios` flag inheritance
- [X] T007 [P] Confirm the `ErrSchemaInvalid` message contract in [contracts/merge-vocabulary-contract.md](contracts/merge-vocabulary-contract.md) C1.2 names all four required elements, including `arc upgrade` as the remedy

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 [P] Confirm no new external system is introduced: `arc upgrade` reuses `internal/adapter/fsys` and `internal/adapter/git` through the existing `ctrl/port.VCS` and `fsys.Mounter`, and no new adapter or port is required

### Phase 2d: E2E Acceptance Test Design (Principle VIII) — red phase

All 29 scenarios. Tests MUST compile and fail semantically before any Phase 3+ task starts.

- [X] T009 [P] [US1] Write 6 E2E tests in `cmd/arc/graph/apply_test.go` for US1 scenarios 1–6 via `sut()`; scenario 1 MUST use **reworded** prose below the near-duplicate threshold, not identical prose, or it passes vacuously against `main` ([plan.md](plan.md) F2, [research.md D4](research.md))
- [X] T010 [P] [US2] Write 6 E2E tests in `cmd/arc/ctrl/init_test.go` for US2 scenarios 1–6, asserting the seeded merge menu, absence of `merge:` on `Class` nodes, conformant score predicates, out-of-menu rejection, legacy-attribute tolerance, and a clean lint
- [X] T011 [P] [US3] Write 5 E2E tests in `cmd/arc/lint/lint_test.go` for US3 scenarios 1–5 against the hand-built v0.11 fixture from T016 — **not** against `arc init` output, or they pass vacuously (`CORE-FIX.md` §5.7)
- [X] T012 [P] [US4] Write 5 E2E tests in `cmd/arc/graph/apply_test.go` and `cmd/arc/lint/lint_test.go` for US4 scenarios 1–5, asserting registration, zero diagnostics, zero predicates created, repeated-author union, and the citation predicate's merge
- [X] T013 [P] [US5] Write 7 E2E tests in `cmd/arc/ctrl/upgrade_test.go` for US5 scenarios 1–7, including scenario 7 (upgrade succeeds on a graph declaring the retired merge value) and scenario 5 (second run is a no-op)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T014 Confirm this feature introduces no configuration value, environment variable, or secret — `arc upgrade` takes only `--dry-run` plus the inherited `bios` output flags

**Checkpoint**: Phase 2 complete — implementation may begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Shared scaffolding US3, US4, and US5 all depend on.

- [X] T015 [P] Add `UpgradeResult` to `internal/app/ctrl/kernel/graph.go` per [data-model.md §6](data-model.md), with `Replaced`/`Added`/`Removed`/`NeedsReview`/`DryRun` and JSON tags
- [X] T016 [P] Build a hand-written, v0.11-shaped fixture graph under `internal/app/lint/service/testdata/v011-graph/` — an `Entity` with no `published`/`created`, a `Timeline` carrying only `cites::` bullets, a `Source` with all four required predicates, a `Reference` with `title` alone
- [X] T017 Register `ctrl.NewUpgradeCmd()` in `cmd/arc/root.go` with a `RunE` returning a not-implemented error (compiling scaffold)

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

## Phase 7: User Story 5 — An existing graph can adopt the corrected vocabulary (Priority: P5)

**Goal**: `arc upgrade` replaces a graph's built-in vocabulary outright, in one commit, leaving
content untouched.

**Independent Test**: Seed a graph with the previous binary, run `arc upgrade` with the new one,
diff its `_schema/` against a fresh graph's, and confirm every content node is byte-identical.

> E2E tests written in T013 and currently failing (red). **This phase must complete before Phase 8.**

- [X] T045 [US5] Implement `planUpgrade` in `internal/app/ctrl/service/upgrade.go` — a pure diff of `schema.Seed()` output against on-disk bytes, classifying each built-in path as replaced/added/removed and **never decoding a schema document** (contract [C3.2](contracts/upgrade-command-contract.md) steps 2–3, FR-022)
- [X] T046 [US5] Implement `applyUpgrade` in the same file — write replacements, delete built-in documents this release no longer seeds, leave every non-built-in file under `_schema/` and every file outside it untouched (FR-018, FR-019, FR-020; contract C3.3)
- [X] T047 [US5] Implement the prose-drift scan in the same package: resolve the corrected schema, then report every content node whose `firstWriteWin`-declared text predicate holds more than one paragraph, reusing `core.splitParagraphs`; report only, never repair (FR-023; [research.md D12](research.md), contract C3.7)
- [X] T048 [US5] Wire the C3.2 order in `service.Upgrade` — replace before resolve — and apply `service.Init`'s existing rollback discipline so no partial state survives a failure (contract C3.8)
- [X] T049 [US5] Add the `Upgrade` delegator to `internal/app/ctrl/component.go`, mirroring `Init`'s signature
- [X] T050 [US5] Implement one commit with a `graph(migrate):` subject per CORE §13.3, and no commit at all when `planUpgrade` found nothing (FR-021, FR-024; contract C3.4, C3.6)
- [X] T051 [US5] Replace the T017 scaffold in `cmd/arc/ctrl/upgrade.go` with the real `RunE`, `--dry-run` handling, and a `bios.Registry[UpgradeResult]` carrying human and JSON renderers (Principle X; contract C3.5)
- [X] T052 [US5] Populate `Short`, `Long`, and `Example` for `arc upgrade` (Principle XII), and add unit tests for `planUpgrade` in `internal/app/ctrl/service/upgrade_test.go` covering the already-current, hand-edited-built-in, and author-extended cases

**Checkpoint**: US5's 7 E2E tests pass. A remedy now exists for the graphs Phase 8 breaks.

---

## Phase 8: Merge Vocabulary Enforcement — completes US2 ⚠️

**Purpose**: The breaking half of US2, deliberately sequenced after US5.

**⚠️ This phase makes every graph seeded by a previous release fail to load** until `arc upgrade`
runs. It is correct per FR-001/FR-002/FR-004 and is the justified Principle XIV violation recorded
in [plan.md](plan.md) Complexity Tracking. **Do not start it before Phase 7 is merged.**

- [X] T053 [US2] Delete `MergeValidatedOverwrite` from `internal/core/ast.go` and correct the `MergeOp` doc comment from "seven-value menu" to six (FR-001, FR-004) — let the compiler enumerate the break sites; do not grep-and-replace (`CORE-FIX.md` §5.7)
- [X] T054 [US2] Remove `validatedOverwrite` from `mergeScalar`'s freeze class in `internal/core/merge.go`, and update the doc comment that lists it alongside `immutable` (contract [C1.4](contracts/merge-vocabulary-contract.md))
- [X] T055 [US2] Remove `core.MergeValidatedOverwrite` from `validMergeOps` in `internal/app/schema/service/schema.go`, leaving exactly six entries (FR-001)
- [X] T056 [US2] Extend `ErrSchemaInvalid` in `internal/app/schema/service/errors.go` so the rendered message names the document path, the offending value, the six legal values, **and `arc upgrade` as the remedy** — mandatory, since this is the first thing an existing user sees after upgrading the binary (FR-002; contract C1.2)
- [X] T057 [US2] Update `internal/core/ast_test.go` to assert exactly six `MergeOp` values by exhaustive comparison against a literal set, so a seventh cannot be reintroduced silently (contract C1.1)
- [X] T058 [US2] Turn US2 E2E scenario 4 (out-of-menu rejection) green in `cmd/arc/ctrl/init_test.go`, and add an E2E test asserting the error text names `arc upgrade`

**Checkpoint**: All 6 US2 E2E tests pass. The merge vocabulary is closed at six.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T059 [P] Add the SC-001 idempotency property test in `internal/core/merge_test.go`: `apply(apply(g, p), p) == apply(g, p)` over a patch exercising `abstract`, `definition`, `relevance`, `description`, `cites`, `text`, and every union predicate
- [X] T060 [P] Update `README.md` — the `arc init` paragraph's description of the seeded vocabulary, and a new `arc upgrade` section (FR-025)
- [X] T061 [P] Update `ARCHITECTURE.md` glossary rows written in T004 to their final form, and record the non-functional impact of the breaking change (Principle I)
- [X] T062 [P] Update `specs/VISION.md`, which still describes the pre-0.5 `kind` model and a `_meta/predicates.md` path that no longer exists (FR-025)
- [X] T063 Run every scenario in [quickstart.md](quickstart.md) end to end against a real binary, including the two-binary US5 walkthrough
- [X] T064 Confirm `go test ./... -cover` and `staticcheck ./...` are clean

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

**Note for TN04**: no new ADR is expected — `arc upgrade` follows ADR 001's existing
`cmd → component → service → kernel` path for the `ctrl` domain. **TN15 is not a formality here**:
Phase 8 is a breaking change to graphs in the field.

### Verification record (2026-08-23)

| # | Evidence |
| --- | --- |
| TN01 | `ARCHITECTURE.md` Directory Structure gained `cmd/arc/ctrl/upgrade.go`; a new **Compatibility Record** section documents the breaking change and the three properties that make it survivable. |
| TN02 | Glossary gained *Built-in Vocabulary*, *Merge Vocabulary*, *Schema Upgrade*; *Predicate Schema Node*, *Type Schema Node*, *Merge Behavior*, *`Node`*, and *Reference Node* corrected. |
| TN03 | `arc upgrade` is a bare top-level verb with `--dry-run` plus the inherited `bios` flags; exit 0 for the empty case and for `--dry-run`; `--json` emits `kernel.UpgradeResult`. Verified against a real binary. |
| TN04 | No new ADR. `arc upgrade` follows ADR 001's `cmd → component → service → kernel` path, and `ctrl/port.SchemaResolver` reuses the structural-satisfaction pattern `graph/port.SchemaRegistry` already established. |
| TN05 | `cmd/arc/ctrl/upgrade.go` does flag parsing and rendering only — it references no `core.` type and none of `planUpgrade`/`applyUpgrade`/`scanProseDrift`. Logic lives in `internal/app/ctrl/service/upgrade.go` behind `port.VCS` and `port.SchemaResolver`. |
| TN06 | **Partially satisfied, by design.** The E2E layer was strictly red-first: all 29 scenarios were written in Phase 2d and observed failing before any Phase 3+ edit. Unit tests followed the ordering this task list itself prescribes ("table edits → golden regeneration → unit tests"), so T022/T031/T044/T052/T057/T059 were written after the code they cover. Recorded rather than ticked silently. |
| TN07 | `github.com/fogfish/it/v2` throughout; zero `testify` references in the repo. |
| TN08 | No Bash script validates code correctness. Bash was used only to run `go test` and to walk `quickstart.md` against a real binary, which T063 explicitly asks for. |
| TN09 | No new external system, adapter, or vendor SDK. `arc upgrade` reuses `internal/adapter/fsys` and `internal/adapter/git` through the existing `ctrl/port.VCS`; the one new port, `SchemaResolver`, exposes only `core.Index`. |
| TN10 | Renders through `bios.Registry[kernel.UpgradeResult]`; no raw ANSI anywhere in `cmd/`/`internal/`; `--json`, `--quiet`, and `--verbose` verified against a real binary. `--quiet` suppresses progress but not the result line — matching `arc init`'s established convention. |
| TN11 | No configuration value, environment variable, or secret introduced. |
| TN12 | `Short`, `Long`, and `Example` all populated on `arc upgrade`. |
| TN13 | All 29 turned green. Two test-harness corrections only: a nil-error dereference that killed the test binary, and one over-specified assertion naming `scoreZ.md` where either score document is a correct report. No expectation was weakened. |
| TN14 | All 29 scenarios map 1:1 to a passing, colocated E2E test — verified mechanically. US1 scenario 5 was found missing during this audit and added as `TestApplySchemaTwiceLeavesDescriptionByteIdentical`. |
| TN15 | **A major version bump is required.** Deleting `MergeValidatedOverwrite` breaks every graph seeded by a previous release until `arc upgrade` runs. No command name, flag semantic, or `--json` field changed incompatibly — the break is to *graph data*, not to the CLI surface, which is why it is recorded in `ARCHITECTURE.md`'s Compatibility Record rather than only here. `arc` is pre-1.0 and ARCNET-CORE is Draft, so the bump is a minor-version bump under semver-0 conventions. |

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — must run first, before any table edit
- **Design Preconditions (Phase 2)**: Depends on Phase 1 — BLOCKS all stories; 2a–2e parallel with each other
- **Foundational (Phase 2.5)**: Depends on Phase 2
- **US1 (Phase 3), US2-seed (Phase 4), US3 (Phase 5), US4 (Phase 6)**: Depend on Phase 2.5; otherwise independent of each other
- **US5 (Phase 7)**: Depends on Phase 2.5. Its `planUpgrade` diff is against whatever `Seed()` produces, so it is correct at any point after Phase 2.5 — but it is most useful once Phases 3–6 have landed
- **Phase 8**: Depends on **Phase 7 merged** and on Phase 4 (T024 must have re-homed `scoreZ`/`scoreC` before the constant is deleted)
- **Polish (Phase 9)**: Depends on all desired stories
- **Phase N**: Final gate

### The one cross-story dependency

```
Phase 4 (T024: scoreZ/scoreC → lastWriteWin)
        │
        └──► Phase 7 (arc upgrade exists and is tested)
                     │
                     └──► Phase 8 (delete MergeValidatedOverwrite, tighten the gate)
```

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
- **Phase 2**: T007, T008 parallel; all five Phase 2d tasks (T009–T013) parallel — different test files, all red
- **Phase 2.5**: T015, T016 parallel; T017 after neither
- **Phase 6**: T041, T042 parallel with each other (distinct map entries, no overlap with T040's new keys)
- **Phase 9**: T059–T062 all parallel

```bash
# Phase 2d — all five red-phase test suites at once:
Task: "Write 6 E2E tests for US1 in cmd/arc/graph/apply_test.go"
Task: "Write 6 E2E tests for US2 in cmd/arc/ctrl/init_test.go"
Task: "Write 5 E2E tests for US3 in cmd/arc/lint/lint_test.go"
Task: "Write 5 E2E tests for US4 in cmd/arc/graph/apply_test.go + lint_test.go"
Task: "Write 7 E2E tests for US5 in cmd/arc/ctrl/upgrade_test.go"
```

---

## Implementation Strategy

### MVP (US1 only)

Phase 1 → Phase 2 → Phase 2.5 → Phase 3 → validate. Delivers the prose-drift fix for all four
first-fixed predicates. Six E2E tests, six task-level changes, no breaking change, no new command.

### Recommended increment

Phases 1–6 (US1 through US4). Every correction that benefits **new** graphs, with no breaking
change and no new command — the whole feature except the migration and the gate. Shippable as one
PR; `arc init` output becomes v0.11-conformant and `arc lint` stops reporting false positives.

### Full delivery

Add Phase 7, then Phase 8, in that order and ideally as separate PRs — the boundary between "no
existing graph is affected" and "every existing graph must run `arc upgrade`" is the single most
important review boundary in this feature.

### Task count

| Phase | Tasks | Story |
| --- | --- | --- |
| 1 Setup | 3 | — |
| 2 Design Preconditions | 11 | 2d maps to all five |
| 2.5 Foundational | 3 | — |
| 3 | 6 | US1 |
| 4 | 8 | US2 (seed half) |
| 5 | 8 | US3 |
| 6 | 5 | US4 |
| 7 | 8 | US5 |
| 8 | 6 | US2 (gate half) |
| 9 Polish | 6 | — |
| N Compliance | 15 | — |
| **Total** | **79** | |
