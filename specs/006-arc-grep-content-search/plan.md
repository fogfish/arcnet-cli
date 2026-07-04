# Implementation Plan: Search Graph Content by Pattern (`arc grep`)

**Branch**: `006-arc-grep-content-search` | **Date**: 2026-07-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-arc-grep-content-search/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Implement `arc grep [<filter>] <pattern>`: scan node file content across the graph for lines matching a regexp `<pattern>`, optionally narrowed by the same `--kind`/`--tag`/`--attr` filter syntax VISION.md's Filtering section defines for every filtering command, and print one line per match — `<kind>  <id>  <line>  <text>` — suitable for piping to standard tools. Per the user's explicit instruction, `arc grep` extends the **existing** `internal/app/graph`/`cmd/arc/graph` domain (a new `Grep` alongside the existing `Apply`), not a new domain — `cmd/arc/graph/apply.go`'s own `PostRunE` hint already anticipates this exact command. The performance-critical content scan is factored into a new, reusable, dependency-free library, **`internal/pkg/grep`** (ADR 001's own documented phase-2 "generic domain library" tier, its first occupant): a bounded-worker-pool (default 8, configurable via `.arc/config.yml`), concurrently-walked, `bufio.Reader`+`sync.Pool`-buffered, extension-filtered (`.md` default) line scanner that classifies each pattern as literal (`bytes.Contains`) or regex once, up front, and treats every file as plain text with zero awareness of node/YAML structure. Because constitution Principle VII confines all `os.*` filesystem calls to `internal/adapter/fsys`, `internal/pkg/grep` is built entirely on stdlib `io/fs` (`fs.FS`/`fs.ReadDirFS`) and receives the already-mounted `fsys.Store` directly — no adapter, no new dependency, no Principle VII exception. A new, genuinely shared `internal/core.Filter` type (this feature is the first of VISION.md's several Filtering-section commands to ship) implements the Kind/Tag/Attr matching semantics against `core.Node`, consumed by `internal/app/graph/service.Grep` both to build the per-match `kind`/`id` labels every output row needs and to exclude non-matching nodes from the content scan entirely (`grep.Options.Include`). UX follows ADR 002 exactly: DS-04's `Registry[T]{Human, Verbose}` split maps the requested color-highlighting/line-fitting behavior onto existing conventions — both transforms are presentation-only, gated on the same `bios.SCHEMA`-is-color signal DS-05 already resolves once at startup, so piped/`NO_COLOR`/non-TTY output is always the full, untruncated, unstyled line (preserving the "suitable for piping" contract) while an interactive terminal gets a highlighted, width-fitted view; `.arc/config.yml`'s previously-dormant, empty `Config` struct gains its first real fields (`grep.workers`, `grep.maxLineWidth`) for the two configurable knobs the user named. No new port is needed (`arc grep` touches neither git nor the kind/predicate registry). Exit-code signaling reuses `arc lint`'s existing `bios.ErrSilent` two-way convention exactly (zero matches vs. a genuine refusal) rather than a novel three-way split — a correction applied to spec.md's own first-drafted Assumptions during this planning phase (research.md D12), per constitution Principle I's rule that a conflict between a spec artifact and established, binding convention must be resolved explicitly, not silently diverged from.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`, `gopkg.in/yaml.v3` (extending the existing `internal/app/config.Load`/`Save` round-trip) — all existing, no new third-party dependency added by this feature; `internal/pkg/grep` itself adds **zero** dependencies beyond the Go standard library (`io/fs`, `bufio`, `bytes`, `regexp`, `sync`, `sort`, `path`, `context`), per research.md D3's decision not to add `golang.org/x/sync/errgroup` for a problem stdlib already solves

**Storage**: The mounted graph root, read exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` (no changes to that package's own contract) — `arc grep` performs **zero writes**, the second strictly read-only command in the codebase after `arc lint` (spec FR-010)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/graph/grep_test.go` via the existing `sut()`/`chdir()`/fixture-graph helpers `cmd/arc/graph/apply_test.go` (and `cmd/arc/lint/lint_test.go`) already establish, one per spec.md acceptance scenario; unit tests for `internal/app/graph/service.Grep` against a fake `fsys.Mounter`/`fsys.Store` (no VCS mock needed — research.md D13, no port); unit tests for `internal/core.Filter.Match` covering Kind OR / Tag AND / Attr exact+pattern AND semantics against table-driven `core.Node` fixtures; unit tests for `internal/pkg/grep.Search` against an in-memory `fstest.MapFS` (no real filesystem needed, proving the `os`-free, `fs.FS`-only design of research.md D2) covering literal vs. regex dispatch (D4), multi-match-per-line collapsing, unreadable-file continuation (D6), and the bounded-pool/parallel-walk behavior under `-race`

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — extends the third `internal/app/<domain>` use-case (`graph`, alongside `ctrl`, `config`, `schema`, `lint`); adds the codebase's first `internal/pkg/<lib>` occupant

**Performance Goals**: Spec SC-004 — a search across a graph of several thousand nodes completes in under 10 seconds; achievable via `internal/pkg/grep`'s bounded-worker-pool concurrent walk/scan (research.md D3) plus the literal-fast-path dispatch (D4) for the common non-regex case

**Constraints**: Target directory MUST already be an initialized graph (spec FR-011, same `guardIsGraph` guard `arc apply`/`arc lint` already use); `<pattern>` MUST be validated as a regexp before any file is opened, with zero scanning on failure (spec FR-008); a single file's read failure MUST NOT abort the run (spec FR-012); `arc grep` MUST make no filesystem or git-history changes under any circumstance (spec FR-010); `internal/pkg/grep` MUST NOT call any `os.*` filesystem function directly (constitution Principle VII — research.md D2); highlighting/truncation MUST NOT alter piped output (spec FR-006/FR-007, SC-005 — research.md D11)

**Scale/Scope**: One new bare-verb command (`arc grep`), one new method on the existing `internal/app/graph` use-case (`Grep`, alongside `Apply`), one new shared type (`internal/core.Filter`), one new shared, dependency-free library (`internal/pkg/grep`), one new field on the existing, previously-empty `internal/app/config/kernel.Config` (`Grep GrepConfig`), one new field on `internal/bios.Schema` (`Match lipgloss.Style`) — no changes to `internal/core.Node`/`ParseNode`/`RenderNode`'s existing public contract, no changes to `internal/adapter/fsys`'s public contract, no new port anywhere

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Directory Structure and Glossary sections MUST be updated in the same PR to add `internal/pkg/grep`, `internal/app/graph`'s new `Grep`/`grep.go` members, `internal/core.Filter`, and new glossary terms (Filter, Match, Grep Run — from spec.md/data-model.md Key Entities). `tasks.md` MUST include this task. The spec.md Assumptions correction (research.md D12, exit-code convention) is itself an example of Principle I's "a conflict between a plan/spec and established convention MUST be resolved explicitly, not silently diverged from" — applied during this planning phase, documented in research.md rather than left implicit |
| II — DDD & Glossary | Yes | PASS — new glossary terms (`Filter`, `Match`, `Grep Run`) defined in data-model.md/spec.md Key Entities, copied into ARCHITECTURE.md per the Principle I obligation above; `Filter` is deliberately elevated to `internal/core` (research.md D8) rather than left private to `graph`, precisely because Principle II favors reuse over a second, divergent definition the next Filtering-section command would otherwise invent |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/graph/grep.go` is Cobra wiring only; `internal/app/graph/{kernel,service}` holds domain logic (no new `port/` needed, research.md D13); `internal/pkg/grep` is a use-case-independent library with no Cobra/`cmd/` import and no dependency on `internal/app/*` (ADR 001's phase-2 tier); highlighting/truncation are presentation-only, applied exclusively in `cmd/arc/graph/grep.go`, never in `kernel.Match`/`service.Grep` (research.md D11) |
| IV — Functional Programming Style | Yes | PASS — `core.Filter.Match`, `grep`'s classification/scan functions, and `kernel.Match` construction are pure/side-effect-isolated (I/O confined to `fsys.Store`/`fs.FS` calls); no inline comments |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — no third-party concurrency dependency added for the bounded pool (research.md D3); the filter-flags options struct stays local to `cmd/arc/graph/grep.go` rather than being speculatively promoted to a shared location before a second Filtering-section command exists (research.md D14); the exit-code convention reuses `arc lint`'s existing two-way pattern instead of introducing a novel third value (research.md D12) |
| VI — TDD | Yes | PASS — E2E, service, `core.Filter`, and `internal/pkg/grep` unit tests written first; `internal/pkg/grep`'s tests run against `fstest.MapFS` (stdlib, no real filesystem, no mock needed) and under `-race` to validate the bounded-pool concurrency design (research.md D3) |
| VII — External Integration & Adapter Consistency | Yes | PASS, by construction — this feature's central architectural decision (research.md D2) is precisely satisfying this principle: `internal/pkg/grep` is built exclusively on stdlib `io/fs`, never `os.*`, receiving `fsys.Store` (which already implements `fs.FS`/`fs.ReadDirFS`) directly from the caller; no new adapter, no second filesystem-access package |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 3 user stories / 11 acceptance scenarios map 1:1 to E2E tests in `cmd/arc/graph/grep_test.go` |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (`arc grep`, continuing `arc init`/`arc apply`/`arc lint`'s precedent); DS-02 options struct for the new `--kind`/`--tag`/`--attr` local flags (research.md D14); DS-03's reserved shorthands untouched, no new shorthand claimed; DS-04's `Registry[T]{Human, Verbose}` split implements the truncation-vs-full-line distinction by construction |
| X — Terminal Output, Color & Interactivity | Yes | PASS — extends `internal/bios.Schema` with one new field (`Match`), both instances (DS-05), gated through the existing, unmodified `SelectSchema`/TTY-detection call — no second TTY check introduced (research.md D11) |
| XI — Configuration, Env & Secrets | Yes | PASS — extends the existing, previously-dormant `.arc/config.yml` `Config` struct with its first real fields (`grep.workers`, `grep.maxLineWidth`, research.md D10) via the unmodified `internal/app/config.Load`/`Save` round-trip; no new configuration file, no secrets involved |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure (invalid pattern, not-a-graph, malformed `--attr` value) declared as a `faults.Type`/`faults.SafeN` constant in `internal/app/graph/service/errors.go` (extending the existing file), wrapped via `.With()` |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — adds a new, additive `--json` schema (`kernel.GrepResult`); no breaking change to any existing `--json` contract (`kernel.ApplyResult` is untouched); `internal/pkg/grep` adding zero third-party dependencies means no new supply-chain surface for `govulncheck` to track |

**ADR 001 port isolation rule 2** (explicit check, since a port is conspicuously *absent* here): satisfied — `internal/app/graph/service.Grep` needs neither `port.VCS` nor `port.SchemaRegistry` (research.md D13), so none is declared; this is the rule working as intended ("as narrow as the use-case's actual need"), not an oversight.

**ADR 001 domain-evolution phases** (explicit check, since this plan introduces the first `internal/pkg/<lib>` occupant): `internal/pkg/grep` matches phase 2 exactly ("further evolution of core types materializes into a stricter definition of applicability boundaries... a self-contained Go module... constrained to dependencies on open-source modules only") — it is deliberately *not* placed in `internal/core` (it has no dependency on `core.Node`/`Kind` at all, research.md D2/D7) and *not* placed inside `internal/app/graph` (it is meant to be reusable by any future command that needs fast, filtered, plain-text content search over a mounted graph, not a graph-I/O-specific concern).

One entry in Complexity Tracking below (the double-read of a node's bytes in `service.Grep`, research.md D9) — a documented, non-speculative simplicity trade, not a structural violation. The spec.md Assumptions correction (research.md D12) is a spec fix applied during planning, not a Constitution Check violation. No other unresolved conflicts.

## Project Structure

### Documentation (this feature)

```text
specs/006-arc-grep-content-search/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md         # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/            # Phase 1 output
│   └── cli-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go               # + registers graph.NewGrepCmd(); no new persistent flags
    └── graph/                 # existing — gains one new command file
        ├── apply.go             # unchanged
        ├── grep.go              # NEW — package graph: NewGrepCmd() *cobra.Command; local
        │                        #   optsFilter{kind,tag,attr} (DS-02, research.md D14); mounts
        │                        #   the graph, calls internal/app/config.Load then
        │                        #   internal/app/graph.Grep, renders via bios.Registry
        │                        #   (Human/Verbose implements truncate-vs-full-line, DS-04);
        │                        #   bios.ErrSilent on zero matches (DS-07, research.md D12)
        └── grep_test.go         # E2E tests, one per spec.md acceptance scenario, via sut()

internal/
├── core/
│   ├── filter.go              # NEW — Filter{Kinds,Tags,Attrs,AttrPatterns}, Filter.Match(Node)
│   └── filter_test.go         # NEW — unit tests, VISION.md Filtering semantics table-driven
│
├── bios/
│   └── theme.go               # + Schema.Match field, both SCHEMA_PLAIN/SCHEMA_COLOR instances
│
├── pkg/                       # NEW tier — first occupant
│   └── grep/                   # NEW — reusable, dependency-free, fs.FS-based content search
│       ├── grep.go               # Search(ctx, fsys, pattern, Options) (Result, error); bounded
│       │                         #   pool (research.md D3), literal/regex dispatch (D4),
│       │                         #   bufio.Reader + sync.Pool (D5), Options.Include (D7)
│       └── grep_test.go          # unit tests against fstest.MapFS, incl. -race
│
└── app/
    ├── config/
    │   └── kernel/
    │       └── config.go        # + Config.Grep GrepConfig{Workers,MaxLineWidth} (research.md D10)
    │
    └── graph/                  # existing — gains Grep alongside Apply
        ├── component.go          # + Grep(ctx, mounter, filter, pattern, cfg, dir) (kernel.GrepResult, error)
        ├── kernel/
        │   └── grep.go            # NEW — Match, GrepResult (data-model.md)
        └── service/
            ├── grep.go            # NEW — Grep use-case: enumerate+parse nodes (research.md D9),
            │                       #   build Filter-membership + kind/id index, call
            │                       #   internal/pkg/grep.Search, map results into kernel.Match
            ├── grep_test.go        # unit tests against fake fsys.Mounter/Store
            └── errors.go           # existing — + ErrInvalidPattern, ErrInvalidAttrFlag

ARCHITECTURE.md               # + Directory Structure/Glossary updated (Principle I obligation above)
```

**Structure Decision**: This feature extends the project's existing third `internal/app/<domain>` use-case (`graph`) with a second primary-port method (`Grep`, alongside `Apply`), per the user's explicit instruction ("implement arc grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`") — no new `internal/app/<domain>` or `cmd/arc/<domain>` package is created. It introduces the codebase's first `internal/pkg/<lib>` occupant (`internal/pkg/grep`), matching ADR 001's own documented domain-evolution phase 2 exactly, and elevates a new, genuinely shared type (`internal/core.Filter`) into the existing core-domain package rather than accreting it privately. `internal/app/config/kernel.Config` (previously an empty struct with zero callers, per ARCHITECTURE.md) gains its first real fields. `internal/bios.Schema` gains one new field, following DS-05's existing extension pattern exactly. No existing package's public contract changes: `internal/core.Node`/`ParseNode`/`RenderNode`, `internal/app/config.Load`/`Save`, `internal/adapter/fsys`, and `internal/app/graph.Apply` are all consumed/extended without modification to their current behavior.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `service.Grep` reads a matched node's file content twice — once whole-file via `core.ParseNode` (for `kind`/`id` labeling and `Filter` membership), once again streamed via `internal/pkg/grep.Search`'s own `bufio.Reader` (research.md D9) | The output format itself (`<kind> <id> <line> <text>`) requires every match to carry a label that only `core.ParseNode` can produce, for every scanned node regardless of whether a filter is applied — and `internal/pkg/grep` must stay fully decoupled from `core.Node`/YAML to remain a genuinely reusable, domain-agnostic library (research.md D2/D7) | Teaching `internal/pkg/grep` to also parse front-matter so one read serves both purposes was rejected: it would erase exactly the boundary this feature's own architecture depends on — a generic content-search library gaining a graph-node-shape dependency for one caller's labeling need is the "second, divergent implementation" Principle V warns against, not a simplification. The extra read is a single additional sequential pass over files `arc lint` already reads today, well within SC-004's several-thousand-node/10s budget |
