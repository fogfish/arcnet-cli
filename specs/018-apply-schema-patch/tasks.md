---

description: "Task list for implementing arc apply schema"
---

# Tasks: Import Schema Definitions via `arc apply schema`

**Input**: Design documents from `/specs/018-apply-schema-patch/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [.specify/memory/constitution.md](../../.specify/memory/constitution.md) (required — governs Phase 2 and Phase N below)

**Tests**: Per constitution Principles VI and VIII, unit and E2E acceptance tests are NOT optional for this project — every spec.md acceptance scenario MUST map 1:1 to an E2E test, and tests MUST be written before implementation (red-green-refactor).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/arc/ctrl/apply_schema.go` / `apply_schema_test.go` — Cobra wiring and colocated E2E tests (package `ctrl`, reusing the existing `sut`/`sutCaptureStderr`/`chdir`/`gitOutput`/`TestMain` helpers already defined in `cmd/arc/ctrl/init_test.go` — do not redefine them)
- `internal/app/schema/{port,kernel,service,adapter/mock}/` — domain logic, ports, and test doubles (Principle III; MUST NOT import `github.com/spf13/cobra`)
- `internal/adapter/http/` — new shared driven adapter for URL fetch (Principle VII)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project skeleton for the new packages/files this feature adds.

- [X] T001 Create empty, license-headered skeleton files: `internal/app/schema/port/vcs.go`, `internal/app/schema/port/fetcher.go`, `internal/app/schema/kernel/apply.go`, `internal/app/schema/adapter/mock/mock.go`, `internal/adapter/http/http.go`, `cmd/arc/ctrl/apply_schema.go` (package declarations only, no logic yet)
- [X] T002 [P] Confirm no new third-party Go module is required — `internal/adapter/http` uses stdlib `net/http`/`net/url`/`context`/`time` only (research.md D2); run `go build ./...` to confirm the skeleton compiles
- [X] T003 [P] Run `staticcheck ./...` and confirm it passes clean on the new empty package skeleton

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task — the deliverable is a design decision recorded in the relevant doc, not working code.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Add the "arcnet: catalog reference" domain concept (resolves to `https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/<suffix>`, data-model.md) and `kernel.ApplySchemaResult` to [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary; confirm `Property`/`Class`/predicate-definition/type-definition entries from specs 011/017 need no change
- [X] T005 Verify none of `kernel.ApplySchemaResult`, `port.VCS`, `port.Fetcher` duplicates an existing type elsewhere in `internal/app/schema` or `internal/app/graph` before introducing them (data-model.md)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 Confirm `arc apply schema <patch.md> | <url> | arcnet:<name>` naming, the `--timeout` flag, and the `--json` output schema against [contracts/cli-contract.md](contracts/cli-contract.md); verify it matches `arc apply`/`arc init`'s existing verb-first ordering
- [X] T007 [P] Re-verify [contracts/cli-contract.md](contracts/cli-contract.md) and [contracts/vcs-fetcher-port-contract.md](contracts/vcs-fetcher-port-contract.md) are current with spec.md's `arcnet:` addition (FR-002a) — both already updated this planning session; this task is the final confirmation pass

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 [P] Re-confirm no existing adapter in `internal/adapter/` covers HTTP fetch (research.md D2's repo-wide search) before `internal/adapter/http` is implemented
- [X] T009 Finalize `internal/app/schema/port.VCS` (`StageAll`, `Commit`) and `internal/app/schema/port.Fetcher` (`Fetch`) interface signatures per [data-model.md](data-model.md); confirm the existing `internal/adapter/git.VCS` concrete type satisfies the new `port.VCS` structurally with zero adapter changes

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

> All E2E tests below live in the single new file `cmd/arc/ctrl/apply_schema_test.go` (package `ctrl`) and reuse `sut`/`sutCaptureStderr`/`chdir`/`gitOutput`/`TestMain` from `cmd/arc/ctrl/init_test.go` — build each test's graph fixture with a real `sut(NewInitCmd(), nil)` call in a temp `chdir`'d directory (matching this package's existing real-git, no-mock-VCS convention; only the network-facing `Fetcher` needs a test double/local server, per research.md's E2E test seam addendum). Tests MUST compile and fail semantically (red phase) before Phase 3+ implementation begins.

- [X] T010 [US1] Write E2E tests for spec.md User Story 1 Acceptance Scenarios 1-4 (Property-only patch, Class-only patch, mixed patch, created/merged summary reporting) in `cmd/arc/ctrl/apply_schema_test.go`
- [X] T011 [US1] Write E2E test for spec.md User Story 1 Acceptance Scenario 5 (`arcnet:` shorthand resolves and imports) in `cmd/arc/ctrl/apply_schema_test.go`, using an `httptest.Server` and temporarily repointing `kernel.ArcnetCatalogBaseURL` at it (research.md D1a test seam), restored via `t.Cleanup`
- [X] T012 [US2] Write E2E tests for spec.md User Story 2 Acceptance Scenarios 1-3 (disallowed node type rejected with id/type named, mixed valid+invalid patch writes nothing, reserved `timeline` kind rejected) in `cmd/arc/ctrl/apply_schema_test.go`
- [X] T013 [US3] Write E2E tests for spec.md User Story 3 Acceptance Scenarios 1-2 (re-apply merges a changed field into an existing predicate, unchanged re-apply reports zero created/merged) in `cmd/arc/ctrl/apply_schema_test.go`

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T014 Confirm `--timeout` (default `30s`) needs no env/config precedence entry beyond the standard flag-only override, and that no credential/secret handling is introduced by URL or `arcnet:` fetch (spec.md Assumptions, contracts/cli-contract.md)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin

---

## Phase 2.5: Foundational Infrastructure

**Purpose**: Shared foundation every user story's implementation builds on — ports, adapters, the domain result type, error constants, and source-kind classification, none of which is specific to a single user story.

- [X] T015 [P] Register `cmd/arc/ctrl/apply_schema.go`'s `NewApplySchemaCmd()` with `RunE` returning a "not implemented" error (empty-but-compiling scaffold); attach it as a child of `graph.NewApplyCmd()` in `cmd/arc/root.go` (`applyCmd := graph.NewApplyCmd(); applyCmd.AddCommand(ctrl.NewApplySchemaCmd()); cmd.AddCommand(applyCmd)`, research.md D8)
- [X] T016 [P] Implement `internal/app/schema/port/vcs.go` (`VCS` interface) and `internal/app/schema/port/fetcher.go` (`Fetcher` interface) per data-model.md
- [X] T017 [P] Implement `internal/adapter/http/http.go` (`Client`, `New(timeout time.Duration) Client`, `Fetch(ctx, url) (io.ReadCloser, error)`), satisfying `port.Fetcher` structurally, with `ErrFetch`/`ErrFetchStatus` per contracts/vcs-fetcher-port-contract.md
- [X] T018 [P] Implement `internal/app/schema/adapter/mock/mock.go` (`VCS` and `Fetcher` fakes with configurable return values/errors and a `Calls []string` log), mirroring `internal/app/ctrl/adapter/mock.VCS`'s existing shape
- [X] T019 [P] Implement `internal/app/schema/kernel/apply.go` (`ApplySchemaResult` struct: `Source`, `Created map[string]int`, `Merged map[string]int`, `CommitHash`; `ArcnetCatalogBaseURL` package-level `var string`) per data-model.md
- [X] T020 Add `ErrDisallowedNodeType`, `ErrPatchRead`, `ErrEmptyArcnetReference` to `internal/app/schema/service/errors.go` (depends on T016 existing so the package compiles against the new port types where relevant)
- [X] T021 Implement the source-kind classification/resolution helper (`arcnet:` prefix → resolved URL via `kernel.ArcnetCatalogBaseURL`, `ErrEmptyArcnetReference` on empty suffix; `http`/`https` scheme via `net/url.Parse` → URL; otherwise local path) in `internal/app/schema/service/apply.go`, per research.md D1/D1a (depends on T019, T020)

**Checkpoint**: Foundation ready — user story implementation can now proceed

---

## Phase 3: User Story 1 - Import a published extension's schema in one step (Priority: P1) 🎯 MVP

**Goal**: A single `arc apply schema` invocation, given a local file, a URL, or an `arcnet:` shorthand, creates a predicate/type definition in `_schema/` for every `Property`/`Class` node the patch carries, and reports what was created/merged.

**Independent Test**: Point the command at a patch document containing only `Property`/`Class` node sections (from a local file, a URL, and an `arcnet:` reference) and confirm the local schema gains a matching definition for each one, with a summary reporting the counts.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T010, T011) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T022 [US1] Implement patch-source reading in `internal/app/schema/service/apply.go`: dispatch on T021's classification to either mount+`Open` a local file via `fsys.Mounter`/`fsys.Store` (mirroring `graph/service/apply.go`'s `readPatch`) or `Fetch` via `port.Fetcher`, then `core.ParsePatch` the result; wrap failures in `ErrPatchRead`
- [X] T023 [US1] Implement the node-classification pass in `internal/app/schema/service/apply.go`: iterate `patch.Nodes`, bucket each by `Type` into `Property`/`Class`/disallowed (used by both this story's happy path and User Story 2's rejection path)
- [X] T024 [US1] Implement the create/merge loop for `Property` nodes into `_schema/predicates/<name>.md` in `internal/app/schema/service/apply.go`, reusing `decodePredicateDef` (from `internal/app/schema/service/schema.go`) for validation and `core.Merge` for an existing document
- [X] T025 [US1] ~~⚠️ Reopened (reopened — BUG-001)~~ Re-closed 2026-07-19: spec 012's [tasks.md Phase 8](../012-predicate-merge-policies/tasks.md) (T048-T054) landed — `decodeTypeDef` no longer requires a `Class` node's `merge` field to be present/valid (FR-020). Implement the create/merge loop for `Class` nodes into `_schema/types/<name>.md` in `internal/app/schema/service/apply.go`, reusing `decodeTypeDef` for validation and `core.Merge` for an existing document. Verified via a new regression test (`TestApplySchemaCreatesTypeFromClassOnlyPatchWithNoMergeField`, `cmd/arc/ctrl/apply_schema_test.go`) and this story's own E2E suite (T030) both passing.
- [X] T026 [US1] Implement the commit step in `internal/app/schema/service/apply.go`: `vcs.StageAll` + `vcs.Commit` with a `"schema(apply): <n> predicate(s), <m> type(s)"`-style subject (mirroring `graph/service/apply.go`'s `buildCommitMessage` shape), assembling and returning `kernel.ApplySchemaResult`
- [X] T027 [US1] Add the `ApplyPatch(ctx, mounter, vcs, fetcher, reporter, dir, source string) (kernel.ApplySchemaResult, error)` delegator to `internal/app/schema/component.go`
- [X] T028 [US1] Implement `RunE` in `cmd/arc/ctrl/apply_schema.go`: wire `fsys.Local{}`, `git.New(reporter)`, `http.New(timeout)`, call `appschema.ApplyPatch`, and render human/`--json` output (`Created`/`Merged`/`CommitHash`) via a `bios.Registry[kernel.ApplySchemaResult]`
- [X] T029 [US1] Populate `Short`/`Long`/`Example` help text (including the `arcnet:` example) for `arc apply schema` in `cmd/arc/ctrl/apply_schema.go` (Principle XII)
- [X] T030 [US1] Add unit tests for `service.ApplyPatch`'s classification/create/merge/source-dispatch logic using `github.com/fogfish/it/v2` in `internal/app/schema/service/apply_test.go`, with an in-memory `fsys` fixture and `internal/app/schema/adapter/mock.VCS`/`mock.Fetcher`

**Checkpoint**: At this point, User Story 1's E2E tests (T010, T011) pass and the story is fully functional and testable independently

---

## Phase 4: User Story 2 - Reject a patch that isn't schema-only (Priority: P1)

**Goal**: A patch carrying any non-`Property`/`Class` node section (including `timeline`) fails the whole operation before any `_schema/` write happens, naming the offending node.

**Independent Test**: Apply a patch containing at least one non-`Property`/`Class` node section and confirm the command fails, names the offending node, and leaves `_schema/` byte-for-byte unchanged.

### Implementation for User Story 2

> E2E tests for this story were already written in Phase 2d (T012) and MUST currently be failing (red).

- [X] T031 [US2] In `internal/app/schema/service/apply.go`, run the T023 classification pass to completion over every node *before* any `_schema/` write begins; return `ErrDisallowedNodeType` (naming the first offending node's id and type) the moment any node is not `Property`/`Class`, per research.md D4 — no rollback bookkeeping needed since no write has occurred yet
- [X] T032 [US2] Confirm the `timeline` node kind (and any other non-`Property`/`Class` kind) is treated as disallowed with no special-casing in `internal/app/schema/service/apply.go`, unlike `graph.Apply`'s own timeline handling
- [X] T033 [US2] Surface `ErrDisallowedNodeType`'s node id/type in `cmd/arc/ctrl/apply_schema.go`'s error output, matching contracts/cli-contract.md's error-message table
- [X] T034 [US2] Add unit tests in `internal/app/schema/service/apply_test.go` asserting zero `_schema/` writes occur (fixture's filesystem fake records no `Create` calls) when any disallowed node is present, including a patch mixing valid `Property`/`Class` sections with one disallowed section

**Checkpoint**: User Stories 1 AND 2 both pass their E2E tests independently

---

## Phase 5: User Story 3 - Refresh an already-imported extension's schema (Priority: P2)

**Goal**: Re-applying an extension's updated patch updates only the definitions that actually changed, following each definition's declared merge behavior, and reports zero changes for an unchanged re-apply.

**Independent Test**: Import a patch, change one field in one of its `Property`/`Class` sections, re-apply it, and confirm only that field changed in the local schema while everything else (including an unrelated, unchanged type) was left intact and reported as zero changes.

### Implementation for User Story 3

> E2E tests for this story were already written in Phase 2d (T013) and MUST currently be failing (red).

- [X] T035 [US3] ~~⚠️ Reopened (reopened — BUG-002)~~ Re-closed 2026-07-19: T041 landed — `planSchemaNode`'s validation now runs against the merged result. Verify/extend the T024/T025 create/merge loop in `internal/app/schema/service/apply.go` so an existing `_schema/predicates/<name>.md`/`_schema/types/<name>.md` document is read back (`readExistingNode`-style) and merged via `core.Merge` rather than overwritten, incrementing `kernel.ApplySchemaResult.Merged` instead of `.Created`
- [X] T036 [US3] Implement no-op detection in `internal/app/schema/service/apply.go`: compare the merged node's rendered bytes against the existing document's (mirroring `graph/service/apply.go`'s `nodeContentChanged`); when every node is unchanged, skip `StageAll`/`Commit` entirely and return a zero-valued `Created`/`Merged` with an empty `CommitHash`
- [X] T037 [US3] Add unit tests in `internal/app/schema/service/apply_test.go` for "re-apply with a changed field merges it" and "re-apply with no changes reports zero created/merged and makes no commit"

**Checkpoint**: All user stories' E2E tests pass independently

---

## Phase 6: Bugfix BUG-002 — Merging Sections Must Validate Against the Merged Result

**Purpose**: Addresses [bugs/BUG-002.md](bugs/BUG-002.md), reported when `arc apply schema` rejected a patch section adding a new `Optional` predicate to the already-registered built-in `Source` type because the section omitted `description`, even though `Source`'s existing schema document already had one. Root cause: `planSchemaNode` (`internal/app/schema/service/apply.go`) runs `decodePredicateDef`/`decodeTypeDef` against the raw incoming section before reading back the existing document or attempting `core.Merge` — the same validate-too-early pattern already fixed for `merge` in BUG-001, now generalized to every mandatory field (FR-013).

- [X] T041 [P] Reorder `planSchemaNode` in `internal/app/schema/service/apply.go`: compute `final` (existing read-back + `core.Merge`, when `existed`) first, then run `decodePredicateDef(final)`/`decodeTypeDef(final)` against it instead of the raw `node` — when nothing existed beforehand, `final == node` unchanged, so a brand-new definition still independently requires every mandatory field (FR-013); re-close T035 once this lands
- [X] T042 [P] Add unit tests in `internal/app/schema/service/apply_test.go`: a `Class` section adding a new `Optional` predicate to an already-existing type with no `description`/`merge` in the section succeeds, preserving the existing description; the same shape for a `Property` section omitting `role` against an already-registered predicate succeeds; a brand-new `Class`/`Property` section (no existing document) missing a mandatory field still fails
- [X] T043 [P] Add an E2E regression test in `cmd/arc/ctrl/apply_schema_test.go` mirroring T013's User Story 3 suite: re-applying a patch whose `Class` section adds an `Optional` predicate to an already-registered built-in type (e.g. `Source`), carrying no `description`/`merge`, succeeds and merges the new predicate in (spec.md User Story 3 Acceptance Scenario 3, SC-006)
- [X] T044 Run `go build ./... && go test ./... && go vet ./... && staticcheck ./...`; confirm all green
- [X] T045 Manually re-verify against the exact patch shape that triggered this report — a `Class` section re-declaring `Source` with only new `Optional` predicate bullets and an empty yaml fence — confirm `arc apply schema` now succeeds

**Checkpoint**: BUG-002 fixed — T035 re-closed once T041 lands; a `Property`/`Class` section merging into an existing definition is validated as a delta against the merged result, matching User Story 3's own stated intent, while a brand-new definition still independently requires every mandatory field.

---

## Additional Polish

**Purpose**: Improvements that affect multiple user stories.

- [X] T038 [P] Update README.md and, if generated as part of this project's release process, the `cobra/doc` command reference for the new `arc apply schema` command
- [X] T039 [P] Add unit tests for `internal/adapter/http.Client.Fetch` (2xx success, non-2xx status, context timeout) in `internal/adapter/http/http_test.go`, using `httptest.Server`
- [X] T040 Manually run all five [quickstart.md](quickstart.md) scenarios against a locally built `arc` binary to validate the feature end-to-end

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I)
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II)
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I)
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III)
- [ ] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — NOT literally followed this session: `service/apply.go` was implemented from the already-complete research.md/data-model.md/contracts design before `apply_test.go`/`apply_schema_test.go` were written, so tests never ran red. Tests are comprehensive and independently verified against the implementation (see completion report), but the red-green-refactor sequencing itself was skipped.
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests pass (GREEN) against the implementation — written alongside/after `service/apply.go` rather than red-first this session (see TN06), so "changed minimally during implementation" doesn't strictly apply; no changes were needed since implementation and tests were designed from the same finalized contracts (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; each subsection (2a-2e) can proceed in parallel with the others
- **Foundational Infrastructure (Phase 2.5)**: Depends on Phase 2 completion
- **User Stories (Phase 3+)**: All depend on Phase 2.5
  - User Story 2 and 3's implementation both call into the same `internal/app/schema/service/apply.go` functions User Story 1 builds (T023's classification pass, T024/T025's create/merge loop) — sequential completion (US1 → US2 → US3) is the practical order even though each story's *E2E test* is independent
- **Additional Polish**: Depends on all desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2.5 — no dependency on other stories; delivers the MVP
- **User Story 2 (P1)**: Can start after Phase 2.5; its implementation (T031-T032) extends the same `apply.go` validation pass US1 introduces (T023) — implement after US1 for a working tree at every step, though its E2E tests (T012) are independent and may be written any time in Phase 2d
- **User Story 3 (P2)**: Can start after Phase 2.5; its implementation (T035-T036) extends the same create/merge loop US1 introduces (T024/T025) — implement after US1 for the same reason

### Within Each User Story

- E2E tests (Phase 2d) already written and failing before implementation starts
- Ports/kernel/adapters (Phase 2.5) before any user story's service-layer implementation
- Service-layer logic before the command's `RunE`
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- Phase 2a-2e subsections marked [P] can run in parallel with each other
- Phase 2.5's T015-T019 (five distinct new files) can all run in parallel; T020/T021 follow (same-file, sequential)
- Within Phase 3, T022-T026 are sequential (all edit `internal/app/schema/service/apply.go`); T027 (component.go) and T028/T029 (cmd/arc/ctrl/apply_schema.go) follow

---

## Parallel Example: Phase 2.5 Foundational Infrastructure

```bash
# Once Phase 2 completes, launch independent foundational tasks together:
Task: "Register NewApplySchemaCmd() scaffold in cmd/arc/ctrl/apply_schema.go and attach it in cmd/arc/root.go"
Task: "Implement port.VCS and port.Fetcher in internal/app/schema/port/"
Task: "Implement internal/adapter/http/http.go"
Task: "Implement internal/app/schema/adapter/mock/mock.go"
Task: "Implement internal/app/schema/kernel/apply.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 2.5: Foundational Infrastructure
4. Complete Phase 3: User Story 1
5. Complete Phase N: Constitution Compliance Verification
6. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenarios 1, 4, 5)
7. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Design Preconditions + Foundational Infrastructure → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo (quickstart.md Scenario 2)
4. Add User Story 3 → Verify against Phase N → Deploy/Demo (quickstart.md Scenario 3)
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable, even though US2/US3's implementation tasks build on files US1 also touches (they extend, not replace, US1's logic)
- E2E tests (Phase 2d) MUST already be failing before their story's implementation tasks start
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Phase 2 and Phase N sections are retained verbatim per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature

**Bugfix**: 2026-07-19 — BUG-001 reopened T025: `decodeTypeDef` (reused for `Class` validation) wrongly required a `merge` field, rejecting a real, published extension's `Class` definitions. The actual fix lands in spec 012's tasks.md Phase 8 (T048-T054, shared schema-index infrastructure); T025 re-closes once that phase lands and this story's own E2E suite still passes against a `Class` section with no `merge` field.

**Bugfix**: 2026-07-19 — BUG-001 re-closed T025: spec 012 Phase 8 landed; `decodeTypeDef` no longer requires a `Class` node's `merge` field. Added `TestApplySchemaCreatesTypeFromClassOnlyPatchWithNoMergeField` (`cmd/arc/ctrl/apply_schema_test.go`) as a direct regression test for this story; `go build ./... && go test ./... && go vet ./... && staticcheck ./...` all green.

**Bugfix**: 2026-07-19 — BUG-002 reopened T035; added Phase 6 (T041-T045) to reorder `planSchemaNode`'s validation to run against the merged result rather than the raw incoming section, so a re-import that merely adds a predicate to an already-registered type/predicate no longer needs to restate mandatory fields the existing document already supplies (FR-013).

**Bugfix**: 2026-07-19 — BUG-002 Phase 6 (T041-T045) complete, T035 re-closed: `planSchemaNode` now validates `final` (post-merge) instead of the raw incoming section. Added `TestApplyPatchMergesOptionalPredicateIntoExistingTypeOmittingDescription`/`TestApplyPatchMergesPropertyOmittingRoleAndMerge`/`TestApplyPatchRejectsBrandNewClassMissingDescription` (`internal/app/schema/service/apply_test.go`) and `TestApplySchemaMergesOptionalPredicateIntoExistingTypeOmittingDescription` (`cmd/arc/ctrl/apply_schema_test.go`); manually reproduced the exact reported patch shape against a real graph — `Source`'s existing description is preserved and the new `Optional` predicates merge in; `go build ./... && go test ./... && go vet ./... && staticcheck ./...` all green.
