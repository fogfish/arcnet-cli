# Implementation Plan: Graph Initialization Inside an Existing Git Repository

**Branch**: `031-init-existing-git-repo` | **Date**: 2026-09-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/031-init-existing-git-repo/spec.md`, plus an explicit `/speckit-plan` instruction: "Use existing features, services and infrastructure to implement requirements. Limit necessary changes at `cmd` and adapters level only. If existing service requires modification ask."

## Summary

`arc init` refuses to run inside an existing git repository, and gains `--skip-git-init` to add a graph to one instead. `cmd/arc/ctrl/init.go` registers the flag and resolves two environment facts through the git adapter before calling the service — the enclosing repository root (`git rev-parse --show-toplevel`) and whether the host's ignore rules exclude the target (`git check-ignore`) — passing both into `service.Init` as a new `kernel.InitOpts` value. `internal/adapter/git` gains four thin wrappers (`RepoRoot`, `IsIgnored`, `StagePaths`, `CommitPaths`); `internal/app/ctrl/port.VCS` swaps `StageAll`/`Commit` for `StagePaths`/`CommitPaths`. `internal/app/ctrl/service/init.go` is modified — unavoidably, see below — to add three guards, a mode branch around `vcs.Init`, a recorded write footprint replacing the static rollback list, and `.arc/.gitignore` replacing the root `.gitignore` write. `internal/app/graph` and `internal/app/lint` are **not** touched.

**On the "cmd and adapters only" constraint**: it cannot be satisfied literally, and this was put to the user during planning rather than decided unilaterally. 11 of the 31 requirements — FR-004/005/006, FR-009, FR-010, FR-012, FR-013/014, FR-015, FR-016/017, FR-018/019, FR-020 — are guard, layout-write and rollback behavior living inside `service.Init` and its unexported `writeLayout`/`rollback` helpers. There is no seam to extend from `cmd`, and Constitution Principle III forbids relocating business rules into the command layer. The user chose the **minimal service surface** option: environment probing stays in `cmd`/adapters, the service receives plain values and keeps its port narrow (one method swapped, not three added), so `internal/app/ctrl/adapter/mock.VCS` grows by two methods and no new test double is introduced. See [research.md](research.md) D6.

**On FR-021–FR-025** (every git-backed command must use the parent repository): the user chose **verification-first**. The git adapter already carries nested-repo fixes from BUG-002 / spec 016 (`-- .` on `StageAll` and `CommitsMatching`, `--relative` on `ChangedPaths`, `./` on `ShowFile`). This feature adds E2E coverage exercising `arc apply`/`arc revert`/`arc lint` against a nested graph and fixes only what actually fails, rather than hardening working code speculatively. See [research.md](research.md) D7.

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra` (one new bool flag), `github.com/fogfish/faults` (six new error sentinels) — **no new module dependency**; git remains a subprocess behind `internal/adapter/git`, per ADR 001

**Storage**: local filesystem via `internal/adapter/fsys` (unchanged interface — `Store.Stat`'s `fs.FileInfo.IsDir()` already distinguishes the file and folder collision classes FR-015 needs, so no `fsys` change); git repository state via `internal/adapter/git`

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` — unit tests in `internal/app/ctrl/service/init_test.go` (existing fakes extended), adapter tests in `internal/adapter/git/git_test.go`, E2E in `cmd/arc/ctrl/init_test.go` and a new nested-graph E2E file (Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged

**Project Type**: Single Cobra CLI binary (Principle III). No new package, no new adapter, no new Cobra command — one new flag on an existing command

**Performance Goals**: two additional `git` subprocess invocations per `arc init` (`rev-parse --show-toplevel`, and `check-ignore` only when a parent repository was found). Negligible against the four-to-five invocations init already makes; well inside Constitution X's ~100ms first-output budget

**Constraints**: guards MUST all precede `resolveLocalRoot` so a refusal never touches disk (FR-005) and so detection works for a not-yet-existing target (research D2) — this reorders the current sequence; `CommitPaths` MUST be a new method rather than a signature change to `Commit`, because the single concrete `git.VCS` type structurally satisfies both `ctrl` and `graph` port interfaces (ADR 001) and `graph`'s declares the pathspec-free shape; `internal/app/graph` and `internal/app/lint` MUST remain unmodified

**Scale/Scope**: one new kernel type + one extended kernel type (`internal/app/ctrl/kernel/graph.go`), four adapter methods + two error sentinels (`internal/adapter/git/git.go`), one port swap (`internal/app/ctrl/port/vcs.go`), two mock methods (`internal/app/ctrl/adapter/mock`), the service rework (`internal/app/ctrl/service/init.go` + four error sentinels in `errors.go`), one delegator signature (`internal/app/ctrl/component.go`), one command file (`cmd/arc/ctrl/init.go`), plus `README.md` and an `ARCHITECTURE.md` note

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | No new ADR needed — ADR 001's adapter tier and port-isolation rules are followed, not amended. `ARCHITECTURE.md` MUST gain a short note (tracked as a task) recording that `ctrl` deliberately resolves repository context in `cmd` and passes it in as a value, while `graph`/`lint` probe through their own ports, so a future contributor does not read the asymmetry as an oversight. |
| II. DDD & Glossary | PASS (action required) | Two new domain terms — **parent repository** and **initialization footprint** — are used throughout spec and plan. Both MUST be added to `ARCHITECTURE.md`'s Glossary (tracked as a task). `InitOpts` is a `kernel`-layer DTO, the same category as `kernel.InitResult`, requiring no Glossary entry of its own. |
| III. Hexagonal Architecture | PASS | `service.Init` imports no `cobra` and no `cmd/`. `cmd/arc/ctrl/init.go` gains flag parsing and two adapter probes — I/O invocation, not business rules; every *decision* made from the probe results stays in the service (see research D6 for why this is the line, and why the cmd-only alternative was rejected as a Principle III violation). |
| IV. Functional Programming Style | PASS | New guards are small pure predicates over `InitOpts` + `fsys.Store`. The footprint is an accumulated return value from `writeLayout`, not mutable shared state. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package or adapter. `port.VCS` **shrinks by two and grows by two** (`StageAll`/`Commit` out, `StagePaths`/`CommitPaths` in) rather than accumulating; detection is deliberately kept off the port. The footprint replaces the static rollback list rather than being added alongside it. |
| VI. TDD | PASS (process gate) | Unit tests in `internal/app/ctrl/service/init_test.go` MUST be written first. Existing `fakeStore`/`mock.VCS` are extended, not replaced — `fakeStore.Stat` already resolves written paths, and its `fakeDirEntry.IsDir()` returns false, which MUST be made configurable for the folder-collision guard. |
| VII. External Integration & Adapter Consistency | PASS | Four new methods on the existing `internal/adapter/git` adapter, following its established shape exactly: `os/exec` confined to `run()`, `faults.Type` sentinels, and the `(zero, nil)` convention for an expected negative answer that `IsTracked`/`ShowFile` already set (research D1, D5). No `os.*` call is added outside `internal/adapter/fsys`. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate) | Every acceptance scenario across the spec's four user stories gets a colocated E2E test in `cmd/arc/ctrl/init_test.go`; User Story 4's go in a new nested-graph E2E file. [quickstart.md](quickstart.md) S1–S9 enumerate them. |
| IX. Command & Flag Design (CLIG) | PASS | `--skip-git-init`: long form only, no shorthand (advanced, infrequent — shorthands are reserved). Exit codes unchanged: the codebase exits 1 uniformly and this feature introduces no new code, matching FR-029. Failures print an actionable message through `main.go`'s single `humanize` path, never a raw Go error. |
| X. Terminal Output, Color & Interactivity | PASS | `--json` gains one additive field; the three existing fields keep names and meanings (FR-028). Human output stays a single concise line with BUG-001's constraints intact. Under `--verbose`, `--skip-git-init` correctly reports two steps rather than three, since no repository is created. |
| XI. Configuration, Environment Variables & Secrets | PASS | This was the binding constraint on clarification 1. The self-contained `.arc/.gitignore` means `arc` never creates, modifies or appends to a file it does not own, so XI's "ask before modifying a config file it does not own / prefer creating over appending" rule is satisfied without a confirmation prompt. |
| XII. Documentation & Help System | PASS (action required) | `Long`/`Example` on the init command MUST document the flag and the refusal it overrides (FR-030), and `README.md`'s command table MUST show it. All new errors use `faults.Safe1`/`.With()`, never ad hoc `fmt.Errorf`. |
| XIII. Distribution & Release Engineering | N/A | No release-pipeline change. |
| XIV. Versioning, Security & Compatibility | PASS | `--json` change is purely additive — existing consumers keep working (FR-028). The one behavior change to an existing path is where the local-state exclusion is written; SC-008 pins the observable outcome as unchanged, and it is documented in [research.md](research.md) D4. |

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/031-init-existing-git-repo/
├── plan.md                        # This file
├── research.md                    # Phase 0 — D1..D7, all git behavior verified empirically
├── data-model.md                  # Phase 1 — InitOpts, InitResult, Footprint, state transition
├── quickstart.md                  # Phase 1 — S1..S9 runnable validation
├── contracts/
│   ├── cli-contract.md            # Flag, mode table, JSON shape, failure contract, port surface
│   └── vcs-adapter-contract.md    # RepoRoot/IsIgnored/StagePaths/CommitPaths + exit-code semantics
├── checklists/requirements.md     # Spec quality checklist (16/16)
├── spec.md
└── tasks.md                       # Phase 2 (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/arc/ctrl/
├── init.go                        # MODIFY: --skip-git-init flag; resolve parent repo + ignore
│                                  #   status via git adapter; pass kernel.InitOpts; render
│                                  #   the new `repository` field and the FR-026 human line
└── init_test.go                   # MODIFY: E2E for US1/US2/US3 (quickstart S1-S8); update the
                                   #   existing root-.gitignore assertions per research D4

internal/adapter/git/
├── git.go                         # MODIFY: + RepoRoot, IsIgnored, StagePaths, CommitPaths
│                                  #   + ErrGitRevParse, ErrGitCheckIgnore
│                                  #   StageAll and Commit RETAINED for internal/app/graph
└── git_test.go                    # MODIFY: adapter tests incl. exit-code discrimination

internal/app/ctrl/
├── component.go                   # MODIFY: Init delegator gains the opts parameter
├── kernel/graph.go                # MODIFY: + InitOpts; InitResult gains Repository/`repository`
├── port/vcs.go                    # MODIFY: StageAll/Commit -> StagePaths/CommitPaths
├── adapter/mock/                  # MODIFY: mock.VCS gains StagePaths/CommitPaths + call log
└── service/
    ├── init.go                    # MODIFY: three new guards ahead of resolveLocalRoot; mode
    │                              #   branch around vcs.Init; writeLayout returns the footprint
    │                              #   and writes .arc/.gitignore instead of root .gitignore;
    │                              #   rollback consumes the footprint
    ├── errors.go                  # MODIFY: + ErrInsideRepository, ErrNoParentRepository,
    │                              #   ErrLayoutCollision, ErrTargetIgnored
    └── init_test.go               # MODIFY: unit tests first (Principle VI)

cmd/arc/graph/
└── nested_repo_test.go            # NEW: US4 / FR-021-025 verification-first E2E (quickstart S9)

internal/app/graph/**              # UNCHANGED — this is why CommitPaths is a new method
internal/app/lint/**               # UNCHANGED
internal/adapter/fsys/**           # UNCHANGED — Stat().IsDir() already covers FR-015
```

**Structure Decision**: no new package. The feature touches one Cobra command (`cmd/arc/ctrl`), one shared adapter (`internal/adapter/git`), and one use-case package (`internal/app/ctrl`, across its `kernel`/`port`/`service`/`adapter/mock` layers). The single new file is the User Story 4 E2E, placed in `cmd/arc/graph` beside the commands it exercises, per Principle VIII's colocation rule.

## Phase 2 sequencing note

`/speckit-tasks` should order work bottom-up so each layer's tests can go green before the next depends on it: adapter methods → port + mock → kernel types → service guards and footprint → cmd wiring and rendering → US4 verification E2E → docs (`ARCHITECTURE.md` Glossary + note, `README.md`). The two Principle I/II documentation actions flagged PASS-with-action above are tasks, not gate failures.

## Complexity Tracking

> No Constitution Check violations. Table intentionally empty.
