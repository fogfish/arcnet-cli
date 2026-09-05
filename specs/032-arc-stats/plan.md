# Implementation Plan: Graph Statistics (`arc stats`)

**Branch**: `032-arc-stats` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/032-arc-stats/spec.md`

## Summary

`arc stats` reports a graph's shape and health in one read-only pass: total
nodes and edges, a per-type node breakdown, the broken-link count, and
per-year ingestion coverage. `--verbose` adds edges by predicate, monthly
ingestion, connectivity health, degree and hub ranking, schema coverage, and
content volume.

The technical approach is a single `service.Stats` use-case in
`internal/app/graph` that walks the graph once, derives two node populations
from that walk ([research D1](./research.md)), and returns a
`kernel.StatsResult` whose detail section is a nil-able pointer computed only
when asked ([research D4](./research.md)). `cmd/arc/graph/stats.go` is a thin
Cobra wrapper mirroring `arc lint`'s shape. No new dependency, no new adapter,
no change to `internal/bios`.

## Technical Context

**Language/Version**: Go 1.26.6 (per `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/charmbracelet/lipgloss`,
`github.com/fogfish/faults`, `github.com/fogfish/it/v2`. **No new dependency** —
this feature reads files already reachable through `internal/adapter/fsys` and
parses them with `internal/core`.

**Storage**: Local filesystem, read-only, through `internal/adapter/fsys`
(Principle VII — `os.*` file APIs appear nowhere else). No git: unlike
`arc lint`, this command reads no history ([research D10](./research.md)), so it
takes no `port.VCS`.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`. E2E tests
colocated as `cmd/arc/graph/stats_test.go`, driven through `RunE` via the
existing `sut()` helper, fixtures under `cmd/arc/graph/testdata/`
(Principles VI, VIII).

**Target Platform**: linux/darwin/windows, amd64+arm64 (windows/arm64 excluded)
per `.goreleaser.yaml`.

**Project Type**: Single Cobra CLI binary (Principle III).

**Performance Goals**: SC-004 — under 5s for 10,000 nodes, growth no worse than
linear. The design is one filesystem walk plus one parse per node, with all
detail figures derived from the already-parsed in-memory node set; no second
pass over disk.

**Constraints**: Strictly read-only (FR-023, SC-007) — the service never calls
`fsys.Store.Create`/`Remove` and takes no VCS port, so read-only is structural
rather than merely intended. Output must be byte-identical across runs
(FR-021, SC-005), which forces explicit sorting of every map-derived list
([research D9](./research.md)).

**Scale/Scope**: One new use-case function, one new kernel type file, one new
command file, one refactor of an existing shared walker. ~7 acceptance
scenarios across 3 user stories plus a cross-command agreement test.

## Constitution Check

*GATE: evaluated before Phase 0, re-evaluated after Phase 1 design.*

**ADRs read before planning** (Principle I requires this, and requires the ADR
to win any conflict): [ADR 001 system architecture](../../adrs/001-system-architecture.md),
[ADR 002 UX design system](../../adrs/002-ux-design-system.md). ADR 003 (MCP
server adapter) is not touched — this feature adds no MCP tool.

| Principle | Gate | Status |
|---|---|---|
| I — ADRs binding | No plan decision contradicts ADR 001/002 | **PASS** — component/kernel/port/service layout and the `bios.Registry` output schema are both followed as-is |
| II — DDD & glossary | Domain terms modeled, glossary updated | **PASS with action** — `ARCHITECTURE.md` does not exist yet (a pre-existing, constitution-acknowledged gap); glossary additions for *type breakdown*, *stub node*, *ingestion period* are carried in [data-model.md](./data-model.md) so they are ready to land when that file is created. Flagged in Complexity Tracking. |
| III — Hexagonal | `cmd/` holds no logic; domain imports no cobra | **PASS** — `cmd/arc/graph/stats.go` parses flags, calls `graph.Stats`, renders; all counting lives in `service` |
| IV — Functional style | No inline comments, ≤25-line functions, immutability | **PASS by design** — statistics are pure folds over a parsed node slice; the plan decomposes them into one small function per figure group |
| V — SOLID / YAGNI | No catch-all packages, acyclic imports, narrowest interface | **PASS** — the walker is parameterized rather than copy-pasted ([D2](./research.md)); `graph` does **not** import `lint`, agreement is enforced by test instead ([D6](./research.md)) |
| VI — TDD | Tests first, compile, fail semantically | **PASS** — task ordering in `/speckit-tasks` must place every test task before its implementation task |
| VII — Adapters | All file I/O via `internal/adapter/fsys` | **PASS** — reuses `fsys.Mounter`/`fsys.Store`; no new adapter, no new external system |
| VIII — E2E traceability | 1:1 scenario→test, colocated, via `RunE` | **PASS** — 7 spec scenarios map to 7 named tests in [quickstart.md](./quickstart.md) |
| IX — CLIG | No new flags, `NoArgs`, meaningful exit codes | **PASS** — FR-024 pins exit semantics; no positional argument |
| X — Output | stdout/stderr split, `--json`, lipgloss, quiet/verbose | **PASS** — reuses `bios.Registry` and `bios.NewReporter(bios.Quiet, !bios.Verbose)`, identical to `arc lint` |
| XI — Config | XDG, precedence, secrets | **N/A** — no configuration, no secrets |
| XII — Docs & errors | `Short`/`Long`/`Example` populated, `faults` for errors | **PASS** — reuses `service.ErrNotAGraph` for FR-011 rather than minting a parallel error |
| XIII — Release | GoReleaser | **N/A** — no packaging change |
| XIV — Versioning | `--json` schema is a stable contract | **PASS with note** — this is a *new* `--json` schema, so it is additive, not breaking. Recorded in [contracts/stats-json-contract.md](./contracts/stats-json-contract.md) as the v1 baseline. |

**Gate result: PASS.** One item (Principle II / `ARCHITECTURE.md`) is a
pre-existing repository gap rather than something this feature introduces; it
is tracked below rather than silently ignored.

### Post-Phase-1 re-evaluation

Re-checked after `data-model.md` and `contracts/` were written. No gate changed
state. Two design choices were specifically re-examined:

- **Principle V, the walker refactor** — confirmed a pure refactor: existing
  callers keep byte-identical results, and the existing Grep/Subgraph/Match
  tests must pass untouched as the proof (task-level requirement).
- **Principle III, the `lint` agreement test** — the contract test lives in
  `cmd/arc/`, the composition root, where importing both use-cases is already
  normal. No domain-to-domain import is introduced.

## Project Structure

### Documentation (this feature)

```text
specs/032-arc-stats/
├── plan.md              # This file
├── spec.md              # Feature specification (with Clarifications)
├── research.md          # Phase 0 — D1..D10 decisions
├── data-model.md        # Phase 1 — kernel types, derivation rules, glossary
├── quickstart.md        # Phase 1 — scenario→test map and validation guide
├── contracts/
│   ├── stats-json-contract.md   # --json schema (v1 baseline)
│   └── cli-contract.md          # command surface, exit codes, output modes
├── checklists/
│   └── requirements.md  # spec quality checklist (16/16)
└── tasks.md             # Phase 2 — NOT created by /speckit-plan
```

### Source Code (repository root)

```text
cmd/arc/
├── root.go                        # MODIFIED: register graph.NewStatsCmd()
└── graph/
    ├── stats.go                   # NEW: Cobra command, human + verbose printers
    ├── stats_test.go              # NEW: E2E, one test per acceptance scenario
    ├── stats_agreement_test.go    # NEW: SC-003 — Stats vs Lint broken-link parity
    └── testdata/
        └── stats/                 # NEW: fixture graphs (known composition)

internal/app/graph/
├── component.go                   # MODIFIED: add Stats primary port
├── kernel/
│   ├── stats.go                   # NEW: StatsResult, StatsDetail, entry types
│   └── stats_test.go              # NEW: unit tests for derived invariants
└── service/
    ├── grep.go                    # MODIFIED: walkNodeFiles → walkFiles + wrappers
    ├── stats.go                   # NEW: Stats use-case
    └── stats_test.go              # NEW: unit tests against fake fsys.Store
```

**Structure Decision**: The user's direction — statistics in
`internal/app/graph` service, command in `cmd/arc/graph` — matches where this
feature belongs on its own merits: `graph` is already the read-side use-case
owning `Grep`, `Subgraph`, `Match`, `NodeLinks`, and the node walker this
feature extends. No new package is created; `internal/app/stats` would have
duplicated the walker and split the read-side domain in two.

The one point where the user's direction and the spec interact is worth naming:
`arc lint` is the closest behavioural sibling (whole-graph, read-only,
`NoArgs`), but it lives in `cmd/arc/lint`. Placing `stats` in `cmd/arc/graph`
as directed keeps it next to the domain it reads, and costs nothing — `root.go`
registers both as flat top-level commands either way.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Two node populations in one result (D1) | FR-002 counts schema documents; SC-003 requires broken-link parity with `arc lint`, which excludes them | A single population breaks one requirement or the other. FR-006a already commits the report to naming its counting rules, so naming its populations is consistent, not extra surface |
| Broken-link rule restated in `graph/service` rather than shared with `lint` (D6) | `graph` importing `lint` is a sibling-to-sibling dependency Principle V's acyclic rule forbids | Extracting to `internal/core` is premature (YAGNI) — the two sides need different return shapes. A cross-command contract test enforces agreement and fails when *either* side drifts, which shared code would not |
| `ARCHITECTURE.md` glossary additions deferred | The file does not exist; the constitution's own v1.2.0 sync report records creating it as an open follow-up needing real domain content, not a stub | Creating a stub `ARCHITECTURE.md` to host three glossary entries would satisfy the letter of Principle II while making the file's absence harder to notice. Terms are staged in `data-model.md` instead |
