# Tasks: Graph Schema as a First-Class Citizen (`_schema/`)

**Input**: Design documents from `/specs/005-graph-schema-first-class/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (`schema-contract.md`), quickstart.md, [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario MUST map 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1/US2/US3, priorities P1/P2/P3 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1, US2, or US3 — maps to spec.md's three user stories
- Exact file paths are included in every description

## Path Conventions (from plan.md)

- `internal/app/schema/{kernel,service}/` + `component.go` — the new `schema` domain use-case; deliberately **no** `port`/`adapter` subdirectory (research.md D2/D5: no use-case-private external dependency beyond the already-shared `internal/adapter/fsys`)
- `internal/app/graph/port/` — gains `schema.go`'s `SchemaRegistry` interface (graph-private)
- `internal/core` — `rules.go` deleted entirely; AST/merge-algebra/codec files unchanged
- `internal/app/config` — `port/`, `adapter/http/`, `adapter/mock/` deleted; `kernel`/`service`'s `Load`/`Save` kept
- `internal/app/ctrl`, `internal/app/graph`, `internal/app/lint` — existing packages, signature/wiring changes only, no new subpackages
- `cmd/arc/{ctrl,graph,lint}/` — existing Cobra commands, wiring changes only, no new command

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create the package skeleton: `internal/app/schema/{kernel,service}/` directories and an empty `internal/app/schema/component.go` per plan.md's Project Structure — no `port`/`adapter` subdirectories (research.md D2/D5)
- [ ] T002 [P] Confirm no new third-party dependency is required — `go.mod` stays unchanged per plan.md Technical Context; note that `net/http` becomes unused by this feature once `internal/app/config/adapter/http` is deleted (Phase 2.5)
- [ ] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the new (still-empty) package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T004 Add **Schema Document**, **Node-Kind Schema Document**, **Predicate Schema Document**, and **Discovered Kind / Predicate** (data-model.md) to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary; rewrite the existing **Metadata Stub** and **Kind Registration** entries, both now retired (Principle I obligation, plan.md Constitution Check row I, research.md D10)
- [ ] T005 Verify no existing `internal/<domain>` package already defines a schema-shaped type before introducing `kernel.CoreMergeRules`/`kernel.CorePredicates` in `internal/app/schema/kernel` (none exist — this is the project's first schema-vocabulary package)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [ ] T006 Confirm this feature introduces **zero** new/changed Cobra commands or flags against contracts/schema-contract.md — no `arc schema` command; `arc init`/`arc apply`/`arc lint`'s existing flag surface is unchanged
- [ ] T007 [P] Review contracts/schema-contract.md (already produced during `/speckit-plan`) for completeness against spec.md's functional requirements — no changes expected, this is a gate check

### Phase 2c: External Integration & Adapter Design (Principle VII, if applicable)

- [ ] T008 [P] Confirm `internal/app/schema` introduces no new external system integration — its only I/O is the already-shared `internal/adapter/fsys`, consumed directly with no private port wrapper (research.md D2/D5, mirroring `internal/app/ctrl/service.Init`'s existing precedent of taking `fsys.Mounter` as a plain parameter)
- [ ] T009 Define `internal/app/graph/port/schema.go`'s `SchemaRegistry` interface shape (`RegisterKind`, `RegisterPredicate` only) per contracts/schema-contract.md — the design gate before any adapter/mock code is written (ADR 001 port isolation rule 1)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [ ] T010 [P] [US1] Write E2E tests in `cmd/arc/ctrl/init_test.go` for spec.md US1's 5 acceptance scenarios (`_schema/nodes`+`_schema/predicates` populated with every core kind/predicate; each node-kind document's `id`/`kind: schema`/`merge`; each predicate document's `id`/`kind: schema`; no `_meta/` folder and no merge-rule content anywhere; initialization succeeds with no network access) using the `sut()` helper — tests MUST compile and fail semantically (red phase)
- [ ] T011 [P] [US2] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md US2's 4 acceptance scenarios (a patch introducing an unregistered kind creates its schema document and still applies successfully using the union default; a patch introducing an unregistered predicate creates its schema document; an already-registered kind/predicate is left unchanged, not duplicated; new schema documents land in the same commit as the triggering patch, verified via `git show --stat`) — red phase
- [ ] T012 [P] [US3] Write an E2E test in `cmd/arc/graph/apply_test.go` for spec.md US3's Acceptance Scenario 3 (hand-editing a registered kind's `_schema/nodes/<kind>.md` `merge` value changes the behavior a later `arc apply` invocation actually uses) — red phase
- [ ] T013 [P] Write E2E tests in `cmd/arc/lint/lint_test.go` for the spec.md Clarifications (Q1/Q3, FR-015): a freshly initialized graph's `_schema/` documents never appear in `arc lint`'s checked-node count or violation list, and an ordinary content node sharing a basename with a schema document (e.g. `entities/hypothesis.md` vs. `_schema/nodes/hypothesis.md`) is not reported as a basename collision — red phase

### Phase 2e: Configuration & Secrets Review (Principle XI, if applicable)

- [ ] T014 Confirm this feature introduces no new configuration surface: `internal/app/config`'s `Load`/`Save` infrastructure is kept but has zero callers after this feature ships (plan.md Complexity Tracking, research.md D8); no secret or credential material is involved anywhere in `internal/app/schema`

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: `internal/app/schema`'s full `Seed`/`Resolve`/`RegisterKind`/`RegisterPredicate` surface, the `internal/core`/`internal/app/config` cleanup that makes room for it, and the `graph.port.SchemaRegistry` interface are shared by every user story — US1 wires `Seed`, US2 wires `Resolve`+`RegisterKind`+`RegisterPredicate`, US3 verifies `Resolve`'s hand-edit sensitivity. This phase builds that shared foundation once; Phase 3+ wires each caller.

### `internal/core` cleanup (research.md D1)

- [ ] T015 Delete `internal/core/rules.go` in its entirety — its content is redistributed into `internal/app/schema/kernel` (T021) and `internal/app/config/kernel` (T018)
- [ ] T016 [P] Update `internal/core/rules_test.go`: remove test cases for the deleted `CoreMergeRules`/`KnownProfileMergeRules`/`ConfigPath`/`MergeRuleSet`'s YAML marshal/unmarshal methods; keep `Union`/`Lookup` test cases against the still-present `MergeRuleSet` type definition (depends on T015)

### `internal/app/config` cleanup (research.md D8)

- [ ] T017 Delete `internal/app/config/port/`, `internal/app/config/adapter/http/`, and `internal/app/config/adapter/mock/` in their entirety (the retired "github downloader")
- [ ] T018 Remove the `MergeRules core.MergeRuleSet` field from `internal/app/config/kernel/config.go`'s `Config` struct; add `ConfigPath = ".arc/config.yml"` there (moved from `internal/core`, research.md D1) (depends on T015)
- [ ] T019 Remove `service.Default`/`service.Resolve` from `internal/app/config/service/config.go` and their `component.go` exports; keep `Load`/`Save` unchanged (depends on T017, T018)
- [ ] T020 [P] Update `internal/app/config/service/config_test.go`: remove `Default`/`Resolve` test cases; keep/adjust the `Load`/`Save` round-trip test against the now-field-empty `Config` (depends on T019)

### `internal/app/schema` build-out

- [ ] T021 [P] Implement `internal/app/schema/kernel/schema.go`: `SchemaKind core.Kind = "schema"`, `NodesDir = "_schema/nodes"`, `PredicatesDir = "_schema/predicates"` path constants, `CoreMergeRules` (moved from `internal/core/rules.go`, T015 — 4 entries: `source:none, entity:union, resource:union-first-writer, timeline:append`), `CorePredicates` (new — the 13 ARCNET-CORE §7.4 names with one-line descriptions each, research.md D7) (depends on T015)
- [ ] T022 [P] Unit tests in `internal/app/schema/kernel/schema_test.go`: `CoreMergeRules` has exactly the 4 expected entries; `CorePredicates` has exactly 13 distinct, camelCase names (depends on T021)
- [ ] T023 Implement `Seed()` in `internal/app/schema/service/schema.go` per data-model.md: for every `CoreMergeRules` entry, render a `core.Node{ID: kind, Kind: schema.SchemaKind, Attrs: {"merge": op}}` via `core.RenderNode` keyed at `NodesDir/<kind>.md`; for every `CorePredicates` entry, render `core.Node{ID: predicate, Kind: schema.SchemaKind}` keyed at `PredicatesDir/<predicate>.md`; pure function, no `context.Context`, no network call (research.md D5) (depends on T021)
- [ ] T024 [P] Unit test for `Seed()` in `internal/app/schema/service/schema_test.go`: returns exactly 17 entries; every entry's rendered content round-trips through `core.ParseNode` back to the expected `id`/`kind`/`merge` (depends on T023)
- [ ] T025 Implement `Resolve(store fsys.Store) (core.MergeRuleSet, map[string]bool, error)` in `internal/app/schema/service/schema.go` per contracts/schema-contract.md: walks `NodesDir`/`PredicatesDir`, `core.ParseNode`s each file; a file that fails to parse is **skipped, not an error** (spec.md Edge Cases); an absent `_schema/` folder resolves to two empty results, not an error (mirrors `config.Resolve`'s retired "absent file is not an error" precedent) (depends on T021)
- [ ] T026 [P] Unit tests for `Resolve` in `internal/app/schema/service/schema_test.go`: a well-formed folder (e.g. `Seed()`'s own output) round-trips correctly; a malformed individual document is skipped without erroring the whole call; an absent `_schema/` folder returns two empty results with a nil error (depends on T025)
- [ ] T027 Implement `RegisterKind(store, kind) (created bool, err error)` and `RegisterPredicate(store, predicate) (created bool, err error)` in `internal/app/schema/service/schema.go` per contracts/schema-contract.md: create-if-absent via `core.RenderNode`+`store.Create`; `created=false` and no write when the path already exists (spec FR-011); `RegisterKind` always writes `merge: union` (spec FR-010, clarified — never any other value) (depends on T021)
- [ ] T028 [P] Unit tests for `RegisterKind`/`RegisterPredicate` in `internal/app/schema/service/schema_test.go`: creates the expected file exactly once with the expected content; a second call against the same kind/predicate returns `created=false` and does not modify the already-present file's content (depends on T027)
- [ ] T029 [P] Implement `internal/app/schema/service/errors.go`: an `ErrSchemaWrite` `faults.Safe1[string]` sentinel for a write failure inside `RegisterKind`/`RegisterPredicate`, matching the codebase's existing `faults.Type`/`.With()` convention
- [ ] T030 [P] Implement `internal/app/schema/component.go`: primary port `Seed()`, `Resolve(store)`, `RegisterKind(store, kind)`, `RegisterPredicate(store, predicate)` — thin delegators into `service`, per contracts/schema-contract.md (depends on T023, T025, T027)
- [ ] T031 [P] Write `internal/app/schema/README.md` documenting the `schema` use-case per ADR 001's layout convention, noting the deliberate absence of `port`/`adapter` subdirectories (research.md D2/D5)

### `internal/app/graph/port` — the structural bridge (research.md D3)

- [ ] T032 [P] Implement `internal/app/graph/port/schema.go`'s `SchemaRegistry` interface (`RegisterKind`, `RegisterPredicate`, matching T009's design) — satisfied structurally by `internal/app/schema`'s concrete component, no explicit `implements` needed (depends on T009)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Bootstrap a graph with a first-class, versioned schema (Priority: P1) 🎯 MVP

**Goal**: `arc init` creates `_schema/nodes/` and `_schema/predicates/`, pre-populated with every ARCNET-CORE kind and predicate as its own readable, committed document; no `_meta/` folder and no merge-rule configuration exist anywhere; initialization remains fully offline-capable; a freshly initialized graph passes `arc lint` cleanly with zero `_schema/`-related noise.

**Independent Test**: Run `arc init` against an empty directory and inspect the resulting file tree and one commit — per quickstart.md Scenarios 1-2.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010, T013) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [ ] T033 [US1] Update `internal/app/ctrl/kernel/graph.go`'s `DefaultLayout`: `Folders` drops `"_meta"`, adds `"_schema/nodes"` and `"_schema/predicates"` (using the same literal path values as `schema.kernel.NodesDir`/`PredicatesDir` — no cross-use-case import, kept in sync by both deriving from research.md D7); rename the `MetaStubs map[string]string` field to `SeedFiles map[string]string`, now empty by default (research.md D9)
- [ ] T034 [US1] Rename `configSeed []byte` to `schemaSeed map[string]string` in `internal/app/ctrl/service/init.go`'s `Init` and `internal/app/ctrl/component.go`'s matching signature; merge `schemaSeed` into a per-call copy of `layout.SeedFiles` before `writeLayout`, exactly where `configSeed` was previously merged in at `core.ConfigPath` (depends on T033)
- [ ] T035 [US1] Update `rollback()` in `internal/app/ctrl/service/init.go` to remove every path in the merged `layout.SeedFiles` instead of the deleted `core.ConfigPath`-specific removal (depends on T034)
- [ ] T036 [US1] Update `cmd/arc/ctrl/init.go`: replace the `appconfig.Default(ctx, newConfigFetcher())` call with `appschema.Seed()`; pass the result as `Init`'s `schemaSeed` argument; delete the now-unused `fetchConfigSeed`/`newConfigFetcher` helpers (depends on T030, T034)
- [ ] T037 [P] [US1] Update `internal/app/ctrl/service/init_test.go` and `cmd/arc/ctrl/init_test.go`'s existing fixtures/mocks referencing `configSeed`/`MetaStubs`/`_meta` to the new `schemaSeed`/`SeedFiles`/`_schema` shape (depends on T034, T036)

### `arc lint` exemption (needed immediately once `_schema/` exists — FR-015, Clarifications Q1/Q3)

- [ ] T038 [US1] In `internal/app/lint/service/lint.go`'s `walkNodeFiles`, add `if full == "_schema" { continue }` alongside the existing `.arc` skip; delete `excludedMetaFiles` and its two now-obsolete entries (research.md D6)
- [ ] T039 [US1] In `internal/app/lint/service/rules_predicates.go`, delete `parsePredicateRegistry` and `predicatesPath`; `internal/app/lint/service/lint.go`'s `Lint` gains a `predicates map[string]bool` parameter, used directly at the former `parsePredicateRegistry` call site; reword `checkPredicateRegistered`'s violation message to name `_schema/predicates/` instead of the retired path (depends on T038)
- [ ] T040 [US1] Update `internal/app/lint/component.go`'s `Lint` signature to add the `predicates` parameter, thin delegation otherwise unchanged (depends on T039)
- [ ] T041 [US1] Update `cmd/arc/lint/lint.go`: replace `appconfig.Resolve(store)` with `appschema.Resolve(store)`, passing both returned values (`rules`, `predicates`) into `applint.Lint` (depends on T030, T040)
- [ ] T042 [P] [US1] Update `internal/app/lint/service/lint_test.go` and `rules_predicates_test.go`'s existing fixtures referencing `_meta`/`predicatesPath`/`parsePredicateRegistry` to the new `predicates`-parameter shape (depends on T039)

**Checkpoint**: At this point, User Story 1's E2E tests (T010, T013) pass — a fresh `arc init` produces a fully-seeded, first-class `_schema/` folder with no `_meta/` folder and no merge-rule content in `.arc/config.yml`, and an immediate `arc lint` run reports a clean pass with zero `_schema/`-related noise

---

## Phase 4: User Story 2 - Schema grows automatically as new content is ingested (Priority: P2)

**Goal**: `arc apply` recognizes a previously-unseen node kind or predicate, registers it into `_schema/` in the same commit as the triggering patch, and still applies the patch's content successfully using the safe default merge behavior.

**Independent Test**: Apply a patch introducing a previously-unseen node kind or predicate and confirm a corresponding schema document exists immediately afterward, in the same commit — per quickstart.md Scenarios 3-4.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red).

- [ ] T043 [US2] `internal/app/graph/service/apply.go`: `Apply`'s signature gains `predicates map[string]bool` and `schema port.SchemaRegistry` parameters (depends on T032)
- [ ] T044 [US2] Inside `Apply`'s existing per-node loop, immediately after the existing `op, ok := rules.Lookup(node.Kind)` miss (line ~144), additionally call `schema.RegisterKind(store, node.Kind)` — alongside the existing, unchanged union-default-plus-warning behavior (research.md D3) (depends on T043)
- [ ] T045 [US2] After `merged` is computed (which carries `.Links`/`.Edges`), collect every distinct predicate name from `merged.Links`'s keys and every non-empty `Link.Predicate` in `merged.Edges`; for each name absent from `predicates`, call `schema.RegisterPredicate(store, name)` (research.md D4) (depends on T043)
- [ ] T046 [US2] Update `internal/app/graph/component.go`'s `Apply` signature to add the same two parameters, thin delegation otherwise unchanged (depends on T044, T045)
- [ ] T047 [US2] Update `cmd/arc/graph/apply.go`: replace `appconfig.Resolve(store)` with `appschema.Resolve(store)`; construct the real `internal/app/schema` component as the `port.SchemaRegistry` argument; pass `predicates` through into `appgraph.Apply` (depends on T030, T046)
- [ ] T048 [P] [US2] Update `internal/app/graph/service/apply_test.go`'s existing fixtures/mocks for `Apply`'s two new parameters (a fake `SchemaRegistry`, an empty/populated `predicates` map) (depends on T043)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently — a patch introducing a previously-unseen kind or predicate registers it into `_schema/` in the same commit as the patch's own content, and a second application of the same kind/predicate no longer produces the unrecognized-kind warning

---

## Phase 5: User Story 3 - Understand and curate the graph's schema directly (Priority: P3)

**Goal**: A hand-edit to a registered kind's `_schema/nodes/<kind>.md` `merge` value is the behavior a later `arc apply` actually uses — confirming the schema folder is genuinely first-class, versioned, editable content, not a read-only cache of some other source of truth.

**Independent Test**: Edit a node kind's schema document's declared merge behavior, apply a further contribution to that kind, and confirm the edited behavior — not the original — is the one applied, per quickstart.md Scenario 5.

### Implementation for User Story 3

> E2E test for this story was already written in Phase 2d (T012) and MUST currently be failing (red) until Phase 4 lands; this phase verifies/hardens the exact behavior against US3's own acceptance scenario.

- [ ] T049 [US3] Verify (via T012's E2E test) that a hand-edited `merge` value in an existing `_schema/nodes/<kind>.md` is what `internal/core.Merge`'s `op` parameter actually receives on a later `arc apply` invocation — this already follows from `cmd/arc/graph/apply.go` calling `appschema.Resolve` fresh on every invocation (T047); adjust T044's call site only if T012 reveals a stale-read issue (depends on T047)
- [ ] T050 [P] [US3] Add a unit test in `internal/app/schema/service/schema_test.go`: `Resolve` reflects a hand-edited `merge` value in an existing `_schema/nodes/<kind>.md` file exactly as `core.ParseNode` parses it, with no caching across calls (depends on T025)

**Checkpoint**: All three user stories pass their E2E tests independently — the schema folder is fully self-describing, auto-extending, and human-editable

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [ ] T051 [P] Update `README.md`'s quick-start example if it currently mentions `_meta/`/`.arc/config.yml` merge rules (constitution Principle XII)
- [ ] T052 [P] Manually run all 6 quickstart.md scenarios against the built binary and confirm expected output/exit codes
- [ ] T053 [P] Add table-driven unit tests in `internal/app/schema/service/schema_test.go` covering every guard/error path end-to-end against a fake `fsys.Store`, asserting `errors.Is(err, service.ErrSchemaWrite)` where applicable (constitution Principle VI)

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [ ] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes: `internal/app/schema` (no `port`/`adapter` subdirectory), the retired `internal/app/config` fetcher, and `internal/core/rules.go`'s removal (Principle I)
- [ ] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [ ] TN03 Command/flag surface matches the Phase 2b design exactly: zero new/changed commands or flags across `arc init`/`arc apply`/`arc lint` (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced beyond what ADR 001/002 already cover (Principle I) — none expected; confirm during review that the `internal/core`/`schema` boundary decision (research.md D1) is fully explained in this plan rather than requiring a new ADR
- [ ] TN05 Domain logic uses ports (interfaces); `graph.port.SchemaRegistry` is satisfied structurally with no direct `internal/app/graph` → `internal/app/schema` import (Principle III, ADR 001 rule 1)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [ ] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [ ] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [ ] TN09 `internal/app/schema` introduces no new external integration (no adapter added); the one new port (`graph.port.SchemaRegistry`) follows the existing structural-satisfaction pattern with no vendor type leaking through it (Principle VII)
- [ ] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`; no new styling introduced beyond the existing `internal/bios` kernel (Principle X)
- [ ] TN11 Configuration precedence respected; `internal/app/config`'s remaining `Load`/`Save` infra introduces no new configuration surface; no secrets logged (Principle XI)
- [ ] TN12 No command/flag help text changes required — confirm no drift was introduced incidentally (Principle XII)
- [ ] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [ ] TN14 All spec.md US1–US3 acceptance scenarios have a passing, colocated E2E test in `cmd/arc/ctrl/init_test.go`, `cmd/arc/graph/apply_test.go`, and `cmd/arc/lint/lint_test.go` (Principle VIII)
- [ ] TN15 Release/versioning impact assessed: `--json` schemas for `arc init`/`arc apply`/`arc lint` gain no removed/renamed fields (additive only, if any); removing `internal/app/config`'s HTTP fetcher removes an internal dependency, not a scriptable-output contract change — no major-version implication (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion; the `internal/core`/`internal/app/config` cleanup (T015-T020) must land before `internal/app/schema`'s build-out (T021-T031) can reuse the freed-up path constants
- **User Stories (Phase 3+)**: All depend on Phase 2.5; User Story 1 is the deepest since it also carries the `arc lint` exemption every later story's fixtures rely on not breaking; User Story 2 depends on User Story 1 only in the sense that its E2E fixtures assume an already-`_schema/`-seeded graph (a real `arc init`-produced fixture), not on US1's code changes directly; User Story 3 is a verification/hardening phase over User Story 2's `Resolve` call site
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories
- **User Story 2 (P2)**: Can start after Phase 2.5; its E2E fixtures need a real, `_schema/`-seeded graph, i.e. User Story 1's `arc init` changes, as test setup — but its own implementation tasks (T043-T048) touch entirely different files (`internal/app/graph/*`) than US1's (`internal/app/ctrl/*`, `internal/app/lint/*`)
- **User Story 3 (P3)**: Depends on User Story 2's `cmd/arc/graph/apply.go` wiring (T047) existing to verify against — a pure verification/hardening phase, adds no new production code path

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Domain/foundation (Phase 2.5) before any story's implementation tasks
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked `[P]` can run in parallel
- Phase 2a-2e subsections marked `[P]` can run in parallel with each other
- Within Phase 2.5: the `internal/core` cleanup (T015-T016) and `internal/app/config` cleanup (T017-T020) can proceed in parallel with each other; `internal/app/schema`'s build-out (T021-T031) depends on both finishing first (it reuses the freed `CoreMergeRules` content and `ConfigPath` placement)
- Within Phase 3: the `ctrl` layout changes (T033-T037) and the `lint` exemption (T038-T042) touch entirely different files and can proceed in parallel
- Once Phase 3 lands, User Story 2 (Phase 4) can begin; User Story 3 (Phase 5) starts once Phase 4's T047 lands

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Launch independent foundational tasks together, once T015/T017-T020 land:
Task: "Implement internal/app/schema/kernel/schema.go (CoreMergeRules, CorePredicates, path constants)"
Task: "Implement internal/app/schema/service/schema.go's Seed()"
Task: "Implement internal/app/schema/service/schema.go's Resolve()"
Task: "Implement internal/app/schema/service/schema.go's RegisterKind/RegisterPredicate"
Task: "Implement internal/app/schema/service/errors.go"
Task: "Implement internal/app/graph/port/schema.go's SchemaRegistry interface"
```

## Parallel Example: Phase 3 User Story 1

```bash
# Once Phase 2.5 lands, launch US1's two independent file groups together:
Task: "Update internal/app/ctrl/kernel/graph.go, service/init.go, component.go, and cmd/arc/ctrl/init.go for schemaSeed"
Task: "Update internal/app/lint/service/lint.go, rules_predicates.go, component.go, and cmd/arc/lint/lint.go for the _schema exemption and predicates parameter"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md Scenarios 1-2 against the built binary
7. Deploy/demo if ready — `arc init` already produces a fully first-class schema at this point, missing only auto-discovery (US2) and the hand-edit-takes-effect confirmation (US3)

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo
4. Add User Story 3 → Verify against Phase N → Deploy/Demo
5. Each story adds value without breaking previous stories

---

## Notes

- `[P]` tasks = different files, no dependencies
- `[Story]` label maps a task to its user story for traceability
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
- **No Phase 0 (Pre-implementation Refactoring) is included**, despite this feature deleting `internal/core/rules.go` and most of `internal/app/config`: unlike `specs/003-apply-patch`'s git-adapter promotion (a genuine no-behavior-change refactor that could stand as its own PR with all existing tests still passing), this deletion is not separable from the new feature — there is no meaningful intermediate state where `arc init` seeds neither the old `.arc/config.yml` merge rules nor the new `_schema/` folder, and existing tests referencing the deleted symbols cannot pass until the replacement (T021-T032) exists. The cleanup is instead folded into Phase 2.5, which already blocks every user story.
- FR-010/FR-011 (always-safe-default registration, never-overwrite) and FR-015 (schema documents exempt from ordinary lint content rules) are enforced by construction in T027/T038 respectively — no separate task double-checks them beyond their own unit/E2E tests
