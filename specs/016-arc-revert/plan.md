# Implementation Plan: Retract a Patch's Contribution from the Graph (`arc revert`)

**Branch**: `016-arc-revert` | **Date**: 2026-07-12 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/016-arc-revert/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add `arc revert <source-id>`, a new command in the existing `internal/app/graph` domain (alongside `apply`/`grep`/`subgraph`), that locates a patch's ingest commit via its `Source-Id:` trailer and retracts its contribution using whichever of two approaches its current eligibility calls for: a plain `git revert` of the ingest commit when nothing has since touched any file it changed (research.md D3/D4), or a per-node reconciliation when it has — removing a node outright if the reverted patch was its sole author (research.md D5/D6, sweeping every backlink including timeline entries, which the existing `cites`-edge parsing already surfaces for free), or, for a node another patch has since enriched, stripping only the reverted patch's own body-text contribution via `git blame` (research.md D7) with a dedicated, self-documenting resolution for the conflict-marker case blame alone cannot attribute (research.md D8). Idempotency is detected by the reverted document's own source-node file existence (Clarifications Session 2026-07-12, research.md D2), reusing `arc apply`'s own existing check rather than a new commit-trailer convention. This is also the first command in the codebase to delete a tracked file, so it introduces (not bypasses) the constitution's binding destructive-operation confirmation gate (research.md D10) as a new, reusable `internal/bios.Confirm` helper.

**Bugfix**: 2026-07-12 — BUG-001 Updated from bugfix patch. Locating the ingest commit (research.md D1) must tolerate more than one historical match for the same `source-id` — the expected result of a prior retract-then-reapply cycle, not a corruption of the one-ingest-commit-per-apply invariant — and act on the newest match rather than refusing (spec.md FR-020).

## Technical Context

**Language/Version**: Go 1.26.5 (`go.mod`)

**Primary Dependencies**: No new third-party dependency. Touches `internal/core` (read-only reuse of `RenderNode`/`ParseNode`/`Merge`'s existing `conflictMarker` shape — no signature change), `internal/adapter/git` (five new `os/exec`-backed methods on the existing shared `VCS` type), and `internal/app/graph/{port,service,kernel,component.go,adapter/mock}` (all existing packages). `internal/bios` gains one new file (`confirm.go`) for the new confirmation primitive (research.md D10).

**Storage**: Local files under the mounted graph directory, via the existing `internal/adapter/fsys`-backed `fsys.Store` — same access pattern as `apply`/`subgraph` (`store.Remove` for node deletion, already used by `apply.go`'s own rollback path).

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). New: `internal/app/graph/service/revert_test.go` (unit, table-driven per research.md D3-D9's decision branches), `cmd/arc/graph/revert_test.go` (E2E, 1:1 with spec.md's acceptance scenarios via the existing `sut()`/`RunE` pattern), `internal/adapter/git/git_test.go` additions for the five new subprocess-backed methods, `internal/bios/confirm_test.go` (TTY-gated prompt behavior). Widened: `internal/app/graph/adapter/mock/mock.go`'s `VCS` fake (research.md D11).

**Target Platform**: Unchanged — linux/darwin/windows amd64+arm64 (`.goreleaser.yaml`); ships inside the existing `arc` binary.

**Project Type**: Single Cobra CLI binary (constitution Principle III) — one new command inside an existing domain package, no new top-level project.

**Performance Goals**: No explicit numeric target (spec.md deliberately sets none, consistent with `arc apply`'s own precedent of a soft "typical single-document" expectation rather than a hard SLA). `CommitsTouching`/`Blame`/`ShowFile` are each one `git` subprocess invocation per touched node file — bounded by the number of paths the reverted patch's ingest commit changed, the same order of magnitude `arc apply` already touches per invocation, not by total repository history size.

**Constraints**: `service.Revert` MUST leave no partial state on any failure (FR-016) — mirrors `apply.go`'s existing bounded-rollback precedent (remove any newly-created scratch state, leave a partially-modified pre-existing file at its last fully-written state, recoverable via git). The per-node reconciliation path (research.md D7-D9) MUST NOT write to a node's `Attrs`/`Edges`/`HRefs` under any circumstance (FR-011) — this is a hard behavioral boundary, not a default. `internal/core` stays untouched and pure (no new function signatures) — this feature is additive at the `internal/app/graph`/`internal/adapter/git` layers only.

**Scale/Scope**: New files: `internal/app/graph/service/revert.go` (+`_test.go`), `internal/app/graph/kernel/revert.go`, `cmd/arc/graph/revert.go` (+`_test.go`), `internal/bios/confirm.go` (+`_test.go`). Touched files: `internal/app/graph/port/vcs.go` (+6 methods, +1 type), `internal/adapter/git/git.go` (+5 methods, +4 error sentinels, +`_test.go` cases), `internal/app/graph/adapter/mock/mock.go` (+5 fields/methods), `internal/app/graph/component.go` (+1 delegator), `internal/app/graph/service/apply.go` (+`removeTimelineEntry`, reusing existing `parseTimelineEntries`/`periodGranularity` — no existing function's signature changes), `ARCHITECTURE.md` (Glossary additions).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (ADRs binding)**: ADR 001's port isolation rule (one shared `git.VCS` concrete type, multiple narrow `port.VCS` interfaces) is followed, not deviated from — the six new methods are added to `internal/app/graph/port.VCS` only; `internal/app/ctrl/port.VCS` and `internal/app/lint/port.VCS` are untouched (research.md D1/D11, contracts/vcs-port-contract.md). ADR 002's DS-06 (`Reporter` port) and DS-04 (`Registry`/renderer pattern) are followed exactly as `apply.go`/`applyRenderers` already establish them — `RevertResult` gets its own `bios.Registry[RevertResult]` and `humanRevertPrinter`, no new rendering mechanism. **New pattern, not a deviation**: ADR 002's own CLIG checklist item ("dangerous or irreversible operations require explicit confirmation... Const. IX") and Constitution Principle IX are both binding but have no prior concrete implementation anywhere in this codebase (verified: no `Confirm`/`--yes`/`--force` exists yet) — research.md D10 closes that gap with a new, reusable `internal/bios.Confirm`, which is implementing an already-binding rule for the first time, not diverging from an accepted pattern. No superseding ADR is needed.
- **Principle II (DDD/Glossary)**: `ARCHITECTURE.md`'s Glossary gains "Ingest Commit", "Exclusively-Owned Node", "Shared Node", "Reconciliation Approach" (spec.md's own Key Entities section, data-model.md) in the same PR — tracked as a task, not deferred.
- **Principle III (Hexagonal architecture)**: `cmd/arc/graph/revert.go` contains only flag parsing/`RunE`/output formatting, mirroring `apply.go` exactly. `internal/app/graph/service/revert.go` contains the reconciliation algorithm and has no `cobra`/`cmd` import. `internal/core` is untouched (no new dependency direction). PASS.
- **Principle IV/V (Functional style, SOLID/YAGNI)**: `removeNode`/`reconcileShared`/the eligibility test are small, composed functions (contracts/revert-algorithm-contract.md's pseudocode), each under the 25-line guidance once written as real Go — `reconcileShared`'s Texts-key loop and the D8 marker-resolution branch are natural candidates to factor separately, tracked in tasks.md. No speculative generalization: only the six git primitives this feature's own algorithm needs are added to the port (no "just in case" methods), and `removeTimelineEntry` is written as a minimal sibling to the already-existing `upsertTimelinePeriod`, not a generalized "timeline writer" abstraction neither function currently needs.
- **Principle VI (TDD)**: `internal/app/graph/service/revert_test.go` (table-driven per research.md's decision branches D3/D5/D7/D8/D9), `cmd/arc/graph/revert_test.go` (E2E), and `internal/bios/confirm_test.go` are written first, against contracts/revert-algorithm-contract.md's decision table and data-model.md's `RevertResult` shape, before `revert.go`/`confirm.go` are implemented (tasks.md Phase 2d/implementation ordering).
- **Principle VII (Adapters)**: The five new git primitives extend the *existing* shared `internal/adapter/git.VCS` adapter (contracts/vcs-port-contract.md) — no second git client, no new external system. `internal/bios.Confirm` reads `os.Stdin`/checks TTY directly (mirrors how `internal/bios`'s existing terminal-detection code already touches `os.Stdout` for color/TTY decisions per DS-05/DS-06) — not a "state store" in Principle VII's sense, so no port/adapter split is warranted for it.
- **Principle VIII (E2E/spec traceability)**: Every spec.md acceptance scenario (US1 x3, US2 x2, US3 x4, plus the Clarifications-driven idempotency case) needs a corresponding case in `cmd/arc/graph/revert_test.go`, colocated, via the existing `sut()`/`RunE` pattern.
- **Principle IX (CLIG/flag design)**: `arc revert <source-id>` — one positional "subject" argument, consistent with `arc apply <patch.md>`'s own precedent. New flag `--force`/`-f` (constitution's own reserved-shorthand table names `-f` "file/force" — used here for its "force" sense, not "file", the same dual-use precedent the table itself anticipates). **Destructive-operation confirmation is mandatory here** (Principle IX, research.md D10) — this is the gate this feature must implement, not an exception to request.
- **Principle X (Terminal output)**: `RevertResult`'s human renderer follows `applyRenderers`' exact shape (`bios.SCHEMA.IconOK`, `bios.Registry`). The new confirmation prompt is TTY-gated and automatically suppressed (refuses instead of hanging) when not a TTY and `--force` is absent — DS-06/Principle X's "never leave the user wondering" plus CLIG's non-interactive-safety rule, satisfied together by the same gate.
- **Principle XI-XIV**: Not implicated — no config/secret change, help text follows `apply.go`'s existing `Short`/`Long`/`Example` shape, no release-pipeline change, no `--json`/`--plain` schema exists yet to break (this feature is `RevertResult`'s first version).

No violations requiring the Complexity Tracking table.

## Project Structure

### Documentation (this feature)

```text
specs/016-arc-revert/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md         # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── vcs-port-contract.md
│   └── revert-algorithm-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                                  # unchanged — reused read-only (RenderNode, ParseNode, conflictMarker shape)
│
├── adapter/git/
│   ├── git.go                             # +5 methods (ChangedPaths, CommitsTouching, RevertCommit, Blame, ShowFile) + 4 error sentinels (contracts/vcs-port-contract.md)
│   └── git_test.go                        # +cases for the 5 new methods
│
├── bios/
│   ├── confirm.go                         # NEW — Confirm(prompt string) (bool, error), TTY-gated (research.md D10)
│   └── confirm_test.go                    # NEW — TTY/non-TTY/--force behavior
│
└── app/graph/
    ├── component.go                       # +Revert delegator (mirrors Apply)
    ├── port/
    │   └── vcs.go                         # +6 methods + BlameLine type (contracts/vcs-port-contract.md)
    ├── kernel/
    │   └── revert.go                      # NEW — RevertResult, NodeOutcome (data-model.md)
    ├── adapter/mock/
    │   └── mock.go                        # +5 configurable fields/methods on the fake VCS (research.md D11)
    └── service/
        ├── revert.go                      # NEW — Revert, removeNode, reconcileShared, eligibility test (contracts/revert-algorithm-contract.md)
        ├── revert_test.go                 # NEW — table-driven per research.md D3/D5/D7/D8/D9
        └── apply.go                       # +removeTimelineEntry sibling to upsertTimelinePeriod; enumerateNodes/buildReverseIndex (subgraph.go) reused as-is, no change there

cmd/arc/graph/
├── revert.go                              # NEW — Cobra wiring: `arc revert <source-id> [--force|-f]`, mirrors apply.go
└── revert_test.go                         # NEW — E2E, 1:1 with spec.md acceptance scenarios (Principle VIII), via existing sut()/RunE pattern

ARCHITECTURE.md                            # Glossary: Ingest Commit / Exclusively-Owned Node / Shared Node / Reconciliation Approach entries added
```

**Structure Decision**: One new command (`arc revert`) added to the existing `graph` domain component, matching `apply`'s exact four-tree shape (`cmd/arc/graph/`, `internal/app/graph/{component.go,kernel,port,service,adapter/mock}`) per the plan input's own instruction — no new domain package. One new shared primitive (`internal/bios.Confirm`) added to the existing cross-command UX package, since it is a reusable UX gate (research.md D10), not something specific to the `graph` domain.

## Complexity Tracking

*No entries — no Constitution Check violation requires justification.*
