---

description: "Task list for 033-arc-lint-skip-rules implementation"
---

# Tasks: Lint Rule Skipping (`arc lint --skip`, `arc lint rules`)

**Input**: Design documents from `/specs/033-arc-lint-skip-rules/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Per Principles VI and VIII, all 9 acceptance scenarios (plus cross-cutting edge cases) map to E2E tests, written before implementation (red-green-refactor).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 — maps to spec.md user stories
- Exact file paths included in every task

## Path Conventions

Per [plan.md](./plan.md) Structure Decision:

- `cmd/arc/lint/` — the `arc lint` and `arc lint rules` Cobra commands and their colocated E2E tests
- `internal/app/lint/kernel/` — `RuleDefinition`/`RuleDefinitions` (no cobra, no fsys)
- `internal/app/lint/service/` — one new error constant only; `Lint` itself is untouched (research D1)

**Every new `.go` file MUST carry the license header from [CLAUDE.md](../../CLAUDE.md).**

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the working baseline. This feature adds **no new dependency**.

- [X] T001 Confirm the baseline is green: `go build ./... && go test ./... && go vet ./...`
- [X] T002 [P] Confirm no new module dependency is required — `github.com/fogfish/faults` is already imported by `internal/app/lint/service/errors.go`
- [X] T003 [P] Confirm no `go.mod` change is required ([plan.md](./plan.md) Technical Context)

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS from the Compliance Checklist. Every subsection is a design gate — the deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Confirm the "lint rule catalog" domain term is recorded in [data-model.md](./data-model.md) Glossary additions, staged for `ARCHITECTURE.md`. **`ARCHITECTURE.md` does not exist** (pre-existing gap, same handling as spec 032); do NOT create a stub file solely to host this term — see [plan.md](./plan.md) Complexity Tracking
- [X] T005 Verify no existing type in `internal/app/lint/kernel/lint.go` already expresses "rule identifier + description" before introducing `RuleDefinition`

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm the `--skip` flag and `arc lint rules` surface in [contracts/cli-contract.md](./contracts/cli-contract.md) match CLIG conventions exactly: long-form-only flag, `arc lint rules` as a `cobra.NoArgs` child of `arc lint`, before implementation begins

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T007 Confirm **not applicable**: this feature introduces no new external system integration and no new adapter — `arc lint rules` reads no file, and `--skip` reads no state beyond what `arc lint` already reads through `internal/adapter/fsys`/`internal/adapter/git`

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> **Ordering note**: T009's tests are authored against `NewLintRulesCmd`'s signature, scaffolded in Phase 2.5 (T017). Author that stub first so the tests **compile**, then write the tests here so they **fail semantically** — exactly what the constitution's red phase requires. No test may be skipped or marked pending, and no "red phase" comment may appear.

- [X] T008 [P] [US1] Write E2E tests for User Story 1 in `cmd/arc/lint/lint_test.go`: `TestLintSkipSuppressesNamedRuleOnly`, `TestLintSkipAllViolationsInSkippedRuleSucceeds`, `TestLintSkipMultipleRulesCommaSeparated`, `TestLintSkipEveryRuleStillEnumeratesNodes`, plus edge cases `TestLintSkipIgnoresWhitespaceAndEmptySegments`, `TestLintSkipGraphSpanningRuleSuppressesBothFiles`, `TestLintSkipJSONOutputOmitsSkippedRuleViolations` — using the existing `sut()` helper and `cmd.Flags().Set("skip", ...)` per [quickstart.md](./quickstart.md) (red phase)
- [X] T009 [P] [US2] Write E2E tests for User Story 2 in new `cmd/arc/lint/rules_test.go`: `TestLintRulesListsEveryRuleWithDescription`, `TestLintRulesWorksOutsideGraph`, `TestLintRulesNameAcceptedBySkip`, `TestLintRulesDeterministicOutput`, `TestLintRulesJSONOutput` (red phase)
- [X] T010 [P] [US3] Write E2E tests for User Story 3 in `cmd/arc/lint/lint_test.go`: `TestLintSkipUnknownRuleRefuses`, `TestLintSkipMixedValidAndInvalidRefuses`, `TestLintSkipUnknownRuleRefusesBeforeGraphCheck` (red phase)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T011 Confirm **not applicable**: no new configuration value, no new environment variable, no secret of any kind is introduced

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin.

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Shared plumbing all three stories depend on — the rule catalog, the `--skip` parser, the error constant, and the compile-enabling `arc lint rules` stub.

- [X] T012 [P] Write failing unit test `TestRuleDefinitionsCoverEveryRule` in `internal/app/lint/kernel/lint_test.go`: asserts `len(RuleDefinitions)` equals the number of declared `Rule` constants and every `Description` is non-empty (red phase)
- [X] T013 Implement `RuleDefinition` type and the 18-entry `RuleDefinitions` catalog (per [data-model.md](./data-model.md)'s table, same order as the `Rule` const block) in `internal/app/lint/kernel/lint.go`, turning T012 green (depends on T012)
- [X] T014 [P] Write failing table-driven unit test `TestParseSkip` in `cmd/arc/lint/lint_test.go` covering every row of [data-model.md](./data-model.md)'s parsing table: empty value, comma list, whitespace/empty segments, duplicate names, unknown names (red phase)
- [X] T015 Implement `parseSkip(csv string) (skip map[kernel.Rule]bool, unknown []string)` in `cmd/arc/lint/lint.go`, turning T014 green (depends on T013, T014)
- [X] T016 Add `ErrUnknownSkipRule = faults.Safe1[string]("unknown rule(s) in --skip: %s — run `arc lint rules` to see valid rule names")` to `internal/app/lint/service/errors.go`, following the existing `ErrInvalidAttrFlag`/`ErrInvalidDepth` precedent ([research D4](./research.md))
- [X] T017 [P] Scaffold `NewLintRulesCmd()` in new `cmd/arc/lint/rules.go` with `RunE` returning a not-implemented error, so T009's tests compile

**Checkpoint**: Phase 2d tests compile and fail semantically. Foundation ready.

---

## Phase 3: User Story 1 - Suppress a rule that doesn't apply to this graph (Priority: P1) 🎯 MVP

**Goal**: `arc lint --skip <rules>` removes the named rules' violations from the report and recomputes the pass/fail outcome, leaving every other rule's violations unchanged.

**Independent Test**: Run `arc lint --skip <ruleA>` against a graph with known violations of `ruleA` and `ruleB`; confirm `ruleB`'s violations are unchanged and `ruleA`'s are absent, including from the pass/fail outcome.

### Implementation for User Story 1

> E2E tests T008 were written in Phase 2d and MUST currently be failing.

- [X] T018 [US1] Add the `--skip` string flag to `NewLintCmd` in `cmd/arc/lint/lint.go`: `cmd.Flags().StringVar(&skip, "skip", "", "Comma-separated rule names to exclude from the report (see `arc lint rules`)")`
- [X] T019 [US1] In `RunE`, call `parseSkip(skip)` **before** `fsys.Local{}.Mount`/`appschema.Resolve`; when `unknown` is non-empty, return `service.ErrUnknownSkipRule.With(errNoCause, strings.Join(unknown, ", "))` without performing any lint check (depends on T015, T016, T018)
- [X] T020 [US1] Implement violation filtering and reconstruction in `cmd/arc/lint/lint.go`: split `result.Violations` into the graph-spanning subset (reusing the existing `graphSpanningViolations`) and node-owned violations, filter both by the skip set together with each `NodeStatus.Violations`, and rebuild the result via `kernel.NewLintResultWithForeign(result.Root, filteredNodes, result.Foreign, filteredGraphSpanning...)` before it reaches the printer (depends on T013, T019)
- [X] T021 [US1] Update `Long` and `Example` on `arc lint` in `cmd/arc/lint/lint.go` to document `--skip` (Principle XII)
- [X] T022 [US1] Confirm T008's E2E tests now pass

**Checkpoint**: US1 E2E tests pass. `--skip` is fully functional as an MVP.

---

## Phase 4: User Story 2 - Discover the rules by name and meaning (Priority: P2)

**Goal**: `arc lint rules` lists every rule `arc lint` implements, each with its exact `--skip`-compatible name and a human-readable description.

**Independent Test**: Run `arc lint rules` in a directory with no graph and confirm every rule in `kernel.RuleDefinitions` is listed with a name and description; run twice and confirm identical output.

### Implementation for User Story 2

> E2E tests T009 were written in Phase 2d and MUST currently be failing beyond T017's not-implemented stub.

- [X] T023 [US2] Implement `humanRulesPrinter` and `var rulesRenderer = bios.Registry[[]kernel.RuleDefinition]{Human: humanRulesPrinter{}}` in `cmd/arc/lint/rules.go` (depends on T013)
- [X] T024 [US2] Complete `NewLintRulesCmd()`'s `RunE` in `cmd/arc/lint/rules.go` to resolve `rulesRenderer` via `bios.ResolveMode()` and render `kernel.RuleDefinitions`, replacing T017's stub (depends on T017, T023)
- [X] T025 [US2] Nest `arc lint rules` under `arc lint` in `cmd/arc/root.go`: replace the flat `cmd.AddCommand(lint.NewLintCmd())` with `lintCmd := lint.NewLintCmd(); lintCmd.AddCommand(lint.NewLintRulesCmd()); cmd.AddCommand(lintCmd)`, mirroring the existing `apply`/`apply schema-patch` nesting (depends on T024)
- [X] T026 [US2] Populate `Short`, `Long`, and `Example` on `arc lint rules` in `cmd/arc/lint/rules.go` (Principle XII)
- [X] T027 [US2] Confirm T009's E2E tests now pass

**Checkpoint**: US1 and US2 both pass their E2E tests independently.

---

## Phase 5: User Story 3 - Get an immediate, clear error on a mistyped rule name (Priority: P3)

**Goal**: An unrecognized `--skip` value refuses immediately, naming every bad value, before any lint check runs — even against a directory that is not a graph.

**Independent Test**: Run `arc lint --skip bogusRule` against both an initialized graph and a plain empty directory; confirm both refuse and both name `"bogusRule"`.

### Implementation for User Story 3

> User Story 3's core behavior was already delivered by T019: parsing and validating `--skip` is one shared code path (research D1/D7) that has to exist before any filtering can happen, so US1's implementation already produces the refusal. This phase closes the loop with its own scenario coverage and fixes ordering if the "before any graph resolution" requirement (FR-008) does not already hold.

> E2E tests T010 were written in Phase 2d and MUST currently be failing until T019 lands.

- [X] T028 [US3] Confirm T010's E2E tests pass against the T019 implementation. If `TestLintSkipUnknownRuleRefusesBeforeGraphCheck` fails because validation runs after `fsys.Local{}.Mount`, reorder `RunE` in `cmd/arc/lint/lint.go` so `parseSkip`'s validation strictly precedes graph resolution (depends on T019)

**Checkpoint**: All three stories pass their E2E tests independently.

---

## Additional Polish

- [X] T029 [P] Update `README.md`'s command reference to mention `arc lint --skip` and `arc lint rules`
- [X] T030 [P] Run `go vet ./...` and confirm clean
- [X] T031 Run the full [quickstart.md](./quickstart.md) validation command set and confirm every Definition-of-Done box

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). Retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — **blocked by the file's non-existence**; the rule catalog is documented in [research.md](./research.md)/[data-model.md](./data-model.md) instead, gap recorded in [plan.md](./plan.md) Complexity Tracking
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — staged in [data-model.md](./data-model.md) pending that file's creation (see T004)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — assess whether the `cmd/`-level filtering deviation or the `lint rules` Cobra-nesting pattern warrants an ADR, or is adequately covered by [research.md](./research.md) D1/D2/D5
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — confirm the noted, deliberate exception (plan.md Constitution Check) stays narrow: `cmd/arc/lint` contributes no new counting rule, only which violations are visible, reusing `kernel.NewLintResultWithForeign` as-is
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no new integration (T007)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose` (Principle X) — `arc lint rules`' human printer reuses `bios.SCHEMA`/`bios.Registry` exactly as `arc lint` does; no new styling introduced
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A (T011)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — T021, T026
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — 9 acceptance scenarios across 3 stories, each with a named test; cross-cutting edge cases additionally covered per [quickstart.md](./quickstart.md)
- [X] TN15 Release/versioning impact assessed (Principle XIV) — expected **minor**: a new flag and a new subcommand are additive; `arc lint --json`'s existing schema shape is unchanged (values omitted, not restructured); `arc lint rules --json` is a new v1 schema

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies
- **Phase 2 (Design Preconditions)**: Depends on Setup — BLOCKS all user stories
- **Phase 2.5 (Foundational)**: **Interleaved with Phase 2d** — T017 scaffolds the signature T009 compiles against; T012/T013 and T014/T015/T016 can proceed independently of T017
- **Phase 3+ (User Stories)**: All depend on Phase 2 and 2.5
- **Polish**: Depends on all three stories
- **Phase N**: Final gate

### User Story Dependencies

- **US1 (P1)**: Depends only on Phase 2/2.5. Delivers the MVP alone
- **US2 (P2)**: Depends on Phase 2/2.5 only — independent of US1's flag/filtering work, since it reads `kernel.RuleDefinitions` directly and touches a different file (`rules.go`). The two stories share only T013/T025 (the catalog, and `root.go`'s one-line nesting edit)
- **US3 (P3)**: Depends on T019 (US1) — its behavior is inherited from US1's validation step, not separately built (research D1/D7)

### Within Each User Story

- T013 (the catalog) before anything that reads it (T015, T020, T023)
- T018 before T019 before T020 (flag → validation → filtering, in that order, in the same file)
- Story complete before moving to the next priority

### Parallel Opportunities

- T002, T003 in Phase 1
- T008, T009, T010 — all three test-authoring tasks target distinct files/scenarios and can be written in parallel once T017 exists
- T012/T013 and T014/T015/T016 — two independent foundational threads (catalog vs. parser+error)
- T029, T030 in Polish

---

## Parallel Example: Foundational Phase

```bash
# Two independent threads, neither blocking the other:
Task: "Write TestRuleDefinitionsCoverEveryRule and implement RuleDefinitions in internal/app/lint/kernel/lint.go"
Task: "Write TestParseSkip and implement parseSkip in cmd/arc/lint/lint.go, plus ErrUnknownSkipRule in internal/app/lint/service/errors.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 — Setup
2. Phase 2 — Design Preconditions (CRITICAL, blocks everything)
3. Phase 2.5 — Foundational (catalog, parser, error constant, stub)
4. Phase 3 — User Story 1
5. Phase N — Constitution verification
6. **STOP and VALIDATE**: `arc lint --skip <rule>` suppresses exactly that rule's violations and nothing else

### Incremental Delivery

1. Setup + Design Preconditions + Foundational → foundation ready
2. Add US1 → verify → **MVP ships** (`--skip` works)
3. Add US2 → verify → `arc lint rules` ships
4. Add US3 → verify → refusal behavior explicitly proven (already present, now tested)

### Risk Notes

- **T020 is the feature's correctness core.** Getting the graph-spanning/node-owned split and the `NewLintResultWithForeign` rebuild right is what makes filtering agree with `arc lint`'s own counting rule; a bug here would silently miscount `Passing`/`Failing` rather than crash
- **T028 may require no code change at all** — if T019 already validates before resolving the graph, T028 is pure verification. Do not add a second validation path defensively; one is correct by design (research D7)
