---

description: "Task list for Lint Conformance Gaps"
---

# Tasks: Lint Conformance Gaps

**Input**: Design documents from `/specs/024-lint-conformance-gaps/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Unit and E2E acceptance tests are NOT optional per constitution Principles VI and VIII — every spec.md acceptance scenario maps 1:1 to an E2E test, written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story (US1 = P1, US2 = P2, US3 = P3, per spec.md) to enable independent implementation and testing of each.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions

This feature adds no new command and no new domain package (plan.md Structure Decision). It
extends `internal/app/lint/{kernel,service}`, `internal/app/graph/service`, and adds one new file,
`internal/core/identity.go`, holding the one primitive both consume.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the existing structure this feature extends is ready; no new package is created.

- [X] T001 Confirm target files exist and are writable: `internal/core/`, `internal/app/lint/kernel/lint.go`, `internal/app/lint/service/`, `internal/app/graph/service/apply.go` — no new directory needed (plan.md Project Structure)
- [X] T002 Confirm no new Go module dependency is required — this feature adds no external dependency (plan.md Technical Context); `go.mod` is unchanged
- [X] T003 [P] Confirm `staticcheck` runs clean on the current `internal/app/lint`, `internal/app/graph/service`, and `internal/core` packages before any change (baseline)

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins). Every subsection below is a design gate — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add "Sowa category combination" (a fixed 4-word tuple, one of exactly twelve legal rows) and "node identity charset" (the forbidden-character invariant every `@id` must satisfy) to [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary, referencing data-model.md §1 and §3
- [X] T005 Verify no existing `internal/<domain>` type already expresses the forbidden-identity-character scan before adding `internal/core/identity.go` — confirmed absent in research.md D6; record that confirmation here before T014 creates the file

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc lint` and `arc apply`'s flag surfaces are unchanged by this feature (no new flag, no changed flag semantics) — only the set of possible `Violation.Rule` values (`arc lint --json`) and error messages (`arc apply`) grows
- [X] T007 [P] Review [contracts/sowa-category-contract.md](contracts/sowa-category-contract.md) and [contracts/identity-charset-contract.md](contracts/identity-charset-contract.md) (already drafted during `/speckit-plan`) against the final task breakdown below; amend either contract if a task reveals a gap

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 Confirm this feature requires no new external integration and no new adapter — all new logic is pure domain code over already-mounted `fsys.Store`/`core.Index` values (plan.md Constitution Check, Principle VII: PASS, N/A)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T009 [P] [US1] Write E2E test(s) in `cmd/arc/graph/apply_test.go` for spec.md US1 Acceptance Scenarios 1-4 (unsafe identity rejected naming character+position; safe identity applies normally; one bad node rejects the whole patch; a schema-node-implied identity — via `node.Type`/predicate name — is rejected too), using the existing `sut()` helper; tests MUST compile and fail semantically (red phase)
- [X] T010 [P] [US2] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md US2 Acceptance Scenarios 1-6 (content-node charset violation named with position; safe identity clean; `Source` citekey clean; schema-node (`_schema/Class`/`_schema/Property`) charset violation; multiple offending characters all named, not just the first; `arc lint` modifies no file), via `sut()` (red phase)
- [X] T011 [P] [US3] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md US3 Acceptance Scenarios 1-4 (legal Sowa combination passes; structurally-valid-but-illegal combination rejected with a suggested legal row; wrong-length category keeps its existing distinct message; `arc lint` modifies no file), via `sut()` (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T012 Confirm this feature introduces no new configuration value, flag, or secret — N/A, no review action required beyond this confirmation

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The forbidden-identity-character primitive is shared by US1 (`arc apply` rejection) and US2 (`arc lint` detection) — research.md D6. It must exist, tested, before either story's implementation tasks.

- [X] T013 [P] Write unit tests in `internal/core/identity_test.go` — one case per forbidden character (`/ \ : * ? " < > | .`) individually, plus one legal control identity, asserting correct 1-indexed rune positions (contract C2.1, data-model.md §3); MUST fail to compile/fail semantically before T014
- [X] T014 [P] Implement the forbidden-character set and rune-scan primitive in `internal/core/identity.go`, returning every offending `(char, position)` pair for a given identity string (data-model.md §3, research.md D5/D6); turns T013 green
- [X] T015 [P] Add `ErrIdentityCharset = faults.Safe2[string, string](...)` to `internal/app/graph/service/errors.go`, alongside `ErrNodeWrite`, formatting its detail fragment from T014's pairs (data-model.md §5)
- [X] T016 [P] Add the `RuleIdentityCharset` constant to the `Rule` block in `internal/app/lint/kernel/lint.go` (data-model.md §2)

**Checkpoint**: Foundation ready — US1 and US2 implementation can now proceed in parallel; US3 does not depend on this phase

---

## Phase 3: User Story 1 - Block unsafe identities before they enter the graph (Priority: P1) 🎯 MVP

**Goal**: `arc apply` rejects the entire operation, before any file is written, when a patch would introduce or modify a node — content or schema-implied — whose identity contains a forbidden character.

**Independent Test**: Run `arc apply` on a patch containing one node with a forbidden character in its identity; confirm the command exits non-zero naming the character and position, and that the graph directory is byte-for-byte unchanged (quickstart.md US1).

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red). Implementation below turns them green with minimal test changes.

- [X] T017 [US1] Implement a pre-scan guard over `patch.Nodes` in `internal/app/graph/service/apply.go`, called immediately after `readPatch` succeeds (`apply.go:229`) and before the main per-node loop writes anything, checking every `node.ID` via T014's primitive and returning `ErrIdentityCharset` on the first offending node (research.md D7, mirrors `guardNoOldFormatNodes`'s scan-before-write shape) — turns T009's single-bad-node and whole-patch-rejected scenarios green
- [X] T018 [US1] Add in-loop checks in `Apply`'s per-node loop, before `schema.RegisterType(store, node.Type)` (`apply.go:275`) and before `schema.RegisterPredicate(store, obs.name, ...)` (`apply.go:317`), rejecting via the loop's existing `reporter.Error` → `rollback(store, createdPaths)` → return shape when the type name or predicate name contains a forbidden character (research.md D7) — turns T009's schema-node-implied-identity scenario green
- [X] T019 [US1] Confirm both T017 and T018's rejection paths leave the graph byte-for-byte unchanged (spec FR-003) — verify `rollback(store, createdPaths)` covers every path created before the failure in both cases; strengthen T009's assertions if a gap is found

**Checkpoint**: User Story 1's E2E tests (T009) pass; `arc apply` fully rejects unsafe identities independently of `arc lint`

---

## Phase 4: User Story 2 - Surface existing unsafe identities already in the graph (Priority: P2)

**Goal**: `arc lint` reports every node — content or schema — whose identity contains a forbidden character, naming each character and its 1-indexed position, without modifying any file.

**Independent Test**: Run `arc lint` against a graph containing a node with a known forbidden character in its identity; confirm the report names the character and position, and that no file in the graph changed (quickstart.md US2).

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T010) and MUST currently be failing (red).

- [X] T020 [US2] Implement `checkIdentityCharset(node core.Node, path string, raw []byte) []kernel.Violation` in `internal/app/lint/service/rules_identity.go`, using T014's primitive and `locateFrontMatterField(raw, `\`"@id"\``)` for the line (research.md D4, data-model.md §2) — turns T010's content-node scenarios green
- [X] T021 [US2] Implement `checkSchemaIdentityCharset(index core.Index) []kernel.Violation` in `internal/app/lint/service/rules_types_case.go`, iterating sorted `index.Types` then sorted `index.Predicates` keys and synthesizing `Path`/`Line: 0` exactly as `checkSchemaTypeCase` does (research.md D3, data-model.md §2) — turns T010's schema-node scenario green
- [X] T022 [US2] Wire `checkIdentityCharset` into `Lint`'s per-node loop and `checkSchemaIdentityCharset` into the `graphSpanning` slice in `internal/app/lint/service/lint.go`, alongside the existing `checkEntityCategory`/`checkSchemaTypeCase` calls
- [X] T023 [P] [US2] Add unit tests in `internal/app/lint/service/rules_identity_test.go` for `checkIdentityCharset`/`checkSchemaIdentityCharset`, covering the multiple-offending-character message shape from data-model.md §4

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Reject Sowa categories that mix incompatible words (Priority: P3)

**Goal**: `arc lint` validates an `Entity`'s four-word category as a whole combination against the twelve legal rows, suggesting the closest legal row on rejection.

**Independent Test**: Run `arc lint` against an `Entity` whose category mixes words from two different legal combinations; confirm the report names the rejected combination and a legal one, without modifying any file (quickstart.md US3).

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T011) and MUST currently be failing (red).

- [X] T024 [P] [US3] Write a unit test in `internal/app/lint/kernel/lint_test.go` exhaustively covering all 144 positional combinations, asserting exactly 12 pass (contract C1.1); MUST fail before T025
- [X] T025 [US3] Replace `sowaPosition1`/`sowaPosition2`/`sowaPosition3`/`sowaLeaf` with the `sowaCategories [12][4]string` table and rewrite `ValidSowaCategory`'s 4-word branch as an exact-row lookup in `internal/app/lint/kernel/lint.go` (data-model.md §1) — turns T024 and T011's legal-combination scenario green
- [X] T026 [US3] Implement the closest-legal-combination suggestion (longest matching leading-word prefix, first table-order match wins ties) inside `ValidSowaCategory`'s rejection path (research.md D2, contract C1.2) — turns T011's illegal-combination scenario green
- [X] T027 [US3] Confirm the wrong-length category message (`len(words) != 4`) is unchanged by T025/T026 (contract C1.3) — add/keep a regression unit test asserting the exact existing message text

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories

- [X] T028 [P] Cross-check the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary entries from T004 read consistently with the final `RuleIdentityCharset`/Sowa-table naming used in code
- [X] T029 [P] Run [quickstart.md](quickstart.md)'s three sections end-to-end against a locally built `arc` binary
- [X] T030 Run `go test ./... -cover`, confirming no regression in `internal/app/lint/kernel/lint_test.go`, `internal/app/lint/service/rules_identity_test.go`, `internal/core/merge_test.go`, `internal/core/markdown_test.go`, and `cmd/arc/graph/apply_test.go`'s existing (pre-feature) cases

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the two new domain concepts from T004 (Principle I)
- [X] TN02 Domain concepts (Sowa category combination, node identity charset) added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 `arc lint`/`arc apply`'s command/flag surface is confirmed unchanged, matching T006 exactly (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced — no new ADR required (Principle I); confirm and record that decision here
- [X] TN05 New domain logic lives entirely in `internal/core`, `internal/app/lint/{kernel,service}`, `internal/app/graph/service` — no `cmd/`-package type or `cobra` import reaches any of them (Principle III)
- [X] TN06 Unit tests (T013, T023, T024) were written first, compiled, and failed semantically before their corresponding implementation tasks (Principle VI)
- [X] TN07 All new/changed unit and E2E tests use `github.com/fogfish/it/v2` exclusively (Principle VI, Mandatory Libraries & Tooling)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation — only `go test` (Principle VI)
- [X] TN09 No new external integration or adapter was introduced (Principle VII) — matches T008
- [X] TN10 N/A — this feature changes no terminal output styling, color, or TTY behavior (Principle X)
- [X] TN11 N/A — this feature introduces no new configuration, flag, or secret (Principle XI) — matches T012
- [X] TN12 N/A — no new/changed command means no `Short`/`Long`/`Example` help text to populate (Principle XII); confirm existing `arc lint`/`arc apply` help text does not now contradict the new rejection/violation behavior
- [X] TN13 E2E tests from Phase 2d (T009, T010, T011) turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 Every spec.md acceptance scenario for this feature (US1 1-4, US2 1-6, US3 1-4) has a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: this feature makes `arc apply` reject patches it previously accepted, and `arc lint` report violations it previously missed — both are behavior tightenings, not a changed command name or flag semantic; assess against Principle XIV whether a deprecation-warning release is warranted before the tightening ships, or whether pre-1.0 status makes that unnecessary (record the decision here). **Decision**: no deprecation-warning release — `arc` is pre-1.0/experimental (ARCHITECTURE.md Compatibility Policy), which already accepts breaking changes to conformance behavior outright, with no migration path or compatibility shim, consistent with how `specs/019`/`specs/023`'s own tightenings shipped directly. A pre-existing test fixture (`internal/app/lint/service/testdata/v011-graph/Entity/Merge Vocabulary.md`) and a pre-existing E2E fixture identity (`cmd/arc/graph/revert_test.go`'s `"TLS 1.3"`) both needed a one-line update to stop tripping the newly-enforced rules — concrete confirmation the tightening is real, not merely theoretical.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a-2e can proceed in parallel with each other
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion; blocks US1 and US2 only (US3 does not depend on it)
- **User Stories (Phase 3+)**: US1/US2 depend on Phase 2 + Phase 2.5; US3 depends on Phase 2 only
  - User stories can proceed in parallel (if staffed) or sequentially in priority order (P1 → P2 → P3)
- **Additional Polish**: Depends on all three user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 (T013-T016) — no dependency on US2/US3
- **User Story 2 (P2)**: Depends on Phase 2.5 (T013-T016) — no dependency on US1/US3
- **User Story 3 (P3)**: Depends only on Phase 2 — no dependency on US1/US2 or Phase 2.5

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- US1: pre-scan guard (T017) before in-loop checks (T018) before the unchanged-graph confirmation (T019) — all touch the same file in sequence, not parallelizable
- US2: content-node check (T020) and schema-node check (T021) touch different files and can run in parallel; both before wiring (T022); unit tests (T023) after wiring
- US3: unit test (T024) before the table replacement (T025) before the suggestion logic (T026) before the regression confirmation (T027) — sequential, same file

### Parallel Opportunities

- T009, T010, T011 (E2E test authoring for the three stories) can run in parallel — different files, no shared dependency
- T013-T016 (Foundational) can run in parallel — different files
- Once Phase 2.5 completes, US1 (Phase 3) and US2 (Phase 4) can proceed in parallel; US3 (Phase 5) can start as soon as Phase 2 completes, independent of both
- T020 and T021 within US2 can run in parallel (different files)

---

## Parallel Example: Foundational + User Story 3

```bash
# Phase 2.5, all four in parallel (different files):
Task: "Write unit tests in internal/core/identity_test.go"
Task: "Implement internal/core/identity.go"
Task: "Add ErrIdentityCharset to internal/app/graph/service/errors.go"
Task: "Add RuleIdentityCharset to internal/app/lint/kernel/lint.go"

# User Story 3 can start the moment Phase 2 (not 2.5) is done, in parallel with the above:
Task: "Write exhaustive 144-combination unit test in internal/app/lint/kernel/lint_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Run quickstart.md's US1 section independently
7. Deploy/demo if ready — `arc apply` alone already closes the corruption-prevention gap

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP — prevents new corruption)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo (surfaces existing/legacy corruption)
4. Add User Story 3 → Verify against Phase N → Deploy/Demo (closes the unrelated Sowa gap)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Design Preconditions + Foundational Infrastructure together
2. Once complete:
   - Developer A: User Story 1 (`arc apply`)
   - Developer B: User Story 2 (`arc lint` charset)
   - Developer C: User Story 3 (`arc lint` Sowa) — can start as soon as Phase 2 alone is done, doesn't need to wait for Phase 2.5
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
