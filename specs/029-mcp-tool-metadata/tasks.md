---

description: "Task list for MCP Tool Metadata Colocation"
---

# Tasks: MCP Tool Metadata Colocation

**Input**: Design documents from `/specs/029-mcp-tool-metadata/`

**Prerequisites**: [plan.md](plan.md) (required), [spec.md](spec.md) (required for user stories), [research.md](research.md), [data-model.md](data-model.md), [contracts/mcp-contract.md](contracts/mcp-contract.md), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to a test, and tests MUST be written before implementation (red-green-refactor). Phase 0 is a behavior-preserving refactor (existing tests are the safety net, per the constitution's own refactor-gate rule); genuine new red→green tests apply to User Stories 2-4, which add new, previously-absent metadata content.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US4)
- Include exact file paths in descriptions

## Path Conventions

All work in this feature is confined to one existing package: `cmd/arc/graph/` (Cobra command + MCP tool registration, constitution Principle III). No `internal/` package is touched — this is a `cmd/`-layer metadata/organization change, not a domain change.

---

## Phase 0: Pre-implementation Refactoring

**Rationale**: This feature requires splitting one 615-line file into per-tool files (spec FR-001/FR-011, research.md D1) before any new metadata content can be added — exactly the "rename, restructure, split files" case this phase exists for. Submitted as its own reviewable step: content (descriptions, args, handlers, render functions) moves unchanged; only the `mcp.Tool` construction changes shape (inline struct literal in `buildServer` → package-level `var` next to its args struct, research.md D2). All existing tests in `cmd/arc/graph/serve_test.go` MUST pass, unchanged, after this phase.

- [X] T001 [P] Create `cmd/arc/graph/serve_tool_node_get.go` (with the required license header, per `CLAUDE.md`): move `nodeGetArgs` and `nodeGetHandler` out of `serve.go` unchanged; add `var nodeGetTool = &mcp.Tool{Name: "node_get", Description: "Fetch a node's full content by id.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}` (content identical to today's inline literal in `buildServer`)
- [X] T002 [P] Create `cmd/arc/graph/serve_tool_node_grep.go` (license header): move `nodeGrepArgs`, `nodeGrepHandler`, `renderMatchTable` out of `serve.go` unchanged; add `var nodeGrepTool = &mcp.Tool{Name: "node_grep", Description: "...", InputSchema: must(inputSchemaFor[nodeGrepArgs]()), Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}` (content identical to today's inline literal)
- [X] T003 [P] Create `cmd/arc/graph/serve_tool_subgraph_get.go` (license header): move `subgraphGetArgs`, `subgraphGetHandler` out of `serve.go` unchanged; add `var subgraphGetTool = &mcp.Tool{...}` (content identical to today's inline literal)
- [X] T004 [P] Create `cmd/arc/graph/serve_tool_context_retrieve.go` (license header): move `contextRetrieveArgs`, `contextRetrieveHandler` out of `serve.go` unchanged; add `var contextRetrieveTool = &mcp.Tool{...}` (content identical to today's inline literal)
- [X] T005 [P] Create `cmd/arc/graph/serve_tool_schema.go` (license header): move `schemaArgs`, `schemaHandler`, `renderSchema`, `joinOrNone` out of `serve.go` unchanged; add `var schemaTool = &mcp.Tool{...}` (content identical to today's inline literal)
- [X] T006 [P] Create `cmd/arc/graph/serve_tool_node_match.go` (license header): move `nodeMatchArgs`, `nodeMatchHandler`, `renderFactTable` out of `serve.go` unchanged; add `var nodeMatchTool = &mcp.Tool{...}` (content identical to today's inline literal)
- [X] T007 Add `must[T any](v T, err error) T` helper (panics on error) to `cmd/arc/graph/serve.go`, next to `inputSchemaFor` (research.md D2)
- [X] T008 Slim `cmd/arc/graph/serve.go`: remove the args structs/handlers/render funcs moved in T001-T006; rewrite `buildServer` to `mcp.AddTool(server, nodeGetTool, nodeGetHandler(dir, index))` / `mcp.AddTool(server, nodeGrepTool, nodeGrepHandler(dir, cfgFile.Grep))` / etc. for all six tools, using the new package vars from T001-T006 (depends on T001-T007)
- [X] T009 Run `go build ./... && go test ./cmd/arc/graph/... -v`; confirm every existing test still passes, unchanged (depends on T008)

---

## Phase 1: Setup

**Purpose**: Establish the pre-refactor baseline and confirm no new dependency is needed.

- [X] T010 Run `go build ./... && go test ./cmd/arc/graph/... -run TestServe -v` against the current, unmodified `serve.go`; record that all tests pass — this is the baseline Phase 0 (T009) must reproduce
- [X] T011 [P] Confirm `go.mod` already pins `github.com/modelcontextprotocol/go-sdk v1.6.1` and `github.com/google/jsonschema-go v0.4.3` (plan.md Technical Context) — no dependency version change required for this feature

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE user-story implementation) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T012 Confirm no new domain entity is introduced (data-model.md: `mcp.Tool`/`jsonschema.Schema` are MCP wire-protocol types realized in `cmd/arc/graph/`, not `internal/core` domain types) — no `ARCHITECTURE.md` Glossary addition required
- [X] T013 Verify `must`/`withExamples` (T007, T024) operate only on `*jsonschema.Schema` (a third-party wire type) and duplicate no existing `internal/core`/`internal/app/graph` type

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T014 Confirm `arc serve`'s Cobra flag surface (`--http`) and its `Long`/`Example` help text require no change — only the MCP-protocol-level `Instructions` string and per-tool `Description`/`InputSchema` change (contracts/mcp-contract.md)
- [X] T015 [P] Review [contracts/mcp-contract.md](contracts/mcp-contract.md) against the post-Phase-0 `cmd/arc/graph/serve_tool_*.go` files to confirm every drafted Description/example maps to a real field on a real tool before Phase 4-6 implementation begins

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T016 Confirm no new adapter/port is introduced (plan.md Technical Context: Storage N/A, no `internal/adapter/fsys` usage change) — N/A, recorded for the compliance record only

### Phase 2d: E2E/Unit Acceptance Test Design (Principle VIII)

- [X] T017 [P] [US1] Add `TestServeToolVarsRegistered` to `cmd/arc/graph/serve_metadata_test.go` (NEW file, license header): confirm all six tool vars (`nodeGetTool`, `nodeGrepTool`, `subgraphGetTool`, `contextRetrieveTool`, `schemaTool`, `nodeMatchTool`) are non-nil and each `.Name` equals its registered wire name — a regression guard over Phase 0's split (data-model.md Validation Rules); passes immediately once Phase 0 is complete, since Phase 0 is the behavior-preserving refactor this guards
- [X] T018 [P] [US2] Add `TestServeToolParameterMetadata` to `cmd/arc/graph/serve_metadata_test.go`: table-driven over every tool's `InputSchema.(*jsonschema.Schema).Properties`, asserting via `github.com/fogfish/it/v2` that each property's `Description != ""` and `len(Examples) >= 1` (data-model.md Validation Rules, spec SC-002). MUST fail (red) against Phase-0-only schemas, which have descriptions but zero `Examples` anywhere
- [X] T019 [P] [US3] Add `TestServeFilterArgumentExamples` to `cmd/arc/graph/serve_metadata_test.go`: for `node_grep`/`subgraph_get`/`context_retrieve`/`node_match`, assert `len(InputSchema.Properties["filter"].Examples) >= 2` (spec SC-003). MUST fail (red) — zero `Examples` today
- [X] T020 [P] [US4] Add `TestServeSessionInstructions` to `cmd/arc/graph/serve_metadata_test.go`: assert `sessionInstructions()`'s return value contains each of the six `<tool>WorkflowNote` constants as a substring (spec SC-004/SC-005, data-model.md Validation Rules). MUST fail (red) — neither `sessionInstructions()` nor any `WorkflowNote` const exists before Phase 6

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T021 Confirm no new configuration value, environment variable, or secret is introduced by this feature (touches only MCP tool metadata construction) — N/A, recorded for the compliance record only

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 3: User Story 1 - One block per tool (Priority: P1) 🎯 MVP

**Goal**: A maintainer can find any one MCP tool's complete contract (name, description, parameters, output description) inside a single colocated block, with no part of that contract declared elsewhere.

**Independent Test**: Pick any one existing MCP tool (e.g. `node_grep`), locate its complete specification within a single contiguous block, and confirm no part of that contract is defined elsewhere.

### Implementation for User Story 1

> Phase 0 already performs this story's structural work (the file split itself). This phase confirms the result against spec.md and turns T017's regression guard green.

- [X] T022 [US1] Manually review each `cmd/arc/graph/serve_tool_*.go` file (quickstart.md step 2): confirm that tool's name, description, full parameter list, and output description are all present in that one file, with nothing declared in `serve.go` or another tool's file
- [X] T023 [US1] Run `TestServeToolVarsRegistered` (T017); confirm GREEN

**Checkpoint**: At this point, User Story 1's acceptance scenarios pass and the colocation improvement is independently shippable

---

## Phase 4: User Story 2 - Parameter descriptions and examples (Priority: P1)

**Goal**: Every tool parameter carries both a description and at least one example value, visible to a connecting client.

**Independent Test**: Inspect any one parameter of any existing tool and confirm it exposes both a non-empty description and at least one example value that is itself valid input for that parameter.

### Implementation for User Story 2

> `TestServeToolParameterMetadata` (T018) was already written in Phase 2d and MUST currently be failing (red). Implementation below MUST turn it green with minimal test changes.

- [X] T024 [US2] Add `withExamples(s *jsonschema.Schema, field string, examples ...any) *jsonschema.Schema` helper to `cmd/arc/graph/serve.go`, next to `stringOrArraySchema`/`inputSchemaFor` (research.md D3)
- [X] T025 [P] [US2] `cmd/arc/graph/serve_tool_node_get.go`: add `nodeGetInputSchema() (*jsonschema.Schema, error)` calling `inputSchemaFor[nodeGetArgs]()` then `withExamples(s, "id", "tls-1-3")`; wire `nodeGetTool.InputSchema = must(nodeGetInputSchema())`; append the output-describing sentence to `nodeGetTool.Description` (contracts/mcp-contract.md `node_get`) — depends on T024
- [X] T026 [P] [US2] `cmd/arc/graph/serve_tool_node_grep.go`: extend `nodeGrepInputSchema()` with `withExamples(s, "pattern", "TODO", "func\\s+\\w+\\(")`; append the output-describing sentence to `nodeGrepTool.Description` (contracts/mcp-contract.md `node_grep`) — depends on T024
- [X] T027 [P] [US2] `cmd/arc/graph/serve_tool_subgraph_get.go`: extend `subgraphGetInputSchema()` with `withExamples(s, "id", "tls-1-3")` and `withExamples(s, "depth", 1, 2)`; append the output-describing sentence to `subgraphGetTool.Description` (contracts/mcp-contract.md `subgraph_get`) — depends on T024
- [X] T028 [P] [US2] `cmd/arc/graph/serve_tool_context_retrieve.go`: extend `contextRetrieveInputSchema()` with `withExamples(s, "query", "post-quantum key exchange")` and `withExamples(s, "limit", 10, 25)`; append the output-describing sentence to `contextRetrieveTool.Description` (contracts/mcp-contract.md `context_retrieve`) — depends on T024
- [X] T029 [P] [US2] `cmd/arc/graph/serve_tool_schema.go`: `schemaArgs{}` has no parameters, so no example work applies (spec FR-003 is vacuously satisfied); confirm `schemaTool.Description` already states its output (contracts/mcp-contract.md `schema` — no change needed)
- [X] T030 [P] [US2] `cmd/arc/graph/serve_tool_node_match.go`: append the output-describing sentence to `nodeMatchTool.Description` (contracts/mcp-contract.md `node_match`); `filter` example values are added in Phase 5 (US3), since `node_match`'s only parameter is the filter argument — depends on T024
- [X] T031 [US2] Run `TestServeToolParameterMetadata` (T018); confirm GREEN (depends on T025-T030)

**Checkpoint**: User Stories 1 AND 2 both pass their tests independently

---

## Phase 5: User Story 3 - Complex filter syntax with examples (Priority: P2)

**Goal**: The structured filter argument, shared by four tools, carries ≥2 example values per tool, each demonstrating a distinct valid usage pattern.

**Independent Test**: Inspect the filter argument on any tool that accepts one, and confirm it carries two or more example values that each demonstrate a distinct, valid usage pattern.

### Implementation for User Story 3

> `TestServeFilterArgumentExamples` (T019) was already written in Phase 2d and MUST currently be failing (red).

- [X] T032 [US3] Add `jsonschema` tags to all six `mcpStatement` fields (`Source`, `SourcePattern`, `Predicate`, `PredicatePattern`, `Target`, `TargetPattern`) in `cmd/arc/graph/serve.go` (research.md D4, contracts/mcp-contract.md shared filter table) — currently undocumented; prerequisite for T033-T036 since it's the shared type all four filter-accepting tools reuse
- [X] T033 [P] [US3] `cmd/arc/graph/serve_tool_node_grep.go`: add `withExamples(s, "filter", ...)` in `nodeGrepInputSchema()` with the 2 patterns from contracts/mcp-contract.md `node_grep` Filter examples (exact-value narrowing; pattern narrowing) — depends on T032
- [X] T034 [P] [US3] `cmd/arc/graph/serve_tool_subgraph_get.go`: add `withExamples(s, "filter", ...)` in `subgraphGetInputSchema()` with the 2 patterns from contracts/mcp-contract.md `subgraph_get` Filter examples (predicate-only traversal scoping; narrowing) — depends on T032
- [X] T035 [P] [US3] `cmd/arc/graph/serve_tool_context_retrieve.go`: add `withExamples(s, "filter", ...)` in `contextRetrieveInputSchema()` with the 2 patterns from contracts/mcp-contract.md `context_retrieve` Filter examples (single-predicate scoping; OR-of-values) — depends on T032
- [X] T036 [P] [US3] `cmd/arc/graph/serve_tool_node_match.go`: add `withExamples(s, "filter", ...)` in `nodeMatchInputSchema()` with the 2 non-trivial patterns from contracts/mcp-contract.md `node_match` Filter examples (both non-empty, since `service.ErrEmptyFilter` rejects an empty filter) — depends on T032
- [X] T037 [US3] Run `TestServeFilterArgumentExamples` (T019); confirm GREEN (depends on T033-T036)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their tests independently

---

## Phase 6: User Story 4 - Workflow and tool-preference guidance (Priority: P3)

**Goal**: The server's overall purpose, recommended workflow, and tool-preference guidance for overlapping tools are colocated per tool and delivered to a connecting client without any tool call.

**Independent Test**: Confirm a client receives a description of the server's purpose and recommended starting workflow at connect time, and that for any two tools whose capabilities overlap, documented guidance states which to prefer and why.

### Implementation for User Story 4

> `TestServeSessionInstructions` (T020) was already written in Phase 2d and MUST currently be failing (red).

- [X] T038 [P] [US4] Add `const nodeGetWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_node_get.go` (contracts/mcp-contract.md `node_get` Workflow note)
- [X] T039 [P] [US4] Add `const nodeGrepWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_node_grep.go` (contracts/mcp-contract.md `node_grep` Workflow note)
- [X] T040 [P] [US4] Add `const subgraphGetWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_subgraph_get.go` (contracts/mcp-contract.md `subgraph_get` Workflow note)
- [X] T041 [P] [US4] Add `const contextRetrieveWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_context_retrieve.go` (contracts/mcp-contract.md `context_retrieve` Workflow note)
- [X] T042 [P] [US4] Add `const schemaWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_schema.go` (contracts/mcp-contract.md `schema` Workflow note)
- [X] T043 [P] [US4] Add `const nodeMatchWorkflowNote = "..."` to `cmd/arc/graph/serve_tool_node_match.go` (contracts/mcp-contract.md `node_match` Workflow note)
- [X] T044 [US4] Replace the `schemaAdvertisement` constant in `cmd/arc/graph/serve.go` with `sessionInstructions() string`, composing the fixed purpose sentence plus all six `WorkflowNote` constants in tool-registration order (research.md D5, contracts/mcp-contract.md "Server-level Instructions"); update `buildServer`'s `mcp.NewServer(..., &mcp.ServerOptions{Instructions: sessionInstructions()})` call — depends on T038-T043
- [X] T045 [US4] Run `TestServeSessionInstructions` (T020); confirm GREEN (depends on T044)

**Checkpoint**: All four user stories pass their tests independently — feature complete

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories

- [X] T046 [P] Check whether `ARCHITECTURE.md` references `serve.go`'s tool list or file layout by name; update it to mention the per-tool file split if so (Principle I)
- [X] T047 [P] Run `go test ./... -cover` and `staticcheck ./...`; confirm no regressions and no unused-symbol warnings (an unreferenced `WorkflowNote` constant would indicate a tool left out of `sessionInstructions()`'s composition)
- [X] T048 Run all 7 steps of [quickstart.md](quickstart.md) end-to-end as final manual validation

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 `ARCHITECTURE.md` reflects architectural changes, if any (T046) (Principle I)
- [X] TN02 Domain concepts added to the `ARCHITECTURE.md` Glossary, if any — none expected per T012 (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes unchanged (T014) (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in `adrs/` with correct numbering, if a new architectural pattern was introduced — none expected; this feature reorganizes an existing ADR-003-governed surface without changing its rules (Principle I)
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated — unaffected by this feature (Principle III)
- [X] TN06 Unit tests (T017-T020) were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, Mandatory Libraries & Tooling)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port — N/A, no new integration (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose` — unaffected by this feature (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags — N/A, no new configuration (T021) (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command — `arc serve`'s Cobra help text is unchanged (T014); MCP tool `Description`/`Instructions` text is the analogous surface for this feature and is populated per Phase 4/6 (Principle XII)
- [X] TN13 Tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated test (T017-T020, T022-T023, T031, T037, T045) (Principle VIII)
- [X] TN15 Release/versioning impact assessed: tool names and JSON wire shapes are unchanged; `Description`/`InputSchema`/`Instructions` content changes are additive and non-breaking for existing MCP clients — no major version bump required (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Pre-implementation Refactoring (Phase 0)**: Depends on Phase 1's baseline (T010) being green; run before Phase 2
- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Phase 0 completion (T009) — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **User Stories (Phase 3-6)**: All depend on Phase 2 completion
  - US1 (Phase 3) has no dependency on US2-US4
  - US2 (Phase 4), US3 (Phase 5), and US4 (Phase 6) each depend only on Phase 0's file split (not on each other) — they touch different fields/consts within the same six files, so run them in priority order (P1 → P2 → P3) to avoid merge conflicts within a shared file, even though they are logically independent
- **Additional Polish**: Depends on all four user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Delivered by Phase 0; Phase 3 confirms it. No dependency on other stories.
- **User Story 2 (P1)**: Can start after Phase 0 (needs the tool vars/`InputSchema()` functions to exist) — independently testable via T018/T031
- **User Story 3 (P2)**: Can start after Phase 0; touches the same `<tool>InputSchema()` functions US2 extends, so is sequenced after US2 to avoid rework, not because of a hard dependency
- **User Story 4 (P3)**: Can start after Phase 0; independent of US2/US3's schema work, touches each tool's `Description`-adjacent `WorkflowNote` const instead

### Within Each User Story

- Tests (Phase 2d) already written and failing before implementation starts
- Per-tool-file tasks marked [P] before the story's final integration/verification task
- Story complete before moving to next priority

### Parallel Opportunities

- T001-T006 (Phase 0's six new files) can run in parallel
- T017-T020 (Phase 2d's four new tests) can run in parallel
- Within Phase 4, T025-T030 (one per tool file) can run in parallel once T024 (the shared `withExamples` helper) is done
- Within Phase 5, T033-T036 (one per tool file) can run in parallel once T032 (shared `mcpStatement` tags) is done
- Within Phase 6, T038-T043 (one `WorkflowNote` const per tool file) can run in parallel

---

## Parallel Example: User Story 2

```bash
# Design (Phase 2d, already complete before this point):
# T018 TestServeToolParameterMetadata in cmd/arc/graph/serve_metadata_test.go

# After T024 (withExamples helper) lands, launch independent per-tool tasks together:
Task: "Add examples to node_get's id parameter in cmd/arc/graph/serve_tool_node_get.go"
Task: "Add examples to node_grep's pattern parameter in cmd/arc/graph/serve_tool_node_grep.go"
Task: "Add examples to subgraph_get's id/depth parameters in cmd/arc/graph/serve_tool_subgraph_get.go"
Task: "Add examples to context_retrieve's query/limit parameters in cmd/arc/graph/serve_tool_context_retrieve.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T010-T011)
2. Complete Phase 0: Pre-implementation Refactoring (T001-T009)
3. Complete Phase 2: Design Preconditions (T012-T021 — CRITICAL, blocks all stories)
4. Complete Phase 3: User Story 1 (T022-T023)
5. Complete Phase N: Constitution Compliance Verification (subset relevant to US1)
6. **STOP and VALIDATE**: colocation is real (quickstart.md step 2), all existing tests still pass
7. Ship the colocation improvement on its own if desired

### Incremental Delivery

1. Setup + Phase 0 + Design Preconditions → Foundation ready
2. Add User Story 1 (Phase 3) → Verify → Ship (MVP: colocated files, unchanged behavior)
3. Add User Story 2 (Phase 4) → Verify → Ship (every parameter documented + exampled)
4. Add User Story 3 (Phase 5) → Verify → Ship (filter argument fully exampled)
5. Add User Story 4 (Phase 6) → Verify → Ship (workflow/preference guidance colocated and client-visible)
6. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests (Phase 2d) MUST already be failing before their story's implementation tasks start, except T017 (US1), which is a regression guard over Phase 0's already-completed refactor, not a new-behavior red test
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections MUST be retained verbatim across features (constitution Governance > Task List Requirements) — only task descriptions are adapted
- `serve_test.go` is intentionally left unsplit (research.md D7) — not in scope for this feature's task list
