---

description: "Task list template for feature implementation"
---

# Tasks: MCP `node_links` and `node_backlinks` Tools

**Input**: Design documents from `/specs/030-mcp-node-links-backlinks/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md) (design decisions D1-D6), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Unit and E2E acceptance tests are NOT optional for this project (constitution Principles VI, VIII). Every spec.md acceptance scenario maps 1:1 to an E2E test; tests are written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Task IDs run sequentially across the whole file (T001, T002, ...); `TN0x` IDs in Phase N are the constitution's fixed verification checklist.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, or independent additions to the same test file, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- `internal/app/graph/kernel/links.go` — `BacklinkEntry` DTO (new)
- `internal/app/graph/service/links.go` — `nodeRelations`, `buildRelationReverseIndex`, `NodeLinks`, `NodeBacklinks` (new; reuses `enumerateNodes`/`guardIsGraph` from `subgraph.go`/`apply.go`, `ErrSeedNotFound` from `errors.go`, all unchanged)
- `internal/app/graph/service/links_test.go` — unit tests (new)
- `internal/app/graph/component.go` — `NodeLinks`/`NodeBacklinks` thin delegators (existing file, additive)
- `cmd/arc/graph/serve_tool_node_links.go` / `serve_tool_node_backlinks.go` — MCP tool registration, handlers, table renderers (new, per-tool-file convention from spec 029)
- `cmd/arc/graph/serve.go` — two `mcp.AddTool` registrations, `sessionInstructionsPurpose`/`sessionInstructions()`, `NewServeCmd`'s `Long` text (existing file, additive)
- `cmd/arc/graph/serve_test.go` — E2E tests (existing file, additive)
- No `cmd/<command>_test.go` Cobra E2E tests this release — no new Cobra command is introduced (plan.md D6)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the branch is ready for feature work. No new package, no new Cobra command, no new dependency (plan.md Technical Context) — this phase is intentionally thin.

- [X] T001 Confirm `go build ./...` and `staticcheck ./...` are clean on `030-mcp-node-links-backlinks` at the branch tip before starting design-precondition work
- [X] T002 [P] Confirm no new entry is needed in `go.mod` (plan.md: no new dependency) — record this explicitly: `git status go.mod go.sum` is clean

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Most design decisions were already recorded in `research.md`/`data-model.md`/`contracts/` during `/speckit-plan`; these tasks finalize/verify them and produce the remaining artifact (E2E/unit test files), not re-derive the decisions.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T003 Verify `kernel.BacklinkEntry` (data-model.md) does not duplicate any existing type in `internal/app/graph/kernel/` before introducing it — `grep -rn "type BacklinkEntry" internal/` must find nothing
- [X] T004 [P] Add a short note to `ARCHITECTURE.md` explaining that `node_links`/`node_backlinks` deliberately report a node's inline prose `HRefs` alongside its structural `Edges`, unlike `Subgraph`'s BFS traversal (`nodeTargets`/`buildReverseIndex`), which stays Edges-only by design — so a future reader of `nodeTargets`'s "HRefs are never navigable" comment does not assume that holds everywhere (plan.md Constitution Check, Principle I)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T005 N/A — no new Cobra command/flag surface is introduced this release (plan.md Constitution Check marks Principle IX N/A this release); record this explicitly
- [X] T006 [P] Review [contracts/mcp-contract.md](contracts/mcp-contract.md)'s `node_links`/`node_backlinks` input shape against `nodeGetArgs`'s existing single-`id`-field convention in `cmd/arc/graph/serve_tool_node_get.go` and confirm no new wire type is needed beyond `nodeLinksArgs`/`nodeBacklinksArgs` (data-model.md)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T007 N/A — this feature adds no new external system, no new adapter (plan.md Constitution Check); confirm `NodeLinks`/`NodeBacklinks` will reuse `enumerateNodes`/`guardIsGraph`/`fsys.Mounter` unchanged and record that explicitly

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> Every task below is written to compile and fail semantically (red) before its story's implementation phase begins.

- [X] T008 [P] [US1] Write E2E test in `cmd/arc/graph/serve_test.go` for User Story 1 Acceptance Scenario 1 (spec.md): `node_links` on a node with several outgoing relations (structural edges and/or inline hrefs) returns one markdown table row per relation shaped `{predicate, target}`, matching the node's own relations exactly — via `connectServeSession`, reusing an existing `testdata` fixture graph
- [X] T009 [P] [US1] Write E2E test in `cmd/arc/graph/serve_test.go` for User Story 1 Acceptance Scenario 2: `node_links` on a node with zero outgoing relations returns a header-only table, not an error
- [X] T010 [P] [US2] Write E2E test in `cmd/arc/graph/serve_test.go` for User Story 2 Acceptance Scenario 1: `node_backlinks` on a node referenced by several other nodes returns one row per incoming relation shaped `{source, predicate}`
- [X] T011 [P] [US2] Write E2E test in `cmd/arc/graph/serve_test.go` for User Story 2 Acceptance Scenario 2: `node_backlinks` on a node no other node references returns a header-only table, not an error
- [X] T012 [P] [US3] Write E2E test in `cmd/arc/graph/serve_test.go` for User Story 3 Acceptance Scenario 1: for a node with both outgoing and incoming relations, every relation where it is the source appears only in its `node_links` result and every relation where it is the target appears only in its `node_backlinks` result — no relation missing from either, none duplicated across both
- [X] T013 [P] Write E2E test in `cmd/arc/graph/serve_test.go` for the spec's unknown-id Edge Case, `node_links`: an id matching no node returns a tool error naming `no node found with basename <id>` (`service.ErrSeedNotFound`), never an empty table
- [X] T014 [P] Write E2E test in `cmd/arc/graph/serve_test.go` for the spec's unknown-id Edge Case, `node_backlinks`: same error behavior as T013, reverse direction
- [X] T015 [P] Write E2E test in `cmd/arc/graph/serve_test.go` for the spec's self-reference Edge Case: a node with a relation whose target is itself produces one ordinary row in its own `node_links` result (`target` = its own id) and one ordinary row in its own `node_backlinks` result (`source` = its own id)
- [X] T016 [P] Write E2E test in `cmd/arc/graph/serve_test.go` for the spec's duplicate-relations Edge Case: two relations between the same pair of nodes (repeated predicate, or two different predicates) each produce their own row in `node_links`/`node_backlinks`, never collapsed into one
- [X] T017 [P] Write E2E test in `cmd/arc/graph/serve_test.go` for the spec's bare-href Edge Case: an inline prose reference with no explicit predicate produces one `node_links` row with an empty `predicate` cell, not a dropped entry
- [X] T018 [P] Write table-driven unit tests in `internal/app/graph/service/links_test.go` for `NodeLinks` (`github.com/fogfish/it/v2`): a node with mixed Edges+HRefs returns their combination in `nodeRelations` order, a node with zero relations returns an empty slice, an unknown id returns `ErrSeedNotFound` — written to fail (red) against the not-yet-implemented `NodeLinks`
- [X] T019 [P] Write table-driven unit tests in `internal/app/graph/service/links_test.go` for `NodeBacklinks`: a target referenced by several nodes' Edges/HRefs returns one `BacklinkEntry` per referencing relation, a target with zero referencing relations returns an empty slice, an unknown id returns `ErrSeedNotFound`, a self-referencing node appears in its own result — written to fail (red) against the not-yet-implemented `NodeBacklinks`/`buildRelationReverseIndex`

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T020 N/A — no new configuration value, no secret handling introduced by this feature; record explicitly

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The one piece of shared plumbing both `node_links` and `node_backlinks` need before either can be implemented — the combined-relation helper and the new kernel DTO. Deliberately minimal: unlike spec 028, `node_links` and `node_backlinks` are two independent capabilities, not one shared matcher with two views onto it, so each gets its own implementation work in its own user-story phase (Phase 3, Phase 4) rather than being finished here.

- [X] T021 [P] Add `kernel.BacklinkEntry{Source, Predicate string}` to `internal/app/graph/kernel/links.go` (data-model.md)
- [X] T022 [P] Implement `nodeRelations(n core.Node) []core.Link` in `internal/app/graph/service/links.go` — `n.Edges` followed by `n.HRefs` (data-model.md); does NOT modify `nodeTargets`/`admittedEdges`/`buildReverseIndex` in `subgraph.go`, which remain Edges-only and unchanged (research.md D3)

**Checkpoint**: Foundation ready — `node_links`/`node_backlinks` implementation can now proceed.

---

## Phase 3: User Story 1 - See what a node points to without fetching its full content (Priority: P1) 🎯 MVP

**Goal**: An agent can call `node_links` with a node id and get back exactly that node's outgoing relations — structural edges and inline prose references alike — without fetching its full content.

**Independent Test**: T008/T009, already red from Phase 2d — run and confirm green after this phase.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T008, T009) and MUST currently be failing (red).

- [X] T023 [US1] Implement `NodeLinks(ctx, mounter, dir, id) ([]core.Link, error)` in `internal/app/graph/service/links.go`: reuse `enumerateNodes`/`guardIsGraph` unchanged, look up `id`, return `ErrSeedNotFound` on a miss, otherwise return `nodeRelations(node)` (depends on T022)
- [X] T024 [US1] Add a `NodeLinks` delegator to `internal/app/graph/component.go`, mirroring `NodeGet`/`Match`'s existing one-line delegation shape (depends on T023)
- [X] T025 [US1] Create `cmd/arc/graph/serve_tool_node_links.go`: `nodeLinksTool`, `nodeLinksArgs{ID string}`, `nodeLinksInputSchema` (reusing `inputSchemaFor`/`withExamples`), `nodeLinksWorkflowNote`, `nodeLinksHandler`, and `renderLinkTable([]core.Link) string` (header `| predicate | target |`, header-only when empty) — mirrors `serve_tool_node_get.go`'s shape, `renderFactTable`'s table-rendering convention (depends on T024)
- [X] T026 [US1] Register `node_links` via `mcp.AddTool` in `cmd/arc/graph/serve.go`'s `buildServer`; update `sessionInstructionsPurpose` ("six tools" → "eight tools", name added) and `sessionInstructions()` to append `nodeLinksWorkflowNote`; update `NewServeCmd`'s `Long` text to mention `node_links` (depends on T025)
- [X] T027 [P] [US1] Run T008/T009 and confirm green (depends on T026)
- [X] T028 [P] [US1] Run T018's `NodeLinks` unit test cases and confirm green (depends on T023)
- [X] T029 [P] [US1] Run T013 (unknown-id edge case, `node_links`) and confirm green (depends on T026)
- [X] T030 [P] [US1] Run T017 (bare-href edge case) and confirm green (depends on T026)

**Checkpoint**: At this point, User Story 1's E2E tests pass and `node_links` is functional and independently testable via MCP.

---

## Phase 4: User Story 2 - Discover what references a known node (Priority: P1)

**Goal**: An agent can call `node_backlinks` with a node id and get back every relation elsewhere in the graph — structural or inline — that targets it, without scanning the graph by hand.

**Independent Test**: T010/T011, already red from Phase 2d. Independent of User Story 1 — no dependency on `node_links`' own tasks beyond the shared Phase 2.5 foundation.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T010, T011) and MUST currently be failing (red).

- [X] T031 [US2] Implement `buildRelationReverseIndex(index nodeIndex) reverseIndex` in `internal/app/graph/service/links.go`: mirrors `buildReverseIndex` (`subgraph.go`) but iterates `nodeRelations(n)` instead of `n.Edges` alone; does NOT modify `buildReverseIndex` itself (research.md D3) (depends on T022)
- [X] T032 [US2] Implement `NodeBacklinks(ctx, mounter, dir, id) ([]kernel.BacklinkEntry, error)` in `internal/app/graph/service/links.go`: reuse `enumerateNodes`/`guardIsGraph` unchanged, look up `id` (return `ErrSeedNotFound` on a miss — an unknown id is distinct from a valid id with zero backlinks), otherwise build `buildRelationReverseIndex` over the full index and map `rev[id]` into `[]kernel.BacklinkEntry` (depends on T021, T031)
- [X] T033 [US2] Add a `NodeBacklinks` delegator to `internal/app/graph/component.go`, mirroring `NodeLinks`/`NodeGet`'s existing shape (depends on T032)
- [X] T034 [US2] Create `cmd/arc/graph/serve_tool_node_backlinks.go`: `nodeBacklinksTool`, `nodeBacklinksArgs{ID string}`, `nodeBacklinksInputSchema`, `nodeBacklinksWorkflowNote`, `nodeBacklinksHandler`, and `renderBacklinkTable([]kernel.BacklinkEntry) string` (header `| source | predicate |`, header-only when empty) — mirrors `serve_tool_node_links.go`'s shape (depends on T033)
- [X] T035 [US2] Register `node_backlinks` via `mcp.AddTool` in `cmd/arc/graph/serve.go`'s `buildServer`; append its name to `sessionInstructionsPurpose`; update `sessionInstructions()` to append `nodeBacklinksWorkflowNote`; update `NewServeCmd`'s `Long` text (depends on T034, T026)
- [X] T036 [P] [US2] Run T010/T011 and confirm green (depends on T035)
- [X] T037 [P] [US2] Run T019's `NodeBacklinks` unit test cases and confirm green (depends on T032)
- [X] T038 [P] [US2] Run T014 (unknown-id edge case, `node_backlinks`) and confirm green (depends on T035)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently; `node_links` and `node_backlinks` are both functional over the real MCP transport.

---

## Phase 5: User Story 3 - Audit a node's full relation footprint before changing it (Priority: P2)

**Goal**: An agent can build a complete, non-overlapping picture of a node's relations in both directions using the two tools together.

**Independent Test**: T012, remains red until both Phase 3 and Phase 4 are complete.

### Implementation for User Story 3

> The E2E test for this story was already written in Phase 2d (T012). This story adds no new production code — it exercises the combination of `node_links` (Phase 3) and `node_backlinks` (Phase 4), which is exactly what `nodeRelations`/`buildRelationReverseIndex` sharing one source of truth (research.md D3, spec SC-003) is meant to guarantee.

- [X] T039 [US3] Run T012 and confirm green; if red, the gap is a `nodeRelations`/`buildRelationReverseIndex` inconsistency (T022/T031), not a missing feature (depends on T026, T035)
- [X] T040 [P] [US3] Run T015 (self-reference edge case) and confirm green (depends on T026, T035)
- [X] T041 [P] [US3] Run T016 (duplicate-relations edge case) and confirm green (depends on T026, T035)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests; `node_links`/`node_backlinks` are complete and mutually consistent.

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories or are explicitly called for by spec.md/plan.md but not gated on any single story's E2E tests.

- [X] T042 [P] Check off `node_links(id)`/`node_backlinks(id)` in `specs/VISION.md`'s Phase 5 — MCP Server roadmap checklist as implemented
- [X] T043 [P] Run [quickstart.md](quickstart.md)'s manual validation scenarios end-to-end against a real built binary (spawn a real `go build`-produced `arc serve` binary over stdio against a fixture graph)
- [X] T044 Final sweep of `cmd/arc/graph/serve.go`'s `buildServer` GoDoc comment and `NewServeCmd`'s `Long` text to confirm both `node_links` and `node_backlinks` are named correctly and consistently (depends on T026, T035)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase is retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the HRefs-inclusion note (Principle I) — depends on T004
- [X] TN02 No new domain concept required a Glossary entry beyond the note in T004 (Principle II) — `core.Link` is reused unchanged; `kernel.BacklinkEntry` is a `kernel`-layer DTO, same category as `kernel.MatchEntry`, which required none
- [X] TN03 N/A — no command/flag surface is introduced this release (Principle IX); confirmed by T005

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced that would require a new ADR (Principle I) — confirm both new handlers remain thin wrappers (ADR 003 rule 1 unaffected)
- [X] TN05 Domain logic (`nodeRelations`, `buildRelationReverseIndex`, `NodeLinks`, `NodeBacklinks`) uses no `cobra`/`mcp` import; Cobra/MCP wiring stays confined to `cmd/arc/graph/serve.go`/`serve_tool_node_links.go`/`serve_tool_node_backlinks.go` (Principle III) — confirm via grep
- [X] TN06 T018/T019's unit tests and T008-T017's E2E tests were written first, compiled, and failed semantically before their corresponding implementation tasks (Principle VI) — confirm red (`undefined: service.NodeLinks`/`undefined: service.NodeBacklinks`/`unknown tool "node_links"`/`unknown tool "node_backlinks"`) before Phase 3/4 implementation
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI) — confirm via grep
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI) — quickstart.md's shell commands are a manual/smoke-test guide, not a substitute for `go test`
- [X] TN09 No new external integration was added; no vendor SDK type leaks through `NodeLinks`/`NodeBacklinks`'s signatures (Principle VII) — signatures are `(context.Context, fsys.Mounter, string, string) ([]core.Link, error)` / `(..., string, string) ([]kernel.BacklinkEntry, error)`
- [X] TN10 N/A for this feature — no terminal output/styling change (Principle X)
- [X] TN11 N/A for this feature — no new configuration value or secret (Principle XI) — confirmed by T020
- [X] TN12 N/A — no new command help text is required this release; `NewServeCmd`'s existing `Long` text is updated only to mention `node_links`/`node_backlinks` (Principle XII) — depends on T026, T035, T044
- [X] TN13 E2E tests from Phase 2d (T008-T017) turned GREEN and changed minimally during implementation (Principle VIII) — depends on T027, T029, T030, T036, T038, T039, T040, T041
- [X] TN14 All spec.md scenarios across User Stories 1-3, plus every Edge Case, have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: `node_links`/`node_backlinks` are wholly new, additive MCP tools — no existing tool name, flag, or output schema changes, so no major version bump is required (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; 2a-2e proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion; BLOCKS every user story phase
- **User Stories (Phase 3, 4, 5)**: All depend on Phase 2.5
  - US1 (Phase 3) and US2 (Phase 4) share only Phase 2.5's `nodeRelations`/`BacklinkEntry` foundation and may proceed in parallel
  - US3 (Phase 5) depends on both Phase 3 and Phase 4 being complete (it exercises both tools together) — the only story that is not independent of the other two
- **Additional Polish**: Depends on US1, US2, US3 all being complete
- **Phase N (Constitution Compliance Verification)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 — no dependency on US2/US3
- **User Story 2 (P1)**: Depends on Phase 2.5 — no dependency on US1/US3
- **User Story 3 (P2)**: Depends on Phase 3 AND Phase 4 both being complete — verification-only, adds no production code of its own

### Parallel Opportunities

- T008-T019 (Phase 2d, additive to independent test functions) can be written in parallel
- Once Phase 2.5 completes, Phase 3 (US1) and Phase 4 (US2) can proceed in parallel — different new files (`serve_tool_node_links.go` vs `serve_tool_node_backlinks.go`), though both eventually touch `serve.go`'s registration block sequentially (T035 depends on T026 for that reason)
- Phase 5 (US3) cannot start until both Phase 3 and Phase 4 are complete
- T042-T043 (Polish) are independent of each other

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2d, already complete before this point):
# T008/T009 E2E tests for US1 in cmd/arc/graph/serve_test.go

# Foundational work these tests need (Phase 2.5, already complete before this point):
# T022 nodeRelations(n core.Node) []core.Link

# Confirm User Story 1 independently:
Task: "Run T008/T009 and confirm green"
Task: "Run T013 (unknown-id edge case) and confirm green"
Task: "Run T017 (bare-href edge case) and confirm green"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (`nodeRelations`, `kernel.BacklinkEntry`)
4. Complete Phase 3: User Story 1 (`node_links`)
5. Complete Phase N: Constitution Compliance Verification (subset applicable so far)
6. **STOP and VALIDATE**: Test User Story 1 independently via quickstart.md's manual scenarios
7. Deploy/demo if ready

### Incremental Delivery

1. Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 (`node_links`) → Verify → Deploy/Demo (MVP!)
3. Add User Story 2 (`node_backlinks`) → Verify → Deploy/Demo
4. Add User Story 3 (combined audit, verification-only) → Verify against Phase N → Deploy/Demo
5. Polish (VISION.md, `serve.go` doc comments, quickstart validation)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Foundational Infrastructure together (single ownership recommended for the shared `links.go`/`serve.go` files, given every later phase builds on them)
2. Once Phase 2.5 completes:
   - Developer A: User Story 1 (`node_links`)
   - Developer B: User Story 2 (`node_backlinks`) — can start in parallel, but its `serve.go` registration (T035) must land after Developer A's (T026) to avoid a merge conflict on the same registration block
3. Once both land, either developer runs User Story 3's verification (Phase 5) and Phase N

---

## Notes

- [P] tasks = different files, or independent additions to the same file with no ordering dependency
- [Story] label maps task to specific user story for traceability
- E2E tests (Phase 2d: T008-T017) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements
- Unlike spec 028 (one shared matcher underpinning two "mostly verification" stories), User Stories 1 and 2 here are each a genuinely separate implementation (`NodeLinks` vs `NodeBacklinks`); User Story 3 is this feature's verification-only story, mirroring 028's own precedent of having exactly one story that adds no new production code
