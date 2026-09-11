---

description: "Task list for 034-serve-dir-public-state"
---

# Tasks: Explicit Serve Context & Shareable Graph State

**Input**: Design documents from `/specs/034-serve-dir-public-state/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/cli-contract.md](contracts/cli-contract.md), [quickstart.md](quickstart.md), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Per Constitution VI and VIII every acceptance scenario in
`spec.md` maps 1:1 to an E2E test written **before** implementation (red-green-refactor).

**Scope directive**: *"The feature does not require any structural changes in the code
base. Only impacted component has to be modified. Keep minimal changes to interfaces,
adjust only the behaviour of `serve` and `init` commands."* — every task below edits a
file that already exists. No task creates a package, a file, a port interface, or a
dependency.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 / US4 — maps to `spec.md`
- Exact file paths included in every task

## Path Conventions

- `cmd/arc/<pkg>/` — Cobra wiring plus colocated `*_test.go` E2E tests (Principles III, VIII)
- `internal/app/<domain>/{kernel,service}/` — use-case value types and logic; no cobra import
- `internal/adapter/fsys/` — the only package permitted to call `os` filesystem functions (Principle VII)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a green baseline. This is a mature repository — there is no
project to initialize and no dependency to add.

- [X] T001 Confirm a clean baseline: `go build ./... && go test ./... && staticcheck ./...` all pass on branch `034-serve-dir-public-state` before any edit
- [X] T002 [P] Record the current `arc init` layout as the regression baseline: run `go test ./internal/app/ctrl/service/ -run TestInitSuccessWritesLayoutAndCommits -v` and note the asserted path list in `internal/app/ctrl/service/init_test.go` (lines ~248, ~473) that Phase 5 will change
- [X] T003 [P] Confirm no new Go module dependency is required — `go.mod` is unchanged by this feature (plan.md Technical Context)

---

## Phase 2: Design Preconditions

**Purpose**: The constitution's PRECONDITIONS (Compliance Checklist). Each subsection
is a design gate whose deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No implementation (Phase 3+) may begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Rewrite the **Arc State Directory** glossary entry in `ARCHITECTURE.md` (~line 282): `.arc/` is now version-controlled; the exclusion applies to `.arc/cache/` only, via `cache/` in the tracked `.arc/.gitignore` plus a self-ignoring `*` in `.arc/cache/.gitignore` (research D3)
- [X] T005 Add the **Arc Cache Directory** glossary entry to `ARCHITECTURE.md`: `.arc/cache/` holds machine-local, reproducible, disposable state; never tracked; absent in a fresh clone by construction (data-model I6); verify no existing `internal/app/*/kernel` type already models it before introducing `arcCacheDir` (Principle II)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc serve [<dir>]` matches `arc init [<dir>]`'s established grammar — single primary subject positional, `cobra.MaximumNArgs(1)`, no new flag, no `--json` schema change — against `contracts/cli-contract.md` §C1
- [X] T007 [P] Confirm the four preflight message shapes and the FR-017 refusal message in `contracts/cli-contract.md` §C1/§C2 are final before any test asserts them verbatim

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 [P] Confirm no new adapter is needed: `internal/adapter/fsys` covers root probing and `internal/adapter/git` covers `check-ignore` via the existing `repoProbe.IsIgnored` (research D5)
- [X] T009 Confirm no port interface changes: `port.VCS`, `fsys.Store`, `fsys.Mounter` keep their exact method sets (`contracts/cli-contract.md` §C4)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> All tests below MUST compile and fail **semantically** before Phase 3 begins. No
> `t.Skip()`, no "red phase" comments. Each test carries its
> `// Scenario X.Y from specs/034-serve-dir-public-state/spec.md` reference.

- [X] T010 [P] [US1] Write E2E tests for US1 scenarios 1-5 in `cmd/arc/graph/serve_test.go` using `sut()` and `connectServeSession()`: serve a graph from an unrelated working directory, no-argument parity, absolute-path launch, refusal for a non-graph path, and `--http` parity with the same `<dir>`
- [X] T011 [P] [US1] Write E2E tests for US1 edge cases in `cmd/arc/graph/serve_test.go`: path is a file, path is unreadable, relative path resolved against the working directory, named `<dir>` wins when the working directory is also a graph, and two positional arguments produce a Cobra usage error with no transport opened
- [X] T012 [P] [US2] Write E2E tests for US2 scenarios 1-4 in `cmd/arc/ctrl/init_test.go` using `sut()` and the existing `gitOutput()` helper: `git ls-files .arc` lists `.arc/.gitkeep` and `.arc/.gitignore`, a clone is recognized by every command, a committed `.arc/config.yml` travels to the clone, and `git status --porcelain` is empty after init
- [X] T013 [P] [US3] Write E2E tests for US3 scenarios 1-4 in `cmd/arc/ctrl/init_test.go`: `.arc/cache/.gitignore` exists with content `*`, a file written into `.arc/cache/` leaves the tree clean and is reported by `git check-ignore`, the initial commit contains `.arc/` content but nothing under `.arc/cache/`, and `--skip-git-init` into a project with its own `.gitignore` leaves that file byte-identical
- [X] T014 [P] [US4] Write E2E tests for US4 scenarios 1-2 in `cmd/arc/ctrl/init_test.go`: a graph fabricated into the legacy layout (`.arc/.gitignore` = `*`, `.arc/` untracked) is still accepted by `arc stats`/`arc serve` with no warning on stdout or stderr, and the same graph after the documented migration yields a usable clone

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T015 Confirm `.arc/config.yml` becoming version-controlled introduces no precedence change — it is already the project-level tier — and that it holds no secret material (only `grep` and `subgraph` numeric knobs in `internal/app/config/kernel/config.go`)

**Checkpoint**: Phase 2 complete — implementation may begin. Phase 2.5 is deliberately
omitted: no shared scaffold is needed, and Phase 3 builds directly on Phase 2.

---

## Phase 3: User Story 1 - Point the MCP server at a graph by path (Priority: P1) 🎯 MVP

**Goal**: `arc serve [<dir>]` serves the named graph from any working directory, with
actionable refusals that happen before any transport opens.

**Independent Test**: from a working directory that is not a graph, run
`arc serve --http :8765 /path/to/graph`, complete an MCP `initialize` handshake, and
confirm the session instructions come from that graph. Then confirm
`arc serve /tmp/does-not-exist` exits non-zero within a second with nothing listening.

**Delivers**: FR-001..FR-008 · SC-001, SC-002, SC-004, SC-007

### Implementation for User Story 1

> E2E tests T010, T011 were written in Phase 2d and MUST currently be failing.

- [X] T016 [P] [US1] Add `ErrGraphDirNotFound`, `ErrGraphDirNotDirectory`, `ErrGraphDirUnreadable` as `faults.Safe1[string]` constants to `internal/app/graph/service/errors.go` with the message shapes fixed in `contracts/cli-contract.md` §C1 (research D2)
- [X] T017 [US1] Classify the graph root in `EnsureGraph` in `internal/app/graph/service/node.go`: stat `"."` through the mounted store before `guardIsGraph` and map `fs.ErrNotExist` / `fs.ErrPermission` / other to the three T016 sentinels; keep the function under 25 lines (Principle IV) and leave the signature unchanged (depends on T016)
- [X] T018 [P] [US1] Add unit tests for the three classification branches plus the unchanged `ErrNotAGraph` path in `internal/app/graph/service/node_test.go`, asserting with `errors.Is` and `github.com/fogfish/it/v2` (depends on T016)
- [X] T019 [US1] Add the package-private `resolveGraphDir(args []string) (string, error)` helper to `cmd/arc/graph/serve.go` per `data-model.md` §5 — `filepath.Abs(args[0])` when one argument is present, `filepath.Abs(".")` otherwise
- [X] T020 [US1] Change `Use: "serve"` → `Use: "serve [<dir>]"` and `Args: cobra.NoArgs` → `Args: cobra.MaximumNArgs(1)` in `NewServeCmd` in `cmd/arc/graph/serve.go` (depends on T019)
- [X] T021 [US1] Replace `dir, err := filepath.Abs(".")` with `dir, err := resolveGraphDir(args)` in `NewServeCmd`'s `RunE` in `cmd/arc/graph/serve.go`; verify `buildServer(ctx, dir)` and all eight tool handlers need no change (depends on T019, T020)
- [X] T022 [US1] Update `Long` and `Example` on `NewServeCmd` in `cmd/arc/graph/serve.go` to document `<dir>`, including the `arc serve ~/graphs/notes` and `arc serve --http :8080 ~/graphs/notes` examples from `contracts/cli-contract.md` §C1 (Principle XII)
- [X] T023 [US1] Run `go test ./cmd/arc/graph/ -run Serve -v` and confirm T010/T011 are green with no change to the test bodies (Principle VIII)

**Checkpoint**: US1 is complete and shippable on its own. An MCP server entry can now
name a graph by absolute path.

---

## Phase 4: User Story 2 - Clone a graph repository and use it immediately (Priority: P1)

**Goal**: `arc init` produces a graph whose `.arc/` state is version-controlled, so a
clone *is* a graph and `.arc/config.yml` travels with the repository.

**Independent Test**: `arc init`, commit a tuned `.arc/config.yml`, `git clone`
elsewhere, and run `arc stats` in the clone with zero setup steps.

**Delivers**: FR-009, FR-011..FR-014, FR-016, FR-017 · SC-003, SC-005, SC-006

**Independence**: does not depend on US1; either P1 story can ship first.

### Implementation for User Story 2

> E2E test T012 was written in Phase 2d and MUST currently be failing.

- [X] T024 [US2] Change `arcIgnoreContent` from `"*\n"` to `"cache/\n"` in `internal/app/ctrl/service/init.go`, and rewrite the `arcIgnorePath` doc comment (~lines 29-40) — its current rationale describes the layout this feature retires (research D3)
- [X] T025 [US2] Change `footprint.Tracked()` in `internal/app/ctrl/service/init.go` to exclude `arcCacheDir` (`.arc/cache`) instead of `arcStateDir`, so `.arc/.gitkeep` and `.arc/.gitignore` enter the commit pathspec; update its doc comment to state why a self-ignoring path must stay out (research D4, data-model §2)
- [X] T026 [P] [US2] Add `StateIgnored bool` to `kernel.InitOpts` in `internal/app/ctrl/kernel/graph.go` with a doc comment stating it is meaningful only when `ParentRepo != ""` (data-model §3)
- [X] T027 [P] [US2] Add `ErrStateIgnored` as a `faults.Safe1[string]` constant to `internal/app/ctrl/service/errors.go` with the message from `contracts/cli-contract.md` §C2
- [X] T028 [US2] Add guard rule R4 (`opts.SkipGitInit && opts.StateIgnored → ErrStateIgnored`) as the last case of `guardRepositoryContext` in `internal/app/ctrl/service/init.go`, keeping R3 ahead of it so a wholly-ignored target still wins (depends on T026, T027)
- [X] T029 [US2] Populate `opts.StateIgnored` in `resolveRepoContext` in `cmd/arc/ctrl/init.go` with a second `probe.IsIgnored(ctx, repo, filepath.ToSlash(filepath.Join(rel, ".arc")))` call, reusing the existing `repoProbe` interface unchanged (depends on T026)
- [X] T030 [US2] Add unit tests to `internal/app/ctrl/service/init_test.go`: `Tracked()` now includes `.arc/.gitkeep` and `.arc/.gitignore` and excludes `.arc/cache/.gitignore`, `.arc/.gitignore` content is `cache/`, and guard R4 refuses before `resolveLocalRoot` runs (mirror the existing `TestInitGuardTargetIgnored` and `TestInitGuardsRunBeforeRootIsResolved` shapes)
- [X] T031 [US2] Run `go test ./cmd/arc/ctrl/ ./internal/app/ctrl/... -v` and confirm T012 is green; update the two written-path assertions at `internal/app/ctrl/service/init_test.go` ~248 and ~473 to the new layout — this is specified behaviour change, not test churn (plan.md Risks)

**Checkpoint**: US1 and US2 both pass independently. A clone of a graph is a graph.

---

## Phase 5: User Story 3 - Machine-local state never leaves the machine (Priority: P2)

**Goal**: `.arc/cache/` exists as the reserved, doubly-protected home for machine-local
state, with no producer in this feature.

**Independent Test**: `arc init`, write an arbitrary file into `.arc/cache/`, and
confirm `git status --porcelain` is empty and `git check-ignore` reports a match.

**Delivers**: FR-010, FR-015 · SC-005

**Depends on**: T024, T025 (US2 must have flipped `.arc/` to public first).

### Implementation for User Story 3

> E2E test T013 was written in Phase 2d and MUST currently be failing.

- [X] T032 [US3] Add `arcCacheDir = ".arc/cache"`, `arcCacheIgnorePath = ".arc/cache/.gitignore"` and `arcCacheIgnoreContent = "*\n"` constants to `internal/app/ctrl/service/init.go` per `data-model.md` §1
- [X] T033 [US3] Append `arcCacheIgnorePath` to the paths `layoutPaths` returns in `internal/app/ctrl/service/init.go`, after `arcStateMarker` and `arcIgnorePath`, so the collision guard and the writer stay derived from one list (depends on T032)
- [X] T034 [US3] Add the `arcCacheIgnorePath → arcCacheIgnoreContent` branch to `contentFor` in `internal/app/ctrl/service/init.go` (depends on T032)
- [X] T035 [US3] Add unit tests to `internal/app/ctrl/service/init_test.go` asserting `.arc/cache/.gitignore` is written with `*\n`, that `absentDirs` records `.arc/cache` deepest-first so rollback removes it before `.arc/`, and that no ignore rule outside `.arc/` is ever created (FR-016)
- [X] T036 [US3] Run `go test ./cmd/arc/ctrl/ ./internal/app/ctrl/... -v` and confirm T013 is green with no change to the test bodies

**Checkpoint**: US1, US2 and US3 all pass independently. The cache is reserved and
protected by two independent rules.

---

## Phase 6: User Story 4 - Bring an existing graph to the new layout (Priority: P3)

**Goal**: graphs created by earlier versions keep working untouched, and their owners
have a short, reversible, documented migration.

**Independent Test**: fabricate the legacy layout, confirm `arc stats` succeeds
silently, apply the three documented commands, and confirm a fresh clone serves.

**Delivers**: FR-018, FR-019, FR-020, FR-021 · SC-008

**Note**: FR-020 is satisfied by writing **no code** — no code path reads the *content*
of `.arc/.gitignore`, so legacy support is free (research D7).

### Implementation for User Story 4

> E2E test T014 was written in Phase 2d and MUST currently be failing.

- [X] T037 [US4] Verify by inspection that `guardIsGraph` in `internal/app/graph/service/apply.go` and `guardNotAlreadyInitialized` in `internal/app/ctrl/service/init.go` stat the `.arc/` **directory** only and never read `.arc/.gitignore`'s content — record the finding; make no code change (FR-018, FR-020)
- [X] T038 [US4] Add the migration section to `README.md`: the three commands from `research.md` D7, the statement that an unmigrated graph keeps working, and the consequence of not migrating — clones are not recognized as graphs (FR-019, FR-021)
- [X] T039 [US4] Confirm no detection, warning, or conversion code was added anywhere for the legacy layout (FR-020, spec Out of Scope)
- [X] T040 [US4] Run `go test ./cmd/arc/ctrl/ -v` and confirm T014 is green

**Checkpoint**: all four user stories pass independently.

---

## Phase 7: Collateral & Documentation

**Purpose**: the one consequence outside `serve`/`init`, plus the documentation
Constitution I and XII make mandatory in the same change.

- [X] T041 Change `probeFoldsCase` in `internal/adapter/fsys/local.go` to prefer `.arc/cache`, falling back to `.arc`, then to the graph root; rewrite its comment, whose current justification (*"already gitignored"*) is exactly the premise this feature retires. The fallback chain is load-bearing: a clone has no `.arc/cache/` (data-model I6) and without it the function returns `safeDefault` = case-**in**sensitive, silently changing node-identity merge behaviour on Linux (research D6, plan.md Scope Deviations)
- [X] T042 [P] Add a unit test to `internal/adapter/fsys/local_test.go` asserting the probe prefers `.arc/cache` when present and still answers correctly when only `.arc/` exists (the clone case)
- [X] T043 [P] Update `README.md` in three places: the bootstrap prose at ~line 121 (`.arc/` is committed; `.arc/cache/` is the excluded part), the folder tree at ~line 142, and the Usage sentence at ~line 215 — *"`arc` operates on the graph in the current working directory"* is now false for `serve` and must name the `arc serve <dir>` form
- [X] T044 [P] Add the MCP client configuration snippet from `contracts/cli-contract.md` §C1 to `README.md`, since declarative agent configuration is the motivating use case (SC-002)
- [X] T045 [P] Update `cmd/arc/ctrl/init.go`'s `Long` text — it currently says `.arc/` is created *"with its own rule excluding it from version control"*, which becomes false (Principle XII: help text MUST NOT drift from behaviour)
- [X] T046 [P] Add the feature entry to `specs/CHANGELOG.md` under the `2026-09-11` heading, following the existing `/speckit-specify` + `/speckit-plan` format

---

## Additional Polish (OPTIONAL)

- [X] T047 Run every scenario in [quickstart.md](quickstart.md) (V1-V8) against a real built binary and confirm the observed output matches the documented expectations
- [X] T048 [P] Run `go test ./... -cover` and confirm coverage has not regressed on `internal/app/ctrl/service` or `cmd/arc/graph`
- [X] T049 [P] Run `staticcheck ./...` clean

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase).
This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — plan.md concludes none was; confirm ADR 001/002/003 are still accurate and unmodified
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [X] TN06 Unit tests compiled and failed semantically without the implementation (Principle VI). **Recorded accurately**: the E2E tier (T010-T014) was written strictly red-first and verified red before Phase 3. The unit tier (T018, T030, T035) was written immediately *after* its code, following this task list's own ordering (T018 is placed after T016/T017; T030/T035 after T024-T034). The property Principle VI protects was therefore verified retroactively instead: with `guardGraphRoot`, guard R4 and the `Tracked()` predicate each neutralised in turn, `TestEnsureGraphMissingRoot…`, `…UnreadableRoot…`, `…FileRoot…`, `TestInitFootprintTracksArcStateButNotCache`, `TestInitGuardStateIgnored` and `TestInitStateIgnoredGuardRunsBeforeRootIsResolved` all fail, and all pass once restored. `TestInitGuardTargetIgnoredWinsOverStateIgnored` passes either way by design — it pins R3's priority, not R4's existence.
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — confirm `port.VCS`, `fsys.Store`, `fsys.Mounter` method sets are byte-identical to `main`
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — plan.md concludes **minor, additive**; confirm no existing invocation changed meaning
- [X] TN16 Every `.go` file touched carries the MIT license header required by `CLAUDE.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies — start immediately
- **Design Preconditions (Phase 2)**: depends on Setup — BLOCKS all user stories; subsections 2a-2e proceed in parallel
- **Phase 2.5**: omitted — no shared scaffold needed
- **US1 (Phase 3)**: depends on Phase 2 only — fully independent of US2/US3/US4
- **US2 (Phase 4)**: depends on Phase 2 only — fully independent of US1
- **US3 (Phase 5)**: depends on Phase 2 **and on T024, T025** (US2 must flip `.arc/` to public before the cache split is meaningful)
- **US4 (Phase 6)**: depends on Phase 2; T038's migration text describes the layout US2/US3 produce
- **Collateral (Phase 7)**: T041/T042 depend on US3 (`.arc/cache/` must exist); T043-T046 depend on US1+US2+US3
- **Polish**: depends on all desired stories
- **Phase N**: final gate — depends on everything

### User Story Dependencies

- **US1 (P1)**: independent. Ships alone.
- **US2 (P1)**: independent. Ships alone.
- **US3 (P2)**: needs US2's constant changes (T024, T025).
- **US4 (P3)**: documentation of US2+US3's outcome; no code.

### Within Each User Story

- E2E tests (Phase 2d) already red before implementation starts
- Error sentinels before the logic that returns them (T016 → T017; T027 → T028)
- Value types before the guards that read them (T026 → T028, T029)
- Service/layout logic before the Cobra wiring that feeds it
- Help text last, once behaviour is settled

### Parallel Opportunities

- T002, T003 in parallel
- T007, T008 in parallel within Phase 2
- **T010-T014 all in parallel** — four test-writing tasks across two files with no shared symbols
- T016 ∥ T018 (sentinels and their tests), and T026 ∥ T027 (kernel field and error constant, different packages)
- **US1 (Phase 3) and US2 (Phase 4) fully in parallel** — disjoint file sets (`cmd/arc/graph` + `internal/app/graph` vs. `cmd/arc/ctrl` + `internal/app/ctrl`)
- T042-T046 in parallel — five different files

---

## Parallel Example: the two P1 stories

```bash
# Phase 2d (already complete, all red):
#   T010, T011 → cmd/arc/graph/serve_test.go
#   T012, T013 → cmd/arc/ctrl/init_test.go

# Then, with two developers, no file overlap at all:
Developer A (US1): internal/app/graph/service/{errors,node,node_test}.go
                   cmd/arc/graph/serve.go
Developer B (US2): internal/app/ctrl/{kernel/graph.go,service/{init,errors,init_test}.go}
                   cmd/arc/ctrl/init.go
```

---

## Implementation Strategy

### MVP First

The smallest shippable increment is **US1 alone** — `arc serve [<dir>]` — eight tasks
touching two production files. It fixes declarative MCP configuration, which is the
acute complaint, and it ships without any change to what `arc init` writes.

1. Phase 1 → Phase 2 → Phase 3 → Phase N. **STOP and validate** with quickstart V1-V3.

### Incremental Delivery

1. Setup + Design Preconditions → foundation ready
2. **US1** → validate V1-V3 → ship (MVP: agents can name a graph)
3. **US2** → validate V4-V6 → ship (clones work; config is shared)
4. **US3** → validate V4 → ship (cache reserved and doubly protected)
5. **US4 + Phase 7** → validate V7-V8 → ship (migration documented; probe relocated)
6. Phase N before merge

Note that US1 and US2 are both P1 and mutually independent — if only one can land,
either is a coherent release on its own.

---

## Notes

- Every task edits an existing file. No task creates a package, file, port interface, or dependency.
- T031's changes to the two written-path assertions are **specified behaviour change**, not test churn — those assertions are the pin points of the old layout and changing them is the point of the feature (plan.md Risks).
- T041 is the one edit outside `serve`/`init`; it is a consequence of making `.arc/` tracked, and is flagged as such in plan.md Scope Deviations rather than folded in quietly.
- T037 and T039 are verification tasks that deliberately produce **no code** — FR-020 is satisfied by not writing any.
- Commit after each task or logical group.
- Phase 2 and Phase N are retained per constitution Governance > Task List Requirements.
