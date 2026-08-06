# Implementation Plan: Batch Apply a Directory of Patches (`arc apply batch`)

**Branch**: `020-apply-batch` | **Date**: 2026-08-06 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/020-apply-batch/spec.md`

## Summary

`arc apply batch <dir>` walks a local directory recursively, classifies every
`*.md` file it finds, orders the applicable patches by their manifest
`published` date, and applies each one through the **existing, unmodified**
single-patch algorithm — one commit per patch. Already-tracked documents are
skipped, non-patch Markdown is passed over, and failures are collected and
reported at the end (or halt the run under `--fail-fast`).

Traversal uses stdlib `io/fs` through the existing
`internal/adapter/fsys.Local` mount — **no new dependencies**. `s3://` and
every other remote source is explicitly out of scope for this feature (see
[Deferred: remote patch sources](#deferred-remote-patch-sources)).

The batch orchestration is a new use-case entry point in
`internal/app/graph` with the Cobra wiring in `cmd/arc/graph`;
`internal/app/graph/service.Apply` — the merge/timeline/commit algorithm — is
**not touched at all**.

## Technical Context

**Language/Version**: Go 1.26.5 (matches `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra`,
`github.com/charmbracelet/lipgloss`, `github.com/fogfish/faults`,
`github.com/fogfish/it/v2` — all already present. **This feature adds zero new
modules to `go.mod`.**

**Storage**: Local filesystem only, through `internal/adapter/fsys.Local`
(stdlib `os.DirFS`/`io/fs`). The patch directory is opened **read-only**; the
graph root is mounted separately by the existing `service.Apply`.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; colocated E2E
tests at `cmd/arc/graph/batch_test.go` driven through the `sut()` helper
(Principles VI, VIII). Fixtures under `cmd/arc/graph/testdata/`.

**Target Platform**: linux/darwin/windows, amd64+arm64 per `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary (Principle III)

**Performance Goals**: SC-011 — a 100-patch batch completes unattended with
visible per-patch progress. The dominant cost is one `git commit` subprocess
per applied patch (inherited from `service.Apply`), not traversal or parsing;
discovery over a few thousand files is sub-second.

**Constraints**: Fully offline. Read-only over the patch directory (FR-021).
No partial graph state on failure (FR-012) — inherited from `service.Apply`'s
existing rollback. Deterministic ordering across runs and machines (FR-005).

**Scale/Scope**: One new subcommand, one new use-case entry point, one new
kernel result type. Expected corpus size: tens to low thousands of patches per
directory.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Gate | Verdict | Notes |
|------|---------|-------|
| I — ADRs binding ([ADR 001](../../adrs/001-system-architecture.md), [ADR 002](../../adrs/002-ux-design-system.md)) | **PASS** | Batch is a new use-case entry point under `/internal/app/graph`, driven by one Cobra command under `/cmd` — exactly ADR 001's primary-adapter shape. No new ADR required; no accepted ADR is contradicted. |
| II — DDD & glossary | **PASS** | Reuses existing domain vocabulary (`core.Patch`, `Document`, `Published`). Three new terms — Batch Plan, Patch Outcome, Batch Summary — are additions to `ARCHITECTURE.md`'s glossary, tracked as a task. No new domain type is invented inside `cmd/`. |
| III — Hexagonal boundaries | **PASS** | Discovery, ordering, and iteration are business logic and therefore live in `internal/app/graph/service`, **not** in `cmd/`. `cmd/arc/graph/batch.go` does flag parsing, service invocation, and rendering only. See [Deviation from the "cmd-level only" instruction](#deviation-from-the-cmd-level-only-instruction). |
| IV — Functional style | **PASS** | Discovery/ordering are pure transformations over a file listing; the only side-effecting step is the per-patch `service.Apply` call. Functions kept under 25 lines by splitting discover / classify / order / run. |
| V — SOLID & YAGNI | **PASS** | No abstraction for remote sources is introduced speculatively (that was the S3 decision). Batch reuses `fsys.Store`'s existing read side; no new port. |
| VI — TDD | **PASS (process gate)** | Unit tests for discovery/classification/ordering in `internal/app/graph/service/batch_test.go` and E2E tests written first, compiling and failing semantically before implementation. |
| VII — Adapters & filesystem | **PASS** | All filesystem access goes through the existing `internal/adapter/fsys`. No `os.*` file/directory call is added anywhere outside it. **No third-party filesystem or object-storage library is introduced** — the v1.5.0 mandate is preserved intact. |
| VIII — E2E traceability | **PASS** | Every acceptance scenario in spec.md maps 1:1 to a test in `cmd/arc/graph/batch_test.go`, exercised via `cmd.RunE` through `sut()`. |
| IX — CLIG / Cobra | **PASS** | `arc apply batch <dir>` — one positional subject argument, one long-form flag (`--fail-fast`, no shorthand: `-f` is reserved for file/force). Registered under the existing `apply` command beside `schema`. `Short`/`Long`/`Example` all populated. |
| X — Output & interactivity | **PASS** | Summary to stdout; per-patch progress and warnings to stderr via `bios.Reporter`. `--json` renders `kernel.BatchResult`; `--quiet` suppresses progress; styling via the existing `bios.SCHEMA` lipgloss theme, no raw ANSI. |
| XI — Config & secrets | **N/A** | No configuration, no credentials, no environment variables introduced. |
| XII — Docs & errors | **PASS** | New error conditions declared once as `faults.Type`/`faults.SafeN` constants in the service package; no ad hoc `fmt.Errorf` wrapping. README and command help updated in the same PR. |
| XIII — Distribution | **N/A** | No release-pipeline change; no new build target. |
| XIV — Versioning & compatibility | **PASS** | Purely additive: a new subcommand and a new `--json` schema. No existing command, flag, or output contract changes, so this is a MINOR-level change with no deprecation needed. |

**Result: no violations. [Complexity Tracking](#complexity-tracking) is empty
by design.**

### Deferred: remote patch sources

The original planning input asked for `github.com/fogfish/stream` and an
`s3://` prefix so batch could read patches from S3. That was **dropped by
explicit decision** during planning, for two verified reasons:

1. **It contradicts the constitution.** v1.5.0 deliberately reverted the
   `fogfish/stream` mandate and now requires `internal/adapter/fsys` be built
   *exclusively* on stdlib `io/fs` — "no third-party filesystem or
   object-storage library." Adopting it would need a constitution amendment
   plus a superseding ADR (Principle I), not a plan-level decision.
2. **The cost the revert cited is real and was re-measured.** Adding
   `github.com/fogfish/stream@v1.4.0` pulls **19 AWS SDK v2 modules**
   (`service/s3`, `config`, `credentials`, `sso`, `ssooidc`, `sts`,
   `feature/ec2/imds`, the internal endpoint/checksum/presign trees, and
   `smithy-go`) into a command that touches no cloud API.

Worth recording for whoever picks this up later: only **one** of the two
reasons v1.5.0 gave for the revert applies here. The second — "an S3 object
store cannot host a git-versionable graph, because `arc init`'s contract is
producing a git commit and git needs a real local working tree" — does **not**
apply to reading patches, since the graph would remain local and git-backed
either way. A future feature that wants remote patch bundles has a genuinely
narrower case to argue (read-only *input* sources, not graph storage) and
should argue it in its own ADR.

Two concrete blockers a future remote-source feature must also solve, both
verified against the current code rather than assumed:

- `service.Apply` re-derives the patch directory with
  `filepath.Dir(patchPath)` ([apply.go:474](../../internal/app/graph/service/apply.go#L474))
  and mounts it with the **same** `fsys.Mounter` it used for the graph root —
  so a remote patch source and a local graph cannot coexist behind that single
  parameter without a signature change.
- Both that call and `cmd/arc/graph/apply.go:142`'s `filepath.Abs` mangle a
  URL scheme: `filepath.Dir("s3://bucket/dir/f.md")` returns `"s3:/bucket/dir"`
  (the `//` is collapsed by `filepath.Clean`), and `filepath.Abs` prefixes the
  whole thing with the working directory.

None of this blocks the present feature: **no functional requirement in
spec.md mentions a remote source.** All 22 FRs are satisfiable against a local
directory.

### Deviation from the "cmd-level only" instruction

The planning input also said "the batching changes are only cmd level
concern." That is honoured in the part that matters — **the patching algorithm
is reused verbatim, with `service.Apply` unchanged** — but the *orchestration*
is placed in `internal/app/graph`, not `cmd/`, because Principle III is
explicit that `cmd/` "MUST NOT contain business logic," and ADR 001 makes the
same point for use-cases. Walking a tree, classifying manifests, sorting by
publication date, and accumulating per-patch outcomes is business logic by any
reading.

The practical cost of putting it in the right layer is one new file in
`internal/app/graph/service`, one kernel result type, and a three-line
delegator in `component.go`. The benefit is that batch orchestration becomes
unit-testable without Cobra (Principle III's stated rationale) and that
`--json` output can be rendered from a domain value rather than assembled in
the command handler.

## Project Structure

### Documentation (this feature)

```text
specs/020-apply-batch/
├── plan.md              # This file
├── research.md          # Phase 0 output — decisions D1..D13
├── data-model.md        # Phase 1 output — BatchResult / PatchOutcome
├── quickstart.md        # Phase 1 output — runnable validation scenarios
├── contracts/
│   ├── cli-contract.md      # arc apply batch command surface, exit codes
│   └── batch-result.schema.md  # --json output schema
├── checklists/
│   └── requirements.md  # from /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go                      # MODIFIED: register batch under applyCmd
    └── graph/
        ├── apply.go                 # UNCHANGED
        ├── batch.go                 # NEW: Cobra wiring + human renderer only
        ├── batch_test.go            # NEW: E2E, one test per acceptance scenario
        └── testdata/
            └── batch/               # NEW: patch-directory fixtures

internal/
├── adapter/
│   └── fsys/                        # UNCHANGED — read side already sufficient
└── app/
    └── graph/
        ├── component.go             # MODIFIED: + ApplyBatch delegator
        ├── kernel/
        │   ├── apply.go             # UNCHANGED
        │   └── batch.go             # NEW: BatchResult, PatchOutcome
        └── service/
            ├── apply.go             # UNCHANGED — the algorithm is reused as-is
            ├── batch.go             # NEW: discover / classify / order / run
            └── batch_test.go        # NEW: unit tests for the pure stages
```

**Structure Decision**: The feature adds one Cobra command
(`cmd/arc/graph/batch.go`) driving one new entry point on the **existing**
`graph` use-case (`internal/app/graph`). No new use-case package, no new
adapter, no new port — batch consumes `fsys.Store`'s read side, which already
satisfies `fs.FS`/`fs.ReadDirFS`, and delegates every write to the untouched
`service.Apply`.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified.

No violations. Section intentionally empty.
