---

description: "Task list for 032-arc-stats implementation"
---

# Tasks: Graph Statistics (`arc stats`)

**Input**: Design documents from `/specs/032-arc-stats/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Per Principles VI and VIII, all 16 acceptance scenarios map 1:1 to E2E tests, written before implementation (red-green-refactor).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 — maps to spec.md user stories
- Exact file paths included in every task

## Path Conventions

Per [plan.md](./plan.md) Structure Decision:

- `cmd/arc/graph/` — the `arc stats` Cobra command and its colocated E2E tests
- `internal/app/graph/kernel/` — domain result types (no cobra, no fsys)
- `internal/app/graph/service/` — the `Stats` use-case
- `cmd/arc/graph/testdata/stats/` — E2E fixtures

**Every new `.go` file MUST carry the license header from [CLAUDE.md](../../CLAUDE.md).**

---

## Phase 0: Pre-implementation Refactoring

**Purpose**: [research D2](./research.md) — `walkNodeFiles` skips `_schema/`, but FR-002 must count schema documents. Parameterize the skip predicate rather than fork the walker (Principle V).

**MUST be a separate PR from feature work. All existing tests MUST pass untouched — that is the proof the refactor is behavior-preserving.**

- [X] T000 Extract `walkFiles(store fsys.Store, skipDir func(string) bool) ([]string, error)` from the existing `walkNodeFiles` body in `internal/app/graph/service/grep.go`, preserving the sorted-output and `.md`-only behavior exactly
- [X] T001 Re-express `walkNodeFiles(store)` in `internal/app/graph/service/grep.go` as a wrapper over `walkFiles` skipping `.arc` and `_schema`, and add `walkGraphFiles(store)` skipping `.arc` only
- [X] T002 Verify the refactor is behavior-preserving: `go test ./internal/app/graph/... ./cmd/arc/...` passes with **zero edits** to any existing test file

**Checkpoint**: Both walkers exist; Grep/Subgraph/Match/Context behavior is provably unchanged.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the working baseline. This feature adds **no new dependency** — it reads files already reachable through `internal/adapter/fsys` and parses them with `internal/core`.

- [X] T003 Confirm the baseline is green: `go build ./... && go test ./... && go vet ./...`
- [X] T004 [P] Confirm `staticcheck` runs clean (matches `.github/workflows/check-code.yml`), so new-package findings are attributable to this feature
- [X] T005 [P] Confirm no `go.mod` change is required — no new module may be added by this feature ([plan.md](./plan.md) Technical Context)

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate — the deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T006 Record this feature's domain terms — *type breakdown*, *stub node*, *ingestion period node*, *census population*, *content population* — in the Glossary additions section of [data-model.md](./data-model.md), staged for `ARCHITECTURE.md`. **`ARCHITECTURE.md` does not exist** (a pre-existing gap the constitution's own v1.2.0 sync report tracks); do NOT create a stub file solely to host these terms — see [plan.md](./plan.md) Complexity Tracking
- [X] T007 Verify no new domain type duplicates an existing one: confirm `isStub` (`internal/app/graph/service/apply.go`) is reused for FR-014's stub definition rather than re-derived, and that `core.Node`/`core.Link` carry every field the statistics need

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T008 Confirm the command surface in [contracts/cli-contract.md](./contracts/cli-contract.md) matches sibling commands: `arc stats`, `cobra.NoArgs`, no command-specific flags, flat top-level registration
- [X] T009 [P] Confirm the `--json` v1 schema in [contracts/stats-json-contract.md](./contracts/stats-json-contract.md) is complete, including invariants C1–C9 and the array-ordering rules that make C7 achievable

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T010 [P] Confirm **no new adapter is needed**: `Stats` reuses `fsys.Mounter`/`fsys.Store` and adds no external system ([research D10](./research.md)). Verify no `os.*` filesystem call is introduced outside `internal/adapter/fsys`
- [X] T011 Confirm **no port interface is needed**: unlike `arc lint`, `Stats` reads no git history and therefore takes no `port.VCS` — this is what makes the read-only guarantee (FR-023) structural rather than conventional

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> **Ordering note**: these tests are authored against the signatures scaffolded in Phase 2.5 (T017–T019). Author 2.5's stubs first so the tests **compile**, then write the tests here so they **fail semantically** — which is exactly what the constitution's red phase requires ("minimal structure to make tests compile is acceptable"). No test may be skipped or marked pending, and no "red phase" comment may appear.

- [X] T012 [P] [US1] Write 5 E2E tests in `cmd/arc/graph/stats_test.go` for US1 scenarios 1.1–1.5 per the map in [quickstart.md](./quickstart.md), using the existing `sut()` helper from `cmd/arc/graph/apply_test.go` (red phase)
- [X] T013 [P] [US2] Write 8 E2E tests in `cmd/arc/graph/stats_test.go` for US2 scenarios 2.1–2.8 (red phase)
- [X] T014 [P] [US3] Write 3 E2E tests in `cmd/arc/graph/stats_test.go` for US3 scenarios 3.1–3.3, asserting against the documented `--json` schema, not merely "non-empty" (red phase)
- [X] T015 [P] Write the cross-command agreement test `TestStatsBrokenLinksMatchLint` in `cmd/arc/graph/stats_agreement_test.go`: run both `applint.Lint` and `graph.Stats` over the `broken/` fixture and assert the broken-link count equals the `linkResolves` violation count (**invariant C5 / SC-003**, [research D6](./research.md)) (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T016 Confirm **not applicable**: this feature introduces no configuration value, reads no environment variable of its own, and handles no secret. Verbosity/JSON/quiet/color come from existing root persistent flags

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Compile-enabling scaffolding shared by all three stories. Stubs only — no counting logic.

- [X] T017 [P] Define `StatsResult`, `StatsDetail`, `TypeCount`, `PredicateCount`, `PeriodCount`, `TargetCount`, `RefRank`, `SchemaCoverage`, `ContentVolume` in `internal/app/graph/kernel/stats.go` with the exact JSON tags from [contracts/stats-json-contract.md](./contracts/stats-json-contract.md); `Detail` MUST be `*StatsDetail` tagged `json:"detail,omitempty"` ([research D4](./research.md))
- [X] T018 Add the `Stats(ctx, mounter, index, dir, detail bool) (kernel.StatsResult, error)` primary port to `internal/app/graph/component.go` as a thin delegator, and a stub `service.Stats` in `internal/app/graph/service/stats.go` returning a not-implemented error
- [X] T019 Scaffold `NewStatsCmd()` in `cmd/arc/graph/stats.go` with `RunE` returning a not-implemented error, and register it in `cmd/arc/root.go` alongside `lint`

**Checkpoint**: Phase 2d tests compile and fail semantically. Foundation ready.

---

## Phase 3: User Story 1 - See the shape and health of a graph at a glance (Priority: P1) 🎯 MVP

**Goal**: One command reports total nodes, total edges, per-type breakdown, broken-link count, and per-year ingestion — the screen that turns an opaque folder of Markdown into something a person can reason about.

**Independent Test**: Run `arc stats` against the `known/` fixture and confirm every summary figure equals its hand-counted value; against `empty/` confirm zeros and exit 0; against `notgraph/` confirm refusal and non-zero exit.

### Implementation for User Story 1

> E2E tests T012 and T015 were written in Phase 2d and MUST currently be failing.

- [X] T020 [P] [US1] Build the six fixture graphs under `cmd/arc/graph/testdata/stats/` (`known/`, `empty/`, `broken/`, `moved/`, `messy/`, `notgraph/`) per [quickstart.md](./quickstart.md), each with a hand-counted composition recorded in a `README.md` beside it so assertions compare against independently known values (SC-002)
- [X] T021 [US1] Implement the single-walk population split in `internal/app/graph/service/stats.go`: `walkGraphFiles` → parse each file → census population (`core.LooksLikeNode` accepts) vs content population (census minus `@type` `Class`/`Property`), accumulating `Unreadable` and `Foreign` separately ([research D1, D3, D8](./research.md))
- [X] T022 [US1] Implement `Nodes`, `ByType` (grouped by declared `@type`, never by path), and `Edges` (occurrence count over the content population) in `internal/app/graph/service/stats.go` — satisfies FR-002, FR-003, FR-004, FR-004a
- [X] T023 [US1] Implement `BrokenLinks` in `internal/app/graph/service/stats.go`: flatten `HRefs`+`Edges` per content node, count distinct unresolved targets **per node** against the content-population basename set ([research D6](./research.md)) — satisfies FR-006, invariant C5
- [X] T024 [US1] Implement yearly `Ingestion` in `internal/app/graph/service/stats.go` by counting entries in `@type: Timeline` nodes whose `@id` parses as `2006`, taking period identity from the node id, not its path ([research D5](./research.md)) — satisfies FR-007, FR-008
- [X] T025 [US1] Implement the `guardIsGraph` refusal reusing `service.ErrNotAGraph` in `internal/app/graph/service/stats.go` — satisfies FR-011; and confirm the empty-graph path returns zeros rather than an error (FR-010)
- [X] T026 [US1] Apply the summary ordering rules from [data-model.md](./data-model.md) — `ByType` by count desc then name asc, `Ingestion` by period asc — sorting **in the service**, never in the printer, so `--json` is deterministic too (FR-005, FR-019, FR-021)
- [X] T027 [US1] Implement `RunE` and the human printer in `cmd/arc/graph/stats.go`: mount via `fsys.Local{}`, resolve the schema index via `appschema.Resolve`, report progress via `bios.NewReporter(bios.Quiet, !bios.Verbose)`, render through a `bios.Registry[kernel.StatsResult]`. The report MUST state both counting rules per FR-006a
- [X] T028 [US1] Add unit tests in `internal/app/graph/service/stats_test.go` for the population split, the per-node distinct-target rule, and period-code parsing, using `github.com/fogfish/it/v2` against a fake `fsys.Store`
- [X] T029 [US1] Populate `Short`, `Long`, and `Example` on the command in `cmd/arc/graph/stats.go` per [contracts/cli-contract.md](./contracts/cli-contract.md) (Principle XII)

**Checkpoint**: US1 E2E tests (T012) and the agreement test (T015) pass. `arc stats` is fully functional as an MVP.

---

## Phase 4: User Story 2 - Investigate the graph in depth (Priority: P2)

**Goal**: `--verbose` adds edges by predicate, monthly ingestion, connectivity health, degree and hubs, schema coverage, and content volume.

**Independent Test**: Run `arc stats --verbose` against `known/` and `broken/` and confirm each detail figure equals its hand-counted value; run without `--verbose` and confirm no detail figure appears.

### Implementation for User Story 2

> E2E test T013 was written in Phase 2d and MUST currently be failing.

- [X] T030 [US2] Thread the `detail bool` parameter from `RunE` through `graph.Stats` into `service.Stats` in `cmd/arc/graph/stats.go`, `internal/app/graph/component.go`, and `internal/app/graph/service/stats.go`, computing `StatsDetail` only when true and leaving `Detail` nil otherwise ([research D4](./research.md)) — satisfies FR-018
- [X] T031 [P] [US2] Implement `ByPredicate` and `IngestionMonthly` in `internal/app/graph/service/stats.go` — satisfies FR-012, FR-013, invariants C2 and C3
- [X] T032 [P] [US2] Implement `Orphans`, `Stubs` (reusing the existing `isStub` shape), and `UnresolvedTargets` with a per-target referencing-node count in `internal/app/graph/service/stats.go` — satisfies FR-014, invariant E/C4
- [X] T033 [P] [US2] Implement `AvgOutDegree`, `MedianOutDegree` (even population → mean of the two central values), and `TopReferenced` (top 10, refs desc then id asc) in `internal/app/graph/service/stats.go` — satisfies FR-015
- [X] T034 [P] [US2] Implement `SchemaCoverage` — Class/Property declared and used counts, plus undeclared predicates — and `TypeCount.Declared`, in `internal/app/graph/service/stats.go` — satisfies FR-016
- [X] T035 [P] [US2] Implement `ContentVolume` — inline refs counted separately from structural edges, attribute values, nodes without `Published` — in `internal/app/graph/service/stats.go` — satisfies FR-017
- [X] T036 [US2] Apply the detail ordering rules from [data-model.md](./data-model.md) to every detail slice in `internal/app/graph/service/stats.go` (FR-019, FR-021)
- [X] T037 [US2] Implement the verbose human printer in `cmd/arc/graph/stats.go` as the `Verbose` entry of the `bios.Registry[kernel.StatsResult]`, styled with `lipgloss` (Principle X — no raw ANSI escapes)
- [X] T038 [US2] Add unit tests in `internal/app/graph/kernel/stats_test.go` for the derived invariants (C1–C4) and in `internal/app/graph/service/stats_test.go` for median-with-even-population and the top-10 tie-break

**Checkpoint**: US1 and US2 both pass their E2E tests independently.

---

## Phase 5: User Story 3 - Track graph growth from a script or pipeline (Priority: P3)

**Goal**: `--json` emits a structured document mirroring the requested verbosity, stable enough to diff across time.

**Independent Test**: Run `arc stats --json` and `arc stats --json --verbose` against `known/`, confirm both parse and match the documented schema, and confirm two consecutive runs are byte-identical.

### Implementation for User Story 3

> E2E test T014 was written in Phase 2d and MUST currently be failing.

- [X] T039 [US3] Verify `--json` routes through the existing `jsonPrinter` with no `bios` change: `bios.ResolveMode()` returns `ModeJSON`, and the nil `Detail` pointer with `omitempty` yields an **absent** key rather than `null` ([research D4](./research.md)) — satisfies FR-020a, invariant C6
- [X] T040 [US3] Confirm nothing but the document reaches stdout in JSON mode — progress goes to stderr via the reporter — in `cmd/arc/graph/stats.go` (FR-020, Principle X)
- [X] T041 [US3] Verify determinism end to end: add `TestStatsJSONDeterministic` assertions comparing raw bytes across two runs in `cmd/arc/graph/stats_test.go`, proving the Phase 3/4 service-level sorting eliminated Go's randomized map iteration order (FR-021, SC-005, invariant C7)

**Checkpoint**: All three stories pass their E2E tests independently.

---

## Additional Polish

- [X] T042 [P] Add `TestStatsIgnoresFolderLayout` in `cmd/arc/graph/stats_test.go` asserting the `moved/` and `known/` fixtures produce identical results — the test that makes FR-004a/C9 real rather than aspirational
- [X] T043 [P] Add `TestStatsSeparatesUnreadableFromForeign` in `cmd/arc/graph/stats_test.go` against the `messy/` fixture ([research D8](./research.md), FR-009)
- [X] T044 [P] Add `TestStatsDoesNotModifyGraph` in `cmd/arc/graph/stats_test.go`, snapshotting the fixture tree before and after (FR-023, SC-007)
- [X] T045 [P] Add `TestStatsPerformance` in `cmd/arc/graph/stats_test.go`: generate a 10,000-node graph in a temp directory and assert completion under 5 seconds (SC-004)
- [X] T046 [P] Update `README.md` with `arc stats` in the command reference
- [X] T047 Run the full [quickstart.md](./quickstart.md) validation command set and confirm every Definition-of-Done box

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). Retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — **blocked by the file's non-existence**; the two-population model and the walker refactor are documented in [research.md](./research.md) D1/D2 instead, and the gap is recorded in [plan.md](./plan.md) Complexity Tracking
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — staged in [data-model.md](./data-model.md) pending that file's creation (see T006)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — assess whether the two-population model warrants an ADR or is adequately covered by [research.md](./research.md) D1
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration (T010/T011)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A (T016)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — 16 scenarios, 16 tests
- [X] TN15 Release/versioning impact assessed (Principle XIV) — expected **minor**: a new command and a new `--json` schema are additive; confirm no existing command's output changed, which the Phase 0 untouched-tests requirement (T002) already evidences

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 0 (Refactoring)**: No dependencies — **separate PR**, must land first
- **Phase 1 (Setup)**: No dependencies
- **Phase 2 (Design Preconditions)**: Depends on Setup — BLOCKS all user stories
- **Phase 2.5 (Foundational)**: **Interleaved with Phase 2d** — T017–T019 scaffold the signatures that T012–T015 compile against. Author the stubs, then the tests; the checkpoint is "tests compile and fail semantically"
- **Phase 3+ (User Stories)**: All depend on Phase 2 and 2.5
- **Polish**: Depends on all three stories
- **Phase N**: Final gate

### User Story Dependencies

- **US1 (P1)**: Depends only on Phase 2/2.5. Delivers the MVP alone
- **US2 (P2)**: Depends on Phase 2/2.5 **and on T021's population split from US1** — the detail figures are derived from the same parsed node set. This is the one genuine cross-story dependency; US2 is independently *testable* but not independently *implementable*
- **US3 (P3)**: Depends on Phase 2/2.5 and on whichever stories' figures it serializes. T039–T041 are verification-heavy because the JSON path is inherited, not built

### Within Each User Story

- Fixtures before assertions (T020 before every US1 test can pass)
- Population split (T021) before any figure derived from it
- Service-level sorting (T026, T036) before the determinism proof (T041)
- Story complete before moving to the next priority

### Parallel Opportunities

- T004, T005 in Phase 1
- T009, T010 across Phase 2b/2c
- T012, T013, T014, T015 — all four test-authoring tasks target distinct scenarios and can be written in parallel once T017–T019 scaffolding exists
- **T031–T035** — the five detail-figure groups touch independent accumulators over the same parsed node set; the largest parallel block in the feature
- T042–T046 in Polish

---

## Parallel Example: User Story 2

```bash
# Foundation in place: T030 has threaded the detail flag through.
# Launch the five detail-figure groups together:
Task: "Implement ByPredicate and IngestionMonthly in internal/app/graph/service/stats.go"
Task: "Implement Orphans, Stubs, UnresolvedTargets in internal/app/graph/service/stats.go"
Task: "Implement AvgOutDegree, MedianOutDegree, TopReferenced in internal/app/graph/service/stats.go"
Task: "Implement SchemaCoverage and TypeCount.Declared in internal/app/graph/service/stats.go"
Task: "Implement ContentVolume in internal/app/graph/service/stats.go"
```

> These five share one file. Parallelize them across separate commits or as one developer's sequential batch; do not run them as concurrent edits to `stats.go`.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 0 — land the walker refactor as its own PR
2. Phase 1 — Setup
3. Phase 2 — Design Preconditions (CRITICAL, blocks everything)
4. Phase 2.5 — Scaffolding, interleaved with 2d test authoring
5. Phase 3 — User Story 1
6. Phase N — Constitution verification
7. **STOP and VALIDATE**: `arc stats` reports a correct summary; the C5 agreement test passes

### Incremental Delivery

1. Phase 0 refactor → merged, existing behavior provably unchanged
2. Setup + Design Preconditions + Foundational → foundation ready
3. Add US1 → verify → **MVP ships**
4. Add US2 → verify → verbose ships
5. Add US3 → verify → machine-readable ships

### Risk Notes

- **T023 and T015 are the feature's correctness core.** SC-003 is the only requirement binding this command to another; if the agreement test is hard to make pass, the population split (T021) is the first place to look, not the counting rule
- **T041's determinism failure mode is intermittent.** Go randomizes map iteration per run, so unsorted output can pass locally and flake in CI. If T026/T036 are skipped, T041 will not reliably catch it — treat the sorting tasks as load-bearing, not cosmetic
- **T020's fixtures gate everything.** Every assertion compares against a hand-counted value; a wrong fixture count produces a test that passes against wrong behavior

---

## Notes

- [P] = different files, no dependencies
- Commit after each task or logical group
- Every new `.go` file needs the license header from [CLAUDE.md](../../CLAUDE.md)
- Principle IV: no inline comments outside GoDoc, no function over 25 lines — the statistics decompose naturally into one small fold per figure group
- Phase 2 and Phase N retained per Governance > Task List Requirements
