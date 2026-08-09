---

description: "Task list for `arc apply batch` implementation"
---

# Tasks: Batch Apply a Directory of Patches (`arc apply batch`)

**Input**: Design documents from `/specs/020-apply-batch/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every one of spec.md's **16 acceptance scenarios** maps 1:1 to an E2E test, and tests are written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Exact file paths included in every task

## Path Conventions

- `cmd/arc/graph/` — Cobra wiring plus colocated `batch_test.go` E2E tests (Principles III, VIII)
- `internal/app/graph/kernel/` — use-case result types (no cobra import)
- `internal/app/graph/service/` — use-case logic plus colocated unit tests
- `cmd/arc/graph/testdata/batch/` — E2E fixtures (Principle VIII)

**No new Go module dependency is added by this feature.** `go.mod` must be
byte-identical before and after — that is a plan.md Constitution Check gate
(Principle VII), verified in T006 and TN16.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Empty-but-compiling scaffolding so Phase 2d's E2E tests compile and fail semantically ("not implemented") rather than failing to build — the constitution's sanctioned minimal structure for the red phase (Principle VIII).

- [X] T001 [P] Create `internal/app/graph/kernel/batch.go` with `Outcome`, `PatchOutcome`, and `BatchResult` exactly as specified in [data-model.md](data-model.md), including JSON tags and `omitempty` placement
- [X] T002 Create `internal/app/graph/service/batch.go` with an `ApplyBatch` signature matching [contracts/cli-contract.md § Use-case contract](contracts/cli-contract.md) whose body returns a not-implemented error
- [X] T003 Add the `ApplyBatch` delegator to `internal/app/graph/component.go`, mirroring the existing `Apply` delegator's three-line shape
- [X] T004 Create `cmd/arc/graph/batch.go` with `NewApplyBatchCmd()`: `Use: "batch <dir>"`, `cobra.ExactArgs(1)`, `SilenceUsage`/`SilenceErrors` true, the `--fail-fast` flag, and a `RunE` returning a not-implemented error
- [X] T005 Register the batch command under `applyCmd` in `cmd/arc/root.go`, beside the existing `ctrl.NewApplySchemaCmd()` registration
- [X] T006 Verify `go build ./...` succeeds, `staticcheck ./...` is clean, and `git diff --exit-code go.mod go.sum` reports no change

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T007 Add **Batch Plan**, **Patch Outcome**, and **Batch Summary** to the Glossary in [ARCHITECTURE.md](../../ARCHITECTURE.md), with definitions and their relationship to the existing Patch / Ingest Commit terms
- [X] T008 Verify no new domain type duplicates an existing one: confirm `core.Patch`, `kernel.ApplyResult`, and `fsys.Store` are reused rather than re-declared, and that nothing new is introduced inside `cmd/` (Principle II)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T009 Confirm the command surface in `cmd/arc/graph/batch.go` matches [contracts/cli-contract.md](contracts/cli-contract.md) exactly — subcommand placement under `apply`, one positional argument, `--fail-fast` long-form only (`-f` stays reserved for file/force)
- [X] T010 [P] Confirm the `--json` field names, types, and enum values emitted by `kernel.BatchResult` match [contracts/batch-result.schema.md](contracts/batch-result.schema.md), including that all slice fields serialise as `[]` and never `null`

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T011 [P] Record that no new adapter or port is required: the read side of the existing `internal/adapter/fsys.Store` (`fs.FS`, `fs.ReadDirFS`) and the existing `port.VCS` cover every external interaction (research.md D1, D11)
- [X] T012 Declare the new expected error conditions as package-level `faults.Type`/`faults.SafeN` constants in `internal/app/graph/service/batch.go` — patch directory not found, patch directory is not a directory, patch directory unreadable — never inline `fmt.Errorf` (research.md D13, Principle XII)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> All E2E tests live in `cmd/arc/graph/batch_test.go` and reuse the existing `sut()` and `sutCaptureStderr()` helpers already defined in `cmd/arc/graph/apply_test.go` (same package). Each test carries a `// Scenario X.Y from specs/020-apply-batch/spec.md` reference and the equivalent command line. Tests MUST compile and fail semantically against Phase 1's scaffold — no `t.Skip()`, no "red phase" comments.

- [X] T013 Build the fixture tree at `cmd/arc/graph/testdata/batch/` exactly as laid out in [quickstart.md § Fixture layout](quickstart.md) — nested subdirectories, a `.hidden/` directory, two non-patch `.md` files, one unparsable patch, and patches whose publication order contradicts their alphabetical order
- [X] T014 [P] [US1] Write E2E tests in `cmd/arc/graph/batch_test.go` for User Story 1's 5 acceptance scenarios: recursive discovery and application, publication-date ordering independent of filename, one commit per patch with correct subject/trailer, nested-subdirectory patches ordered globally, and the summary counts (red phase)
- [X] T015 [P] [US2] Write E2E tests for User Story 2's 4 acceptance scenarios: fully-ingested re-run is a no-op, mixed new/ingested directory applies only the new, interrupted run resumes, and two patches for the same document in one run yield one commit (red phase)
- [X] T016 [P] [US3] Write E2E tests for User Story 3's 4 acceptance scenarios: valid patches all apply despite a malformed one, failures reported by path with reason, non-zero exit, and an all-failed run producing zero commits (red phase)
- [X] T017 [P] [US4] Write E2E tests for User Story 4's 3 acceptance scenarios: `--fail-fast` halts mid-plan, unprocessed count reported with non-zero exit, and re-running after the fix resumes (red phase)
- [X] T018 [P] Write E2E tests for the spec's Edge Cases in `cmd/arc/graph/batch_test.go`: non-patch `.md` passed over, non-Markdown ignored entirely, empty directory succeeds, missing path refused, file-not-directory refused, uninitialised graph refused, same-date deterministic tie-break, hidden directories never descended, absent publication date treated as a failure, conflict flagged mid-run still counts as applied, and the patch directory left byte-identical (red phase)
- [X] T019 [P] Write the `--json` E2E test asserting against the documented schema in [contracts/batch-result.schema.md](contracts/batch-result.schema.md) — counter invariants, `not_a_patch` outside the sum, non-null arrays, and stdout carrying only JSON (red phase; Principle VIII forbids "non-empty" assertions here)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T020 Record that this feature introduces no configuration value, environment variable, or credential — the flag→env→config precedence chain and secret-handling rules are therefore not engaged (Principle XI)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The discovery → classification → ordering pipeline and the preflight guards, which every user story depends on. Pure functions over a mounted `fs.FS`, unit-testable without Cobra and without git.

- [X] T021 [P] Write unit tests in `internal/app/graph/service/batch_test.go` for discovery: recursion into subdirectories, dot-directories skipped, non-`.md` files ignored, symlinked directories not descended (red phase, research.md D3)
- [X] T022 [P] Write unit tests for classification: a file without a patch manifest counts as `not_a_patch` and never fails; a file declaring `kind: patch` that does not parse becomes a failed candidate; an absent or uninterpretable `published` date becomes a failed candidate (red phase, research.md D4, FR-020)
- [X] T023 [P] Write unit tests for ordering: publication date ascending, equal dates broken by relative path, and classification failures appended last rather than sorting to the front on their zero timestamp (red phase, research.md D5, D5b)
- [X] T024 Implement preflight guards in `internal/app/graph/service/batch.go` using T012's `faults` constants: target exists, target is a directory, working directory is an initialised graph — all evaluated before any patch is read (FR-002)
- [X] T025 Implement discovery in `internal/app/graph/service/batch.go` with `fs.WalkDir` over `fsys.Local{}.Mount(patchDir)`, returning `fs.SkipDir` for dot-directories and considering only regular `.md` files (FR-001, FR-019)
- [X] T026 Implement classification in `internal/app/graph/service/batch.go` using the existing `core.LooksLikePatch` then `core.ParsePatch`, producing the `candidate` shape from [data-model.md](data-model.md) (FR-003, FR-020)
- [X] T027 Implement ordering in `internal/app/graph/service/batch.go` — `published` ascending, `path` tie-break, failed candidates appended last — producing the `plan` fixed before any commit (FR-004, FR-005)

**Checkpoint**: T021-T023 unit tests green; the plan can be computed for any directory with zero writes performed

---

## Phase 3: User Story 1 - Ingest a whole corpus in one command (Priority: P1) 🎯 MVP

**Goal**: Point the command at a directory and have every applicable patch applied in publication order, one commit each, with a summary of what happened.

**Independent Test**: Run `arc apply batch` against the fixture tree in a fresh graph and confirm `git log --oneline` shows one commit per applied patch with the oldest publication first, regardless of filename order.

### Implementation for User Story 1

> E2E tests for this story were written in Phase 2d (T014) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T028 [US1] Implement the execution loop in `internal/app/graph/service/batch.go`: iterate the ordered plan, calling the **unchanged** `service.Apply` once per candidate with `patchPath` reconstructed as `filepath.Join(absPatchDir, filepath.FromSlash(rel))` (FR-006, FR-007, research.md D6)
- [X] T029 [US1] Project each `kernel.ApplyResult` into a `kernel.PatchOutcome` and accumulate `BatchResult`'s counters in `internal/app/graph/service/batch.go`, preserving every invariant listed in [data-model.md](data-model.md) (FR-013)
- [X] T030 [US1] Emit one per-patch progress line through the batch reporter in `internal/app/graph/service/batch.go` as each patch reaches its outcome (FR-015, research.md D9)
- [X] T031 [P] [US1] Implement the human summary renderer and `bios.Registry[kernel.BatchResult]` in `cmd/arc/graph/batch.go`, following the shape in [contracts/cli-contract.md § Output](contracts/cli-contract.md) and styling through `bios.SCHEMA` only — no raw ANSI (Principle X)
- [X] T032 [US1] Implement `RunE` in `cmd/arc/graph/batch.go`: resolve absolute graph and patch directories, build the two reporters per research.md D9 (batch progress on unless `--quiet`; per-node detail `--verbose`-gated), invoke `appgraph.ApplyBatch`, and print the rendered summary to stdout
- [X] T033 [US1] Implement the FR-022 nothing-to-apply path: succeed, make no changes, and state plainly that there was nothing to apply
- [X] T034 [P] [US1] Populate `Short`, `Long`, and `Example` for the batch command in `cmd/arc/graph/batch.go` (Principle XII)

**Checkpoint**: T014's E2E tests pass; the command delivers end-to-end value on its own

---

## Phase 4: User Story 2 - Re-run safely over a partly-ingested directory (Priority: P1)

**Goal**: Re-running over the same directory skips what is already tracked, so an interrupted or repeated run is safe and resumable.

**Independent Test**: Run the batch twice over the same directory; the second run produces zero commits, reports every patch as already tracked, and exits `0`.

### Implementation for User Story 2

> E2E tests for this story were written in Phase 2d (T015) and MUST currently be failing (red).

- [X] T035 [US2] Map `ApplyResult.Skipped` to the `skipped` outcome and its counter in `internal/app/graph/service/batch.go`, ensuring a skipped patch contributes no commit hash and no created/merged counts (FR-008)
- [X] T036 [US2] Confirm by test that the already-tracked check is evaluated when each patch is reached rather than once up front, so a second patch for the same document later in the same run is skipped (FR-009) — this is inherited from `service.Apply` and must be verified, not reimplemented
- [X] T037 [P] [US2] Render skipped counts in the human summary and confirm the `--json` `skipped` counter in `cmd/arc/graph/batch.go`

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - One bad patch does not abandon the rest (Priority: P2)

**Goal**: A malformed patch is recorded and reported without stopping the run or corrupting the graph, and the process exits non-zero.

**Independent Test**: Run the batch over a directory containing one unparsable patch among valid ones; every valid patch commits, the bad one is named with a reason, `git status` is clean, and the exit code is `1`.

### Implementation for User Story 3

> E2E tests for this story were written in Phase 2d (T016) and MUST currently be failing (red).

- [X] T038 [US3] Implement continue-on-error accumulation in `internal/app/graph/service/batch.go`: a failing patch records the `failed` outcome plus a human-readable reason and the loop proceeds to the next candidate (FR-010)
- [X] T039 [US3] Confirm by test that a failed patch leaves no partial node file and no dangling commit, and that commits produced earlier in the same run remain intact (FR-012) — inherited from `service.Apply`'s existing rollback, verified not reimplemented
- [X] T040 [P] [US3] Render the failure block — one line per failure with path and reason — in the human summary in `cmd/arc/graph/batch.go` (FR-013)
- [X] T041 [US3] Return `bios.ErrSilent` from `RunE` in `cmd/arc/graph/batch.go` when `BatchResult.Failed > 0`, after the summary has already printed, so the process exits `1` with no second error line (FR-014, research.md D8)

**Checkpoint**: User Stories 1-3 pass independently; a mixed-quality corpus imports cleanly with an honest exit code

---

## Phase 6: User Story 4 - Halt the batch at the first failure (Priority: P3)

**Goal**: `--fail-fast` stops the run at the first failing patch, leaving the remainder unprocessed and counted.

**Independent Test**: Run with `--fail-fast` over a directory whose failing patch sits mid-order; patches before it are committed, patches after it are untouched and reported as unprocessed, exit code `1`.

### Implementation for User Story 4

> E2E tests for this story were written in Phase 2d (T017) and MUST currently be failing (red).

- [X] T042 [US4] Implement the `--fail-fast` halt in `internal/app/graph/service/batch.go`: stop the loop at the first `failed` outcome and mark every remaining planned candidate `unprocessed` (FR-011)
- [X] T043 [P] [US4] Render the unprocessed count in the human summary and confirm the `--json` `unprocessed` counter in `cmd/arc/graph/batch.go`
- [X] T044 [US4] Confirm by test that re-running after repairing the offending patch skips the already-committed documents and resumes from the repaired one (US4 scenario 3, via FR-008)

**Checkpoint**: All four user stories pass their E2E tests independently

---

## Additional Polish

**Purpose**: Cross-cutting concerns that span every user story.

- [X] T045 Aggregate every applied patch's `Conflicts` into `BatchResult.Conflicts` (de-duplicated, sorted) and surface them once in the final summary, so a conflict flagged early in a long run is not lost in scrollback (FR-016)
- [X] T046 Aggregate every applied patch's `Warnings` into `BatchResult.Warnings` (de-duplicated, first-seen order) and emit them to stderr without aborting the run (FR-017)
- [X] T047 [P] Document `arc apply batch` in [README.md](../../README.md) alongside the existing `arc apply` and `arc apply schema` entries
- [X] T048 [P] Run every scenario in [quickstart.md](quickstart.md) end-to-end against a real temporary graph and record any divergence
- [X] T049 Confirm the existing `cmd/arc/graph/apply_test.go` and schema-apply suites still pass **unmodified** — if either needed editing, the change leaked past the intended boundary ([quickstart.md § Regression guard](quickstart.md))
- [X] T050 Run `go test -coverprofile=profile.cov ./...` and `staticcheck ./...`; both clean

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

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
- [X] TN16 Filesystem I/O goes through `io/fs`-based `internal/adapter/fsys` with no `os.*` file/directory call added outside it, and **no third-party filesystem or object-storage library was introduced** — `go.mod` and `go.sum` are unchanged (Principle VII, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately. Produces the compiling scaffold Phase 2d's tests need.
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phases 2 and 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

**Phase 0 (Pre-implementation Refactoring) is omitted**: this feature restructures no existing code. `internal/app/graph/service/apply.go` is not modified at all (plan.md § Structure Decision), which is the point of the design.

### User Story Dependencies

- **User Story 1 (P1)**: Depends only on Phases 2 and 2.5 — the MVP
- **User Story 2 (P1)**: Depends on US1's execution loop (T028-T029); adds the skipped outcome on top of it
- **User Story 3 (P2)**: Depends on US1's execution loop; adds failure accumulation and the exit code
- **User Story 4 (P3)**: Depends on US3's failure accumulation (T038) — `--fail-fast` is a halt condition on a failure path that must exist first

These stories are **not** fully independent of one another: all four drive the same loop. They remain independently *testable* — each has its own E2E scenarios and its own checkpoint — but they are best implemented in priority order rather than in parallel across developers.

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Service-layer outcome accumulation before command-layer rendering
- Story complete before moving to the next priority

### Parallel Opportunities

- T001 is independent of T004; both precede T002/T003/T005
- All of Phase 2d (T014-T019) can be written in parallel — different test functions, one shared fixture tree from T013, which must land first
- Phase 2.5's three unit-test tasks (T021-T023) are parallel; the three implementations (T025-T027) are sequential within `batch.go`
- Renderer tasks (T031, T037, T040, T043) touch `cmd/arc/graph/batch.go`; only T031 is safely parallel with service work, the rest serialise against each other
- Polish tasks T047 and T048 are parallel

---

## Parallel Example: Phase 2d

```bash
# T013 must land first — every test below reads the same fixture tree.

# Then launch the acceptance-test writing tasks together:
Task: "US1 E2E tests (5 scenarios) in cmd/arc/graph/batch_test.go"
Task: "US2 E2E tests (4 scenarios) in cmd/arc/graph/batch_test.go"
Task: "US3 E2E tests (4 scenarios) in cmd/arc/graph/batch_test.go"
Task: "US4 E2E tests (3 scenarios) in cmd/arc/graph/batch_test.go"
Task: "Edge-case E2E tests in cmd/arc/graph/batch_test.go"
Task: "--json schema E2E test in cmd/arc/graph/batch_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. **STOP and VALIDATE**: bulk ingest works end-to-end, one commit per patch, publication order correct

At this point the command already delivers its core value. What is missing is
graceful behaviour around re-runs and failures, not the feature itself.

### Incremental Delivery

1. Setup + Design Preconditions + Foundational → plan computable, zero writes
2. Add User Story 1 → bulk ingest works (MVP)
3. Add User Story 2 → re-runs and resumption are safe
4. Add User Story 3 → a bad patch no longer abandons the corpus; exit code is honest
5. Add User Story 4 → `--fail-fast` for pipelines
6. Polish → conflicts/warnings aggregation, README, quickstart validation
7. Phase N verification before merge

### Suggested PR Split

- **PR 1** — Phase 1 scaffold (compiles, does nothing, registered command returns not-implemented)
- **PR 2** — Phases 2 + 2.5 (design records, red tests, discovery/classification/ordering pipeline green at unit level)
- **PR 3** — Phase 3 (US1, the MVP)
- **PR 4** — Phases 4-6 (US2, US3, US4)
- **PR 5** — Polish + Phase N

---

## Implementation notes (recorded on completion)

Three things landed differently from the letter of the task text. All three
are recorded here rather than silently absorbed.

1. **The unit-test file is `internal/app/graph/service/batch_internal_test.go`,
   not `batch_test.go`** (T021-T023). Discovery, classification, and ordering
   are unexported, so their tests must live in `package service`, and this
   package's established convention for a white-box test file is the
   `_internal_test.go` suffix (`revert_internal_test.go`). Using the bare
   name would have put a `package service` file beside `apply_test.go`'s
   `package service_test`, contradicting the repo's own signal for which is
   which.

2. **`bios.Humanize` was extracted** (`internal/bios/errors.go`, with
   `cmd/arc/main.go` reduced to a one-line delegate). Not in plan.md's file
   list. A per-patch failure reason is carried as *data* — rendered in the
   summary and serialized into the `--json` contract (FR-013, FR-018) — so it
   never passes through `cmd/arc/main.go`, the site that until now owned the
   faults `[pkg.Func 339]` prefix stripping. Without this, every `reason`
   string shipped a source location to both audiences. The rules moved to the
   shared kernel unchanged; `cmd/arc/main_test.go` still passes unmodified,
   and `internal/bios/errors_test.go` covers the extracted function directly.

3. **`BatchResult.Directory` is the resolved absolute path**, not the user's
   literal argument. The use-case contract hands `ApplyBatch` only the
   resolved `patchDir`, and echoing the raw argument would need a second
   parameter carrying nothing the absolute form does not already say
   unambiguously. [data-model.md](data-model.md) and
   [contracts/batch-result.schema.md](contracts/batch-result.schema.md) were
   corrected to match.

Two additions beyond the task list, both closing real coverage gaps:

- `TestBatchUnregisteredKindWarnsWithoutAbortingTheRun` — T046 aggregates
  warnings but no task asked for an E2E test proving FR-017's "does not abort
  the run" half.
- Unit tests for `unionSorted`/`unionFirstSeen`/`progressLine`/`record` —
  FR-016's sort-and-dedupe and FR-017's first-seen-order are distinguishable
  only by direct assertion; end-to-end they both just "show up once".

Fixture additions beyond quickstart.md's original layout, now reflected
there: `bibliography.bib` (proves a non-Markdown file appears in *no* count,
not even `not_a_patch`) and the `testdata/batch-failfast/` bundle, whose
`b-blocker.patch.md` fails during *application* rather than classification —
the only shape that lets `--fail-fast` halt mid-order with a non-zero
`unprocessed` count, per research.md D5b.

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to a specific user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Several tasks (T036, T039, T044, T049) are deliberately **verification** tasks rather than implementation: the behaviour is inherited unchanged from `service.Apply`, and re-implementing it in the batch layer would be the bug
- Phase 2 and Phase N sections are retained per constitution Governance > Task List Requirements; TN16 is an addition, not a replacement, covering this feature's specific dependency gate
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
