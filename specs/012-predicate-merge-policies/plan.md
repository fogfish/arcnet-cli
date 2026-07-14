# Implementation Plan: Per-Predicate Merge Reconciliation for arc apply

**Branch**: `012-predicate-merge-policies` | **Date**: 2026-07-08 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/012-predicate-merge-policies/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Replace `arc apply`'s whole-node merge dispatch (one `MergeOp` chosen by a node's `@type`, read from `_schema/types/<name>.md`'s `merge` field) with per-predicate dispatch: every predicate present on a merged node reconciles according to its own declared `MergeOp`, looked up in the schema index (spec 011) built from `_schema/predicates/<name>.md`. This requires widening the internal `MergeOp` vocabulary from today's collapsed 5 values to CORE §9.3's genuine 7 (`immutable`/`union`/`firstWriteWin`/`fillIfEmpty`/`lastWriteWin`/`append`/`validatedOverwrite`), replacing `internal/core.Merge`'s whole-node dispatch with a per-key loop over `Attrs`/`Texts`/`Edges`/`HRefs`, and correcting the seed vocabulary in `internal/app/schema/kernel/schema.go` so a freshly initialized graph exhibits the new behavior immediately. `Merge`'s signature changes to accept the schema `Index` directly (already a core-domain type — no new interface, no layering violation) instead of one pre-resolved `MergeOp`. `lastWriteWin` is resolved by application order (git's own last-commit-wins convention), not by a timestamp declared inside each contribution — see research.md D5a for why, and the corresponding spec.md revision made during this planning session. No CLI surface (flags, commands, `--json` schema) changes.

## Technical Context

**Language/Version**: Go 1.26 (`go.mod`)

**Primary Dependencies**: No new dependency. Touches `internal/core` (stdlib `time` only) and `internal/app/{graph,schema}/{kernel,service}` — all existing packages, no adapter/port change.

**Storage**: N/A for this feature specifically (graph files under the mounted directory, via the existing `internal/adapter/fsys`-backed `fsys.Store` — unchanged access pattern, no new I/O).

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). `internal/core/merge_test.go` gets a substantial table-driven rewrite (per `MergeOp` × shape); `internal/app/graph/service/apply_test.go` and `cmd/arc/graph/apply_test.go` get scenario updates/additions mapped to spec.md's revised acceptance scenarios; three files (`internal/core/ast_test.go`, `internal/app/lint/service/lint_test.go`, `internal/app/lint/service/rules_frontmatter_test.go`) need a mechanical rename of retired `MergeOp` constants used only as inert fixture values.

**Target Platform**: Unchanged — linux/darwin/windows amd64+arm64 (`.goreleaser.yaml`); this feature ships inside the existing `arc` binary, no new build target.

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature adds no `cmd/` package; it is a domain/service-layer behavior correction only.

**Performance Goals**: No new goal. Per-predicate dispatch replaces one `switch` on a whole-node op with a loop over the node's own (already-iterated) `Attrs`/`Texts` maps and `Edges`/`HRefs` slices — same asymptotic cost as today's `mergeAttrs`/`mergeTexts`, just keyed per-entry by a map lookup into `index.Predicates` instead of a single pre-resolved flag triple.

**Constraints**: `core.Merge` MUST remain pure (no `context.Context`, no I/O) per ADR 001's domain-layer rule and this function's existing contract. `kernel.ApplyResult`'s shape and `arc apply`'s commit-message format MUST NOT change (contracts/merge-behavior-contract.md).

**Scale/Scope**: Touches ~6 non-test files and ~6 test files (research.md "Summary of blast radius" table); zero new files outside `internal/core`/`internal/app/{graph,schema}` and their tests, plus an `ARCHITECTURE.md` glossary update.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (ADRs binding)**: ADR 001's domain-layer dependency rule (`internal/core` depends only on itself/open-source modules) is satisfied — `core.Index`/`PredicateDef`/`TypeDef` already live in `internal/core/rules.go`; threading `Index` into `core.Merge` introduces no new dependency direction (research.md D1). No ADR conflict found; none needed.
- **Principle II (DDD/Glossary)**: `ARCHITECTURE.md`'s Glossary entries for "Merge Behavior", "Predicate Schema Node", "Source Node"/"Entity/Resource Node" describe the retired whole-node model and must be updated in the same PR (research.md blast-radius table) — tracked as a task, not deferred.
- **Principle III (Hexagonal architecture)**: No `cmd/` change; `internal/core` stays free of Cobra/`cmd` imports; `Merge` stays pure. PASS.
- **Principle IV/V (Functional style, SOLID/YAGNI)**: The per-predicate dispatch collapses to 3 underlying scalar behaviors (research.md D5's "freeze"/"flagOnDiverge"/"alwaysOverwrite" classes) shared via one generic `mergeScalar[T comparable]` function (research.md D3) rather than 7 near-duplicate branches — less code than a naive 1:1 translation, and removes an existing indirection layer (`kernel/schema.go`'s 6 now-redundant local `mergeXxx` aliases, research.md D2). PASS.
- **Principle VI (TDD)**: `internal/core/merge_test.go`'s rewrite and `apply_test.go`/`cmd/arc/graph/apply_test.go`'s new cases are written first, against the data-model.md truth table, before touching `merge.go`/`apply.go` (tasks.md Phase 2d/implementation ordering).
- **Principle VII (Adapters)**: No new external integration; no filesystem/VCS port change.
- **Principle VIII (E2E/spec traceability)**: Every spec.md acceptance scenario (revised User Story 3 included) needs a corresponding case in `cmd/arc/graph/apply_test.go`, colocated, via the existing `sut()`/`RunE` pattern — no new command to wire.
- **Principle IX-XIV**: Not implicated — no flag/command/output-schema/config/release change.
- ~~**Principle X**: Not implicated~~ **Bugfix (BUG-001, 2026-07-08)**: reopened — `--verbose`'s existing per-node-only report (spec 003 FR-021) no longer communicates enough once reconciliation is per-predicate; FR-017 adds a per-predicate report line, and `internal/core.Merge`'s return shape gains a per-predicate outcome trail to source it (no new flag/command, so IX/XI-XIV remain unaffected — only X, terminal output content).

No violations requiring the Complexity Tracking table.

## Project Structure

### Documentation (this feature)

```text
specs/012-predicate-merge-policies/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md         # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── merge-behavior-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                                  # shared domain tier (ADR 001) — no internal/app dependency
│   ├── ast.go                             # MergeOp: 5 → 7 constants (research.md D2)
│   ├── ast_test.go                        # rename retired constants in roundtrip test
│   ├── rules.go                           # PredicateDef/TypeDef/Index — unchanged shape
│   ├── merge.go                           # Merge signature change (D1); per-key dispatch loop (D5); mergePublished folded into generic scalar dispatch (D3); BUG-001: per-predicate outcome trail alongside conflicts
│   └── merge_test.go                      # table-driven rewrite: MergeOp × {Attrs, Texts, Edges} + idempotency/commutativity cases; BUG-001: outcome-trail assertions
│
└── app/
    ├── schema/
    │   ├── kernel/
    │   │   ├── schema.go                  # delete 6 local mergeXxx aliases; CorePredicateDefs/CoreTypeDefs reference new constants directly; BUG-001: every role:"text" predicate repointed to MergeAppend (FR-018)
    │   │   └── schema_test.go             # update any assertion tied to old constant names; BUG-001: assert role:"text" ⇒ MergeAppend
    │   └── service/
    │       ├── schema.go                  # validMergeOps: 5 → 7 values
    │       └── schema_test.go             # update any assertion tied to old constant names
    │
    └── graph/
        └── service/
            ├── apply.go                   # delete whole-node op computation; pass index straight into core.Merge (D4); BUG-001: emit a per-predicate Reporter.Step under --verbose (FR-017)
            └── apply_test.go              # update outcome for tests keyed to old whole-node behavior; add cases per spec's revised acceptance scenarios; BUG-001: assert per-predicate verbose output

cmd/arc/graph/
└── apply_test.go                          # E2E cases 1:1 with spec.md acceptance scenarios (Principle VIII), via existing sut()/RunE pattern

internal/app/lint/service/
├── lint_test.go                           # mechanical MergeOp constant rename (fixture data only)
└── rules_frontmatter_test.go              # mechanical MergeOp constant rename (fixture data only)

ARCHITECTURE.md                            # Glossary: Merge Behavior / Predicate Schema Node / Source Node / Entity-Resource Node entries updated
```

**Structure Decision**: No new command and no new `internal/app/<domain>` use-case. This feature modifies two existing use-cases' service layers (`internal/app/graph/service`, `internal/app/schema/{kernel,service}`) and their shared domain package (`internal/core`), per ADR 001's existing `componentX` layout — nothing new to scaffold.

## Complexity Tracking

*No entries — no Constitution Check violation requires justification.*

**Bugfix**: 2026-07-08 — BUG-001 Updated from bugfix patch. Reopened the Principle X ("not implicated") determination; `internal/core/merge.go`, `internal/app/graph/service/apply.go`, and `internal/app/schema/kernel/schema.go` (plus their test files) are back in scope to add a per-predicate verbose report (FR-017) and repoint every `role: text` predicate to `MergeAppend` (FR-018).

**Bugfix**: 2026-07-12 — BUG-002 Updated from bugfix patch. No architectural or Constitution Check section changes — the plan's own design (`internal/core.Merge` returning a per-predicate outcome trail, `internal/app/graph/service/apply.go` sourcing `--verbose` from it) was correct; only `mergeTexts`/`mergeAttrs`'s `isListMerge` branch inside `internal/core/merge.go` (already in scope per BUG-001 above) needs a follow-up fix so the outcome it reports reflects the actual merge result (FR-019), not just which dispatch class ran.
