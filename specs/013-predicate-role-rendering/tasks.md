---

description: "Task list for Schema-Driven Link Rendering (specs/013-predicate-role-rendering)"
---

# Tasks: Schema-Driven Link Rendering

**Input**: Design documents from `/specs/013-predicate-role-rendering/`

**Prerequisites**: [plan.md](plan.md) (required), [spec.md](spec.md) (required for user stories), [research.md](research.md), [data-model.md](data-model.md), [contracts/render-shape-contract.md](contracts/render-shape-contract.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. This feature has no new command/adapter/domain package (plan.md "Structure Decision"): every task below touches existing files.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Story priority note (deviation from strict P1→P2→P3 phase order, flagged per Constitution Principle I's "raise, don't silently diverge" norm)

spec.md assigns **US1 = P1**, **US2 = P2**, **US3 = P1**. Despite sharing US1's P1 priority, US3 ("round-trip
stability for already-canonical documents") is implemented and phased **after** US2: US3's own Acceptance
Scenario 1 defines "canonical schema-driven shape" explicitly in terms of US2's single-predicate heading
omission ("single-predicate heading omitted where applicable"), so US3 cannot be fully implemented or tested
until US2's omission rule exists. Implementation order is therefore **US1 → US2 → US3**, not priority order.

---

## Phase 1: Setup

**Purpose**: Establish a clean, verified baseline before touching any file — this feature modifies existing
files only; there is no new package/command to scaffold (plan.md Structure Decision).

- [ ] T001 Confirm a clean baseline: `go build ./...` and `go test ./...` both pass on branch
      `013-predicate-role-rendering` before any change in this feature begins
- [ ] T002 [P] Re-read [internal/core/markdown.go](../../internal/core/markdown.go)'s `renderNodeBody`
      (lines ~735-800), `RenderNode` (~719-733), and `RenderPatch` (~802-845) once more immediately before
      editing, to confirm line numbers/helper names (`renderLinkBullet`, `titleCaseType`) referenced in
      research.md/data-model.md still match — this file is the entire feature's blast-radius center

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the
Compliance Checklist.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T003 Update [ARCHITECTURE.md](../../ARCHITECTURE.md)'s **Node** glossary entry: append one sentence to
      its `Edges` clause distinguishing "parsing still ignores original grouping, unchanged" (spec 010) from
      "rendering now derives flat-vs-grouped from each predicate's own schema `Role`" (this feature),
      cross-referencing `specs/013-predicate-role-rendering` (research.md D9)
- [ ] T004 Confirm no new domain type is introduced: `Index`/`PredicateDef`/`Role`/`Label` already exist in
      [internal/core/rules.go](../../internal/core/rules.go) unchanged (data-model.md "No new Key Entity") —
      record this confirmation, no code change

### Phase 2b: Command & Flag Contract Design (Principle IX)

**N/A for this feature** — no new command, subcommand, or flag is introduced; `arc subgraph`/`arc serve`'s
existing `--json` contracts (`kernel.SubgraphResult`) are unchanged (plan.md Constitution Check, Principles
IX-XIV). The interface contract this feature *does* introduce (the `RenderNode`/`RenderPatch` signature and
rendering algorithm) is already fixed in [contracts/render-shape-contract.md](contracts/render-shape-contract.md).

### Phase 2c: External Integration & Adapter Design (Principle VII)

**N/A for this feature** — no new external system, port, or adapter (plan.md Constitution Check, Principle
VII). `resolveIndexOrDefault` (Phase 2.5 below) is a thin convenience over the already-existing
`internal/app/schema.Resolve`/`fsys.Store`, not a new adapter.

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T005 Record the acceptance-scenario → test mapping below (no code yet — this is the design record;
      the actual failing-test tasks are embedded at the start of each story's phase, T009/T010, T017, T022,
      because they cannot compile until Phase 2.5's `Index` parameter lands on `RenderNode`/`RenderPatch`):
      - US1 scenarios 1-3 → `internal/core/markdown_test.go` (unit) + `cmd/arc/graph/subgraph_test.go` (E2E)
      - US2 scenarios 1-2 → `internal/core/markdown_test.go` (unit) + `cmd/arc/graph/apply_test.go` (E2E,
        alongside the existing `TestApplyCreatesTimelineEntriesChronologically`)
      - US3 scenarios 1-2 → `internal/core/markdown_test.go` (unit only — round-trip stability is a pure
        function property, no CLI surface needed beyond what US1/US2's own E2E tests already exercise)

### Phase 2e: Configuration & Secrets Review (Principle XI)

**N/A for this feature** — no new configuration value, flag, or secret (plan.md Constitution Check, Principle
XI not implicated).

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational — `Index` threaded through every call site, behavior unchanged

**Purpose**: The mechanical, behavior-preserving signature change every user story depends on (research.md
D1/D6/D7): add the `index Index` parameter everywhere `RenderNode`/`RenderPatch`/`renderNodeBody` are declared
and called, without yet changing what gets rendered (still always-flat) — isolates "plumbing compiles, all
existing tests still green" from "new behavior" (Phase 3+), per constitution Principle VI's red/green
separation.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete and
`go build ./... && go test ./...` both pass.

- [ ] T006 In [internal/core/markdown.go](../../internal/core/markdown.go): change
      `func RenderNode(n Node) ([]byte, error)` to `func RenderNode(n Node, index Index) ([]byte, error)`,
      `func RenderPatch(p Patch) ([]byte, error)` to `func RenderPatch(p Patch, index Index) ([]byte, error)`,
      and `func renderNodeBody(n Node) []byte` to `func renderNodeBody(n Node, index Index) []byte` (threading
      `index` through both callers of `renderNodeBody` inside `RenderNode`/`RenderPatch`); `index` is accepted
      but not yet consulted — rendering stays byte-identical to today (contracts/render-shape-contract.md's
      algorithm lands in Phase 3-4, not here)
- [ ] T007 [P] In [internal/app/schema/service/schema.go](../../internal/app/schema/service/schema.go):
      `Seed()` builds `core.Index{Predicates: kernel.CorePredicateDefs, Types: kernel.CoreTypeDefs}` once and
      passes it to both `core.RenderNode` calls (`predicateNode`/`typeNode`, lines ~60/68); `registerIfAbsent`
      (called by `RegisterType`/`RegisterPredicate`) passes `core.Index{}` with a one-line comment noting this
      is safe because the node it renders never carries `Edges` (research.md D6)
- [ ] T008 [P] In [internal/app/graph/service/apply.go](../../internal/app/graph/service/apply.go): give
      `nodeContentChanged(existing, merged core.Node)` and `writeNode(store fsys.Store, path string, node
      core.Node)` a new `index core.Index` parameter each, threaded from `Apply`'s own existing `index`
      parameter at both call sites inside `Apply`
- [ ] T009 [P] In [cmd/arc/graph/subgraph.go](../../cmd/arc/graph/subgraph.go): add a `resolveIndexOrDefault(store
      fsys.Store) core.Index` helper (research.md D7 — resolves via `appschema.Resolve(store)`, returns
      `core.Index{}` on any error); give `humanSubgraphPrinter` an `index core.Index` field used in
      `Show`'s `core.RenderPatch(r.Patch, p.index)` call; in `RunE`, call `resolveIndexOrDefault(store)`
      right after `store` is mounted and construct the printer with it instead of using the package-level
      `subgraphRenderers` var directly for the Human case
- [ ] T010 [P] In [cmd/arc/graph/serve.go](../../cmd/arc/graph/serve.go): resolve `index :=
      resolveIndexOrDefault(store)` once in `buildServer` (reusing T009's helper — colocate it in
      `subgraph.go` since both files are in package `graph`); give `nodeGetHandler(dir string, index
      core.Index)` and `subgraphGetHandler(dir string, cfg configkernel.SubgraphConfig, index core.Index)`
      the new parameter, used in their `core.RenderNode(node, index)`/`core.RenderPatch(result.Patch, index)`
      calls; update `buildServer`'s two call sites accordingly
- [ ] T011 In [internal/core/markdown_test.go](../../internal/core/markdown_test.go): add a package-level
      `testIndex` fixture (`core.Index{Predicates: map[string]core.PredicateDef{...}}`) covering exactly the
      predicates this file's existing tests reference (`mentions: {Role: "link"}`, `mentionedIn: {Role:
      "link"}`, `replaces: {Role: "edge"}`, `broader`/`conformsTo`/any other predicate literal already present
      in a fixture — grep the file for every `Predicate:` string literal to enumerate the full set); update
      every existing `core.RenderNode(...)`/`core.RenderPatch(...)` call site (~20+, per research.md D6) to
      pass `testIndex` as the second argument. **Do not change any assertion yet** — this task's sole goal is
      "compiles, all existing tests still green" (Principle VI's green-before-refactor discipline); behavior-
      changing test rewrites happen in Phase 3/US3 below
- [ ] T012 Run `go build ./... && go test ./...`; confirm the entire suite is green with zero behavior change
      before proceeding to Phase 3

**Checkpoint**: Foundation ready — `Index` flows end-to-end into every render call site; rendering output is
still byte-identical to pre-feature behavior; all existing tests pass unmodified in intent.

---

## Phase 3: User Story 1 - Consistent shape for a schema-declared predicate everywhere it appears (Priority: P1) 🎯 MVP

**Goal**: A predicate's flat-vs-grouped rendering is derived from its own schema `Role`, identically wherever
that predicate occurs in the graph — never from how a particular document happened to be shaped.

**Independent Test**: Author two nodes by hand (one with a `link`-role predicate written flat, one with an
`edge`-role predicate written grouped); run them through arc's render path and confirm each corrects to its
schema-declared shape.

### Tests for User Story 1 (write first, MUST fail before implementation below)

- [ ] T013 [P] [US1] In `internal/core/markdown_test.go`, rewrite
      `TestRenderNodeEdgesFlatBulletedListNoGroupedHeadings` (research.md D8): same fixture (`entity` node
      with `replaces`/`mentions`/`mentionedIn` edges) — assert `replaces` renders as a flat
      `"- replaces:: [[SSL Protocol]]"` bullet with **no** heading, while `mentions`/`mentionedIn` **each**
      render under their own `"## Mentions"`/`"## MentionedIn"` heading (labels default-capitalized per
      `testIndex`, since neither declares an explicit `Label`); rename the test to reflect the new behavior
      (e.g. `TestRenderNodeSchemaDrivenFlatAndGroupedMixOnOneNode`). This test MUST fail against Phase 2.5's
      still-always-flat implementation
- [ ] T014 [P] [US1] In `internal/core/markdown_test.go`, add `TestRenderNodeLinkRolePredicateUsesCustomLabel`:
      a `link`-role predicate whose `PredicateDef.Label` is non-empty in `testIndex` (e.g. mirror
      `CorePredicateDefs`'s real `"required": {Label: "Requires"}`) renders its heading as `"## Requires"`,
      not the default-capitalized predicate name
- [ ] T015 [P] [US1] In `internal/core/markdown_test.go`, add
      `TestRenderNodeUnregisteredPredicateDefaultsToFlatEdge`: a predicate absent from `testIndex` entirely
      renders as a flat bullet with no heading (spec FR-013, research.md D3)
- [ ] T016 [P] [US1] In `cmd/arc/graph/subgraph_test.go`, add a new case seeding a fixture entity with a
      `link`-role predicate occurrence (`mentions`) plus an `edge`-role occurrence, run `arc subgraph`, and
      assert the exported patch's Markdown shows the `mentions` occurrence grouped under `"## Mentions"` while
      the edge-role occurrence stays a flat bullet — proving the schema-driven shape survives the full
      `appgraph.Subgraph` → `resolveIndexOrDefault` → `core.RenderPatch` path, not just the unit-level function

### Implementation for User Story 1

- [ ] T017 [US1] In `internal/core/markdown.go`, add `resolveRenderRole(index Index, predicate string)
      string` (research.md D3, mirroring `merge.go`'s `resolveMergeOp` exactly: `index.Predicates[predicate]`
      lookup, `.Role` if found, else `"edge"`) and a label-resolution step inline in `renderNodeBody` (or a
      small `labelFor(index Index, predicate string) string` helper: `index.Predicates[predicate].Label` if
      non-empty, else `titleCaseType(predicate)`, research.md D4) (depends on T006)
- [ ] T018 [US1] In `internal/core/markdown.go`'s `renderNodeBody`, replace the current unconditional "render
      every `Edges` entry as one flat bulleted list" block (~lines 787-793) with the partition algorithm from
      data-model.md/contracts/render-shape-contract.md: partition `n.Edges` by `resolveRenderRole`; render the
      edge-role bucket as one bare bulleted list (`renderLinkBullet` per line, original relative order,
      unchanged format) first; then render each link-role bucket (grouped by predicate name) as `"## " +
      labelFor(...) + "\n"` followed by its occurrences, groups ordered by resolved label ascending
      (`sort.Strings`) — physical order: flat list, then heading blocks, landing in `walkNodeBody`'s existing
      bare-list-then-heading-blocks parser grammar unchanged (depends on T017)
- [ ] T019 [US1] Run `go test ./internal/core/... -run TestRenderNode -v` and `go test ./cmd/arc/graph/... -run
      TestSubgraph -v`; confirm T013-T016 now pass (green)

**Checkpoint**: User Story 1's E2E test (T016) and unit tests (T013-T015) pass; a predicate's rendered shape
is fully schema-driven and independently testable/demoable.

---

## Phase 4: User Story 2 - Heading omitted when a type's body is a single link-role predicate (Priority: P2)

**Goal**: When a node's entire body is one link-role predicate's occurrences (e.g. `timeline`'s `entries`),
no redundant heading is rendered.

**Independent Test**: Write a timeline node whose only body content is `entries` occurrences — confirm no
heading; add any second predicate's content and confirm the heading reappears.

### Tests for User Story 2 (write first, MUST fail before implementation below)

- [ ] T020 [P] [US2] In `internal/core/markdown_test.go`, add
      `TestRenderNodeSingleLinkRolePredicateBodyOmitsHeading`: a `timeline`-typed `Node` whose only `Edges`
      are `entries` occurrences (role `link` in `testIndex`) renders those occurrences as a bare bulleted
      list with **no** `"## Entries"` heading
- [ ] T021 [P] [US2] In `internal/core/markdown_test.go`, add
      `TestRenderNodeSingleLinkRolePredicateHeadingReappearsWithOtherContent`: the same `entries`-only fixture
      plus one additional predicate's occurrence present in `Edges` (either a second link-role predicate or
      any edge-role predicate) causes `"## Entries"` to reappear (spec Acceptance Scenario 2, Edge Case:
      "two-or-more distinct link-role predicates" also covered here)
- [ ] T022 [US2] In `cmd/arc/graph/apply_test.go`, alongside the existing
      `TestApplyCreatesTimelineEntriesChronologically` (~line 223), add a new case (or extend it) asserting
      the generated `timeline/yearly/<YYYY>.md` file's `entries` list contains **no** `"## "` heading anywhere
      — the real end-to-end `arc apply` path for the one production type (`timeline`) this omission rule
      actually governs today

### Implementation for User Story 2

- [ ] T023 [US2] In `internal/core/markdown.go`'s `renderNodeBody` (T018's partition logic), add the
      single-group omission check (research.md D5, presence-based — **no** `Node.Type`/`TypeDef.Required`/
      `Optional` lookup): if the edge-role bucket is empty and the link-role buckets contain occurrences of
      exactly one distinct predicate name, render that one bucket as a bare bulleted list (same shape/position
      as the flat list) instead of a `"## Label"` block (depends on T018)
- [ ] T024 [US2] Run `go test ./internal/core/... -run TestRenderNodeSingleLinkRole -v` and `go test
      ./cmd/arc/graph/... -run TestApplyCreatesTimelineEntries -v`; confirm T020-T022 now pass (green)

**Checkpoint**: User Stories 1 AND 2 both pass their tests independently; `timeline` nodes render with zero
superfluous headings (spec SC-004).

---

## Phase 5: User Story 3 - Round-trip stability for already-canonical documents (Priority: P1)

**Goal**: Reading and re-writing a node already in canonical schema-driven shape produces byte-identical
output; a non-canonical input normalizes toward the canonical shape rather than preserving its original shape.

**Independent Test**: Take a node already in canonical shape, run it through the read/write path, diff against
the original — the diff must be empty. Take a node in a shape inconsistent with its predicates' declared
roles, run it through the same path, and confirm it normalizes.

### Tests for User Story 3 (write first, MUST fail before implementation below)

- [ ] T025 [P] [US3] In `internal/core/markdown_test.go`, rewrite
      `TestCosmeticExceptionGroupedHeadingFlattensOnRoundTrip` (research.md D8): rename to reflect
      normalization-toward-role, not always-toward-flat (e.g.
      `TestNormalizationCorrectsShapeTowardPredicateRole`); keep the existing `boldLabelThreeBlocksPatch`-based
      case asserting content preservation (`Predicate`/`Target` survive, spec FR-010), but assert the
      re-rendered shape matches each involved predicate's `testIndex`-declared role (grouped stays/becomes
      grouped if `link`-role, flattens if `edge`-role) instead of asserting "no `## ` anywhere"; add a second,
      new sub-case: a node whose `link`-role predicate was originally written as a flat bullet (the opposite
      direction) is asserted to become grouped on re-render
- [ ] T026 [P] [US3] In `internal/core/markdown_test.go`, extend `TestIdempotentRoundTrip` (or add a sibling
      test) with a fixture mixing an edge-role and a link-role predicate on one node — assert
      `RenderNode(ParseNode(RenderNode(n, testIndex)), testIndex)` is byte-equal to `RenderNode(n, testIndex)`
      (spec FR-008), extending the existing pattern at line ~1003 to the new mixed-shape case rather than only
      the previously-all-flat one
- [ ] T027 [US3] Add `TestRenderPatchStableAcrossHeadingGroupReordering` (or extend an existing `RenderPatch`
      test): confirm that re-rendering never reorders anything beyond what contracts/render-shape-contract.md
      permits (heading-block position by label, edge-list-vs-link-groups position) — no `Link`'s
      `Predicate`/`Target`/`Alias` is ever altered, dropped, or duplicated (spec FR-010)

### Implementation for User Story 3

- [ ] T028 [US3] Verify T018/T023's implementation already satisfies T025-T027 as written (round-trip
      stability and normalization are expected to fall out of the partition algorithm's determinism — sorting
      link-role groups by label ascending, D2/D5 — rather than needing new production code); if any case
      fails, fix `renderNodeBody`'s ordering/grouping logic in `internal/core/markdown.go` until it does
      (depends on T018, T023)
- [ ] T029 [US3] Run `go test ./internal/core/... -v` in full; confirm every rewritten and new test in Phases
      3-5 passes together with zero regressions in unrelated existing tests

**Checkpoint**: User Stories 1, 2, AND 3 all pass their tests independently; round-trip byte-stability and
schema-driven normalization are both verified (spec SC-001, SC-002).

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories, beyond what's already required above.

- [ ] T030 [P] Run [quickstart.md](quickstart.md) Scenarios A-D manually against a real `arc init`-seeded
      graph, confirming the written scenarios' expected output actually matches
- [ ] T031 [P] Run `staticcheck ./...` and confirm it is clean on every file this feature touched
      (`internal/core/markdown.go`, `internal/app/schema/service/schema.go`,
      `internal/app/graph/service/apply.go`, `cmd/arc/graph/subgraph.go`, `cmd/arc/graph/serve.go`)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase).

### Design Phase Verification

- [ ] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the rendering-behavior clarification from T003
      (Principle I)
- [ ] TN02 No new domain concept was introduced beyond T004's confirmation (Principle II)
- [ ] TN03 N/A — no command/flag surface change (Principle IX; Phase 2b)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 No new architectural pattern introduced; no new ADR required (Principle I)
- [ ] TN05 `internal/core` stays free of `cmd`/Cobra imports; `RenderNode`/`RenderPatch` remain pure functions
      (no `context.Context`, no I/O) (Principle III)
- [ ] TN06 T013-T016, T020-T022, T025-T027 were written and confirmed failing (red) before their
      corresponding implementation tasks (T017-T019, T023-T024, T028-T029) (Principle VI)
- [ ] TN07 All new/changed tests use `github.com/fogfish/it/v2` exclusively — no `testify`/stdlib-only
      comparisons introduced (Principle VI)
- [ ] TN08 No Bash script was used to validate unit-level rendering correctness (Principle VI)
- [ ] TN09 N/A — no new external integration/adapter (Principle VII)
- [ ] TN10 N/A — no terminal-output/color change; `arc subgraph`/`arc serve`'s stdout content changes
      (headings appear) but their styling/TTY/`NO_COLOR` handling is untouched (Principle X)
- [ ] TN11 N/A — no new configuration value or secret (Principle XI)
- [ ] TN12 N/A — no new/changed command help text (Principle XII)
- [ ] TN13 E2E tests (T016, T022) turned GREEN and changed minimally beyond what T013-T027 already specify
      (Principle VIII)
- [ ] TN14 Every spec.md acceptance scenario (US1 1-3, US2 1-2, US3 1-2) has a passing test per the T005
      mapping (Principle VIII)
- [ ] TN15 Release/versioning impact: this changes `arc subgraph`/`arc serve`'s human-readable Markdown
      *content* (not their `--json` schema or command/flag surface) — per constitution Principle XIV, only
      `--json`/`--plain` are stable scriptable contracts, so this is a minor/patch-level content change, not a
      breaking one requiring a major version bump

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS Phase 2.5 and all user stories
- **Foundational (Phase 2.5)**: Depends on Phase 2 — BLOCKS all user stories (every story's tests need the new
  `Index` parameter to even compile)
- **User Story 1 (Phase 3, P1)**: Depends on Phase 2.5 — no dependency on US2/US3
- **User Story 2 (Phase 4, P2)**: Depends on Phase 2.5 **and** Phase 3 (T018's partition logic is what T023
  extends with the omission special case — not independent of US1's implementation, only of US1's *tests*)
- **User Story 3 (Phase 5, P1)**: Depends on Phase 2.5, Phase 3, **and** Phase 4 (its own acceptance criteria
  are defined in terms of "canonical shape," which includes US2's omission rule — see the priority-order note
  above)
- **Additional Polish**: Depends on Phases 3-5 all complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### Within Each User Story

- Tests (written first, confirmed red) → implementation → confirm green, per task ordering above
- Story complete before moving to the next phase

### Parallel Opportunities

- T007-T010 (four different call-site files) can run in parallel once T006 lands
- Within each story's "Tests" block, tasks marked [P] touch independent test functions in the same file and
  can be drafted in parallel by different people, though they'll typically be committed together since they
  share one file (`internal/core/markdown_test.go`)
- T030/T031 (Additional Polish) can run in parallel with each other

---

## Parallel Example: Phase 2.5 (Foundational)

```bash
# After T006 (core signature change) lands, these four call-site updates are independent files:
Task: "Update internal/app/schema/service/schema.go's Seed()/registerIfAbsent per T007"
Task: "Update internal/app/graph/service/apply.go's nodeContentChanged/writeNode per T008"
Task: "Update cmd/arc/graph/subgraph.go's humanSubgraphPrinter/RunE per T009"
Task: "Update cmd/arc/graph/serve.go's buildServer/handlers per T010"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions
3. Complete Phase 2.5: Foundational (`Index` threaded everywhere, behavior unchanged, all tests still green)
4. Complete Phase 3: User Story 1 — schema-driven flat/grouped rendering, no omission special case yet
5. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenario A)
6. Ship if the omission special case (US2) and explicit round-trip hardening (US3) can follow in a fast-follow

### Incremental Delivery

1. Setup + Design Preconditions + Foundational → clean, compiling baseline with zero behavior change
2. Add User Story 1 → verify against Phase N subset → demo (MVP)
3. Add User Story 2 → verify → demo (timeline nodes stop showing a redundant heading)
4. Add User Story 3 → verify → demo/ship (round-trip guarantees locked down and asserted)
5. Each story adds value without breaking the previous stories' tests

---

## Notes

- [P] tasks = different files (or clearly independent test functions), no dependencies
- [Story] label maps task to specific user story for traceability
- This feature's three user stories are **not** independently deployable slices in the usual sense (they share
  one function, `renderNodeBody`) — "independently testable" here means each has its own dedicated test(s)
  that isolate its specific behavior claim, not that US2/US3 could ship without US1's code existing
- Commit after each phase checkpoint (T012, T019, T024, T029)
- Phase 2 and Phase N sections are retained per constitution Governance > Task List Requirements; several
  Phase 2 subsections are marked N/A with justification rather than populated, since this feature adds no new
  command/flag/adapter/config surface (plan.md Constitution Check already established this)
