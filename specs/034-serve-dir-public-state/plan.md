# Implementation Plan: Explicit Serve Context & Shareable Graph State

**Branch**: `034-serve-dir-public-state` | **Date**: 2026-09-11 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/034-serve-dir-public-state/spec.md`

**Scope directive**: *"The feature does not require any structural changes in the
code base. Only impacted component has to be modified. Keep minimal changes to
interfaces, adjust only the behaviour of `serve` and `init` commands."*

## Summary

Two behavioural changes, no structural ones.

**`arc serve` gains an optional `<dir>`.** `buildServer(ctx, dir)` and all eight MCP
tool handlers already take the graph directory as a parameter — nothing downstream of
`RunE` reads the working directory. The entire change is `Args: cobra.NoArgs` →
`cobra.MaximumNArgs(1)`, one resolution line, and updated help text.

**`arc init` splits `.arc/` into a tracked public part and an ignored cache.** The
ignore rule in `.arc/.gitignore` changes from `*` to `cache/`, a new
`.arc/cache/.gitignore` holds `*`, and `footprint.Tracked()` excludes
`.arc/cache/` instead of all of `.arc/`. That makes a clone of a graph *be* a graph,
and makes `.arc/config.yml` travel with the repository for the first time.

Three supporting edits fall out of those two: `EnsureGraph` learns to say *"does not
exist"* instead of *"is not an initialized graph"* (research D2); a new guard refuses
`--skip-git-init` when the host repository would exclude `.arc/` (research D5); and
`fsys.probeFoldsCase` moves its throwaway temp file into `.arc/cache/`, because its
existing justification — *"already gitignored"* — is exactly what this feature retires
(research D6).

No new package, no new port interface, no signature change to any exported function,
no ADR. Legacy graphs keep working with zero compatibility code.

## Technical Context

**Language/Version**: Go 1.25 (per `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/fogfish/faults`,
`github.com/fogfish/it/v2`, `github.com/charmbracelet/lipgloss`,
`github.com/modelcontextprotocol/go-sdk/mcp` — all already present. **No new
dependency.**

**Storage**: local filesystem via `internal/adapter/fsys` (`io/fs` + `os.DirFS`);
git via `internal/adapter/git`

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E via the colocated
`sut()` helper in `cmd/arc/{graph,ctrl}` and `connectServeSession` in
`cmd/arc/graph/serve_test.go`

**Target Platform**: linux/darwin/windows, amd64+arm64 (`.goreleaser.yaml`)

**Project Type**: Single Cobra CLI binary (Constitution III)

**Performance Goals**: preflight refusal within ~1s (SC-007) — one `Stat` call, met
trivially. No change to server or graph-walk performance.

**Constraints**: no new external dependency; no `os.*` filesystem call outside
`internal/adapter/fsys`; additive CLI surface only (no major version bump).

**Scale/Scope**: 6 source files, 4 test files, 3 documentation files. Roughly 60 lines
of production change.

## Constitution Check

*GATE: passed before Phase 0; re-checked after Phase 1 — see bottom.*

| Principle | Assessment |
|-----------|------------|
| **I** Architecture & ADRs | **No new ADR.** No architectural pattern changes: a command gains an optional positional argument (ADR 002's existing grammar) and a static layout constant gains an entry. ADR 001 (hexagonal layering) and ADR 003 (MCP server adapter) are both untouched. `ARCHITECTURE.md` glossary **must** be updated — *Arc State Directory* is rewritten and *Arc Cache Directory* added (research D9). ✅ |
| **II** DDD & Glossary | "Arc Cache Directory" is a new domain term and enters the glossary. Ubiquitous language holds: the spec, the constant (`arcCacheDir`), the on-disk path, and the help text all say *cache*. ✅ |
| **III** Hexagonal boundaries | Nothing crosses a layer it did not already cross. `cmd/` keeps only argument resolution and wiring; `resolveGraphDir` is pure path arithmetic. Layout and guard decisions stay in `internal/app/ctrl/service`. ✅ |
| **IV** Functional style | `resolveGraphDir` is 5 lines and pure. The `EnsureGraph` stat-classification is a `switch` under the 25-line limit. No inline comments beyond GoDoc. ✅ |
| **V** SOLID / YAGNI | No catch-all package. `.arc/cache/` is reserved but **empty of producers** — the spec's FR-015 explicitly forbids writing cache content in this feature, which is YAGNI honoured, not violated. ✅ |
| **VI** TDD | Unit tests first: `internal/app/ctrl/service/init_test.go` (layout paths, `Tracked()`, guard R4), `internal/app/graph/service` (the three new error classes). Red before green. ✅ |
| **VII** Ports & adapters | **No port interface changes.** `port.VCS`, `fsys.Store`, `fsys.Mounter` all keep their exact method sets. The `.arc/` ignore probe reuses `repoProbe.IsIgnored`, already on the interface. All `os.*` filesystem calls stay inside `internal/adapter/fsys`. ✅ |
| **VIII** E2E & traceability | Every acceptance scenario maps 1:1 to a test in the already-existing colocated files (`cmd/arc/graph/serve_test.go`, `cmd/arc/ctrl/init_test.go`), driven through `RunE` via `sut()`. No new test tree. ✅ |
| **IX** CLIG / Cobra | `<dir>` is the command's single primary "subject" argument — exactly the case Principle IX permits a positional for. `cobra.MaximumNArgs(1)` yields concise help on misuse. No new flag, no second parsing path. Additive only. ✅ |
| **X** Terminal output | Refusals go to stderr with a non-zero exit; no new colored output, no spinner, no `--json` schema change. Ctrl-C handling in `serve` is untouched. ✅ |
| **XI** Configuration | `.arc/config.yml` becomes version-controlled. It is a **project-level** config file — already the correct tier in the precedence chain — so no precedence change. `arc` still never touches an ignore file outside `.arc/` (FR-016). ✅ |
| **XII** Docs & errors | Four new `faults.Safe1` sentinels, declared as package constants and wrapped with `.With()` — no ad hoc `fmt.Errorf`. `Short`/`Long`/`Example` updated on both commands; `README.md` updated in the same change. ✅ |
| **XIII** Release | No change to `.goreleaser.yaml` or the build matrix. ✅ |
| **XIV** Versioning | **Additive, minor.** No command renamed, no flag renamed, no `--json` schema altered. Existing `arc serve` invocations behave byte-for-byte as before (SC-004). What changes is the *contents of a commit `arc init` creates* — new graphs only; existing graphs are untouched and keep working (FR-018). No major bump, no deprecation cycle needed. ✅ |

**Gate result: PASS.** No violations; Complexity Tracking is empty.

**Accepted ADRs re-read before planning** (Principle I): ADR 001 (system
architecture), ADR 002 (UX design system), ADR 003 (MCP server adapter). No plan
decision contradicts any of them.

## Project Structure

### Documentation (this feature)

```text
specs/034-serve-dir-public-state/
├── plan.md                    # This file
├── spec.md                    # Feature specification
├── research.md                # Phase 0 — D1..D9
├── data-model.md              # Phase 1 — layout, footprint, InitOpts, sentinels
├── quickstart.md              # Phase 1 — V1..V8 validation scenarios
├── contracts/
│   └── cli-contract.md        # Phase 1 — arc serve / arc init contracts
├── checklists/
│   └── requirements.md        # Spec quality checklist (16/16)
└── tasks.md                   # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
cmd/
├── arc/graph/
│   ├── serve.go               # EDIT  Use/Args/Long/Example + resolveGraphDir
│   └── serve_test.go          # EDIT  US1 scenarios + edge cases
└── arc/ctrl/
    ├── init.go                # EDIT  resolveRepoContext probes <rel>/.arc; Long text
    └── init_test.go           # EDIT  US2/US3 scenarios + FR-017

internal/
├── app/ctrl/
│   ├── kernel/graph.go        # EDIT  InitOpts.StateIgnored
│   └── service/
│       ├── init.go            # EDIT  layout constants, layoutPaths, contentFor,
│       │                      #       Tracked(), guardRepositoryContext R4
│       ├── errors.go          # EDIT  ErrStateIgnored
│       └── init_test.go       # EDIT  layout/footprint/guard unit tests
├── app/graph/service/
│   ├── node.go                # EDIT  EnsureGraph root classification
│   └── errors.go              # EDIT  three ErrGraphDir* sentinels
└── adapter/fsys/
    └── local.go               # EDIT  probeFoldsCase prefers .arc/cache (research D6)

ARCHITECTURE.md                # EDIT  glossary: Arc State Directory, Arc Cache Directory
README.md                      # EDIT  bootstrap prose, folder tree, Usage, migration
specs/CHANGELOG.md             # EDIT  entry for this feature
```

**Structure Decision**: No new package, no new file. Every edit lands in a file that
already exists, in the two command packages named by the scope directive plus the two
use-case packages behind them. `internal/adapter/fsys/local.go` is the single edit
outside that boundary; see Scope Deviations below.

## Implementation Order

Six slices, each independently verifiable. US1 and US2 are both P1 and fully
independent — either can ship alone.

| # | Slice | Files | Delivers |
|---|-------|-------|----------|
| 1 | Graph-root error classification | `graph/service/{node,errors}.go` + tests | research D2; FR-004's three edge cases |
| 2 | **US1** — `arc serve [<dir>]` | `cmd/arc/graph/serve.go` + `serve_test.go` | FR-001..FR-008; SC-001, SC-002, SC-004, SC-007 |
| 3 | **US2/US3** — layout split | `ctrl/service/init.go` + `init_test.go` | FR-009..FR-016; SC-003, SC-005, SC-006 |
| 4 | FR-017 guard | `ctrl/kernel/graph.go`, `ctrl/service/{init,errors}.go`, `cmd/arc/ctrl/init.go` | FR-017 |
| 5 | Probe relocation | `adapter/fsys/local.go` | research D6 |
| 6 | **US4** — docs | `ARCHITECTURE.md`, `README.md`, `specs/CHANGELOG.md`, Cobra `Long` | FR-019..FR-021; SC-008 |

Slice 1 precedes slice 2 because slice 2's acceptance scenarios assert the messages
slice 1 introduces. Slices 3 and 4 are ordered only for convenience — 4 depends on 3
having made `.arc/` tracked, which is what makes the refusal meaningful.

## Scope Deviations

Two places where this plan goes beyond a literal reading of the directive. Both are
consequences of the requested change rather than independent improvements, and both
are called out rather than folded in quietly.

**1. `internal/adapter/fsys/local.go` — `probeFoldsCase` (research D6).** The
function writes a throwaway temp file into `.arc/`, justified in its own comment by
*"already gitignored, so a probe file that somehow outlives a crash is never mistaken
for graph content."* Making `.arc/` tracked invalidates that premise: a leaked probe
file becomes visible in `git status` and committable. The fix is a three-line change
of preference order — `.arc/cache` → `.arc` → root. The fallback chain is required,
not decorative: a clone has no `.arc/cache/` (data-model I6), and without a fallback
the function would return its `safeDefault` of *case-insensitive*, silently changing
node-identity merge behaviour on Linux in every clone.

**2. `internal/app/graph/service` — `EnsureGraph` (research D2).** Shared by `serve`,
`stats` and `apply`. The latter two always pass `filepath.Abs(".")`, which by
construction exists and is a directory, so the new branches are unreachable for them
and no observable behaviour changes outside `serve`.

Not deviations, despite touching files outside `cmd/`: `internal/app/ctrl/service` is
where `arc init`'s behaviour actually lives (`cmd/arc/ctrl/init.go` is thin wiring by
Constitution III), and `kernel.InitOpts` gains one unexported-in-effect bool field on
an internal value struct.

## Risks

| Risk | Mitigation |
|------|-----------|
| A self-ignoring `.arc/cache/.gitignore` cannot be tracked, so it is absent in clones | The load-bearing rule is `cache/` in the **tracked** `.arc/.gitignore`; the `*` file is belt-and-braces (research D3). Invariant I5 holds in every checkout. |
| Existing init unit tests pin `.arc/.gitignore == "*\n"` and the written-path lists | Expected and intentional — those assertions *are* the old layout. Constitution VI's "minimal churn" rule concerns tests changing to chase an implementation, not tests changing because the specified behaviour changed. |
| A user's graph sits in a repo whose `.gitignore` carries `.arc/` | Slice 4's guard R4 refuses before any write, naming the consequence (FR-017). |
| `git check-ignore` behaviour for a path that does not yet exist | It answers on path patterns, not existence; ordering relative to directory creation is not a concern (research D5). |
| Silent breakage for graphs created by earlier versions | Impossible by construction: no code path reads the *content* of `.arc/.gitignore`. `guardIsGraph` stats the `.arc/` directory only (research D7). |

## Complexity Tracking

*No Constitution Check violations. This section is intentionally empty.*

---

## Post-Design Constitution Re-Check

Re-evaluated after `research.md`, `data-model.md`, `contracts/` and `quickstart.md`:

- **No new package, no new file, no new dependency.** Every edit is to an existing file.
- **No port interface changed** — `port.VCS`, `fsys.Store`, `fsys.Mounter` keep their
  exact method sets (contract C4). The one interface addition anywhere is a `bool`
  field on the internal `kernel.InitOpts` struct.
- **No ADR required or contradicted**; the mandatory `ARCHITECTURE.md` glossary update
  is scheduled in slice 6.
- **Error handling is `faults`-only** — four new `Safe1` constants, `.With()` at every
  site, no `fmt.Errorf` wrapping.
- **E2E coverage is 1:1 with spec scenarios** in the existing colocated test files
  (research D8), and `quickstart.md` maps all eight success criteria to V1-V8.
- **Additive CLI surface**; Constitution XIV needs no major bump.

**Gate result: PASS.** Ready for `/speckit-tasks`.
