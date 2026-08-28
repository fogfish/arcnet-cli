---

description: "Task list template for feature implementation"
---

# Tasks: MCP `node_match` Filter Tool

**Input**: Design documents from `/specs/028-node-match-filter/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md) (design decisions D1-D6), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Unit and E2E acceptance tests are NOT optional for this project (constitution Principles VI, VIII). Every spec.md acceptance scenario maps 1:1 to an E2E test; tests are written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Task IDs run sequentially across the whole file (T001, T002, ...); `TN0x` IDs in Phase N are the constitution's fixed verification checklist.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- `internal/core/filter.go` — `core.Fact` and `Filter.MatchingFacts` (no `cobra`/`mcp` import, Principle III)
- `internal/app/graph/kernel/match.go` — `MatchEntry`/`MatchResult` DTOs
- `internal/app/graph/service/match.go` — `Match` use-case function
- `internal/app/graph/component.go` — thin `Match` delegator
- `cmd/arc/graph/serve.go` — `nodeMatchArgs`/`nodeMatchHandler`/`renderFactTable`, MCP tool registration
- Colocated `*_test.go` files hold both unit tests (`internal/core`, `internal/app/graph/service`) and E2E tests (`cmd/arc/graph/serve_test.go`)
- No `cmd/<command>_test.go` Cobra E2E tests this release — no new Cobra command is introduced (plan.md D6)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the branch is ready for feature work. No new package, no new Cobra command, no new dependency (plan.md Technical Context) — this phase is intentionally thin.

- [X] T001 Confirm `go build ./...` and `staticcheck ./...` are clean on `028-node-match-filter` at the branch tip before starting design-precondition work — `go build ./...` and `go vet ./...` are clean; `staticcheck` itself cannot run in this environment (Go export-data version mismatch unrelated to this branch), recorded rather than silently skipped
- [X] T002 [P] Confirm no new entry is needed in `go.mod` (plan.md: no new dependency) — record this explicitly rather than silently skipping the check: `git status go.mod go.sum` is clean, no new dependency added

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate; most of the actual design decisions were already recorded in `research.md`/`data-model.md`/`contracts/` during `/speckit-plan` — these tasks finalize/verify them and produce the one remaining artifact (E2E/unit test files), not re-derive the decisions.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T003 Verify `core.Fact` (data-model.md) does not duplicate any existing `internal/core` type before introducing it — `grep -rn "type Fact" internal/` found no existing type
- [X] T004 [P] Add a "Fact" row to `ARCHITECTURE.md`'s Glossary describing `{Property, Value string}` and its relationship to `Filter`/`Statement`/`Matcher` (data-model.md)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T005 N/A — no new Cobra command/flag surface is introduced this release (plan.md Constitution Check marks Principle IX N/A; the CLI surface is explicitly deferred to a future release per user instruction); record this explicitly rather than skipping silently
- [X] T006 [P] Review [contracts/mcp-contract.md](contracts/mcp-contract.md)'s `node_match` input/output shape against `mcpFilter`'s current fields in `cmd/arc/graph/serve.go` and confirm no new wire type is needed for `statements` (reuse only, per research.md D1's scope) — confirmed: `mcpFilter`/`mcpStatement`/`stringOrArray`/`toCoreFilter`/`inputSchemaFor` are reused verbatim, no changes needed

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T007 N/A — this feature adds no new external system, no new adapter (plan.md Constitution Check); confirm no `internal/adapter/fsys` change is needed and record that explicitly rather than skipping this subsection silently — confirmed: `service.Match` reuses `fsys.Mounter`/`walkNodeFiles`/`readGrepNode` unchanged

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> Every task below is written to compile and fail semantically (red) before its story's implementation phase begins.

- [X] T008 [P] [US1] Write E2E tests in `cmd/arc/graph/serve_test.go` for User Story 1's Acceptance Scenarios 1-2 (spec.md): `node_match` with a type-constraining statement returns one row per matching node shaped `{id, property: "type", value: "Source"}`; a filter matching zero nodes returns an empty (header-only) table, not an error — via `connectServeSession`, reusing an existing `testdata` fixture graph
- [X] T009 [P] [US2] Write E2E tests in `cmd/arc/graph/serve_test.go` for User Story 2's Acceptance Scenarios 1-2: a filter combining a type statement and a tag statement returns two distinct rows (one per satisfying fact) for a node satisfying both; a node whose array-valued `tags` attribute has two elements each matching the same statement produces two separate rows for that node
- [X] T010 [P] [US3] Write E2E tests in `cmd/arc/graph/serve_test.go` for User Story 3's Acceptance Scenario 1: a predicate-only statement (e.g. `predicate: "cites"`) returns one row per citing node, `property` equal to `cites`, `value` equal to the cited node's id
- [X] T011 [P] Write an E2E test in `cmd/arc/graph/serve_test.go` for the spec's empty-filter Edge Case: an empty/missing `filter` returns a validation error (`service.ErrEmptyFilter`), never an empty table and never every node's facts
- [X] T012 [P] Write table-driven unit tests in `internal/core/filter_test.go` for `Filter.MatchingFacts` (`github.com/fogfish/it/v2`): a type-fact match, an attribute-fact match, a multi-statement node producing two distinct facts, an array-valued attribute producing one entry per matching element, two statements independently satisfied by the exact same fact collapsing to one entry, zero statements/zero matches yielding `nil` — written to fail (red) against the not-yet-implemented `MatchingFacts`
- [X] T013 [P] Write unit tests in `internal/app/graph/service/match_test.go` for `service.Match`: an empty filter returns `ErrEmptyFilter` before any node file is opened, a filter matching zero nodes returns an empty `Matches` slice, an unreadable node file is recorded in `Unreadable` and excluded (mirrors `service.Grep`'s existing precedent) — written to fail (red) against the not-yet-implemented `Match`

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T014 N/A — no new configuration value, no secret handling introduced by this feature; record explicitly

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The shared plumbing every user story's E2E tests need before any of them can turn green — the new domain type, kernel DTOs, the `service.Match` skeleton (enumeration plus the empty-filter guard), the delegator, and MCP tool registration. `Filter.MatchingFacts`'s type-fact and attribute-fact coverage is delivered here since User Story 1 needs it immediately; edge-fact coverage is deliberately deferred to User Story 3 (Phase 5), mirroring `specs/027-triple-filter-model`'s own precedent of landing attribute-fact matching before edge-fact matching as separate, individually-verifiable steps.

- [X] T015 [P] Add `core.Fact{Property, Value string}` to `internal/core/filter.go` (data-model.md)
- [X] T016 Implement `Filter.MatchingFacts(node Node) []Fact` in `internal/core/filter.go` covering the synthesized type fact and attribute facts only (mirrors `statementSatisfiedBy`'s first two fact sources) — not yet `node.Edges` (that is US3, Phase 5) — deduplicated by `(Property, Value)` and sorted for deterministic output, until T012's non-edge cases pass (depends on T015)
- [X] T017 [P] Add `ErrEmptyFilter` sentinel to `internal/app/graph/service/errors.go` (research.md D2)
- [X] T018 [P] Add `kernel.MatchEntry`/`kernel.MatchResult` to `internal/app/graph/kernel/match.go` (data-model.md)
- [X] T019 Implement `service.Match` in `internal/app/graph/service/match.go`: reject an empty/zero-statement filter via `ErrEmptyFilter` before any node file is opened; reuse `walkNodeFiles`/`readGrepNode`/`guardIsGraph` from `grep.go`/`apply.go` unchanged; keep nodes where `filter.Match(node)`; map `filter.MatchingFacts(node)` into `kernel.MatchEntry{ID, Property, Value}` per kept node; record unreadable files in `Unreadable` (depends on T016, T017, T018)
- [X] T020 [P] Add a `Match` delegator to `internal/app/graph/component.go`, mirroring `Grep`/`Subgraph`/`ContextRetrieve`'s existing one-line delegation shape (depends on T019)
- [X] T021 Add `nodeMatchArgs{Filter mcpFilter}` (required, non-pointer — unlike the other tools' optional `*mcpFilter`), `nodeMatchHandler`, and `renderFactTable` to `cmd/arc/graph/serve.go`, reusing `mcpFilter`/`toCoreFilter`/`inputSchemaFor` verbatim (depends on T020)
- [X] T022 Register `node_match` via `mcp.AddTool` in `buildServer` (`cmd/arc/graph/serve.go`); update `schemaAdvertisement` and `NewServeCmd`'s `Long` text to mention `node_match` as a fifth, MCP-only, read-only tool with a required filter (depends on T021)

**Checkpoint**: Foundation ready — attribute/type-based matching works end-to-end over the real MCP transport; US1's E2E tests can now turn green. US3's edge-fact scenario (T010) remains red until Phase 5.

---

## Phase 3: User Story 1 - Find nodes by structured criteria without paying for full content (Priority: P1) 🎯 MVP

**Goal**: An agent can call `node_match` with an attribute/type-constraining filter and get back exactly the facts that justified each match, without fetching each candidate node's full content.

**Independent Test**: T008, already red from Phase 2d — run and confirm green after this phase (no new production code expected beyond Phase 2.5; if red, the gap is in T016/T019, not a missing feature).

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T008) and MUST currently be failing (red). Phase 2.5 already implements everything this story needs; this phase closes the loop.

- [X] T023 [US1] Run T008 and confirm green; if red, fix the smallest possible gap in T016/T019 rather than adding new capability (depends on T016, T019, T022) — `TestServeNodeMatchReturnsOneRowPerMatchingFact`/`TestServeNodeMatchEmptyResultForNoMatches` both PASS
- [X] T024 [P] [US1] Run T011 (empty-filter validation) and confirm green (depends on T019, T022) — `TestServeNodeMatchEmptyFilterReturnsValidationError` PASS
- [X] T025 [P] [US1] Run T012's non-edge unit test cases and confirm green (depends on T016) — all non-edge `TestMatchingFacts*` cases PASS

**Checkpoint**: At this point, User Story 1's E2E tests (T008, T011) pass and `node_match`'s core lookup capability is functional and independently testable via MCP.

---

## Phase 4: User Story 2 - See exactly which fact caused a multi-condition match (Priority: P2)

**Goal**: `node_match` reports one entry per distinct satisfying fact — never collapsing genuinely different facts into one row, and never duplicating the same fact when two statements independently agree on it.

**Independent Test**: T009, already red from Phase 2d. Independent of User Story 3 — no dependency on edge-fact matching.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red).

- [X] T026 [US2] Run T009 and confirm green (depends on T016, T019) — no new production code is expected: T016's per-statement fact walk and its `(Property, Value)`-keyed dedup already produce this behavior; if red, the gap is in T016's dedup/ordering logic, not a missing feature — `TestServeNodeMatchMultiStatementReportsBothFacts`/`TestServeNodeMatchArrayAttributeReportsEachElement` both PASS
- [X] T027 [P] [US2] Run T012's multi-statement/array-element/dedup-collapse unit test cases and confirm green (depends on T016) — `TestMatchingFactsMultiStatementProducesTwoDistinctFacts`/`TestMatchingFactsArrayValuedAttributeOneEntryPerMatchingElement`/`TestMatchingFactsTwoStatementsSameFactCollapseToOneEntry` all PASS

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently; multi-fact evidence reporting is proven correct, independent of User Story 3's edge-fact capability.

---

## Phase 5: User Story 3 - Discover nodes by relation without knowing the source (Priority: P3)

**Goal**: A predicate-only filter statement matches nodes via their outgoing edges, not only their attributes — `node_match` becomes usable as a relation-discovery tool, not only an attribute-lookup tool.

**Independent Test**: T010, remains red until this phase.

### Implementation for User Story 3

> The E2E test for this story was already written in Phase 2d (T010) and MUST currently be failing (red) — Phase 2.5 deliberately did not implement edge-fact coverage.

- [X] T028 [US3] Extend `Filter.MatchingFacts` in `internal/core/filter.go` to additionally walk `node.Edges` as `(node.ID, edge.Predicate, edge.Target)` facts, exactly mirroring `statementSatisfiedBy`'s third fact source (depends on T016)
- [X] T029 [P] [US3] Add the edge-fact unit test cases from T012 to `internal/core/filter_test.go` — a node matched purely via an outgoing edge, a node matched via both an attribute and an edge from two different statements — and confirm they turn green (depends on T028) — `TestMatchingFactsEdgeFactAlone`/`TestMatchingFactsAttributeAndEdgeFromDifferentStatements` both PASS (written alongside T012, confirmed red before T028, green after)
- [X] T030 [US3] Run T010 and confirm green (depends on T028) — `TestServeNodeMatchPredicateOnlyStatementMatchesEdgeFact` PASS (after fixing a test-assertion bug of its own: it was checking for substring "| paper-b |", which also occurs as the matched row's *value* column, not just an id column — narrowed to a "\n| paper-b |" row-start check)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests; `node_match`'s full fact vocabulary (type, attribute, relation) is complete.

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories or are explicitly called for by spec.md/plan.md but not gated on any single story's E2E tests.

- [X] T031 [P] Check off `node_match(filter)` in `specs/VISION.md`'s Phase 5 — MCP Server roadmap checklist as implemented
- [X] T032 [P] Run [quickstart.md](quickstart.md)'s manual validation scenarios end-to-end against a real built binary — spawned a real `go build`-produced `arc serve` binary as a subprocess over stdio (`mcp.CommandTransport`) against a fixture graph with two Source nodes and a `cites` edge: a `{predicate: type, target: Source}` filter returned the expected two-row markdown table, and an empty-statements filter returned the expected `filter must contain at least one statement` tool error while the process kept running
- [X] T033 Update `buildServer`'s GoDoc comment (currently lists "node_get/node_grep/subgraph_get/context_retrieve") and `NewServeCmd`'s `Long` text in `cmd/arc/graph/serve.go` to include `node_match`

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase is retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the "Fact" addition (Principle I) — depends on T004
- [X] TN02 Domain concept (`Fact`) added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — depends on T004
- [X] TN03 N/A — no command/flag surface is introduced this release (Principle IX); confirmed by T005

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced that would require a new ADR (Principle I) — confirmed: `nodeMatchHandler` remains a thin wrapper (ADR 003 rule 1 unaffected)
- [X] TN05 Domain logic (`core.Fact`/`Filter.MatchingFacts`, `service.Match`) uses no `cobra`/`mcp` import; Cobra/MCP wiring stays confined to `cmd/arc/graph/serve.go` (Principle III) — confirmed via grep
- [X] TN06 T012/T013's unit tests and T008-T011's E2E tests were written first, compiled, and failed semantically before their corresponding implementation tasks (Principle VI) — confirmed red (`f.MatchingFacts undefined`/`undefined: service.Match`/`unknown tool "node_match"`) before Phase 2.5 implementation
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI) — confirmed via grep
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI) — quickstart.md's shell commands are a manual/smoke-test guide, not a substitute for `go test`
- [X] TN09 No new external integration was added; no vendor SDK type leaks through `service.Match`'s signature (Principle VII) — signature is `(context.Context, fsys.Mounter, core.Filter, string) (kernel.MatchResult, error)`
- [X] TN10 N/A for this feature — no terminal output/styling change (Principle X)
- [X] TN11 N/A for this feature — no new configuration value or secret (Principle XI) — confirmed by T014
- [X] TN12 N/A — no new command help text is required this release; `NewServeCmd`'s existing `Long` text is updated only to mention `node_match` (Principle XII) — depends on T022, T033
- [X] TN13 E2E tests from Phase 2d (T008-T011) turned GREEN and changed minimally during implementation (Principle VIII) — depends on T023, T024, T026, T030 — only one test-side fix was needed (a substring-collision assertion bug in T010's own test, not production code)
- [X] TN14 All spec.md scenarios across User Stories 1-3, plus the empty-filter Edge Case, have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `node_match` is a wholly new, additive MCP tool — no existing tool name, flag, or output schema changes, so no major version bump is required (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; 2a-2e proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion; BLOCKS every user story phase
- **User Stories (Phase 3, 4, 5)**: All depend on Phase 2.5
  - US1 (Phase 3) and US2 (Phase 4) require only Phase 2.5's attribute/type-fact coverage and may proceed in parallel
  - US3 (Phase 5) is the one story that adds genuinely new production code (edge-fact coverage) rather than only verifying Phase 2.5's output
- **Additional Polish**: Depends on US1, US2, US3 all being complete
- **Phase N (Constitution Compliance Verification)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 — no dependency on US2/US3
- **User Story 2 (P2)**: Depends on Phase 2.5 — no dependency on US1/US3; verifies behavior Phase 2.5 already implements
- **User Story 3 (P3)**: Depends on Phase 2.5; adds `Filter.MatchingFacts`'s edge-fact walk (T028), independent of US1/US2's own tasks

### Parallel Opportunities

- T008-T013 (Phase 2d, all different files) can be written in parallel
- Once Phase 2.5 completes, Phase 3 (US1) and Phase 4 (US2) can proceed in parallel; Phase 5 (US3) can start in parallel too, since T028 touches `filter.go` in a way that does not conflict with US1/US2's (test-only) tasks
- T031-T033 (Polish) are independent of each other

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2d, already complete before this point):
# T008 E2E tests for US1 in cmd/arc/graph/serve_test.go

# Foundational work these tests need (Phase 2.5, already complete before this point):
# T016 Filter.MatchingFacts (type + attribute facts)
# T019 service.Match

# Confirm User Story 1 independently:
Task: "Run T008 and confirm green"
Task: "Run T011 (empty-filter validation) and confirm green"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (`Fact`, `MatchingFacts` attribute/type coverage, `service.Match`, MCP wiring)
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification (subset applicable so far)
6. **STOP and VALIDATE**: Test User Story 1 independently via quickstart.md's manual scenarios
7. Deploy/demo if ready

### Incremental Delivery

1. Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify → Deploy/Demo
4. Add User Story 3 (edge-fact matching) → Verify against Phase N → Deploy/Demo
5. Polish (VISION.md, `serve.go` doc comments, quickstart validation)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Foundational Infrastructure together (single ownership recommended for `filter.go`/`serve.go`, given every later phase builds on them)
2. Once Phase 2.5 completes:
   - Developer A: User Story 1 (verification only)
   - Developer B: User Story 2 (verification only)
   - Developer C: User Story 3 (`Filter.MatchingFacts`'s edge-fact walk, T028)
3. All three stories integrate and run Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- E2E tests (Phase 2d: T008-T011) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements
- User Stories 1 and 2 are this feature's "mostly verification" stories — Phase 2.5 already implements what they need; User Story 3 is the one story that adds genuinely new production code (edge-fact matching), matching `specs/027-triple-filter-model`'s own precedent of separating attribute-fact and edge-fact matching into distinct, individually-verifiable steps
