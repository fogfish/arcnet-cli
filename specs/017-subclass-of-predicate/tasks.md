# Tasks: Type Inheritance via `rdfs:subClassOf`

**Input**: Design documents from `/specs/017-subclass-of-predicate/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/type-schema-document.md](contracts/type-schema-document.md), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario maps to an E2E test (via the existing `arc lint`/`arc init` command surface, since this feature adds no new command of its own), and tests are written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (spec.md P1-P2). This feature's real complexity is one shared resolution engine every story exercises differently, so the engine itself is built once in Phase 2.5 (Foundational); each user-story phase then turns that story's own already-written E2E test green and handles only story-specific fixture/migration work.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US4)

## Path Conventions

- `internal/app/schema/kernel/` — seed data (`CorePredicateDefs`, `CoreTypeDefs`, new `CoreTypeBases`)
- `internal/app/schema/service/` — resolution logic (`Resolve`, `Seed`, decode/flatten)
- `internal/app/lint/service/` — existing conformance checks; not expected to change, but its hand-built test fixture may need updating
- `cmd/arc/lint/`, `cmd/arc/ctrl/` — existing commands whose E2E tests exercise this feature end-to-end (no new command)
- Paths below match plan.md's Structure Decision (this feature touches only `internal/app/schema`'s two subpackages, plus adjacent test fixtures)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the baseline before any change begins — this feature adds no new package, command, or Go module dependency.

- [X] T001 Confirm `internal/app/schema/kernel` and `internal/app/schema/service` remain the only production packages this feature touches, per plan.md's Structure Decision — no new directory, command, or `go.mod` dependency is required
- [X] T002 [P] Run `staticcheck ./internal/app/schema/...` and `go test ./internal/app/schema/... ./internal/app/lint/... ./cmd/arc/lint/... ./cmd/arc/ctrl/...` to record a clean baseline before changes begin

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T003 Add `rdfs:subClassOf` (predicate), `Node` (type), and "effective/inherited contract" to [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary section, describing them at the same level of detail as existing `Property`/`Class` entries
- [X] T004 Confirm (already verified during `/speckit-plan` research) that no existing `internal/app/schema` type models type-to-type inheritance before introducing data-model.md's `rawType`/resolve() structures — no action expected, this is a recorded design check

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T005 N/A — this feature adds no new command, subcommand, or flag; `arc lint`/`arc init`'s existing CLI surface is unchanged (plan.md Constitution Check)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T006 N/A — no new external system integration; `_schema/types/*.md` continues to be read via the existing `fsys.Store` port, unchanged (contracts/type-schema-document.md)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> No new command exists for this feature, so its E2E surface is the existing `arc init`/`arc lint` commands, exercised exactly as `cmd/arc/lint/lint_test.go`'s existing `initGraph`/`sut()` helpers already do.

- [X] T007 [P] [US1] Write E2E test(s) in `cmd/arc/ctrl/init_test.go` asserting `arc init` seeds `_schema/types/Node.md` (Required: `published`, `created`; Optional: `tags`, `text`, `updated`, `scoreZ`, `scoreC`) and that `source`/`entity`/`resource`/`timeline`'s seeded documents each carry an `rdfs:subClassOf:: [[Node]]` edge (spec Acceptance Scenario US1.1); tests MUST compile and fail (red)
- [X] T008 [P] [US1] Write E2E test(s) in `cmd/arc/lint/lint_test.go` asserting `arc lint` flags a `source` node missing `published` (now inherited-only via `Node`) exactly as it would a directly-required predicate, and passes when the node carries it (spec Acceptance Scenarios US1.2/US1.3); red phase
- [X] T009 [P] [US2] Write E2E test(s) in `cmd/arc/lint/lint_test.go`: author two independent custom base types (each requiring a distinct predicate) plus a third type `rdfs:subClassOf` both, then confirm `arc lint` enforces both inherited predicates on a node of the composed type, with no duplicate violation when the same predicate is required by two bases (spec Acceptance Scenarios US2.1-US2.3); red phase
- [X] T010 [P] [US3] Write E2E test(s) in `cmd/arc/lint/lint_test.go`: a three-level `rdfs:subClassOf` chain and a diamond-shaped hierarchy (two branches converging on one common ancestor), confirming the bottom type's effective contract includes the top ancestor's predicate exactly once (spec Acceptance Scenarios US3.1-US3.2); red phase
- [X] T011 [P] [US4] Write E2E test(s) in `cmd/arc/lint/lint_test.go`: a type declaring `rdfs:subClassOf` toward an unregistered type name fails clearly; two types declaring `rdfs:subClassOf` each other (including the direct self-reference case) fail clearly with no hang or crash (spec Acceptance Scenarios US4.1-US4.2); red phase

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T012 N/A — no new configuration values, environment variables, or secrets are introduced by this feature

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure — the `rdfs:subClassOf` resolution engine

**Purpose**: The recursive flattening resolver every user story depends on. Built once here; each user-story phase below only turns its own E2E test(s) green against this shared engine.

- [X] T013 [P] Register `rdfs:subClassOf` in `kernel.CorePredicateDefs` (`Role: "edge"`, `Merge: core.MergeUnion`, `Aligned: "rdfs:subClassOf"`, description per data-model.md) in `internal/app/schema/kernel/schema.go`
- [X] T014 [P] Add a `"Node"` entry to `kernel.CoreTypeDefs` (`Required: ["published", "created"]`, `Optional: ["tags", "text", "updated", "scoreZ", "scoreC"]`, `Merge: core.MergeUnion`) in `internal/app/schema/kernel/schema.go`
- [X] T015 Add a new `kernel.CoreTypeBases map[string][]string` (`"source"`, `"entity"`, `"resource"`, `"timeline"` each → `["Node"]`) and remove, from each of those four `CoreTypeDefs` entries, every `Required`/`Optional` predicate now supplied by `Node` (`tags`, `text`, `published`, `created`, `updated`, `scoreZ`, `scoreC` — never `indexed`/`mentions`/`mentionedIn`), per data-model.md's reshaped-types table, in `internal/app/schema/kernel/schema.go` (depends on: T014)
- [X] T016 [P] Add `ErrSchemaCycle` and `ErrSchemaUnresolvedBase` `faults` constants (naming the offending type, and type + unresolved reference, respectively) to `internal/app/schema/service/errors.go`, matching `ErrSchemaInvalid`'s existing pattern
- [X] T017 Extend `typeNode()` in `internal/app/schema/service/schema.go` to also append `core.Link{Predicate: "rdfs:subClassOf", Target: base}` edges from `kernel.CoreTypeBases[name]`, so `Seed()`'s rendered output includes them (depends on: T013, T015)
- [X] T018 Extend `decodeTypeDef` in `internal/app/schema/service/schema.go` to bucket `rdfs:subClassOf` edges into a new, unexported raw record alongside the existing `required`/`optional` buckets — `core.TypeDef`'s own public shape stays unchanged (depends on: T013)
- [X] T019 Implement the memoized recursive `resolve()` flattening pass in `internal/app/schema/service/schema.go` per data-model.md's pseudocode: union of own + all (transitively resolved) base types' `Required`/`Optional`, required-always-wins-over-optional, the implicit `Node` base added for every type except `Node`/`Property`/`Class`, active-recursion-stack cycle detection, unresolved-base-type detection (depends on: T018)
- [X] T020 Wire `resolve()` into `resolveTypes`/`Resolve` in `internal/app/schema/service/schema.go` so `core.Index.Types` always holds fully flattened `core.TypeDef`s, returning `ErrSchemaCycle`/`ErrSchemaUnresolvedBase` (via `faults`) the moment either condition is detected, before any other schema-dependent work proceeds (depends on: T016, T019)
- [X] T021 [P] Unit tests (red→green) for the resolver in `internal/app/schema/service/schema_test.go`: single inheritance (no own predicates), own + inherited combined, multiple bases with overlapping predicates (dedup), three-level chain, diamond hierarchy, direct self-reference cycle, longer cycle, unresolved base-type reference, required-wins-over-optional (depends on: T020)

**Checkpoint**: Foundation ready — every user story phase below only needs to turn its own E2E test green.

---

## Phase 3: User Story 1 - Reuse a shared base type instead of redeclaring its contract (Priority: P1) 🎯 MVP

**Goal**: A freshly initialized graph's `source`/`entity`/`resource`/`timeline` types genuinely inherit `Node`'s contract, whether declared explicitly or not, and `arc lint` enforces the inherited predicates exactly as it would directly declared ones.

**Independent Test**: `arc init`, then author a `source` node missing `published`; `arc lint` reports it missing exactly as it would today for a directly-required predicate.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T007, T008) and MUST currently be failing (red).

- [X] T022 [P] [US1] Update `internal/app/schema/kernel/schema_test.go`'s existing assertions (`TestCoreTypeDefsContainsCoreTypesAndSchemaTypesThemselves`, `TestCoreTypeDefsRequiredListsMatchCoreSection11`, `TestCoreTypeDefsOptionalListsIncludeCrossCuttingPredicates`) for 7 registered types (adds `Node`) and the reshaped `source`/`entity`/`resource`/`timeline` `Required`/`Optional` lists (depends on: T014, T015)
- [X] T023 [P] [US1] Update `internal/app/schema/service/schema_test.go`'s `Seed()` shape assertions (predicate/type counts, per-type round-trip checks) for the new predicate and type, and the new `rdfs:subClassOf` edges on the four seeded content types (depends on: T017)
- [X] T024 [US1] Review `internal/app/lint/service/lint_test.go`'s hand-built `core.Index{Types: {...}}` fixture (it currently copies `kernel.CoreTypeDefs["source"]` etc. directly, bypassing `Resolve`'s flattening) and update it to include `"Node"` and reflect each referenced type's effective (flattened) contract wherever the reshaped `Required`/`Optional` lists change which of that test's existing assertions hold (depends on: T014, T015)
- [X] T025 [US1] Turn T007 and T008's E2E tests green (depends on: T020, T022, T023, T024)

**Checkpoint**: At this point, User Story 1's E2E tests pass and the story is fully functional and testable independently.

---

## Phase 4: User Story 2 - Combine several base types into one composed type (Priority: P1)

**Goal**: A type declared `rdfs:subClassOf` more than one base type inherits the union of every base's contract, with no duplicate reporting when bases overlap.

**Independent Test**: Two custom base types each requiring a distinct predicate; a third type `rdfs:subClassOf` both; `arc lint` enforces both inherited predicates on a node of the composed type.

### Implementation for User Story 2

> E2E test for this story was already written in Phase 2d (T009) and MUST currently be failing (red).

- [X] T026 [US2] Turn T009's E2E test green (depends on: T020) — this story's behavior is already implemented by Phase 2.5's resolver; no new production code is expected here beyond confirming the fixture passes

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently.

---

## Phase 5: User Story 3 - Inheritance chains resolve correctly across multiple levels (Priority: P2)

**Goal**: A hierarchy more than one level deep (including a diamond shape) flattens correctly, with a common ancestor's predicates counted exactly once regardless of how many paths reach it.

**Independent Test**: A three-level chain where each level requires a distinct predicate; the bottom type's effective contract requires all three.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T010) and MUST currently be failing (red).

- [X] T027 [US3] Turn T010's E2E test green (depends on: T020) — proves the resolver's memoization/dedup handles depth and diamond shapes correctly; no new production code expected beyond confirming the fixture passes

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently.

---

## Phase 6: User Story 4 - Malformed base-type declarations are caught, not silently ignored (Priority: P2)

**Goal**: An unresolved `rdfs:subClassOf` target, or a cycle of any length, fails schema loading clearly — never a hang, crash, or silent pass.

**Independent Test**: A type `rdfs:subClassOf` a nonexistent type name, and two types `rdfs:subClassOf` each other; both are reported clearly.

### Implementation for User Story 4

> E2E test for this story was already written in Phase 2d (T011) and MUST currently be failing (red).

- [X] T028 [US4] Turn T011's E2E test green (depends on: T020); if `cmd/arc/lint/lint.go` currently swallows or generically wraps errors from `appschema.Resolve` in a way that hides `ErrSchemaCycle`/`ErrSchemaUnresolvedBase`'s specific message, adjust its error surfacing minimally so the underlying message reaches the user — read the command's current error handling first and change only if needed

**Checkpoint**: All four user stories pass their E2E tests independently.

---

## Additional Polish

- [X] T029 [P] Update [quickstart.md](quickstart.md) if any step's exact command or expected output drifted during implementation
- [X] T030 [P] Review `internal/app/schema/README.md` and update it to document `rdfs:subClassOf`/`Node`, if that README describes the on-disk schema format (as `internal/app/lint/README.md` does for lint)
- [X] T031 Run `staticcheck ./internal/app/schema/... ./internal/app/lint/... ./cmd/arc/lint/... ./cmd/arc/ctrl/...` and `go test ./...` for the full repository, confirming zero regressions outside this feature's own updated fixtures
- [X] T032 Manually run quickstart.md's 5-step validation walkthrough against a locally built `arc` binary

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — none expected beyond the T003 Glossary addition; this feature introduces no new package boundary or ADR-worthy pattern
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — `rdfs:subClassOf`, `Node`, effective/inherited contract (T003)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX) — N/A, no command/flag changed (T005)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — assess whether the `rdfs:subClassOf`-as-dedicated-edge convention and the `Node` universal-base convention warrant a follow-up ADR (research.md D1/D5); not required to ship, but should be a deliberate yes/no, not an omission
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — confirm no `cobra` import entered `internal/app/schema`
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — T021 before T013-T020's implementation is complete
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A, no new terminal output; any error message change (T028) reuses `arc lint`'s existing output path
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — N/A, no command changed
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII) — T025-T028
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — T007-T011 cover US1-US4's acceptance scenarios
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — no CLI surface changed, but existing graphs may newly fail `arc lint` (`entity`/`resource` gain required `published`/`created`; `timeline` gains required `published` for the first time, per research.md D5's flagged consequence) — assess whether this data-contract tightening itself warrants a minor/major version note in the changelog even though no flag or command changed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion — BLOCKS all user stories (the shared resolver)
- **User Stories (Phase 3-6)**: All depend on Phase 2.5; US1 (Phase 3) should complete first since it establishes the reshaped seed data (T014/T015) other stories' fixtures build on top of, but US2-US4 (Phases 4-6) do not depend on US1's own E2E tests passing, only on Phase 2.5's resolver
- **Additional Polish**: Depends on all four user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### Within Foundational Infrastructure

- T013 (predicate) and T014 (Node type) are independent, both before T015 (reshape the four content types, needs T014's Node contract to know what to remove)
- T016 (errors) is independent of T013-T015
- T017 (Seed edge rendering) needs T013 + T015
- T018 (raw decode) needs T013
- T019 (resolve algorithm) needs T018
- T020 (wire into Resolve) needs T016 + T019
- T021 (resolver unit tests) needs T020

### Parallel Opportunities

- T013, T014, T016 can run in parallel (different concerns, same file for T013/T014 but non-overlapping map entries — coordinate if working concurrently)
- T007-T011 (all five E2E test-writing tasks, Phase 2d) can run in parallel — different test functions, largely independent fixtures
- Once Phase 2.5 completes, Phases 4, 5, 6 (US2, US3, US4) can proceed in parallel with each other and do not need to wait for Phase 3 (US1)'s own fixture-migration tasks (T022-T024), only for Phase 2.5's resolver (T020)

---

## Parallel Example: Foundational Infrastructure

```bash
# Launch independent foundational tasks together:
Task: "Register rdfs:subClassOf in kernel.CorePredicateDefs in internal/app/schema/kernel/schema.go"
Task: "Add Node entry to kernel.CoreTypeDefs in internal/app/schema/kernel/schema.go"
Task: "Add ErrSchemaCycle/ErrSchemaUnresolvedBase to internal/app/schema/service/errors.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (the resolver — this is most of the real implementation work)
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently via quickstart.md steps 1-2
7. Deploy/demo if ready

### Incremental Delivery

1. Setup + Design Preconditions + Foundational Infrastructure → resolver ready, fully unit-tested
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP — the seeded graph's own types demonstrate inheritance)
3. Add User Story 2 → multi-base composition confirmed
4. Add User Story 3 → depth/diamond hierarchies confirmed
5. Add User Story 4 → malformed-hierarchy safety confirmed
6. Each story adds confidence without breaking previous stories — the resolver itself does not change shape after Phase 2.5

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- This feature's stories are unusually convergent on one shared piece of logic (Phase 2.5's resolver) rather than each needing distinct new production code — US2-US4's own phases are intentionally thin verification-only phases, not because the stories are unimportant, but because a correct resolver (built once, tested against all four stories' scenarios) satisfies all of them simultaneously
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted
- research.md D5's flagged consequence (timeline gaining required `published`) should be re-confirmed as intentional before T015/T020 ship — it is implemented exactly as specified, but is the one behavior change in this feature most likely to surprise an existing graph's maintainer
