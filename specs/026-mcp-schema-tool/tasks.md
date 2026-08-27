---

description: "Task list template for feature implementation"
---

# Tasks: Graph Ontology Schema Tool (`schema`)

**Input**: Design documents from `/specs/026-mcp-schema-tool/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: The examples below include test tasks. Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. This feature is a functional extension of `cmd/arc/graph/serve.go` only (plan.md Structure Decision) — no new Cobra command, no new domain package, no new adapter, no new dependency, no new domain function: `schema` renders the `index core.Index` value `buildServer` already resolves (research.md D1).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/arc/graph/serve.go` — the existing `arc serve` MCP wiring (MODIFIED, not new): `schemaArgs`, `renderSchema`, `schemaHandler`, `mcp.AddTool` registration, `mcp.ServerOptions.Instructions`, `NewServeCmd`'s `Long` text
- `cmd/arc/graph/serve_test.go` — colocated E2E tests (MODIFIED), using the existing `connectServeSession`/`textOf` helpers (Principle VIII)

No `internal/` package is touched — `schema` reads only the already-resolved `index core.Index` value `buildServer` computes for the three existing tools (research.md D1).

---

## Phase 1: Setup

**Purpose**: Confirm the existing baseline this feature extends is ready — no new project structure, module, or dependency is introduced (research.md D1).

- [X] T001 Confirm `go.mod` already provides everything this feature needs (`github.com/modelcontextprotocol/go-sdk/mcp`) and run `go build ./...` to confirm the baseline compiles clean before any change

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T002 Update [ARCHITECTURE.md](../../ARCHITECTURE.md)'s MCP Tool glossary entry and `cmd/arc/graph/serve.go` file-tree comment (currently listing `node_get`/`node_grep`/`subgraph_get`/`context_retrieve`) to name `schema`, per plan.md's Constitution Check action item
- [X] T003 Verify `schemaArgs`/`renderSchema`/`schemaHandler` do not duplicate an existing type or function — confirm the plan to render the already-existing `core.Index`/`core.PredicateDef`/`core.TypeDef` (`internal/core/rules.go`) directly, with zero new domain types (research.md D1, data-model.md)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T004 [P] Confirm [contracts/mcp-contract.md](contracts/mcp-contract.md) fully specifies `schema`'s (empty) input schema, success reply shape, and the absence of an error reply — N/A for CLI flags: this feature adds no Cobra command or flag (plan.md Constitution Check)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T005 [P] Confirm no new adapter or domain call is required — `schema` reads the `index` value `buildServer` already resolves via the existing `resolveIndexOrDefault`/`internal/app/schema.Resolve` call (research.md D1)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T006 [P] [US1] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md User Story 1's 4 acceptance scenarios (built-in-only graph returns every built-in predicate/class with descriptions; a graph with a project-specific addition includes both built-in and custom entries; a class's required/optional attributes are readable from the reply; a predicate's role/merge behavior are readable from the reply), using `connectServeSession`/`textOf`; tests MUST compile and fail semantically (red phase)
- [X] T007 [P] [US2] Write E2E tests in `cmd/arc/graph/serve_test.go` for spec.md User Story 2's 2 acceptance scenarios (a freshly connected session's `InitializeResult.Instructions` names `schema` and recommends it first; the same guidance is present on a second, independent connection) (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T008 Confirm no new configuration value is introduced — `schema` takes no flags or environment input of its own; no secret material is involved

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The tool registration every user story's behavior builds on top of.

- [X] T009 Scaffold `schemaArgs` (empty `struct{}`) and `schemaHandler(dir string, index core.Index) func(...)` in `cmd/arc/graph/serve.go`, returning an empty `TextContent` and calling `logCall("schema", "", nil)` — compiles, not yet rendering (data-model.md)
- [X] T010 Register `schema` via `mcp.AddTool` with `readOnlyHint: true` in `buildServer`, passing the already-resolved `index` (depends on T009) — the server now compiles with the new tool registered, and the Phase 2d E2E tests (T006) fail semantically (empty reply) rather than failing to compile; T007's Instructions-related tests still fail to compile against a `nil` `ServerOptions.Instructions` until Phase 4

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Discover the graph's ontology in one call (Priority: P1) 🎯 MVP

**Goal**: A single `schema` call returns every currently defined predicate (name + description) and class (description + required/optional predicate names) in the graph, built-in vocabulary plus any project-specific additions.

**Independent Test**: Connect to `arc serve` against a freshly `arc init`-ed graph and call `schema` with no arguments; confirm the reply lists every built-in predicate and class from `internal/app/schema/kernel.CorePredicateDefs`/`CoreTypeDefs`, each with its description, and each class with its required/optional predicate names.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T006) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T011 [US1] Implement `renderSchema(index core.Index) string` in `cmd/arc/graph/serve.go`: sorted-by-name `## Predicates` bullet list (`- **name**: description`), sorted-by-name `## Classes` subsections (`### name`, description, `Required: ...`/`Optional: ...` with `(none)` for empty lists) — mirrors `renderMatchTable`'s existing pure-function shape (research.md D2, data-model.md) — depends on T009
- [X] T012 [US1] Wire `schemaHandler` to call `renderSchema(index)` and return it as the tool's single `TextContent` reply (contracts/mcp-contract.md) — depends on T011, T010

**Checkpoint**: At this point, User Story 1's E2E tests (T006) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Learn about the schema operation without prior documentation (Priority: P2)

**Goal**: Every connecting client is told, at session start and before any tool call, that `schema` exists and should be called first — via `mcp.ServerOptions.Instructions`, present consistently on every connection.

**Independent Test**: Connect a fresh client to `arc serve` and inspect `InitializeResult.Instructions` before calling any tool; confirm it names `schema` and recommends calling it first. Reconnect a second time and confirm the same guidance appears.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T007) and MUST currently be failing (red).

- [X] T013 [US2] Add a package-level `const schemaAdvertisement string` in `cmd/arc/graph/serve.go` naming `schema` and recommending it as the session's first call (research.md D3)
- [X] T014 [US2] Change `buildServer`'s `mcp.NewServer(&mcp.Implementation{...}, nil)` call to pass `&mcp.ServerOptions{Instructions: schemaAdvertisement}` instead of `nil` (research.md D3, data-model.md) — depends on T013
- [X] T015 [US2] Update `NewServeCmd`'s `Long` help text in `cmd/arc/graph/serve.go` to name `schema` alongside the existing four tools and note it as the recommended first call (research.md D4, Principle XII)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Use ontology descriptions to form correct requests (Priority: P3)

**Goal**: The ontology returned by `schema` carries enough information (names, descriptions, required/optional attributes) that a client can pick the right class or predicate for a task from the reply alone.

**Independent Test**: Given only `schema`'s reply (no other documentation), confirm a known class's name and description clearly identify what kind of information it represents, and a known predicate's name and description clearly identify what relationship it expresses.

> This story adds no new behavior of its own — it is a quality property of User Story 1's already-implemented output, not a separate code path (spec.md marks its Independent Test as "validated indirectly" through US1/US2). No new E2E test-writing task exists in Phase 2d for it; the task below only strengthens existing assertions.

### Implementation for User Story 3

- [X] T016 [US3] Extend T006's E2E assertions in `cmd/arc/graph/serve_test.go` to check that a specific known predicate's and a specific known class's *exact* description text (not just presence of the name) appear in the `schema` reply, verbatim, so a reader (human or agent) relying solely on the reply text can distinguish that entity from every other one — depends on T012

**Checkpoint**: All three stories' E2E tests (T006, T007, T016) pass independently and together

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories

- [X] T017 [P] Run every scenario in [quickstart.md](quickstart.md) manually against a real `arc serve` process and an MCP client (e.g. MCP Inspector), confirming outcomes match, including the read-only `git status --porcelain` before/after check
- [X] T018 [P] Confirm `go vet ./...` and the project's configured `staticcheck` run clean on every new/modified file
- [X] T019 Final pass on [ARCHITECTURE.md](../../ARCHITECTURE.md)'s MCP Tool table and `cmd/arc/graph` file-tree comment to confirm `schema` is documented consistently with `node_get`/`node_grep`/`subgraph_get`/`context_retrieve` (Principle I)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — confirm no new domain concept was actually needed (research.md D1, D2)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX) — N/A for this feature (no new Cobra command or flag); confirm the MCP tool's input schema matches contracts/mcp-contract.md exactly instead

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — confirm ADR 003 already covers this feature's pattern and no new ADR is needed (plan.md Constitution Check)
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — confirm `schema`'s handler makes no domain call at all beyond reading the already-resolved `index` value
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — confirm applies to `renderSchema` if any table-driven unit test is added beyond the E2E suite; not required if E2E coverage alone is deemed sufficient for a pure, one-function rendering path
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — confirm no new adapter was actually added (research.md D1)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A: this feature produces no terminal/stdout output of its own (MCP tool-call content and session `Instructions` only, same deviation ADR 003 already documents for `arc serve`)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — confirm no new configuration was actually added (research.md D1)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — confirm `schema`'s MCP tool `Description`, `NewServeCmd`'s updated `Long` text, and `mcp.ServerOptions.Instructions` are all populated and consistent
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — including User Story 3's scenarios, satisfied via T016's strengthened assertions on User Story 1's output rather than a separate code path
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — confirm this is purely additive (a new MCP tool plus a previously-unset `Instructions` field on an already-running server) with no breaking change to `node_get`/`node_grep`/`subgraph_get`/`context_retrieve`/`arc serve`'s existing flag surface

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3-5)**: All depend on Phase 2.5
  - US1 has no dependency on US2. US2's implementation (T013-T015) is independent of US1's (T011-T012) — both build on Phase 2.5's registration but touch disjoint parts of `buildServer`/`NewServeCmd`. US3 (T016) depends on US1's T012 since it strengthens US1's own test assertions
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories
- **User Story 2 (P2)**: Can start after Phase 2.5 — no dependency on User Story 1's implementation (different code path: `ServerOptions.Instructions` vs. `renderSchema`); its own E2E tests (T007) are independent
- **User Story 3 (P3)**: Can start only after User Story 1's T012 lands — it strengthens US1's own reply assertions rather than exercising new behavior

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- `renderSchema` before wiring it into `schemaHandler` (US1)
- The `Instructions` constant before passing it into `mcp.NewServer` (US2)
- Story complete before moving to next priority

### Parallel Opportunities

- T004 and T005 (Phase 2b/2c confirmation tasks) can run in parallel
- T006 and T007 (E2E test-writing for US1 and US2) can run in parallel — they touch different scenarios in the same file, so coordinate merge order
- T011 (US1 implementation) and T013 (US2 implementation) can run in parallel — disjoint code (`renderSchema` vs. the `Instructions` constant)
- T017 and T018 (polish) can run in parallel

---

## Parallel Example: User Story 1 and User Story 2

```bash
# Design (Phase 2, already complete before this point):
# T006 E2E test(s) for US1 acceptance scenarios in cmd/arc/graph/serve_test.go
# T007 E2E test(s) for US2 acceptance scenarios in cmd/arc/graph/serve_test.go

# Launch independent implementation tasks together:
Task: "Implement renderSchema in cmd/arc/graph/serve.go"
Task: "Add schemaAdvertisement const and wire it into mcp.ServerOptions.Instructions in cmd/arc/graph/serve.go"
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
   - Developer A: User Story 1 (`renderSchema`)
   - Developer B: User Story 2 (`Instructions`) — fully independent of Developer A's work
   - Developer C: User Story 3 — waits on Developer A's T012
3. Stories complete and integrate independently; each runs Phase N verification before merge
