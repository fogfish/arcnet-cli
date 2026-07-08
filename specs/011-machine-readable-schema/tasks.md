# Tasks: Machine-Readable Predicate & Type Schema

**Input**: Design documents from `/specs/011-machine-readable-schema/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`schema-document-contract.md`, `schema-index-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to a test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story. Because this feature introduces one shared domain type (`core.Index`) every consumer depends on, Go's compiler enforces a single, whole-module Foundational phase before any story-specific behavior can even build — see Phase 2.5, mirroring `specs/010-predicate-node-model`'s precedent.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `internal/core/rules.go` + `_test.go` sibling — new `Index`/`PredicateDef`/`TypeDef`, retiring `MergeRuleSet`
- `internal/app/schema/{kernel,service}/**` + `component.go` — the schema use-case's own reshape (no `port`/`adapter` of its own, unchanged precedent)
- `internal/app/{graph,lint,ctrl}/**` + their `_test.go` siblings — mechanical signature-propagation ripple (no new business logic beyond what's noted)
- `cmd/arc/{ctrl,graph,lint}/**` — call-site updates plus E2E tests; no new flags/commands
- No new package, no new command, no new port/adapter (plan.md Structure Decision)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the ground this feature builds on, before touching any file

- [X] T001 Confirm no new package/directory is needed — this feature reshapes the existing `internal/app/schema` use-case and mechanically updates its existing consumers (plan.md Project Structure); no scaffolding step required
- [X] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged (plan.md Technical Context: reuses existing `goldmark`/`goldmark-meta`/`yaml.v3`/`fogfish/faults`/`fogfish/it/v2`)
- [X] T003 [P] Run `staticcheck ./...` and `go build ./...` and confirm both pass clean before any change, establishing a baseline to diff against

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Update [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary: replace the **Node-Kind Schema Document**/**Predicate Schema Document** entries with **Type Schema Node**/**Predicate Schema Node** (data-model.md), add a **Schema Index** entry, update **Canonical Folder** (`_schema/nodes/`→`_schema/types/`) and **Merge Behavior** (now sourced from a Type Schema Node's `merge` bridge field, spec FR-015) (plan.md Constitution Check rows I/II obligation)
- [X] T005 Verify no existing `internal/core` type already models `Index`/`PredicateDef`/`TypeDef` before introducing them — confirmed none exists (research.md D1)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm this feature introduces **zero** new/changed Cobra commands, flags, or help text — `arc apply`'s existing auto-registration warning changes wording only (kind→type terminology) (gate check, no `cmd/` flag/command changes)

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [X] T007 Confirm this feature introduces no new external integration or adapter — the only I/O touched (`fsys.Store` via the existing, unmodified `Store`/`Mounter`) is unchanged; no new port

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T008 [P] [US1] Write E2E tests in `cmd/arc/ctrl/init_test.go` for spec.md US1's 4 acceptance scenarios (every core predicate registered at `_schema/predicates/<name>.md` with role/merge/label?/aligned? plus description; every core type registered at `_schema/types/<name>.md` with required/optional bullets plus description; no `_schema/nodes/` folder exists; offline `arc init` still succeeds from built-in defaults) — tests MUST compile and fail semantically (red phase)
- [X] T009 [P] [US2] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US2's 5 acceptance scenarios (a newly discovered predicate is auto-registered as a full `Property` node with role/merge, not a stub; a newly discovered type is auto-registered as a full `Class` node with empty required/optional; an already-registered predicate/type document is left unchanged; auto-registered schema documents land in the triggering patch's own commit; a missing/malformed `_schema/` document aborts `arc apply` before any write) — red phase
- [X] T010 [P] [US3] Write unit tests in `internal/app/schema/service/schema_test.go` and an E2E test in `cmd/arc/lint/lint_test.go` for spec.md US3's 4 acceptance scenarios (`Resolve` reports every registered predicate's role/merge/label/aligned; `Resolve` reports every registered type's required/optional sets; `arc apply` and `arc lint` recognize identical predicate/type sets from the same graph; editing a predicate's role or a type's required/optional list changes what the next `Resolve` call reports) — red phase

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [X] T011 Confirm this feature introduces no new configuration surface and no secret/credential material anywhere in `internal/core` or the packages it ripples into

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `core.Index` (replacing `core.MergeRuleSet`/`map[string]bool`) is the one shared value every user story and every downstream package depends on — nothing compiles until it and its mechanical ripple through `internal/app/schema`, `internal/app/graph`, `internal/app/lint`, `internal/app/ctrl`, and their `cmd/` callers are done (plan.md Summary). This phase builds that shared foundation once; Phase 3+ then only needs to turn each story's own E2E tests from Phase 2d green.

### `internal/core` — shared schema-index type (research.md D1, D9; contracts/schema-index-contract.md)

- [X] T012 [P] Add `PredicateDef{Role, Merge, Label, Aligned, Description}`, `TypeDef{Merge, Required []string, Optional []string, Description}`, and `Index{Predicates map[string]PredicateDef, Types map[string]TypeDef}` to `internal/core/rules.go`; delete `MergeRuleSet` and its `.Lookup`/`.Union` methods (research.md D1, D9)
- [X] T013 [P] Unit tests in `internal/core/rules_test.go`: `Index.Predicates`/`Index.Types` lookup (present/absent via plain map access), `PredicateDef`/`TypeDef` field construction — written first, red phase (depends on T012 existing as a compile target)

**Checkpoint**: `core.Index` exists; nothing outside `internal/core` compiles yet — expected, next steps fix that

### `internal/app/schema/kernel` — full CORE vocabulary seed data (research.md D6)

- [X] T014 Rename `NodesDir` → `TypesDir = "_schema/types"` in `internal/app/schema/kernel/schema.go` (keep `PredicatesDir` unchanged)
- [X] T015 Replace `CoreMergeRules`/`CorePredicates`/`coreKindDescriptions`/`KindDescription` with `CorePredicateDefs map[string]core.PredicateDef` — every predicate CORE §10 documents: identity (`@id`, `@type`), content (`tags`, `text`), metadata/control (`published`, `created`, `updated`), structural (`mentions`, `mentionedIn`), semantic (`broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`), citation (`cites`, `citesAsEvidence`, `citesAsAuthority`, `supports`, `confirms`, `extends`, `critiques`, `disputes`, `refutes`, `isCitedBy`), type-specific (`title`, `abstract`, `authors`, `url`, `doi`, `category`, `aliases`, `definition`, `notes`, `ref`, `year`, `status`, `relevance`, `granularity`, `entries`, `heading`), schema-own (`role`, `merge`, `label`, `aligned`, `description`, `required`, `optional`) — and `CoreTypeDefs map[string]core.TypeDef` for `source`/`entity`/`resource`/`timeline` (CORE §11) plus `Property`/`Class` themselves, each with its `Required`/`Optional`/`Description`/`Merge` (research.md D6) in `internal/app/schema/kernel/schema.go` (depends on T012)
- [X] T016 [P] Unit tests in `internal/app/schema/kernel/schema_test.go`: `CorePredicateDefs` contains every name T015 lists, each with a valid `Role` (one of meta/text/href/edge/link) and `Merge`; `CoreTypeDefs` contains `source`/`entity`/`resource`/`timeline`/`Property`/`Class`, each with non-empty `Description` and CORE-§11-correct `Required` lists — written first, red phase (depends on T015 existing as a compile target)

**Checkpoint**: `internal/app/schema/kernel` carries the full CORE vocabulary as typed data; `go test ./internal/app/schema/kernel/...` passes

### `internal/app/schema/service` — Seed/Resolve/Register rewrite (research.md D2, D4, D5; contracts/schema-document-contract.md)

- [X] T017 Rewrite `Seed()` in `internal/app/schema/service/schema.go`: render one conformant `Property` node per `CorePredicateDefs` entry (`Attrs["role"]`/`["merge"]` mandatory, `["label"]`/`["aligned"]` when the entry declares one, `Texts["description"]`) and one conformant `Class` node per `CoreTypeDefs` entry (`Attrs["merge"]`, `Texts["description"]`, `Edges` with `Predicate: "required"`/`"optional"` bullets) — schema-document-contract.md (depends on T015)
- [X] T018 Rewrite `Resolve(store fsys.Store) (core.Index, error)` in `internal/app/schema/service/schema.go`: check `.arc/` presence first, returning the existing "not an initialized graph" error family if absent (research.md D2); then walk `_schema/predicates/`/`_schema/types/`, decoding each document into `PredicateDef`/`TypeDef`, failing the entire load — never skipping — on an absent `_schema/` subfolder or any document missing/invalid `role`/`merge`/`@type`/description (spec FR-014) (depends on T012, T017)
- [X] T019 [P] Add `ErrSchemaMissing`/`ErrSchemaInvalid` (`faults.Type`/`faults.SafeN`, naming the offending file and field) in `internal/app/schema/service/errors.go`, alongside the existing `ErrSchemaWrite` (depends on T018)
- [X] T020 Rename `RegisterKind` → `RegisterType` in `internal/app/schema/service/schema.go`; write a conformant `Class` node (`merge: union`, empty `Required`/`Optional`, a placeholder description) for a newly discovered type, never overwriting an existing document (research.md D5, D7)
- [X] T021 Rewrite `RegisterPredicate` in `internal/app/schema/service/schema.go`: write a conformant `Property` node (`role: edge`, `merge: union`, a placeholder description) for a newly discovered predicate, never overwriting an existing document (research.md D5)
- [X] T022 [P] Rewrite `internal/app/schema/service/schema_test.go` against a fake `fsys.Store`: `Seed`/`Resolve`/`RegisterType`/`RegisterPredicate` with the new richer shapes, the malformed-document fail-fast path, the missing-`_schema/`-folder fail-fast path, and the not-a-graph-checked-first path — written first, red phase (depends on T017-T021 existing as compile targets)

**Checkpoint**: `internal/app/schema` fully reshaped; `go test ./internal/app/schema/...` passes

### `internal/app/schema` — primary port + docs

- [X] T023 [P] Update `internal/app/schema/component.go`: `Resolve` returns `core.Index`; `RegisterKind` → `RegisterType` (depends on T018, T020)
- [X] T024 [P] Update `internal/app/schema/README.md`: `_schema/types/` (not `/nodes/`), the richer Property/Class document shape, `core.Index` (depends on T014, T017)

### `internal/app/ctrl` — folder layout rename

- [X] T025 [P] Update `internal/app/ctrl/kernel/graph.go`: `DefaultLayout.Folders` — `"_schema/nodes"` → `"_schema/types"` (depends on T014)

### `internal/app/graph` — signature ripple (no business-logic change beyond noted renames)

- [X] T026 [P] Update `internal/app/graph/port/schema.go`: `SchemaRegistry.RegisterKind` → `RegisterType` (depends on T020)
- [X] T027 [P] Update `internal/app/graph/kernel/apply.go`: `Warnings` message wording "kind" → "type" terminology (research.md D7)
- [X] T028 Update `internal/app/graph/service/apply.go`: `Apply(..., index core.Index, ...)` replaces `rules core.MergeRuleSet, predicates map[string]bool`; `rules.Lookup(node.Type)` → `index.Types[node.Type]` presence+`.Merge` lookup; `predicates[name]` → `index.Predicates[name]` presence; `schema.RegisterKind` → `schema.RegisterType` call rename; `coreKindFolders`/`nodeFolder`/`pluralizeKind` UNCHANGED (plan.md Scale/Scope, spec.md Assumptions) (depends on T012, T026, T027)
- [X] T029 [P] Update `internal/app/graph/component.go`: `Apply`'s `core.MergeRuleSet` param → `core.Index` (depends on T028)
- [X] T030 [P] Update `internal/app/graph/service/apply_test.go` fixtures to the new `Index` param plus the richer schema-document shape (depends on T028)

### `internal/app/lint` — signature ripple (no business-logic change beyond noted renames)

- [X] T031 [P] Update `internal/app/lint/component.go`: `Lint`'s `core.MergeRuleSet` param → `core.Index` (depends on T012)
- [X] T032 Update `internal/app/lint/service/lint.go`: `Lint(..., index core.Index, dir)` replaces `rules`/`predicates` params, passed through to `checkUnrecognizedKind`/`checkPredicateRegistered` (depends on T012, T031)
- [X] T033 [P] Update `internal/app/lint/service/rules_frontmatter.go`: `checkUnrecognizedKind(node, path, index core.Index)` checks `index.Types[node.Type]` presence instead of `core.MergeRuleSet.Lookup` (depends on T032)
- [X] T034 [P] Update `internal/app/lint/service/rules_predicates.go`: `checkPredicateRegistered(..., index.Predicates)` — same intent, new backing type; `citoPredicates` UNCHANGED (spec.md Assumptions) (depends on T032)
- [X] T035 [P] Update `internal/app/lint/service/{lint_test.go,rules_frontmatter_test.go,rules_predicates_test.go}` fixtures to the new `Index` param (depends on T032, T033, T034)

### `cmd/arc` — call-site updates (no CLI-visible change beyond T006's noted wording)

- [X] T036 [P] Confirm `cmd/arc/ctrl/init.go`'s `schemaSeed()` still wraps `appschema.Seed()` unchanged in shape (depends on T017)
- [X] T037 [P] Update `cmd/arc/graph/apply.go`: `appschema.Resolve(store)` now returns `core.Index`; pass straight through to `appgraph.Apply` (depends on T018, T028)
- [X] T038 [P] Update `cmd/arc/lint/lint.go`: `appschema.Resolve(store)` now returns `core.Index`; pass straight through to `applint.Lint` (depends on T018, T032)
- [X] T039 Run `go build ./...` and `staticcheck ./...` and confirm the whole module compiles and lints clean (depends on T012-T038)

**Checkpoint**: Foundation ready — `go build ./...` succeeds; Phase 2d's tests (T008-T010) can now actually run (red or green) instead of failing to compile

---

## Phase 3: User Story 1 - A freshly initialized graph fully describes its own vocabulary (Priority: P1) 🎯 MVP

**Goal**: `arc init` seeds a graph whose every core predicate and type is a real, machine-readable document — no existence-only stubs, no `_schema/nodes/` folder.

**Independent Test**: Initialize a fresh graph and inspect `_schema/predicates/`/`_schema/types/` per quickstart.md Scenario 1.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T008) and MUST currently be failing (red) until Phase 2.5 completes. Phase 2.5 already contains all the production code this story needs — this phase confirms its own E2E tests are green with no further src changes.

- [X] T040 [US1] Confirm E2E tests T008 in `cmd/arc/ctrl/init_test.go` pass against the Phase 2.5 foundation with no further production-code changes (depends on T039, T008)
- [X] T041 [P] [US1] Add/finalize `testdata/` fixtures (or inline assertions) in `cmd/arc/ctrl/init_test.go` confirming every seeded `_schema/predicates/<name>.md` declares role/merge/label?/aligned?/description and every seeded `_schema/types/<name>.md` declares required/optional/description, per T015's full list (depends on T017, T008)
- [X] T042 [US1] Verify, via `cmd/arc/ctrl/init_test.go`, that no `_schema/nodes/` folder exists after `arc init` (depends on T025, T040)

**Checkpoint**: User Story 1's E2E tests (T008) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Applying content keeps recognizing and now fully registers new vocabulary (Priority: P2)

**Goal**: `arc apply` auto-registers a previously unseen predicate/type as a full, machine-readable document, and every schema-aware command fails fast — before any write — on a missing or malformed schema.

**Independent Test**: Apply a patch introducing a novel predicate/type into a fresh graph and inspect the resulting schema document; separately, corrupt `_schema/` and confirm `arc apply` refuses cleanly — per quickstart.md Scenario 2.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red) until Phase 2.5 completes. Phase 2.5 already contains all the production code this story needs.

- [X] T043 [US2] Confirm E2E tests T009 in `cmd/arc/graph/apply_test.go` pass against the Phase 2.5 foundation (depends on T039, T009)
- [X] T044 [P] [US2] Verify, via `cmd/arc/graph/apply_test.go`, that a newly discovered predicate/type is registered as a full conformant node (not a bare stub) and lands in the same commit as the triggering patch (spec Acceptance Scenarios 1, 2, 4) (depends on T028, T043)
- [X] T045 [P] [US2] Verify, via `cmd/arc/graph/apply_test.go` AND `cmd/arc/lint/lint_test.go`, that a missing or malformed `_schema/` document aborts the command before any write (spec Acceptance Scenario 5, FR-014) (depends on T018, T043)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - The schema becomes a reusable index, not tool-internal knowledge (Priority: P3)

**Goal**: `arc apply` and `arc lint` consult the identical, correctly-decoded `core.Index` built from a graph's own `_schema/` documents — no separate hardcoded copy, no guessing from Markdown shape.

**Independent Test**: Load the schema index for a graph with a mix of core and discovered vocabulary and confirm it reports correct role/merge/required/optional; edit a document and confirm the next load reflects it — per quickstart.md Scenario 3.

### Implementation for User Story 3

> Tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red) until Phase 2.5 completes.

- [X] T046 [US3] Confirm unit tests T010 in `internal/app/schema/service/schema_test.go` and the E2E test in `cmd/arc/lint/lint_test.go` pass against the Phase 2.5 foundation (depends on T039, T010)
- [X] T047 [P] [US3] Verify, via a shared fixture graph exercised by both `cmd/arc/graph/apply_test.go` and `cmd/arc/lint/lint_test.go`, that both commands recognize an identical predicate/type set (spec Acceptance Scenario 3) (depends on T043, T046)
- [X] T048 [P] [US3] Verify, via `internal/app/schema/service/schema_test.go`, that editing a predicate's declared role/merge or a type's required/optional list between two `Resolve` calls changes what the second call reports, with no code change (spec Acceptance Scenario 4) (depends on T022, T046)

**Checkpoint**: User Story 3's tests (T010) pass; all three user stories' tests are green together

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T049 [P] Manually run all three quickstart.md scenarios end-to-end against a built `arc` binary, confirming every documented output matches actual behavior
- [X] T050 [P] Remove now-dead helpers left over from the existence-only shape (`KindDescription`, `coreKindDescriptions`, any now-unreferenced `MergeRuleSet`-era helper) across every touched file
- [X] T051 Search `README.md` and any generated command reference for `_schema/nodes/` examples; update to `_schema/types/`

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — Glossary-only change (T004), no Directory Structure change
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — Schema Index, Type Schema Node, Predicate Schema Node (T004)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: zero changes (Principle IX) — confirmed by T006

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — N/A, no new pattern; `core.Index` follows the exact precedent `core.MergeRuleSet` already set
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — unchanged; no `cmd/` business logic introduced; `core.Index` placed in `internal/core` specifically to avoid a new inter-use-case kernel dependency (research.md D1)
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — T013/T016/T022 before T012/T015/T017-T021
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration (T007)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A, no output-formatting change beyond warning-message wording (T027)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A (T011)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — N/A, no command changed; the new fail-fast schema errors (T019) use `faults.Type`/`faults.SafeN`-style human-readable guidance naming the offending file/field, not a raw parse error
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII) — T008-T010 turned green by T040-T048; changes to existing E2E tests are fixture/param-shape rewrites, not new test logic
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E or unit test (Principle VIII) — all 13 acceptance scenarios across US1-US3
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — **Yes, flagged**: the on-disk `_schema/nodes/`→`_schema/types/` rename and `Resolve`'s skip-malformed→fail-fast reversal both break existing-graph compatibility with no prior deprecation warning (plan.md Complexity Tracking); accepted pre-1.0 (`0.1.x` release train) and explicitly spec-mandated, not hidden

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3-5)**: All depend on Phase 2.5 — can proceed in parallel (if staffed) or sequentially in priority order (P1 → P2 → P3)
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories
- **User Story 2 (P2)**: Can start after Phase 2.5 — independently testable, though its fixture graphs may reuse US1's seeded output
- **User Story 3 (P3)**: Can start after Phase 2.5 — exercises the same `core.Index` US1/US2 already prove correct, but is independently testable via `internal/app/schema/service/schema_test.go` alone

### Within Each User Story

- Tests (Phase 2d) already written and failing before implementation starts
- Foundational `core.Index`/`internal/app/schema` rewrite (Phase 2.5) before any story-specific verification
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- Once Phase 2.5 completes, all user stories can start in parallel (if team capacity allows)
- Tasks marked [P] within Phase 2.5 (different files, no dependency on an incomplete task) can run in parallel

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# internal/core and internal/app/schema/kernel first (everything else depends on them):
Task: "Add Index/PredicateDef/TypeDef to internal/core/rules.go, delete MergeRuleSet"
Task: "Unit tests in internal/core/rules_test.go"

# Once those compile, the mechanical ripple can fan out in parallel:
Task: "Update internal/app/graph/port/schema.go: RegisterKind -> RegisterType"
Task: "Update internal/app/graph/kernel/apply.go: Warnings wording"
Task: "Update internal/app/lint/component.go: Lint's param -> core.Index"
Task: "Update internal/app/ctrl/kernel/graph.go: _schema/nodes -> _schema/types"
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
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections MUST be retained verbatim across features (constitution Governance > Task List Requirements) — only task descriptions are adapted
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
