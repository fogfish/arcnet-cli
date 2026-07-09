# Implementation Plan: Schema-Driven Link Rendering

**Branch**: `013-predicate-role-rendering` | **Date**: 2026-07-08 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/013-predicate-role-rendering/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Replace `internal/core/markdown.go`'s current always-flat rendering of `Node.Edges` with schema-driven
rendering: `RenderNode`/`RenderPatch`/`renderNodeBody` gain an `Index` parameter (spec 011's already-resolved
schema index, exactly mirroring spec 012's precedent for `core.Merge`), partition `Edges` by each occurrence's
predicate name into `role: edge` (flat bullet, unchanged format) versus `role: link` (grouped under a
`"## Label"` heading, label from the predicate's own schema `label` field, default capitalized predicate
name), and omit that heading entirely when a node's body is exactly one link-role predicate's occurrences
(e.g. `timeline`'s `entries`) with no other predicate present (evaluated by presence in `Edges`, not by the
type's Class-node `Required`/`Optional` permission list — research.md D5 explains why presence-based is both
simpler and the literal, already-ratified spec.md requirement). A predicate absent from the index defaults to
`edge`/flat — the conservative, already-implicit default. This changes zero parsing behavior (spec 010's
unified `Edges` list is untouched, FR-011) and zero merge behavior (spec 012 untouched, FR-012); it changes
only the write path, threaded through every existing `RenderNode`/`RenderPatch` call site (research.md D6),
with two read-only commands (`arc subgraph`, `arc serve`) resolving their `Index` defensively so they keep
working against a bare, schema-less directory exactly as they do today (research.md D7).

**Bugfix note (BUG-001, 2026-07-09)**: The `"## Label"` heading grouping described above governs `RenderNode`
(a standalone graph node file, ARCNET-CORE §5) only. `RenderPatch` (a patch-exchange document) has a distinct,
fixed structure per ARCNET-CORE §14.2 — `H1 = @type`, `H2 = @id`, "Markdown headings are reserved for type and
identity; node bodies use bold labels, never headings" — so `RenderPatch`'s link-role groups MUST render under
a `**Label**` bold-label paragraph instead of a heading. The two formats therefore need two distinct body-
rendering code paths sharing only the role/label/omission decision logic (`resolveRenderRole`/`labelFor`/the
single-group-omission check), not one shared `renderNodeBody`/`renderEdges` producing identical Markdown
markup for both. See spec.md FR-014 (added) and FR-001/FR-004 (scope-clarified).

## Technical Context

**Language/Version**: Go 1.26 (`go.mod`)

**Primary Dependencies**: No new dependency. Touches `internal/core` (stdlib `bytes`/`sort`/`strings` only,
already imported by `markdown.go`) and `internal/app/{schema,graph}/service`, `cmd/arc/graph` — all existing
packages, no adapter/port change.

**Storage**: N/A specifically for this feature (graph files under the mounted directory, via the existing
`internal/adapter/fsys`-backed `fsys.Store` — unchanged access pattern, no new I/O beyond `arc subgraph`/
`arc serve` each gaining one additional `_schema/` read per invocation, research.md D7).

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` (constitution Principles VI, VIII).
`internal/core/markdown_test.go` gets every `RenderNode`/`RenderPatch` call site updated with a new `Index`
argument (~20+ sites) plus two existing tests rewritten to assert schema-driven behavior instead of
always-flat (research.md D8); `internal/app/schema/service/schema_test.go`, `internal/app/graph/service/
apply_test.go`, and `cmd/arc/graph/{subgraph,serve,apply}_test.go` get mechanical call-site updates plus
targeted new cases for the flat/grouped/omission behavior.

**Target Platform**: Unchanged — linux/darwin/windows amd64+arm64 (`.goreleaser.yaml`); ships inside the
existing `arc` binary, no new build target.

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature adds no `cmd/` package;
it corrects rendering behavior inside two existing use-cases' service layers plus the shared core domain.

**Performance Goals**: No new goal. Partitioning `Edges` by role is a single pass plus one small sort over
distinct link-role predicate names present (typically single digits per node) — negligible relative to the
existing per-node Markdown render/parse cost.

**Constraints**: `RenderNode`/`RenderPatch` MUST remain pure (no `context.Context`, no I/O) — `Index` is a
plain value parameter, not a resolved-on-demand dependency, matching `core.Merge`'s existing contract
(ADR 001 domain-layer rule). `arc subgraph`/`arc serve` MUST NOT start requiring `.arc`/`_schema` to exist
where they don't today (research.md D7) — this is a rendering-shape feature, not a new preflight requirement.

**Scale/Scope**: Touches 1 core file (`internal/core/markdown.go`) plus its test file, 4 call-site files
(`internal/app/schema/service/schema.go`, `internal/app/graph/service/apply.go`, `cmd/arc/graph/subgraph.go`,
`cmd/arc/graph/serve.go`) and their tests, plus one `ARCHITECTURE.md` glossary touch-up (research.md D9). No
new files outside `specs/013-predicate-role-rendering/`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (ADRs binding)**: No ADR conflict. `internal/core`'s dependency rule (depends only on itself)
  is preserved — `Index`/`PredicateDef` already live in `internal/core/rules.go`; threading `Index` into
  `RenderNode`/`RenderPatch` is the identical shape spec 012 already used for `core.Merge` (research.md D1).
- **Principle II (DDD/Glossary)**: `ARCHITECTURE.md`'s **Node** glossary entry's `Edges` clause gets one
  clarifying sentence (research.md D9) distinguishing "parsing ignores original grouping" (unchanged, spec
  010) from "rendering now derives grouping from schema `Role`" (this feature) — tracked as a task, not
  deferred. No new domain term is introduced (`Role`/`Label` are already glossary'd via **Predicate Schema
  Node**).
- **Principle III (Hexagonal architecture)**: No `cmd/` business logic added — `cmd/arc/graph/subgraph.go`/
  `serve.go` gain only a resolve-and-pass-through call, matching `cmd/arc/graph/apply.go`'s own existing
  pattern exactly. `RenderNode`/`RenderPatch` stay pure, free functions in `internal/core`. PASS.
- **Principle IV/V (Functional style, SOLID/YAGNI)**: One new small helper (`resolveRenderRole`, mirroring
  `merge.go`'s existing `resolveMergeOp` idiom exactly — research.md D3) plus a straightforward partition/
  group/sort inside `renderNodeBody`; no new type, no new package. The presence-based single-group omission
  (research.md D5) is deliberately simpler than the permission-based (`Required`/`Optional` lookup)
  alternative the `/speckit-plan` input text proposed, and is what spec.md's own resolved Edge Cases actually
  require — simplicity and correctness align here, not trade off. PASS.
- **Principle VI (TDD)**: `internal/core/markdown_test.go`'s two behavior-inverting rewrites and new
  flat/grouped/omission/round-trip cases (research.md D8, quickstart.md Scenarios A-D) are written first,
  against data-model.md's partition table, before touching `renderNodeBody`.
- **Principle VII (Adapters)**: No new external integration; no filesystem/VCS port change.
  `resolveIndexOrDefault` (research.md D7) is a thin, already-`fsys.Store`-typed convenience in
  `cmd/arc/graph`, not a new adapter.
- **Principle VIII (E2E/spec traceability)**: Every spec.md acceptance scenario needs a corresponding case —
  User Story 1/3 primarily via `internal/core/markdown_test.go` (the behavior lives entirely in the pure
  render function); User Story 2's timeline-omission case additionally needs a `cmd/arc/graph/apply_test.go`
  or `subgraph_test.go` case exercising a real `timeline` node end-to-end, via the existing `sut()`/`RunE`
  pattern — no new command to wire.
- **Principle IX-XIV**: Not implicated — no flag/command/output-schema/config/release change. `arc subgraph`/
  `arc serve`'s stdout/MCP-reply *content* changes (grouped headings appear) but their `--json` contracts
  (`kernel.SubgraphResult`) and command surface do not.

No violations requiring the Complexity Tracking table.

## Project Structure

### Documentation (this feature)

```text
specs/013-predicate-role-rendering/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── render-shape-contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── core/                                  # shared domain tier (ADR 001) — no internal/app dependency
│   ├── markdown.go                        # RenderNode/RenderPatch/renderNodeBody gain `index Index`;
│   │                                       # new resolveRenderRole/labelFor helpers; role-partitioned
│   │                                       # render algorithm (research.md D2-D5). BUG-001 (2026-07-09):
│   │                                       # the render algorithm's link-role grouping MUST diverge by
│   │                                       # caller — `## Label` heading for RenderNode (CORE §5),
│   │                                       # `**Label**` bold-label paragraph for RenderPatch (CORE
│   │                                       # §14.2) — a shared body-shape-agnostic helper for the
│   │                                       # role/label/omission decision, plus two distinct rendering
│   │                                       # code paths, not one shared renderNodeBody/renderEdges
│   │                                       # producing identical markup for both
│   └── markdown_test.go                   # every RenderNode/RenderPatch call site gains an Index arg;
│                                           # TestRenderNodeEdgesFlatBulletedListNoGroupedHeadings and
│                                           # TestCosmeticExceptionGroupedHeadingFlattensOnRoundTrip
│                                           # rewritten (research.md D8); new flat/grouped/omission/
│                                           # round-trip cases
│
└── app/
    ├── schema/
    │   └── service/
    │       ├── schema.go                  # Seed(): build a static Index from kernel.CorePredicateDefs/
    │       │                               # CoreTypeDefs once, pass to both RenderNode calls;
    │       │                               # registerIfAbsent: pass core.Index{} (documented safe —
    │       │                               # rendered node never carries Edges, research.md D6)
    │       └── schema_test.go             # mechanical Index-arg update at any direct RenderNode call
    │
    └── graph/
        └── service/
            ├── apply.go                   # nodeContentChanged/writeNode gain `index Index`, threaded
            │                               # from Apply's own existing index parameter (spec 012)
            └── apply_test.go              # mechanical update if these unexported funcs are exercised
                                            # directly; otherwise unaffected (covered via Apply's E2E path)

cmd/arc/graph/
├── subgraph.go                            # humanSubgraphPrinter gains an `index core.Index` field;
│                                           # RunE resolves it via new resolveIndexOrDefault(store)
│                                           # (research.md D7) right after store is mounted
├── subgraph_test.go                       # new case: schema-driven grouped output for a link-role
│                                           # predicate in a subgraph export; existing cases unaffected
│                                           # (their fixtures carry only edge-role predicates today)
├── serve.go                               # buildServer resolves index once via resolveIndexOrDefault;
│                                           # nodeGetHandler/subgraphGetHandler take it as a parameter
│                                           # and use it in their RenderNode/RenderPatch calls; new
│                                           # resolveIndexOrDefault helper defined here
├── serve_test.go                          # mechanical/new case coverage mirroring subgraph_test.go
├── apply.go                               # unaffected (already resolves/passes `index` into
│                                           # appgraph.Apply; RenderNode calls live inside apply.go's
│                                           # service layer, already covered above)
└── apply_test.go                          # new case: a timeline node's entries omission end-to-end
                                            # (spec User Story 2), via the existing sut()/RunE pattern

ARCHITECTURE.md                            # Glossary: Node entry's Edges clause gets one clarifying
                                            # sentence distinguishing parse-time (unchanged) from
                                            # render-time (schema-driven) grouping (research.md D9)
```

**Structure Decision**: No new command and no new `internal/app/<domain>` use-case. This feature corrects
rendering behavior inside `internal/core` (the shared domain package that already owns `RenderNode`/
`RenderPatch`) and threads the resulting new `Index` parameter through the small, already-enumerated set of
existing call sites in `internal/app/{schema,graph}/service` and `cmd/arc/graph` — nothing new to scaffold.

## Complexity Tracking

*No entries — no Constitution Check violation requires justification.*

**Bugfix**: 2026-07-09 — BUG-001 Updated from bugfix patch. Summary and Project Structure amended to require
`RenderNode`/`RenderPatch` to diverge on link-role rendering shape (heading vs. bold-label paragraph) per
ARCNET-CORE §5 vs §14.2, rather than sharing one identical render algorithm.
