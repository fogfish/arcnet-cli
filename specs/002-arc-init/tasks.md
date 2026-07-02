# Tasks: Initialize a New Knowledge Graph (`arc init`)

**Input**: Design documents from `/specs/002-arc-init/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`cli-contract.md`, `vcs-port-contract.md`, `fsys-port-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/ctrl/` — Cobra command wiring for `arc init` and its colocated E2E test
- `internal/bios/` — shared output/color/reporter kernel (ADR 002 DS-04/05/06), reused by every future command
- `internal/adapter/fsys/` — shared, cross-use-case filesystem adapter (stdlib `io/fs`/`io.Writer` only, no `os.*` calls anywhere else)
- `internal/app/ctrl/{kernel,port,adapter,service}/` — the `ctrl` (graph management / control plane) domain use-case, per ADR 001's `componentX` layout
- `testdata/ctrl/` — E2E fixtures, if any golden files are needed

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create the package skeleton: `cmd/arc/ctrl/`, `internal/bios/`, `internal/adapter/fsys/`, `internal/app/ctrl/{kernel,port,adapter/git,adapter/mock,service}/` directories per plan.md's Project Structure
- [X] T002 Add `github.com/charmbracelet/lipgloss` and `github.com/fogfish/faults` to `go.mod`/`go.sum` (`go get github.com/charmbracelet/lipgloss github.com/fogfish/faults`) per [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling)
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the new (still-empty) package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the domain terms from data-model.md — Graph Root, Canonical Folder, Metadata Stub, Arc State Directory, Initial Commit — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (first-ever population of this file, Principle I obligation from plan.md Constitution Check row I)
- [X] T005 Verify no existing `internal/<domain>` package already defines a `GraphRoot`/`ArcNetCoreLayout`/`InitResult`-shaped type before introducing them in `internal/app/ctrl/kernel` (there are none yet — this is the first `internal/` package)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc init`'s bare-verb grammar (ADR 002 DS-01, research.md D6) and the DS-03 persistent flags (`--quiet`/`-q`, `--verbose`/`-v`, `--json`, `--color`/`-C`) to be added to `cmd/arc/root.go` against contracts/cli-contract.md
- [X] T007 [P] Review contracts/cli-contract.md and contracts/fsys-port-contract.md/contracts/vcs-port-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 [P] Confirm no existing adapter in the repository already covers git subprocess access or filesystem mounting before creating `internal/app/ctrl/adapter/git` and `internal/adapter/fsys` (none exist — this is the first feature to add `internal/`)
- [X] T009 Define `port.VCS` (`internal/app/ctrl/port/vcs.go`, contracts/vcs-port-contract.md) and `fsys.Store`/`fsys.Mounter` (`internal/adapter/fsys/types.go`, contracts/fsys-port-contract.md) interface shapes as the design gate before any adapter code is written

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T010 [US1] Write E2E tests in `cmd/arc/ctrl/init_test.go` for spec.md US1's 4 acceptance scenarios (canonical folders/stubs/`.arc/`/`.gitignore` created; exactly one commit `graph(init): empty knowledge graph`; clean working tree with `.arc/` untracked; canonical folders present via placeholder in git history), using the existing `sut()`/`run()` helpers from `cmd/arc/root_test.go` — tests MUST compile and fail semantically (red phase)
- [X] T011 [US1] Write additional E2E tests in `cmd/arc/ctrl/init_test.go` for the edge cases spec.md ties to US1's core creation path but that aren't formal acceptance scenarios: FR-010 (target is a file, not a directory), FR-011 (`git` unavailable on `PATH`), FR-015 (target directory non-empty and not a graph), and the `--json` output contract (contracts/cli-contract.md) — per quickstart.md Scenarios 4–6; red phase
- [X] T012 [US2] Write E2E tests in `cmd/arc/ctrl/init_test.go` for spec.md US2's 2 acceptance scenarios (creates a non-existent named `<dir>`, current directory left untouched; reports the resolved path on success) — red phase
- [X] T013 [US3] Write E2E test in `cmd/arc/ctrl/init_test.go` for spec.md US3's 1 acceptance scenario (re-running against a directory that already contains `.arc/` refuses with no changes, FR-014) — red phase

> T010–T013 all target the same new file (`cmd/arc/ctrl/init_test.go`) and are therefore sequential, not `[P]`, despite each being scoped to one story.

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T014 Confirm `arc init` introduces no configuration file, environment variable, or secret handling (plan.md Constitution Check marks Principle XI N/A for this feature) — no action beyond this confirmation

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: This is the first feature to add any `internal/` package, so nearly all of `arc init`'s actual capability — the shared output kernel, the filesystem adapter, and the `ctrl` domain scaffolding — is genuinely foundational: every one of US1/US2/US3 exercises the same `service.Init` function, differing only in which guard fires or whether `<dir>` is supplied. This phase builds that shared foundation; Phase 3+ wires story-specific behavior on top of it.

- [X] T015 [P] Implement `internal/bios/theme.go`: `Schema`, `SCHEMA_PLAIN`, `SCHEMA_COLOR`, package var `SCHEMA` per ADR 002 DS-05
- [X] T016 [P] Implement `internal/bios/output.go`: `Mode`, `ResolveMode()`, `Printer[T]`, `Registry[T]`, `jsonPrinter[T]`, `nonePrinter[T]` per ADR 002 DS-04
- [X] T017 [P] ✅ Fixed (BUG-001) Implement `internal/bios/reporter.go`: `Reporter` interface, `stderrReporter`, `silentReporter`, `newReporter(quiet, silent bool) Reporter` per ADR 002 DS-06 — original implementation passed a newline-embedded string into `lipgloss.Style.Render(...)`, corrupting line breaks (research.md D2 Bugfix); see T044
- [X] T018 Add the DS-03 persistent flags (`--quiet`/`-q`, `--verbose`/`-v`, `--json`, `--color`/`-C`) and a `PersistentPreRun` that selects `bios.SCHEMA` (TTY/`NO_COLOR`/`TERM=dumb`-aware, ADR 002 DS-05) to `cmd/arc/root.go` (depends on T015, T016, T017)
- [X] T019 [P] Implement `internal/adapter/fsys/errors.go`: `faults.Type`/`faults.SafeN` sentinel constants `ErrRootNotDirectory`, `ErrRootCreate`, `ErrCreate`, `ErrRemove` per data-model.md "Error sentinels"
- [X] T020 Implement `internal/adapter/fsys/resolve.go`: `ResolveLocalRoot(root string) (created bool, err error)` and `RemoveLocalRoot(root string) error`, plain `os`/`path/filepath` only (research.md D3/D4) (depends on T019)
- [X] T021 [P] Unit tests for `ResolveLocalRoot`/`RemoveLocalRoot` in `internal/adapter/fsys/resolve_test.go` against `t.TempDir()` (constitution Principle VI) (depends on T020)
- [X] T022 [P] Implement `internal/adapter/fsys/types.go`: `File`, `Store`, `Mounter` interfaces per contracts/fsys-port-contract.md (stdlib `io/fs`/`io.Writer` only)
- [X] T023 Implement `internal/adapter/fsys/local.go`: `Local` wraps `os.DirFS(root)` for reads, adds `Create`/`Remove` (backed by `os.MkdirAll`/`os.Create`/`os.Remove`, wrapping `*os.File` with a `Discard()` method) for writes (depends on T022, T019)
- [X] T024 [P] Integration tests for `Local` in `internal/adapter/fsys/local_test.go` against `t.TempDir()` (constitution Principle VI) (depends on T023)
- [X] T025 [P] Implement `internal/app/ctrl/port/vcs.go`: `VCS` interface (`IsAvailable`, `Init`, `StageAll`, `Commit`) per contracts/vcs-port-contract.md
- [X] T026 ✅ Fixed (BUG-001) Implement `internal/app/ctrl/adapter/git/git.go`: `os/exec`-backed `VCS` implementation, each operation wrapped with `bios.Reporter` Start/Done/Error (research.md D1/D2) (depends on T025, T017) — `Commit` returned the full 40-character SHA (`git rev-parse HEAD`) instead of the short hash the contract requires; see T046
- [X] T027 [P] Integration test for the git adapter in `internal/app/ctrl/adapter/git/git_test.go` against a real `git` binary and `t.TempDir()` (constitution Principle VI) (depends on T026)
- [X] T028 [P] Implement `internal/app/ctrl/adapter/mock/mock.go`: in-memory fake `VCS` with configurable return values/errors and a call log, for service unit tests (depends on T025)
- [X] T029 [P] Implement `internal/app/ctrl/kernel/graph.go`: `GraphRoot`, `ArcNetCoreLayout` (with `var DefaultLayout`), `InitResult` value types per data-model.md
- [X] T030 [P] Unit tests for `ArcNetCoreLayout`'s static shape in `internal/app/ctrl/kernel/graph_test.go` (depends on T029)
- [X] T031 [P] Implement `internal/app/ctrl/service/errors.go`: `faults.Type`/`faults.SafeN` sentinel constants `ErrGitUnavailable`, `ErrAlreadyInitialized`, `ErrTargetNotEmpty`, `ErrLayoutWrite` per data-model.md
- [X] T032 Implement `internal/app/ctrl/component.go`: primary port `Init(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, dir string) (kernel.InitResult, error)` as a thin delegator into `service.Init` (depends on T029, T025, T022)
- [X] T033 [P] Write `internal/app/ctrl/README.md` documenting the `ctrl` use-case per ADR 001's layout convention

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Bootstrap a new graph in the current directory (Priority: P1) 🎯 MVP

**Goal**: Running `arc init` with no arguments in an empty directory produces the full canonical layout, `_meta/` stubs, `.arc/` state directory, `.gitignore`, and exactly one git commit — a ready-to-use empty graph.

**Independent Test**: Run the built `arc` binary with no arguments in an empty temp directory; inspect the resulting file tree and `git log`/`git status` per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010, T011) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T034 [US1] Implement `internal/app/ctrl/service/init.go`: `Init` calls `fsys.ResolveLocalRoot` then `mounter.Mount`, runs the FR-010 (surfaced by `ResolveLocalRoot`), FR-011 (`VCS.IsAvailable`), and FR-015 (`Store.ReadDir(".")` non-empty) guards before any write, writes `ArcNetCoreLayout`'s folders/`.gitkeep`/`_meta` stubs/`.gitignore` via `Store.Create`, then `VCS.Init`/`StageAll`/`Commit` with the exact CORE §11.3 subject line, reporting each step via `bios.Reporter`; on any failure after guards pass, rolls back via `fsys.RemoveLocalRoot` (root freshly created) or per-path `Store.Remove` against the known `ArcNetCoreLayout` list (research.md D4) (depends on T020, T023, T026, T029, T031, T032)
- [X] T035 [US1] ✅ Fixed (BUG-001) Implement `cmd/arc/ctrl/init.go`: `NewInitCmd() *cobra.Command` with a DS-02 `optsInit` options struct, calls `ctrl.Init` (wiring `fsys.Local{}` and the real git adapter), renders the result via a `bios.Registry[kernel.InitResult]` (human confirmation line + `--json` per contracts/cli-contract.md), `PostRunE` next-step hint per ADR 002 DS-12 (depends on T034, T016, T018) — `humanInitPrinter.Show` passed a newline-embedded string into `lipgloss.Style.Render(...)`, the `Reporter` was wired unconditionally instead of `--verbose`-gated, and the `PostRunE` hint text is stale; see T044, T045, T047
- [X] T036 [US1] Populate `Short`/`Long`/`Example` help text for `arc init` per contracts/cli-contract.md's DS-11 shape (constitution Principle XII) in `cmd/arc/ctrl/init.go`
- [X] T037 [US1] Register `ctrl.NewInitCmd()` into `cmd/arc/root.go`'s command tree (depends on T035)

**Checkpoint**: At this point, User Story 1's E2E tests (T010, T011) pass and `arc init` is fully functional and independently testable in an empty local directory

---

## Phase 4: User Story 2 - Bootstrap a graph in a named target directory (Priority: P2)

**Goal**: `arc init <dir>` creates the graph at an explicit path, creating `<dir>` if it doesn't exist, without touching the current working directory, and reports the resolved path on success.

**Independent Test**: Run `arc init ./some/new/path` from a temp directory and confirm the graph is created there while the invoking directory is untouched, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T012) and MUST currently be failing (red).

- [X] T038 [US2] Extend `optsInit` in `cmd/arc/ctrl/init.go` to accept the optional `<dir>` positional argument (defaulting to `.` per FR-008) and pass it through to `ctrl.Init` (depends on T035)
- [X] T039 [US2] Confirm `service.Init`'s `fsys.ResolveLocalRoot` create-if-missing path (FR-009) is exercised end-to-end for a non-existent named `<dir>`, and that `InitResult.Root` carries the resolved path reported in the success message (FR-012) (depends on T034)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Protected against accidentally destroying an existing graph (Priority: P3)

**Goal**: Re-running `arc init` against a directory that already contains `.arc/` refuses cleanly, with zero filesystem or git changes.

**Independent Test**: Run `arc init` twice against the same directory and confirm the second run makes no changes, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T013) and MUST currently be failing (red).

- [X] T040 [US3] Implement the FR-014 guard in `internal/app/ctrl/service/init.go`: after mounting, `Store.Stat(".arc")` succeeding means the graph already exists — refuse with `service.ErrAlreadyInitialized`, making zero writes (depends on T034)

**Checkpoint**: All three user stories pass their E2E tests independently — full feature complete

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T041 [P] Update `README.md`'s quick-start example to mention `arc init` (constitution Principle XII)
- [X] T042 [P] Manually run all 6 quickstart.md scenarios against the built binary and confirm expected output/exit codes
- [X] T043 [P] Add table-driven unit tests in `internal/app/ctrl/service/init_test.go` covering every guard combination (FR-010/011/014/015) against the `adapter/mock` VCS and fakes of `fsys.Mounter`/`fsys.Store`, asserting `errors.Is(err, service.ErrXxx)` per case (constitution Principle VI)

---

## Bugfix: BUG-001 — over-verbose default output, broken alignment, full commit hash, stale hint

**Purpose**: Fix `specs/002-arc-init/bugs/BUG-001.md`. See `research.md` D2 Bugfix for full root-cause analysis and the revised decision, and `contracts/cli-contract.md`/`contracts/vcs-port-contract.md` for the updated contract.

- [X] - [ ] T044 [P] Fix `internal/bios/reporter.go`'s `stderrReporter.Done`/`.Error`: stop passing a string with an embedded/trailing `\n` into `lipgloss.Style.Render(...)` (it pads the phantom empty line instead of emitting a line break) — render the styled text, then write the newline outside the styled span. Apply the identical fix to `humanInitPrinter.Show` in `cmd/arc/ctrl/init.go` (depends on T017 reopened)
- [X] - [ ] T045 Gate `internal/bios.Reporter` selection in `cmd/arc/ctrl/init.go` on `--verbose`/`-v` (previously only gated on `--quiet`): silent by default, faint/gray (`SCHEMA.Hint`-equivalent) progress under `--verbose`, always silent under `--quiet` regardless of `--verbose`; consolidate the git adapter's four reported steps into three — "Checking git availability", "Preparing git repository" (`git init` + `git add -A`), "Committing empty graph" (depends on T044, T026 reopened)
- [X] - [ ] T046 [P] Change `internal/app/ctrl/adapter/git/git.go`'s `VCS.Commit` to return the short hash (`git rev-parse --short HEAD`) instead of the full SHA (`git rev-parse HEAD`), per contracts/vcs-port-contract.md Bugfix (depends on T026 reopened)
- [X] - [ ] T047 [P] Update `cmd/arc/ctrl/init.go`'s `PostRunE` hint text from `(use "arc list" to see what's in your new graph)` to `(use "arc apply <patch.md>" to load content into your new graph)` per contracts/cli-contract.md Bugfix (depends on T035 reopened)
- [X] - [ ] T048 [P] Add/extend a unit test for `internal/bios/reporter.go` (`internal/bios/reporter_test.go`, new file) asserting `Done`/`Error` output ends in a real newline with no injected padding spaces, so this regression cannot reappear silently (constitution Principle VI)
- [X] - [ ] T049 Update `cmd/arc/ctrl/init_test.go`'s existing E2E assertions and/or add a new case verifying: default-mode output is exactly the single confirmation line (no per-step progress on stderr) and the commit hash reported is short (contracts/cli-contract.md, spec.md FR-016)

**Checkpoint**: BUG-001 resolved — default `arc init` output is a single concise line, `--verbose` shows three faint/gray steps with correct alignment, commit hashes are short everywhere, and the next-step hint references `arc apply`

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/bios`, `internal/adapter/fsys`, `internal/app/ctrl` directory-structure explanation (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `arc init [<dir>]`, DS-03 persistent flags, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I)
- [X] TN05 Domain logic uses ports (interfaces); `cmd/arc/ctrl` wiring and `internal/adapter/{fsys,git}` adapters remain separated (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations (`git` subprocess, filesystem) follow the port/adapter pattern; no vendor SDK or `os.*` types leak through a port (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 No configuration file, environment variable, or secret handling was introduced (Principle XI, confirmed N/A per T014)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc init` (Principle XII)
- [X] TN13 E2E tests from Phase 2d (T010–T013) turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md US1/US2/US3 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/ctrl/init_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `arc init` establishes the first `--json` output contract for this command — no prior contract to break, so no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5 (nearly everything they need is built there); User Story 1 is the deepest since it implements `service.Init` itself — User Stories 2 and 3 both extend the same file and therefore depend on T034 (User Story 1's core implementation) as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements the core `service.Init` and `cmd/arc/ctrl/init.go` that US2/US3 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its two tasks (T038, T039) touch files US1 already created (T034, T035) — sequenced after US1 in practice, though its E2E test (T012) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Same shape as US2 — its one task (T040) extends `service/init.go` from T034

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/adapter foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: `internal/bios`'s three files (T015-T017), `internal/adapter/fsys`'s `errors.go`/`types.go` (T019, T022), `internal/app/ctrl/port/vcs.go` (T025), `internal/app/ctrl/kernel/graph.go` (T029), and `internal/app/ctrl/service/errors.go` (T031) have no dependencies on each other and can all run in parallel
- Once Phase 2.5 completes, User Story 1's four tasks (T034-T037) are sequential (each depends on the previous); User Stories 2 and 3 can then proceed in parallel with each other once US1 is done, since T038/T039 and T040 touch different concerns within the files US1 created

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Implement internal/bios/theme.go per ADR 002 DS-05"
Task: "Implement internal/bios/output.go per ADR 002 DS-04"
Task: "Implement internal/bios/reporter.go per ADR 002 DS-06"
Task: "Implement internal/adapter/fsys/errors.go sentinel constants"
Task: "Implement internal/adapter/fsys/types.go (File, Store, Mounter interfaces)"
Task: "Implement internal/app/ctrl/port/vcs.go (VCS interface)"
Task: "Implement internal/app/ctrl/kernel/graph.go value types"
Task: "Implement internal/app/ctrl/service/errors.go sentinel constants"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md Scenario 1 against the built binary
7. Deploy/demo if ready — `arc init` with no arguments is already a usable MVP

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking previous stories

---

## Notes

- `[P]` tasks = different files, no dependencies
- `[Story]` label maps a task to its user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
- User Stories 2 and 3 are not fully file-independent from User Story 1 here (they extend `service/init.go` and `cmd/arc/ctrl/init.go` that US1 creates) — this reflects that all three stories exercise one shared `Init` use-case, not three separate features; each remains independently *testable* via its own E2E test written in Phase 2d

**Bugfix**: 2026-07-02 — BUG-001: Reopened T017, T026, T035 (⚠️ marked); added T044-T049 to fix over-verbose default output, broken line alignment, full-length commit hash, and stale `PostRunE` hint text. See the "Bugfix: BUG-001" section above.
