---

description: "Task list for implementing CamelCase node class names"
---

# Tasks: CamelCase Node Class Names

**Input**: Design documents from `/specs/019-camelcase-node-types/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/arc/graph/apply_test.go` — colocated E2E tests for `arc apply`, reusing the existing `sut`/`sutCaptureStderr`/`chdir`/`TestMain` helpers already defined in that file (package `graph`) — do not redefine them
- `cmd/arc/ctrl/init_test.go` / `cmd/arc/lint/lint_test.go` — colocated E2E tests for `arc init`/`arc lint`, each package already defining its own `sut`/`chdir`/`TestMain` helpers — do not redefine them
- `internal/core/` — patch-parsing domain logic (no `cobra` import, Principle III)
- `internal/app/schema/kernel/` — built-in schema data (`CoreTypeDefs`/`CoreTypeBases`)
- `internal/app/lint/{kernel,service}/` — lint rule vocabulary and rule implementations
- `internal/app/graph/service/` — `arc apply`'s domain logic

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the baseline before this feature's rename/validation work begins. No new package, module dependency, or command is introduced (plan.md Technical Context).

- [ ] T001 Run `go build ./...` and `go test ./...` from the repository root to confirm a clean baseline before any change in this feature
- [ ] T002 [P] Run `staticcheck ./...` to confirm a clean baseline before any change in this feature

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [ ] T003 Update [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary "Type Schema Node" and "Text Predicate / Prose Field" entries to state the new CamelCase-first-letter naming invariant for `@type`/Class identifiers (spec FR-002/FR-004/FR-005/FR-008) and update their lowercase example casing (`an entity's definition` → `an Entity's definition`) to match
- [ ] T004 Verify no new domain type is needed for this feature (data-model.md confirms `core.Node`, `core.TypeDef`, `kernel.Violation` are all reused unchanged) before writing any new code

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [ ] T005 Confirm [contracts/cli-contract.md](contracts/cli-contract.md) against the current `arc apply`/`arc init`/`arc lint` command surface: no command, flag, or `--json`/`--plain` field is added/renamed — only new error text and one new `Rule` enum value (`"typeCase"`)

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [ ] T006 Confirm this feature introduces no new external system, port, or adapter (plan.md Constitution Check: Principle VII N/A) before proceeding

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> Tests below MUST compile and fail semantically (red phase) before Phase 3+ implementation begins.

- [ ] T007 [P] [US1] Write E2E tests in `cmd/arc/graph/apply_test.go` for spec.md User Story 1 Acceptance Scenarios 1-3: a patch whose H1 begins lowercase is rejected (non-zero exit, no node file written, no new commit), a patch whose H1 begins uppercase succeeds with the class stored using the heading's exact casing, and a multi-H1-section patch with one non-compliant heading rejects the whole document
- [ ] T008 [P] [US2] Write E2E test in `cmd/arc/ctrl/init_test.go` for spec.md User Story 2 Acceptance Scenarios 1-2: every file under `_schema/types/` seeded by `arc init` begins with an uppercase letter, and no two seeded class names differ only by casing
- [ ] T009 [P] [US3] Write E2E tests in `cmd/arc/lint/lint_test.go` for spec.md User Story 3 Acceptance Scenarios 1-3: a lowercase-named schema type definition produces a `typeCase` violation, a node whose `@type` begins lowercase produces a `typeCase` violation, and a graph where every class name is CamelCase reports no `typeCase` violation

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [ ] T010 Confirm this feature introduces no new configuration value, environment variable, or secret-handling path (spec.md Assumptions) before proceeding

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: The built-in content-type rename (FR-002) and its mechanical propagation across every literal comparison/lookup that depends on the old lowercase names (research.md D4/D5/D7). This is genuinely shared prerequisite work: leaving any one of these files un-renamed while another is renamed would break existing, already-passing behavior (e.g. a "Source" node would land in a `Sources/` folder instead of `sources/`, or stop matching `checkSourceCitekey`'s own-citekey rule) — so this phase lands as one atomic unit before any user story's own new behavior (the apply gate, the lint rule) is exercised end-to-end.

- [ ] T011 [P] Rename `CoreTypeDefs`/`CoreTypeBases` map keys in `internal/app/schema/kernel/schema.go`: `source`→`Source`, `entity`→`Entity`, `resource`→`Resource`, `timeline`→`Timeline` (`Node`/`Property`/`Class` and `CoreTypeBases`' `["Node"]` values are already correct and unchanged) (research.md D4)
- [ ] T012 [P] Update `internal/core/markdown.go`'s `textPredicateFor` switch cases: `"source"`→`"Source"`, `"entity"`→`"Entity"`, `"resource"`→`"Resource"`; leave the `"hypothesis"`/`"aporia"`/`"thought"` cases untouched — they are not built-in seeded types (research.md D7)
- [ ] T013 [P] Update `internal/app/graph/service/apply.go`: `coreKindFolders` map keys become `"Source"`/`"Entity"`/`"Resource"` (values `"sources"`/`"entities"`/`"resources"` unchanged), `node.Type == "timeline"` (line ~216) → `"Timeline"`, `node.Type == "source"` (line ~304) → `"Source"`, `sourcePath := nodeFolder("source")` (line ~179) → `nodeFolder("Source")` (research.md D5/D7)
- [ ] T014 [P] Update `internal/app/lint/service/rules_identity.go`: `node.Type != "source"` (line ~22) → `"Source"`, `node.Type != "entity"` (line ~37) → `"Entity"` (research.md D7)
- [ ] T015 [P] Update `internal/app/lint/service/rules_links.go`: `node.Type == "source" || node.Type == "timeline"` (line ~57) → `"Source"`/`"Timeline"`, `kind == "source"` (line ~65) → `"Source"` (research.md D7)
- [ ] T016 [P] Update `internal/app/lint/service/rules_history.go`: `node.Type != "source"` (line ~24) → `"Source"` (research.md D7)
- [ ] T017 [P] Update `cmd/arc/graph/apply.go`'s `pluralizeKind`: `kind == "entity"` (line ~35) → `"Entity"` (research.md D5)
- [ ] T018 [P] Update `internal/core/ast.go`'s `Node.Type` doc comment (line ~62), which currently names `"source"`, `"entity"`, `"resource"`, `"timeline"` as examples, to the new CamelCase names — documentation accuracy only, no runtime behavior (research.md D7)
- [ ] T019 Run `go build ./... && go test ./...`; fix every pre-existing test/fixture broken by T011-T018 (hand-constructed `core.Node{Type: "source", ...}`-style literals, explicit lowercase `"@type"` yaml-fence values in fixtures meant to represent already-valid patches, and any hardcoded-lowercase assertions) across the existing suite — the pre-existing test suite MUST be green again before Phase 3 begins (depends on T011-T018)

**Checkpoint**: Foundational rename complete, existing suite green — user story implementation can now proceed

---

## Phase 3: User Story 1 - `arc apply` rejects non-CamelCase class headings (Priority: P1) 🎯 MVP

**Goal**: `arc apply` preserves a patch's H1 heading casing verbatim when deriving a node's class, and refuses (whole-document, no partial write) any patch whose class-defining H1 heading or explicit `@type` does not begin with an uppercase letter.

**Independent Test**: Run `arc apply` against a fixture patch whose H1 is `# entity`; confirm non-zero exit, no node written, no commit. Run against a fixture patch whose H1 is `# Entity`; confirm success and the stored `@type` is `Entity` verbatim.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T007) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [ ] T020 [US1] Add `ErrTypeCasing = faults.Safe1[string]("class name %q must be CamelCase — start with an uppercase letter")` to `internal/core/errors.go` (research.md D3)
- [ ] T021 [US1] Add an unexported `isCamelCase(s string) bool` helper to `internal/core/markdown.go`: false for an empty string, else `unicode.IsUpper` on the first rune (research.md D1)
- [ ] T022 [US1] In `internal/core/markdown.go`'s `patchNodeIdentity` (lines ~174-209): delete `typ = strings.ToLower(typeHeading)`, use `typeHeading` verbatim as the default `typ`; gate `typeHeading` through `isCamelCase`, returning `ErrTypeCasing.With(typeHeading)` on failure; when an explicit `@type` is present, additionally gate it through `isCamelCase`, returning `ErrTypeCasing.With(explicit)` on failure — independent of whether the heading itself passed (FR-004/FR-005/FR-008, research.md D2) (depends on T020, T021)
- [ ] T023 [US1] Add unit tests in `internal/core/markdown_test.go`: a CamelCase H1 heading is preserved verbatim (no lowercasing) in the parsed node's `Type`; a lowercase H1 heading returns `ErrTypeCasing`; a CamelCase H1 with a lowercase explicit `@type` returns `ErrTypeCasing` naming the explicit value; a patch with two H1 sections where only the second is lowercase still fails the whole parse (depends on T022)
- [ ] T024 [US1] Confirm T007's E2E tests in `cmd/arc/graph/apply_test.go` now pass; additionally assert no new commit is created and the target node folder gains no file on rejection (FR-005) (depends on T022)

**Checkpoint**: At this point, User Story 1's E2E tests (T007) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Built-in schema and `arc init` use CamelCase class names (Priority: P2)

**Goal**: Every built-in class name `arc init` seeds begins with an uppercase letter, with no lowercase-first-letter duplicate.

**Independent Test**: Run `arc init` in an empty directory; confirm every file under `_schema/types/` begins with an uppercase letter.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T008) and MUST currently be failing (red) until Phase 2.5's T011 lands. No further production code change is needed beyond Phase 2.5 — this phase closes the loop with verification and a regression-guarding unit test.

- [ ] T025 [US2] Confirm T008's E2E test in `cmd/arc/ctrl/init_test.go` passes against Phase 2.5's T011 rename; assert every seeded `_schema/types/*.md` filename begins uppercase and no two seeded classes differ only by casing (FR-002/FR-003) (depends on T011)
- [ ] T026 [US2] Add a table-driven unit test in `internal/app/schema/service/schema_test.go` (`github.com/fogfish/it/v2`) asserting every key `Seed()` produces under `_schema/types/` begins with an uppercase letter, guarding against future regression (depends on T011)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - `arc lint` flags non-CamelCase class names (Priority: P3)

**Goal**: `arc lint` reports a `typeCase` violation for every schema type definition and every node's own `@type` reference that does not begin with an uppercase letter.

**Independent Test**: Run `arc lint` against a fixture graph containing a class named `entity` (or a node typed with a lowercase class); confirm the report includes a `typeCase` violation naming it. Run against a fixture where every class name is CamelCase; confirm no such violation.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red).

- [ ] T027 [US3] Add `RuleTypeCase Rule = "typeCase"` to `internal/app/lint/kernel/lint.go`'s `Rule` const block; add it to `TestRuleConstantsAreDistinct`'s list in `internal/app/lint/kernel/lint_test.go` (research.md D6)
- [ ] T028 [P] [US3] Implement `checkNodeTypeCase(node core.Node, path string) []kernel.Violation` in new file `internal/app/lint/service/rules_types_case.go`: one `kernel.RuleTypeCase` violation, `Line: 0`, when `node.Type` fails a `^[A-Z][a-zA-Z0-9]*$`-shaped regex (mirroring `camelCasePattern`'s idiom, inverted) — FR-007, research.md D6
- [ ] T029 [US3] Implement `checkSchemaTypeCase(index core.Index) []kernel.Violation` in `internal/app/lint/service/rules_types_case.go`: one graph-spanning `kernel.RuleTypeCase` violation per `index.Types` key failing the same check, `Path` set to `kernel.TypesDir+"/"+name+".md"`, iterated in sorted-key order for deterministic output — FR-006, research.md D6 (depends on T028, same file)
- [ ] T030 [US3] Wire `checkNodeTypeCase` into `internal/app/lint/service/lint.go`'s existing per-node loop (alongside `checkPredicateCase`, ~line 137); append `checkSchemaTypeCase(index)`'s output onto the existing `graphSpanning` slice (alongside `checkUniqueBasenames`, ~line 122) so it flows into `kernel.NewLintResult` (depends on T027, T028, T029)
- [ ] T031 [US3] Add unit tests in new `internal/app/lint/service/rules_types_case_test.go`, mirroring `rules_predicates_test.go`'s structure: a CamelCase node type produces no violation, a lowercase node type produces exactly one `RuleTypeCase` violation, a CamelCase schema type index produces no violation via `checkSchemaTypeCase`, a lowercase schema type key produces exactly one (depends on T028, T029)
- [ ] T032 [US3] Confirm T009's E2E tests in `cmd/arc/lint/lint_test.go` pass; additionally confirm a freshly-`arc init`'d graph (post Phase 2.5/T011) reports zero `typeCase` violations (depends on T030)

**Checkpoint**: All three user stories' E2E tests pass independently

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories.

- [ ] T033 [P] Update README.md if it documents `_schema/types/` filenames or example `@type` values with the old lowercase casing
- [ ] T034 [P] Sweep ARCHITECTURE.md for any other lowercase built-in type example missed by T003 (e.g. in the Checklist Rule glossary entry) and correct to CamelCase
- [ ] T035 Manually run all four [quickstart.md](quickstart.md) scenarios against a locally built `arc` binary to validate the feature end-to-end

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [ ] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [ ] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [ ] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [ ] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — not expected for this feature (no new pattern, per plan.md Constitution Check)
- [ ] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [ ] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [ ] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [ ] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration
- [ ] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — unaffected by this feature
- [ ] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A, no new configuration
- [ ] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — no command changed; verify no help text now describes stale lowercase casing
- [ ] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [ ] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [ ] TN15 Release/versioning impact assessed: this feature changes `arc apply`'s acceptance behavior and `arc init`/`arc lint`'s output content in a scriptable-breaking way without renaming a flag/field — contracts/cli-contract.md documents it; confirm the release notes call out the major-version-worthy behavior change (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion; T011-T018 are independent files and can run in parallel, T019 depends on all of them
- **User Stories (Phase 3+)**: All depend on Phase 2.5
  - User Story 2's implementation (T025-T026) is a verification/regression-guard pass over work Phase 2.5's T011 already delivered — it has no dependency on User Story 1's own new code (T020-T024)
  - User Story 3's implementation (T027-T032) depends only on Phase 2.5 (a correctly-cased `index.Types`/graph to validate against), not on User Story 1's apply-time gate
  - User Story 1 (T020-T024) is independent of User Story 2/3's own new code
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on User Story 2/3's own implementation; delivers the MVP
- **User Story 2 (P2)**: Can start after Phase 2.5 — its underlying rename already landed there; this phase is verification-only
- **User Story 3 (P3)**: Can start after Phase 2.5 — no dependency on User Story 1/2's own implementation

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- `internal/core` changes (US1) before the E2E test can turn green
- `internal/app/lint/kernel` (Rule constant) before `internal/app/lint/service` (check functions) before `lint.go` wiring (US3)
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- Phase 2.5's T011-T018 (eight distinct/independent files) can all run in parallel; T019 follows once all land
- Once Phase 2.5 completes, User Story 1 (T020-T024), User Story 2 (T025-T026), and User Story 3 (T027-T032) can proceed in parallel (if team capacity allows) — none depends on another's new code

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Once Phase 2 completes, launch independent rename tasks together:
Task: "Rename CoreTypeDefs/CoreTypeBases keys in internal/app/schema/kernel/schema.go"
Task: "Update textPredicateFor switch cases in internal/core/markdown.go"
Task: "Update coreKindFolders and node.Type comparisons in internal/app/graph/service/apply.go"
Task: "Update node.Type comparisons in internal/app/lint/service/rules_identity.go"
Task: "Update node.Type comparisons in internal/app/lint/service/rules_links.go"
Task: "Update node.Type comparison in internal/app/lint/service/rules_history.go"
Task: "Update pluralizeKind in cmd/arc/graph/apply.go"
Task: "Update the Node.Type doc comment in internal/core/ast.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure (CRITICAL — the built-in rename every story needs)
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenarios 2, 3)
7. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready (quickstart.md Scenario 1 already passes here)
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo (mostly already delivered by Foundational Infrastructure)
4. Add User Story 3 → Verify against Phase N → Deploy/Demo (quickstart.md Scenario 4)
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable once Phase 2.5 lands, even though Phase 2.5 itself is what delivers most of User Story 2's own spec requirement
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
