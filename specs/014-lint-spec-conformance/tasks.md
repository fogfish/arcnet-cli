# Tasks: Full ARCNET-CORE §16 Conformance Checks for `arc lint`

**Input**: Design documents from `/specs/014-lint-spec-conformance/`

**Prerequisites**: [plan.md](plan.md) (required), [spec.md](spec.md) (required for user stories), [research.md](research.md), [data-model.md](data-model.md), [contracts/lint-rules-contract.md](contracts/lint-rules-contract.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US5)
- Every task names an exact file path

## Path Conventions

- `cmd/arc/lint/lint.go` / `lint_test.go` — Cobra command (unchanged flags/RunE structure) and its colocated E2E tests
- `internal/app/lint/kernel/lint.go` — `Rule` constants, `Violation`/`LintResult` value types (existing, extended)
- `internal/app/lint/service/*.go` — domain rule functions (existing four `rules_*.go` files, one new file)
- `internal/app/lint/service/*_test.go` — colocated unit tests, `github.com/fogfish/it/v2` only

---

## Phase 0: Pre-implementation Refactoring (fixture correction)

**Why included**: research.md D6 identified that two existing fixture surfaces are currently
"conformant" only because today's lint never checks the `## Requires`/`## Optional` contract at all —
once the new checks (User Stories 1-2) ship, both surfaces would immediately fail every test that
currently asserts "0 violations" against them. Per the plan's explicit constraint, these must be fixed
*before* the new checks are wired in, so the new checks' own tests (Phase 3+) start from genuinely
conformant fixtures rather than baking in a false pass. **MUST be submitted as a separate PR/commit from
the new-check implementation work**; every existing lint test (unit and E2E) MUST still pass after this
phase, unchanged in what it asserts.

- [X] T001 Fix `conformantSource`/`conformantEntity` in `cmd/arc/lint/lint_test.go` to genuinely satisfy `kernel.CoreTypeDefs` (`internal/app/schema/kernel/schema.go`): add the missing `abstract` prose to `conformantSource` (`source`'s `## Requires` includes it) and the missing `definition` prose + `mentionedIn` backlink to `conformantEntity` (`entity`'s `## Requires` includes both); replace `conformantEntity`'s current `mentions` edge (not listed under `entity`'s `## Requires` or `## Optional` — would newly fail FR-002) with `mentionedIn`, its intended inverse-backlink predicate. Verify `checkDerivedProvenance` (`internal/app/lint/service/rules_links.go`) still treats `Widget.md` as linking back to its source after this predicate-name change — it must, since the check looks for *any* link to a `source`-kind node, not `mentions` specifically, but confirm by running `go test ./cmd/arc/lint/...` before moving on. All of `lint_test.go`'s existing `Test...` functions MUST still pass unmodified after this change.
- [X] T002 [P] Fix `coreIndexFixtureLint` in `internal/app/lint/service/lint_test.go` to use real `Required`/`Optional`/`Role` data instead of its current hand-rolled, deliberately loose stand-in (zero-value `TypeDef`s, a single zero-value `PredicateDef{}}` for `"mentions"`): replace its `Types`/`Predicates` maps with `kernel.CoreTypeDefs`/`kernel.CorePredicateDefs` (import `github.com/fogfish/arcnet-cli/internal/app/schema/kernel`) for the `source`/`entity`/`resource`/`timeline` entries it needs, preserving the existing `"hypothesis"` extension-type entry in `TestLintIncludesNodeInNonStandardFolder` as a hand-added addition on top. Update `conformantSourceFixture`/`conformantEntityFixture` in the same file identically to T001's fixes (same `abstract`/`definition`/`mentionedIn` additions). All of `lint_test.go`'s existing `Test...` functions in this package MUST still pass unmodified after this change.

**Checkpoint**: `go test ./cmd/arc/lint/... ./internal/app/lint/...` passes fully, using fixtures now genuinely conformant with `CoreTypeDefs`/`CorePredicateDefs` — safe foundation for the new checks below.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Minimal scaffolding so new-check unit tests can compile before their implementation exists (Principle VI red phase).

- [X] T003 Add four new `Rule` constants to `internal/app/lint/kernel/lint.go`: `RuleTypeRequires Rule = "typeRequires"`, `RuleTypeOptional Rule = "typeOptional"`, `RuleIdentityQuoting Rule = "identityQuoting"`, `RulePredicateRole Rule = "predicateRole"` (contracts/lint-rules-contract.md's Rule table) — no other change to this file.
- [X] T004 [P] Create `internal/app/lint/service/rules_type_conformance.go` with the license header (per CLAUDE.md) and three empty-but-compiling stub functions — `checkTypeRequires`, `checkTypeOptional`, `checkPredicateRole` — each returning `nil` for now, matching the existing `checkXxx(node core.Node, path string, raw []byte, ...) []kernel.Violation` shape used by `rules_predicates.go`/`rules_identity.go`.

**Checkpoint**: `go build ./...` succeeds; no behavior change yet.

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate — the deliverable is a design decision recorded in a doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T005 Confirm no new domain entity/aggregate/value object is introduced — `data-model.md` shows this feature only adds new *readers* of already-`ARCHITECTURE.md`-documented entities (`Schema Index`, `Predicate Schema Node`, `Type Schema Node`, `Node`, `Checklist Rule`). Update `ARCHITECTURE.md`'s existing `Checklist Rule` glossary row's example list (currently "unique basenames, resolvable links, ... one ingest commit per document, absence of merge-conflict markers") to also mention the new Requires/Optional/identity-quoting/schema-driven-citation/predicate-role checks, and that lint now covers CORE §16 (not just §14).
- [X] T006 [P] Verify no new domain type is introduced inside `cmd/arc/lint` that duplicates an existing `internal/app/lint` or `internal/core` type (Principle II) — confirmed by data-model.md: all five checks read only pre-existing `core.Node`/`core.TypeDef`/`core.PredicateDef` fields.

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T007 Confirm no new flag, subcommand, or output-schema change is introduced (contracts/lint-rules-contract.md already documents the addendum: same CLI usage, same exit codes, same `--json` shape, only new `rule`/`message` string values inside the existing shape). Update `NewLintCmd`'s `Long` help text in `cmd/arc/lint/lint.go` to mention the fuller checklist coverage (Requires/Optional predicate contract, identity-key quoting, schema-driven citation predicates, predicate role conformance) alongside the existing description.

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 Confirm no new external integration or adapter is required — every new check reads the `core.Index` parameter `cmd/arc/lint/lint.go`'s `RunE` already resolves via the existing `appschema.Resolve(store)` call before invoking `Lint`; no new port interface, no new adapter package.

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> Depends on Phase 0 (T001) — these tests build on `buildConformantGraph`, which must already be genuinely conformant. Each test below MUST compile and fail semantically (red phase) until its story's implementation phase lands.

- [X] T009 [P] [US1] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md User Story 1's acceptance scenarios: a node missing a type-required predicate is reported under `[typeRequires]` naming the predicate/type/file (Scenario 1); a node carrying every required predicate produces no such violation (Scenario 2); a node of an unregistered type produces no `typeRequires` violation, only the existing `unrecognizedKind` one (Scenario 3)
- [X] T010 [P] [US2] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md User Story 2's acceptance scenarios: a node carrying a predicate listed under neither `## Requires` nor `## Optional` is reported under `[typeOptional]` (Scenario 1); a node carrying only Requires/Optional-listed predicates produces no such violation (Scenario 2); `"@id"`/`"@type"` are never reported as not-permitted (Scenario 3)
- [X] T011 [P] [US3] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md User Story 3's acceptance scenarios: missing `"@id"`/`"@type"` reported under `[identityQuoting]` naming which key (Scenario 1); a bare/unquoted `@id`/`@type` key reported under `[identityQuoting]` naming the key/file/line, while the node still parses correctly with no `frontMatter` violation (Scenario 2); both keys present and quoted produces no such violation (Scenario 3)
- [X] T012 [P] [US4] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md User Story 4's acceptance scenarios: a graph-registered `cito:`-aligned predicate not in today's hardcoded list is accepted with no `[citationPredicate]` violation (Scenario 1); an unregistered or non-`cito:`-aligned predicate used as a citation is still reported (Scenario 2); a graph with zero `cito:`-aligned predicates reports every citation usage as a violation, no built-in fallback (Scenario 3)
- [X] T013 [P] [US5] Write E2E test(s) in `cmd/arc/lint/lint_test.go` for spec.md User Story 5's acceptance scenarios: a predicate occurrence in a structural position inconsistent with its declared schema role is reported under `[predicateRole]` naming the predicate/role/file/line (Scenario 1); a node where every occurrence matches its predicate's declared role produces no such violation (Scenario 2); an unregistered predicate produces no `predicateRole` violation, only the existing `predicateRegistered` one (Scenario 3)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin. `go test ./cmd/arc/lint/...` currently fails on T009-T013 (expected — red phase).

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Shared logic User Stories 1, 2, and 5 all depend on — enumerating every predicate occurrence a node carries, tagged with its structural category, per data-model.md D5's field mapping.

- [X] T014 Implement a shared occurrence-enumeration helper in `internal/app/lint/service/rules_type_conformance.go` — walks `node.Attrs` (category `meta`), `node.Texts` (category `text`), `node.Edges` (category `edge-or-link`, via `Link.Predicate`), and `node.HRefs` entries with a non-empty `Predicate` flagged separately as citation-tagged/role-check-exempt (research.md D4) vs. empty-`Predicate` `HRefs` (category `href`) — returning, per distinct predicate name, the set of categories it occurs in plus a located line per occurrence (reuse `locatePredicateToken`/`locateFrontMatterField` from `locate.go` per category). `checkTypeRequires`/`checkTypeOptional`/`checkPredicateRole` (Phases 3, 4, 7) all consume this helper's output.
- [X] T015 [P] Add a raw-text regex helper for User Story 3 to `internal/app/lint/service/locate.go` — detects a bare (unquoted) `@id`/`@type` front-matter key line (research.md D1: `^@id\s*:`/`^@type\s*:`, distinguished from the existing quoted-key pattern `locateFrontMatterField` already matches via its literal `"@id"`/`"@type"` call sites).

**Checkpoint**: Foundation ready — User Stories 1, 2, 3, 5 can now proceed (User Story 4 has no foundational dependency beyond Phase 0/1).

---

## Phase 3: User Story 1 - Catch a node missing a predicate its type requires (Priority: P1) 🎯 MVP

**Goal**: Every node whose type is registered is checked against that type's `## Requires` list; any listed predicate absent from the node is reported.

**Independent Test**: Author a node of a registered type missing one `## Requires`-listed predicate; run lint; confirm exactly that predicate is reported under `[typeRequires]`, naming the predicate, type, and file.

### Implementation for User Story 1

> E2E tests were already written in Phase 2d (T009) and MUST currently be failing (red). Implementation below turns them green with minimal test changes.

- [X] T016 [US1] Implement `checkTypeRequires` in `internal/app/lint/service/rules_type_conformance.go` (depends on T014): for a node whose `node.Type` is a key in `index.Types`, report one `kernel.Violation{Rule: kernel.RuleTypeRequires, ...}` per predicate in `index.Types[node.Type].Required` absent from the occurrence set T014 produced; skip entirely when `node.Type` is not registered (FR-003). Message per contracts/lint-rules-contract.md's template.
- [X] T017 [US1] Wire `checkTypeRequires` into `Lint`'s existing "Checking predicates and citations" phase in `internal/app/lint/service/lint.go` (same loop as the existing `checkPredicateCase`/`checkPredicateRegistered`/`checkCitationPredicate` calls)
- [X] T018 [P] [US1] Add table-driven unit tests for `checkTypeRequires` in `internal/app/lint/service/rules_type_conformance_test.go`, covering: a required predicate present (no violation), a required predicate absent (one violation, correct `Rule`/message), an unregistered type (no violation — check skipped), a type whose `## Requires` is empty (no violation possible)

**Checkpoint**: User Story 1's E2E tests (T009) pass; this story is independently functional and testable.

---

## Phase 4: User Story 2 - Catch a node carrying a predicate its type doesn't permit (Priority: P1)

**Goal**: Every node whose type is registered is checked so that every predicate it carries — other than `"@id"`/`"@type"` — is listed under that type's `## Requires` or `## Optional`.

**Independent Test**: Author a node of a registered type carrying a predicate listed under neither `## Requires` nor `## Optional`; run lint; confirm exactly that predicate is reported under `[typeOptional]`.

### Implementation for User Story 2

> E2E tests were already written in Phase 2d (T010) and MUST currently be failing (red).

- [X] T019 [US2] Implement `checkTypeOptional` in `internal/app/lint/service/rules_type_conformance.go` (depends on T014): for a node whose `node.Type` is a key in `index.Types`, report one `kernel.Violation{Rule: kernel.RuleTypeOptional, ...}` per distinct predicate in T014's occurrence set that is in neither `index.Types[node.Type].Required` nor `.Optional`; skip entirely when `node.Type` is not registered (FR-003); never flag the identity predicates (they are never in T014's occurrence set to begin with — `core.Node.Attrs` already excludes `"@id"`/`"@type"` per its own doc comment, so this is structural, not a special case to add)
- [X] T020 [US2] Wire `checkTypeOptional` into `Lint`'s "Checking predicates and citations" phase in `internal/app/lint/service/lint.go`, alongside T017's call site
- [X] T021 [P] [US2] Add table-driven unit tests for `checkTypeOptional` in `internal/app/lint/service/rules_type_conformance_test.go`, covering: a predicate listed under `## Optional` (no violation), a predicate listed under neither (one violation), a type with an empty `## Optional` and a present non-Required predicate (violation — FR-002's "absent Optional means nothing extra permitted" edge case), a predicate listed under both `## Requires` and `## Optional` (no violation — malformed-type-schema tolerance per spec Edge Cases)

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently — the core Requires/Optional contract (spec's primary gap) is now fully enforced.

---

## Phase 5: User Story 3 - Catch missing or incorrectly quoted `@id`/`@type` front-matter keys (Priority: P2)

**Goal**: Every node's front matter is checked for `"@id"`/`"@type"` presence and correct (quoted-string) form.

**Independent Test**: Author a node with a bare `@id`/`@type` key, and a second missing one entirely; run lint; confirm both are reported under `[identityQuoting]` with file/line.

### Implementation for User Story 3

> E2E tests were already written in Phase 2d (T011) and MUST currently be failing (red).

- [X] T022 [US3] Implement `checkIdentityKeyQuoting` in `internal/app/lint/service/rules_frontmatter.go` (depends on T015): for each of `"@id"`/`"@type"`, report `kernel.Violation{Rule: kernel.RuleIdentityQuoting, ...}` when the key is missing from front matter, and separately when T015's regex finds the bare/unquoted form present; only ever called for nodes that reached `service.Lint`'s post-parse checks (a node whose front matter fails to parse at all already `continue`s past this point under the existing `RuleFrontMatter` violation, per `lint.go`'s existing control flow — no double-reporting)
- [X] T023 [US3] Wire `checkIdentityKeyQuoting` into `Lint`'s "Checking predicates and citations" phase in `internal/app/lint/service/lint.go` (or the "Checking basenames and links" phase alongside the other per-node front-matter checks — match whichever existing phase groups identity checks, per `lint.go`'s current phase boundaries)
- [X] T024 [P] [US3] Add table-driven unit tests for `checkIdentityKeyQuoting` in `internal/app/lint/service/rules_frontmatter_test.go`, covering: both keys present and quoted (no violation), `"@id"` unquoted (one violation naming `@id`), `"@type"` unquoted (one violation naming `@type`), a key entirely missing (one violation, distinct message from the unquoted case)

**Checkpoint**: User Stories 1, 2, AND 3 all pass their E2E tests independently.

---

## Phase 6: User Story 4 - Recognize citation predicates from the graph's own schema (Priority: P2)

**Goal**: `checkCitationPredicate` sources its valid-predicate vocabulary from the graph's schema (any predicate whose `Aligned` field has prefix `cito:`) instead of a hardcoded Go list.

**Independent Test**: Register a new `cito:`-aligned predicate in a graph's schema, use it as a citation; run lint; confirm no false-positive violation, while a non-`cito:`-aligned/unregistered predicate is still rejected.

### Implementation for User Story 4

> E2E tests were already written in Phase 2d (T012) and MUST currently be failing (red).

- [X] T025 [US4] Rewrite `checkCitationPredicate` in `internal/app/lint/service/rules_predicates.go` to accept a `registry map[string]core.PredicateDef` parameter and treat predicate `p` as valid iff `strings.HasPrefix(registry[p].Aligned, "cito:")`; delete the hardcoded `citoPredicates` map (research.md D3). Message text (contracts/lint-rules-contract.md) is unchanged; only the determination source changes.
- [X] T026 [US4] Update `checkCitationPredicate`'s call site in `internal/app/lint/service/lint.go` to pass `index.Predicates`
- [X] T027 [P] [US4] Update `internal/app/lint/service/rules_predicates_test.go`'s existing `TestCheckCitationPredicate*` tests for the new signature (pass an explicit registry fixture per test), and add new cases: a `cito:`-aligned predicate not in today's old hardcoded list is accepted (registry-driven, no code-level allowlist), a registered-but-non-`cito:`-aligned predicate is rejected, an empty registry rejects every citation usage (no built-in fallback, FR-007 Acceptance Scenario 3)

**Checkpoint**: User Stories 1, 2, 3, AND 4 all pass their E2E tests independently.

---

## Phase 7: User Story 5 - Catch a predicate used in a structural position its schema role doesn't permit (Priority: P3)

**Goal**: Every predicate occurrence on a node is checked against its schema-declared `Role`.

**Independent Test**: Author a node with a `text`-role predicate written as an edge bullet (or an `edge`-role predicate written as prose); run lint; confirm the mismatch is reported under `[predicateRole]` naming the predicate/role/file/line.

### Implementation for User Story 5

> E2E tests were already written in Phase 2d (T013) and MUST currently be failing (red).

- [X] T028 [US5] Implement `checkPredicateRole` in `internal/app/lint/service/rules_type_conformance.go` (depends on T014): for each occurrence T014 produced, look up `index.Predicates[predicate]`; skip when unregistered (FR-009) or when its `Role` is empty/not one of `meta`/`text`/`href`/`edge`/`link` (research.md D7); skip `HRefs` occurrences flagged citation-tagged by T014 (research.md D4); otherwise report `kernel.Violation{Rule: kernel.RulePredicateRole, ...}` when the occurrence's category (per data-model.md D5's table: `meta`↔`meta`, `text`↔`text`, `href`↔`href`, `edge-or-link`↔(`edge` or `link`)) doesn't match the registered `Role`
- [X] T029 [US5] Wire `checkPredicateRole` into `Lint`'s "Checking predicates and citations" phase in `internal/app/lint/service/lint.go`
- [X] T030 [P] [US5] Add table-driven unit tests for `checkPredicateRole` in `internal/app/lint/service/rules_type_conformance_test.go`, covering: matching role/category (no violation), a `text`-role predicate found as an `Edges` occurrence (violation), an `edge`-role predicate found in `Texts` (violation), an unregistered predicate (skipped, no violation), a predicate with an empty/unrecognized `Role` (skipped, no violation), a citation-tagged `HRefs` occurrence of an `edge`-role predicate like `citesAsEvidence` (skipped, no violation — research.md D4)

**Checkpoint**: All five user stories (US1-US5) pass their E2E tests independently — the full CORE §16 checklist gap this feature targets is closed.

---

## Additional Polish

- [X] T031 [P] Run [quickstart.md](quickstart.md)'s six scenarios manually end-to-end against a real `arc init`-seeded graph, confirming each matches its documented expected output
- [X] T032 Run `go test ./... -cover` and confirm no pre-existing test outside `cmd/arc/lint`/`internal/app/lint` newly fails (spec's own scope note: no other command's tests invoke `service.Lint`, so none should be affected, but this is the final confirmation)
- [X] T033 [P] Run `staticcheck ./...` and confirm it is clean on all new/changed files (constitution Mandatory Libraries & Tooling)

---

## Bugfix: BUG-001 — `kernel.CoreTypeDefs`/`CorePredicateDefs` seed-data gaps false-positive against spec-conformant graphs

**Purpose**: Fix `specs/014-lint-spec-conformance/bugs/BUG-001.md`. No task above (T001-T033) is reopened —
`checkTypeRequires`/`checkTypeOptional` correctly implement FR-001/FR-002 exactly as designed; the defect
is in the seed data those checks read (`internal/app/schema/kernel/schema.go`'s `CoreTypeDefs`/
`CorePredicateDefs`, spec 011's own deliverable), not in this feature's own check logic. Discovered via
this feature's own new checks, immediately post-implementation, by reproducing against a real
`arc init`-seeded graph. See spec.md FR-014–FR-020/SC-007 and research.md's "Bugfix: BUG-001" section
(D8-D10) for the exact, concrete per-type/per-predicate decisions these tasks implement.

- [X] T034 In `internal/app/schema/kernel/schema.go`'s `CorePredicateDefs`, add `"indexed": {Role: "meta", Merge: core.MergeImmutable, Aligned: "arc:indexed", Description: "..."}` (spec.md FR-019, research.md D9) — a real, already-in-use predicate (`internal/app/graph/service/apply.go`'s `setAttr(merged.Attrs, "indexed", stamp)`) that was never registered
- [X] T035 [P] In the same file's `CorePredicateDefs`, add `"scoreZ": {Role: "meta", Merge: core.MergeValidatedOverwrite, Aligned: "arc:scoreZ", Description: "..."}` and `"scoreC": {Role: "meta", Merge: core.MergeValidatedOverwrite, Aligned: "arc:scoreC", Description: "..."}` (spec.md FR-020, research.md D10) — do not touch `cmd/arc/graph/apply_test.go`'s hyphenated `score-c`/`score-z` fixture, which deliberately tests unregistered-predicate fallback (research.md D10's explicit note)
- [X] T036 In `CoreTypeDefs["source"].Optional`, add `created`, `updated`, `indexed`, `scoreZ`, `scoreC` (spec.md FR-014/FR-019/FR-020, research.md D8 table) — existing entries (`authors`, `url`, `cites`, `tags`, `doi`) and `Required` list unchanged
- [X] T037 In `CoreTypeDefs["entity"].Optional`, add `notes`, `published`, `created`, `updated`, `indexed`, `scoreZ`, `scoreC`, `mentions`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related` (spec.md FR-014–FR-020, research.md D8 table) — existing entries (`aliases`, `tags`) and `Required` list (including `mentionedIn`) unchanged
- [X] T038 In `CoreTypeDefs["resource"].Optional`, add `tags`, `text`, `published`, `created`, `updated`, `indexed`, `scoreZ`, `scoreC`, `mentions`, `mentionedIn`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related` (spec.md FR-014–FR-020, research.md D8 table) — existing entries (`url`, `isCitedBy`, `authors`, `year`, `doi`, `status`, `notes`) and `Required` list unchanged
- [X] T039 In `CoreTypeDefs["timeline"].Optional`, add `tags`, `text`, `created`, `updated`, `indexed`, `scoreZ`, `scoreC`, `mentions`, `mentionedIn` (spec.md FR-014/FR-016/FR-019/FR-020, research.md D8 table — `published` deliberately excluded, FR-015) — existing entry (`heading`) and `Required` list unchanged
- [X] T040 [P] In `internal/app/schema/kernel/schema_test.go`, add `"indexed"`, `"scoreZ"`, `"scoreC"` to `TestCorePredicateDefsContainsFullCoreVocabulary`'s `names` slice (updates the exact-count assertion accordingly)
- [X] T041 [P] In `internal/app/schema/kernel/schema_test.go`, add a new `TestCoreTypeDefsOptionalListsIncludeCrossCuttingPredicates` table-driven test asserting each of the four `CoreTypeDefs` entries' `Optional` list contains exactly the predicates research.md D8's table specifies for it — closes the test gap `TestCoreTypeDefsRequiredListsMatchCoreSection11` left open (it only ever asserted `Required`, never `Optional`), which is what let this bug go unnoticed since spec 011 shipped
- [X] T042 [P] Add a regression E2E test in `cmd/arc/lint/lint_test.go`: an `entity` node using `conformsTo` (verbatim per ARCNET-CORE §11.3's own worked example) and separately carrying `notes` produces no `[typeOptional]` violation; a `resource` node using a §10.5 semantic predicate likewise produces none — reproduces and closes this bug's own originally-reported false positive
- [X] T043 Run `go test ./...`, `staticcheck ./...`, and re-run [quickstart.md](quickstart.md)'s six scenarios end-to-end against a real `arc init`-seeded graph, confirming the broadened seed data introduces no regression

**Checkpoint**: `go test ./...` passes fully; the `entity`/`conformsTo` false positive from BUG-001's report no longer reproduces; `schema_test.go` now asserts `Optional` lists, not just `Required`.

---

## Bugfix: BUG-002 — timeline period files never satisfy their own schema, and the spec's own annotated-bullet format is unparseable

**Purpose**: Fix `specs/014-lint-spec-conformance/bugs/BUG-002.md`. No task above (including BUG-001's
T034-T043) is reopened. See spec.md FR-021–FR-024/SC-008/SC-009 and research.md's "Bugfix: BUG-002"
section (D11-D14) for the exact, concrete decisions these tasks implement, including the explicit product
decision to reuse `cites` in place of `entries` (D12) and its consequences.

- [X] T044 In `internal/app/schema/kernel/schema.go`'s `CorePredicateDefs`, add `"period": {Role: "meta", Merge: core.MergeImmutable, Aligned: "arc:period", Description: "..."}` (spec.md FR-021, research.md D11)
- [X] T045 In the same file's `CorePredicateDefs`, remove the `"entries"` entry entirely (spec.md FR-022, research.md D12 — no real graph ever produced a genuine `entries`-tagged edge, so nothing depends on it remaining registered)
- [X] T046 In the same file's `CorePredicateDefs["cites"]`, change `Merge` from `core.MergeUnion` to `core.MergeAppend` and broaden `Description` to name both usages (a source's own citation of an external resource, and a timeline's chronological reference to a source it contains) — `Role`/`Aligned` unchanged (spec.md FR-022, research.md D12)
- [X] T047 In `CoreTypeDefs["timeline"]`, change `Required` from `[]string{"granularity", "entries"}` to `[]string{"granularity", "cites", "period"}` (spec.md FR-021/FR-022, research.md D11/D12) — `Optional` list (broadened by T039/BUG-001) unchanged
- [X] T048 [P] In `internal/core/timeline.go`, change `TimelineEntry`'s rendered format from `"- [[%s]] — *%s* (%s) — %s"` to `"- cites:: [[%s]] — *%s* (%s) — %s"` and update its doc comment, which currently (incorrectly, per this bugfix) states "the timeline node's own Edges carry only the bare target" (spec.md FR-023, research.md D13)
- [X] T049 [P] In `internal/core/timeline_test.go`, update `TestTimelineEntry`'s expected string to the new `cites::`-prefixed format (research.md D13)
- [X] T050 In `internal/app/graph/service/apply.go`, change `timelineEntryPattern` from `` ^- \[\[([^\]]+)\]\].* — (\d{4}-\d{2}-\d{2})$ `` to `` ^- (?:cites:: )?\[\[([^\]]+)\]\].* — (\d{4}-\d{2}-\d{2})$ `` (optional prefix, not required — tolerates re-parsing an already-existing pre-fix period file without losing or duplicating entries) (research.md D13)
- [X] T051 In `internal/core/markdown.go`, relax `listItemPattern`'s trailing anchor from `\]\]$` to `` \]\](?:\s.*)?$ `` so a predicate-tagged wikilink followed by whitespace-then-annotation text parses into `Link{Predicate, Target}` instead of being silently dropped entirely (spec.md FR-024, research.md D14) — do not touch `inlineLinkPattern` (out of this bugfix's scope, research.md D14's explicit note)
- [X] T052 [P] Add a unit test in `internal/core/markdown_test.go` (or wherever `parseListItemLink`/list-item parsing is already tested) asserting that `entries:: [[rescorla-2026-tls13]] — *TLS 1.3: Design and Rationale* (Eric Rescorla) — 2026-04-12` (ARCNET-CORE §11.5's own literal worked-example line) parses into `Link{Predicate: "entries", Target: "rescorla-2026-tls13"}` with the trailing annotation discarded — this specifically verifies FR-024's parser fix using the predicate name the upstream spec's own example happens to use, independent of this bugfix's own `cites` naming decision (D12)
- [X] T053 [P] Add a regression E2E test in `cmd/arc/lint/lint_test.go` (or `internal/app/lint/service/lint_test.go`): a full `arc init` → `arc apply` → `arc lint` run (or an equivalent fixture) produces zero `[typeRequires]`/`[typeOptional]` violations on the generated `timeline/yearly/`/`timeline/monthly/` period files — reproduces and closes this bug's own originally-reported false positive
- [X] T054 Run `go test ./...`, `staticcheck ./...`, and re-verify end-to-end against a real graph: `arc init` → `arc apply` a patch (the reporter's own `dmitry-2026-graph` example) → `arc lint` reports zero timeline-related violations; also re-run [quickstart.md](quickstart.md)'s six scenarios to confirm no regression

**Checkpoint**: `go test ./...` passes fully; the exact `dmitry-2026-graph` end-to-end repro from BUG-002's report produces zero `[typeRequires]`/`[typeOptional]` violations on either generated timeline period file; ARCNET-CORE §11.5's own worked-example line parses into a populated `Edges` entry instead of being silently dropped.

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects the `Checklist Rule` glossary update from T005, if any (Principle I)
- [X] TN02 Domain concepts confirmed as already present in [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary — no new entities were introduced by this feature (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: no flag/exit-code/schema change, only `Long` help text refreshed (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 No new architectural pattern was introduced — no new ADR required (Principle I)
- [X] TN05 New rule functions remain plain domain functions with no Cobra/`cmd/`-package import (Principle III)
- [X] TN06 Unit tests (T018, T021, T024, T027, T030) were written first per their phase, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 No new external integration was introduced; N/A for adapter/port review (Principle VII)
- [X] TN10 N/A — this feature produces no new terminal output beyond existing `Violation` rendering (Principle X)
- [X] TN11 N/A — no new configuration/secret surface introduced (Principle XI)
- [X] TN12 `Long` help text refreshed for `arc lint` per T007 (Principle XII)
- [X] TN13 E2E tests from Phase 2d (T009-T013) turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature (User Stories 1-5, all Acceptance Scenarios) have a passing, colocated E2E test in `cmd/arc/lint/lint_test.go` (Principle VIII)
- [X] TN15 Release/versioning impact assessed: additive only — new possible `rule` string values inside the existing `--json` schema, no command/flag/schema-breaking change, no major version bump required (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Pre-implementation Refactoring (Phase 0)**: No dependencies — separate PR, run first
- **Setup (Phase 1)**: Depends on Phase 0 (fixtures must be correct before new-check stubs are exercised by any test)
- **Design Preconditions (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories; 2a/2b/2c can proceed in parallel with each other; 2d depends on Phase 0
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3-7)**: All depend on Phase 2.5 (US1/US2/US5 need T014; US3 needs T015; US4 needs only Phase 0/1)
  - User stories can proceed in parallel (if staffed) or sequentially in priority order (US1/US2 → US3/US4 → US5)
- **Additional Polish**: Depends on all five user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2.5 (T014) — no dependency on other stories
- **User Story 2 (P1)**: Depends on Phase 2.5 (T014) — independently testable from US1, though both touch the same new file (`rules_type_conformance.go`) and the same `lint.go` orchestrator call-site region
- **User Story 3 (P2)**: Depends on Phase 2.5 (T015) — no dependency on other stories
- **User Story 4 (P2)**: Depends only on Phase 0/1 — no dependency on other stories, can start immediately after Setup
- **User Story 5 (P3)**: Depends on Phase 2.5 (T014) — no dependency on other stories

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Rule function implementation before its `lint.go` orchestrator wiring
- Orchestrator wiring before/alongside unit tests
- Story complete before moving to next priority

### Parallel Opportunities

- T001/T002 (Phase 0) touch different files — parallelizable
- T009-T013 (Phase 2d E2E test-writing) touch the same file (`cmd/arc/lint/lint_test.go`) — sequential in practice despite differing story labels, unless split by a team into separate local branches merged after
- T014/T015 (Phase 2.5) touch different files — parallelizable
- Once Phase 2.5 completes, US1/US2/US3/US5 implementation can proceed in parallel by different developers (different rule functions in the same new file need care merging; US3/US4 touch entirely separate existing files and are cleanly parallel)
- T018, T021, T024, T027, T030 (per-story unit tests) are each `[P]` within their own phase

---

## Parallel Example: User Stories 1 and 2 (both P1, same new file)

```bash
# Foundational (Phase 2.5, already complete before this point):
# T014 shared occurrence-enumeration helper in rules_type_conformance.go

# T016 (US1) and T019 (US2) both add functions to rules_type_conformance.go —
# implement sequentially within that file to avoid merge conflicts, but their
# unit tests are independent files/functions and can be written in parallel:
Task: "Add table-driven unit tests for checkTypeRequires in rules_type_conformance_test.go"
Task: "Add table-driven unit tests for checkTypeOptional in rules_type_conformance_test.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 — the core Requires/Optional contract)

1. Complete Phase 0: fixture correction (separate PR)
2. Complete Phase 1: Setup
3. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
4. Complete Phase 2.5: Foundational Infrastructure
5. Complete Phase 3: User Story 1
6. Complete Phase 4: User Story 2
7. Complete Phase N: Constitution Compliance Verification
8. **STOP and VALIDATE**: Requires/Optional enforcement works end-to-end, independent of US3-US5
9. Deploy/demo if ready — this alone closes the spec's primary named gap

### Incremental Delivery

1. Phase 0 → Phase 1 → Phase 2 → Phase 2.5 → Foundation ready
2. Add User Story 1 + User Story 2 together (both P1, share the new file) → Verify against Phase N → MVP
3. Add User Story 3 (identity quoting) → Verify → Deploy/Demo
4. Add User Story 4 (schema-driven citations) → Verify → Deploy/Demo
5. Add User Story 5 (predicate role) → Verify → Deploy/Demo
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

1. One contributor completes Phase 0 (fixture correction) first, alone — everything else blocks on it
2. Team completes Setup + Design Preconditions + Foundational Infrastructure together
3. Once complete:
   - Developer A: User Stories 1 + 2 (`rules_type_conformance.go`'s Requires/Optional functions)
   - Developer B: User Story 3 (`rules_frontmatter.go`) + User Story 4 (`rules_predicates.go`) — disjoint files
   - Developer C: User Story 5 (`rules_type_conformance.go`'s role function, after A lands T014-dependent scaffolding)
4. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 0's fixture correction is a prerequisite discovered by research.md D6, not optional busywork — skipping it means every new check's own tests would need to work around already-non-conformant fixtures, defeating the point of fixing them once, up front
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements

**Bugfix**: 2026-07-09 — BUG-001: added T034-T043 (no task above reopened — T001-T033 correctly implemented
this feature's own checks; the defect is in the seed data those checks read, spec 011's
`kernel.CoreTypeDefs`/`CorePredicateDefs`, never this feature's own logic). Added spec.md FR-014–FR-020
and SC-007; recorded the concrete per-type/per-predicate scope decisions in research.md's "Bugfix:
BUG-001" section (D8-D10). Reported and reproduced immediately after T001-T033 were completed and this
feature's new `typeOptional` check was run against a real graph for the first time: an `entity` node
using `conformsTo` — verbatim from ARCNET-CORE §11.3's own worked example — was falsely rejected, because
`kernel.CoreTypeDefs["entity"].Optional` (`["aliases", "tags"]`) omitted `notes` and all nine §10.5
semantic predicates the spec's own worked example lists. See `bugs/BUG-001.md` for the full report.

**Bugfix**: 2026-07-10 — BUG-002: added T044-T054 (no task above reopened — T001-T043 correctly
implemented what they set out to; this bugfix's defects live in `internal/core/timeline.go` (spec 009),
`internal/core/markdown.go` (spec 010), and `internal/app/graph/service/apply.go` (spec 003/009), plus
`kernel.CoreTypeDefs`/`CorePredicateDefs` again). Added spec.md FR-021–FR-024 and SC-008/SC-009; recorded
the concrete decisions in research.md's "Bugfix: BUG-002" section (D11-D14). Reported and reproduced via
the reporter's own `dmitry-2026-graph` example: every timeline period file `arc apply` produces failed
`[typeRequires]`(`entries`)/`[typeOptional]`(`period`). Three stacked causes: `period` was written but
never registered (same shape as BUG-001's `indexed`); `entries` was required by schema but
`internal/core.TimelineEntry` never actually wrote a predicate-tagged edge for it, by original,
now-corrected design; and, independently, `internal/core.ParseNode`'s list-item parser was confirmed to
silently drop ARCNET-CORE §11.5's own literal worked-example bullet shape entirely (predicate-tagged
wikilink followed by trailing annotation) — the most severe of the three, since it is silent data loss,
not merely a lint false positive. Per the reporter's explicit choice (offered as a question, since it
otherwise conflicts with the fetched authoritative `ARCNET-CORE.md` text), this bugfix reuses the
existing `cites` predicate for timeline membership rather than keeping `entries`, changing `cites`'s own
`merge` from `union` to `append` to preserve the ordering guarantee `entries` had. See `bugs/BUG-002.md`
for the full report.
