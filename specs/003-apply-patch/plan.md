# Implementation Plan: Apply a Document Patch to the Graph (`arc apply`)

**Branch**: `003-apply-patch` | **Date**: 2026-07-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-apply-patch/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc apply <patch.md>`: parse a CORE §12 document patch, create-or-merge each H1/H2 node section into the graph per its kind's CORE §10 merge operation, derive and append CORE §9.4 timeline entries, and produce exactly one git commit (CORE §11.3), skipping cleanly when the document is already tracked (CORE §11.2). This is the graph's first *mutation* use-case beyond `arc init`, so it introduces three new architectural layers: **`internal/core`** — the graph AST (ARCNET-AST §4-6) as plain Go types, a `github.com/yuin/goldmark`-backed Markdown↔AST codec confined entirely to this package, the CORE §10 merge algebra, and the CORE §9/§10 kind/merge-rule vocabulary; **`internal/app/graph`** — the graph-mutation use-case (mirrors `internal/app/ctrl`'s `kernel/port/service/component.go` layout, package `graph`, hosted by a matching `cmd/arc/graph` command package); and **`internal/app/config`** — load/save/resolve for a new YAML-based `.arc/config.yml`, which this feature also causes `arc init` to seed with the graph format's built-in merge rules (`source: none`, `entity: union`, `resource: union-first-writer`), giving a graph a way to additionally register a domain-specific node kind's merge rule (spec User Story 3) without any dependency from `internal/app/ctrl` on `internal/app/config` (research.md D5 — both depend only on the shared `internal/core` constant; ADR 001's "use-cases are strictly decoupled" rule is preserved by composing the two at the `cmd/` wiring layer). Git access is promoted from `internal/app/ctrl/adapter/git` to the shared `internal/adapter/git` (research.md D4, mirroring `internal/adapter/fsys`'s existing precedent), since two use-cases now need it, gaining one new method (`IsTracked`, CORE §11.2's idempotency check) behind a second, narrower, `graph`-private port interface — the same concrete adapter satisfies both. UX follows ADR 002 exactly as `arc init` already established it (bare-verb grammar, DS-04 output registry, DS-05/06 schema/reporter, DS-07 error handling, DS-12 hints) with zero new UX decisions required.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `github.com/fogfish/it/v2` (all existing); **new**: `github.com/yuin/goldmark` + `github.com/yuin/goldmark-meta` (Markdown/front-matter parsing, confined to `internal/core`, research.md D3) and `gopkg.in/yaml.v3` (`.arc/config.yml` and `MergeRuleSet` (de)serialization, research.md D2/D5); the system `git` binary via `os/exec`, unchanged from `specs/002-arc-init` (now behind the promoted shared `internal/adapter/git`, research.md D4)

**Storage**: The mounted graph root, accessed exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` (no changes to that package) — never raw `os.*` calls in domain/service code

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/graph/apply_test.go` via the existing `sut()` helper, one per spec.md acceptance scenario (constitution Principles VI, VIII); unit tests for `internal/core` (`ParsePatch`/`ParseNode`/`RenderNode` round-trips, `Merge` per `MergeOp`, `TimelinePeriods`/`TimelineEntry`) with no I/O and no mocks needed (pure functions); unit tests for `internal/app/graph/service` and `internal/app/config/service` against fakes of `fsys.Mounter`/`fsys.Store` and a mock `graph.port.VCS`; an integration test for the promoted `internal/adapter/git.Git.IsTracked` against a real `git` binary and `t.TempDir()`

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — second and third `internal/app/<domain>` use-cases (`graph`, `config`) after `ctrl`, and the project's first `internal/core` package

**Performance Goals**: Spec SC-001 — applying a typical single-document patch (parse, create/merge nodes, update timeline, commit) completes in under 5 seconds locally; trivial given no network calls and goldmark's linear-time parsing

**Constraints**: Target directory MUST already be an initialized graph (spec FR-014 — unlike `arc init`, `apply` never creates a graph root); no partial graph state may remain on any failure path (spec FR-015); a patch declaring an unrecognized node kind refuses the whole application, not just that node (spec FR-018, all-or-nothing); applying is fully local/offline (spec Assumptions)

**Scale/Scope**: One new bare-verb command (`arc apply`), one new core domain package (`internal/core`) with the AST/parser/merge/timeline/kind-vocabulary, two new use-case packages (`internal/app/graph` with `Apply`; `internal/app/config` with `Resolve`/`Save`), one adapter promotion (`internal/app/ctrl/adapter/git` → `internal/adapter/git`, gaining `IsTracked`), one existing-feature touch (`internal/app/ctrl/kernel.DefaultLayout` gains a `.arc/config.yml` seed entry), two new external dependencies (`goldmark` + `goldmark-meta`, `yaml.v3`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — `ARCHITECTURE.md`'s Directory Structure and Glossary sections MUST be updated in the same PR to add `internal/core`, `internal/app/graph`, `internal/app/config`, the promoted `internal/adapter/git`, and new glossary terms (Patch, Node Contribution, Merge Behavior, Kind Registration, Ingest Commit — from spec.md Key Entities). `tasks.md` MUST include this task. |
| II — DDD & Glossary | Yes | PASS — new glossary terms defined in data-model.md/spec.md Key Entities, to be copied into `ARCHITECTURE.md` per the Principle I obligation above |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/graph` is Cobra wiring only; `internal/app/graph/{kernel,port,service}` and `internal/app/config/{kernel,service}` hold domain logic and ports per ADR 001's `componentX` layout; `internal/core` holds shared, use-case-independent domain logic (AST, merge algebra) per ADR 001's own "core domain" evolution phase; `internal/adapter/git` is ADR 001's shared "phase 2" adapter tier, exactly like `internal/adapter/fsys` |
| IV — Functional Programming Style | Yes | PASS — `internal/core.Merge`/`ParsePatch`/`RenderNode` are pure functions with no side effects; no inline comments; enforced during implementation |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — `graph.port.VCS` is narrower than `ctrl.port.VCS` (no `IsAvailable`/`Init`, since `apply` never bootstraps a repo); goldmark is not wrapped in an injectable port since it has no I/O of its own to fake (research.md D1 rationale); `validated-overwrite`'s "designated field" semantics are deliberately left unimplemented rather than speculatively designed (research.md D6) |
| VI — TDD | Yes | PASS — E2E tests and service/core unit tests written first; `internal/adapter/git.IsTracked` integration test uses the real local `git` against `t.TempDir()`, matching `specs/002-arc-init`'s precedent for the rest of that adapter |
| VII — External Integration & Adapter Consistency | Yes | PASS — git subprocess access goes through the *same* promoted `internal/adapter/git` adapter both use-cases share (no duplicate git client, research.md D4); all filesystem I/O goes through the existing, unmodified `internal/adapter/fsys`; goldmark/yaml.v3 are pure in-process libraries with no external system to port (research.md D1) |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 4 user stories / ~13 acceptance scenarios map 1:1 to E2E tests in `cmd/arc/graph/apply_test.go` |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (research.md D9, continuing `specs/002-arc-init`'s D6 precedent), DS-02 single positional argument, DS-07 `SilenceUsage`/`SilenceErrors` + centralized error formatting, zero new UX decisions needed |
| X — Terminal Output, Color & Interactivity | Yes | PASS — reuses the existing `internal/bios` DS-04/05/06 kernel unchanged; success/skip messages state what changed (or why nothing changed) |
| XI — Configuration, Env & Secrets | Yes | PASS — `.arc/config.yml` is a project-local config file, not a secrets file; no XDG applicability (graph-root-scoped, not user/system-scoped); the known "not synced via git" limitation is flagged explicitly (research.md D5), not silently accepted |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure declared as a `faults.Type`/`faults.SafeN` constant, wrapped via `.With()` (research.md D10, same convention as `specs/002-arc-init` D7) |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — extends the existing `--json` contract with a new `ApplyResult` schema (additive, not breaking); no telemetry introduced; two new dependencies (`goldmark`, `yaml.v3`) are permissively licensed, pure-Go, no transitive vendor lock-in |

**ADR 001 port isolation rule 1** (explicit check, since this plan deliberately promotes an adapter mid-project): satisfied — `ctrl.port.VCS` and the new `graph.port.VCS` remain two separate, narrow, use-case-private interfaces; only the concrete adapter is shared, exactly as the rule describes for this scenario (research.md D4).

No violations requiring justification — Complexity Tracking section is empty.

## Project Structure

### Documentation (this feature)

```text
specs/003-apply-patch/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── cli-contract.md
│   ├── ast-contract.md
│   ├── vcs-port-contract.md
│   └── config-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go               # + registers graph.NewApplyCmd(); DS-03 flags unchanged
    ├── ctrl/
    │   └── init.go             # imports internal/adapter/git instead of internal/app/ctrl/adapter/git;
    │                           #   otherwise unchanged (config.yml seeding is a kernel.DefaultLayout
    │                           #   data change, not a code path change here — research.md D5)
    └── graph/                  # NEW — Cobra wiring for the graph (graph I/O) domain
        ├── apply.go             # package graph: NewApplyCmd() *cobra.Command — arg parsing, calls
        │                        #   internal/app/config.Resolve then internal/app/graph.Apply,
        │                        #   renders via bios.Registry, PostRunE conflict hint
        └── apply_test.go        # E2E tests, one per spec.md US1-US4 acceptance scenario, via sut()

internal/
├── core/                       # NEW — graph AST as core domain (ARCNET-AST §4-6, research.md D1/D2)
│   ├── ast.go                   # Kind, MergeOp, Link, LinkBlock, Node, Patch
│   ├── ast_test.go
│   ├── markdown.go              # ParsePatch, ParseNode, RenderNode — goldmark confined here (D3)
│   ├── markdown_test.go
│   ├── merge.go                 # Merge (CORE §10 algebra, D6), conflict marker rendering (D7)
│   ├── merge_test.go
│   ├── timeline.go              # TimelinePeriods, TimelineEntry (CORE §9.4, D8)
│   ├── timeline_test.go
│   ├── rules.go                 # CoreMergeRules, KnownProfileMergeRules, MergeRuleSet, ConfigPath (D5)
│   └── rules_test.go
│
├── adapter/
│   ├── fsys/                   # unchanged
│   └── git/                    # PROMOTED from internal/app/ctrl/adapter/git (D4)
│       ├── git.go                # os/exec-backed VCS implementation, satisfies ctrl.port.VCS AND
│       │                         #   graph.port.VCS structurally; + new IsTracked method
│       └── git_test.go           # integration test, real git binary, t.TempDir()
│
└── app/
    ├── ctrl/                    # existing — unchanged except the adapter/git import path (D4) and
    │   │                         #   kernel.DefaultLayout gaining one MetaStubs entry (D5)
    │   ├── kernel/graph.go        # + DefaultLayout.MetaStubs[core.ConfigPath] = core.CoreMergeRules.YAML()
    │   ├── adapter/git/            # DELETED — moved to internal/adapter/git
    │   └── adapter/mock/           # unchanged, ctrl-private
    │
    ├── config/                  # NEW — .arc/config.yml load/save/resolve (research.md D5)
    │   ├── kernel/
    │   │   └── config.go           # Config{MergeRules core.MergeRuleSet}
    │   ├── service/
    │   │   ├── config.go           # Load/Save/Resolve against fsys.Store
    │   │   ├── errors.go           # ErrConfigMalformed, ErrConfigConflict
    │   │   └── config_test.go
    │   ├── README.md
    │   └── component.go            # primary port: Resolve(store) (core.MergeRuleSet, error),
    │                               #   Save(store, kernel.Config) error
    │
    └── graph/                   # NEW — graph I/O / graph-mutation domain (research.md D1)
        ├── kernel/
        │   ├── apply.go            # ApplyResult
        │   └── apply_test.go
        ├── port/
        │   └── vcs.go              # VCS: IsTracked, StageAll, Commit (graph-private, D4)
        ├── adapter/
        │   └── mock/
        │       └── mock.go         # fake VCS for service unit tests
        ├── service/
        │   ├── apply.go            # Apply use-case: guards (FR-003/014/018), per-node create/merge via
        │   │                       #   core.Merge, timeline update via core.TimelinePeriods/TimelineEntry,
        │   │                       #   commit via port.VCS
        │   ├── errors.go           # ErrNotAGraph, ErrPatchRead, ErrNodeWrite
        │   └── apply_test.go       # unit tests against adapter/mock + fakes of fsys.Mounter/Store
        ├── README.md
        └── component.go            # primary port: Apply(ctx, mounter, vcs, rules, dir, patchPath)
                                     #   (kernel.ApplyResult, error)

ARCHITECTURE.md                   # + Directory Structure/Glossary updated (Principle I obligation above)
```

**Structure Decision**: This feature adds the project's first `internal/core` package (shared graph AST + merge algebra + kind vocabulary, no use-case dependency in either direction) and its second and third `internal/app/<domain>` use-cases (`graph`, `config`), following `internal/app/ctrl`'s already-established `kernel/port/adapter/service/component.go` layout exactly. It promotes `internal/app/ctrl/adapter/git` to the shared `internal/adapter/git` (mirroring `internal/adapter/fsys`'s existing precedent) since two use-cases now need git access — flagged as a **Phase 0: Pre-implementation Refactoring** task per the constitution's Task List Requirements (no behavior change to `arc init`, submitted as its own PR before the rest of this feature's tasks). The command surface adds `cmd/arc/graph/apply.go`, a sibling package to `cmd/arc/ctrl`, registered into the existing root command; `cmd/arc/ctrl/init.go` itself changes only its git adapter import path. `internal/app/ctrl` and `internal/app/config` never import each other — both depend only on the shared `internal/core.CoreMergeRules` constant (research.md D5), preserving ADR 001's use-case decoupling rule.

## Complexity Tracking

*No entries — Constitution Check has no unresolved violations. The `internal/app/ctrl/adapter/git` → `internal/adapter/git` promotion is precedented (matches `internal/adapter/fsys`) and explicitly sanctioned by ADR 001 port isolation rule 1, not a deviation requiring justification.*
