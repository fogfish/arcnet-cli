---

description: "Task list template for feature implementation"
---

# Tasks: Context Retrieve Tool (`context_retrieve`)

**Input**: Design documents from `/specs/025-context-retrieve-tool/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: The examples below include test tasks. Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. This feature is a functional extension of existing components (plan.md Structure Decision) — no new Cobra command, no new domain package, no new adapter, no new dependency (research.md D1, D2, D10).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/arc/graph/serve.go` — the existing `arc serve` MCP wiring (MODIFIED, not new): tool arg structs, handlers, `mcp.AddTool` registration
- `cmd/arc/graph/serve_test.go` — colocated E2E tests (MODIFIED), using the existing `connectServeSession`/`textOf` helpers (Principle VIII)
- `internal/app/graph/kernel/context.go` — NEW result type (no `cobra` import, Principle III)
- `internal/app/graph/service/context.go`, `context_test.go` — NEW retrieval logic + unit tests (`github.com/fogfish/it/v2`, Principle VI)
- `internal/app/graph/service/errors.go`, `component.go` — MODIFIED (one new error sentinel, one new delegator)

---

## Phase 1: Setup

**Purpose**: Confirm the existing baseline this feature extends is ready — no new project structure, module, or dependency is introduced (research.md D1, D2).

- [X] T001 Confirm `go.mod` already provides everything this feature needs (`github.com/modelcontextprotocol/go-sdk/mcp`, `github.com/fogfish/faults`) and run `go build ./...` to confirm the baseline compiles clean before any change

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T002 Add `kernel.ContextRetrieveResult` and the "Retrieval Candidate" concept to [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary and MCP Tool table row (currently listing only `node_get`/`node_grep`/`subgraph_get`), per plan.md's Constitution Check action item
- [X] T003 Verify `kernel.ContextRetrieveResult`, `matchesAttrs`, and the ranking comparator do not duplicate an existing `internal/app/graph` type or function — confirm the plan to reuse `degree`/`bfs`/`buildReverseIndex`/`capPool`/`walkNodeFiles`/`enumerateNodes`/`slugify` (already private to `internal/app/graph/service`) verbatim (research.md D2, D5-D7)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T004 [P] Confirm [contracts/mcp-contract.md](contracts/mcp-contract.md) fully specifies `context_retrieve`'s input schema, success reply, and error reply — N/A for CLI flags: this feature adds no Cobra command or flag (research.md D10, plan.md Constitution Check)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T005 [P] Confirm no new adapter is required — `context_retrieve` reuses the existing `fsys.Mounter`/`fsys.Local` port already used by `Grep`/`Subgraph`/`NodeGet` (research.md D1, D2)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T006 [P] [US1] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md User Story 1's 4 acceptance scenarios (topic query returns direct + neighbor matches as full node objects, default limit, deterministic repeat call, empty-result-not-error), using `connectServeSession`/`textOf`; tests MUST compile and fail semantically (red phase)
- [X] T007 [P] [US2] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md User Story 2's 3 acceptance scenarios (filter narrows result, filter excludes an otherwise-reachable neighbor, fully-excluding filter returns empty not error) (red phase)
- [X] T008 [P] [US3] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md User Story 3's 4 acceptance scenarios (default limit 10, explicit limit N, limit larger than candidate count, invalid limit returns a clear error) (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T009 Confirm no new configuration value is introduced — neighbor-expansion caps reuse the already-loaded `cfgFile.Subgraph` (`DirectCap`/`BacklinkCap`) and content-match concurrency reuses `cfgFile.Grep.Workers` (research.md D9); no secret material is involved

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The shared result type, error sentinel, delegator, and MCP tool registration every user story's implementation builds on top of.

- [X] T010 [P] Define `kernel.ContextRetrieveResult` in `internal/app/graph/kernel/context.go` (data-model.md)
- [X] T011 [P] Add `ErrInvalidLimit` sentinel to `internal/app/graph/service/errors.go` (data-model.md, research.md D8)
- [X] T012 Scaffold `service.ContextRetrieve(ctx, mounter, filter, query string, limit int, grepCfg, subgraphCfg, dir string) (kernel.ContextRetrieveResult, error)` in `internal/app/graph/service/context.go`: mount, `guardIsGraph`, validate `limit` via `ErrInvalidLimit`, return an empty result — compiles, not yet three-pass (depends on T010, T011)
- [X] T013 Add `ContextRetrieve` delegator to `internal/app/graph/component.go`, mirroring `Grep`/`Subgraph`/`NodeGet` (depends on T012)
- [X] T014 Scaffold `contextRetrieveArgs` (`query`, `filter *mcpFilter`, `limit *int`), `contextRetrieveHandler` (limit default-resolution: nil → 10, mirroring `subgraphGetArgs.Depth`), and register `context_retrieve` via `mcp.AddTool` with `readOnlyHint: true` in `cmd/arc/graph/serve.go` (depends on T013) — the server now compiles with the new tool registered, and the Phase 2d E2E tests (T006-T008) fail semantically rather than failing to compile

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Assemble working context from a topic in one call (Priority: P1) 🎯 MVP

**Goal**: A single `context_retrieve` call returns full node objects for every node relevant to a free-text query — direct content/attribute matches plus their directly-connected neighbors — deduplicated, ranked, and capped at the default limit of 10.

**Independent Test**: Seed a graph with a node whose content matches a query and a second node connected to it only by an edge; call `context_retrieve` with that query and no `filter`/`limit`; confirm the reply's patch document contains full node objects for both, with no duplicates.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T006) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T015 [P] [US1] Implement the content-match pass in `internal/app/graph/service/context.go`: call `grep.Search` with pattern `"(?i)" + regexp.QuoteMeta(query)` over the `walkNodeFiles`-enumerated `.md` set (research.md D3, spec FR-002a/FR-003)
- [X] T016 [P] [US1] Implement `matchesAttrs(node core.Node, query string) bool` (case-insensitive substring over every `node.Attrs` `Predicate.Value`) as the attribute-match pass in `internal/app/graph/service/context.go` (research.md D4, spec FR-002b)
- [X] T017 [US1] Implement the neighbor-expansion pass: for every id found by the content- or attribute-match pass, gather direct (`nodeTargets`) and backlink (reverse index) neighbors at depth 1, pool across all seeds, and apply `capPool` with `cfg.Subgraph.DirectCap`/`BacklinkCap` (research.md D5, spec FR-002c/FR-009) — depends on T015, T016
- [X] T018 [US1] Implement dedup-by-id plus the two-tier ranking comparator (direct-match tier before neighbor-only tier; within a tier, `degree(index, rev, id)` descending, then id ascending) and truncate to `limit` (research.md D6, spec FR-005/FR-006, Clarifications 2026-08-24) — depends on T017
- [X] T019 [US1] Synthesize the `core.Patch` reply (document id `"context:" + slugify(query) + "@" + timestamp`, title `"Context: " + query`, `Nodes` = the ranked/truncated candidates), reusing `slugify` (research.md D7) — depends on T018
- [X] T020 [US1] Wire `contextRetrieveHandler` to call `service.ContextRetrieve`, render `string(core.RenderPatch(result.Patch, index))` as the tool's `TextContent` reply, and emit the `logCall("context_retrieve", ...)` stderr line (contracts/mcp-contract.md, spec FR-013) — depends on T019, T014
- [X] T021 [P] [US1] Add unit tests for `service.ContextRetrieve` (content-match hit, attribute-match hit, neighbor-expansion hit, dedup of a node reachable two ways, empty-result-not-error) in `internal/app/graph/service/context_test.go` using `github.com/fogfish/it/v2`

**Checkpoint**: At this point, User Story 1's E2E tests (T006) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Narrow retrieval to a relevant slice of the graph (Priority: P2)

**Goal**: An optional `filter` object restricts every one of the three passes — including neighbor-expansion targets — to nodes satisfying it.

**Independent Test**: Seed a graph where a query matches nodes of more than one kind; call `context_retrieve` with that query plus a filter restricting to one kind; confirm only nodes of that kind appear, including a would-be neighbor-expansion node that the filter excludes.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T007) and MUST currently be failing (red).

- [X] T022 [US2] Restrict all three passes to nodes satisfying the decoded `core.Filter` — content-scan inclusion, attribute-match candidacy, and neighbor-expansion target acceptance — in `internal/app/graph/service/context.go` (spec FR-004) — depends on T020
- [X] T023 [P] [US2] Add unit tests confirming a filter-excluded neighbor is dropped even though reachable from an included direct match, and that a fully-excluding filter yields an empty result, not an error, in `internal/app/graph/service/context_test.go`

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Control how much context comes back (Priority: P3)

**Goal**: `limit` defaults to 10 and bounds the reply size; an invalid `limit` produces a clear error without crashing the server.

**Independent Test**: Seed a graph where a query matches more nodes than the default limit; call `context_retrieve` with different explicit `limit` values; confirm the returned node count never exceeds the requested limit and an invalid limit is rejected cleanly.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T008) and MUST currently be failing (red).

- [X] T024 [US3] Resolve `contextRetrieveArgs.Limit`'s default (`nil` → `10`) in `contextRetrieveHandler`, mirroring `subgraphGetArgs.Depth`'s existing pattern, in `cmd/arc/graph/serve.go` (spec FR-006) — depends on T014
- [X] T025 [US3] Confirm `service.ContextRetrieve` rejects `limit <= 0` via `ErrInvalidLimit` before any file is read (spec FR-007, research.md D8) — depends on T011, T012 (validation already scaffolded in Phase 2.5; this task wires the handler's resolved `limit` into it end-to-end)
- [X] T026 [P] [US3] Add unit tests for limit behavior (fewer candidates than limit returns all with no padding, more candidates than limit truncates to the highest-ranked, `limit <= 0` returns `ErrInvalidLimit`) in `internal/app/graph/service/context_test.go`

**Checkpoint**: All three stories' E2E tests (T006-T008) pass independently and together

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories

- [X] T027 [P] Run every scenario in [quickstart.md](quickstart.md) manually against a real `arc serve` process and an MCP client (e.g. MCP Inspector), confirming outcomes match, including the read-only `git status --porcelain` before/after check (spec SC-004)
- [X] T028 [P] Confirm `go vet ./...` and the project's configured `staticcheck` run clean on every new/modified file
- [X] T029 Final pass on [ARCHITECTURE.md](../../ARCHITECTURE.md)'s MCP Tool table and `cmd/arc/graph` file-tree comment to confirm `context_retrieve` is documented consistently with `node_get`/`node_grep`/`subgraph_get` (Principle I)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX) — N/A for this feature (no new Cobra command or flag, research.md D10); confirm the MCP tool's input schema matches contracts/mcp-contract.md exactly instead

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — confirm ADR 003 already covers this feature's pattern and no new ADR is needed (plan.md Constitution Check)
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — confirm no new adapter was actually added (research.md D1, D2)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A: this feature produces no terminal/stdout output of its own (MCP tool-call content only, same deviation ADR 003 already documents for `arc serve`)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — confirm no new configuration was actually added (research.md D9)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — confirm `context_retrieve`'s MCP tool `Description` field is populated (no Cobra help text applicable)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — confirm this is purely additive (a new MCP tool on an already-running server) with no breaking change to `node_get`/`node_grep`/`subgraph_get`/`arc serve`'s existing flag surface

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3-5)**: All depend on Phase 2.5
  - US1 has no dependency on US2/US3. US2 and US3 each depend on US1's implementation existing (T020) since both add behavior on top of the same `service.ContextRetrieve`/`contextRetrieveHandler` — but each has its own independent E2E test suite and can be verified independently once US1 lands
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories
- **User Story 2 (P2)**: Can start after Phase 2.5; its implementation task (T022) depends on US1's T020 (same function, same handler) but its own E2E tests (T007) are independent
- **User Story 3 (P3)**: Can start after Phase 2.5; its implementation tasks (T024-T025) depend on US1's T014/T020 for the same reason; its own E2E tests (T008) are independent

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Content-match and attribute-match passes (independent files/functions) before neighbor expansion, which depends on both
- Ranking/truncation before reply rendering
- Story complete before moving to next priority

### Parallel Opportunities

- T004 and T005 (Phase 2b/2c confirmation tasks) can run in parallel
- T006, T007, T008 (E2E test-writing for all three stories) can run in parallel — they touch different scenarios in the same file, so coordinate merge order
- T010 and T011 (Phase 2.5 scaffolding) can run in parallel
- T015 and T016 (content-match vs. attribute-match passes) can run in parallel
- T021, T023, T026 (unit test additions) can run in parallel with each other once their respective implementation tasks land
- T027 and T028 (polish) can run in parallel

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2, already complete before this point):
# T006 E2E test(s) for US1 acceptance scenarios in cmd/arc/graph/serve_test.go

# Launch independent implementation tasks for User Story 1 together:
Task: "Implement content-match pass in internal/app/graph/service/context.go"
Task: "Implement matchesAttrs attribute-match pass in internal/app/graph/service/context.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenario 1)
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
   - Developer B: User Story 2 (starts on E2E tests immediately; implementation waits on T020)
   - Developer C: User Story 3 (starts on E2E tests immediately; implementation waits on T014/T020)
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
