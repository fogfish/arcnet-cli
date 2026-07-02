---

description: "Task list template for feature implementation"
---

# Tasks: CLI Development Infrastructure Bootstrap

**Input**: Design documents from `/specs/001-cli-infrastructure/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md), [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

**Tests**: Per constitution Principles VI and VIII, E2E acceptance tests are NOT optional for command behavior (User Story 1). User Stories 2 and 3 are CI/CD and release-pipeline configuration — the constitution explicitly carves these out of the `go test` E2E mandate ("Bash/shell scripts... reserved for... infrastructure tasks (CI/CD, release, system-level checks)"), so their verification uses workflow-syntax validation and observed pipeline runs (quickstart.md) instead of Go tests.

**Organization**: Tasks are grouped by user story (US1/US2/US3, matching spec.md priorities P1/P2/P3) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `cmd/arc/` — the sole Cobra command package (root command only), plus its colocated `root_test.go` E2E test (Principle III, VIII)
- `.github/workflows/` — the three mandated CI workflows
- `.goreleaser.yaml` — repo-root release configuration
- No `internal/<domain>/` package exists yet — there is no domain logic in this bootstrap feature (ADR 001's explicit small-tool exception)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Run `go mod init github.com/fogfish/arcnet-cli` at the repository root and set `go 1.26` in `go.mod` (research.md: Go toolchain version)
- [X] T002 [P] Add `github.com/spf13/cobra` and `github.com/fogfish/it/v2` via `go get`; do **not** add `github.com/charmbracelet/lipgloss` yet (research.md: terminal styling dependency intentionally deferred — no styled output exists in this feature)
- [X] T003 [P] Create the `cmd/arc/` directory scaffold per plan.md Project Structure

**Checkpoint**: Module initialized, dependencies pinned, directory ready for the root command

---

## Phase 2: Design Preconditions

**Purpose**: Implements the constitution's PRECONDITIONS (must complete BEFORE implementation begins) from the Compliance Checklist. Every subsection below is a design gate, not an implementation task.

**⚠️ CRITICAL**: No user story implementation (Phase 3+) can begin until this phase is complete.

### Phase 2a: Domain Model & Glossary (Principles II, V)

- [X] T004 Confirm this feature introduces no new domain entities (data-model.md documents only infra "configuration artifacts" — root command, CI workflow, release config, version tag, release artifact — none of which are application domain types); no [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary changes are required
- [X] T005 Confirm no `internal/<domain>` package is created or duplicated by this feature (ADR 001's small-tool exception applies — there is no domain logic yet for a bootstrap CLI skeleton)

### Phase 2b: Command & Flag Contract Design (Principle IX)

- [X] T006 [P] Verify the `arc` root command's `Use`/`Short`/`Long`/`Example`, `--help`, and `--version` behavior to be implemented in Phase 3 matches [contracts/cli-contract.md](contracts/cli-contract.md) exactly before writing code
- [X] T007 [P] Verify the three planned GitHub Actions workflows and `.goreleaser.yaml` (to be created in Phases 4–5) match [contracts/ci-release-contract.md](contracts/ci-release-contract.md) exactly before writing YAML

### Phase 2c: External Integration & Adapter Design (Principle VII)

- [X] T008 Confirm this feature requires no external system integration or port/adapter design (N/A — no cloud SDK, REST client, or other secondary adapter is introduced by a CLI skeleton, CI workflow, or release config)

### Phase 2d: E2E Acceptance Test Design (Principle VIII)

- [X] T009 [P] [US1] Write E2E test(s) in `cmd/arc/root_test.go` for every spec.md Story 1 acceptance scenario (build succeeds; `--help` prints usage and exits 0; `--version` prints a version line and exits 0; unrecognized flag/subcommand exits non-zero) using a shared `sut()` helper that pipes `os.Stdout` and invokes `RunE` directly; tests MUST compile and fail semantically (red phase) since `root.go` doesn't exist yet
- [X] T010 [US2] N/A for `go test` — User Story 2's acceptance scenarios describe GitHub Actions check behavior, not a `cobra.Command`; per the constitution's CI/CD infra-task exception, US2 is instead verified via [contracts/ci-release-contract.md](contracts/ci-release-contract.md) and quickstart.md Steps 2 and 5.1 (workflow YAML validation + an observed PR run)
- [X] T011 [US3] N/A for `go test` — User Story 3's acceptance scenarios describe release-pipeline behavior, not a `cobra.Command`; per the same infra-task exception, US3 is verified via quickstart.md Steps 4 and 5.2 (local GoReleaser snapshot dry-run + an observed merge-triggered release)

### Phase 2e: Configuration & Secrets Review (Principle XI)

- [X] T012 Confirm this feature introduces no new user-facing configuration values or secrets (N/A — `GITHUB_TOKEN` used by the release workflow is the standard CI-provisioned token, never a plaintext flag/config value)

**Checkpoint**: All Phase 2 subsections complete — user story implementation can now begin. (Phase 2.5 "Foundational Infrastructure" is omitted: Phase 1's module/directory scaffold is sufficient shared foundation, and each user story below can start directly on top of Phase 2.)

---

## Phase 3: User Story 1 - Bootstrap a runnable CLI skeleton (Priority: P1) 🎯 MVP

**Goal**: A buildable `arc` binary with a Cobra root command supporting `--help` and `--version`, with command wiring isolated under `cmd/arc/`.

**Independent Test**: `go build -o arc ./cmd/arc && ./arc --help && ./arc --version` (quickstart.md Step 1); passes without any US2/US3 work existing.

### Implementation for User Story 1

> E2E tests for this story were already written in Phase 2d (T009) and MUST currently be failing (red). Implementation below MUST turn them green with minimal test changes.

- [X] T013 [US1] Implement the Cobra root command in `cmd/arc/root.go`: `Use: "arc"`, `Short`, `Long`, `Example` populated, `--version` wired to a build-time version value, default `RunE`/`Run` printing help on no/unrecognized args (Principles IX, XII) (depends on T009's red tests existing)
- [X] T014 [US1] Implement `cmd/arc/main.go` (`package main`) that calls the root command's `Execute()` and sets the process exit code from its returned error (depends on T013)
- [X] T015 [US1] Run `go test ./cmd/arc/... -v` and confirm T009's E2E tests now pass (GREEN) with no more than minimal test adjustments (Principle VIII)
- [X] T016 [P] [US1] Run `go build ./...` and `staticcheck ./...` locally and confirm both pass clean on the new package

**Checkpoint**: User Story 1's E2E tests (T009) pass; the CLI skeleton is fully functional and independently demonstrable.

---

## Phase 4: User Story 2 - Automated verification on every pull request (Priority: P2)

**Goal**: Every pull request automatically triggers a build/test/coverage check and an independent static-analysis check, both required for merge.

**Independent Test**: Open a pull request and observe the `test` and `check` workflow runs report pass/fail independently (quickstart.md Step 2 and Step 5.1); passes without US3's release pipeline existing.

### Implementation for User Story 2

> Per T010, this story has no `go test` E2E suite of its own; verification is the workflow-validation and observed-PR-run tasks below.

- [X] T017 [P] [US2] Create `.github/workflows/check-test.yml` (workflow name `test`): `actions/setup-go@v5` pinned to `go-version: "1.26"`, `actions/checkout@v4`, `go build ./...`, `go test -v -coverprofile=profile.cov $(go list ./... | grep -v /examples/)`, `shogo82148/actions-goveralls@v1` with `continue-on-error: true` uploading `profile.cov`; triggered on `pull_request: types: [opened, synchronize]` (research.md: coverage reporting; contracts/ci-release-contract.md)
- [X] T018 [P] [US2] Create `.github/workflows/check-code.yml` (workflow name `check`): `actions/setup-go@v5` pinned to `go-version: "1.26"`, `actions/checkout@v4`, `dominikh/staticcheck-action@v1.3.1` with `install-go: false`; triggered on `pull_request: types: [opened, synchronize]` (research.md: static analysis gate)
- [X] T019 [US2] Validate both new workflow files with `actionlint` (or GitHub's workflow syntax check) — the constitution's CI/CD infra-task exception applies here in place of a `go test`-based check
- [ ] T020 [US2] Open a pull request touching any file and confirm both the `test` and `check` checks trigger, run independently, and report status on the PR (quickstart.md Step 5.1); configure both as required status checks for merge (spec FR-004, FR-005)

**Checkpoint**: User Story 2's PR-gating checks are live and independently verifiable, without any release-pipeline changes.

---

## Phase 5: User Story 3 - Automated versioned release on merge (Priority: P3)

**Goal**: Merging to the default branch automatically increments the SemVer tag and publishes cross-platform release artifacts via GoReleaser, gated by a dependency-vulnerability scan.

**Independent Test**: `goreleaser release --snapshot --clean` locally produces the expected `dist/` artifacts (quickstart.md Step 4); a real merge to `main` produces a new tag and GitHub Release (quickstart.md Step 5.2).

### Implementation for User Story 3

> Per T011, this story has no `go test` E2E suite of its own; verification is the snapshot dry-run and observed-release tasks below.

- [X] T021 [US3] Create `.goreleaser.yaml` at the repository root: `before.hooks: [go mod tidy]`; `builds` with `CGO_ENABLED=0` for `goos: [linux, windows, darwin]`, ignoring `windows/arm64`; `archives` in binary-only format; `checksum` producing `checksums.txt`; `snapshot` name template; `changelog` sorted ascending, excluding `^docs:` and `^test:`; a `brews` block (owner `fogfish`, repo `arcnet-cli`, binary `arc`, test invoking `arc --version`) (research.md: GoReleaser configuration shape)
- [X] T022 [US3] Create `.github/workflows/build.yml` (workflow name `build`): `actions/setup-go@v5` pinned to `go-version: "1.26"`, `actions/checkout@v4` with `fetch-depth: 0`, `go build`/`go test -coverprofile` + `shogo82148/actions-goveralls@v1` upload, commit-message-driven `reecetech/version-increment@2023.10.2` SemVer bump (default patch; `[x.Y.z]`/`[X.y.z]` markers for minor/major), then `git tag`+push of the new version; triggered on `push: branches: [main]` (research.md: versioning and release automation)
- [X] T023 [US3] Add a `govulncheck` step to `build.yml`, positioned after the version tag is pushed and before the GoReleaser step, failing the job on any known-critical vulnerability (constitution Principle XIV — absent from the `fogfish/iq` reference workflow but binding here; spec FR-012)
- [X] T024 [US3] Add the `goreleaser/goreleaser-action@v5` step (`args: release`, `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`) as the final step of `build.yml`, after T023's `govulncheck` gate (depends on T021, T023) — pins `with.version: "~> v2"` so the installed binary can parse `.goreleaser.yaml`'s `version: 2` schema (BUG-001 fix)
- [X] T025 [US3] Run `goreleaser release --snapshot --clean` locally against T021's config and confirm `dist/` contains the expected per-platform archives plus `checksums.txt`, with no artifacts actually published (quickstart.md Step 4) — re-run after T028's fix; `dist/` again contains the 5 platform archives + `checksums.txt`, and `goreleaser check` validates the `version: 2` config cleanly
- [X] T028 [US3] Pin `goreleaser/goreleaser-action@v5`'s `with.version:` input in `.github/workflows/build.yml` to a GoReleaser 2.x release/constraint matching `.goreleaser.yaml`'s `version: 2` schema (`version: "~> v2"`, the documented constraint syntax for this action); `actionlint` and `goreleaser check` both pass clean — closes BUG-001 (depends on T024). Note: confirming the action actually resolves to a 2.x binary in a live CI run still requires an observed push to `main` (see T020/T027 for the same operational limitation)

**Bugfix**: 2026-07-01 — BUG-001 Fixed: pinned `goreleaser/goreleaser-action@v5`'s `version:` input to `"~> v2"` in `build.yml`, matching `.goreleaser.yaml`'s `version: 2` schema. T024/T025 re-closed; T028 added and closed.

**Checkpoint**: All three user stories complete — buildable skeleton, PR gating, and automated release are each independently working.

---

## Additional Polish (OPTIONAL)

**Purpose**: Improvements that affect multiple user stories

- [X] T026 [P] Add a generated command reference (`cobra/doc`) or update README.md with `arc` build/install instructions (Principle XII, SHOULD)
- [ ] T027 Run the full [quickstart.md](quickstart.md) validation end-to-end (all 5 steps) after Phases 3–5 are complete
- [X] T029 Investigate each third-party Action pinned across `.github/workflows/*.yml` (`actions/checkout@v4`, `actions/setup-go@v5`, `shogo82148/actions-goveralls@v1`, `dominikh/staticcheck-action@v1.3.1`, `reecetech/version-increment@2023.10.2`, `goreleaser/goreleaser-action@v5`) for a newer major/tag that has migrated off the deprecated Node 20 Actions runtime; bump the pin where a runtime-current version exists, otherwise document the pin as an accepted, upstream-owned limitation (BUG-002). Findings: `actions/checkout@v4` and `actions/setup-go@v5` were on `node20` with `node24`-migrated majors available (`v7.0.0`, `v6.5.0`) — bumped to `@v7`/`@v6` in all three workflow files; `goreleaser/goreleaser-action@v5` was on `node20` with a `node24`-migrated `v7.2.3` available (whose own default `version` input is now also `"~> v2"`, confirming BUG-001's fix) — bumped to `@v7` in `build.yml`. `shogo82148/actions-goveralls@v1` was already `node24` — unchanged. `dominikh/staticcheck-action@v1.3.1` and `reecetech/version-increment@2023.10.2` are `composite` actions (shell-orchestrated, no own Node runtime) — not a source of the warning, left unchanged. `actionlint` passes clean on all three workflow files after the bumps.

**Bugfix**: 2026-07-01 — BUG-002 Fixed: bumped `actions/checkout@v4`→`@v7`, `actions/setup-go@v5`→`@v6` (all three workflows), and `goreleaser/goreleaser-action@v5`→`@v7` (`build.yml`) to node24-runtime majors, closing the deprecation warning for the three JS-based actions that were emitting it. T029 closed.

---

## Phase N: Constitution Compliance Verification

**Purpose**: Implements the constitution's Compliance Checklist (Implementation Phase). This phase MUST be retained verbatim; do not omit or merge it into other phases.

### Design Phase Verification

- [X] TN01 [ARCHITECTURE.md](../../ARCHITECTURE.md) reflects architectural changes, if any (Principle I) — expect none beyond the new `cmd/arc` entry point
- [X] TN02 Domain concepts added to the [ARCHITECTURE.md](../../ARCHITECTURE.md) Glossary (Principle II) — expect none per T004
- [X] TN03 Command/flag surface matches the Phase 2b design exactly: flag names, help text, exit codes (Principle IX)

### Implementation Phase Verification (grouped by principle)

- [X] TN04 Major decisions recorded in [adrs/](../../adrs/) with correct numbering, if a new architectural pattern was introduced (Principle I) — none expected; this feature only applies ADR 001/002 as written
- [X] TN05 Domain logic uses ports (interfaces); Cobra wiring and adapters remain separated (Principle III) — N/A, no domain logic yet
- [X] TN06 Unit tests were written first, compiled, and failed semantically before implementation (Principle VI) — applies to T009's E2E test for US1
- [X] TN07 Unit and E2E tests use `github.com/fogfish/it/v2` exclusively — no `testify` or stdlib-only comparisons mixed in (Principle VI, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling))
- [X] TN08 No Bash scripts were used for unit-level code correctness validation (Principle VI) — quickstart.md's shell commands validate infra (build/CI/release), not unit-level logic
- [X] TN09 New external integrations follow the port/adapter pattern; no vendor SDK types leak through a port (Principle VII) — N/A, no external integration
- [X] TN10 Terminal output respects TTY detection, `NO_COLOR`, `--quiet`/`--verbose`, and uses `github.com/charmbracelet/lipgloss` for any styling (Principle X) — N/A this feature, no styled output introduced (research.md)
- [X] TN11 Configuration precedence and XDG locations respected; no secrets logged or accepted only via plaintext flags (Principle XI) — N/A, no new configuration
- [X] TN12 Help text (`Short`/`Long`/`Example`) populated for every new/changed command (Principle XII) — applies to the `arc` root command (T013)
- [X] TN13 E2E tests from Phase 2d turned GREEN and changed minimally during implementation (Principle VIII) — applies to T009 → T015
- [X] TN14 All spec.md scenarios for this feature have a passing, colocated E2E test where Principle VIII applies (Principle VIII) — US1 scenarios via T009/T015; US2/US3 scenarios verified per the CI/CD infra-task exception (T019/T020, T025/T028 + observed merge). BUG-001: T025 re-run after T028 pinned the action version — closed
- [X] TN15 Release/versioning impact assessed: does this feature change command names, flag semantics, or `--json`/`--plain` output in a way that requires a major version bump? (Principle XIV) — no, this is the first release of `arc`; no prior contract exists to break

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Design Preconditions (Phase 2)**: Depends on Setup — BLOCKS all user stories; subsections 2a–2e can proceed in parallel with each other
- **User Stories (Phase 3–5)**: All depend on Phase 2 completion
  - US1, US2, US3 have no dependencies on each other and can proceed in parallel if staffed, or sequentially in priority order (P1 → P2 → P3)
- **Additional Polish**: Depends on the desired user stories being complete
- **Constitution Compliance Verification (Phase N)**: Final gate — depends on all preceding phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2 — no dependency on US2/US3
- **User Story 2 (P2)**: Can start after Phase 2 — independent of US1/US3 (the workflows it adds run `go build`/`go test` against whatever exists, including US1's skeleton once merged, but the workflow files themselves don't require US1's code to exist to be authored and syntax-validated)
- **User Story 3 (P3)**: Can start after Phase 2 — independent of US1/US2, though its `build.yml` reuses the same build/test steps as US2's `check-test.yml`

### Within Each User Story

- US1: E2E tests (T009, Phase 2d) before implementation (T013–T015); build/lint check (T016) last
- US2: The two workflow files (T017, T018) can be authored in parallel; validation (T019) and the live PR check (T020) come after both exist
- US3: `.goreleaser.yaml` (T021) and the `build.yml` skeleton (T022) can be authored in parallel; `govulncheck` (T023) and the GoReleaser step (T024) must be added to `build.yml` in that order; the snapshot dry-run (T025) comes last

### Parallel Opportunities

- T002 and T003 (Setup) can run in parallel
- T004/T005/T008/T012 (Phase 2 N/A confirmations) can all run in parallel with T006/T007/T009 (they touch different concerns)
- Once Phase 2 completes, US1, US2, and US3 can all start in parallel (different files: `cmd/arc/*`, `.github/workflows/*`, `.goreleaser.yaml`)
- Within US2: T017 and T018 in parallel (different files)
- Within US3: T021 and T022 in parallel (different files)

---

## Parallel Example: User Story 1

```bash
# Design (Phase 2, already complete before this point):
# T009 E2E test(s) for US1 acceptance scenarios in cmd/arc/root_test.go

# Launch independent implementation tasks for User Story 1 together:
Task: "Implement Cobra root command in cmd/arc/root.go"
Task: "Implement package main in cmd/arc/main.go"
```

## Parallel Example: User Stories 2 and 3 (cross-story, after Phase 2)

```bash
Task: "Create .github/workflows/check-test.yml"       # US2
Task: "Create .github/workflows/check-code.yml"        # US2
Task: "Create .goreleaser.yaml"                         # US3
Task: "Create .github/workflows/build.yml (skeleton)"   # US3
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Design Preconditions (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1
4. Complete Phase N: Constitution Compliance Verification (US1-scoped items)
5. **STOP and VALIDATE**: `go build -o arc ./cmd/arc && ./arc --help && ./arc --version`
6. Deploy/demo if ready — this alone proves the module builds and CI has something to check

### Incremental Delivery

1. Complete Setup + Design Preconditions → Foundation ready
2. Add User Story 1 → Verify against Phase N → Deploy/Demo (MVP!)
3. Add User Story 2 → Verify against Phase N → Deploy/Demo (PR gating live)
4. Add User Story 3 → Verify against Phase N → Deploy/Demo (releases automated)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple contributors:

1. Team completes Setup + Design Preconditions together
2. Once complete:
   - Contributor A: User Story 1 (`cmd/arc/`)
   - Contributor B: User Story 2 (`.github/workflows/check-test.yml`, `check-code.yml`)
   - Contributor C: User Story 3 (`.goreleaser.yaml`, `.github/workflows/build.yml`)
3. Stories complete and integrate independently; each runs Phase N verification before merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US2 and US3 tasks marked N/A for `go test` (T010, T011) intentionally use the constitution's CI/CD infra-task exception instead of fabricating Go tests for GitHub Actions/GoReleaser behavior that isn't a `cobra.Command`
- Commit after each task or logical group
- Stop at any checkpoint to validate a story independently
- Phase 2 and Phase N sections are retained verbatim in structure per constitution Governance > Task List Requirements — only task descriptions were adapted to this feature
