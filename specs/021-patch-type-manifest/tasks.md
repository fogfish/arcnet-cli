---

description: "Task list for feature 021 — Patch Manifest Identity (`@type: patch`)"
---

# Tasks: Patch Manifest Identity (`@type: patch`)

**Input**: Design documents from `/specs/021-patch-type-manifest/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [constitution.md](../../.specify/memory/constitution.md)

**Tests**: NOT optional. Per constitution Principles VI and VIII, all 12 `spec.md` acceptance scenarios map 1:1 to E2E tests, and every test is written before the implementation that turns it green.

**Organization**: Grouped by user story. The three stories are independently testable, but they share `internal/core/markdown.go` and several test files — `[P]` is applied only where the file sets are genuinely disjoint.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 / US2 / US3 — maps to the user story in [spec.md](./spec.md)

## Path Conventions

Per [plan.md § Project Structure](./plan.md#project-structure). Production change is confined to `internal/core/` plus two call sites in `internal/app/graph/service/`. No `cmd/` production file changes — `cmd/` appears only as E2E test and fixture surface.

## The red phase for this feature

This feature modifies existing, currently-green code. The red phase is produced in **Phase 2.5** by migrating every fixture and inline test constant from `kind: patch` to `"@type": patch`. That migration makes the existing suite fail for exactly the right semantic reason — *the parser does not recognize `@type` yet* — which is the constitution's definition of starting red. Do not implement anything in `internal/core/markdown.go` before Phase 2.5 is complete and the suite is confirmed red.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish and record the pre-change baseline. No new package, module dependency, or tooling is introduced by this feature.

- [X] T001 Confirm a clean baseline: `go build ./... && go test ./... -cover` is fully green on `021-patch-type-manifest` before any edit; record the pass count
- [X] T002 [P] Capture the FR-011 golden: run `arc subgraph <seed> --json` against a fixture graph and save the exact bytes to `/tmp/subgraph-baseline.json` — T038 diffs against this to prove the `--json` contract is untouched
- [X] T003 [P] Confirm `staticcheck ./...` runs clean at baseline, so any finding later in this feature is attributable to it

---

## Phase 2: Design Preconditions

**Purpose**: The constitution's PRECONDITIONS from the Compliance Checklist. Each subsection is a design gate — the deliverable is a recorded decision, not working code.

**⚠️ CRITICAL**: No implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Update the **Patch** glossary row in [ARCHITECTURE.md](../../ARCHITECTURE.md) (line ~244) to name `"@type": patch` as the manifest's identity declaration alongside `document`/`published`/`title`/`stats`
- [X] T005 Update the **Batch Plan** glossary row in [ARCHITECTURE.md](../../ARCHITECTURE.md) (line ~273) — it currently reads "candidates that declare `kind: patch` but fail to parse"; restate in terms of the recognition rule in [contracts/patch-manifest.md §2](./contracts/patch-manifest.md) and note that a retired-key file is a failed candidate, never a passed-over file (FR-008)
- [X] T006 Confirm no new domain type is introduced: `core.Patch` gains no field and no type is added anywhere (research [D2](./research.md)). Record the confirmation — this is what makes FR-011 true by construction

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T007 Confirm and record that **no** command, subcommand, flag, alias, `Args`, or exit-code surface changes in this feature; `cmd/arc/root.go` and every `cmd/**/*.go` production file are untouched
- [X] T008 [P] Verify [contracts/patch-manifest.md](./contracts/patch-manifest.md) and [contracts/cli-contract.md](./contracts/cli-contract.md) are complete against FR-001..FR-016 — both were authored in Phase 1 planning; this is the sign-off gate

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T009 **N/A — record and move on.** No external system is integrated and no adapter is added or modified. All filesystem I/O continues through the existing `internal/adapter/fsys`; no `os.*` file call is introduced anywhere

### Phase 2d: E2E & Unit Acceptance Test Design (Principles VI, VIII)

Write these tests now. They MUST compile and MUST fail semantically. No `t.Skip()`, no "red phase" comments, no placeholder assertions. Failure assertions use `errors.Is` against the `internal/core` constants (declared in T018), never string matching.

- [X] T010 [P] [US1] Write the five-case recognition table test in `internal/core/markdown_test.go` covering every row of [data-model.md §2](./data-model.md) — `@type` only / `kind` only / both agreeing / both conflicting / neither — plus the `"@type": Patch` casing case and the `kind: source` alone case. The table is total, so this is the complete unit proof of recognition
- [X] T011 [P] [US1] Write the bare-unquoted-`@type` test in `internal/core/markdown_test.go` asserting `ErrIdentityQuoting`, plus an assertion that its message text matches `lint`'s `RuleIdentityQuoting` wording (research [D3](./research.md))
- [X] T012 [P] [US1] Write E2E tests in `cmd/arc/graph/apply_test.go` for US1 scenarios 1 and 2 (apply succeeds with one ingest commit; re-apply is idempotent with no second commit) via `sut()`
- [X] T013 [P] [US1] Write E2E tests in `cmd/arc/graph/batch_test.go` for US1 scenario 3 (a directory of `"@type": patch` files applies in date-then-path order) via `sut()`
- [X] T014 [P] [US1] Write E2E tests in `cmd/arc/ctrl/apply_schema_test.go` for US1 scenario 4 (schema patch recognized from local file, URL, and `arcnet:<name>`) via `sut()`
- [X] T015 [P] [US2] Write E2E tests in `cmd/arc/graph/subgraph_test.go` for US2 scenarios 1–3: emitted manifest's first key is `"@type": patch` with no `kind` anywhere; the emitted patch re-applies to a fresh graph (round-trip closure); `--json` output is byte-identical to the T002 golden
- [X] T016 [P] [US3] Write E2E tests in `cmd/arc/graph/apply_test.go` for US3 scenarios 1, 2, 3 and 5 (retired key rejected naming file + replacement; conflicting keys rejected naming both values; both keys agreeing applies; neither key produces the pre-existing message verbatim), each asserting exit non-zero, the message, and that the commit count is unchanged (FR-007)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T017 **N/A — record and move on.** No configuration value, environment variable, or secret is introduced or read by this feature

**Checkpoint**: Phase 2 complete — the new tests compile and fail, and the design is signed off

---

## Phase 2.5: Foundational — error constants and the fixture migration

**Purpose**: Shared foundation all three stories depend on. T018 is the minimal structure that lets Phase 2d's tests compile (explicitly permitted by Principle VI). T019–T026 produce the red phase described above.

**Migration rule**: only executable and normative occurrences migrate; dated records are left alone. See research [D7](./research.md) for the full classification.

- [X] T018 Declare the four `faults` constants in `internal/core/errors.go` per [data-model.md §4](./data-model.md): `ErrManifestTypeConflict` (`faults.Safe2[string, string]`), `ErrManifestLegacyKind` (`faults.Type`), `ErrManifestNotAPatch` (`faults.Safe1[string]`), `ErrIdentityQuoting` (`faults.Safe1[string]`). Leave `ErrManifestInvalid`'s text untouched (FR-006). Add the license header if the file lacks it
- [X] T019 [P] Migrate the 9 patch fixtures under `cmd/arc/graph/testdata/batch/` and `cmd/arc/graph/testdata/batch-failfast/` from `kind: patch` to `"@type": patch` — `batch-failfast/{a-early,b-blocker,c-late}.patch.md`, `batch/.hidden/ignored.patch.md`, `batch/2024/tls13.patch.md`, `batch/2026/{karpathy,pqkex}.patch.md`, `batch/broken/truncated.patch.md`, `batch/nested/deep/legacy.patch.md`. Note `nested/deep/legacy.patch.md` is named for its *subject matter*, not its manifest key — it migrates like the rest (research [D8](./research.md))
- [X] T020 [P] Update the explanatory prose inside `cmd/arc/graph/testdata/batch/notes.md`, which currently describes itself as "front matter that does *not* declare `kind: patch`". It must remain a non-patch that is passed over; only the stated reason changes
- [X] T021 [P] Migrate the 27 inline patch constants in `internal/core/markdown_test.go` to `"@type": patch`
- [X] T022 [P] Migrate the 33 inline patch constants in `cmd/arc/graph/apply_test.go` to `"@type": patch`
- [X] T023 [P] Migrate the 9 inline patch constants in `cmd/arc/ctrl/apply_schema_test.go` to `"@type": patch`
- [X] T024 [P] Migrate the 8 inline patch constants in `internal/app/graph/service/apply_test.go` and the 3 in `internal/app/graph/service/batch_internal_test.go` to `"@type": patch`
- [X] T025 [P] Migrate the 8 inline patch constants in `internal/app/schema/service/apply_test.go` to `"@type": patch`
- [X] T026 [P] Migrate the 6 inline patch constants in `cmd/arc/graph/batch_test.go` and the 5 in `cmd/arc/graph/revert_test.go` to `"@type": patch`
- [X] T027 Confirm the suite is now **red for the right reason**: run `go test ./...` and verify failures are manifest-recognition failures (`ErrManifestInvalid` on `"@type"`-keyed input), not compile errors and not unrelated breakage. This is the gate before any `markdown.go` edit

**Checkpoint**: Foundation ready — tests compile, suite is semantically red, implementation can begin

---

## Phase 3: User Story 1 — Apply a spec-conformant patch (Priority: P1) 🎯 MVP

**Goal**: `arc` recognizes and ingests patches declaring `"@type": patch`, across `arc apply`, `arc apply batch`, and `arc apply schema`. This is the import half of the interoperability break and the minimum viable outcome.

**Independent Test**: Transcribe a patch by hand from ARCNET-CORE §14.2.2, run `arc apply` against a fresh graph, confirm the nodes land and one ingest commit is recorded — without touching the emitting side. [quickstart.md Scenario 1](./quickstart.md).

### Implementation for User Story 1

> E2E and unit tests T010–T014 are already written and failing.

- [X] T028 [US1] Add the unexported `patchManifestType(manifest map[string]any) error` to `internal/core/markdown.go`, implementing the five-row decision table from [data-model.md §2](./data-model.md) in order — conflict first, then accept, then not-a-patch, then legacy, then fall through to `ErrManifestInvalid`
- [X] T029 [US1] Rewrite `decodePatchManifest` (`internal/core/markdown.go:327`) to delegate identity to T028's helper, replacing the `manifest["kind"] != "patch"` gate. Keep the function under Principle IV's 25-line ceiling; `document`/`published`/`title`/`stats` decoding is unchanged
- [X] T030 [US1] Widen `LooksLikePatch` (`internal/core/markdown.go:95`) to return true for `manifest["@type"] == "patch"`. **Leave the `kind` branch for T041** — US3 owns retired-key detection. Update its doc comment, which currently says `"kind: patch"`
- [X] T031 [US1] Add the bare-identity-key detector to `internal/core` and wire it into `ParsePatch`'s `parseDocument` error branch: when the raw front matter carries an unquoted `@id`/`@type`, return `ErrIdentityQuoting` instead of wrapping the raw yaml error in `ErrManifestInvalid` (research [D1](./research.md), [D3](./research.md)). Keep it a private boolean detector — do not export a lint-shaped locator API
- [X] T032 [US1] Wrap `core.ParsePatch`'s error with the file path in `internal/app/graph/service.readPatch` (`internal/app/graph/service/apply.go:485`) as `ErrPatchRead.With(err, patchPath)`, matching the mount/open branches above it and `schema/service.readPatchSource`'s existing convention (research [D5](./research.md))
- [X] T033 [US1] **Hazard from T032**: widen existing E2E assertions that match the bare parse-error text in `cmd/arc/graph/apply_test.go` and `internal/app/graph/service/apply_test.go` — every `arc apply` parse failure now carries a `failed to read patch file <path>: ` prefix, including pre-existing ones such as spec 019's `ErrTypeCasing`. `errors.Is` assertions are unaffected; change only the message expectations, and change as few as possible (Principle VIII)
- [X] T034 [US1] Update the `kind: patch` references in production doc comments: `internal/core/markdown.go` (3 sites) and `internal/app/graph/service/apply.go` (2 sites, including `guardNoOldFormatNodes`'s comment about a patch file left in the graph tree)
- [X] T035 [US1] Run `go test ./internal/core/... ./cmd/arc/graph/... ./cmd/arc/ctrl/...` and confirm T010–T014 are green

**Checkpoint**: US1's tests pass. `arc` can ingest conformant patches from any tool — the MVP is deliverable here.

---

## Phase 4: User Story 2 — Emit patches other tools can read (Priority: P2)

**Goal**: Every patch `arc` writes declares `"@type": patch`, closing the export half of the break and making the round trip work.

**Independent Test**: Run `arc subgraph`, inspect the emitted manifest, confirm `"@type": patch` is the first key and no `kind` appears. [quickstart.md Scenario 2](./quickstart.md).

### Implementation for User Story 2

> E2E test T015 is already written and failing.

- [X] T036 [US2] Change `renderPatchManifest` (`internal/core/markdown.go:1354`) to emit the identity key via `appendQuotedKeyYAMLPair(root, "@type", "patch")` — the same helper `renderAttrYAML` already uses for node `@id`/`@type` — replacing `appendYAMLPair(root, "kind", "patch")`. Field order, date format, `title`/`stats` omission, and flow styling are unchanged
- [X] T037 [US2] Update `RenderPatch`'s doc comment in `internal/core/markdown.go`, which currently documents the manifest as "(kind: patch, document, published, title, stats)"
- [X] T038 [US2] Extend the existing `RenderPatch` ↔ `ParsePatch` round-trip tests in `internal/core/markdown_test.go` (`TestRenderPatchRoundTrips*`) to assert the manifest identity survives the trip, and add an assertion that rendered output contains no `kind` key anywhere (FR-002, FR-010, SC-003). Same file as T021 — sequential with it, not parallel
- [X] T039 [US2] Verify FR-011 by diffing fresh `arc subgraph --json` output against the T002 golden; they must be byte-identical. `core.Patch` gained no field, so any difference is a defect
- [X] T040 [US2] Run `go test ./internal/core/... ./cmd/arc/graph/...` and confirm T015 is green, including the round-trip closure assertion

**Checkpoint**: US1 and US2 both pass independently. The tool is self-consistent — what it writes, it can read, and so can any conformant tool.

---

## Phase 5: User Story 3 — Unambiguous rejection of non-conformant manifests (Priority: P3)

**Goal**: Non-conformant manifests are refused with a message that names the file, the offending key, and the replacement — and are never silently skipped in a batch run.

**Independent Test**: Feed `arc apply` a `kind: patch` file and a self-contradictory file; confirm each is refused with an actionable message and the graph is unmodified. [quickstart.md Scenarios 4, 5, 7](./quickstart.md).

### Implementation for User Story 3

> E2E test T016 is already written and failing. T028's decision table already produces the correct *errors*; this story makes them reach the user attached to the right file, through the right command.

- [X] T041 [US3] Add the retired-key branch to `LooksLikePatch` in `internal/core/markdown.go`: also return true when `manifest["kind"] == "patch"`. Document in the doc comment that this is **error routing, not acceptance** — `decodePatchManifest` remains the sole gate, and every such file is still rejected (data-model [§3](./data-model.md))
- [X] T042 [P] [US3] Add `cmd/arc/graph/testdata/batch/legacy-kind.patch.md` carrying `kind: patch` and no `@type` (research [D8](./research.md)) — the only fixture that can demonstrate FR-008/SC-005
- [X] T043 [US3] Write the E2E test in `cmd/arc/graph/batch_test.go` for US3 scenario 4: `legacy-kind.patch.md` is reported **by name** under `failed`, and `not_a_patch` is unchanged. The count assertion *is* the SC-005 test. Depends on T042
- [X] T044 [US3] Update the batch expected counts in `cmd/arc/graph/batch_test.go` and `internal/app/graph/service/batch_internal_test.go` for the one added candidate — `failed` +1, `notAPatch` unchanged
- [X] T045 [US3] Update `classifyPatchFiles`'s doc comment in `internal/app/graph/service/batch.go` (2 `kind: patch` sites). No logic change is needed — it already routes on `LooksLikePatch`, which T041 widened; confirm this by test rather than by edit
- [X] T046 [US3] Change `isPatchDocument` in `internal/app/graph/service/revert.go:299` from "`ParsePatch` succeeds" to `core.LooksLikePatch(raw)`, satisfying FR-009's single-recognition-rule requirement (research [D6](./research.md)). **Flag for review**: this changes `arc revert` behaviour for a malformed patch left in the graph tree — it is now skipped rather than processed as a node. Skipping is the safe direction for a destructive command (Principle IX), but it is the one behaviour change in this feature unrelated to the manifest key
- [X] T047 [US3] Add the E2E regression test in `cmd/arc/graph/revert_test.go` covering T046: a patch file left in the graph tree is skipped by `arc revert`, in both `"@type"` and retired-key forms
- [X] T048 [US3] Add the E2E test in `internal/app/graph/service/apply_test.go` for the edge case where a retired-key patch sits inside the graph tree: `guardNoOldFormatNodes` must surface the actionable `ErrManifestLegacyKind` (traceable to the file) rather than `ParseNode`'s generic old-format heuristic
- [X] T049 [US3] Add the E2E test in `cmd/arc/graph/apply_test.go` for the edge case where a `"@type": patch` document has a spec-019 CamelCase violation in its body: the *body* error must remain the reported reason, not the identity check
- [X] T050 [US3] Run `go test ./...` and confirm the full suite is green

**Checkpoint**: All three stories pass independently. Every rejection path is named, actionable, and non-silent.

---

## Additional Polish

**Purpose**: The documentation half of FR-014, plus the release record. Per research [D7](./research.md), dated records are deliberately **not** migrated: past `spec.md`, `research.md`, `data-model.md`, `tasks.md`, `bugs/*.md`, `specs/CHANGELOG.md`, `CORE-FIX.md`, and this feature's own `spec.md`.

- [X] T051 [P] Migrate `kind: patch` to `"@type": patch` in the runnable quickstart guides: `specs/{003-apply-patch,005-graph-schema-first-class,007-arc-subgraph,009-node-timestamp-attrs,010-predicate-node-model,011-machine-readable-schema,013-predicate-role-rendering,018-apply-schema-patch,019-camelcase-node-types,020-apply-batch}/quickstart.md` — **10, not 9**: `007-arc-subgraph/quickstart.md` carries 4 occurrences the task list omitted, and it is inside SC-003's own grep scope. These are executable validation guides — leaving them on the retired key ships guides that fail on execution
- [X] T052 [P] Migrate the two normative contracts: `specs/003-apply-patch/contracts/cli-contract.md` and `specs/007-arc-subgraph/contracts/cli-contract.md`, adding a pointer to [contracts/patch-manifest.md](./contracts/patch-manifest.md) as the superseding grammar
- [X] T053 [P] Update the `arc apply` line in `specs/VISION.md`, which describes the manifest as `kind: patch`
- [X] T054 Add the `specs/CHANGELOG.md` entry recording the breaking format change: `@type: patch` is now the sole recognized and emitted manifest identity; `kind: patch` is refused with a migration message; no transitional acceptance; `arc lint` and `--json` unaffected
- [X] T055 Run the SC-003 repository check from [quickstart.md](./quickstart.md): `grep -rn 'kind: patch'` across `cmd/`, `internal/`, `ARCHITECTURE.md`, `specs/VISION.md`, `specs/*/quickstart.md`, `specs/*/contracts/` returns **zero** matches
- [X] T056 Walk all 10 [quickstart.md](./quickstart.md) scenarios end to end against a real built binary and confirm every stated expected outcome
- [X] T057 Run `staticcheck ./...` and `go test ./... -cover`; confirm clean, and that coverage has not regressed against the T001 baseline

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
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI)
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI)
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII)
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI)
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII)
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test (Principle VIII)
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV)

### Feature-specific notes for Phase N

- **TN01/TN02** are satisfied by T004/T005.
- **TN03** is satisfied by T007 — the surface is provably unchanged.
- **TN04**: no new ADR. This feature implements an external specification (ARCNET-CORE §14.2.1); it decides no architecture. ADRs 001/002/003 were reviewed against the plan and none is contradicted.
- **TN09/TN10/TN11/TN12** are N/A — no adapter, no output styling change, no configuration, no command help text change.
- **TN15** is the one live gate: the answer is **yes, this is a breaking format change**, shipping without the Principle XIV deprecation window and without a major bump. Both deviations are justified in [plan.md § Complexity Tracking](./plan.md#complexity-tracking) and must be confirmed still-valid at merge. `--json` and `--plain` — the two contracts the constitution names as stable — are provably unaffected (T039).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — T002's golden must be captured *before* any edit
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS everything; 2a–2e proceed in parallel
- **Foundational (Phase 2.5)**: Depends on Phase 2. T018 must precede T027; T019–T026 are mutually parallel
- **User Stories (Phase 3–5)**: All depend on Phase 2.5's red gate (T027)
- **Polish**: Depends on US1–US3 complete
- **Phase N**: Final gate

### User Story Dependencies

- **US1 (P1)**: Depends only on Phase 2.5. Delivers the MVP alone
- **US2 (P2)**: Depends only on Phase 2.5 for correctness, but its round-trip closure assertion (T015 scenario 2) needs US1's parser to be green. **Sequence US1 before US2** in a single-developer run
- **US3 (P3)**: Depends only on Phase 2.5. T028's decision table (US1) already produces the errors; if US3 ran first, its rejection tests would pass but its batch test would not

### Within Each Story

- Decision table (T028) before its consumers (T029, T030)
- `LooksLikePatch`'s `@type` branch (T030, US1) before its `kind` branch (T041, US3)
- Fixture (T042) before the test that uses it (T043)

### File-sharing constraints (why `[P]` is absent in places)

- `internal/core/markdown.go` — T028–T031 (US1), T036–T037 (US2), T041 (US3) all edit it. Never parallel
- `internal/core/markdown_test.go` — T010, T011 (Phase 2d), T021 (migration), T038 (US2). Sequential
- `cmd/arc/graph/apply_test.go` — T012, T016 (Phase 2d), T022 (migration), T033, T049. Sequential
- `cmd/arc/graph/batch_test.go` — T013, T026, T043, T044. Sequential

### Parallel Opportunities

- T002, T003 in Setup
- T008 within Phase 2b; T010–T016 across Phase 2d (disjoint files)
- **T019–T026**: the whole fixture migration, eight mutually independent file sets — the largest parallel block in the feature
- T042 alongside T041
- T051, T052, T053 in Polish

---

## Parallel Example: the fixture migration (Phase 2.5)

```bash
# After T018 declares the error constants, launch all eight migrations together:
Task: "Migrate 9 patch fixtures under cmd/arc/graph/testdata/{batch,batch-failfast}/"
Task: "Update explanatory prose in cmd/arc/graph/testdata/batch/notes.md"
Task: "Migrate 27 inline constants in internal/core/markdown_test.go"
Task: "Migrate 33 inline constants in cmd/arc/graph/apply_test.go"
Task: "Migrate 9 inline constants in cmd/arc/ctrl/apply_schema_test.go"
Task: "Migrate 8+3 inline constants in internal/app/graph/service/{apply_test.go,batch_internal_test.go}"
Task: "Migrate 8 inline constants in internal/app/schema/service/apply_test.go"
Task: "Migrate 6+5 inline constants in cmd/arc/graph/{batch_test.go,revert_test.go}"
# Then T027 confirms the suite is red for the right reason.
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup — capture the `--json` golden first
2. Phase 2: Design Preconditions (CRITICAL — blocks everything)
3. Phase 2.5: Error constants, then the fixture migration, then confirm red (T027)
4. Phase 3: User Story 1
5. **STOP and VALIDATE**: [quickstart.md Scenario 1](./quickstart.md) — `arc` ingests a hand-written ARCNET-CORE §14.2.2 patch, which is impossible today

US1 alone is a coherent, shippable increment: the tool can read the ecosystem's patches. It cannot yet write patches the ecosystem can read, so US2 should follow closely.

### Incremental Delivery

1. Setup + Preconditions + Foundational → red, ready
2. US1 → import works (MVP)
3. US2 → export works; the tool is self-consistent and round-trips
4. US3 → refusals are actionable and non-silent
5. Polish → documentation and CHANGELOG catch up with behaviour
6. Phase N → merge gate

### Parallel Team Strategy

The fixture migration (T019–T026) is the natural fan-out point and the bulk of the work by file count. With two developers: one takes `internal/core` (T021, then US1's parser work), the other takes the `cmd/` fixture and test surface (T022–T026), converging at T027.

---

## Notes

- **Total: 72 tasks** — T001–T057 plus the 15 verbatim Phase N gates (TN01–TN15) — across 3 user stories
- 99 inline test constants and 9 testdata fixtures migrate; 1 new fixture is added
- Failure assertions use `errors.Is` against `internal/core` constants — never string matching (Mandatory Libraries & Tooling)
- Every new `.go` file needs the license header from [CLAUDE.md](../../CLAUDE.md); every file touched that lacks it gets it
- The one decision worth challenging at review is **T046** (`arc revert`'s recognition rule) — it is flagged in the task itself and in research [D6](./research.md)
- Phase 2 and Phase N are retained per constitution Governance > Task List Requirements
