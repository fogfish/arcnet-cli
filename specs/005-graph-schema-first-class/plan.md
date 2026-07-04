# Implementation Plan: Graph Schema as a First-Class Citizen (`_schema/`)

**Branch**: `005-graph-schema-first-class` | **Date**: 2026-07-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/005-graph-schema-first-class/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Introduce `internal/app/schema` as the graph's fifth `internal/app/<domain>` use-case and the single place in this codebase that isolates ARCNET-CORE's declared vocabulary of node kinds, merge behaviors, and predicates (per the user's explicit instruction). It replaces `_meta/` and `.arc/config.yml`'s merge-rule content with `_schema/nodes/*.md` + `_schema/predicates/*.md` — one versioned, human-readable document per node kind and predicate, each with `id`/`kind: schema` (node-kind documents additionally carry `merge`). `arc init` seeds all 17 core kinds/predicates (ARCNET-CORE §9 kinds, §7.4 predicate vocabulary) purely from built-in Go constants — no network fetch (the retired `config.Default`'s HTTP downloader is not reimplemented here, per the user's plan-input instruction that it "is not relevant anymore"). `arc apply`'s existing per-node loop gains a discovery hook: an unrecognized kind or predicate is registered into `_schema/` in the same commit as the triggering patch, via a new `graph`-private `port.SchemaRegistry` interface satisfied structurally by `schema`'s concrete component (mirroring the existing `port.VCS`-across-three-use-cases precedent). `arc lint` is updated in two small, targeted ways: `_schema/` is excluded from its content walk entirely (satisfying both the "schema docs are exempt from ordinary content rules" and "schema basenames occupy a separate namespace" clarifications by construction), and its predicate registry now comes from `schema.Resolve` instead of parsing `_meta/predicates.md`. `internal/app/config` keeps its `Load`/`Save` infrastructure alive per explicit instruction but loses its merge-rule field and its HTTP fetcher entirely, becoming dormant until a future, unrelated configuration need arrives.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `github.com/fogfish/it/v2`, `github.com/yuin/goldmark`/`goldmark-meta` (via existing `internal/core.ParseNode`/`RenderNode`, unchanged), `gopkg.in/yaml.v3` (unchanged, `internal/core`'s YAML encoding used by `RenderNode`'s front-matter) — all existing; **no new third-party dependency**, and one existing one (`net/http`, via `internal/app/config/adapter/http`) is *removed* (research.md D5, D8)

**Storage**: The mounted graph root, read/written exclusively through the existing, unmodified `internal/adapter/fsys` `Store`/`Mounter` — `internal/app/schema` introduces no new storage technology, only new paths (`_schema/nodes/`, `_schema/predicates/`) within the same store

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; new unit tests for `internal/app/schema/service` (`Seed`, `Resolve` incl. the malformed-doc-skip path, `RegisterKind`/`RegisterPredicate` incl. the no-op-if-present path) against a fake `fsys.Store`; E2E tests colocated at `cmd/arc/ctrl/init_test.go` (Scenario 1/5 additions) and `cmd/arc/graph/apply_test.go` (Scenario 3/4 additions) via the existing `sut()` helper, one per spec.md acceptance scenario (constitution Principles VI, VIII); existing `internal/core/rules_test.go`, `internal/app/config/service/config_test.go`, `internal/app/graph/service/apply_test.go`, `internal/app/lint/service/{lint_test.go,rules_frontmatter_test.go,rules_predicates_test.go}`, and `cmd/arc/ctrl/init_test.go` all require updates to stop referencing the deleted `core.CoreMergeRules`/`core.ConfigPath`/`appconfig.Default`/`appconfig.Resolve`/`_meta`/`predicatesPath` symbols (Phase 0 pre-implementation refactoring scope, research.md D1/D6/D8/D9)

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — fifth `internal/app/<domain>` use-case (`schema`, after `ctrl`, `config`, `graph`, `lint`); **no new Cobra command** — `schema` has no `cmd/arc/schema` package, it is consumed only by `arc init` and `arc apply` (and referenced, read-only, by `arc lint`'s wiring)

**Performance Goals**: Not independently meaningful — `Seed()`/`Resolve()`/`RegisterKind`/`RegisterPredicate` are O(17) to O(few-hundred) file operations at most, dwarfed by `arc init`'s/`arc apply`'s existing git operations; no new performance target beyond the unchanged `specs/002-arc-init`/`specs/003-apply-patch` SLAs

**Constraints**: `internal/app/schema` MUST NOT perform network I/O (research.md D5 — a deliberate simplification, not an oversight); `Resolve` MUST NOT error on a malformed individual schema document, only skip it (spec.md Edge Cases); `RegisterKind`/`RegisterPredicate` MUST NOT overwrite an existing document (spec FR-011); schema-document writes triggered by `arc apply`'s discovery MUST land in the same commit as the triggering patch (spec FR-012)

**Scale/Scope**: One new use-case package with no `port`/`adapter` subdirectory (research.md D2/D5 — no use-case-private external dependency to abstract); one new interface in `internal/app/graph/port` (`SchemaRegistry`); signature changes to `internal/app/ctrl.Init` (`configSeed []byte` → `schemaSeed map[string]string`), `internal/app/graph.Apply` (+`predicates map[string]bool`, +`schema port.SchemaRegistry`), and `internal/app/lint.Lint` (+`predicates map[string]bool`); deletions from `internal/core` (`rules.go` in its entirety) and `internal/app/config` (`port/`, `adapter/http/`, `adapter/mock/`, `service.Default`, `service.Resolve`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Directory Structure and Glossary sections MUST be updated in the same PR (research.md D10): add `internal/app/schema`, rewrite the **Metadata Stub**/**Kind Registration** glossary entries into **Schema Document**/**Node-Kind Schema Document**/**Predicate Schema Document**, update **Canonical Folder**'s folder list. This plan also explicitly raises, rather than silently resolves, the tension between the user's "isolate ALL ARCNET-CORE abstractions... within schema domain" instruction and ADR 001's use-case decoupling rule — resolved by keeping graph-format-wide AST/merge-algebra primitives in `internal/core` (tier-1, non-use-case) and moving only ARCNET-CORE's *declared-defaults* data into `schema` (research.md D1). `tasks.md` MUST include the ARCHITECTURE.md task. |
| II — DDD & Glossary | Yes | PASS — new glossary terms defined in data-model.md, copied into ARCHITECTURE.md per the Principle I obligation above |
| III — Hexagonal Architecture | Yes | PASS — `internal/app/schema/{kernel,service,component.go}` holds domain logic; no `cmd/` package for schema at all (it has no primary adapter of its own, consistent with it not being a directly-invokable use-case — `arc init`/`arc apply` are its only callers, both via `cmd/`); `graph`/`lint`'s services never import `internal/app/schema` (research.md D2) |
| IV — Functional Programming Style | Yes | PASS — `Seed`, `Resolve`, `RegisterKind`, `RegisterPredicate` are small, single-purpose functions; no inline comments; enforced during implementation |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS, with one explicitly user-directed exception — `internal/app/schema` has no `port`/`adapter` subpackage (no external dependency beyond the already-shared `fsys.Store`, research.md D2/D5); `internal/app/config`'s now-fully-unused `Load`/`Save` infrastructure is kept alive only because the user explicitly instructed it, not discovered independently (flagged in Complexity Tracking below, research.md D8) |
| VI — TDD | Yes | PASS — service unit tests and E2E tests written first, one per new/changed spec.md acceptance scenario; `internal/app/schema/service`'s tests use a fake `fsys.Store`, no real filesystem or network access |
| VII — External Integration & Adapter Consistency | Yes | PASS — `internal/app/schema` performs zero external I/O beyond the already-shared `internal/adapter/fsys` (no new adapter introduced); `graph.port.SchemaRegistry` follows ADR 001 port isolation rule 1 exactly (narrow, graph-private, satisfied structurally by schema's concrete component — same technique as the existing three-use-case `port.VCS`/`internal/adapter/git.Git` pairing) |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 3 user stories / 12 acceptance scenarios map to E2E test additions in `cmd/arc/ctrl/init_test.go` and `cmd/arc/graph/apply_test.go`; no new command means no new `cmd/arc/schema_test.go` is needed |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — zero new flags, zero new commands; `arc apply`'s existing `--verbose` output gains one more `Reporter.Step` line class (a registered-kind/predicate discovery note), consistent with DS-04's existing `Registry[T]{Human,Verbose}` split, no new wiring pattern introduced |
| X — Terminal Output, Color & Interactivity | Yes | PASS — reuses the existing `internal/bios` kernel unchanged; `arc apply`'s existing unrecognized-kind warning line is unchanged in shape, schema registration itself produces no new *default*-mode output (a state change with nothing new to report beyond what FR-016's existing created/merged stats already cover) |
| XI — Configuration, Env & Secrets | Yes | PASS — no new configuration surface; `.arc/config.yml`'s remaining `Load`/`Save` infra is unused by any command in this feature (research.md D8), so no precedence/XDG concern arises here |
| XII — Documentation & Help System | Yes | PASS — no command help text changes (no new/changed flags or commands); new expected-failure paths (a malformed schema doc, a write failure mid-registration) reuse the existing `faults.Type`/`faults.SafeN` + `.With()` convention, matching every prior feature |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — `kernel.ApplyResult`'s `--json` schema is additive-only if any field is added for registered-kind/predicate counts (no field removed or renamed); removing `internal/app/config`'s HTTP fetcher removes a dependency, never a breaking change to any `--json`/`--plain` contract |

**ADR 001 port isolation rule 1** (explicit check, since this plan introduces a *fourth* narrow, use-case-private port satisfied by a shared/structural concrete type — following `ctrl.port.VCS`/`graph.port.VCS`/`lint.port.VCS`'s precedent, this time for `graph.port.SchemaRegistry` satisfied by `internal/app/schema`'s own component rather than by `internal/adapter/git.Git`): satisfied — the interface declares only the two methods `graph.Apply` actually calls, and `internal/app/schema` is never imported directly by `internal/app/graph`'s package, only referenced through this port at the `cmd/` wiring layer.

Two entries in Complexity Tracking below (the user-directed exception keeping `internal/app/config`'s infrastructure alive though unused, and the explicit `internal/core` vs. `schema` boundary decision) — both documented, non-speculative, directly traceable to explicit user instructions, not silent scope creep. No other unresolved violations.

## Project Structure

### Documentation (this feature)

```text
specs/005-graph-schema-first-class/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md         # Phase 1 output
├── contracts/            # Phase 1 output
│   └── schema-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── ctrl/
    │   ├── init.go            # CHANGED: appconfig.Default → appschema.Seed(); configSeed []byte →
    │   │                       #   schemaSeed map[string]string param into appctrl.Init
    │   └── init_test.go        # CHANGED: assert _schema/ tree instead of _meta/, no .arc/config.yml seed
    ├── graph/
    │   ├── apply.go            # CHANGED: appconfig.Resolve → appschema.Resolve (rules + predicates);
    │   │                       #   wires internal/app/schema's component as graph.port.SchemaRegistry
    │   │                       #   into appgraph.Apply
    │   └── apply_test.go       # CHANGED: new-kind/new-predicate discovery scenarios (spec US2)
    └── lint/
        └── lint.go             # CHANGED: appconfig.Resolve → appschema.Resolve (rules + predicates)

internal/
├── core/
│   ├── rules.go            # DELETED — content redistributed per research.md D1
│   ├── rules_test.go       # CHANGED — tests for the deleted symbols removed/moved
│   ├── ast.go               # UNCHANGED (Kind, MergeOp, Node, Patch, Link, LinkBlock stay)
│   ├── merge.go             # UNCHANGED (Merge algebra stays)
│   └── markdown.go          # UNCHANGED (ParseNode/RenderNode/ParsePatch stay)
│
├── app/
│   ├── schema/              # NEW — fifth domain use-case (research.md D1/D2/D5)
│   │   ├── kernel/
│   │   │   ├── schema.go       # CoreMergeRules (moved from internal/core), CorePredicates (new,
│   │   │   │                   #   ARCNET-CORE §7.4, research.md D7), SchemaKind = "schema",
│   │   │   │                   #   NodesDir/PredicatesDir path constants
│   │   │   └── schema_test.go
│   │   ├── service/
│   │   │   ├── schema.go       # Seed, Resolve, RegisterKind, RegisterPredicate
│   │   │   ├── errors.go       # ErrSchemaMalformed (unused per-file — skipped, not surfaced, per
│   │   │   │                   #   research.md D2 "Resolve never errors on one bad doc")
│   │   │   └── schema_test.go  # unit tests against a fake fsys.Store, no real disk/network
│   │   ├── README.md
│   │   └── component.go        # primary port: Seed(), Resolve(store), RegisterKind(store, kind),
│   │                            #   RegisterPredicate(store, predicate) — no port/adapter subdir
│   │                            #   (research.md D2/D5: no use-case-private external dependency)
│   │
│   ├── config/               # CHANGED — merge-rule content and HTTP fetcher removed (research.md D8)
│   │   ├── kernel/
│   │   │   └── config.go        # Config struct loses MergeRules field; ConfigPath moves here from
│   │   │                        #   internal/core (research.md D1)
│   │   ├── port/                 # DELETED (Fetcher was its only interface)
│   │   ├── adapter/
│   │   │   ├── http/              # DELETED (the "github downloader")
│   │   │   └── mock/              # DELETED (only existed to test the deleted Fetcher)
│   │   ├── service/
│   │   │   ├── config.go          # Load/Save kept; Resolve/Default deleted
│   │   │   └── config_test.go     # CHANGED — Resolve/Default test cases removed
│   │   └── component.go           # Resolve/Default exports removed; Save/Load kept
│   │
│   ├── ctrl/                 # CHANGED
│   │   ├── kernel/
│   │   │   └── graph.go          # Folders: _meta → _schema/nodes, _schema/predicates;
│   │   │                          #   MetaStubs renamed SeedFiles (research.md D9)
│   │   ├── service/
│   │   │   └── init.go            # configSeed []byte → schemaSeed map[string]string param;
│   │   │                          #   writeLayout/hasStub/rollback logic unchanged, just renamed field
│   │   └── component.go           # Init(...) signature's last param renamed/retyped
│   │
│   ├── graph/                # CHANGED
│   │   ├── port/
│   │   │   └── vcs.go             # + schema.go: SchemaRegistry interface (research.md D3)
│   │   ├── service/
│   │   │   └── apply.go           # discovery hook after rules.Lookup miss + after merged Links/Edges
│   │   │                          #   computed (research.md D3/D4); signature gains predicates,schema
│   │   └── component.go           # Apply(...) signature gains predicates, schema params
│   │
│   └── lint/                 # CHANGED
│       └── service/
│           ├── lint.go            # walkNodeFiles skips "_schema" (research.md D6); Lint signature
│           │                      #   gains predicates param, drops internal parsePredicateRegistry call
│           ├── rules_predicates.go # parsePredicateRegistry + predicatesPath deleted; violation message
│           │                       #   reworded to name _schema/predicates/
│           └── rules_predicates_test.go # CHANGED — fixtures updated for the new predicates param

ARCHITECTURE.md               # + Directory Structure/Glossary updated (Principle I obligation above)
```

**Structure Decision**: This feature adds the project's fifth `internal/app/<domain>` use-case (`schema`), the first to have no `cmd/` package of its own (it is consumed only by `arc init`/`arc apply`'s existing commands, never invoked directly) and the first to have no `port`/`adapter` subdirectory (no use-case-private external dependency beyond the already-shared `internal/adapter/fsys`, research.md D2/D5). It introduces a fourth narrow, use-case-private port satisfied by a shared concrete type (`graph.port.SchemaRegistry`, satisfied by `schema`'s own component — the first time this technique is used for something other than `internal/adapter/git.Git`, still following the identical ADR 001 rule 1 pattern). It deletes `internal/core/rules.go` entirely, redistributing ARCNET-CORE's declared-defaults content into `schema` while keeping the graph-format-wide AST/merge-algebra/codec in `internal/core` unchanged (research.md D1) — the explicit resolution of the tension between the user's "isolate ALL ARCNET-CORE abstractions... within schema domain" instruction and ADR 001's use-case-decoupling rule, raised here rather than silently resolved (constitution Principle I). `internal/app/config` loses its merge-rule field and HTTP fetcher but keeps its `Load`/`Save` infrastructure alive, unused, per explicit instruction (research.md D8).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `internal/app/config`'s `Load`/`Save`/`kernel.Config` are kept fully alive with zero callers anywhere in `cmd/` after this feature ships | Explicit user instruction: "keep the config infrastructure alive... it is not relevant anymore" refers only to the github downloader, not to `Load`/`Save` themselves | Deleting `internal/app/config` entirely (the YAGNI-consistent default absent this instruction) was rejected because the user explicitly asked for the infrastructure to remain available for an unspecified future configuration need — removing it would contradict a direct instruction, not merely tidy up |
| ARCNET-CORE's declared vocabulary (`CoreMergeRules`, `CorePredicates`) moves into `internal/app/schema/kernel`, while the AST types and `Merge` algebra that operate over that vocabulary stay in `internal/core`, splitting what the user described as one instruction ("isolate ALL ARCNET-CORE abstractions... within schema domain") across two packages | ADR 001 forbids a use-case from being imported by another use-case's package (`graph`/`lint` would have to import `internal/app/schema` to call `core.Merge`/`core.RenderNode` if those moved) | Moving everything into `internal/app/schema` literally, as the instruction reads most broadly, was rejected because it is an unresolvable conflict with a currently-Accepted, BINDING ADR (constitution Principle I: "Resolving cross-cutting design tension by ignoring an ADR is FORBIDDEN... the conflict MUST be raised explicitly and resolved via a new ADR, not by quietly diverging") — this plan raises the conflict here and resolves it by scoping "ARCNET-CORE abstractions" to the spec's own actual subject (kind/merge/predicate *defaults*, i.e. what `_schema/` documents), not the graph-format's general-purpose AST/algebra primitives every use-case already depends on |

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Expand the tree below with the concrete packages this
  feature touches (real command/domain names). This project is a single
  Cobra-based Go CLI (constitution Principle III) — there is no web/mobile
  layout to choose between; only the internal package boundaries vary
  per feature.
-->

```text
cmd/
└── <command>/
    ├── <command>.go        # Cobra command: flag parsing, RunE, output formatting only
    └── <command>_test.go   # E2E test(s), one per spec.md acceptance scenario, via sut() (Principle VIII)

internal/
└── <domain>/
    ├── <type>.go           # domain types, port interfaces — no cobra, no cmd/ imports (Principle III)
    ├── <type>_test.go      # unit tests, github.com/fogfish/it/v2 (Principle VI)
    └── adapter/
        └── <adapter>.go    # driven adapter implementing the port (Principle VII)

testdata/                   # fixtures colocated with the E2E test(s) above (Principle VIII)
```

**Structure Decision**: [Name the concrete command(s) and domain package(s)
this feature adds or touches, replacing the `<command>`/`<domain>` placeholders
above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
