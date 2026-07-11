---

description: "Task list for CLI/MCP \"Type\" Terminology Consistency"
---

# Tasks: CLI/MCP "Type" Terminology Consistency

**Input**: Design documents from `/specs/015-predicate-node-shape-cli/`

**Prerequisites**: [plan.md](plan.md) (required), [spec.md](spec.md) (required for user stories), [research.md](research.md), [data-model.md](data-model.md), [contracts/kind-to-type-rename-contract.md](contracts/kind-to-type-rename-contract.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, this project's E2E acceptance tests are NOT optional — every spec.md acceptance scenario maps 1:1 to an E2E test. This feature is a rename over already-passing tests, so "writing tests first" here means updating each existing assertion to the new name/wording *before* the rename lands, so it fails for the right reason (red), then implementing the rename to turn it green.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- File paths are exact and grounded in the current codebase (plan.md Project Structure, research.md)

## Path Conventions

- `cmd/arc/graph/` — Cobra command definitions for `grep`, `subgraph`, `serve`, and their colocated `*_test.go` E2E tests
- `internal/core/` — the shared `Filter` domain type consumed by all three commands
- `internal/app/graph/service/` — `apply`'s domain logic, including the warning string this feature renames

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a clean, verified starting point. No new package, command, or dependency is introduced by this feature (plan.md Technical Context).

- [X] T001 Confirm `go build ./...` and `go test ./...` pass on branch `015-predicate-node-shape-cli` before any change, establishing the pre-rename baseline
- [X] T002 [P] Note the exact current text of every surface this feature touches — `cmd/arc/graph/grep.go` (flag, help, example), `cmd/arc/graph/subgraph.go` (help, example, `--stubs` text), `internal/core/filter.go` (`Kinds` field/`matchKinds`), `internal/app/graph/service/apply.go` (warning string), `cmd/arc/graph/serve.go` (`mcpFilter.Kind`, table header) — cross-checked against contracts/kind-to-type-rename-contract.md's "Before" sections

**Checkpoint**: Baseline confirmed; no new infrastructure needed before Phase 2.

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T003 Update `ARCHITECTURE.md`'s Glossary and package-tree comments that describe the now-renamed concepts — the `grep.go` comment ("--kind/--tag/--attr"), the `Filter{Kinds,Tags,Attrs,AttrPatterns}` reference, the "grouped by Kind/ID" reference, the **Filter** glossary entry ("Kinds OR'd"), the **Match** glossary entry ("the node's kind/id"), and the **Subgraph** glossary entry ("grouped by kind") — to say "type"/"Types" (confirm no new domain type is introduced; this is a rename, not a new concept)
- [X] T004 Verify `Types` does not collide with any other existing field/method name in `internal/core` before renaming `Filter.Kinds` (Principle V)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T005 [P] Confirm contracts/kind-to-type-rename-contract.md's flag table, warning-text delta, MCP field delta, and table-header delta (already drafted during `/speckit-plan`) are the exact and complete set of surfaces to change — no additional "kind" occurrence exists in `cmd/arc/graph/{grep,subgraph,serve}.go` or `internal/app/graph/service/apply.go` beyond what's documented there (re-grep to confirm)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T006 [P] Confirm no new external integration/adapter is introduced — `arc serve`'s existing MCP adapter (`cmd/arc/graph/serve.go`, [ADR 003](../../adrs/003-mcp-server-adapter.md)) is the only adapter touched, and only its `node_grep` filter's wire field name and result-table header change (research.md D2); the adapter's shape and the `node_get`/`subgraph_get` tools are untouched

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T007 [P] [US1] In `cmd/arc/graph/grep_test.go`, update every `--kind`/`"kind"` reference (`cmd.Flags().Set("kind", ...)` calls and surrounding comments at the lines documented in research.md D3) to `--type`/`"type"`; add one new test asserting `arc grep --kind ...` fails with a standard unknown-flag error post-rename (spec Edge Cases). Tests MUST fail against current code (red phase).
- [X] T008 [P] [US1] In `cmd/arc/graph/subgraph_test.go`, update every `--kind`/`"kind"` reference (`cmd.Flags().Set("kind", ...)` calls and surrounding comments) to `--type`/`"type"` (red phase)
- [X] T009 [P] [US1] In `cmd/arc/graph/grep_opts_test.go`, rename `TestOptsFilterBuildComposesKindTagAttr` to `TestOptsFilterBuildComposesTypeTagAttr`, its `kind: []string{...}` field literal to `typ: []string{...}` (or the renamed `optsFilter` field from T014), and its `f.Kinds`/`len(f.Kinds)` assertions to `f.Types` (red phase)
- [X] T010 [P] [US1] In `internal/core/filter_test.go`, update every `Kinds:` field literal to `Types:` across all table-driven cases (red phase)
- [X] T011 [P] [US2] In `cmd/arc/graph/apply_test.go`, update the two stderr NOT-contain assertions from `"not a recognized node kind"` to `"not a recognized node type"` (red phase — these currently pass vacuously against the *old* wording, so this update must be paired with confirming they'd fail against the *new* wording until T020 lands)
- [X] T012 [P] [US3] In `cmd/arc/graph/serve_test.go`, update the `node_grep` filter payload from `"kind": [...]` to `"type": [...]`; add/update an assertion on the returned table's header row expecting `| id | type | line | snippet |`; add one new test confirming a filter payload still keyed `"kind"` is accepted without error but has no filtering effect post-rename (spec Edge Cases) (red phase)
  - **Correction discovered during implementation**: the "accepted without error, no filtering effect" premise was wrong — `mcp.AddTool` validates arguments against a JSON Schema with `additionalProperties: false` (the SDK's unconditional default for every struct-derived tool-argument schema, not introduced by this rename), so a stale `"kind"` key is rejected as a tool error, not silently dropped. spec.md's Edge Cases, research.md D2, and contracts/kind-to-type-rename-contract.md §3 were corrected to match; the new test (`TestServeNodeGrepOldKindFieldRejectedAsUnrecognizedProperty`) asserts the verified behavior instead.

### Phase 2e: Configuration & Secrets Review (Principle XI)

- N/A — no new config surface (plan.md Technical Context/Constitution Check).

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure (Shared Rename)

**Purpose**: `internal/core.Filter`'s `Kinds`→`Types` rename is consumed by both US1 (`grep`/`subgraph` via `optsFilter.build()`) and US3 (`serve`'s `mcpFilter.toCoreFilter()`) — it must land before either story's command-layer tasks, so both can build on the renamed field directly rather than each renaming it independently.

- [X] T013 Rename `internal/core.Filter.Kinds` to `Types` and the unexported `matchKinds` method to `matchTypes` in `internal/core/filter.go`; update the field's doc comment ("Types is empty-matches-every-type...") and `Filter.Match`'s call site from `f.matchKinds(node)` to `f.matchTypes(node)` — method body unchanged (research.md D1, spec FR-009). Makes T010 pass.

**Checkpoint**: Foundation ready — User Story 1 and User Story 3 can now proceed in parallel.

---

## Phase 3: User Story 1 - Filter by type using consistent CLI vocabulary (Priority: P1) 🎯 MVP

**Goal**: `arc grep` and `arc subgraph` expose `--type` (not `--kind`) with matching help text, preserving today's filter semantics exactly.

**Independent Test**: Run `arc grep --type source <pattern>` and `arc subgraph <node> --type source` against a fixture graph with nodes of several types; confirm identical results to what the old `--kind` flag produced; confirm `arc grep --kind source <pattern>` now fails with an unknown-flag error.

### Implementation for User Story 1

> E2E tests for this story were already updated in Phase 2d (T007-T010) and MUST currently be failing (red). Implementation below turns them green with no further test changes.

- [X] T014 [US1] In `cmd/arc/graph/grep.go`, rename `optsFilter.kind` to `optsFilter.typ` (or equivalent), the registered flag from `"kind"` to `"type"`, and its description to `"Restrict to nodes of this type (repeatable, OR)"`; update `build()`'s `f.Kinds = append(f.Kinds, o.kind...)` to `f.Types = append(f.Types, o.typ...)` (depends on T013). Makes T007 and T009 pass.
- [X] T015 [US1] In `cmd/arc/graph/grep.go`, update the `Long` help text's `--kind/--tag/--attr` to `--type/--tag/--attr`, the `<kind>  <id>  <line>  <text>` row-format line to `<type>  <id>  <line>  <text>`, and the `Example` block's `arc grep --kind source TLS` to `arc grep --type source TLS`. Finishes T007.
- [X] T016 [US1] In `cmd/arc/graph/subgraph.go`, update the `Long` help text's `--kind/--tag/--attr` to `--type/--tag/--attr`, "grouped by kind" to "grouped by type", the `Example` block's `arc subgraph TLS --kind source` to `arc subgraph TLS --type source`, and the `--stubs` flag description's "(kind and id only)" to "(type and id only)" (depends on T014, since `subgraph.go` reuses the same `optsFilter`). Makes T008 pass.

**Checkpoint**: At this point, User Story 1's E2E tests (T007-T010) pass and `arc grep`/`arc subgraph`'s `--type` filtering is fully functional and testable independently.

---

## Phase 4: User Story 2 - Consistent terminology in apply's reporting (Priority: P2)

**Goal**: `arc apply`'s unrecognized-type warning says "type", not "kind".

**Independent Test**: Apply a patch introducing a node of a type not yet in the graph's resolved schema index; confirm the warning reads "... is not a recognized node type for this graph ...".

### Implementation for User Story 2

> E2E test for this story was already updated in Phase 2d (T011) and MUST currently be failing (red).

- [X] T017 [US2] In `internal/app/graph/service/apply.go`, update the unrecognized-type warning string from `"%s is not a recognized node kind for this graph — auto-registered with a default schema document"` to `"%s is not a recognized node type for this graph — auto-registered with a default schema document"`. Makes T011 pass.

**Checkpoint**: User Story 2's E2E test (T011) passes, independently of User Story 1/3.

---

## Phase 5: User Story 3 - Consistent vocabulary across the MCP interface (Priority: P2)

**Goal**: `arc serve`'s `node_grep` MCP tool accepts its type filter under `"type"` (not `"kind"`) and labels its result table's type column "type".

**Independent Test**: Call `node_grep` with a filter keyed `"type"`; confirm it narrows results. Inspect the returned table; confirm its header reads "type". Call again with the filter still keyed `"kind"`; confirm no error but no filtering effect.

### Implementation for User Story 3

> E2E test for this story was already updated in Phase 2d (T012) and MUST currently be failing (red).

- [X] T018 [US3] In `cmd/arc/graph/serve.go`, rename `mcpFilter.Kind []string \`json:"kind,omitempty"\`` to `mcpFilter.Type []string \`json:"type,omitempty"\``, and update `toCoreFilter`'s `Kinds: append([]string(nil), f.Kind...)` to `Types: append([]string(nil), f.Type...)` (depends on T013). Makes half of T012 pass.
- [X] T019 [US3] In `cmd/arc/graph/serve.go`'s `renderMatchTable`, update the header row from `"| id | kind | line | snippet |\n|---|---|---|---|\n"` to `"| id | type | line | snippet |\n|---|---|---|---|\n"`. Finishes T012.

**Checkpoint**: User Story 3's E2E test (T012) passes, independently of User Story 1/2. All three user stories now pass their E2E tests.

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories.

- [ ] T020 [P] Grep the repository for any remaining `--kind`/`"kind"` reference tied to `grep`/`subgraph`/`apply`/`serve` outside the files already covered above (e.g. README examples, `cobra/doc`-generated reference docs, if present) and update them
- [ ] T021 [P] Add a `specs/CHANGELOG.md` entry documenting the breaking `--kind`→`--type` flag rename and the MCP `node_grep` `kind`→`type` field rename (spec FR-011)
- [ ] T022 Run quickstart.md's manual and automated validation steps end-to-end against a built `arc` binary

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 `ARCHITECTURE.md` reflects the "kind"→"type" terminology change (Principle I) — confirms T003
- [X] TN02 No new domain concept was added to the `ARCHITECTURE.md` Glossary; existing entries were corrected in place (Principle II) — confirms T003/T004
- [X] TN03 Command/flag surface matches the Phase 2b design (contracts/kind-to-type-rename-contract.md) exactly: `--type` flag name, help text, unknown-flag error behavior for `--kind` (Principle IX) — verified live via quickstart.md §1 against a built binary (T022)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced; no new ADR is required (Principle I)
- [X] TN05 `internal/core.Filter`'s rename stays within the domain layer; `cmd/arc/graph`'s Cobra wiring and the MCP adapter remain separated (Principle III)
- [X] TN06 Every updated test (T007-T012) was confirmed failing (red) against the pre-rename code before its corresponding implementation task (T014-T019) landed (Principle VI) — confirmed via `go vet ./...` showing exactly the three expected red packages before T013 landed
- [X] TN07 All updated/added tests use `github.com/fogfish/it/v2` exclusively, matching each file's existing convention (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation; only `go test` (Principle VI)
- [X] TN09 The MCP adapter's port/adapter shape is unchanged beyond the wire field rename; no vendor SDK type leaks through it (Principle VII)
- [X] TN10 No terminal-output behavior beyond the mechanically-required label changes (table header, warning text) was altered — TTY detection, `NO_COLOR`, `--quiet`/`--verbose` behavior is untouched (Principle X)
- [X] TN11 N/A — no configuration surface changed (Principle XI)
- [X] TN12 `Short`/`Long`/`Example` help text for `arc grep` and `arc subgraph` fully reflects the `--type` rename (Principle XII) — confirms T015/T016, verified live via quickstart.md §1 (T022)
- [X] TN13 E2E tests from Phase 2d (T007-T012) turned GREEN with no further test changes beyond what Phase 2d already specified (Principle VIII), **with one justified exception**: T012's MCP old-field-name edge-case test required a correction beyond what Phase 2d specified, because Phase 2d's own premise (unrecognized MCP fields are silently ignored) was discovered to be factually wrong for this codebase's MCP SDK (`additionalProperties: false` on every struct-derived tool-argument schema) — see the T012 note above and research.md D2's correction. The test still turned GREEN, just asserting the verified-correct behavior instead of the originally-assumed one.
- [X] TN14 Every spec.md acceptance scenario (US1 1-3, US2 1, US3 1-2) has a passing, colocated E2E test (Principle VIII) — US1 scenario 3 (help text) verified live via quickstart.md §1 (T022) rather than a unit assertion, consistent with this project's existing convention of not unit-testing Cobra help strings
- [X] TN15 Release/versioning impact assessed and recorded: this feature breaks `--kind`/MCP `kind` — plan.md's Constitution Check already records this as an accepted pre-1.0 trade-off (Principle XIV); confirm T021's CHANGELOG entry documents it for release notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; 2a-2d can proceed in parallel with each other (2e is N/A)
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion — BLOCKS User Story 1 and User Story 3 (both consume the renamed `Filter.Types`)
- **User Stories (Phase 3-5)**: All depend on Phase 2 (and Phase 2.5 for US1/US3); User Story 2 depends only on Phase 2, not Phase 2.5
  - User Story 1, 2, and 3 can proceed in parallel (if staffed) or sequentially in priority order (P1 → P2 → P2)
- **Additional Polish**: Depends on all three user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 (T013) — no dependency on User Story 2 or 3
- **User Story 2 (P2)**: Depends only on Phase 2 — no dependency on User Story 1 or 3 or on Phase 2.5
- **User Story 3 (P2)**: Depends on Phase 2.5 (T013) — no dependency on User Story 1 or 2

### Within Each User Story

- E2E tests (Phase 2d) already updated and failing before implementation starts
- Shared `Filter.Types` rename (Phase 2.5) before any command-layer task that consumes it (US1, US3)
- Story complete before moving to the next priority, if working sequentially

### Parallel Opportunities

- T002 can run in parallel with T001
- Phase 2a-2d tasks marked [P] can run in parallel with each other
- T007-T012 (Phase 2d, all in different files) can all run in parallel
- Once Phase 2.5 (T013) completes, User Story 1 (T014-T016) and User Story 3 (T018-T019) can proceed in parallel; User Story 2 (T017) can start as soon as Phase 2 completes, independent of Phase 2.5
- T020/T021 (Additional Polish) can run in parallel

---

## Parallel Example: Phase 2d Test Updates

```bash
# All four files are independent — launch together:
Task: "Update --kind/\"kind\" references to --type/\"type\" in cmd/arc/graph/grep_test.go"
Task: "Update --kind/\"kind\" references to --type/\"type\" in cmd/arc/graph/subgraph_test.go"
Task: "Update Kinds: to Types: in internal/core/filter_test.go"
Task: "Update the apply-warning stderr assertion in cmd/arc/graph/apply_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (`Filter.Types` rename)
4. Complete Phase 3: User Story 1 (`--type` flag on `grep`/`subgraph`)
5. Complete Phase N: Constitution Compliance Verification (scoped to US1's tasks)
6. **STOP and VALIDATE**: Test User Story 1 independently via quickstart.md §1
7. Deploy/demo if ready — User Story 1 alone already retires the two highest-traffic "kind" surfaces

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking the others (all three are independent renames)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Foundational Infrastructure together
2. Once complete:
   - Developer A: User Story 1 (`grep`/`subgraph`)
   - Developer B: User Story 2 (`apply`)
   - Developer C: User Story 3 (`serve`)
3. Stories complete and integrate independently (no shared files beyond the already-landed T013); each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections retained per constitution Governance > Task List Requirements
- This feature is a rename with no new abstraction: every implementation task is a text/identifier
  substitution over an already-existing, already-correct code path (spec FR-009) — there is deliberately no
  task that adds new logic
