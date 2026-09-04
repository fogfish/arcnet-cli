---

description: "Task list for 031-init-existing-git-repo"
---

# Tasks: Graph Initialization Inside an Existing Git Repository

**Input**: Design documents from `/specs/031-init-existing-git-repo/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Per Principles VI and VIII every acceptance scenario maps 1:1 to an E2E test, and tests are written before implementation (red-green-refactor).

**Bugfix**: 2026-09-04 — [BUG-001](bugs/BUG-001.md) Updated from bugfix patch. Reopened T013/T048/T052 (US4's parity coverage exercised repository *nesting* only, never a graph root *shared* with host-project files — the shape FR-010 newly legalized) and TN14 (the new scenarios have no E2E test yet). Added Phase 7 (T058-T070) to remove `arc apply`'s whole-graph scan, give its read path an accurate error type, and add `arc lint`'s foreign-file index. **Not false completions**: each reopened task correctly implemented the scope it was given; the scope itself was short by one axis.

**Bugfix (round 2)**: 2026-09-04 — [BUG-001](bugs/BUG-001.md) `/speckit-bugfix-verify` found five gaps in round 1. Reopened T056 (quickstart now has an S10 it does not run) and TN15 (its release assessment covers one additive `--json` field; T066 adds a second and T063 changes a user-visible error message); moved Phase 7 after Additional Polish so task ids read in order; corrected the BUG-001 critical-path summary, which showed T063 waiting on T062 when it depends on T061 only; and qualified the bare "BUG-002" reference in Phase 6, which pointed at a bug report this feature does not have.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1–US4, mapping to spec.md user stories

## Path Conventions

Per [plan.md](plan.md) Structure Decision — no new package. Work spans one Cobra command (`cmd/arc/ctrl`), one shared adapter (`internal/adapter/git`), and one use-case package (`internal/app/ctrl` across `kernel`/`port`/`service`/`adapter/mock`). `internal/app/graph`, `internal/app/lint` and `internal/adapter/fsys` stay unmodified.

**Story coupling, stated honestly**: US1, US2 and US3 are all P1 and each is independently *testable*, but they share one file — `internal/app/ctrl/service/init.go`. They are sequential-by-file, not parallel, even though their E2E tests are independent. US4 is genuinely independent of the other three.

---

## Phase 1: Setup

**Purpose**: Establish the baseline this feature's changes will be measured against.

- [X] T001 Record the green baseline: run `go test ./...` and `go vet ./...`, and list every assertion in `cmd/arc/ctrl/init_test.go` that depends on a root `.gitignore` — these are the assertions [research.md](research.md) D4 predicts must change, and the list is the evidence for SC-008
- [X] T002 [P] Confirm `staticcheck ./...` runs clean before any edit, so new findings are attributable to this feature
- [X] T003 [P] Add a `newHostRepo(t *testing.T) string` test helper in `cmd/arc/ctrl/init_test.go` that creates a temp git repo with one commit and a configured identity — used by every US1/US2/US3 E2E test below

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate — the deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add **parent repository** and **initialization footprint** to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary, with the definitions used in [data-model.md](data-model.md)
- [X] T005 Verify `kernel.InitOpts` duplicates no existing `internal/app/ctrl/kernel` or `internal/core` type before introducing it; confirm `InitResult` is the right home for the `repository` field rather than a new result type

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm the `--skip-git-init` flag surface against [contracts/cli-contract.md](contracts/cli-contract.md) C1–C2: long form only, no shorthand, default `false`, and the four-row mode table is exhaustive
- [X] T007 [P] Confirm the `--json` schema in [contracts/cli-contract.md](contracts/cli-contract.md) C3 is additive-only — `path`, `commit`, `foldersCreated` unchanged in name, type and meaning (FR-028), `repository` present and non-empty in both modes (FR-027)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 [P] Confirm `internal/adapter/git` is the correct existing adapter to extend and that no new adapter is warranted (it is — ADR 001's application-level adapter tier already owns every git subprocess)
- [X] T009 Finalize the `internal/app/ctrl/port.VCS` shape per [contracts/cli-contract.md](contracts/cli-contract.md) C7: `StageAll`/`Commit` out, `StagePaths`/`CommitPaths` in, detection deliberately **off** the port; confirm `CommitPaths` must be a new adapter method rather than a `Commit` signature change because `internal/app/graph/port.VCS` declares the pathspec-free shape

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

All four tasks write tests that MUST compile and fail semantically (red) before Phase 2.5 begins.

- [X] T010 [P] [US1] Write E2E tests in `cmd/arc/ctrl/init_test.go` for User Story 1 scenarios 1–5 ([quickstart.md](quickstart.md) S2): refusal at a repo root, refusal in a subfolder, refusal for a non-existent subfolder with nothing left on disk, error message naming both the repo root and the flag, and standalone init still succeeding outside any repo
- [X] T011 [P] [US2] Write E2E tests in `cmd/arc/ctrl/init_test.go` for User Story 2 scenarios 1–6 ([quickstart.md](quickstart.md) S3, S6, S7): graph created at a populated repo root, exactly one new commit containing only initialization files, the user's modified/staged/untracked files untouched, nested subfolder init, already-initialized refusal (FR-011), and the no-enclosing-repository refusal
- [X] T012 [P] [US3] Write E2E tests in `cmd/arc/ctrl/init_test.go` for User Story 3 scenarios 1–6 ([quickstart.md](quickstart.md) S4, S5, S8): host `.gitignore` byte-for-byte unchanged and absent from the commit, local state excluded via `.arc/.gitignore` alone, file collision, folder collision naming the subfolder recovery, successful subfolder recovery, and rollback leaving pre-existing files intact
- [X] T013 [P] [US4] ⚠️ Reopened (BUG-001) ✅ Closed (T058-T061 added the root-mode fixture) Write E2E tests in a new `cmd/arc/graph/nested_repo_test.go` for User Story 4 scenarios 1–5 ([quickstart.md](quickstart.md) S9): `arc apply`, `arc revert` and `arc lint` against a graph nested inside a host repo carrying unrelated history, each asserting parity with the standalone equivalent (reopened — BUG-001: `initHostGraph` places the graph at `<repo>/kb` and the unrelated markdown at `<repo>/other/unrelated.md`, **outside** the graph subtree, so no scenario ever puts a foreign file where a tree-walking command would meet it. The root-mode shape of spec.md US2 Scenario 1 is untested by any test in the repository; coverage is completed by T058–T061)
- [X] T014 Confirm every acceptance scenario across the spec's four user stories has exactly one E2E test and that all of them currently fail for the right reason (semantic failure, not a compile error)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T015 Confirm the feature introduces no configuration value, environment variable or secret, and record that Constitution XI's "never modify a config file it does not own" rule is satisfied structurally by the `.arc/.gitignore` decision ([research.md](research.md) D4) rather than by a confirmation prompt

**Checkpoint**: All Phase 2 subsections complete — implementation can begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The adapter, port, kernel and signature plumbing every user story depends on. Behavior is deliberately unchanged at the end of this phase — the same files are written and the same commit is produced — so the whole existing test suite (minus the root-`.gitignore` assertions from T001) stays green.

**⚠️ BLOCKS all user stories.**

- [X] T016 [P] Add `ErrGitRevParse` and `ErrGitCheckIgnore` sentinels to `internal/adapter/git/git.go` following the existing `faults.Type` block
- [X] T017 [P] Implement `RepoRoot(ctx, dir) (string, error)` in `internal/adapter/git/git.go` per [contracts/vcs-adapter-contract.md](contracts/vcs-adapter-contract.md) A1 — `rev-parse --show-toplevel`, returning `("", nil)` when not in a repository, following the `IsTracked`/`ShowFile` expected-negative convention
- [X] T018 [P] Implement `IsIgnored(ctx, dir, path) (bool, error)` in `internal/adapter/git/git.go` per A2 — `check-ignore -q`, discriminating exit 1 (not ignored, expected) from exit ≥2 (genuine failure) using the same `exec.ExitError` code check `IsTracked` uses
- [X] T019 [P] Implement `StagePaths(ctx, dir, paths) error` in `internal/adapter/git/git.go` per A3 — `add -- <paths>`, returning nil without invoking git on an empty slice so it can never degrade to staging everything
- [X] T020 [P] Implement `CommitPaths(ctx, dir, message, paths) (string, error)` in `internal/adapter/git/git.go` per A4 — `commit -m <msg> -- <paths>` then `rev-parse --short HEAD`, added as a **new** method leaving `Commit` and `StageAll` intact for `internal/app/graph`
- [X] T021 Add adapter unit tests in `internal/adapter/git/git_test.go` for T017–T020, covering the exit-code discrimination in A1/A2 and the pathspec-commit isolation verified in [research.md](research.md) D3 — a pre-staged file must survive uncommitted (FR-014)
- [X] T022 Swap `StageAll`/`Commit` for `StagePaths`/`CommitPaths` in `internal/app/ctrl/port/vcs.go` (depends on T009)
- [X] T023 Add `StagePaths`/`CommitPaths` to `internal/app/ctrl/adapter/mock/mock.go`, recording paths in the existing `Calls` log so tests can assert the exact pathspec (depends on T022)
- [X] T024 [P] Add `kernel.InitOpts` (`ParentRepo`, `SkipGitInit`, `TargetIgnored`) and the `Repository`/`repository` field on `kernel.InitResult` in `internal/app/ctrl/kernel/graph.go` per [data-model.md](data-model.md)
- [X] T025 Thread the new plumbing through without behavior change: `writeLayout` in `internal/app/ctrl/service/init.go` returns the ordered footprint it wrote; `Init` stages and commits that footprint via `StagePaths`/`CommitPaths` (FR-013, FR-014); `Init` accepts `kernel.InitOpts`; `internal/app/ctrl/component.go`'s delegator and `cmd/arc/ctrl/init.go`'s call site pass it (depends on T022, T023, T024)

**Checkpoint**: Everything compiles, existing suite green, behavior unchanged. User stories can now proceed — sequentially through `service/init.go`.

---

## Phase 3: User Story 1 — Refuse to nest a repository inside a repository (Priority: P1) 🎯 MVP

**Goal**: `arc init` inside an existing repository fails, touches nothing, and names both the enclosing repository and the flag that overrides the refusal.

**Independent Test**: Inside a versioned project, run `arc init` targeting a new subfolder; verify non-zero exit, an actionable message, and no `sub/` left on disk.

**Why this is the MVP**: it alone stops the tool producing nested repositories — the harmful default — without needing any other story.

> E2E tests written in T010 and currently failing (red).

- [X] T026 [US1] Add `ErrInsideRepository` to `internal/app/ctrl/service/errors.go` with the FR-006 message naming the repository root and `--skip-git-init`
- [X] T027 [US1] Resolve the parent repository in `cmd/arc/ctrl/init.go`: walk from the requested target to its nearest **existing** ancestor, call `git.RepoRoot` there, and pass the result as `InitOpts.ParentRepo` (FR-001, FR-002, FR-003; [research.md](research.md) D2 — this must happen before the directory is created)
- [X] T028 [US1] Add guard R1 to `internal/app/ctrl/service/init.go` (`!SkipGitInit && ParentRepo != ""` → `ErrInsideRepository`), positioned **before** `resolveLocalRoot` so a refusal never touches disk (FR-004, FR-005)
- [X] T029 [US1] Reorder the remaining guards in `internal/app/ctrl/service/init.go` ahead of `resolveLocalRoot` per [data-model.md](data-model.md)'s state transition, so every refusal is write-free rather than relying on rollback
- [X] T030 [US1] Add unit tests in `internal/app/ctrl/service/init_test.go` for R1 and the reordering, asserting the mock records no `Init`/`StagePaths`/`CommitPaths` calls and the fake store records no writes

**Checkpoint**: T010 green. Nested repositories can no longer be created. Shippable on its own.

---

## Phase 4: User Story 2 — Add a graph to an existing project (Priority: P1)

**Goal**: `--skip-git-init` creates the graph in place, uses the parent repository, and commits only what initialization itself produced.

**Independent Test**: In a populated versioned project with unrelated pending changes, run `arc init --skip-git-init`; verify no nested repository, exactly one new commit containing only graph files, and every unrelated change untouched.

> E2E tests written in T011 and currently failing (red).

- [X] T031 [US2] Add `ErrNoParentRepository` to `internal/app/ctrl/service/errors.go` with the FR-012 message
- [X] T032 [US2] Register the `--skip-git-init` bool flag in `cmd/arc/ctrl/init.go` per [contracts/cli-contract.md](contracts/cli-contract.md) C1, passing it as `InitOpts.SkipGitInit` (FR-008)
- [X] T033 [US2] Add guard R2 to `internal/app/ctrl/service/init.go` (`SkipGitInit && ParentRepo == ""` → `ErrNoParentRepository`)
- [X] T034 [US2] Make `guardTargetEmpty` in `internal/app/ctrl/service/init.go` apply only when `!SkipGitInit` (FR-010), leaving today's message and behavior untouched for the standalone path (FR-007)
- [X] T035 [US2] Branch around `vcs.Init` in `internal/app/ctrl/service/init.go`: skip repository creation when `SkipGitInit`, and direct staging and committing at `InitOpts.ParentRepo` rather than the graph root (FR-009)
- [X] T036 [US2] Populate `InitResult.Repository` in `internal/app/ctrl/service/init.go` — `ParentRepo` when skipping, the graph root otherwise — so it is non-empty in both modes (FR-027)
- [X] T037 [US2] Render the new field and mode in `cmd/arc/ctrl/init.go`: `repository` in JSON output, and the FR-026 human line naming the existing repository, preserving BUG-001's single-line/single-`\n` constraints ([contracts/cli-contract.md](contracts/cli-contract.md) C3–C4)
- [X] T038 [US2] Add unit tests in `internal/app/ctrl/service/init_test.go` for R2, the mode branch (mock records no `Init` call), the staging/commit directory, and `Repository` in both modes

**Checkpoint**: T011 green. Graphs can be added to existing projects. US1 + US2 together deliver the requested capability.

---

## Phase 5: User Story 3 — Preserve the host project's existing files (Priority: P1)

**Goal**: Adding a graph never overwrites, adopts or modifies anything the user owns, and any failure leaves the target exactly as found.

**Independent Test**: In a project with its own ignore rules and a folder colliding with a canonical layout name, run `arc init --skip-git-init`; verify the collision is refused with the subfolder recovery named, the ignore file is byte-for-byte unchanged, and local state is still excluded.

> E2E tests written in T012 and currently failing (red).

- [X] T039 [P] [US3] Add `ErrLayoutCollision` and `ErrTargetIgnored` to `internal/app/ctrl/service/errors.go`, with FR-031's message naming the conflicting path and the subfolder recovery
- [X] T040 [US3] Replace the root `.gitignore` write in `writeLayout` (`internal/app/ctrl/service/init.go`) with `.arc/.gitignore` containing `*`, applied identically in both modes, and confirm no host ignore file is ever created, read or modified (FR-016, FR-017; [research.md](research.md) D4)
- [X] T041 [US3] Add the FR-015 collision guard to `internal/app/ctrl/service/init.go`: fail before any write when a layout file path exists, or when any of the eight `kernel.DefaultLayout.Folders` exists — empty or not — using `fsys.Store.Stat`'s `IsDir()` to separate the two classes
- [X] T042 [US3] Probe ignore status in `cmd/arc/ctrl/init.go` via `git.IsIgnored`, only when a parent repository was found, passing it as `InitOpts.TargetIgnored`
- [X] T043 [US3] Add guard R3 to `internal/app/ctrl/service/init.go` (`SkipGitInit && TargetIgnored` → `ErrTargetIgnored`), converting the late, confusing "nothing to commit" failure into an upfront refusal (FR-020)
- [X] T044 [US3] Rework `rollback` in `internal/app/ctrl/service/init.go` to remove exactly the recorded footprint instead of the static layout list, and to never remove a directory that pre-existed the run (FR-018, FR-019)
- [X] T045 [US3] Update the `fakeStore` fixtures in `internal/app/ctrl/service/init_test.go` so `fakeDirEntry.IsDir()` is configurable, which the folder-collision guard needs (currently hardcoded `false`)
- [X] T046 [US3] Add unit tests in `internal/app/ctrl/service/init_test.go` for both collision classes, R3, and footprint-scoped rollback leaving pre-existing paths intact
- [X] T047 [US3] Update the root-`.gitignore` assertions catalogued in T001 — in `cmd/arc/ctrl/init_test.go` — to assert on `.arc/.gitignore` instead, holding SC-008's observable outcomes fixed: local state untracked, working tree clean, one commit

**Checkpoint**: T012 green. Safe to run against a real project. All three P1 stories complete.

---

## Phase 6: User Story 4 — Every git-backed command works from the parent repository (Priority: P2)

**Goal**: A graph nested inside a larger project behaves identically to a standalone one across every command that touches version control.

**Independent Test**: Create a graph in a subfolder of a project with unrelated history; run apply, revert and lint; verify parity with the standalone equivalent.

**Approach**: verification-first ([research.md](research.md) D7). The adapter already carries the nested-repo fixes from [spec 016 BUG-002](../016-arc-revert/bugs/BUG-002.md); these tasks prove them and fix only what actually fails. *(Reference qualified — Bugfix BUG-001, 2026-09-04: this feature has no BUG-002 of its own, and the bare id sent readers to a missing file.)*

> E2E tests written in T013 and currently failing (red) — but many are expected to pass as soon as they can run.

- [X] T048 [US4] ⚠️ Reopened (BUG-001) ✅ Closed (T069 extended the triage with the root-sharing axis) Run the T013 suite (`cmd/arc/graph/nested_repo_test.go`) and record in the task notes which scenarios pass on the existing adapter and which fail, establishing FR-021 as verified rather than assumed — this triage is the deliverable, and it determines whether T049–T051 are needed at all (reopened — BUG-001: the triage in [notes.md](notes.md) is accurate for the git-adapter layer it examined, and T049–T051 stay closed on its evidence. It is incomplete only in that no scenario reached the filesystem-walk layer, where the actual defect lives. Re-run and extend the triage table with the root-sharing axis once T058–T061 exist)
- [X] T049 [US4] **Closed as no-change-needed** — T048 shows no staging leakage; `StageAll`'s `-A -- .` already confines it (evidence: `TestNestedApplyCommitsOnlyGraphSubtree`, [notes.md](notes.md) T048)
- [X] T050 [US4] **Closed as no-change-needed** — T048 shows no history-query leakage; `CommitsMatching`'s `-- .` already scopes it (evidence: `TestNestedLintScopesHistoryToGraphSubtree`)
- [X] T051 [US4] **Closed as no-change-needed** — T048 shows no path-resolution leakage; `ChangedPaths --relative` and `ShowFile ./` already handle it (evidence: `TestNestedRevertAffectsOnlyGraphSubtree`)
- [X] T052 [US4] ⚠️ Reopened (BUG-001) ✅ Closed (TestRootModeApplyParityWithStandalone) Confirm FR-025 parity in `cmd/arc/graph/nested_repo_test.go`: every T013 scenario produces the same observable outcome nested as standalone (reopened — BUG-001: parity was confirmed across the nesting axis only. Re-confirm across the root-sharing axis, where `arc apply` today fails outright on a graph that a standalone run applies cleanly)

**Checkpoint**: T013 green. The nested-graph promise is verified rather than assumed.

---

## Additional Polish

- [X] T053 [P] Update `Long` and `Example` on the init command in `cmd/arc/ctrl/init.go` to document `--skip-git-init`, when to use it, and the default refusal it overrides (FR-030)
- [X] T054 [P] Update `README.md`'s command table and the `arc init` narrative to cover the flag and the subfolder recovery pattern
- [X] T055 [P] Add the [ARCHITECTURE.md](../../ARCHITECTURE.md) note recording why `ctrl` resolves repository context in `cmd` and passes it as a value while `graph`/`lint` probe through their own ports, so the asymmetry does not read as an oversight
- [X] T056 ⚠️ Reopened (BUG-001) ✅ Closed (S1-S10 run against a real build; S10 verified end to end) Run every [quickstart.md](quickstart.md) scenario **S1–S10** against a real build and confirm the stated expectations (reopened — BUG-001: accurate for S1–S9, which is exactly the problem — every host-repo scenario there initializes into a subfolder, so none could reach the defect. S10 adds the root-sharing shape; this task now covers it) (depends on T070 — S10 cannot pass until Phase 7 lands, so this task moves to the end of the run even though it is listed here)
- [X] T057 Run `go test ./...`, `go vet ./...` and `staticcheck ./...`; confirm no new findings against the T001/T002 baseline

---

## Phase 7: Bugfix BUG-001 — `arc apply` Scans a Tree It No Longer Owns

**Bugfix**: 2026-09-04 — [BUG-001](bugs/BUG-001.md) Updated from bugfix patch. `guardNoOldFormatNodes` walks every `*.md` under the graph root and parses each as a node before any write. That was sound while `arc init` refused a non-empty target; FR-010 legalized the shared root and made it fatal — a host project's `README.md` aborts every `arc apply` with `failed to write README.md: … missing mandatory "@id" field`, a message naming a write that never happens. The guard also defends against the pre-0.5 node format, which no shipping `arc` produces, so it is removed rather than narrowed. `arc lint` and `arc grep` must still walk; they index foreign files instead of failing on them (spec.md FR-032-FR-035).

**Goal**: `arc apply` reads only what its patch targets; `arc lint` and `arc grep` walk the shared root without mistaking the host project's markdown for graph content.

**Independent Test**: `git init` a project with a `README.md`, `arc init --skip-git-init` at its root, then `arc apply`, `arc lint` and `arc grep` — each behaves as it does on a standalone graph, and `README.md` is untouched and uncommitted.

### Red phase — the scenarios no existing test reaches

- [X] T058 [P] [US4] Extend `cmd/arc/graph/nested_repo_test.go` with a **root-mode** fixture — host repo whose root carries `README.md`, `CONTRIBUTING.md` and a `docs/**.md` tree, with the graph initialized at that same root (no `kb/` subfolder) — and add failing E2E cases asserting `arc apply` parity with the standalone equivalent, `README.md` byte-for-byte unchanged, and no foreign file staged or committed (FR-032, FR-033, SC-009). This fixture is the axis `initHostGraph` does not cover, so it belongs beside it, not in place of it
- [X] T059 [P] [US4] Add a failing E2E case in `cmd/arc/lint/lint_test.go`: `arc lint` on the T058 root-mode graph reports zero violations attributable to foreign files, and its "N nodes checked" count equals the graph's own node count (FR-035, SC-010). Today it reports `README.md:1 — [frontMatter] … missing mandatory "@id" field` and counts it as `1 nodes checked, 0 passing, 1 failing`
- [X] T060 [P] [US4] Add a regression-lock test for `arc grep` on the T058 root-mode graph: foreign files land in the existing `Unreadable` index ([kernel/grep.go](../../internal/app/graph/kernel/grep.go)) and never in `Matches` (FR-035). `grep` already behaves correctly — this pins the behavior the other two commands are being brought in line with, so it cannot regress silently
- [X] T061 [P] [US4] Add a failing test for FR-034: a patch targeting a canonical node path already occupied by a foreign file fails, naming that path and the mandatory field it lacks, with a **read**-path message, zero writes and zero commits

### Green phase — `arc apply` stops walking

- [X] T062 [US4] Delete `guardNoOldFormatNodes` and its call site in `internal/app/graph/service/apply.go` (the guard body and the `Apply` call preceding `readPatch`), satisfying FR-033. Removal, not narrowing: the pre-0.5 format it detects is retired, and the read-triggered detection spec 010 US3 actually asks for is already carried by `core.ParseNode` on every path `apply` genuinely reads (depends on T058, T061)
- [X] T063 [US4] Add `ErrNodeRead` (`"failed to read %s"`) to `internal/app/graph/service/errors.go` and route every read-path rejection in `readExistingNode` through it, leaving `ErrNodeWrite` to `writeNode`/`writeRaw` where a write is genuinely attempted (FR-034, FR-029) (depends on T061)
- [X] T064 [US4] Confirm by test that `apply`'s remaining reads are exactly the patch-targeted paths — assert the opened-file count tracks the patch's node count and is unchanged when the host project's foreign markdown grows from zero to many (FR-033, SC-009) (depends on T062)

### Green phase — `arc lint` gains the index `arc grep` already has

- [X] T065 [US4] Introduce the foreign-file index in `internal/app/lint/service/lint.go`: classify each walked file, exclude foreign ones from rule evaluation and from the nodes-checked count, and collect them, mirroring `internal/app/graph/service.Grep`'s `unreadable` accumulation rather than inventing a second convention (Principle V) (depends on T059)
- [X] T066 [US4] Extend the lint result kernel type with the foreign-file list and surface it in `--json` as an additive field, and in human-readable output as a distinct non-failing note — never as a violation and never affecting exit status (FR-035, FR-028's additive-contract precedent) (depends on T065)

### Cleanup — the removed guard's dependents

- [X] T067 [P] [US4] Update the two comments in `internal/app/graph/service/revert.go` that cite `guardNoOldFormatNodes` as precedent for `isPatchDocument`'s patch-file tolerance. `isPatchDocument` itself is correct and stays — only its stated rationale needs a live referent (depends on T062)
- [X] T068 [P] [US4] Retire or re-scope the tests that encode the removed whole-graph walk: `TestApplyRetiredKeyPatchInGraphTreeSurfacesLegacyKindError` in `cmd/arc/graph/apply_test.go`, and spec 010's T074/T075 old-format fixtures under `cmd/arc/graph/testdata/`. A retired-key patch sitting in the graph tree is no longer `apply`'s business unless the patch targets it — assert that instead of the old abort (depends on T062)

### Verification

- [X] T069 [US4] Re-run T048's triage and extend [notes.md](notes.md)'s table with the root-sharing axis, so the record shows both axes verified rather than one (depends on T058-T068)
- [X] T070 [US4] Run `go test ./...`, `go vet ./...` and `staticcheck ./...`; confirm no new findings against the T001/T002 baseline, and re-run [quickstart.md](quickstart.md) S9 **and S10** — S10 is the root-sharing scenario added by this bugfix and carries the BUG-001 reproduction as its regression check (depends on T069)

**Checkpoint**: a graph sharing its root with a host project is fully usable — `apply`, `lint`, `grep` and `revert` all behave as they do standalone, and the host project's own markdown is never read as graph content.

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). Retained verbatim.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX) — including FR-029: every new failure exits non-zero with an actionable message routed through `cmd/arc/main.go`'s `humanize`, never a raw Go error

### Implementation Phase Verification (grouped by principle)

- [X] TN04 **N/A as expected** — Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — expected N/A: ADR 001 is followed, not amended
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — verify no `exec.Cmd`/`exec.ExitError` escapes `internal/adapter/git`
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 ⚠️ Reopened (BUG-001) ✅ Closed (FR-032-FR-035 and SC-009/SC-010 each carry E2E coverage) All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — reopened: FR-032–FR-035, SC-009/SC-010 and the three new edge cases have no E2E test yet (T058–T061)
- [X] TN15 ⚠️ Reopened (BUG-001) ✅ Closed — both changes are additive/behavioral, not breaking: lint's `foreign` is a new `--json` field (no existing field renamed or removed), and `ErrNodeRead` changes an error message's wording on a path that already exited non-zero. Minor bump, no major Release/versioning impact assessed (Principle XIV) — the `--json` change is additive (FR-028) and the `.gitignore` relocation preserves SC-008's observable outcomes, so no major bump is expected; confirm explicitly (reopened — BUG-001: the recorded assessment covers one additive `--json` field. T066 adds a **second** — `arc lint`'s foreign-file list — and T063 introduces a new error sentinel on a path that previously reported `ErrNodeWrite`. Re-assess across both; the error-message change is user-visible even though the exit status is not)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a–2e proceed in parallel
- **Foundational (Phase 2.5)**: Depends on Phase 2 — BLOCKS all user stories
- **User Stories (Phase 3–6)**: All depend on Phase 2.5
- **Polish**: Depends on all desired stories
- **Phase 7 (Bugfix BUG-001)**: Depends on US2 and US4 existing — it corrects the walk-layer consequence of FR-010, so it cannot precede the flag that creates the shared root. It is placed **after** Additional Polish so task ids read in order, but it is not polish: the reopened closing gates (T056, TN14, TN15) all depend on it and are the last things to run *(Bugfix BUG-001, 2026-09-04)*
- **Phase N**: Final gate

### User Story Dependencies

- **US1 (P1)**: After Phase 2.5. No dependency on other stories.
- **US2 (P1)**: After Phase 2.5. Independently testable, but edits the same file as US1 — sequence after it.
- **US3 (P1)**: After Phase 2.5. Independently testable, but edits the same file as US1/US2 — sequence after them.
- **US4 (P2)**: After Phase 2.5. Genuinely independent — different files entirely, can run concurrently with US1–US3 if staffed.

### Within Each User Story

- E2E tests (Phase 2d) already written and failing
- Error sentinels → cmd probes → service guards → unit tests
- Story complete before the next priority

### Parallel Opportunities

- T002, T003 in Setup
- All of Phase 2a–2e; T007, T008 and all four Phase 2d test-writing tasks (T010–T013) are [P] — different concerns, and T013 is a different file entirely
- **T016–T020 in Phase 2.5** — the five git adapter methods are independent additions to one file; parallel if split by author, otherwise a fast sequential batch. T024 is [P] against them (different package)
- T039 against other US3 work; T053–T055 in Polish
- **US4 (T048–T052) against all of US1–US3** — the only genuine cross-story parallelism, since it touches `cmd/arc/graph` and `internal/app/graph` rather than `internal/app/ctrl`
- **Phase 7 (BUG-001)**: T058–T061 are all [P] (four different test files/concerns). T062–T064 (`internal/app/graph`) and T065–T066 (`internal/app/lint`) are two independent green-phase tracks and may run concurrently. T067 and T068 are [P] against each other once T062 lands

### Critical Path

`T001 → T014 → T022/T023/T024 → T025 → T028 (US1) → T035 (US2) → T041/T044 (US3) → T056 → TN15`

**Bugfix BUG-001 path** *(2026-09-04)*: `T058/T061 → T062 → T064`, `T061 → T063`, and `T059 → T065 → T066`, joining at `T069 → T070 → T056/TN14/TN15`. T063 depends on T061 only — it is the error-sentinel change and does not wait on the guard removal

---

## Implementation Strategy

**MVP = Phase 1 + Phase 2 + Phase 2.5 + Phase 3 (US1).** That alone stops `arc init` creating nested repositories, which is the harmful default and the reason this feature exists. It ships without `--skip-git-init` existing at all — users inside a repository simply get a clear refusal.

**Increment 2 = + US2.** The refusal now has an escape hatch, and the requested capability — adding a graph to an existing project — works end to end.

**Increment 3 = + US3.** Makes increment 2 safe to run against a real, populated project rather than a clean one. Do not ship US2 to users without US3 in the same release: US2 alone can adopt a user's existing folder as graph content.

**Increment 4 = + US4.** Verifies the nested-graph promise across the rest of the command surface. Independent of the others and can be worked concurrently.

**Increment 5 = + Phase 7 (BUG-001).** Increment 2 is not actually usable without it: `--skip-git-init` at a project root produces a graph whose very next `arc apply` fails. Treat Phase 7 as completing US2's promise, not as optional follow-up work.

**Sequencing caution**: US1–US3 all edit `internal/app/ctrl/service/init.go`. Treat them as three sequential commits on one file, not three parallel workstreams. The parallelism in this feature is inside Phase 2.5 (adapter methods) and in US4.
