# Implementation Plan: Apply a Document Patch to the Graph (`arc apply`)

**Branch**: `003-apply-patch` | **Date**: 2026-07-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-apply-patch/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc apply <patch.md>`: parse a CORE §12 document patch, create-or-merge each H1/H2 node section into the graph per its kind's CORE §10 merge operation, derive and append CORE §9.4 timeline entries, and produce exactly one git commit (CORE §11.3), skipping cleanly when the document is already tracked (CORE §11.2). This is the graph's first *mutation* use-case beyond `arc init`, so it introduces three new architectural layers: **`internal/core`** — the graph AST (ARCNET-AST §4-6) as plain Go types, a `github.com/yuin/goldmark`-backed Markdown↔AST codec confined entirely to this package (which strips `[[wikilink]]` bracket markup out of prose into `HRefs` on parse and safely reconstructs it on render, respecting word-boundary and no-double-linking rules — research.md D3b), the CORE §10 merge algebra, and the CORE §9/§10 kind/merge-rule vocabulary; **`internal/app/graph`** — the graph-mutation use-case (mirrors `internal/app/ctrl`'s `kernel/port/service/component.go` layout, package `graph`, hosted by a matching `cmd/arc/graph` command package); and **`internal/app/config`** — load/save/resolve/default for a new YAML-based `.arc/config.yml`. `arc init` now seeds this file by fetching `github.com/fogfish/arcnet-spec`'s canonical config over HTTP (a new, `config`-private port/adapter, stdlib `net/http`), falling back to the graph format's built-in merge rules (`source: none`, `entity: union`, `resource: union-first-writer`) whenever that fetch fails for any reason — initialization itself never fails on this basis (`specs/002-arc-init/spec.md` FR-017, research.md D5 revised). A graph additionally registers a domain-specific node kind's merge rule by hand-editing this file (spec User Story 3); a patch node whose kind is *not* registered is still applied — using the safe `union` default, with a warning — rather than refused (spec FR-018, revised from the original design after `/speckit-plan`). `internal/app/ctrl` and `internal/app/config` never import each other (research.md D5 — both depend only on the shared `internal/core` constant and are composed at the `cmd/` wiring layer, preserving ADR 001's "use-cases are strictly decoupled" rule). Git access is promoted from `internal/app/ctrl/adapter/git` to the shared `internal/adapter/git` (research.md D4, mirroring `internal/adapter/fsys`'s existing precedent), since two use-cases now need it, gaining one new method (`IsTracked`, CORE §11.2's idempotency check) behind a second, narrower, `graph`-private port interface — the same concrete adapter satisfies both. UX follows ADR 002 exactly as `arc init` already established it (bare-verb grammar, DS-04 output registry, DS-05/06 schema/reporter, DS-07 error handling, DS-12 hints), with one first-time use: `SCHEMA.StatusWarn`/`IconWarn` (defined but unused since `specs/002-arc-init`) now renders the unrecognized-kind warning.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `github.com/fogfish/it/v2` (all existing); **new**: `github.com/yuin/goldmark` + `github.com/yuin/goldmark-meta` (Markdown/front-matter parsing, confined to `internal/core`, research.md D3) and `gopkg.in/yaml.v3` (`.arc/config.yml` and `MergeRuleSet` (de)serialization, research.md D2/D5); the system `git` binary via `os/exec`, unchanged from `specs/002-arc-init` (now behind the promoted shared `internal/adapter/git`, research.md D4); stdlib `net/http` (no new module — `arc init`'s config-seed fetch, `internal/app/config/adapter/http`, research.md D5 revised)

**Storage**: The mounted graph root, accessed exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` (no changes to that package) — never raw `os.*` calls in domain/service code

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/graph/apply_test.go` and updated `cmd/arc/ctrl/init_test.go` via the existing `sut()` helper, one per spec.md/`specs/002-arc-init/spec.md` acceptance scenario (constitution Principles VI, VIII); unit tests for `internal/core` (`ParsePatch`/`ParseNode`/`RenderNode` round-trips, `Merge` per `MergeOp`, `TimelinePeriods`/`TimelineEntry`) with no I/O and no mocks needed (pure functions) — including table-driven cases for `RenderNode`'s inline wikilink reconstruction (research.md D3b): a repeated target name where only one occurrence should link, a target substring embedded mid-word that must NOT be linked, a target immediately preceded by whitespace that must be linked, and a target whose display text already sits inside existing brackets that must not be double-wrapped; unit tests for `internal/app/graph/service` and `internal/app/config/service` against fakes of `fsys.Mounter`/`fsys.Store`, a mock `graph.port.VCS`, and a mock `config.port.Fetcher` (`config.Default`'s fetch-succeeds/fetch-fails/malformed-payload cases — no real network call in `go test`, constitution Principle VI); an integration test for the promoted `internal/adapter/git.Git.IsTracked` against a real `git` binary and `t.TempDir()`

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — second and third `internal/app/<domain>` use-cases (`graph`, `config`) after `ctrl`, and the project's first `internal/core` package

**Performance Goals**: Spec SC-001 — applying a typical single-document patch (parse, create/merge nodes, update timeline, commit) completes in under 5 seconds locally; trivial given no network calls and goldmark's linear-time parsing

**Constraints**: Target directory MUST already be an initialized graph (spec FR-014 — unlike `arc init`, `apply` never creates a graph root); no partial graph state may remain on any failure path (spec FR-015); a patch declaring an unrecognized node kind is applied using the safe `union` default and warns, never refused on this basis alone (spec FR-018, revised); `arc apply` itself remains fully local/offline (spec Assumptions); `arc init`'s config-seed fetch MUST NOT block or fail initialization when the network is unavailable (`specs/002-arc-init/spec.md` FR-017)

**Scale/Scope**: One new bare-verb command (`arc apply`), one new core domain package (`internal/core`) with the AST/parser/merge/timeline/kind-vocabulary, two new use-case packages (`internal/app/graph` with `Apply`; `internal/app/config` with `Resolve`/`Save`/`Default`, plus a config-private `port.Fetcher`/`adapter/http`), one adapter promotion (`internal/app/ctrl/adapter/git` → `internal/adapter/git`, gaining `IsTracked`), one existing-feature signature/behavior touch (`internal/app/ctrl.Init` gains a `configSeed []byte` parameter; `cmd/arc/ctrl/init.go` gains the config-fetch composition and a new `--verbose` step; `specs/002-arc-init/spec.md` gains FR-017), two new external dependencies (`goldmark` + `goldmark-meta`, `yaml.v3`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — `ARCHITECTURE.md`'s Directory Structure and Glossary sections MUST be updated in the same PR to add `internal/core`, `internal/app/graph`, `internal/app/config`, the promoted `internal/adapter/git`, and new glossary terms (Patch, Node Contribution, Merge Behavior, Kind Registration, Ingest Commit — from spec.md Key Entities); `specs/002-arc-init/spec.md` was also amended in this planning pass (FR-017, Assumptions) to keep it accurate against `arc init`'s new config-seed behavior — a cross-feature spec edit, not an `ARCHITECTURE.md` concern, but noted here since Principle I governs keeping documentation and behavior in sync. `tasks.md` MUST include the `ARCHITECTURE.md` task. |
| II — DDD & Glossary | Yes | PASS — new glossary terms defined in data-model.md/spec.md Key Entities, to be copied into `ARCHITECTURE.md` per the Principle I obligation above |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/graph` is Cobra wiring only; `internal/app/graph/{kernel,port,service}` and `internal/app/config/{kernel,port,service}` hold domain logic and ports per ADR 001's `componentX` layout; `internal/core` holds shared, use-case-independent domain logic (AST, merge algebra) per ADR 001's own "core domain" evolution phase; `internal/adapter/git` is ADR 001's shared "phase 2" adapter tier, exactly like `internal/adapter/fsys`; the new HTTP fetch stays use-case-private (`internal/app/config/adapter/http`) since only `config` needs it (research.md D5 revised) |
| IV — Functional Programming Style | Yes | PASS — `internal/core.Merge`/`ParsePatch`/`RenderNode` are pure functions with no side effects; no inline comments; enforced during implementation |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — `graph.port.VCS` is narrower than `ctrl.port.VCS` (no `IsAvailable`/`Init`, since `apply` never bootstraps a repo); goldmark is not wrapped in an injectable port since it has no I/O of its own to fake (research.md D1 rationale); `validated-overwrite`'s "designated field" semantics are deliberately left unimplemented rather than speculatively designed (research.md D6); the config-seed fetch retries zero times, relying on the always-safe fallback rather than speculative retry/backoff complexity (research.md D5 revised) |
| VI — TDD | Yes | PASS — E2E tests and service/core unit tests written first; `internal/adapter/git.IsTracked` integration test uses the real local `git` against `t.TempDir()`, matching `specs/002-arc-init`'s precedent for the rest of that adapter; `config.Default`'s HTTP fetch is isolated behind `port.Fetcher` and exercised only via a mock in unit tests — no real network call in `go test` |
| VII — External Integration & Adapter Consistency | Yes | PASS, with a flagged partial gap — git subprocess access goes through the *same* promoted `internal/adapter/git` adapter both use-cases share (no duplicate git client, research.md D4); all filesystem I/O goes through the existing, unmodified `internal/adapter/fsys`; the new HTTP fetch goes through `port.Fetcher`/`internal/app/config/adapter/http`, never a bare `http.Get` call in service code; goldmark/yaml.v3 remain pure in-process libraries with no external system to port (research.md D1). **Gap**: the fetch's fixed 3s timeout is not overridable by a flag or config value, as this principle otherwise requires — flagged explicitly in research.md D5 (revised) and `contracts/config-contract.md`, not silently accepted; a documented, minor, self-contained gap rather than a blocking violation, since the fallback makes the timeout's exact value low-stakes (a slow-but-eventually-successful fetch and an immediate failure both resolve to the same safe outcome). |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 4 user stories / ~13 acceptance scenarios map 1:1 to E2E tests in `cmd/arc/graph/apply_test.go`; `cmd/arc/ctrl/init_test.go` gains cases for `specs/002-arc-init/spec.md` FR-017 (fetch-succeeds and fetch-fails-fallback paths, via a mock `Fetcher` injected at the wiring layer) |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (research.md D9, continuing `specs/002-arc-init`'s D6 precedent), DS-02 single positional argument, DS-07 `SilenceUsage`/`SilenceErrors` + centralized error formatting, zero new UX decisions needed |
| X — Terminal Output, Color & Interactivity | Yes | PASS — reuses the existing `internal/bios` DS-04/05/06 kernel unchanged; success/skip messages state what changed (or why nothing changed); `SCHEMA.StatusWarn`/`IconWarn` (defined, unused since `specs/002-arc-init`) get their first real caller for the unrecognized-kind warning line (research.md D5 revised) |
| XI — Configuration, Env & Secrets | Yes | PASS — `.arc/config.yml` is a project-local config file, not a secrets file; no XDG applicability (graph-root-scoped, not user/system-scoped); the known "not synced via git" limitation is flagged explicitly (research.md D5), not silently accepted; the config-seed fetch touches a *public, unauthenticated* URL with no secret/credential involved, so no secret-handling rule from this principle applies |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure declared as a `faults.Type`/`faults.SafeN` constant, wrapped via `.With()` (research.md D10, same convention as `specs/002-arc-init` D7); an unrecognized node kind is explicitly *not* modeled as an error (research.md D5 revised, D10) — its warning message is still human-readable and actionable, satisfying this principle's intent for user-facing text without being a `faults` constant |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — extends the existing `--json` contract with a new `ApplyResult` schema, including a new `warnings` field (additive, not breaking); no telemetry introduced; three new dependencies (`goldmark`, `yaml.v3`, and stdlib `net/http` which needs no `go.mod` entry) are permissively licensed / part of the standard library, no transitive vendor lock-in; the config-seed fetch is a one-off, user-visible operational request (not usage telemetry) against a public, versioned spec repository the tool already documents as its own upstream source of truth |

**ADR 001 port isolation rule 1** (explicit check, since this plan deliberately promotes an adapter mid-project): satisfied — `ctrl.port.VCS` and the new `graph.port.VCS` remain two separate, narrow, use-case-private interfaces; only the concrete adapter is shared, exactly as the rule describes for this scenario (research.md D4).

One entry in Complexity Tracking below (Principle VII's non-overridable fetch timeout) — a narrow, deliberately-scoped deviation, not a structural violation. No other unresolved violations.

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
    │   └── init.go             # imports internal/adapter/git (D4); + constructs configadapter.New(),
    │                           #   calls appconfig.Default(ctx, fetcher), marshals to YAML bytes,
    │                           #   passes as appctrl.Init's new configSeed param; + one --verbose step
    │                           #   and a usedFallback note (research.md D5 revised)
    └── graph/                  # NEW — Cobra wiring for the graph (graph I/O) domain
        ├── apply.go             # package graph: NewApplyCmd() *cobra.Command — arg parsing, calls
        │                        #   internal/app/config.Resolve then internal/app/graph.Apply,
        │                        #   renders via bios.Registry, prints Warnings (SCHEMA.StatusWarn),
        │                        #   PostRunE conflict hint
        └── apply_test.go        # E2E tests, one per spec.md US1-US4 acceptance scenario, via sut()

internal/
├── core/                       # NEW — graph AST as core domain (ARCNET-AST §4-6, research.md D1/D2)
│   ├── ast.go                   # Kind, MergeOp, Link, LinkBlock, Node, Patch
│   ├── ast_test.go
│   ├── markdown.go              # ParsePatch, ParseNode, RenderNode — goldmark confined here (D3);
│   │                             #   bracket-strip-on-parse / reinsert-on-render for HRefs (D3b)
│   ├── markdown_test.go
│   ├── merge.go                 # Merge (CORE §10 algebra, D6), conflict marker rendering (D7)
│   ├── merge_test.go
│   ├── timeline.go              # TimelinePeriods, TimelineEntry (CORE §9.4, D8)
│   ├── timeline_test.go
│   ├── rules.go                 # CoreMergeRules, KnownProfileMergeRules, MergeRuleSet (+ Lookup,
│   │                             #   Union), ConfigPath (D5)
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
    ├── ctrl/                    # existing — adapter/git import path (D4) AND component.go/service.Init
    │   │                         #   signature change (D5 revised: + configSeed []byte param)
    │   ├── component.go           # Init(ctx, mounter, vcs, dir, configSeed []byte) (kernel.InitResult, error)
    │   ├── service/init.go        # writeLayout call now uses a per-call copy of DefaultLayout with
    │   │                          #   MetaStubs[core.ConfigPath] = string(configSeed) added; rollback
    │   │                          #   also removes core.ConfigPath on mid-run failure
    │   ├── adapter/git/            # DELETED — moved to internal/adapter/git
    │   └── adapter/mock/           # unchanged, ctrl-private
    │
    ├── config/                  # NEW — .arc/config.yml load/save/resolve/default (research.md D5 revised)
    │   ├── kernel/
    │   │   └── config.go           # Config{MergeRules core.MergeRuleSet}
    │   ├── port/
    │   │   └── fetcher.go          # Fetcher: Fetch(ctx, url) ([]byte, error) — config-private
    │   ├── adapter/
    │   │   ├── http/
    │   │   │   ├── client.go        # net/http-backed Fetcher, 3s timeout, no retries
    │   │   │   └── client_test.go   # integration test, real HTTP against httptest.Server
    │   │   └── mock/
    │   │       └── mock.go          # fake Fetcher for Default's unit tests
    │   ├── service/
    │   │   ├── config.go           # Load/Save/Resolve against fsys.Store; Default against port.Fetcher
    │   │   ├── errors.go           # ErrConfigMalformed, ErrConfigConflict
    │   │   └── config_test.go
    │   ├── README.md
    │   └── component.go            # primary port: Resolve(store) (core.MergeRuleSet, error),
    │                               #   Save(store, kernel.Config) error,
    │                               #   Default(ctx, fetcher) (kernel.Config, usedFallback bool)
    │
    └── graph/                   # NEW — graph I/O / graph-mutation domain (research.md D1)
        ├── kernel/
        │   ├── apply.go            # ApplyResult (+ Warnings []string, D5 revised)
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

**Structure Decision**: This feature adds the project's first `internal/core` package (shared graph AST + merge algebra + kind vocabulary, no use-case dependency in either direction) and its second and third `internal/app/<domain>` use-cases (`graph`, `config`), following `internal/app/ctrl`'s already-established `kernel/port/adapter/service/component.go` layout exactly. It promotes `internal/app/ctrl/adapter/git` to the shared `internal/adapter/git` (mirroring `internal/adapter/fsys`'s existing precedent) since two use-cases now need git access — flagged as a **Phase 0: Pre-implementation Refactoring** task per the constitution's Task List Requirements (no behavior change to `arc init`'s user-facing contract, submitted as its own PR before the rest of this feature's tasks). The command surface adds `cmd/arc/graph/apply.go`, a sibling package to `cmd/arc/ctrl`, registered into the existing root command. `cmd/arc/ctrl/init.go` changes more substantially than the adapter-promotion alone: it now also constructs `internal/app/config`'s real `Fetcher` and calls `appconfig.Default`, passing the result into a new `configSeed` parameter on `internal/app/ctrl`'s `Init` (research.md D5 revised) — this is deliberate composition at the wiring layer, not a violation of use-case decoupling, since `internal/app/ctrl`'s own packages (`kernel`, `service`, `component.go`) still never import `internal/app/config`; only `cmd/`, which is expected to know about multiple use-cases, does. `internal/core.CoreMergeRules` remains the shared constant both `ctrl` (fallback content) and `config` (`Resolve`'s built-in floor) depend on independently.

## Complexity Tracking

The `internal/app/ctrl/adapter/git` → `internal/adapter/git` promotion is precedented (matches `internal/adapter/fsys`) and explicitly sanctioned by ADR 001 port isolation rule 1 — not a deviation requiring justification, no table entry needed.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle VII: `internal/app/config/adapter/http`'s fetch timeout (fixed 3s) is not overridable by a flag or config value | `.arc/config.yml` does not exist yet at the moment this fetch runs — it is what is being seeded — so there is no config-value channel to read an override from without a chicken-and-egg problem, and `arc init` has no command-local flags today to attach one to | Adding a new `arc init --config-timeout` flag solely to satisfy this one sub-rule was rejected as scope creep the user did not ask for, and low-value given the fallback: a fetch that is merely slow and one that fails outright both resolve to the exact same safe, non-blocking outcome, so the timeout's precise value is low-stakes. A follow-up (e.g. an `ARC_CONFIG_TIMEOUT` environment variable, consistent with Principle XI's env-var convention) is the natural fix if 3s ever proves wrong in practice — noted in research.md D5 (revised) as a flagged, not silently accepted, gap. |
