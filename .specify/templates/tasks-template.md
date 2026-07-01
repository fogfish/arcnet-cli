---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, [.specify/memory/constitution.md](../memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: The examples below include test tasks. Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/<command>/` — Cobra command definitions, flag/argument parsing, and their colocated `<command>_test.go` E2E tests (constitution Principle III, VIII)
- `internal/<domain>/` — domain logic and port interfaces; MUST NOT import `github.com/spf13/cobra` (Principle III)
- `internal/<domain>/adapter/` — driven adapters implementing domain ports (cloud SDKs, REST clients, filesystem, cache) (Principle VII)
- `testdata/` — E2E test fixtures, colocated with the test file (Principle VIII)
- Paths shown below assume this layout — adjust based on plan.md structure

<!--
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.

  The /speckit-tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Command/flag contracts from contracts/

  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment

  Phase 2 and Phase N below are MANDATORY per constitution Governance >
  Task List Requirements: AI agents generating tasks.md MUST retain these
  two phases verbatim, adapting only the task descriptions to the feature.
  Omitting them violates the constitution and blocks feature completion.

  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 0: Pre-implementation Refactoring (OPTIONAL)

**Include only when the feature requires significant changes to existing code** (rename, restructure, extract interfaces, split files). MUST be submitted as a separate PR from feature work. All existing tests MUST pass after refactoring.

- [ ] T000 [Describe refactor] in [file path] — all existing tests still pass

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project structure per implementation plan (`cmd/<command>/`, `internal/<domain>/`)
- [ ] T002 Initialize Go module dependencies (`github.com/spf13/cobra`, `github.com/fogfish/it/v2`, `github.com/charmbracelet/lipgloss` per [Mandatory Libraries & Tooling](../memory/constitution.md#mandatory-libraries--tooling))
- [ ] T003 [P] Configure `staticcheck` and confirm it runs clean on the new package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T004 Identify domain entities, aggregates, and value objects for this feature; add them to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary
- [ ] T005 Verify no new domain type duplicates an existing `internal/<domain>` type before introducing it

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [ ] T006 Design subcommand name(s), flags, and `--json`/`--plain` output schema; confirm noun/verb ordering matches existing commands
- [ ] T007 [P] Document the command contract in `contracts/` (flag table, exit codes, example invocations)

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [ ] T008 [P] Check for an existing adapter covering the required external system before designing a new one
- [ ] T009 Define the port interface (narrow, capability-scoped) that the new adapter must satisfy

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T010 [P] [US1] Write E2E test(s) in `cmd/<command>/<command>_test.go` for every acceptance scenario in `spec.md` mapped to US1, using the `sut()` helper; tests MUST compile and fail semantically (red phase)
- [ ] T011 [P] [US2] Write E2E test(s) in `cmd/<command>/<command>_test.go` for every acceptance scenario mapped to US2 (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [ ] T012 Confirm any new config values follow the flag → env → project config → user config → system config precedence and, if secret, are never accepted as plaintext flag values

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure / Command Boilerplate (CUSTOMIZE)

**Purpose**: Feature-specific shared foundation that multiple user stories depend on. Include one or both of the following patterns as applicable; omit entirely if Phase 3 can start directly on top of Phase 2.

Examples (adjust based on your project):

- [ ] T013 [P] Register new Cobra command(s) with `RunE` returning a "not implemented" error (empty-but-compiling scaffold)
- [ ] T014 [P] Scaffold adapter package with empty methods satisfying the Phase 2c port interface
- [ ] T015 Wire configuration loading and error-handling for the new command tree

**Checkpoint**: Foundation ready — user story implementation can now proceed in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own, e.g., "Run `tool <command> --flag value` against a mock adapter and confirm stdout/exit code"]

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [ ] T016 [P] [US1] Implement [domain type] in `internal/<domain>/<type>.go`
- [ ] T017 [US1] Implement port + adapter in `internal/<domain>/adapter/<adapter>.go` (depends on T009)
- [ ] T018 [US1] Implement `RunE` for the command in `cmd/<command>/<command>.go` (depends on T016, T017)
- [ ] T019 [US1] Add unit tests for domain logic using `github.com/fogfish/it/v2` in `internal/<domain>/<type>_test.go`
- [ ] T020 [US1] Populate `Short`/`Long`/`Example` help text for the command (Principle XII)

**Checkpoint**: At this point, User Story 1's E2E tests (T010) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red).

- [ ] T021 [P] [US2] Implement [domain type] in `internal/<domain>/<type>.go`
- [ ] T022 [US2] Implement `RunE` for the command in `cmd/<command>/<command>.go`
- [ ] T023 [US2] Integrate with User Story 1 components (if needed)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

[Add more user story phases as needed, following the same pattern: implementation only — E2E tests were written in Phase 2d]

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates (README, generated `cobra/doc` reference)
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX [P] Additional unit tests beyond Phase 2d/implementation coverage
- [ ] TXXX Run quickstart.md validation

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [ ] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [ ] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [ ] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I)
- [ ] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [ ] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../memory/constitution.md#mandatory-libraries--tooling))
- [ ] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [ ] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII)
- [ ] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [ ] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [ ] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [ ] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [ ] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [ ] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Pre-implementation Refactoring (Phase 0)**: No dependencies — separate PR, run first if included
- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2 (and Phase 2.5 if present)
  - User stories can proceed in parallel (if staffed) or sequentially in priority order (P1 → P2 → P3)
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2 (and 2.5, if present) — No dependencies on other stories
- **User Story 2 (P2)**: Can start after Phase 2 (and 2.5) — May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Phase 2 (and 2.5) — May integrate with US1/US2 but should be independently testable

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain types before ports/adapters
- Ports/adapters before the command's `RunE`
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- Once Phase 2 (and 2.5) completes, all user stories can start in parallel (if team capacity allows)
- Domain types within a story marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2, already complete before this point):
# T010 E2E test(s) for US1 acceptance scenarios in cmd/<command>/<command>_test.go

# Launch independent implementation tasks for User Story 1 together:
Task: "Implement [domain type] in internal/<domain>/<type>.go"
Task: "Implement port + adapter in internal/<domain>/adapter/<adapter>.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure, if included
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently
7. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Foundational Infrastructure together
2. Once complete:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections MUST be retained verbatim across features (constitution Governance > Task List Requirements) — only task descriptions are adapted
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
