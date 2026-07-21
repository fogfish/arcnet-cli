---

description: "Task list for Per-Predicate Merge Reconciliation for arc apply"
---

# Tasks: Per-Predicate Merge Reconciliation for arc apply

**Input**: Design documents from `/specs/012-predicate-merge-policies/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/merge-behavior-contract.md](contracts/merge-behavior-contract.md), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional — every spec.md acceptance scenario maps 1:1 to an E2E test in `cmd/arc/graph/apply_test.go`, and tests are written before implementation (red-green-refactor). This feature is unusual in that all three user stories are different test angles on **one shared engine** (`internal/core.Merge`'s new per-predicate dispatch) rather than independently-implemented slices — so all tests (E2E and unit, per story) are written and red in Phase 2d, and Phase 2.5's single Foundational implementation turns all of them green at once. Phases 3-5 are per-story confirmation checkpoints, not separate implementation.

**Organization**: Tasks are grouped by user story for traceability; see the note above for why implementation itself is concentrated in Phase 2.5.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

- `internal/core/` — shared domain tier (ADR 001); MUST NOT import `internal/app/*` or Cobra
- `internal/app/graph/service/`, `internal/app/schema/{kernel,service}/` — the two use-cases this feature touches
- `cmd/arc/graph/` — the sole primary adapter (Cobra), E2E tests colocated here
- No new files/directories outside the above and `ARCHITECTURE.md`

---

## Phase 0: Pre-implementation Refactoring

**Rationale**: Widening `MergeOp` from 5 to 7 named values (research.md D2) is a mechanical, behavior-preserving rename if done before any dispatch-logic change — submitted as its own PR first, per constitution Phase 0 guidance, so the larger Phase 2.5 rewrite lands on a clean, already-renamed foundation. All existing tests MUST pass unchanged after this phase (whole-node dispatch behavior is untouched here — only names change).

- [X] T001 Rename `internal/core/ast.go`'s `MergeOp` constants: `MergeNone` → `MergeImmutable` (`"immutable"`), `MergeUnionFirstWriter` → `MergeFirstWriteWin` (`"firstWriteWin"`), `MergeValidatedOverwrite`'s string value `"validated-overwrite"` → `"validatedOverwrite"`; add (not yet wired into any dispatch) `MergeFillIfEmpty` (`"fillIfEmpty"`) and `MergeLastWriteWin` (`"lastWriteWin"`); `MergeUnion`/`MergeAppend` unchanged
- [X] T002 [P] Update `internal/core/merge.go`'s existing whole-node `switch op` cases in `Merge` to reference the renamed constants from T001, with identical behavior (no per-predicate logic yet)
- [X] T003 [P] Update `internal/core/ast_test.go`'s `MergeOp` roundtrip test (around line 127) to the renamed constants/string values from T001
- [X] T004 [P] Update `internal/app/schema/kernel/schema.go`: delete the 6 local `mergeXxx` aliases (lines 33-39); make `CorePredicateDefs` and `CoreTypeDefs` reference the new constants from T001 directly (values become genuinely distinct here; still inert until Phase 2.5 wires per-predicate dispatch)
- [X] T005 [P] Update `internal/app/schema/service/schema.go`'s `validMergeOps` map (line 41) to the new 7-value set from T001
- [X] T006 [P] Rename retired constants in `internal/app/lint/service/lint_test.go` and `internal/app/lint/service/rules_frontmatter_test.go` fixtures (`MergeNone`→`MergeImmutable`, `MergeUnionFirstWriter`→`MergeFirstWriteWin`) — fixture data only, no logic in `internal/app/lint` reads `.Merge`
- [X] T007 Run `go build ./... && go test ./...` and confirm zero behavior change (every existing test still green) before proceeding past this phase

---

## Phase 1: Setup

**Purpose**: Confirm baseline health before feature work begins — no new module, package, or dependency (research.md: no new dependency).

- [X] T008 Confirm `go build ./...` and `go test ./...` pass on branch `012-predicate-merge-policies` at Phase 0's completion (baseline for this feature)
- [X] T009 [P] Confirm `staticcheck ./...` runs clean on the baseline

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate, not implementation.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T010 Draft updated `ARCHITECTURE.md` Glossary entries for "Merge Behavior" (the 7-value canonical vocabulary, per-predicate not per-type), "Predicate Schema Node" (its `merge` field is now load-bearing), and "Source Node"/"Entity/Resource Node" (drop their parenthetical whole-node `MergeOp` — e.g. "CORE's fixed, always-recognized `MergeNone` kind" — since type no longer determines merge behavior); finalized copy lands in T036
- [X] T011 [P] Confirm `core.Index` (already declared in `internal/core/rules.go`) is reused directly in `core.Merge`'s new signature rather than introducing a new resolver type or interface (research.md D1) — no new domain type to check for duplication

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T012 Confirm this feature introduces no new command, flag, or `--json`/`--plain` schema change — `cmd/arc/graph/apply.go`'s Cobra wiring is untouched; record this confirmation (N/A for this feature)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T013 Confirm this feature introduces no new external integration or adapter — `core.Merge` remains pure (no `context.Context`, no I/O); record this confirmation (N/A for this feature)

### Phase 2d: Acceptance Test Design — E2E and Unit, All Red (Principle VI, VIII)

> All tests below MUST be written and MUST fail (red) before any Phase 2.5 implementation task begins.

- [X] T014 [P] [US1] Write E2E test(s) in `cmd/arc/graph/apply_test.go` for spec.md User Story 1's 4 acceptance scenarios: an `immutable` `ref` rejects a later differing contribution; a `lastWriteWin` `status` takes the newest applied value; a `union` `tags` accumulates across patches; three predicates on one node each resolve independently within a single `arc apply` — include any patch/node fixtures the scenarios need
- [X] T015 [P] [US1] Write unit tests in `internal/core/merge_test.go` for `immutable`/`union`/`append` dispatched **by declared MergeOp, not by observed value count** (research.md D5c) on `Attrs`, replacing the old whole-node-keyed tests (`TestMergeNoneNoOp`, parts of `TestMergeAppendUnionsEdgesAndAttrs`) with per-predicate equivalents
- [X] T016 [P] [US2] Write E2E test(s) in `cmd/arc/graph/apply_test.go` for User Story 2's 4 acceptance scenarios: `firstWriteWin` flags a genuine divergence; `lastWriteWin` never flags; `union` never flags; `fillIfEmpty` flags only after its first value is set, not on the first write itself
- [X] T017 [P] [US2] Write unit tests in `internal/core/merge_test.go` for `firstWriteWin`/`fillIfEmpty` conflict-marker behavior on `Attrs` and `Texts`, plus negative-case assertions that `union`/`append`/`lastWriteWin`/`immutable`/`validatedOverwrite` never produce a conflict marker even on genuine divergence (spec FR-012)
- [X] T018 [P] [US3] Write E2E test(s) in `cmd/arc/graph/apply_test.go` for User Story 3's 4 acceptance scenarios: replaying an already-applied patch is a byte-for-byte no-op; two patches touching independent predicates converge identically in either order; a `lastWriteWin` predicate's result matches whichever patch was applied last (order-sensitive by design, research.md D5a); a conflict marker is not re-wrapped on replay
- [X] T019 [P] [US3] Write unit tests in `internal/core/merge_test.go` for idempotency (`Merge(Merge(x, a, index, id), a, index, id) == Merge(x, a, index, id)`, every `MergeOp`) and commutativity on independent predicates (every `MergeOp` except `lastWriteWin`), plus a dedicated test asserting `lastWriteWin`'s documented order-sensitivity exception

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T020 Confirm this feature introduces no new configuration value or secret (N/A for this feature)

**Checkpoint**: All Phase 2 subsections complete, all Phase 2d tests written and red — Foundational implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The shared per-predicate dispatch engine every user story's tests (T014-T019) depend on. This is where the actual behavior change lands.

- [X] T021 Implement generic `mergeScalar[T comparable](existing, incoming T, zero T, op MergeOp) (merged T, diverges bool)` in `internal/core/merge.go`, covering the 3 dispatch classes from data-model.md's truth table: **freeze** (`immutable`, `validatedOverwrite`), **flagOnDiverge** (`firstWriteWin`, `fillIfEmpty` — identical code path per spec FR-006), **alwaysOverwrite** (`lastWriteWin`)
- [X] T022 Change `core.Merge`'s signature in `internal/core/merge.go` to `Merge(existing, incoming Node, index Index, sourceID string) (Node, []string, error)` per contracts/merge-behavior-contract.md; delete the old whole-node `mergeCore` and its `(fillEmpty, flagConflicts, unionText)` triple
- [X] T023 [P] Implement the per-key `Attrs` reconciliation loop in `internal/core/merge.go`: for each key in the union of `existing.Attrs`/`incoming.Attrs`, resolve `index.Predicates[key].Merge` (fallback `MergeUnion` with the same warning-and-continue precedent `apply.go` already uses for unrecognized types, research.md D6), then dispatch to `unionPredicates` (`union`/`append`) or `mergeScalar` (the other five) — dispatch chosen by declared op, not by how many values are present (research.md D5c)
- [X] T024 [P] Implement the per-key `Texts` reconciliation loop in `internal/core/merge.go`: same resolution as T023, dispatching to `mergeText`'s paragraph-append (`append`, and `union` as a documented fallback, research.md D5/D5b) or `mergeScalar` (the other five); delete the old hardcoded `k != "notes"` carve-out — `"notes"`'s own `firstWriteWin` declaration now produces the same behavior with no special-casing
- [X] T025 [P] Fold `Node.Published` into the generic dispatch in `internal/core/merge.go`: replace `mergePublished` with `mergeScalar[time.Time]` parameterized by `index.Predicates["published"].Merge` (research.md D3); delete `mergePublished`
- [X] T026 [P] Confirm `Edges`/`HRefs` keep unconditional `unionLinks` dedup unchanged (research.md D5d — no seeded link/edge predicate uses a scalar-natured op; add a fallback-to-union comment only if the code isn't already self-evident, per constitution Principle IV's no-inline-comments-except-non-obvious rule)
- [X] T027 Update `internal/app/graph/service/apply.go`: delete the `op := typeDef.Merge` / `MergeUnion`-fallback computation (current lines ~220-231); call `core.Merge(existing, node, index, patch.Document)` directly; keep the `typeDef, ok := index.Types[node.Type]` lookup and its existing warning/`RegisterType` call unchanged (still needed for unrecognized-kind detection, just no longer feeds `.Merge` anywhere)
- [X] T028 Retire `ErrUnknownMergeOp` as a `Merge`-level return path in `internal/core/merge.go`/`errors.go` (contracts/merge-behavior-contract.md: an individual predicate's invalid `MergeOp` can only originate from a schema document that already failed `internal/app/schema/service.decodePredicateDef` validation before `Merge` is ever reached)

**Checkpoint**: Foundation ready — run `go test ./internal/core/... ./internal/app/graph/... ./cmd/arc/graph/...`; all of T014-T019's tests should now be green

---

## Phase 3: User Story 1 - Per-predicate reconciliation (Priority: P1) 🎯 MVP

**Goal**: Every predicate on a merged node reconciles by its own declared rule, not by its node's type.

**Independent Test**: Apply two patches contributing `ref`/`status`/`tags` to the same `resource` node and confirm each field's outcome matches its own declared rule (spec.md Independent Test, User Story 1).

- [X] T029 [US1] Confirm T014 (E2E) and T015 (unit) are green after Phase 2.5; align any test expectations with real fixture output with minimal changes (Principle VIII)

**Checkpoint**: User Story 1 passes independently — MVP delivered.

---

## Phase 4: User Story 2 - Conflict flagging scoped correctly (Priority: P2)

**Goal**: The conflict marker fires only for `firstWriteWin`/`fillIfEmpty`-after-first-write, never for the other five behaviors.

**Independent Test**: Apply diverging contributions to one predicate of each rule and confirm a marker appears only where it should (spec.md Independent Test, User Story 2).

- [X] T030 [US2] Confirm T016 (E2E) and T017 (unit) are green after Phase 2.5; specifically re-verify `TestApplyFlagsConflict` in `internal/app/graph/service/apply_test.go` (currently keyed to whole-node `resource`/`union-first-writer` behavior) now asserts conflict scoping driven by the touched predicate's own rule

**Checkpoint**: User Stories 1 AND 2 both pass independently.

---

## Phase 5: User Story 3 - Replay and reorder safety (Priority: P3)

**Goal**: Every behavior except `lastWriteWin` is commutative and idempotent; `lastWriteWin` is deliberately application-order-sensitive.

**Independent Test**: Apply a patch twice, and apply two patches in both orders; confirm identical results except for the documented `lastWriteWin` exception (spec.md Independent Test, User Story 3).

- [X] T031 [US3] Confirm T018 (E2E) and T019 (unit) are green after Phase 2.5; confirm `vcs.IsTracked`'s existing whole-document replay short-circuit in `internal/app/graph/service/apply.go` is untouched, and that `rollback(store, createdPaths)` on a mid-merge error still leaves no partial writes with the new `Merge` signature

**Checkpoint**: All three user stories pass independently.

---

## Additional Polish

- [X] T032 [P] Finalize `ARCHITECTURE.md`'s Glossary entries drafted in T010 (Merge Behavior, Predicate Schema Node, Source Node, Entity/Resource Node)
- [X] T033 [P] Manually run `specs/012-predicate-merge-policies/quickstart.md`'s four scenarios end-to-end against a built `arc` binary
- [X] T034 Run `go vet ./...` and `staticcheck ./...` across the full repo and confirm clean

---

## Phase 6: Bugfix BUG-001 — Verbose Per-Predicate Reporting & `role: text` Append Default

**Purpose**: Addresses [specs/012-predicate-merge-policies/bugs/BUG-001.md](bugs/BUG-001.md), reported after live validation: `arc apply --verbose` stayed node-level after per-predicate dispatch shipped (FR-017), and `abstract`/`definition`/`notes`/`relevance`/`description` (all `role: text`) were seeded `firstWriteWin` instead of `append` (FR-018), which is what produced an unexpected conflict marker on a real graph's `LLM`/`Graph OS` entities.

- [X] T035 [P] ⚠️ Reopened — BUG-002 Widen `internal/core/merge.go`'s `Merge` to also return a per-predicate outcome trail (e.g. `[]PredicateOutcome{Name, Op, Outcome}` for every predicate present on either side, not only flagged ones) alongside the existing `conflicts []string` — additive, no existing caller's use of the first two return values changes (reopened — BUG-002: the `isListMerge` branch inside `mergeTexts`/`mergeAttrs` always reports `OutcomeAppended`/`OutcomeCreated`, never `OutcomeUnchanged`, even when the merged value is byte-identical to the existing one; see T043)
- [X] T036 [P] Update `internal/core/merge_test.go`: assert the new outcome trail's contents (name/op/outcome) for a representative case of each of the seven `MergeOp`s
- [X] T037 ⚠️ Reopened — BUG-002 Update `internal/app/graph/service/apply.go`'s node-processing loop to emit one additional `Reporter.Step` per present predicate (name, resolved op, outcome) when `--verbose` is set, sourced from T035's new return value, alongside the existing one-line-per-node summary (FR-017) (reopened — BUG-002: correctly wires whatever T035 returns, but T035's own `union`/`append` outcome is currently wrong in the no-op case; see T043)
- [X] T038 [P] Update `internal/app/graph/service/apply_test.go`/`cmd/arc/graph/apply_test.go`: add assertions that `--verbose` output contains a per-predicate line for a merged node exercising at least one predicate of each dispatch class (freeze/flagOnDiverge/alwaysOverwrite/list)
- [X] T039 [P] Update `internal/app/schema/kernel/schema.go`'s `CorePredicateDefs`: repoint `abstract`/`definition`/`notes`/`relevance`/`description` from `MergeFirstWriteWin` to `MergeAppend` (FR-018); update `internal/app/schema/kernel/schema_test.go`/`internal/app/schema/service/schema_test.go` assertions tied to the old values
- [X] T040 Update every test fixture/assertion elsewhere in the repo that currently depends on one of these five predicates flagging a conflict (notably `internal/core/merge_test.go`'s `abstract`-based `TestMergeFirstWriteWin*`/`TestMergeFirstWriteWinReplayDoesNotRewrapMarker` cases and `cmd/arc/graph/apply_test.go`'s `TestApply012US2ConflictFlaggingScopedToFirstWriteWin`/`TestApply012US3ReplayDoesNotRewrapConflictMarker`, which use `abstract` as their firstWriteWin exemplar) — repoint them to a genuinely `role: meta`, `firstWriteWin`-declared predicate (e.g. `category`) instead, since `abstract` no longer flags
- [X] T041 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T042 [P] Manually re-run quickstart.md Scenario B (or an equivalent ad hoc check) confirming `abstract` now appends instead of flagging, and confirming `--verbose` shows a per-predicate line

**Checkpoint**: TN10 (reopened above) can be re-closed once T037/T038 land and are confirmed passing.

---

## Phase 7: Bugfix BUG-002 — `union`/`append` Outcome Must Reflect the Actual Merge Result

**Purpose**: Addresses [specs/012-predicate-merge-policies/bugs/BUG-002.md](bugs/BUG-002.md), reported after applying `dmitry-2026-graph.md` then `dmitry-2026-article.md` to a fresh graph: `arc apply --verbose` reported `definition: append -> appended` for the `LLM` entity even though `definition`'s value never changed (the incoming paragraph was a byte-identical duplicate, correctly dropped by `mergeText`'s own near-duplicate detection). Root cause: `mergeTexts`/`mergeAttrs`'s `isListMerge` branch (`internal/core/merge.go`) derives its reported outcome from which merge behavior dispatched, not from whether the dispatch actually changed the value — the one dispatch class where `OutcomeUnchanged` was never reachable (FR-019).

- [X] T043 [P] Fix `internal/core/merge.go`'s `mergeTexts` and `mergeAttrs` `isListMerge` branches: after computing the merged value (`mergeText(ev, iv)` / `unionPredicates(ev, iv)`), compare it against the existing value before choosing the outcome — `OutcomeCreated` when the key was absent from `existing`, `OutcomeUnchanged` when the merged value equals the existing value, `OutcomeAppended` only when it genuinely differs (FR-019)
- [X] T044 [P] Add `internal/core/merge_test.go` case(s): a re-contribution whose paragraph is a full (and, separately, a Jaccard near-) duplicate of existing `append`-declared prose reports `OutcomeUnchanged`, not `OutcomeAppended`; likewise for a `union`-declared list predicate re-contributing an already-present value
- [X] T045 [P] Update `cmd/arc/graph/apply_test.go` (or `internal/app/graph/service/apply_test.go`): add a `--verbose` assertion that re-applying a patch whose contribution to an `append`/`union` predicate is fully duplicate reports `unchanged`, not `appended`, in the per-predicate report line (SC-007)
- [X] T046 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T047 [P] Manually re-verify against the exact files that triggered this report (a `dmitry-2026-graph.md`-then-`dmitry-2026-article.md`-shaped fixture) that `definition: append -> unchanged` is now reported for the `LLM` entity's fully-duplicate contribution

**Checkpoint**: BUG-002 fixed — T035/T037 (reopened above) re-closed once T043 lands and T044/T045 pass; `mergeTexts`/`mergeAttrs`'s `isListMerge` branch reports `unchanged` for a genuine no-op, matching every other dispatch class's own accuracy standard.

---

## Phase 8: Bugfix 018/BUG-001 — `Class`-Level `merge` Field Must Not Be Mandatory

**Purpose**: Addresses [specs/018-apply-schema-patch/bugs/BUG-001.md](../018-apply-schema-patch/bugs/BUG-001.md), reported when `arc apply schema -v ../arcnet-spec/schema/domain-article.md` rejected a well-formed, published extension's `Hypothesis` `Class` definition with "schema document Hypothesis has a missing or invalid merge". Root cause: this feature's own FR-015 retired the whole-node `merge` field from reconciliation (it is never consulted once per-predicate dispatch runs), but `internal/app/schema/service.decodeTypeDef` — used by `Resolve` and therefore by `arc apply`, `arc lint`, and `arc apply schema` alike — was never updated to stop treating that now-functionally-inert field as a mandatory, validated one (FR-020).

- [X] T048 [P] Update `decodeTypeDef` in `internal/app/schema/service/schema.go`: stop requiring a `Class` node's `merge` attribute to be present/valid — an absent or unrecognized value resolves to the zero-value `core.MergeOp` ("no whole-node merge declared") rather than returning `"merge"` as an invalid field (FR-020); `decodePredicateDef`'s `Property`-level `merge` check is unchanged (still mandatory)
- [X] T049 [P] Update `internal/app/schema/service/schema_test.go`: add a case asserting `resolveTypes`/`Resolve` succeed for a `Class` document with no `merge` field (and, separately, still fail for a `Property` document missing `merge`, confirming FR-020 narrows validation for `Class` documents only)
- [X] T050 [P] Add an E2E regression test (`cmd/arc/ctrl/apply_schema_test.go` or `internal/app/schema/service/apply_test.go`): `arc apply schema` against a patch whose `Class` section carries no `merge` field succeeds and creates the type definition (018 spec.md User Story 1, Acceptance Scenario 2; SC-008)
- [X] T051 Confirm `Seed()` (`internal/app/schema/service/schema.go`) and `RegisterType` still emit `merge: union` on built-in/auto-registered `Class` documents for shape continuity (data-model.md) — no behavior change forced there; add/confirm an existing test still asserts this
- [X] T052 Update [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Type Schema Node glossary entry to state `merge` is optional (validated only for continuity, never required) on a Type Schema Node, mandatory only on a Predicate Schema Node
- [X] T053 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T054 Manually re-verify against the exact command that triggered this report — `arc apply schema` (or `-v`) against a `Class`-only patch section with no `merge` field — confirm it now succeeds

**Checkpoint**: 018/BUG-001 fixed — a `Class` document's `merge` field is validated as optional (mandatory only for `Property`), unblocking `arc apply schema`'s own headline scenario (importing a published extension) for any CORE-conformant `Class` definition, regardless of whether it declares the now-vestigial field.

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). Retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the per-predicate dispatch model (Principle I) — depends on T032
- [X] TN02 Domain concepts (7-value `MergeOp` vocabulary, per-predicate Node Reconciliation) added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — depends on T010/T032
- [X] TN03 Command/flag surface confirmed unchanged (Principle IX) — depends on T012

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern introduced; no new ADR needed — `core.Index` reuse confirmed to introduce no layering violation (Principle I, research.md D1) — depends on T011
- [X] TN05 `core.Merge` remains pure (no I/O, no `internal/app`/Cobra import); domain logic still uses ports where applicable (Principle III) — depends on T022
- [X] TN06 Unit tests (T015, T017, T019) were written and confirmed failing (Phase 2d) before Phase 2.5's implementation (T021-T028) turned them green (Principle VI)
- [X] TN07 `internal/core/merge_test.go` and `internal/app/graph/service/apply_test.go` use `github.com/fogfish/it/v2` exclusively, no mixed assertion library (Principle VI, Mandatory Libraries & Tooling)
- [X] TN08 No Bash scripts used for unit-level validation; quickstart.md's Scenario D (`go test ./...`) is the CI-gating check, Scenarios A-C are documented manual/smoke validation only (Principle VI)
- [X] TN09 No new external integration or adapter introduced; no vendor SDK type leaks through a port (Principle VII) — depends on T013
- [X] TN10 ⚠️ Reopened (reopened — BUG-001) — resolved ~~N/A — no terminal output change (Principle X)~~ — `--verbose`'s existing per-node-only report is now known to be insufficient once reconciliation is per-predicate; depends on T037/T038 below adding the per-predicate report (FR-017)
- [X] TN11 N/A — no configuration change (Principle XI) — depends on T020
- [X] TN12 N/A — no command help text change (Principle XII)
- [X] TN13 E2E tests from Phase 2d (T014, T016, T018) turned GREEN via T029/T030/T031 and changed minimally during implementation (Principle VIII)
- [X] TN14 All 12 of spec.md's acceptance scenarios (4 per user story × 3 stories) have a passing, colocated E2E test in `cmd/arc/graph/apply_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: no command/flag/`--json`/`--plain` semantics change, so no major version bump is required (Principle XIV); node file *content* may differ for previously-mis-dispatched predicates going forward — this is tool-produced data, not a scriptable output contract, and every such difference is documented as intentional in research.md D5c (BUG-001: this now additionally covers `abstract`/`definition`/`notes`/`relevance`/`description`'s `firstWriteWin`→`append` reassignment, documented as intentional in data-model.md's BUG-001 bugfix note)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 0**: No dependencies — separate PR, run first
- **Phase 1 (Setup)**: Depends on Phase 0's completion (T007)
- **Phase 2 (Design Preconditions)**: Depends on Setup — BLOCKS all user stories; 2a-2e can proceed in parallel with each other; 2d's tasks (T014-T019) are all independent of each other ([P])
- **Phase 2.5 (Foundational)**: Depends on Phase 2 completion (all of T014-T019 written and red)
- **User Stories (Phase 3-5)**: All depend on Phase 2.5 — each is a confirmation checkpoint, not new implementation, so they can be verified in any order (or in parallel) once Phase 2.5 is done
- **Additional Polish**: Depends on Phases 3-5 complete
- **Phase N**: Final gate — depends on all preceding phases

### Parallel Opportunities

- T002-T006 (Phase 0) can run in parallel — different files, same rename source of truth (T001)
- T014-T019 (Phase 2d) can all run in parallel — different concerns, all against not-yet-changed code, so no file contention beyond independent additions to `merge_test.go`/`apply_test.go`/`cmd/arc/graph/apply_test.go`
- T023-T026 (Phase 2.5) touch different functions within `merge.go`; T021 (the generic primitive) should land first since T023-T025 call it — sequence T021 → T022 → {T023, T024, T025, T026} [P] → T027 → T028
- T029/T030/T031 (Phases 3-5) are independent confirmation checkpoints and can run in parallel

---

## Parallel Example: Phase 2d (all three stories at once)

```bash
Task: "Write E2E tests for US1 in cmd/arc/graph/apply_test.go"
Task: "Write unit tests for US1 in internal/core/merge_test.go"
Task: "Write E2E tests for US2 in cmd/arc/graph/apply_test.go"
Task: "Write unit tests for US2 in internal/core/merge_test.go"
Task: "Write E2E tests for US3 in cmd/arc/graph/apply_test.go"
Task: "Write unit tests for US3 in internal/core/merge_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 0 (separate PR) → Phase 1 (Setup)
2. Complete Phase 2 (Design Preconditions), including all of Phase 2d's red tests
3. Complete Phase 2.5 (Foundational) — this is the entire behavior change
4. Complete Phase 3 (US1 confirmation)
5. Complete the User-Story-1-relevant slice of Phase N
6. **STOP and VALIDATE**: run quickstart.md Scenario A

### Incremental Delivery

Because Phase 2.5 implements all three stories' behavior in one pass (they share one engine), "incremental delivery" here means incrementally *confirming* stories rather than incrementally *building* them: Phase 3 confirms US1, Phase 4 confirms US2, Phase 5 confirms US3 — all three checkpoints can realistically close in the same PR as Phase 2.5, since there is no meaningful shippable state between them.

---

## Notes

- [P] tasks = different files/concerns, no dependencies
- This feature's three "stories" are test angles on one shared engine, not independently shippable slices — Phase 2.5 is where nearly all code changes; Phases 3-5 are verification, not new implementation
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements
- Commit after each phase, not each task, given how tightly Phase 2.5's tasks are coupled (T021 is a prerequisite for T023-T025 to even compile)

**Bugfix**: 2026-07-08 — BUG-001 Updated from bugfix patch. Added Phase 6 (T035-T042); reopened TN10; annotated TN15.

**Bugfix**: 2026-07-12 — BUG-002 Updated from bugfix patch. Reopened T035/T037; added Phase 7 (T043-T047) to fix `mergeTexts`/`mergeAttrs`'s `isListMerge` branch reporting `appended` for a genuine no-op merge (FR-019).

**Bugfix**: 2026-07-19 — 018/BUG-001 Updated from bugfix patch. Added Phase 8 (T048-T054) to make `decodeTypeDef`'s `merge` check optional for `Class` documents (FR-020), unblocking `arc apply schema`'s import of a published extension's `Class` definitions that carry no `merge` field.
