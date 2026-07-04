# Tasks: Validate Graph Conformance (`arc lint`)

**Input**: Design documents from `/specs/004-arc-lint/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`cli-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `cmd/arc/lint/` — Cobra command wiring for `arc lint` and its colocated E2E test
- `internal/adapter/git/` — existing shared git adapter, gains one method (`CommitsMatching`)
- `internal/app/lint/{kernel,port,adapter/mock,service}/` — the new `lint` domain use-case, per ADR 001's `componentX` layout
- `internal/app/config` — existing, consumed unchanged (`Resolve`)
- `internal/core` — existing, consumed unchanged (`ParseNode`)
- `testdata/lint/` — fixture graphs, one deliberately-broken-per-rule plus one fully conformant

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create the package skeleton: `internal/app/lint/{kernel,port,adapter/mock,service}/`, `cmd/arc/lint/`, `testdata/lint/` directories per plan.md's Project Structure
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged per plan.md Technical Context (research.md reuses existing `goldmark`/`yaml.v3` transitively via `internal/core`/`internal/app/config`)
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the new (still-empty) package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the domain terms from spec.md Key Entities / data-model.md — Violation, Lint Run, Checklist Rule, Predicate Registry, Extension Profile Checklist — to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle I obligation, plan.md Constitution Check row I)
- [X] T005 Verify no existing `internal/<domain>` package already defines a `Violation`/`LintResult`-shaped type before introducing them in `internal/app/lint/kernel` (none exist — this is the project's first lint-shaped result type)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc lint`'s bare-verb grammar, no positional arguments, and zero new flags (research.md D14 — `--verbose`/`-v` is reused, not reinvented) against contracts/cli-contract.md
- [X] T007 [P] Review contracts/cli-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 [P] Confirm `internal/adapter/git` is the sole git adapter and that this feature introduces no second git client — only one new method (`CommitsMatching`) on the existing type (research.md D12)
- [X] T009 Define `internal/app/lint/port/vcs.go`'s `VCS` interface shape (`CommitsMatching` only, contracts-equivalent to data-model.md's Ports section) as the design gate before any adapter/mock code is written

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T010 [P] [US1] Write E2E tests in `cmd/arc/lint/lint_test.go` for spec.md US1's 3 acceptance scenarios (a fully conformant graph reports a clean pass and exits 0; a graph with violations lists every one with rule/file/line and exits non-zero; violations spanning multiple rules and files are all reported in the same run) using the `sut()` helper — tests MUST compile and fail semantically (red phase)
- [X] T011 [P] [US2] Write E2E tests in `cmd/arc/lint/lint_test.go` for spec.md US2's 3 acceptance scenarios (an unresolved `[[link]]` is reported with file/line; a resolved link produces no violation; two independently created nodes sharing a basename are reported naming both files) — red phase
- [X] T012 [P] [US3] Write E2E tests in `cmd/arc/lint/lint_test.go` for spec.md US3's 2 acceptance scenarios (a file containing unresolved git conflict markers is reported with the first marker's line; a conflict-marker-free graph reports none) — red phase
- [X] T013 [P] Write E2E tests in `cmd/arc/lint/lint_test.go` for the Edge Cases tied to guard/UX behavior: target not an initialized graph (FR-017), and `--verbose` listing every node's pass/fail status while the default mode lists only failing nodes (research.md D14, the user's explicit `-v` requirement) — per quickstart.md Scenario 4 — red phase

> T010–T013 all target the same new file (`cmd/arc/lint/lint_test.go`) and are therefore sequential in practice despite each being scoped to one story (mirrors `specs/003-apply-patch/tasks.md`'s T016–T020 note).

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T014 Confirm `arc lint` introduces no new configuration surface — it reads the existing `.arc/config.yml` via `internal/app/config.Resolve` unchanged, and no secret or credential material is involved (plan.md Constitution Check row XI)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The node-enumeration walk, line locator, kernel value types, and the one new git-adapter method are genuinely foundational — every one of US1–US3 runs the same walk and reports through the same `kernel.LintResult` shape, differing only in which check(s) their own E2E scenarios exercise. This phase builds that shared foundation; Phase 3+ adds each rule-check and the command surface on top of it.

### `internal/adapter/git` — one new method (research.md D12)

- [X] T015 [P] Add `CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)` to `internal/adapter/git/git.go`, wrapping `git log --all --fixed-strings --grep=<needle> --format=%H`
- [X] T016 [P] Add integration test coverage for `CommitsMatching` in `internal/adapter/git/git_test.go` against a real `git` binary and `t.TempDir()`: zero matches, exactly one match, more than one match (depends on T015)

### `internal/app/lint/kernel` — value types

- [X] T017 [P] Implement `internal/app/lint/kernel/lint.go`: the 12 `Rule` constants, `Violation`, `NodeStatus`, `LintResult` per data-model.md
- [X] T018 [P] Implement the Sowa category decode tables in `internal/app/lint/kernel/lint.go` per research.md D7 (`sowaPosition1`/`sowaPosition2`/`sowaPosition3`/`sowaLeaf` fixed word-sets)
- [X] T019 [P] Unit tests in `internal/app/lint/kernel/lint_test.go`: `LintResult.Passing`/`.Failing` derivation from a `[]NodeStatus` fixture, and that every `Rule` constant has a distinct value (depends on T017)

### `internal/app/lint/port` / `adapter/mock`

- [X] T020 [P] Implement `internal/app/lint/port/vcs.go`: `VCS` interface (`CommitsMatching` only) per data-model.md (depends on T009)
- [X] T021 [P] Implement `internal/app/lint/adapter/mock/mock.go`: in-memory fake `VCS` with configurable return values/errors and a call log, for service unit tests (depends on T020)

### `internal/app/lint/service` — errors and line locator

- [X] T022 [P] Implement `internal/app/lint/service/errors.go`: `faults.Safe1[string]` sentinel constants `ErrNotAGraph`, `ErrPredicatesUnreadable` per data-model.md
- [X] T023 Implement `internal/app/lint/service/locate.go` per research.md D3: a front-matter/`kind` line locator, a `[[Target]]`/predicate-qualified-inline-form occurrence locator, a predicate-token/block-label locator, and a conflict-marker line scanner — all operating on raw `[]byte`, no goldmark
- [X] T024 [P] Unit tests for every `locate.go` function in `internal/app/lint/service/locate_test.go` against fixtures with known byte/line positions (depends on T023)

### `internal/app/lint/service` — node enumeration (research.md D2)

- [X] T025 Implement `internal/app/lint/service/lint.go`'s node-enumeration walk: recursive `fsys.Store.ReadDir`, excluding `.arc/` and the two `_meta` registry stubs (`_meta/predicates.md`, `_meta/aliases.md`), parsing each remaining `*.md` via `core.ParseNode`; a parse failure is recorded as a `RuleFrontMatter` `Violation` and that file is excluded from every subsequent check; also builds the basename→path index (research.md D4) needed by every other rule (depends on T017, T022)
- [X] T026 [P] Unit tests for the enumeration walk in `internal/app/lint/service/lint_test.go`: excludes `.arc/` and the two `_meta` stubs, includes a node in a non-standard/domain folder, records `RuleFrontMatter` for an unparseable file without aborting the rest of the walk (constitution Principle VI) (depends on T025)

### `internal/app/lint/service` — predicate registry parsing (research.md D9)

- [X] T027 Implement `_meta/predicates.md` parsing in `internal/app/lint/service/rules_predicates.go`: a bullet-list + inline-code-span scan building `map[string]bool`; an absent file is treated as "every predicate unregistered" (not an error); a genuine read failure returns `ErrPredicatesUnreadable` (depends on T022)
- [X] T028 [P] Unit tests for the predicates-registry parser in `internal/app/lint/service/rules_predicates_test.go`: well-formed registry with several entries, absent file, and a real read failure (constitution Principle VI) (depends on T027)

### Wiring skeleton

- [X] T029 [P] Implement `internal/app/lint/component.go`: primary port `Lint(ctx, mounter, vcs, rules, dir) (kernel.LintResult, error)`, a thin delegator into `service.Lint` (depends on T025)
- [X] T030 [P] Write `internal/app/lint/README.md` documenting the `lint` use-case per ADR 001's layout convention

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Confirm a graph is conformant before trusting it (Priority: P1) 🎯 MVP

**Goal**: Every base CORE §14 checklist rule runs across every enumerated node in one pass; every violation is reported with its rule, file, and line; the run never stops at the first violation found; default output lists only failing nodes plus an overall summary, and the exit code distinguishes pass from fail.

**Independent Test**: Run `arc lint` against a graph with one deliberately introduced violation per checklist rule and confirm every one is listed with file/line, then against a fully conformant graph and confirm a clean pass — per quickstart.md Scenarios 1 and 1b.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010, T013) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T031 [US1] Implement `checkUniqueBasenames` (FR-002/research.md D4) in `internal/app/lint/service/rules_frontmatter.go`, over T025's basename index — a collision produces one `Violation` with `RelatedPaths` naming every colliding file (depends on T025)
- [X] T032 [US1] Implement `checkLinksResolve` (FR-003/research.md D5) in `internal/app/lint/service/rules_links.go`: walk every node's `HRefs`/`Edges`/`Links[*].Seq`, check each `Target` against T025's basename index, locate the violation via `locate.go` (depends on T025, T023)
- [X] T033 [US1] Implement `checkDerivedProvenance` (FR-006/research.md D8) in `internal/app/lint/service/rules_links.go`: for every non-`source` node, confirm at least one resolved link (per T032's index) targets a `source`-kind node (depends on T032)
- [X] T034 [US1] Implement `checkSourceCitekey` (FR-004/research.md D6) in `internal/app/lint/service/rules_identity.go`: for every `source` node, compare `core.Node.ID` against the actual on-disk basename from T025's walk (depends on T025, T023)
- [X] T035 [US1] Implement `checkEntityCategory` (FR-005/research.md D7) in `internal/app/lint/service/rules_identity.go` against T018's Sowa decode tables (depends on T018, T023)
- [X] T036 [US1] Implement `checkPredicateCase` and `checkPredicateRegistered` (FR-007/FR-008/research.md D9) in `internal/app/lint/service/rules_predicates.go` as two distinct violations, the second checked against T027's parsed registry (depends on T027, T023)
- [X] T037 [US1] Implement `checkCitationPredicate` (FR-009/research.md D10) in `internal/app/lint/service/rules_predicates.go`: every `HRefs` entry with a non-empty `Predicate` checked against the fixed `cito:`-aligned set (depends on T023)
- [X] T038 [US1] Implement `checkIngestCommit` (FR-010/research.md D12) in `internal/app/lint/service/rules_history.go`: for every `source` node, call `port.VCS.CommitsMatching(ctx, dir, "Source-Id: "+id)`, violation on zero or more than one match (depends on T020, T025)
- [X] T039 [US1] Implement `checkUnrecognizedKind` (FR-011/FR-018/research.md D11) in `internal/app/lint/service/rules_frontmatter.go`: every node whose `Kind` is absent from the resolved `core.MergeRuleSet` is a `RuleUnrecognizedKind` violation (depends on T025)
- [X] T040 [US1] Implement `internal/app/lint/service/lint.go`'s `Lint(ctx, mounter, vcs, rules, dir) (kernel.LintResult, error)` top-level orchestration: guard `ErrNotAGraph` (`Store.Stat(".arc")`), run T025's enumeration, run every check (T031–T039) across every parseable node, aggregate into `kernel.LintResult{Nodes, Violations, Passing, Failing}` — never stopping at the first violation (FR-013) (depends on T031, T032, T033, T034, T035, T036, T037, T038, T039)
- [X] T041 [US1] Report each of the four documented `Reporter` phases (data-model.md) via `bios.Reporter` around T040's phases (depends on T040)
- [X] T042 [US1] Implement `internal/app/lint/component.go`'s real delegation and `cmd/arc/lint/lint.go`'s `NewLintCmd()`: mounts the graph via `fsys.Local{}`, calls `internal/app/config.Resolve` then `internal/app/lint.Lint`, constructs the real `internal/adapter/git.Git` VCS and `bios.NewReporter(bios.Quiet, !bios.Verbose)` (depends on T029, T040)
- [X] T043 [US1] Implement `humanLintPrinter` (the `bios.Registry`'s `Human` renderer) in `cmd/arc/lint/lint.go`: lists only `NodeStatus` entries with violations plus the overall summary line, per contracts/cli-contract.md
- [X] T044 [US1] Implement `verboseLintPrinter` (the `bios.Registry`'s `Verbose` renderer) in `cmd/arc/lint/lint.go`: lists every node's individual pass/fail status plus the identical summary line, per contracts/cli-contract.md and research.md D14 (the user's explicit `-v` requirement)
- [X] T045 [US1] Wire `bios.Registry[kernel.LintResult]{Human: humanLintPrinter{}, Verbose: verboseLintPrinter{}}` and resolve/print via `bios.ResolveMode()` in `cmd/arc/lint/lint.go` (depends on T043, T044)
- [X] T046 [US1] Implement the DS-07 exit-code contract in `cmd/arc/lint/lint.go`: return a sentinel error from `RunE` (after the result has already been printed) when `LintResult.Violations` is non-empty, so the existing top-level `Execute()` maps it to a non-zero exit code (FR-016) (depends on T045)
- [X] T047 [US1] Populate `Short`/`Long`/`Example` help text for `arc lint` per contracts/cli-contract.md's DS-11 shape (constitution Principle XII) in `cmd/arc/lint/lint.go`
- [X] T048 [US1] Register `lint.NewLintCmd()` into `cmd/arc/root.go`'s command tree (depends on T042)

**Checkpoint**: At this point, User Story 1's E2E tests (T010, T013) pass and `arc lint` is fully functional and independently testable against a graph exercising every base CORE §14 rule except the merge-conflict pre-pass (US3)

---

## Phase 4: User Story 2 - Catch a broken link introduced by a hand edit or a bad patch (Priority: P2)

**Goal**: An unresolved `[[link]]` is reported precisely (and only that); a resolved link produces no noise; a basename collision between two independently created nodes names every colliding file.

**Independent Test**: Introduce a `[[link]]` to a nonexistent basename into an otherwise-conformant graph and confirm exactly that link is reported, per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red) until Phase 3's `checkLinksResolve`/`checkUniqueBasenames` land; this phase verifies and hardens their exact reported shape against US2's own acceptance scenarios.

- [X] T049 [US2] Verify `checkLinksResolve`'s `Violation.Message` names the exact unresolved target string, and confirm a resolvable link produces zero output for that link (spec User Story 2 Acceptance Scenarios 1-2); adjust T032 if T011's E2E test reveals a message-shape mismatch (depends on T032)
- [X] T050 [US2] Verify `checkUniqueBasenames`'s `Violation.RelatedPaths` names every colliding path (not just the first two) for a 3-or-more-way collision (spec User Story 2 Acceptance Scenario 3 generalization); adjust T031 if needed (depends on T031)
- [X] T051 [P] [US2] Add unit tests in `internal/app/lint/service/rules_links_test.go` and `internal/app/lint/service/rules_frontmatter_test.go`: an unresolved link, a resolved link (no violation), and a three-way basename collision (constitution Principle VI) (depends on T049, T050)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Spot a graph left mid-merge-conflict (Priority: P3)

**Goal**: A node file containing unresolved git merge-conflict markers is reported precisely, before any other check runs against that same file's (invalid) content.

**Independent Test**: Write a node file containing unresolved conflict markers, run `arc lint`, and confirm it is reported with the first marker's line and no secondary, confusing violations for the same file, per quickstart.md Scenario 3.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T012) and MUST currently be failing (red).

- [X] T052 [US3] Implement `checkConflictMarkers` (FR-012/research.md D13) as a pre-pass in `internal/app/lint/service/lint.go`'s per-file loop, running *before* `core.ParseNode` is attempted (T025): scan raw lines for `<<<<<<<`, `=======`, `>>>>>>>`; on a match, record one `RuleMergeConflict` violation at the first marker's line and exclude that file from every other check (T025's "excluded from further checks" rule) (depends on T025, T023)
- [X] T053 [P] [US3] Add unit tests in `internal/app/lint/service/lint_test.go`: a conflicted file yields exactly one `RuleMergeConflict` violation and no secondary front-matter/link violations for that same file; a conflict-marker-free graph yields zero `RuleMergeConflict` violations (constitution Principle VI) (depends on T052)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently — full base-checklist feature complete (FR-011's extension-profile depth remains deliberately scoped per plan.md Complexity Tracking)

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T054 [P] Update `README.md`'s quick-start example to mention `arc lint` (constitution Principle XII)
- [X] T055 [P] Manually run all 4 quickstart.md scenarios against the built binary and confirm expected output/exit codes, including the read-only verification at the end of quickstart.md
- [X] T056 [P] Add table-driven unit tests in `internal/app/lint/service/lint_test.go` covering every guard/rule combination end-to-end against `adapter/mock`'s `VCS` and fakes of `fsys.Mounter`/`fsys.Store`, asserting `errors.Is(err, service.ErrXxx)` where applicable (constitution Principle VI)

---

## Bugfix: BUG-001 — `checkDerivedProvenance` false-positives on the tool's own `timeline` index files

**Purpose**: Fix `specs/004-arc-lint/bugs/BUG-001.md`. See `research.md` D8's new Bugfix paragraph for the revised decision and the clarified spec.md FR-006.

- [X] T057 [US1] In `internal/app/lint/service/rules_links.go`'s `checkDerivedProvenance`, exempt `node.Kind == "timeline"` alongside the existing `node.Kind == "source"` exemption (research.md D8 Bugfix, spec.md FR-006) (depends on `checkDerivedProvenance`'s existing implementation, T033)
- [X] T058 [P] Add a unit test in `internal/app/lint/service/rules_links_test.go` asserting a `timeline`-kind node is exempt from `checkDerivedProvenance` regardless of whether it has any resolvable links — constitution Principle VI (depends on T057)

**Checkpoint**: BUG-001 resolved — `arc lint` no longer reports a `derivedProvenance` violation against any `timeline`-kind node (yearly or monthly period file)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/app/lint`, `cmd/arc/lint`, and the extended `internal/adapter/git` directory-structure explanation (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `arc lint`, no new flags, DS-03 persistent flags, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I) — none expected for this feature; confirm during review
- [X] TN05 Domain logic uses ports (interfaces); `cmd/arc/lint` wiring and `internal/adapter/git`/`internal/adapter/fsys` adapters remain separated (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 The one new external integration (`git log` via `CommitsMatching`) follows the port/adapter pattern through the existing shared `internal/adapter/git`; no vendor SDK or subprocess type leaks through `lint.port.VCS` (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence respected; no new configuration surface was introduced; no secrets logged (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for `arc lint` (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md US1–US3 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/lint/lint_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `arc lint` is a new command with a new, additive `--json` `LintResult` schema (no prior contract to break); no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5; User Story 1 is the deepest since it implements every base rule check and the full command surface — User Stories 2 and 3 extend the same files and therefore depend on Phase 3's tasks as well as Phase 2.5
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; implements every base check and `cmd/arc/lint/lint.go` that US2/US3 extend
- **User Story 2 (P2)**: Can start after Phase 2.5, but its tasks (T049-T051) verify/harden checks US1 already implements (T031, T032) — sequenced after US1 in practice, though its E2E test (T011) is independent and was written in Phase 2d
- **User Story 3 (P3)**: Its one new check (T052) is additive to `service.Lint`'s per-file loop (T040) — sequenced after US1, though its E2E test (T012) is independent

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/adapter foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: the git-adapter method (T015-T016), the kernel value types (T017-T019), the port/mock (T020-T021), and the errors/locate/enumeration/predicate-registry chain in `internal/app/lint/service` have no cross-dependencies beyond what's noted and can largely proceed in parallel
- Within Phase 3: T031-T039 (the nine individual rule checks) have no dependencies on each other beyond the shared T025/T023 foundation and can all run in parallel; T040 (orchestration) depends on all of them
- Once Phase 3 lands, User Stories 2 and 3 can proceed in parallel with each other

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Add CommitsMatching to internal/adapter/git/git.go"
Task: "Implement internal/app/lint/kernel/lint.go (Rule, Violation, NodeStatus, LintResult, Sowa tables)"
Task: "Implement internal/app/lint/port/vcs.go (VCS interface)"
Task: "Implement internal/app/lint/adapter/mock/mock.go"
Task: "Implement internal/app/lint/service/errors.go sentinel constants"
Task: "Implement internal/app/lint/service/locate.go"
```

## Parallel Example: Phase 3 Rule Checks

```bash
# Once T025 (enumeration) and T023 (locator) exist, launch every rule check together:
Task: "Implement checkUniqueBasenames in rules_frontmatter.go"
Task: "Implement checkLinksResolve in rules_links.go"
Task: "Implement checkSourceCitekey in rules_identity.go"
Task: "Implement checkEntityCategory in rules_identity.go"
Task: "Implement checkPredicateCase/checkPredicateRegistered in rules_predicates.go"
Task: "Implement checkCitationPredicate in rules_predicates.go"
Task: "Implement checkIngestCommit in rules_history.go"
Task: "Implement checkUnrecognizedKind in rules_frontmatter.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md Scenarios 1 and 1b against the built binary
7. Deploy/demo if ready — `arc lint` already checks every base CORE §14 rule at this point, missing only the merge-conflict pre-pass (US3) and US2's message-shape hardening

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
- No Phase 0 (Pre-implementation Refactoring) is included — unlike `specs/003-apply-patch`'s git-adapter promotion, this feature's one git-adapter change (`CommitsMatching`) is purely additive, not a rename/restructure/split of existing code
- User Stories 2 and 3 are not fully file-independent from User Story 1 here (they extend `internal/app/lint/service/*.go` files US1 creates) — this reflects that all three stories exercise one shared `Lint` use-case and one shared per-file loop, not three separate features; each remains independently *testable* via its own E2E test written in Phase 2d
- FR-011's extension-kind profile-checklist depth is deliberately scoped to kind-recognition only (plan.md Complexity Tracking, research.md D11) — no task above attempts a deeper per-kind schema check, since no mechanism to declare one exists yet in this codebase

**Bugfix**: 2026-07-04 — BUG-001: `checkDerivedProvenance` (T033) correctly implemented spec.md FR-006 exactly as originally written; FR-006 itself over-generalized CORE §3.4's "a node distilled from a document" to every non-`source` kind, incorrectly including the format's own reserved `timeline` index kind. Clarified FR-006 to exempt `timeline` alongside `source`; added T057 (the code exemption) and T058 (regression test). No task reopened. See the "Bugfix: BUG-001" section above.
