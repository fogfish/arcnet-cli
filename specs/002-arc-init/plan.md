# Implementation Plan: Initialize a New Knowledge Graph (`arc init`)

**Branch**: `002-arc-init` | **Date**: 2026-07-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-arc-init/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc init [<dir>]`: create the canonical graph folder layout, `_meta/` registry stubs, the `.arc/` state directory and its `.gitignore` exclusion, `.gitkeep` placeholders, and exactly one git commit (`graph(init): empty knowledge graph`), guarded so that an already-initialized graph or a non-empty target directory is refused untouched (spec FR-014, FR-015). Graph management becomes its own domain, `internal/app/ctrl` (control plane), mirrored by a `cmd/arc/ctrl` command package (see research.md naming note on the `ctrl`/`crtl` typo assumption). Git is a first-class dependency, invoked as the real `git` binary via `os/exec` behind a `port.VCS` interface, with every git operation reported to the user through the ADR 002 DS-06 `Reporter` port. This is the first feature to need styled/state-changing terminal output, so it also stands up the shared `internal/bios` kernel (DS-04 output modes/registry, DS-05 color schema, DS-06 reporter) that every future command will reuse, activating `github.com/charmbracelet/lipgloss` per constitution Principle X. It is also the first feature to define expected errors, so every failure path (missing git, already-initialized graph, non-empty target, mid-run I/O failure) is declared as a `github.com/fogfish/faults` constant and wrapped via `.With()`, per the newly-added constitution Principle XII / Mandatory Libraries & Tooling mandate (research.md D7). All graph-content filesystem I/O goes through a new shared `internal/adapter/fsys` package built exclusively on stdlib `io/fs`, `io.Reader`, and `io.Writer` (constitution v1.5.0, Mandatory Libraries & Tooling: "Filesystem Abstraction") — no third-party filesystem library, and `os`'s file/directory functions confined entirely to that one package. This corrects two earlier drafts of this plan: the first had domain code calling `os.MkdirAll`/`os.WriteFile` directly; the second adopted `github.com/fogfish/stream`, which was itself dropped after review — its local and S3 backends shared one Go package (so even local-only use pulled in the full AWS SDK v2 tree) and its S3 backend would not have delivered a real capability anyway, since `arc init`'s git-commit contract (CORE §11) needs a local working tree an S3 object store isn't. `fsys` keeps two concerns apart: `os`/`path/filepath` resolves (and, for `init`, creates) the local root *before* anything is mounted, and `io/fs`/`io.Writer` handle all I/O against an already-resolved root, as a `Store` (wrapping `os.DirFS` plus a minimal `Create`/`Remove` write-side extension) behind a `Mounter`. There is no remote backend; `arc init` mounts local directories only (research.md D3, three times revised).

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod` and `specs/001-cli-infrastructure/plan.md`

**Primary Dependencies**: `github.com/spf13/cobra` (existing); `github.com/charmbracelet/lipgloss` (newly activated — constitution Principle X, ADR 002 DS-05); `github.com/fogfish/faults` (newly activated — constitution Principle XII, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling)) for all error context annotation in `internal/app/ctrl` and `internal/bios`; no filesystem library — `internal/adapter/fsys` is built on stdlib `io/fs`/`io.Reader`/`io.Writer` only (constitution Principle VII, Mandatory Libraries & Tooling: "Filesystem Abstraction", v1.5.0); the system `git` binary, invoked as an external process, not a Go module dependency (research.md D1)

**Storage**: The mounted graph root is the graph itself — a local directory, accessed exclusively through `internal/adapter/fsys`'s `Store`/`Mounter` (stdlib `io/fs`-backed) — never raw `os.*` calls in domain/service code (research.md D3, three times revised)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/ctrl/init_test.go` via the existing `sut()`/`run()` helpers, one per spec.md acceptance scenario (constitution Principles VI, VIII); unit tests for `internal/app/ctrl/service` against fakes of `fsys.Mounter`/`fsys.Store` and a mock `port.VCS` adapter; an integration test for `internal/adapter/fsys`'s `Local` type and for `internal/app/ctrl/adapter/git` that exercise the real local filesystem and real `git` binary against `t.TempDir()` (constitution Principle VI: real file I/O against a temp directory is sanctioned, no filesystem mocking needed at that layer)

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — first feature to add an `internal/` package (constitution Principle III now takes full effect; ADR 001's small-tool exception no longer applies)

**Performance Goals**: Spec SC-001 — full initialization (layout + git init/add/commit) completes in under 5 seconds on a typical local filesystem; trivial given no network calls

**Constraints**: Git MUST be available on `PATH`, checked and reported as a clear error before any write if missing (spec FR-011); no partial graph state may remain on any failure path (spec FR-013, research.md D4); initialization is fully local/offline (spec Assumptions); the tool does not configure git identity (`user.name`/`user.email`) itself (spec Assumptions)

**Scale/Scope**: One new bare-verb command (`arc init`), one new domain package (`internal/app/ctrl`) with one use-case (`Init`), one new secondary adapter (`internal/app/ctrl/adapter/git`) plus its mock, one new shared filesystem adapter package (`internal/adapter/fsys`) with a single `Local` implementation, one new shared kernel package (`internal/bios`) covering DS-04/DS-05/DS-06, and new persistent root flags (`--quiet`/`-q`, `--verbose`/`-v`, `--json`, `--color`/`-C`) on `cmd/arc/root.go`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — this is the first feature to populate the still-empty `ARCHITECTURE.md`: Phase 1 design adds the domain model, glossary entries (Graph Root, Canonical Folder, Metadata Stub, Arc State Directory, Initial Commit), and the `internal/`+`cmd/arc/ctrl` directory-structure explanation required by Principles I and III. `tasks.md` MUST include an `ARCHITECTURE.md` update task. |
| II — DDD & Glossary | Yes | PASS — glossary terms defined in data-model.md, to be copied into `ARCHITECTURE.md` per Principle I obligation above |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/ctrl` is Cobra wiring only; `internal/app/ctrl/{kernel,port,service,adapter}` holds domain logic, ports, and adapters per ADR 001's `componentX` layout; `internal/adapter/fsys` is ADR 001's shared "phase 2" adapter tier, reused across future use-cases |
| IV — Functional Programming Style | Yes | PASS — no inline comments, short composable functions, immutable value types for kernel entities; enforced during implementation |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — narrow `port.VCS` interface scoped to exactly the three operations `Init` needs; `fsys.Store` is the narrow `io/fs`-based subset this use-case calls plus one small `Create`/`Remove` write-side extension, no third-party library, no unused S3 stub (research.md D3, three times revised) |
| VI — TDD | Yes | PASS — E2E tests and service unit tests written first per constitution; `fsys.Local` and git adapter integration tests use the real local filesystem / real `git` against `t.TempDir()` |
| VII — External Integration & Adapter Consistency | Yes | PASS — git subprocess access goes through `port.VCS` + `adapter/git` (mocked for unit tests); all filesystem I/O goes through `fsys.ResolveLocalRoot`/`fsys.Mounter`/`fsys.Store` (`os` calls confined entirely to `internal/adapter/fsys`, mocked for unit tests) per the constitution's Filesystem Abstraction mandate; no vendor/subprocess types leak through either port (research.md D1, D2, D3) |
| VIII — E2E Acceptance Testing | Yes | PASS — 7 acceptance scenarios across spec.md US1–US3 map 1:1 to E2E tests in `cmd/arc/ctrl/init_test.go` |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (research.md D6), DS-02 options struct for the `<dir>` argument, DS-03 persistent flags added to root, DS-07 `SilenceUsage`/`SilenceErrors` + centralized error formatting |
| X — Terminal Output, Color & Interactivity | Yes | PASS — DS-04 output registry, DS-05 lipgloss schema, DS-06 reporter for git progress; success message states what changed (path + commit); this feature activates the previously-deferred lipgloss dependency |
| XI — Configuration, Env & Secrets | No | N/A — no config file, no secrets involved in initialization |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected error (already-initialized graph, non-empty directory, missing git, mid-run I/O failure) is declared as a `faults.Type`/`faults.SafeN` constant and wrapped via `.With()`, never ad hoc `fmt.Errorf` (constitution Principle XII, [Mandatory Libraries & Tooling](../../.specify/memory/constitution.md#mandatory-libraries--tooling); research.md D7) |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline established in `specs/001-cli-infrastructure` |
| XIV — Versioning/Security | Yes | PASS — this feature establishes the first `--json` output contract for `arc init`; no telemetry introduced; no breaking change (nothing prior to break) |

No violations requiring justification — Complexity Tracking section is empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-arc-init/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md         # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/            # Phase 1 output
│   ├── cli-contract.md
│   ├── vcs-port-contract.md
│   └── fsys-port-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── main.go              # unchanged: package main, calls newRootCmd().Execute()
    ├── root.go               # + DS-03 persistent flags (--quiet/-q, --verbose/-v, --json, --color/-C),
    │                         #   PersistentPreRun selecting bios.SCHEMA, registers ctrl.NewInitCmd()
    ├── root_test.go          # unchanged existing tests
    └── ctrl/
        ├── init.go            # package ctrl: NewInitCmd() *cobra.Command — flag parsing (optsInit),
        │                      #   calls internal/app/ctrl.Init, renders via bios.Registry, PostRunE hint
        └── init_test.go       # E2E tests, one per spec.md US1-US3 acceptance scenario, via sut()/run()

internal/
├── bios/                     # shared kernel (ADR 002 DS-04, DS-05, DS-06) — new in this feature
│   ├── output.go              # Mode, ResolveMode(), Printer[T], Registry[T], jsonPrinter[T], nonePrinter[T]
│   ├── output_test.go
│   ├── theme.go                # Schema, SCHEMA_PLAIN, SCHEMA_COLOR, SCHEMA package var
│   ├── theme_test.go
│   ├── reporter.go             # Reporter port, stderrReporter, silentReporter, newReporter(quiet, silent)
│   └── reporter_test.go
│
├── adapter/
│   └── fsys/                  # shared, cross-use-case filesystem adapter — new in this feature
│       ├── resolve.go          # ResolveLocalRoot/RemoveLocalRoot: os/path/filepath, root
│       │                       #   resolution only, no mounted I/O (D3); sole os.* call site
│       ├── resolve_test.go     # real local filesystem, t.TempDir()
│       ├── types.go            # File, Store, Mounter interfaces (stdlib io/fs + io.Writer only)
│       ├── errors.go           # faults.Type/SafeN sentinel constants (ErrRootNotDirectory,
│       │                       #   ErrRootCreate, ErrCreate, ErrRemove)
│       ├── local.go            # Local.Mount: wraps os.DirFS(root) for reads, adds Create/Remove
│       │                       #   (os.MkdirAll/os.Create/os.Remove) for writes — sole os.* call site
│       └── local_test.go       # integration test, real local filesystem, t.TempDir()
│
└── app/
    └── ctrl/                  # graph management (control plane) domain — new in this feature
        ├── kernel/
        │   ├── graph.go        # GraphRoot, ArcNetCoreLayout, InitResult value types
        │   └── graph_test.go
        ├── port/
        │   └── vcs.go          # VCS interface: IsAvailable, Init, StageAll, Commit (private to ctrl —
        │                       #   filesystem access is NOT declared here; service.Init depends on
        │                       #   internal/adapter/fsys.Mounter/.Store directly, see research.md D3)
        ├── adapter/
        │   ├── git/
        │   │   ├── git.go      # os/exec-backed VCS implementation, reports via bios.Reporter
        │   │   └── git_test.go # integration test, real git binary, t.TempDir()
        │   └── mock/
        │       └── mock.go     # fake VCS for service unit tests
        ├── service/
        │   ├── errors.go        # faults.Type/SafeN sentinel constants for every expected failure (D7)
        │   ├── init.go         # Init use-case: fsys.ResolveLocalRoot then Mounter.Mount, guards (D4),
        │   │                   #   layout creation via Store.Create (D3), VCS calls (D1/D2)
        │   └── init_test.go    # unit tests against adapter/mock + fakes of fsys.Mounter/Store/
        │                       #   ResolveLocalRoot, table-driven guard scenarios, asserting
        │                       #   errors.Is(err, ...Xxx) per case
        ├── README.md            # use-case documentation (ADR 001 layout)
        └── component.go        # primary port: Init(ctx, mounter fsys.Mounter, vcs port.VCS, dir string)
                                 #   (kernel.InitResult, error)

testdata/
└── ctrl/                      # fixtures for init_test.go, if any golden files are needed

ARCHITECTURE.md                 # + populated for the first time: layout explanation, Glossary entries
                                 #   (Graph Root, Canonical Folder, Metadata Stub, Arc State Directory,
                                 #   Initial Commit), Principle IX subcommand-naming decision record
```

**Structure Decision**: This feature adds the project's first `internal/` packages, in three tiers: the shared `internal/bios` kernel (DS-04/05/06, reusable by every future command), the shared `internal/adapter/fsys` filesystem adapter (`Store`/`Mounter`/`Local`, stdlib `io/fs`-based, reusable by every future use-case that mounts a graph root), and the first domain use-case package `internal/app/ctrl` (graph management / control plane), following ADR 001's `componentX` layout (`kernel/`, `port/`, `adapter/`, `service/`, `component.go`). The command surface adds `cmd/arc/ctrl/init.go`, a sibling package to `cmd/arc`'s root, registered into the existing root command. No other `internal/<domain>` package is touched or created.

## Complexity Tracking

*No entries — Constitution Check has no unresolved violations.*

## Bugfix Log

**Bugfix**: 2026-07-02 — BUG-001: `arc init`'s default output over-reported (per-step git progress shown unconditionally instead of only under `--verbose`) and rendered with broken line alignment (a `lipgloss.Style.Render()` newline-handling bug). See `research.md` D2 Bugfix for the full revised decision and root-cause analysis, `contracts/cli-contract.md`/`contracts/vcs-port-contract.md` for the updated stdout/stderr contract (verbose-gated progress, short commit hash, revised `PostRunE` hint), and `spec.md` FR-016 for the newly-added default-conciseness requirement. Reopened tasks: T015-T017, T026, T035 (see `tasks.md`).
