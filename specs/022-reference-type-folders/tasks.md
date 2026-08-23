---
description: "Task list for 022-reference-type-folders"
---

# Tasks: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Input**: Design documents from `specs/022-reference-type-folders/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Constitution Principles VI and VIII mandate that every `spec.md` acceptance scenario maps 1:1 to a colocated E2E test, written before implementation (red-green-refactor).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 — maps to `spec.md` user stories
- Exact file paths are given in every task

## Path Conventions

- `cmd/arc/<group>/` — Cobra commands + colocated E2E tests (Principles III, VIII)
- `internal/core/` — format-level parse/render, no app dependencies
- `internal/app/<domain>/kernel/` — domain value types (the three tables this feature changes)
- `internal/app/<domain>/service/` — domain logic

## ⚠️ Read before starting

1. **Export `GOROOT` first.** The shell default points at an uninstalled Go; every `go` command fails with `cannot find GOROOT directory`.
2. **This feature changes no command surface.** No new command, package, flag, or `--json` field. If a task seems to require one, stop — it is out of scope (`spec.md` FR-022).
3. **Never assert a path with a filesystem stat.** macOS/APFS is case-insensitive and would pass `source/` for `Source/`. Compare the exact string handed to the store (contract C7).
4. **Do not touch `pluralizeKind`** in [cmd/arc/graph/apply.go:31](../../cmd/arc/graph/apply.go#L31). It formats a display count (`+3 entities`), not a folder. research.md D4.

---

## Phase 1: Setup

**Purpose**: Establish a verified-green baseline.

- [X] T001 Export the toolchain and confirm a green baseline: `export GOROOT="$(ls -d /opt/homebrew/Cellar/go/*/libexec | tail -1)"; export PATH="$GOROOT/bin:$PATH"; go build ./... && go test ./...` from repo root — all packages must pass before any edit
- [X] T002 [P] Confirm `staticcheck` runs clean on the baseline so a later failure is attributable to this feature — **note the toolchain**: the locally installed `staticcheck`, and even `staticcheck@2025.1.1`, cannot read go1.27's export data (`export data version 4 is greater than maximum supported version 2`) and report only that internal error. Run it under the toolchain CI actually uses: `GOTOOLCHAIN=go1.26.6 go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...`. Clean at baseline and clean at T053.

---

## Phase 2: Design Preconditions

**Purpose**: The constitution's PRECONDITIONS (Compliance Checklist). Each subsection is a design gate — the deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No implementation (Phase 3+) may begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T003 Add a **Reference** entry to the Glossary in [ARCHITECTURE.md](../../ARCHITECTURE.md): an external work the graph points to but has not ingested; required `title`/`ref`/`relevance`; optional `url`/`authors`/`year`/`doi`/`status`/`isCitedBy`/`notes`; filed at `Reference/<id>.md`; distinct from `Source` (ingested) and `Resource` (ingested fragment)
- [X] T004 Rewrite the **Entity/Resource Node** Glossary entry in [ARCHITECTURE.md](../../ARCHITECTURE.md) (line ~247) — it still carries the retired "referenced material" meaning of `Resource`; replace with the v0.11 ingested-fragment meaning
- [X] T005 Add a **Type Folder** / **Functional Folder** pair of Glossary entries to [ARCHITECTURE.md](../../ARCHITECTURE.md) capturing contract C1/C2: a type folder's name equals its type name character-for-character; `_schema/` and `timeline/` are the two exemptions
- [X] T006 Verify no new Go type is introduced by this feature (data-model.md §1 — only table *values* and two function bodies change); record the confirmation in the PR description

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T007 Record explicitly that the command/flag/`--json` surface is **unchanged** by this feature — no subcommand, flag, exit code, or output-schema change. No `contracts/` command table is required; [contracts/folder-layout-contract.md](contracts/folder-layout-contract.md) and [contracts/core-type-vocabulary-contract.md](contracts/core-type-vocabulary-contract.md) are the two contracts this feature carries

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 Confirm **not applicable**: no new external system, no new adapter, `internal/adapter/fsys` unchanged (plan.md Technical Context)

### Phase 2d: E2E Acceptance Test Design (Principle VIII) — RED PHASE

> Every task below writes tests that MUST compile and MUST fail semantically before Phase 3 begins.

- [X] T009 [P] [US1] Write E2E tests in [cmd/arc/ctrl/init_test.go](../../cmd/arc/ctrl/init_test.go) for `spec.md` US1 scenarios 1–3 and 5: a freshly initialized graph's `_schema/Class/Resource.md` requires exactly `text`/`tags`/`mentionedIn` and offers `notes`; contains none of `ref`/`relevance`/`url`/`authors`/`year`/`doi`/`status`/`isCitedBy`; `_schema/Class/Reference.md` exists with the C3 lists and `subClassOf:: [[Node]]`; the graph lints clean
- [X] T010 [P] [US1] Write E2E tests in [cmd/arc/graph/apply_test.go](../../cmd/arc/graph/apply_test.go) for `spec.md` US1 scenarios 4 and 6: applying a patch carrying a `Reference` node succeeds with no unknown-type diagnostic, and a `Reference` node with `title`/`ref`/leading prose lints with zero type-conformance violations
- [X] T011 [P] [US2] Write E2E tests in [cmd/arc/graph/apply_test.go](../../cmd/arc/graph/apply_test.go) for `spec.md` US2 scenarios 1, 2, 4, 5, 6: a `Resource` node's leading prose stores under `text` and nothing under `relevance`; a `Reference` node's under `relevance`; an unknown type's under `text`; `Source`/`Entity` unchanged (`abstract`/`definition`); lint reports no missing-`text`/undeclared-`relevance` on the applied `Resource`
- [X] T012 [P] [US2] Write an E2E test in [cmd/arc/graph/revert_test.go](../../cmd/arc/graph/revert_test.go) for `spec.md` US2 scenario 3: apply then revert a source contributing one `Resource` and one `Reference`; each node's prose is recognized under the key it was written with and no prose is orphaned
- [X] T013 [P] [US3] Write an E2E test in [cmd/arc/ctrl/init_test.go](../../cmd/arc/ctrl/init_test.go) for `spec.md` US3 scenario 1: `arc init` creates exactly the eight folders of contract C3, asserted as **exact string set equality** against the store, and creates none of the six retired names
- [X] T014 [P] [US3] Write E2E tests in [cmd/arc/graph/apply_test.go](../../cmd/arc/graph/apply_test.go) for `spec.md` US3 scenarios 2–4: each core node lands at the exact path `<TypeName>/<id>.md`; an unrecognized domain type `Thought` lands at `Thought/<id>.md` (**not** `Thoughts/`, **not** `thoughts/`); `Timeline` nodes stay under `timeline/yearly|monthly/` with no flat `Timeline/` folder created
- [X] T015 [P] [US3] Write an E2E test in [cmd/arc/ctrl/apply_schema_test.go](../../cmd/arc/ctrl/apply_schema_test.go) for `spec.md` US3 scenario 5: an applied schema patch writes its predicate node under `_schema/Property/` and its type node under `_schema/Class/`
- [X] T016 [P] [US3] Write E2E tests in [cmd/arc/graph/revert_test.go](../../cmd/arc/graph/revert_test.go) and [cmd/arc/graph/subgraph_test.go](../../cmd/arc/graph/subgraph_test.go) for `spec.md` US3 scenarios 6–8: revert locates and removes nodes from type-named folders and rewrites referrers; subgraph → apply into a second fresh graph is path-stable; subgraph/grep/lint still exclude `.arc/` and `_schema/`

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T017 Confirm **not applicable**: no config value, environment variable, or secret is introduced or read differently by this feature

**Checkpoint**: All Phase 2 subsections complete — implementation may begin.

---

## Phase 2.5: Foundational — Drift-Guard Invariants

**Purpose**: The two invariants research.md identified as this feature's highest-risk silent failures. Both are written **before** any production change and both start red. Neither belongs to a single story; both must hold for all three.

- [X] T018 [P] Write a failing unit test in [internal/app/ctrl/kernel/graph_test.go](../../internal/app/ctrl/kernel/graph_test.go) asserting contract C6: every content type in `schema/kernel.CoreTypeBases` appears **verbatim** as an entry in `DefaultLayout.Folders`, and every entry in `DefaultLayout.Folders` that is neither under `timeline/` nor under `_schema/` is some content type's name. Import `internal/app/schema/kernel` (test-only; creates no production dependency)
- [X] T019 [P] Write a failing unit test in [internal/app/graph/service/revert_internal_test.go](../../internal/app/graph/service/revert_internal_test.go) asserting that `revertLeadingKey(t)` equals `core.TextPredicateFor(t, true)` for the five core types `Source`, `Entity`, `Resource`, `Timeline`, `Reference`. Do **not** assert over the three domain types (`hypothesis`, `aporia`, `thought`) — they are already divergent and out of scope (research.md D6b, follow-up 1)

**Checkpoint**: T018 fails for the right reason — `DefaultLayout.Folders` genuinely disagrees with the target.

**T019 is green at baseline, and that is correct.** The task text expected it red; the premise
does not hold. research.md D6b's own table shows the two prose-key tables *agree* on `Source`,
`Entity`, and `Resource`, and both fall through to `text` for the then-unknown `Reference` —
they diverge only on `hypothesis`/`aporia`/`thought`, the three domain types T019 deliberately
excludes. So no assertion T019 is allowed to make can fail before T030. Its value is
forward-looking: it turns red the moment T030 changes `core`'s table without T031 changing
`revert`'s, which is exactly the drift it exists to catch. Verified red in that window during
Phase 4 (see the Phase 4 checkpoint).

---

## Phase 3: User Story 1 — Corrected core type vocabulary (Priority: P1) 🎯 MVP

**Goal**: A freshly initialized graph declares five core content types, with `Resource` meaning an ingested fragment and `Reference` meaning an un-ingested external work. Neither claims the other's predicates.

**Independent Test**: Initialize a fresh graph, read back `_schema/Class/Resource.md` and `_schema/Class/Reference.md`, and confirm both match contract C2/C3. Lint reports clean.

> E2E tests T009–T010 were written in Phase 2d and MUST currently be failing.

- [X] T020 [US1] Redefine `CoreTypeDefs["Resource"]` in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go#L118) per data-model.md §1.1: `Required: ["text","tags","mentionedIn"]`, `Optional: ["notes"]`, `Merge` unchanged, description replaced with the ingested-fragment meaning. Delete the retired structural/semantic optionals — §11.4 lists `notes` alone
- [X] T021 [US1] Add `CoreTypeDefs["Reference"]` in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go) per data-model.md §1.2: `Merge: MergeFirstWriteWin`, `Required: ["title","ref","relevance"]`, `Optional: ["url","authors","year","doi","status","isCitedBy","notes"]`, description per contract C3
- [X] T022 [US1] Add `"Reference": {"Node"}` to `CoreTypeBases` in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go#L159)
- [X] T023 [P] [US1] Reword five predicate descriptions in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go) per data-model.md §1.4: `ref`, `status`, `relevance`, `cites`, `isCitedBy` — each currently names "resource" while describing external-work semantics
- [X] T024 [P] [US1] Update the `Node` type description in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go) to enumerate five inheriting content types, not four
- [X] T025 [US1] Update `TestCoreTypeDefsContainsCoreTypesAndSchemaTypesThemselves` in [internal/app/schema/kernel/schema_test.go](../../internal/app/schema/kernel/schema_test.go#L57) — the count assertion moves from 7 to 8 and the name list gains `Reference`
- [X] T026 [US1] Update `TestCoreTypeDefsRequiredListsMatchCoreSection11` in [internal/app/schema/kernel/schema_test.go](../../internal/app/schema/kernel/schema_test.go#L77) — `Resource` required changes from `ref`,`relevance` to `text`,`tags`,`mentionedIn`; add a `Reference` assertion
- [X] T027 [US1] Add a unit test in [internal/app/schema/kernel/schema_test.go](../../internal/app/schema/kernel/schema_test.go) asserting data-model.md §4 rule 5: every predicate named in `Reference`'s Required ∪ Optional is a key of `CorePredicateDefs` — this feature introduces no new predicate
- [X] T028 [US1] Add seed golden assertions in [internal/app/schema/service/schema_test.go](../../internal/app/schema/service/schema_test.go): `Seed()` emits `_schema/Class/Reference.md`, and its `_schema/Class/Resource.md` contains none of `ref`/`relevance`/`url`/`authors`/`year`/`doi`/`status`/`isCitedBy`
- [X] T029 [US1] Repair lint and apply fixtures whose `Resource` nodes carry the retired external-work shape and now violate type conformance — sweep [internal/app/lint/service/rules_type_conformance_test.go](../../internal/app/lint/service/rules_type_conformance_test.go), [cmd/arc/lint/lint_test.go](../../cmd/arc/lint/lint_test.go), [internal/app/graph/service/apply_test.go](../../internal/app/graph/service/apply_test.go). **Note the US1↔US2 coupling** (see Dependencies): a fixture whose `Resource` prose is keyed `relevance` stays red until T031 lands

**Checkpoint**: T009, T010, T025–T028 green. `Reference` is a first-class type; `Resource` carries the corrected definition.

---

## Phase 4: User Story 2 — Prose round-trips under the correct predicate (Priority: P2)

**Goal**: A `Resource` node's leading prose is stored and read back under `text`; a `Reference` node's under `relevance`. Write side and read side agree for every type.

**Independent Test**: Apply a patch with one `Resource` and one `Reference`, inspect both files, then revert and confirm each body is reconstructed from the key it was written with.

> E2E tests T011–T012 were written in Phase 2d and MUST currently be failing.

- [X] T030 [US2] Change `textPredicateFor`'s leading switch in [internal/core/markdown.go](../../internal/core/markdown.go#L384) per data-model.md §2.2: `Resource` → `text`, add `Reference` → `relevance`. `Source`/`Entity`/default and the unconditional trailing `notes` are unchanged
- [X] T031 [US2] Mirror the change in `revertLeadingKey` in [internal/app/graph/service/revert.go](../../internal/app/graph/service/revert.go#L398): `Resource` → `text`, add `Reference` → `relevance`. Leave `hypothesis`/`aporia`/`thought` untouched (research.md D6b) — this turns T019 green
- [X] T032 [P] [US2] Add a table-driven unit test in [internal/core/markdown_test.go](../../internal/core/markdown_test.go) covering `TextPredicateFor` for all five core types × `{leading, trailing}` (10 cases) plus one unrecognized type, per contract C4
- [X] T033 [P] [US2] Add parse→render→parse fixtures in [internal/core/markdown_test.go](../../internal/core/markdown_test.go) for `Resource` and `Reference`: a node with leading prose survives the round trip with its leading-prose key preserved and its bytes stable. (research.md D5 — this pins the observable key; it cannot catch a parse/render mismatch, because both sides call the same function)
- [X] T034 [US2] Update the GoDoc on `textPredicateFor`/`TextPredicateFor` in [internal/core/markdown.go](../../internal/core/markdown.go#L367-L382) to name the five core types, and record in `revertLeadingKey`'s GoDoc that the duplicate is kept in sync for core types only, with the three domain entries a known divergence tracked as research.md follow-up 1

**Checkpoint**: T011, T012, T019, T032, T033 green. No node stores prose under a predicate its own type forbids.

---

## Phase 5: User Story 3 — Type-named folders (Priority: P3)

**Goal**: Every type folder's name equals its type name, character for character. Folder → `@type` is string equality.

**Independent Test**: `arc init` creates exactly the eight folders of contract C3; applying a patch with all five core types plus an unknown domain type files each at `<TypeName>/<id>.md`.

> E2E tests T013–T016 were written in Phase 2d and MUST currently be failing.

### Production changes

- [X] T035 [US3] Delete the `coreKindFolders` map and rewrite `nodeFolder` as the identity function in [internal/app/graph/service/apply.go](../../internal/app/graph/service/apply.go#L32-L56) per data-model.md §2.1. Delete the `strings.HasSuffix`/`kind + "s"` fallback. **Preserve** the existing GoDoc documenting that `nodeFolder` is never called with `Timeline` — that invariant still holds and still needs stating
- [X] T036 [US3] Check whether `strings` is still used in [internal/app/graph/service/apply.go](../../internal/app/graph/service/apply.go) after T035 and remove the import if not — `staticcheck` will fail the build otherwise
- [X] T037 [P] [US3] Change the two path constants in [internal/app/schema/kernel/schema.go](../../internal/app/schema/kernel/schema.go#L18-L21) to `PredicatesDir = "_schema/Property"` and `TypesDir = "_schema/Class"`. Do **not** rename the Go identifiers (research.md D2). Update their GoDoc to state the new paths
- [X] T038 [P] [US3] Rewrite `DefaultLayout.Folders` in [internal/app/ctrl/kernel/graph.go](../../internal/app/ctrl/kernel/graph.go#L31-L42) to the eight entries of contract C3 — this turns T018 green
- [X] T039 [P] [US3] Add a GoDoc line to `pluralizeKind` in [cmd/arc/graph/apply.go](../../cmd/arc/graph/apply.go#L31) stating that it formats a human-readable count for the ingest summary and is unrelated to folder derivation (research.md D4). **Change no behaviour**
- [X] T040 [P] [US3] Update the folder references in the GoDoc/comments of [internal/core/rules.go](../../internal/core/rules.go#L12-L24), [internal/app/schema/component.go](../../internal/app/schema/component.go#L11), [internal/app/schema/service/schema.go](../../internal/app/schema/service/schema.go#L122-L216), [internal/app/schema/service/errors.go](../../internal/app/schema/service/errors.go#L33), [internal/app/schema/service/apply.go](../../internal/app/schema/service/apply.go#L32-L247), and [cmd/arc/lint/lint.go](../../cmd/arc/lint/lint.go#L43) — each names `_schema/predicates/` or `_schema/types/`
- [X] T041 [P] [US3] Update the user-facing violation message in [internal/app/lint/service/rules_predicates.go](../../internal/app/lint/service/rules_predicates.go#L84) — it prints the literal `"_schema/predicates/"`

### Unit tests

- [X] T042 [P] [US3] Add a unit test in [internal/app/graph/service/apply_test.go](../../internal/app/graph/service/apply_test.go) asserting contract C1: `nodeFolder(t) == t` for every core content type **and** for an unrecognized domain type (`Thought` → `Thought`, not `Thoughts`). Assert the returned **string**, never a filesystem stat
- [X] T043 [P] [US3] Add a unit test in [internal/app/graph/service/apply_test.go](../../internal/app/graph/service/apply_test.go) asserting `nodePath` composes to the exact string `<Type>/<id>.md`, and in [internal/app/graph/service/revert_internal_test.go](../../internal/app/graph/service/revert_internal_test.go) that `referrerPath` still diverts a `Timeline` referrer to `timeline/yearly|monthly/`

### Fixture migration

> Kept as its own tranche so the behavioural diff in T035–T043 stays reviewable. Pure mechanical substitution: `sources/`→`Source/`, `entities/`→`Entity/`, `resources/`→`Resource/`, `_schema/predicates/`→`_schema/Property/`, `_schema/types/`→`_schema/Class/`.

- [X] T044 [US3] Migrate fixture paths in the four highest-density test files: [cmd/arc/graph/apply_test.go](../../cmd/arc/graph/apply_test.go) (74 occurrences), [cmd/arc/lint/lint_test.go](../../cmd/arc/lint/lint_test.go) (52), [internal/app/graph/service/subgraph_test.go](../../internal/app/graph/service/subgraph_test.go) (37), [internal/app/graph/service/revert_test.go](../../internal/app/graph/service/revert_test.go) (35)
- [X] T045 [P] [US3] Migrate fixture paths in the `cmd/arc/graph` remainder: [subgraph_test.go](../../cmd/arc/graph/subgraph_test.go) (26), [batch_test.go](../../cmd/arc/graph/batch_test.go) (26), [revert_test.go](../../cmd/arc/graph/revert_test.go) (16), [serve_test.go](../../cmd/arc/graph/serve_test.go) (9), [grep_test.go](../../cmd/arc/graph/grep_test.go) (8)
- [X] T046 [P] [US3] Migrate fixture paths in `internal/app/graph/service`: [apply_test.go](../../internal/app/graph/service/apply_test.go) (22), [grep_test.go](../../internal/app/graph/service/grep_test.go) (14), [revert_internal_test.go](../../internal/app/graph/service/revert_internal_test.go) (3), [node_test.go](../../internal/app/graph/service/node_test.go) (3), [batch_internal_test.go](../../internal/app/graph/service/batch_internal_test.go) (3)
- [X] T047 [P] [US3] Migrate fixture paths in `internal/app/lint`: [rules_links_test.go](../../internal/app/lint/service/rules_links_test.go) (10), [rules_frontmatter_test.go](../../internal/app/lint/service/rules_frontmatter_test.go) (9), [rules_identity_test.go](../../internal/app/lint/service/rules_identity_test.go) (8), [lint_test.go](../../internal/app/lint/service/lint_test.go) (7), [rules_history_test.go](../../internal/app/lint/service/rules_history_test.go) (5), [rules_types_case_test.go](../../internal/app/lint/service/rules_types_case_test.go) (4), [rules_type_conformance_test.go](../../internal/app/lint/service/rules_type_conformance_test.go) (4), [kernel/lint_test.go](../../internal/app/lint/kernel/lint_test.go) (4)
- [X] T048 [P] [US3] Migrate fixture paths in the remainder: [cmd/arc/ctrl/init_test.go](../../cmd/arc/ctrl/init_test.go) (10), [internal/app/ctrl/service/init_test.go](../../internal/app/ctrl/service/init_test.go) (6), [internal/app/schema/service/schema_test.go](../../internal/app/schema/service/schema_test.go) (4), [internal/app/ctrl/kernel/graph_test.go](../../internal/app/ctrl/kernel/graph_test.go) (4), [internal/adapter/fsys/local_test.go](../../internal/adapter/fsys/local_test.go) (4), [internal/pkg/grep/grep_test.go](../../internal/pkg/grep/grep_test.go) (2)

**Checkpoint**: T013–T016, T018, T042–T043 green. `go test ./...` fully green.

---

## Phase 6: Polish & Cross-Cutting

- [X] T049 Run the completion grep from [quickstart.md](quickstart.md): `git grep -nE '"(sources|entities|resources|references)"|_schema/(types|predicates)' -- '*.go'` — must return nothing. Any hit in `internal/` or `cmd/` is a missed rename
- [X] T050 [P] Walk every scenario in [quickstart.md](quickstart.md) against a real binary, including Scenario 8 (a pre-feature graph must fail closed with `ErrSchemaMissing` naming `_schema/Property`, and leave the graph byte-identical)
- [X] T051 [P] Update the package-tree comment in [ARCHITECTURE.md](../../ARCHITECTURE.md) (lines ~121-122) and the **Canonical Folder** (~232), **Predicate Schema Node** (~234) and **Type Schema Node** (~235) Glossary entries — all four name the retired folder set
- [X] T052 [P] Add a `specs/CHANGELOG.md` entry recording the breaking change: folder rename, `Reference` added, `Resource` redefined, no migration path
- [X] T053 Re-run `staticcheck` and `go test ./... -cover`; confirm clean and that coverage has not regressed against the T001 baseline

---

## Phase N: Constitution Compliance Verification

**Purpose**: The constitution's Compliance Checklist (Implementation Phase). Retained verbatim per Governance > Task List Requirements.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — T003–T005, T051
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — `Reference`, Type Folder, Functional Folder
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX) — here, verifiably **unchanged**

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — expected N/A: no new pattern; the declined `arc migrate` registry would have needed one
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — no `cmd/` logic added
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — Phase 2d and Phase 2.5 precede Phase 3
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI)
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI) — quickstart.md is a manual walkthrough, not a correctness gate
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `lipgloss` for any styling (Principle X) — unchanged; confirm the ingest summary still reads `+3 entities` (T039)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — N/A, no command changed
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII) — 20 scenarios across US1–US3
- [X] TN15 Release/versioning impact assessed (Principle XIV) — **MAJOR-equivalent**: the on-disk graph layout and the seeded type vocabulary both change incompatibly, and no migration is provided. Pre-v0.11 graphs stop loading


### Verification record

Each box above was checked by running something, not by inspection alone.

- **TN01/TN02** — `ARCHITECTURE.md`: new **Reference Node**, **Type Folder**, and **Functional
  Folder** glossary entries; **Entity/Resource Node**, **Canonical Folder**, **Predicate Schema
  Node**, **Type Schema Node**, **`Node`**, and **Text Predicate / Prose Field** rewritten; the
  package-tree comment renamed. `grep` confirms no retired folder literal survives in the file.
- **TN03/TN05/TN12** — `git diff f64863b -- cmd/ ':(exclude)cmd/**/*_test.go'` is **two comment
  hunks and nothing else**: `pluralizeKind`'s new GoDoc and one renamed path in a `lint.go`
  comment. No command, flag, help string, exit code, or `--json` field changed, and no logic
  was added to `cmd/`.
- **TN04** — no ADR added; no new architectural pattern was introduced. `nodeFolder` collapsed
  to the identity function (a net deletion), no package, port, or adapter was created, and
  `go.mod` is unchanged. The declined `arc migrate` registry is the decision that *would* have
  needed one.
- **TN06** — red-first held, with three recorded exceptions, all of them tests that assert
  behaviour this feature deliberately leaves alone and so cannot start red:
  `TestInitSeededGraphLintsClean`, `TestApplyKeepsTimelineNodesBucketedAndCreatesNoFlatFolder`,
  and `TestTraversalOverTypeNamedFoldersStillExcludesArcAndSchema`. T019 is the fourth and is
  documented at the Phase 2.5 checkpoint — it was verified red in the T030→T031 window, which
  is the drift it exists to catch.
- **TN07** — `git grep testify -- '*.go'` returns nothing; every new assertion is `it/v2`.
- **TN08** — no Bash was used as a correctness gate. `quickstart.md` was walked manually (T050)
  as a confirmation of the compiled tests, never in place of them.
- **TN09/TN11** — N/A, confirmed in the Phase 2 record in `plan.md` (T008, T017).
- **TN10** — verified against a real binary: `arc apply` prints `+3 entities, +1 Source` while
  the folder it wrote is `Entity/`. `pluralizeKind` and `nodeFolder` are demonstrably
  independent, which is research.md D4's hazard closed rather than merely noted.
- **TN13/TN14** — all 20 acceptance scenarios map 1:1 to a colocated, passing E2E test:

  | Scenario | Test | File |
  |---|---|---|
  | US1-1 | `TestInitSeedsResourceWithIngestedFragmentContract` | `cmd/arc/ctrl/init_test.go` |
  | US1-2 | `TestInitSeedsResourceWithoutExternalWorkPredicates` | `cmd/arc/ctrl/init_test.go` |
  | US1-3 | `TestInitSeedsReferenceType` | `cmd/arc/ctrl/init_test.go` |
  | US1-4 | `TestApplyAcceptsReferenceNodeWithoutUnknownTypeDiagnostic` | `cmd/arc/graph/apply_test.go` |
  | US1-5 | `TestInitSeededGraphLintsClean` | `cmd/arc/ctrl/init_test.go` |
  | US1-6 | `TestApplyReferenceNodeIsTypeConformant`, `TestLintReferenceCarriesExternalWorkPredicatesConformantly` | `cmd/arc/graph/apply_test.go`, `cmd/arc/lint/lint_test.go` |
  | US2-1 | `TestApplyStoresLeadingProseUnderTheTypesOwnPredicate` | `cmd/arc/graph/apply_test.go` |
  | US2-2 | `TestApplyStoresLeadingProseUnderTheTypesOwnPredicate` | `cmd/arc/graph/apply_test.go` |
  | US2-3 | `TestRevertReconstructsResourceAndReferenceProseFromTheirOwnKeys` | `cmd/arc/graph/revert_test.go` |
  | US2-4 | `TestApplyResourceNodeCarriesTextNotRelevance` | `cmd/arc/graph/apply_test.go` |
  | US2-5 | `TestApplyLeavesOtherTypesLeadingProseDerivationUnchanged` (Thought) | `cmd/arc/graph/apply_test.go` |
  | US2-6 | `TestApplyLeavesOtherTypesLeadingProseDerivationUnchanged` (Source/Entity) | `cmd/arc/graph/apply_test.go` |
  | US3-1 | `TestInitCreatesExactlyTheTypeNamedFolders` | `cmd/arc/ctrl/init_test.go` |
  | US3-2 | `TestApplyFilesCoreNodesInTypeNamedFolders` | `cmd/arc/graph/apply_test.go` |
  | US3-3 | `TestApplyFilesUnknownDomainTypeUnderItsVerbatimName` | `cmd/arc/graph/apply_test.go` |
  | US3-4 | `TestApplyKeepsTimelineNodesBucketedAndCreatesNoFlatFolder` | `cmd/arc/graph/apply_test.go` |
  | US3-5 | `TestApplySchemaWritesIntoTypeNamedSchemaFolders` | `cmd/arc/ctrl/apply_schema_test.go` |
  | US3-6 | `TestRevertLocatesAndRewritesNodesInTypeNamedFolders` | `cmd/arc/graph/revert_test.go` |
  | US3-7 | `TestTraversalOverTypeNamedFoldersStillExcludesArcAndSchema` | `cmd/arc/graph/subgraph_test.go` |
  | US3-8 | `TestSubgraphExportReappliesPathStablyIntoFreshGraph` | `cmd/arc/graph/subgraph_test.go` |

- **TN15** — release impact is **breaking**. The on-disk layout and the seeded vocabulary both
  change incompatibly and no migration is provided; a pre-v0.11 graph stops loading. Recorded
  in `specs/CHANGELOG.md`. No version file exists to edit (the release version comes from the
  git tag), so the action this leaves open is the tag itself: on a pre-1.0 line a breaking
  change is a **minor** bump — `0.1.x` → `0.2.0`, not `0.1.14`. Tagging is a release step and
  was deliberately not performed here.

---

## Dependencies & Execution Order

### Phase order

```
Phase 1 (Setup)
   └─> Phase 2 (Design Preconditions, incl. 2d red-phase E2E)
          └─> Phase 2.5 (Drift-guard invariants, red)
                 ├─> Phase 3 (US1, P1)  ─┐
                 ├─> Phase 4 (US2, P2)  ─┼─> Phase 6 (Polish) ─> Phase N
                 └─> Phase 5 (US3, P3)  ─┘
```

### Story dependencies

| Story | Depends on | Notes |
|---|---|---|
| **US1** — type vocabulary | Phase 2.5 | Independent of US2/US3 in production code |
| **US2** — prose predicate | Phase 2.5 | Independent in production code; T031 turns T019 green |
| **US3** — folder layout | Phase 2.5 | Independent in production code; T038 turns T018 green |

**The one real coupling — US1 ↔ US2 in the fixtures, not the code.** Redefining `Resource` (T020) makes `text` a required predicate, while its prose is still keyed `relevance` until T030 lands. Any existing lint/apply fixture carrying a `Resource` node therefore reports a type-conformance violation in the window between T020 and T030. Two consequences:

- T029 cannot fully go green on its own — schedule Phase 3 and Phase 4 back to back, or land T030/T031 immediately after T020.
- Do **not** "fix" a fixture in that window by adding a `text` predicate by hand. The correct fix is T030.

US3 has no such coupling: the folder rename is orthogonal to both the vocabulary and the prose key.

### Parallel opportunities

- **Phase 2d**: T009–T016 are eight independent test-writing tasks across five files. `apply_test.go` is touched by T010, T011, T014 — sequence those three or partition the file by test function.
- **Phase 2.5**: T018 and T019 are fully independent (different packages).
- **Phase 3**: T023 and T024 are independent description edits; T020–T022 touch the same two maps and should be sequential.
- **Phase 4**: T032 and T033 are independent (same file, different test functions — coordinate or sequence).
- **Phase 5**: T037, T038, T039, T040, T041 are five independent files. T045–T048 are four independent fixture tranches and are the single biggest parallel win in the feature.
- **Phase 6**: T050, T051, T052 are independent.

### Suggested MVP

**Phase 1 → 2 → 2.5 → 3 (US1)**, immediately followed by **Phase 4 (US2)** because of the fixture coupling above. That pair delivers the substance of the v0.11 role correction — `Reference` exists, `Resource` means what it should, and no ingest silently mis-keys its prose. US3 (the folder rename) is the largest diff but the smallest semantic change, and can land as a second increment.

---

## Task Summary

| Phase | Tasks | Count |
|---|---|---|
| 1 — Setup | T001–T002 | 2 |
| 2 — Design Preconditions | T003–T017 | 15 |
| 2.5 — Drift-guard invariants | T018–T019 | 2 |
| 3 — US1 (P1) type vocabulary | T020–T029 | 10 |
| 4 — US2 (P2) prose predicate | T030–T034 | 5 |
| 5 — US3 (P3) folder layout | T035–T048 | 14 |
| 6 — Polish | T049–T053 | 5 |
| N — Constitution compliance | TN01–TN15 | 15 |
| **Total** | | **68** |

**Per story**: US1 = 10 implementation + 2 E2E (T009–T010) = 12 · US2 = 5 + 2 (T011–T012) = 7 · US3 = 14 + 4 (T013–T016) = 18
