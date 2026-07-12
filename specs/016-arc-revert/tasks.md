---

description: "Task list for arc revert (specs/016-arc-revert)"
---

# Tasks: Retract a Patch's Contribution from the Graph (`arc revert`)

**Input**: Design documents from `/specs/016-arc-revert/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Unit and E2E acceptance tests are NOT optional for this project (constitution Principles VI, VIII) — every spec.md acceptance scenario maps 1:1 to an E2E test in `cmd/arc/graph/revert_test.go`, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (spec.md: US1 P1, US2 P2, US3 P3) to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- `cmd/arc/graph/revert.go` (+`revert_test.go`) — Cobra command, colocated E2E tests
- `internal/app/graph/{component.go,port,kernel,service,adapter/mock}` — graph domain (existing package, this feature extends it)
- `internal/adapter/git/git.go` (+`git_test.go`) — shared git adapter
- `internal/bios/confirm.go` (+`confirm_test.go`) — new shared UX primitive
- `testdata/` — E2E fixtures colocated with `cmd/arc/graph/revert_test.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the existing build is clean and stub the new files this feature adds — no new module dependency (plan.md Technical Context).

- [ ] T001 Confirm `go build ./...` and `go test ./...` pass on `016-arc-revert` before any change (baseline)
- [ ] T002 [P] Create empty file skeletons with the CLAUDE.md-mandated license header only: `internal/app/graph/kernel/revert.go`, `internal/app/graph/service/revert.go`, `cmd/arc/graph/revert.go`, `internal/bios/confirm.go`

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (Compliance Checklist) — a design gate, not implementation code, except where Go's own type/interface declarations *are* the design artifact (ports).

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T003 Add "Ingest Commit", "Exclusively-Owned Node", "Shared Node", "Reconciliation Approach" to `ARCHITECTURE.md`'s Glossary section (data-model.md's Domain Entities)
- [ ] T004 Confirm none of `kernel.RevertResult`/`kernel.NodeOutcome`/`port.BlameLine` (data-model.md) duplicates an existing `internal/app/graph` type — record the check; none found in research.md/data-model.md's own review

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [ ] T005 Confirm `arc revert <source-id> [--force|-f]`'s noun-verb ordering and single positional "subject" argument matches `arc apply <patch.md>`'s existing precedent (plan.md Constitution Check)
- [ ] T006 [P] Confirm contracts/revert-algorithm-contract.md's top-level decision pseudocode and contracts/vcs-port-contract.md's git-command table are complete design references for T016-T031 below (no separate flag-table doc needed — single boolean flag)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [ ] T007 [P] Confirm `internal/adapter/git.VCS` (already shared across `ctrl`/`graph`/`lint` ports) is the correct adapter to extend — no new git client (research.md D1/D11)
- [ ] T008 Add the six new method signatures (`CommitsMatching`, `ChangedPaths`, `CommitsTouching`, `RevertCommit`, `Blame`, `ShowFile`) plus the `BlameLine` type to the `VCS` interface in `internal/app/graph/port/vcs.go` (contracts/vcs-port-contract.md)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T009 [P] [US1] Write E2E tests in `cmd/arc/graph/revert_test.go` for US1's 3 acceptance scenarios (whole-commit revert of the just-applied patch: nodes removed/restored, exactly one commit, summary reports what was removed), via `sut()`; MUST compile and fail semantically (red phase)
- [ ] T010 [P] [US2] Write E2E tests in `cmd/arc/graph/revert_test.go` for US2's 2 acceptance scenarios (older, non-overlapping patch reverts cleanly; unrelated later patches untouched) (red phase)
- [ ] T011 [P] [US3] Write E2E tests in `cmd/arc/graph/revert_test.go` for US3's 4 acceptance scenarios (exclusive-node removal + backlink sweep; shared-node text-only stripping; conflict-marker provenance; mixed exclusive/shared nodes in one revert) (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [ ] T012 Confirm N/A: `arc revert` introduces no new config file, environment variable, or secret — only the `--force`/`-f` flag (no precedence chain to verify)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Command Boilerplate

**Purpose**: Empty-but-compiling scaffolding so Phase 2d's E2E tests (T009-T011) compile and fail semantically (red), not fail to build.

- [ ] T013 [P] Register `arc revert` in `cmd/arc/graph/revert.go` with a `RunE` returning a "not implemented" error, wired into the root command tree
- [ ] T014 [P] Scaffold `RevertResult`/`NodeOutcome` struct shapes (data-model.md) in `internal/app/graph/kernel/revert.go`
- [ ] T015 [P] Scaffold the six new `port.VCS` methods (T008) as not-implemented stubs on `internal/adapter/git.VCS` (`internal/adapter/git/git.go`) and as configurable fake methods on `internal/app/graph/adapter/mock.VCS` (`internal/app/graph/adapter/mock/mock.go`, research.md D11)
- [ ] T016 Wire a stub `component.Revert` delegator (`internal/app/graph/component.go`) calling a stub `service.Revert` that returns "not implemented"

**Checkpoint**: Foundation ready — T009-T011's E2E tests now compile and fail semantically (red)

---

## Phase 3: User Story 1 - Undo the patch just applied (Priority: P1) 🎯 MVP

**Goal**: The whole-commit revert path (research.md D3/D4) — reverting the patch that was just applied, with nothing since to reconcile against.

**Independent Test**: Apply one patch, immediately revert it by its identifier, confirm the graph is byte-for-byte identical to its pre-apply state aside from the revert's own new commit.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [ ] T017 [US1] Implement `CommitsMatching`, `ChangedPaths`, `CommitsTouching`, `RevertCommit` on `internal/adapter/git.VCS` in `internal/adapter/git/git.go` (contracts/vcs-port-contract.md's git-command mapping) plus `ErrGitDiffTree`/`ErrGitRevert` error sentinels
- [ ] T018 [P] [US1] Add `internal/adapter/git/git_test.go` cases for the four new methods (success + failure exit codes)
- [ ] T019 [US1] Implement `internal/bios.Confirm(prompt string) (bool, error)` in `internal/bios/confirm.go` (research.md D10: TTY-gated, refuses when non-interactive without `--force`) plus `internal/bios/confirm_test.go`
- [ ] T020 [US1] Implement `service.Revert`'s D1 (locate ingest commit via `CommitsMatching`)/D2 (idempotency via `IsTracked`)/D3 (per-path eligibility loop)/D4 (`RevertCommit` call) logic in `internal/app/graph/service/revert.go`, per contracts/revert-algorithm-contract.md's top-level decision pseudocode
- [ ] T021 [US1] Implement `cmd/arc/graph/revert.go`'s real `RunE`: mount graph, resolve `--force`/`-f`, call `bios.Confirm` unless `--force`, call `appgraph.Revert`, render via a new `revertRenderers` (`bios.Registry[kernel.RevertResult]`, mirrors `applyRenderers` in `apply.go`)
- [ ] T022 [US1] Populate `Short`/`Long`/`Example` help text for `arc revert` in `cmd/arc/graph/revert.go` (Principle XII)
- [ ] T023 [P] [US1] Add unit tests for `service.Revert`'s D1-D4 branches in `internal/app/graph/service/revert_test.go`, using the widened mock `VCS`

**Checkpoint**: At this point, User Story 1's E2E tests (T009) pass and `arc revert` works end-to-end for the simple case

---

## Phase 4: User Story 2 - Retract an old patch that nothing has touched since (Priority: P2)

**Goal**: Confirm D3's eligibility test generalizes beyond "is this literally HEAD" — no new git primitive; this story is a correctness/regression proof over T020's existing per-path loop.

**Independent Test**: Apply patch A, apply an unrelated patch B (no shared files), revert A, confirm B's contribution is untouched and A still takes the whole-commit path.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red).

- [ ] T024 [P] [US2] Add table-driven cases to `internal/app/graph/service/revert_test.go` for the "not HEAD but nothing touched since" eligibility branch (D3), asserting the same whole-commit outcome as the literal-HEAD case
- [ ] T025 [US2] Add an E2E fixture pair (two independent patches) under `cmd/arc/graph/testdata/` and the corresponding assertion in `cmd/arc/graph/revert_test.go` that the unrelated patch's files are byte-for-byte unchanged after the revert

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Retract a patch whose nodes were later enriched by other patches (Priority: P3)

**Goal**: Per-node reconciliation (research.md D5-D9) — the feature's crux: exclusive-node removal with a graph-wide backlink sweep (including timeline entries, for free per D6's `cites`-edge discovery), shared-node text-only stripping via `git blame`, and conflict-marker provenance for the case blame alone cannot attribute.

**Independent Test**: Apply patch D1 creating node A, apply patch D2 adding further content to node A, revert D1, confirm A still exists, still carries everything D2 contributed, and carries nothing that only D1 contributed.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red).

- [ ] T026 [US3] Implement `Blame` and `ShowFile` on `internal/adapter/git.VCS` in `internal/adapter/git/git.go` (contracts/vcs-port-contract.md) plus `ErrGitBlame`/`ErrGitShow` error sentinels
- [ ] T027 [P] [US3] Add `internal/adapter/git/git_test.go` cases for `Blame`/`ShowFile` (including `ShowFile`'s "path absent at this commit" non-error case)
- [ ] T028 [US3] Implement the per-node exclusivity test (D5, reusing `CommitsTouching`) and `removeNode` (D6: `store.Remove`, reuse `enumerateNodes`/`buildReverseIndex` from `internal/app/graph/service/subgraph.go`, filter referrers' `Edges`, rewrite via `core.RenderNode`) in `internal/app/graph/service/revert.go`
- [ ] T029 [US3] Implement `removeTimelineEntry` — a structural sibling to `upsertTimelinePeriod` — in `internal/app/graph/service/apply.go` (reuses `parseTimelineEntries`/`periodGranularity`), wired into T028's referrer-rewrite branch for `@type: timeline` referrers
- [ ] T030 [US3] Implement `reconcileShared`'s Texts-key/paragraph blame-mapping (D7) in `internal/app/graph/service/revert.go`: walk `renderNodeBody`'s physical order (leading key, other keys alphabetically, trailing key) to build the line→(Texts key, paragraph index) map, intersect with `Blame`'s `ingestHash`-attributed lines, strip matched paragraphs (mirroring `internal/core/merge.go`'s `splitParagraphs`), rewrite via `core.RenderNode` — never touching `Attrs`/`Edges`/`HRefs` (FR-011)
- [ ] T031 [US3] Implement conflict-marker provenance resolution (D8) in `internal/app/graph/service/revert.go`: marker-shape detection before D7's blame path is attempted; D8(a) plain-text `sourceID` match against the marker's trailing token; D8(b) `CommitsTouching` + `ShowFile` + `core.ParseNode` oldest-first historical walk to find the predicate's true first writer
- [ ] T032 [US3] Implement the D9 no-attribution no-op case (`NodeOutcome.Kind = "unchanged"`, no write, not an error) in `internal/app/graph/service/revert.go`
- [ ] T033 [US3] Wire the per-node path into `service.Revert`'s top-level dispatch (branch on D3's eligibility result) and the per-node commit message (`Reverted-Document:` trailer — never `Source-Id:`, data-model.md's collision-avoidance rationale) in `internal/app/graph/service/revert.go`
- [ ] T034 [P] [US3] Add unit tests for D5-D9 in `internal/app/graph/service/revert_test.go`: exclusive removal, backlink sweep over an ordinary referrer and a timeline-period referrer, paragraph stripping, D8(a) and D8(b) conflict-marker cases, D9 no-op case
- [ ] T035 [US3] Wire `--verbose` per-node `Reporter.Step` output (`<path>: removed (...)` / `reconciled (...)` / `unchanged (...)`) per contracts/revert-algorithm-contract.md's Reporter events section, in `internal/app/graph/service/revert.go`

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests — full spec.md coverage

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories

- [ ] T036 [P] Add E2E coverage for spec.md's remaining Edge Cases in `cmd/arc/graph/revert_test.go`: unknown `source-id` (FR-002), already-retracted no-op (FR-003/D2), target not an initialized graph (FR-004), interrupted/partial-failure leaves no trace (FR-016)
- [ ] T037 [P] Documentation: regenerate the `cobra/doc` command reference (if produced by this project) and confirm `README.md`'s command list mentions `arc revert`
- [ ] T038 Run `quickstart.md`'s seven scenarios (A-G) manually against a built `arc` binary
- [ ] T039 [P] `staticcheck ./...` clean on every new/changed file

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [ ] TN01 `ARCHITECTURE.md` reflects architectural changes, if any (Principle I) — the new `internal/bios.Confirm` primitive and the widened `internal/app/graph/port.VCS` are documented
- [ ] TN02 Domain concepts added to the `ARCHITECTURE.md` Glossary (Principle II) — T003
- [ ] TN03 Command/flag surface matches the Phase 2b design exactly: `arc revert <source-id> [--force|-f]`, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 Major decisions recorded in `adrs/` with correct numbering, if a new architectural pattern was introduced — assess whether `internal/bios.Confirm` (research.md D10, the first destructive-operation confirmation gate in this codebase) warrants its own ADR entry or is adequately covered by ADR 002's existing, already-binding CLIG checklist item (Principle I)
- [ ] TN05 Domain logic uses ports (interfaces); Cobra wiring (`cmd/arc/graph/revert.go`) and adapters (`internal/adapter/git`) remain separated from `internal/app/graph/service` (Principle III)
- [ ] TN06 Unit tests (T023, T024, T034) were written first, compiled, and failed semantically before implementation (Principle VI)
- [ ] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [ ] TN08 No Bash scripts were used for unit-level code correctness validation — `quickstart.md`'s scenarios (T038) are a manual/smoke validation only, not a substitute for `go test` (Principle VI)
- [ ] TN09 The five new git primitives extend the existing `internal/adapter/git.VCS` adapter only; no vendor SDK/`exec.Cmd`/`exec.ExitError` type leaks through `port.VCS` (Principle VII)
- [ ] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`; the new confirmation prompt (T019) and any styled revert output use `github.com/charmbracelet/lipgloss`, never raw ANSI (Principle X)
- [ ] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags — N/A confirmed at T012 (Principle XI)
- [ ] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc revert` (Principle XII) — T022
- [ ] TN13 E2E tests from Phase 2d (T009-T011) turned GREEN and changed minimally during implementation (Principle VIII)
- [ ] TN14 All spec.md scenarios for this feature (US1 x3, US2 x2, US3 x4, plus the Edge Cases in T036) have a passing, colocated E2E test (Principle VIII)
- [ ] TN15 Release/versioning impact assessed: `arc revert` is a wholly new command and `RevertResult`'s `--json` shape is its first version — no existing `--json`/`--plain` contract is broken, so no major version bump is required (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; 2a-2e can proceed in parallel with each other
- **Command Boilerplate (Phase 2.5)**: Depends on Phase 2 completion (specifically T008's port signatures)
- **User Stories (Phase 3-5)**: All depend on Phase 2.5 — US1 has no dependency on US2/US3; US2 depends only on US1's T020 eligibility loop existing (extends it, does not duplicate it); US3 depends on US1's T020 dispatch point existing (adds the per-node branch) but not on US2
- **Additional Polish**: Depends on US1-US3 being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on US2/US3
- **User Story 2 (P2)**: Can start after Phase 2.5 — extends US1's T020 eligibility loop with additional test coverage; should remain independently testable
- **User Story 3 (P3)**: Can start after Phase 2.5 — adds a new dispatch branch alongside US1's T020; independently testable via its own fixture (a shared node), no dependency on US2's fixture

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Adapter/port methods before service logic that calls them
- Service logic before the command's `RunE`
- Story complete before moving to next priority

### Parallel Opportunities

- T002's four file skeletons — parallel
- Phase 2a-2e subsections — parallel with each other
- T009-T011 (Phase 2d E2E tests, one per story) — parallel
- T013-T015 (Phase 2.5 scaffolding) — parallel
- T017/T019 (git adapter methods vs. `bios.Confirm`) — parallel, different files
- T026/T027 (US3's `Blame`/`ShowFile` + their tests) — parallel with US1/US2 work once Phase 2.5 is done, if staffed separately

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2, already complete before this point):
# T009 E2E test(s) for US1 acceptance scenarios in cmd/arc/graph/revert_test.go

# Launch independent implementation tasks for User Story 1 together:
Task: "Implement CommitsMatching/ChangedPaths/CommitsTouching/RevertCommit in internal/adapter/git/git.go"
Task: "Implement internal/bios.Confirm in internal/bios/confirm.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Command Boilerplate
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenario A/B)
7. Deploy/demo if ready — `arc revert` already handles the common "undo my last mistake" case

### Incremental Delivery

1. Setup + Design Preconditions + Command Boilerplate → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo (older-patch coverage)
4. Add User Story 3 → Verify against Phase N → Deploy/Demo (the crux: shared-node safety)
5. Each story adds value without breaking previous stories — US1's whole-commit path is untouched by US3's new per-node branch

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Command Boilerplate together
2. Once complete:
   - Developer A: User Story 1 (+ `internal/bios.Confirm`)
   - Developer B: User Story 2 (extends US1's eligibility loop with regression coverage — best done just after US1 lands)
   - Developer C: User Story 3 (`Blame`/`ShowFile`, the D5-D9 reconciliation algorithm — the largest, most independent chunk)
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements
- US3 (T026-T035) is this feature's largest chunk of genuinely new logic (research.md D5-D9) — the conflict-marker provenance walk (T031/D8) is the single highest-risk task; consider implementing and unit-testing it in isolation before wiring it into T033's dispatch
