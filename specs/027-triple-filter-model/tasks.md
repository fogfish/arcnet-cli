---

description: "Task list template for feature implementation"
---

# Tasks: Triple Filter for Node Attributes and Graph Edges

**Input**: Design documents from `/specs/027-triple-filter-model/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md) (design decisions D1-D10), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Unit and E2E acceptance tests are NOT optional for this project (constitution Principles VI, VIII). Every spec.md acceptance scenario maps 1:1 to an E2E test; tests are written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Task IDs run sequentially across the whole file (T001, T002, ...); `TN0x` IDs in Phase N are the constitution's fixed verification checklist.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- `internal/core/filter.go` — the `Matcher`/`Statement`/`Filter` domain type (no `cobra`/`mcp` import, Principle III)
- `internal/app/graph/service/{subgraph,context}.go` — traversal-scoping logic (BFS neighbor closures, reverse-index shape)
- `cmd/arc/graph/{grep,subgraph,serve}.go` — flag/JSON decode → domain call → render only
- Colocated `*_test.go` files hold both unit tests (`internal/core`) and E2E tests (`cmd/arc/graph`, `internal/app/graph/service`)

---

## Phase 0: Pre-implementation Refactoring

**Rationale for inclusion**: `internal/core.Filter`'s public shape (`Types`/`Tags`/`Attrs`/`AttrPatterns`) is deleted outright and replaced with `Statements []Statement` (spec: "replaces `core.Filter` 100%"). This phase isolates the mechanical, behavior-preserving half of that rewrite — new Go types, `Match` still evaluating only attribute facts (exactly today's coverage, just re-expressed) — from the new-capability work in Phase 3/4, so `--type`/`--tag`/`--attr`/`--attr~=` CLI output is provably byte-identical (spec FR-014/SC-002) before anything new is layered on. MUST be submitted as its own PR/commit; all existing black-box CLI tests (`cmd/arc/graph/grep_test.go`, `cmd/arc/graph/subgraph_test.go`) MUST pass unchanged after this phase.

- [X] T001 Rewrite `internal/core/filter_test.go` (table-driven, `github.com/fogfish/it/v2`) to cover the new `Matcher`/`Statement`/`Filter` API reproducing exactly today's covered scenarios — zero-value matches everything, type-as-OR (via the synthesized `(id, "type", Type)` fact, research.md D2), tags-as-AND, attr exact-match scalar/array, attr pattern-match scalar/array, combined-AND, matching-zero-nodes — written to fail (red) against the not-yet-rewritten `filter.go`
- [X] T002 Implement `Matcher{Values []string, Patterns []*regexp.Regexp}` with `IsWildcard()`/`Match(string) bool` in `internal/core/filter.go` (research.md D1)
- [X] T003 Implement `Statement{Source, Predicate, Target Matcher}` in `internal/core/filter.go`; delete `Types`/`Tags`/`Attrs`/`AttrPatterns` fields and their `matchTypes`/`matchTags`/`matchAttrs`/`matchAttrPatterns`/`matchAttrValue`/`matchAttrPattern` helpers
- [X] T004 Implement `Filter{Statements []Statement}` and `Filter.Match(node Node) bool` in `internal/core/filter.go`: ANDs every statement; each statement satisfied by the synthesized `(node.ID, "type", node.Type)` fact (D2) or any `node.Attrs[name]` entry (skip `Value == nil`, mirroring today's `attrStrings`) — **not yet** `node.Edges` (that is US2, Phase 4, FR-004) — until T001 passes (green)
- [X] T005 Change `optsFilter.build()`'s final step in `cmd/arc/graph/grep.go` to emit `[]core.Statement` per research.md D4's lowering table, keeping the existing intermediate `map[string]string`/`map[string]*regexp.Regexp` accumulation (dedup-by-name, last-write-wins) byte-for-byte unchanged
- [X] T006 Rewrite `cmd/arc/graph/grep_opts_test.go`'s four assertions (`TestOptsFilterBuildParsesExactAttrValue`, `...ParsesPatternAttrValue`, `...RejectsMalformedAttrValue`, `...ComposesTypeTagAttr`) to assert against the new `f.Statements` shape instead of the deleted `f.Attrs`/`f.AttrPatterns`/`f.Types`/`f.Tags` fields, preserving each test's original intent
- [X] T007 Run `go test ./internal/core/... ./cmd/arc/graph/...` and confirm every pre-existing `cmd/arc/graph/grep_test.go`/`cmd/arc/graph/subgraph_test.go` scenario (black-box, via `sut()`) still produces byte-identical output — no source changes to those two files in this phase

**Checkpoint**: `core.Filter` is fully replaced at the Go-API level; every existing CLI-observable behavior is provably unchanged; no new capability exists yet.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the branch is ready for feature work. No new package, no new Cobra command, no new dependency (plan.md Technical Context) — this phase is intentionally thin.

- [X] T008 Confirm `go build ./...` and `staticcheck ./...` are clean on `027-triple-filter-model` at Phase 0's tip before starting design-precondition work
- [X] T009 [P] Confirm no new entry is needed in `go.mod` (plan.md: no new dependency) — record this explicitly rather than silently skipping the check

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate; most of the actual design decisions were already recorded in `research.md`/`data-model.md`/`contracts/` during `/speckit-plan` — these tasks finalize/verify them and produce the one remaining artifact (E2E test files), not re-derive the decisions.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T010 Verify `Matcher`/`Statement` (introduced in Phase 0) do not duplicate any existing `internal/core` type before Phase 3/4 extend them further
- [X] T011 [P] Update `ARCHITECTURE.md`'s Glossary: correct the existing "Filter" row to describe `Statements []Statement`/`Match`/`Traversal`/`Narrowing` (not `Types`/`Tags`/`Attrs`/`AttrPatterns`), and add new rows for "Statement" and "Matcher" (data-model.md)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T012 Review [contracts/cli-contract.md](contracts/cli-contract.md) against `cmd/arc/graph/subgraph.go`'s current flag registration and confirm `--predicate`'s naming/repeatability matches `--type`'s existing convention exactly before implementing it in Phase 3
- [X] T013 [P] Review [contracts/mcp-contract.md](contracts/mcp-contract.md)'s `statements` JSON shape against `mcpFilter`'s current fields in `cmd/arc/graph/serve.go` and confirm the field-naming rule (no `s`/`p`/`o`, no `subject`/`object` — spec FR-016) is satisfied

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T014 N/A — this feature adds no new external system, no new adapter (plan.md Constitution Check); confirm no `internal/adapter/fsys` or other adapter change is needed and record that explicitly rather than skipping this subsection silently

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> Every task below is written to compile and fail semantically (red) before its story's implementation phase begins.

- [X] T015 [P] [US1] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for User Story 1's Acceptance Scenarios 1-5 (spec.md): `--predicate cites` excludes a `mentions`-only target; omitting `--predicate` includes both; a `--predicate` matching no edge yields the seed alone; `--predicate` applies at every hop of a depth>1 expansion; `--predicate` combined with `--type`/`--tag`/`--attr` narrows the surviving set on top of scoping — via `sut()`, using a new `testdata` fixture graph with at least two distinct edge predicates from one seed
- [X] T016 [P] [US1] Write E2E tests in `cmd/arc/graph/serve_test.go` for `subgraph_get`'s and `context_retrieve`'s predicate-scoped-expansion scenarios (mirrors T015, over MCP `mcp.CallTool`), including the "no `filter` argument at all" case producing output identical to today's `subgraph_get` reply
- [X] T017 [P] [US2] Write E2E tests in `cmd/arc/graph/grep_test.go` for User Story 2's Acceptance Scenarios 1-3: a node matched via an attribute-fact statement, a different node matched via an edge-fact statement (e.g. `predicate: "cites", target: "paper-b"`), a combined-statement AND requiring one statement satisfied by an attribute and another by an edge on the same node
- [X] T018 [P] [US2] Write E2E tests in `cmd/arc/graph/serve_test.go` for `node_grep`'s edge-fact-matching scenario (same shape as T017, over MCP) and for `subgraph_get`/`context_retrieve`'s final-narrowing Acceptance Scenario 4 (the seed always survives narrowing regardless of match)
- [X] T019 [P] [US3] Write/expand E2E regression tests in `cmd/arc/graph/grep_test.go` and `cmd/arc/graph/subgraph_test.go` asserting **byte-identical** stdout for every pre-existing `--type`/`--tag`/`--attr`/`--attr~=` fixture combination already covered by those files today, run against the post-Phase-0 binary, to be re-run (not rewritten) after Phase 3/4 land

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T020 N/A — no new configuration value, no secret handling introduced by this feature; record explicitly

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The MCP wire-shape redesign (research.md D7) is genuinely shared between US1 (`subgraph_get`) and US2 (`node_grep`) — both need `mcpFilter`/`toCoreFilter` speaking `statements` before either story's MCP-facing scenarios can turn green. `context_retrieve` already has a `Filter` field and picks up the new shape for free from this same change.

- [X] T021 Redesign `mcpFilter` in `cmd/arc/graph/serve.go` as `{ Statements []mcpStatement }` where `mcpStatement` carries optional `Source`/`Predicate`/`Target` (string-or-array) and `SourcePattern`/`PredicatePattern`/`TargetPattern` (string-or-array) fields, per contracts/mcp-contract.md; delete the old `Type`/`Tags`/`Attrs`/`AttrPatterns` fields
- [X] T022 Rewrite `toCoreFilter()` in `cmd/arc/graph/serve.go` to decode `mcpStatement` values into `core.Statement`/`core.Matcher`, compiling every `*Pattern` field via `regexp.Compile` and returning `service.ErrInvalidFilterPattern` on the first invalid one (unchanged error contract, new trigger surface — research.md D7/contracts/mcp-contract.md)
- [X] T023 Delete the old `type`/`tags`/`attrs`/`attrPatterns` JSON-shape assertions in `cmd/arc/graph/serve_test.go`'s existing `node_grep`/`context_retrieve` filter tests (no legacy path is kept — spec is explicit: no dual schema) and replace them with `statements`-shaped equivalents proving the same underlying matches still occur

**Checkpoint**: Foundation ready — `node_grep` and `context_retrieve` already speak the new wire shape; US1 and US2 implementation can now proceed in parallel.

---

## Phase 3: User Story 1 - Scope neighbor expansion to a specific relation (Priority: P1) 🎯 MVP

**Goal**: `arc subgraph` and the MCP `subgraph_get`/`context_retrieve` tools can restrict one-hop (and multi-hop) BFS expansion to edges of a caller-named relation, instead of always following every edge a node carries.

**Independent Test**: Build a graph where a seed has edges of two distinct predicates to different targets; run `arc subgraph <seed> --predicate <name>` and confirm only the matching-predicate target is reached; confirm omitting `--predicate` reaches both (T015/T016, already red from Phase 2d).

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T015, T016) and MUST currently be failing (red). Implementation below turns them green with minimal test changes.

- [X] T024 [P] [US1] Add `Statement.IsTraversalConstraint() bool` (`Source`/`Target` both wildcard, `Predicate` not) to `internal/core/filter.go` (research.md D3)
- [X] T025 [US1] Add `Filter.Traversal() Filter` / `Filter.Narrowing() Filter` to `internal/core/filter.go`, partitioning `Statements` per T024; add unit tests in `internal/core/filter_test.go` proving the partition is exhaustive and disjoint and that `Narrowing()` of a filter containing only a traversal constraint is vacuously true for every node (depends on T024)
- [X] T026 [US1] In `internal/app/graph/service/subgraph.go`, replace `nodeTargets`/the `map[string][]string` `reverseIndex` with `backlinkEdge{Source, Predicate string}`-based `reverseIndex map[string][]backlinkEdge` and `admittedEdges(n core.Node, scope core.Filter) []core.Link` / `admittedBacklinks(id string, rev reverseIndex, scope core.Filter) []backlinkEdge` (research.md D5) — an empty `scope` admits every edge, preserving today's default
- [X] T027 [US1] Update `Subgraph`'s two `neighbors` closures (direct/backlink, passed to `bfs`) in `internal/app/graph/service/subgraph.go` to build on `admittedEdges`/`admittedBacklinks` with `scope := filter.Traversal()`; update `addCandidate` to call `filter.Narrowing().Match(n)` instead of `filter.Match(n)` (depends on T025, T026)
- [X] T028 [US1] Apply the identical `Traversal()`/`Narrowing()` split to `internal/app/graph/service/context.go`'s neighbor-pooling BFS closures and its `addCandidate`'s `direct == false` branch — the `direct == true` branch (content/attribute matches) and the upfront `filterIncluded`/`pathIncluded` pass keep using the full, unsplit `filter` (research.md D3) (depends on T025, T026)
- [X] T029 [P] [US1] Add `--predicate <name>` (`StringArrayVar`, repeatable, OR-within) to `optsFilter` in `cmd/arc/graph/grep.go`, registered only from `cmd/arc/graph/subgraph.go`'s `NewSubgraphCmd` (not `arc grep`'s command construction — research.md D10); extend `opts.build()` to lower all `--predicate` repeats into one traversal-constraint `Statement`
- [X] T030 [US1] Update `cmd/arc/graph/subgraph.go`'s `Long`/`Example` text for `--predicate` per contracts/cli-contract.md (depends on T029)
- [X] T031 [US1] Add `Filter *mcpFilter` to `subgraphGetArgs` in `cmd/arc/graph/serve.go`; wire it through `toCoreFilter()` into `appgraph.Subgraph`'s existing `filter core.Filter` parameter, replacing the hardcoded `core.Filter{}` (research.md D8) (depends on T021/T022)
- [X] T032 [US1] Update `schemaAdvertisement` and `subgraph_get`/`context_retrieve`'s `mcp.Tool.Description` in `cmd/arc/graph/serve.go` to mention predicate-scoped traversal is available (research.md D9)
- [X] T033 [US1] Run T015/T016 and confirm green; run T007's byte-identical regression subset for `arc subgraph` (no `--predicate`) once more to confirm the default path is untouched (depends on T027, T029, T031)

**Checkpoint**: At this point, User Story 1's E2E tests (T015, T016) pass and predicate-scoped expansion is fully functional and independently testable via both `arc subgraph` and MCP.

---

## Phase 4: User Story 2 - Match nodes by either their attributes or their relations (Priority: P2)

**Goal**: A selection criterion (`arc grep`/`node_grep`, and `subgraph_get`/`context_retrieve`'s final narrowing) matches a node whether the satisfying fact is one of its attributes or one of its outgoing edges.

**Independent Test**: A node whose attribute matches a statement and a separate node whose edge (not attributes) matches an equivalent statement are both selected by the same filter (T017/T018, already red from Phase 2d). Independent of User Story 1 — no dependency on `Traversal()`/`Narrowing()` or BFS scoping.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T017, T018) and MUST currently be failing (red).

- [X] T034 [US2] Extend `Filter.Match` in `internal/core/filter.go` (built in Phase 0 on attribute facts only) to additionally test every `node.Edges` entry as a `(node.ID, edge.Predicate, edge.Target)` fact against every statement (FR-004) — a statement is satisfied by *either* an attribute fact or an edge fact, node-wide
- [X] T035 [P] [US2] Add unit tests in `internal/core/filter_test.go`: a node matched purely via an edge fact, a node matched purely via an attribute fact, a combined filter where different statements are satisfied by different facts on the same node (spec FR-005 wording) (depends on T034)
- [X] T036 [US2] Run T017/T018 and confirm green (depends on T034)
- [X] T037 [US2] Confirm (add a regression assertion if none exists) that `--type`/`--tag`/`--attr`-derived statements, which always constrain `Target`, are unaffected by T034's edge-fact addition — no existing CLI fixture accidentally starts matching via an unrelated edge (depends on T034)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently; edge-fact matching works for `arc grep`/`node_grep` and for `subgraph_get`/`context_retrieve`'s result narrowing without requiring `--predicate`/a traversal constraint.

---

## Phase 5: User Story 3 - Existing command-line filtering keeps working unchanged (Priority: P3)

**Goal**: `--type`/`--tag`/`--attr`/`--attr~=` on `arc grep`/`arc subgraph` produce output identical to before this feature, after everything above has landed.

**Independent Test**: Re-run every pre-existing CLI filtering fixture (T019, already written in Phase 2d against the Phase-0-only binary) against the fully-implemented feature (Phases 0-4) and confirm no diff.

### Implementation for User Story 3

> This story delivers no new capability — Phase 0 already implemented CLI compatibility. This phase's job is closing the loop: re-verify T019 against the final binary, since Phase 3/4 touched files T019 depends on (`filter.go`, `serve.go`).

- [X] T038 [US3] Re-run T019's byte-identical regression suite against the Phase 0 + Phase 2.5 + Phase 3 + Phase 4 binary; diff any deviation against research.md D2's one deliberately-called-out change (case-insensitive `--type` matching) — any other deviation is a defect, not an intentional change
- [X] T039 [P] [US3] Add one CLI regression scenario combining `--predicate` (US1, new) with `--type`/`--tag`/`--attr` (existing) in `cmd/arc/graph/subgraph_test.go`, confirming the pre-existing flags' narrowing semantics are unaffected by the new flag's presence or absence (depends on T029)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests; the full feature is functionally complete.

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories or are explicitly called for by spec.md/plan.md but not gated on any single story's E2E tests.

- [X] T040 [P] Rewrite `specs/VISION.md`'s "Filtering" section (Type/Tag/Attribute-filter prose plus the old "MCP filter object" JSON example) to describe the triple `statements` model as current behavior (research.md D9)
- [X] T041 [P] Run `go doc`/regenerate any `cobra/doc`-generated command reference if the project produces one, to pick up `--predicate`'s help text (Principle XII)
- [X] T042 Run [quickstart.md](quickstart.md)'s four scenarios end-to-end against a real built binary and check off its validation checklist
- [X] T043 [P] Re-read `internal/app/graph/service/subgraph.go`'s and `context.go`'s GoDoc comments referencing the old `nodeTargets`/flat `reverseIndex` shape (e.g. `boundaryTargets`'s comment) and update them to describe `admittedEdges`/`admittedBacklinks`/`backlinkEdge`

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase is retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the "Filter"/"Statement"/"Matcher" changes (Principle I) — depends on T011
- [X] TN02 Domain concepts (`Statement`, `Matcher`) added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — depends on T011
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: `--predicate`'s name, help text, and `--type`/`--tag`/`--attr`'s unchanged names/help text (Principle IX) — depends on T012, T030

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced that would require a new ADR (Principle I) — confirmed: MCP handlers remain thin wrappers (ADR 003 rule 1 unaffected)
- [X] TN05 Domain logic (`Filter`/`Statement`/`Matcher`, `admittedEdges`/`admittedBacklinks`) uses no `cobra`/`mcp` import; Cobra wiring (`grep.go`/`subgraph.go`) and MCP wiring (`serve.go`) remain separated from `internal/core`/`internal/app/graph/service` (Principle III)
- [X] TN06 T001/T024's unit tests and T015-T019's E2E tests were written first, compiled, and failed semantically before their corresponding implementation tasks (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI) — quickstart.md's shell commands are a manual/smoke-test guide, not a substitute for `go test`
- [X] TN09 No new external integration was added; no vendor SDK type leaks through `internal/core`'s `Filter`/`Statement`/`Matcher` (Principle VII)
- [X] TN10 N/A for this feature — no terminal output/styling change (`arc subgraph`'s no-color convention, research.md-referenced, is unaffected) (Principle X)
- [X] TN11 N/A for this feature — no new configuration value or secret (Principle XI) — confirmed by T020
- [X] TN12 Help text (`Short`/`Long`/`Example`) updated for `arc subgraph`'s new `--predicate` flag (Principle XII) — depends on T030
- [X] TN13 E2E tests from Phase 2d (T015-T019) turned GREEN and changed minimally during implementation (Principle VIII) — depends on T033, T036, T038
- [X] TN14 All spec.md scenarios across User Stories 1-3 have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: the MCP `filter` JSON shape is a breaking change to a scriptable contract consumed by MCP clients — per Principle XIV this SHOULD be preceded by a deprecation warning in a prior minor release, but spec.md explicitly scopes MCP wire compatibility out of this feature ("no compatibility field, no version negotiation... MCP clients are not a compatibility boundary for this change"); this is a recorded, deliberate exception, not an oversight — confirm the release notes/CHANGELOG call it out explicitly as a breaking MCP change

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 0 (Pre-implementation Refactoring)**: No dependencies — separate PR, run first
- **Phase 1 (Setup)**: Depends on Phase 0 (build/staticcheck baseline is checked against the rewritten `filter.go`)
- **Phase 2 (Design Preconditions)**: Depends on Phase 1 — BLOCKS all user stories; 2a-2e proceed in parallel with each other
- **Phase 2.5 (Foundational Infrastructure)**: Depends on Phase 2 completion; BLOCKS US1's MCP tasks (T031) and US2's MCP tasks (T018's implementation, T036)
- **User Stories (Phase 3, 4, 5)**: All depend on Phase 2.5
  - US1 (Phase 3) and US2 (Phase 4) are independent of each other and may proceed in parallel
  - US3 (Phase 5) depends on US1 and US2 having landed (it re-verifies the combined binary), unlike a typical independent P3 story
- **Additional Polish**: Depends on US1, US2, US3 all being complete
- **Phase N (Constitution Compliance Verification)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 (T021/T022 for `subgraph_get`'s filter field) — no dependency on US2
- **User Story 2 (P2)**: Depends on Phase 2.5 (T021/T022 for `node_grep`'s new wire shape) — no dependency on US1; `Filter.Match`'s edge-fact extension (T034) is independent of `Traversal()`/`Narrowing()` (T024/T025)
- **User Story 3 (P3)**: Depends on US1 + US2 (re-verifies the fully-combined binary) — the one story in this feature that is NOT independent of the others, because its entire purpose is confirming their combination didn't regress anything

### Parallel Opportunities

- T015-T019 (Phase 2d, all different files) can be written in parallel
- Once Phase 2.5 completes, Phase 3 (US1) and Phase 4 (US2) can proceed in parallel (different files: `subgraph.go`/`context.go`'s traversal closures vs. `filter.go`'s `Match` body; both touch `filter.go` but in non-overlapping ways — T024/T025 vs. T034 should be sequenced by whoever owns `filter.go` to avoid a merge conflict, not a behavioral dependency)
- T040-T043 (Polish) are independent of each other

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2d, already complete before this point):
# T015 E2E tests for US1 in cmd/arc/graph/subgraph_test.go
# T016 E2E tests for US1 in cmd/arc/graph/serve_test.go

# Launch independent implementation tasks for User Story 1 together:
Task: "Add Statement.IsTraversalConstraint to internal/core/filter.go (T024)"
Task: "Add --predicate flag to optsFilter in cmd/arc/graph/grep.go (T029)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 0: Pre-implementation Refactoring (mechanical `Filter` rewrite, CLI-compat proven)
2. Complete Phase 1: Setup
3. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
4. Complete Phase 2.5: Foundational Infrastructure (MCP wire-shape redesign)
5. Complete Phase 3: User Story 1 (predicate-scoped traversal)
6. Complete Phase N: Constitution Compliance Verification (subset applicable so far)
7. **STOP and VALIDATE**: Test User Story 1 independently via quickstart.md Scenarios 1-2
8. Deploy/demo if ready

### Incremental Delivery

1. Phase 0 + Phase 1 + Phase 2 + Phase 2.5 → Foundation ready, CLI compatibility already proven
2. Add User Story 1 → Verify → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify → Deploy/Demo
4. Add User Story 3 (regression closeout) → Verify against Phase N → Deploy/Demo
5. Polish (VISION.md, ARCHITECTURE.md, quickstart validation)

### Parallel Team Strategy

With multiple developers:

1. Team completes Phase 0 + Setup + Design Preconditions + Foundational Infrastructure together (Phase 0 especially benefits from single ownership, given it touches `filter.go` and `grep.go` that every later phase builds on)
2. Once Phase 2.5 completes:
   - Developer A: User Story 1 (`subgraph.go`/`context.go` traversal closures, `--predicate`, `subgraph_get`'s filter field)
   - Developer B: User Story 2 (`filter.go`'s edge-fact `Match` extension)
3. Both stories integrate and run Phase N verification before merge; User Story 3's regression closeout runs last, against the merged result

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- E2E tests (Phase 2d: T015-T019) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements
- User Story 3 is this feature's one exception to "stories are independent" — its entire content is verifying US1+US2's combination, per spec.md's own framing of it as a non-regression guarantee rather than new capability
