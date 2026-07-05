# Implementation Plan: Node Provenance Timestamps (`published`/`indexed`/`updated`)

**Branch**: `009-node-timestamp-attrs` | **Date**: 2026-07-05 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/009-node-timestamp-attrs/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Promote `published` to a typed `internal/core.Node.Published time.Time` field ("de-facto core standard attribute," per the user's explicit instruction) — extracted out of the front-matter manifest by `ParseNode`/`parsePatchBody` and rendered back in by `RenderNode`/`RenderPatch` (`renderAttrYAML` gains a `published` parameter) at exactly the sorted-attribute position it would have occupied as an ordinary `Attrs` key, so on-disk shape is unaffected. `internal/core.Merge`'s shared `mergeCore` helper gains one line filling `Published` from `incoming` only when `existing.Published` is zero, never flagging it as a conflict (`MergeNone`'s existing early return already leaves it untouched). `internal/app/graph/service.Apply` captures one Application Timestamp (`time.Now().UTC()`, RFC 3339, matching `service/subgraph.go`'s existing convention) per invocation and, in its per-node loop: on **create**, sets `Published`/`Attrs["indexed"]` unless the node is a stub (`isStub`, matching the exact zero-beyond-`ID`/`Kind` shape `arc subgraph --stubs` already emits, spec 007 FR-017); on **merge**, renders `existing`/`merged` via `core.RenderNode` and compares bytes (`nodeContentChanged`) to decide whether the merge actually changed anything before stamping `Attrs["updated"]` — one mechanism that correctly handles `MergeNone`'s no-op and any other op's own no-op re-contribution uniformly, with no per-op special-casing. `_schema/nodes`/`_schema/predicates` documents need no exemption code at all: they are written exclusively through `internal/app/schema/service.RegisterKind`/`RegisterPredicate`, a code path this feature's create/merge loop never reaches. No exported signature changes anywhere (`Apply`, `component.Apply`, `kernel.ApplyResult`, `port.VCS`/`port.SchemaRegistry` are all untouched) — this is entirely an `internal/core` + `internal/app/graph/service` change.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: No new dependency. Reuses `github.com/yuin/goldmark`/`goldmark-meta` and `gopkg.in/yaml.v3` (already `internal/core`'s codec dependencies, unchanged) and stdlib `time`/`bytes`.

**Storage**: The mounted graph root, accessed exclusively through the existing, unmodified `internal/adapter/fsys` `Store`/`Mounter` — no new I/O path, no change to how files are read/written, only to the `core.Node` values `service.Apply` constructs before calling the existing `writeNode`.

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII). Table-driven unit tests extend `internal/core/ast_test.go` (new `Node.Published` zero/non-zero cases), `internal/core/markdown_test.go` (`ParseNode`/`RenderNode`/`RenderPatch` round-trip a `Published` value; a `"published"` key never leaks into `Attrs`), and `internal/core/merge_test.go` (15 existing cases — new cases: `Published` fills from `incoming` when `existing.Published` is zero, for every non-`none` op; is preserved unchanged when `existing.Published` is already non-zero even if `incoming.Published` differs; `MergeNone` leaves `Published` untouched, matching its existing whole-node no-op). `internal/app/graph/service/apply_test.go` (16 existing cases — new cases: a stub-shaped node section creates a file with neither `Published` nor `indexed`; a `_schema/`-registering patch section leaves the registered schema document with none of the three attributes; a no-op re-contribution of a `union`/`union-first-writer`/`append` kind adds no `updated`; `MergeNone`'s re-contribution adds no `updated`, matching spec 003 FR-007's existing guarantee; every node created by one `Apply` call shares one `indexed` value; every node actually merged by one `Apply` call shares that same value under `updated`). `cmd/arc/graph/apply_test.go` (29 existing E2E cases via `sut()` — new cases map 1:1 to spec.md's acceptance scenarios: US1 scenarios 1-4, US2 scenarios 1-4, US3 scenario 3 (`arc subgraph` preserving `published`, already covered by `cmd/arc/graph/subgraph_test.go`'s existing round-trip infrastructure, extended with a `Published`-bearing fixture node)).

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`; no platform-specific code introduced.

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — no new `internal/app/<domain>` use-case, no new `cmd/` package; this feature extends the existing `internal/core` domain and the existing `internal/app/graph` use-case's `Apply`.

**Performance Goals**: No measurable change — one extra `time.Now()` call, one extra map-key branch per node, and (for merges only) one extra pair of `RenderNode` calls already cheap enough that `service.Apply`'s own `writeNode` performs an equivalent render on every node regardless.

**Constraints**: `published`, once non-zero on a node, MUST NOT be overwritten by a later patch (spec FR-010); `indexed` MUST NOT be modified by any later merge (spec FR-006); `updated` MUST be set if and only if the merge actually changed the node's rendered content, byte-for-byte (spec FR-007/FR-008); a stub node and a `_schema/` document MUST carry none of the three attributes (spec FR-002/FR-003); no exported signature of `Apply`/`component.Apply`/`kernel.ApplyResult` may change (spec is scoped to node content only).

**Scale/Scope**: Three files touched in `internal/core` (`ast.go`, `markdown.go`, `merge.go`), one file touched in `internal/app/graph/service` (`apply.go`); zero new packages, zero new commands, zero new ports/adapters, zero new external dependencies.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Glossary MUST gain entries for the new domain vocabulary this feature introduces (Provenance Timestamp Attributes / `published`, `indexed`, `updated`; Application Timestamp) in the same PR, per Principle II below. No ADR is superseded or contradicted — this feature extends `internal/core.Node`/`Merge` and `internal/app/graph/service.Apply` in place, following ADR 001's existing `internal/core` (shared domain) vs. `internal/app/<use-case>` (service logic) split exactly as `specs/003-apply-patch` established it; no new package, so no new entry in ARCHITECTURE.md's Directory Structure tree is needed, only Glossary. `tasks.md` MUST include this glossary-update task. |
| II — DDD & Glossary | Yes | PASS, with the same obligation as above — `published`/`indexed`/`updated` and "Application Timestamp" are new, user-facing domain concepts (they appear in every node file a user reads) and belong in the Glossary alongside existing entries like "Timeline Entry"/"Merge Behavior". |
| III — Hexagonal Architecture | Yes | PASS — all new logic lives in `internal/core` (shared, use-case-independent domain, unchanged import direction) and `internal/app/graph/service` (existing use-case's own service package); no `cmd/` change at all, since no CLI-visible behavior changes (flags, help text, `--json` schema); `internal/core.Merge`/`mergeCore`/`mergePublished` remain pure functions with no I/O, consistent with the existing contract. |
| IV — Functional Programming Style | Yes | PASS — `mergePublished`, `isStub`, `nodeContentChanged`, `setAttr` are small, single-purpose, side-effect-scoped-to-the-call functions; no inline comments beyond existing GoDoc conventions; `nodeContentChanged`'s only "side effect" (calling `core.RenderNode` twice) is calling an already-pure function, not performing I/O itself. |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — `indexed`/`updated` deliberately stay plain `Attrs` strings rather than gaining symmetric typed `Node` fields (research.md D4), since `internal/core.Merge` has no merge semantics to apply to them; no injected `Clock` interface is introduced where the existing codebase (`service/subgraph.go`) already calls `time.Now()` directly for an equivalent per-invocation timestamp (research.md D5) — matching precedent rather than adding a speculative test seam nothing else in the codebase needs. |
| VI — TDD | Yes | PASS — new/extended table-driven cases in `ast_test.go`/`markdown_test.go`/`merge_test.go`/`apply_test.go` (service) are written first per constitution Principle VI, using `github.com/fogfish/it/v2` exclusively; each new case (Constitution Check's Testing section above) must compile and fail semantically (missing `Published` field / missing `indexed`/`updated` stamping) before implementation. |
| VII — External Integration & Adapter Consistency | Yes | PASS — no new external integration; the only I/O this feature touches (`fsys.Store` reads/writes via `writeNode`/`readExistingNode`) is unchanged, still exclusively through the existing `internal/adapter/fsys`. |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's three user stories / 11 acceptance scenarios map 1:1 onto new/extended cases in `cmd/arc/graph/apply_test.go` (create/merge/stub/schema/no-op scenarios) and `cmd/arc/graph/subgraph_test.go` (US3 scenario 3, `published` surviving export). |
| IX — CLIG/Cobra (ADR 002) | No | N/A — no command, flag, help text, or output-mode change; `arc apply`'s CLI surface is byte-for-byte identical before and after this feature. |
| X — Terminal Output, Color & Interactivity | No | N/A — no Reporter phase added or changed; the existing `labelApplyingNodes` phase and its per-node `reporter.Step` calls are reused unmodified (research.md D9 — only the `outcome` string's *content* is unaffected, since this feature does not touch it, per D9's explicit scope boundary). |
| XI — Configuration, Environment Variables & Secrets | No | N/A — no configuration surface touched. |
| XII — Documentation & Help System | No | N/A — no help text change (no flag/command changed); no new error type introduced (this feature has no new failure mode — `RenderNode`/`ParseNode` already handle every value this feature adds through their existing, already-`faults`-wrapped error paths in `service.Apply`). |
| XIII — Distribution & Release Engineering | No | N/A — no release pipeline change. |
| XIV — Versioning/Security | Yes | PASS — `Node.Published`'s new `json:"published,omitempty"` field is purely additive to `kernel.SubgraphResult.Patch.Nodes`'s existing `--json` schema (no field removed/retyped, no existing consumer broken); flagged, not silently accepted, that `omitempty` is a no-op for a zero `time.Time` (research.md D10) — a cosmetic gap with no consumer yet affected, not a breaking change. |

No Complexity Tracking entries — every deviation considered (D4's decision not to add typed `Indexed`/`Updated` fields, D5's decision not to inject a `Clock`) is a *simplicity* choice in the constitution's favor (Principle V), not a violation requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/009-node-timestamp-attrs/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — D1-D11 design decisions
├── data-model.md         # Phase 1 output — Node.Published, Provenance Timestamp Attributes,
│                          #   Application Timestamp, new helper function signatures
├── quickstart.md         # Phase 1 output — 3 runnable scenarios, one per user story
├── contracts/            # Phase 1 output
│   ├── ast-contract.md    # delta over specs/003-apply-patch/contracts/ast-contract.md
│   └── apply-contract.md  # delta over service.Apply's behavior contract
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                        # existing package — no new files
│   ├── ast.go                     # + Published time.Time field on Node (json:"published,omitempty")
│   ├── ast_test.go                 # + Published zero/non-zero cases
│   ├── markdown.go                # + extractPublished(manifest) (time.Time, map[string]any);
│   │                                #   ParseNode/parsePatchBody call it instead of leaving
│   │                                #   "published" in Attrs; renderAttrYAML gains a `published
│   │                                #   time.Time` parameter, merged into its existing sorted-keys
│   │                                #   render loop; renderFrontMatter/RenderPatch's per-node fence
│   │                                #   both pass n.Published through
│   ├── markdown_test.go            # + ParseNode/RenderNode/RenderPatch round-trip cases for Published
│   ├── merge.go                   # + mergePublished(existing, incoming time.Time) time.Time;
│   │                                #   mergeCore sets merged.Published = mergePublished(...) —
│   │                                #   MergeNone's existing early return needs no change
│   └── merge_test.go               # + Published fill-when-zero / preserve-when-set cases per op
│
└── app/
    └── graph/
        └── service/
            ├── apply.go            # + appliedAt/stamp captured once near the top; + isStub(node),
            │                        #   nodeContentChanged(existing, merged), setAttr(attrs, k, v);
            │                        #   per-node loop: create path fills Published/indexed unless
            │                        #   isStub; merge path stamps updated only when
            │                        #   nodeContentChanged is true
            └── apply_test.go       # + create/merge/stub/schema/no-op-merge cases

cmd/
└── arc/
    └── graph/
        ├── apply_test.go          # + E2E cases, one per spec.md acceptance scenario, via sut()
        └── subgraph_test.go       # + one case: a Published-bearing fixture node's value survives
                                     #   arc subgraph's extraction unchanged (spec FR-011)

ARCHITECTURE.md                    # + Glossary entries: Provenance Timestamp Attributes
                                     #   (published/indexed/updated), Application Timestamp
                                     #   (Principle I/II obligation above)
```

**Structure Decision**: No new command, no new `internal/app/<domain>` use-case, no new port/adapter — this feature is a targeted extension of two already-existing packages: `internal/core` (the `Node` type and its codec/merge functions) and `internal/app/graph/service` (`Apply`'s per-node loop). `cmd/arc/graph/apply.go`, `internal/app/graph/component.go`, `internal/app/graph/kernel/apply.go`, `internal/app/graph/port/*.go`, and every `internal/app/schema/*` file are all untouched.

## Complexity Tracking

*Empty — no Constitution Check violations to justify (see the "No Complexity Tracking entries" note above the Project Structure section).*
