# Tasks: Node Provenance Timestamps (`published`/`indexed`/`updated`)

**Input**: Design documents from `/specs/009-node-timestamp-attrs/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`ast-contract.md`, `apply-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `internal/core/{ast.go,markdown.go,merge.go}` + their `_test.go` siblings — the shared, use-case-independent domain change (`Node.Published`, codec, merge algebra)
- `internal/app/graph/service/apply.go` + `apply_test.go` — the one use-case service touched (per-node create/merge loop)
- `cmd/arc/graph/{apply_test.go,subgraph_test.go}` — E2E tests only; no `cmd/` production code changes
- No new package, no new command, no new port/adapter (plan.md Structure Decision)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the ground this feature builds on, before touching any file

- [X] T001 Confirm no new package/directory is needed — this feature extends the existing `internal/core` and `internal/app/graph/service` files only (plan.md Project Structure); no scaffolding step required
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged (plan.md Technical Context: reuses existing `goldmark`/`yaml.v3`/stdlib `time`/`bytes`)
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on `internal/core` and `internal/app/graph/service` before any change, establishing a clean baseline

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add **Provenance Timestamp Attributes** (`published`/`indexed`/`updated`) and **Application Timestamp** to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary, per data-model.md (Principle I/II obligation, plan.md Constitution Check rows I/II)
- [X] T005 Verify no existing `internal/core` or `internal/app/graph` type already models a per-node timestamp concept before introducing `Node.Published` — confirmed none exists (research.md D1)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm this feature introduces **zero** new/changed Cobra commands, flags, or `--json` schema fields beyond the purely additive `Node.Published` (contracts/apply-contract.md gate check) — no `cmd/` change at all
- [X] T007 [P] Review contracts/ast-contract.md and contracts/apply-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's 11 functional requirements — gate check, no changes expected

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T008 Confirm this feature introduces no new external integration or adapter — the only I/O touched (`fsys.Store` via the existing, unmodified `writeNode`/`readExistingNode`) is unchanged; no injected `Clock` port is introduced (research.md D5) — this stays a pure Go-stdlib-only change

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T009 [P] [US1] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US1's 4 acceptance scenarios (a created ordinary node carries `published` equal to the patch's date; every node one application creates shares an identical `indexed` value; a stub-shaped patch section creates a node with neither; an auto-registered `_schema/` document carries neither), using `sut()` — tests MUST compile and fail semantically (red phase)
- [X] T010 [P] [US2] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US2's 4 acceptance scenarios (a real merge stamps `updated` identical to that run's `indexed`; a `"none"`-kind re-contribution adds no `updated`, file byte-unchanged; a stub later merged with real content fills `published` and gets `updated`, but never gains `indexed`; a no-op `union` re-contribution — nothing new contributed — adds no `updated`) — red phase
- [X] T011 [P] [US3] Write an E2E test in `cmd/arc/graph/subgraph_test.go` for spec.md US3 Acceptance Scenario 3 (a node's `published` value survives `arc subgraph` extraction unchanged) — red phase

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T012 Confirm this feature introduces no new configuration surface and no secret/credential material anywhere in `internal/core` or `internal/app/graph/service`

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `Node.Published`, its codec integration, and its merge-fill behavior are shared by every user story — US1 needs it rendered on create, US2 needs it filled/preserved on merge, US3 needs it to survive `RenderPatch`. This phase builds that shared `internal/core` foundation once; Phase 3+ wires each story's own `service.Apply` behavior on top of it.

### `internal/core` — the typed `Published` field (research.md D1, D2, D3)

- [X] T013 [P] Add `Published time.Time \`json:"published,omitempty"\`` field to `Node` in `internal/core/ast.go` (data-model.md)
- [X] T014 [P] Unit tests in `internal/core/ast_test.go`: a zero-value `Node`'s `Published.IsZero()` is true; a `Node` literal with `Published` set retains it (depends on T013)
- [X] T015 Implement `extractPublished(manifest map[string]any) (time.Time, map[string]any)` in `internal/core/markdown.go`, reusing the existing `decodeManifestDate` (depends on T013)
- [X] T016 Wire `extractPublished` into `ParseNode` (a `"published"` front-matter key is decoded into `Node.Published`, no longer left in `Attrs`) and into `parsePatchBody`'s per-node construction, immediately after `decodeYAMLBlock` returns (depends on T015)
- [X] T017 Add a `published time.Time` parameter to `renderAttrYAML` in `internal/core/markdown.go`; when non-zero, format `"2006-01-02"` and merge it into the existing sorted-attribute-keys render loop; update `renderFrontMatter` (for `RenderNode`) and `RenderPatch`'s per-node fence construction to pass `n.Published` (depends on T013)
- [X] T018 [P] Unit tests in `internal/core/markdown_test.go`: `ParseNode` extracts `"published"` into `Node.Published` and never leaves it in `Attrs`; `RenderNode`/`RenderPatch` render a non-zero `Published` back into front matter/yaml fence at its sorted position; `ParseNode(RenderNode(n))` round-trips `Published` exactly (depends on T016, T017)
- [X] T019 Implement `mergePublished(existing, incoming time.Time) time.Time` in `internal/core/merge.go` (returns `existing` if non-zero, else `incoming`); wire `merged.Published = mergePublished(existing.Published, incoming.Published)` into `mergeCore` (depends on T013)
- [X] T020 [P] Unit tests in `internal/core/merge_test.go`: `Published` fills from `incoming` when `existing.Published` is zero, for `MergeUnion`/`MergeUnionFirstWriter`/`MergeAppend`/`MergeValidatedOverwrite`; `Published` is preserved unchanged when `existing.Published` is already non-zero even if `incoming.Published` differs, and never appears in the returned `conflicts`; `MergeNone` leaves `Published` untouched, matching its existing whole-node no-op (depends on T019)

### `internal/app/graph/service` — stamping helpers (research.md D4, D6, D7)

- [X] T021 [P] Implement `isStub(node core.Node) bool` in `internal/app/graph/service/apply.go` (data-model.md) — true when `Attrs`/`Text`/`Notes`/`HRefs`/`Edges`/`Links` are all empty, the exact shape `service/subgraph.go`'s `--stubs` flag already emits
- [X] T022 [P] Implement `nodeContentChanged(existing, merged core.Node) (bool, error)` in `internal/app/graph/service/apply.go` — renders both sides via `core.RenderNode` and compares bytes (research.md D6)
- [X] T023 [P] Implement `setAttr(attrs map[string]any, key string, value any) map[string]any` in `internal/app/graph/service/apply.go` — nil-safe map-set helper, used for both `"indexed"` and `"updated"`

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Know when a node's source was published and when it entered the graph (Priority: P1) 🎯 MVP

**Goal**: Every ordinary content node a patch creates carries `published` (the patch's declared date) and `indexed` (this application's timestamp); a stub node and an auto-registered `_schema/` document carry neither.

**Independent Test**: Apply a patch for a document with a known `published` date into a fresh graph, then inspect every newly created node file — per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T024 [US1] In `internal/app/graph/service.Apply`, capture `appliedAt := time.Now().UTC()` once near the top of the function (alongside the existing Reporter-phase timers) and `stamp := appliedAt.Format(time.RFC3339)`, reused for the rest of the invocation (research.md D5) (depends on T021, T022, T023)
- [X] T025 [US1] In `service.Apply`'s per-node loop's create path (`!existed`): when `!isStub(node)`, set `node.Published = patch.Published` unless the node's own patch section already carried a non-zero `Published` (research.md D11), and `node.Attrs = setAttr(node.Attrs, "indexed", stamp)` (depends on T024)
- [X] T026 [US1] Verify (no code change beyond T025's `isStub` guard) that a stub-shaped create leaves `Published`/`indexed` both unset
- [X] T027 [US1] Verify (no code change) that `_schema/` auto-registration (the existing `schema.RegisterKind`/`RegisterPredicate` calls already inside the loop) is structurally unreachable from T025's stamping code — confirmed by research.md D8

**Checkpoint**: At this point, User Story 1's E2E tests (T009) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Know when a node was last touched by an incoming contribution (Priority: P2)

**Goal**: A merge that actually changes a node's content gets `updated`, sharing this application's `indexed` value; a `"none"`-kind or no-op re-contribution adds nothing, leaving the file byte-for-byte unchanged.

**Independent Test**: Apply a patch that merges into an already-existing node, confirm `updated` appears; re-apply an unchanged follow-up patch, confirm the file is untouched — per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red).

- [X] T028 [US2] In `service.Apply`'s merge path (`existed`), after `core.Merge` returns `merged`, call `nodeContentChanged(existing, merged)`; when true, `merged.Attrs = setAttr(merged.Attrs, "updated", stamp)` (depends on T024, T022)
- [X] T029 [US2] Verify (no code change) that `MergeNone`'s existing whole-node no-op (`Merge` returns `existing` verbatim) makes `nodeContentChanged` false automatically, so no `"none"`-kind guard is needed beyond T028
- [X] T030 [US2] Verify (no code change) that `ApplyResult.Created`/`.Merged` counters and each node's `reporter.Step` outcome string are left exactly as spec 003 implemented them, independent of `nodeContentChanged`'s result (research.md D9 — deliberate scope boundary)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Read a node's provenance directly from the file, with no separate command (Priority: P3)

**Goal**: `published`/`indexed`/`updated` are readable in plain text from any node file; `published` survives being re-serialized by another capability (`arc subgraph`).

**Independent Test**: Browse a node created/merged by a completed application and confirm its timestamps read directly; extract it via `arc subgraph` and confirm `published` is unchanged — per quickstart.md Scenario 3.

### Implementation for User Story 3

> The E2E test for this story was already written in Phase 2d (T011) and MUST currently be failing (red). This story needs no new production code beyond Phase 2.5's `renderAttrYAML`/`RenderPatch` wiring (T017) — it is a verification story.

- [X] T031 [US3] Verify (no code change) that `arc subgraph`'s existing `core.RenderPatch` call path carries a node's `Published` through unchanged, now that `renderAttrYAML` (T017) includes it in the per-node yaml fence
- [X] T032 [P] [US3] Manually run quickstart.md Scenario 3 against a built `arc` binary and confirm the extracted node's `published` value matches the original exactly, distinct from the subgraph patch's own synthetic manifest-level `published` date (research.md D11)

**Checkpoint**: User Story 3's E2E test (T011) passes; all three user stories' E2E tests are green together

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T033 [P] Manually run all three quickstart.md scenarios end-to-end against a built `arc` binary, confirming every documented output matches actual behavior

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — Glossary-only change (T004), no Directory Structure change
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — Provenance Timestamp Attributes, Application Timestamp (T004)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: zero changes (Principle IX) — confirmed by T006

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — N/A, no new pattern; this feature extends existing `internal/core`/`internal/app/graph` in place
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — unchanged; no `cmd/` code touched
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — T014/T018/T020 before T013/T015-T017/T019
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration (T008)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A, no output change
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A (T012)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — N/A, no command changed
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII) — T009-T011 turned green by T024-T031
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — all 11 acceptance scenarios across US1-US3
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — No; `Node.Published`'s addition to `kernel.SubgraphResult`'s `--json` schema is purely additive (research.md D10, plan.md Constitution Check row XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3-5)**: All depend on Phase 2.5
  - US1 has no dependency on US2/US3; US2 depends on the same `nodeContentChanged`/`stamp` machinery T024 establishes (shared with US1, not a cross-story dependency on US1's own behavior); US3 needs no new production code at all, only Phase 2.5's `renderAttrYAML` change
- **Additional Polish**: Depends on all three user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on US2/US3
- **User Story 2 (P2)**: Can start after Phase 2.5 — shares `stamp`/`nodeContentChanged` infrastructure with US1's `service.Apply` changes (T024 is written once, used by both T025 and T028), but is independently testable via T010's E2E tests regardless of US1's own create-path behavior
- **User Story 3 (P3)**: Can start after Phase 2.5 — purely a verification story once T017 ships; no dependency on US1/US2's `service.Apply` changes

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- `internal/core` changes (Phase 2.5) before `internal/app/graph/service` changes (Phase 3+)
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- T009/T010/T011 (E2E test authoring for US1/US2/US3) can run in parallel — different acceptance scenarios, though all land in `cmd/arc/graph/apply_test.go`/`subgraph_test.go` so coordinate to avoid merge conflicts in the same file
- T013/T014 (ast.go/ast_test.go), T021/T022/T023 (independent helper functions in apply.go) can run in parallel
- Once Phase 2.5 completes, US1/US2/US3 implementation can proceed in parallel (if staffed)

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Independent internal/core additions:
Task: "Add Published time.Time field to Node in internal/core/ast.go"
Task: "Unit tests for zero/non-zero Node.Published in internal/core/ast_test.go"

# Independent internal/app/graph/service helpers (different functions, same file — coordinate on apply.go):
Task: "Implement isStub(node core.Node) bool in internal/app/graph/service/apply.go"
Task: "Implement nodeContentChanged(existing, merged core.Node) (bool, error) in internal/app/graph/service/apply.go"
Task: "Implement setAttr(attrs map[string]any, key string, value any) map[string]any in internal/app/graph/service/apply.go"
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
   - Developer A: User Story 1 (create-path stamping)
   - Developer B: User Story 2 (merge-path stamping, byte-comparison)
   - Developer C: User Story 3 (verification + quickstart validation)
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies (or independent functions within `apply.go`, called out explicitly above)
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections MUST be retained verbatim across features (constitution Governance > Task List Requirements) — only task descriptions are adapted
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
