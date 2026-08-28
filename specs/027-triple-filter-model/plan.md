# Implementation Plan: Triple Filter for Node Attributes and Graph Edges

**Branch**: `027-triple-filter-model` | **Date**: 2026-08-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/027-triple-filter-model/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Replace `internal/core.Filter`'s four ad-hoc axes (`Types`/`Tags`/`Attrs`/`AttrPatterns`) with a single conjunctive list of triple `Statement`s, each independently constraining a `(Source, Predicate, Target)` position via a shared `Matcher` (OR-of-exact-values, OR-of-regexps, or wildcard — research.md D1). `Filter.Match` now evaluates both a node's attribute facts (plus one synthesized `(id, "type", Type)` fact, D2) and its edge facts (`node.Edges`, already exactly `(id, Predicate, Target)`) uniformly, closing the gap where relations were invisible to filtering. A new `Filter.Traversal()`/`Filter.Narrowing()` split (D3) lets `internal/app/graph/service.Subgraph`/`ContextRetrieve` reuse the same `filter core.Filter` parameter to both gate which edges BFS follows (new capability) and narrow the final result set (today's existing behavior), without a second parameter and without the one-directional-relation trap of requiring a just-reached node to independently re-satisfy the very statement that reached it. `arc grep`'s and `arc subgraph`'s `--type`/`--tag`/`--attr` flags keep their exact CLI contract by lowering to specific statement shapes (D4) that can never be misclassified as traversal constraints. `arc subgraph` gains one new, purely additive `--predicate` flag (D10) so its own BFS — not only the MCP tools — can be scoped by relation. `cmd/arc/graph/serve.go`'s `mcpFilter`/`toCoreFilter` is redesigned around a `statements` JSON array with LLM-legible field names (D7), `subgraph_get` gains a `filter` argument it lacks today (D8), and session/tool-description advertisement plus `VISION.md`'s Filtering section are updated to describe the new model (D9). No new dependency, no new adapter, no new Cobra command.

## Technical Context

**Language/Version**: Go 1.26.6 (match `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra` (CLI flags, unchanged), `github.com/modelcontextprotocol/go-sdk/mcp` (already `arc serve`'s transport, ADR 003) — no new dependency added by this feature

**Storage**: N/A for the filter mechanism itself — node data continues to be read via `internal/adapter/fsys` through the existing `enumerateNodes`/`walkNodeFiles` enumeration `Filter`/`Subgraph`/`ContextRetrieve` already use; no new I/O path

**Testing**: `go test ./...` with `github.com/fogfish/it/v2` for unit tests (table-driven `Filter.Match`/`Traversal`/`Narrowing` cases) and colocated E2E tests extending `cmd/arc/graph/{grep,subgraph,serve}_test.go`'s existing `sut()`/`connectServeSession` helpers (constitution Principles VI, VIII)

**Target Platform**: linux/darwin/windows, amd64+arm64 (per `.goreleaser.yaml`; windows/arm64 excluded) — unchanged by this feature

**Project Type**: Single Cobra CLI binary (constitution Principle III) — this feature touches one domain package (`internal/core`, the `Filter`/`Statement`/`Matcher` types), one service package (`internal/app/graph/service`, traversal-scoping in `subgraph.go`/`context.go`), and two existing command files (`cmd/arc/graph/grep.go`'s shared `optsFilter`, `cmd/arc/graph/subgraph.go`'s new flag, `cmd/arc/graph/serve.go`'s MCP wire shape) — no new package, no new adapter

**Performance Goals**: `Filter.Match`/`Traversal`/`Narrowing` remain in-memory, allocation-light value operations over an already-parsed `core.Node` — no measurable change to `arc grep`/`arc subgraph`'s existing per-node scan cost (dominated by file I/O and regexp scanning, unchanged by this feature); BFS's added per-edge predicate check is O(edges) per node, negligible next to the existing O(nodes) enumeration

**Constraints**: `Filter`/`Filter.Match` remain a pure, no-I/O value type (existing contract, unchanged); `--type`/`--tag`/`--attr` CLI output must be byte-identical to today (spec FR-014/SC-002); the MCP `filter` wire shape break is deliberate and unguarded — no dual schema, no version negotiation, no deprecation period (spec, explicitly out of scope for MCP compatibility)

**Scale/Scope**: One domain type replacement (`internal/core/filter.go`), one new domain file/section (`Statement`/`Matcher`, same package), traversal-scoping changes confined to `internal/app/graph/service/subgraph.go`'s `nodeTargets`/`buildReverseIndex`/`bfs`-neighbor-closures and `context.go`'s equivalent neighbor pooling, CLI-side changes confined to `cmd/arc/graph/grep.go`'s `optsFilter`/`cmd/arc/graph/subgraph.go`'s new flag, MCP-side changes confined to `cmd/arc/graph/serve.go`'s `mcpFilter`/`toCoreFilter`/tool descriptions/`schemaAdvertisement`; one prose update to `specs/VISION.md`'s Filtering section; one glossary-entry update to `ARCHITECTURE.md`'s existing "Filter" row

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Architecture Documentation & ADRs | PASS (action required during implementation) | No ADR conflict: ADR 003 rule 1 (MCP handler stays a thin wrapper) holds — `toCoreFilter` still only decodes JSON into a domain value and calls the identical `appgraph.Grep`/`Subgraph`/`ContextRetrieve` functions. `ARCHITECTURE.md`'s existing "Filter" glossary row (currently describing the four-axis shape) MUST be updated in the implementing PR to describe `Statement`/`Matcher`/`Traversal`/`Narrowing` instead — tracked as a task, not a plan-time gate failure. |
| II. DDD & Glossary | PASS | `Statement`/`Matcher` are the only new domain concepts; both are named and documented here (data-model.md) before implementation, and `ARCHITECTURE.md`'s Glossary MUST gain entries for them alongside the corrected "Filter" row (same PR as the code change, per Principle II's own rule). No flag/domain-name mismatch is introduced: `--predicate` names the same concept `core.Statement.Predicate`/`core.Link.Predicate` already use. |
| III. Hexagonal Architecture | PASS | `Statement`/`Matcher`/`Filter` live in `internal/core` (no `cobra`/`mcp` import); traversal-scoping logic lives in `internal/app/graph/service`; `cmd/arc/graph/{grep,subgraph,serve}.go` remain flag/JSON decode → domain call → render only. No new adapter, no new port. |
| IV. Functional Programming Style | PASS | `Matcher.Match`, `Statement.IsTraversalConstraint`, `Filter.Match`/`Traversal`/`Narrowing`, `admittedEdges`/`admittedBacklinks` are all small, pure, side-effect-free functions (research.md D1/D3/D5), mirroring `filter.go`'s existing per-axis helper shape (`matchTypes`, `matchAttrValue`, etc.) that this feature generalizes rather than abandons. |
| V. Code Quality & Simplicity (YAGNI) | PASS | No new package, no new adapter, no materialized `Fact` type (data-model.md explicitly rejects one), `bfs`'s own signature is left untouched (research.md D5) — the diff is confined to the minimum needed for triple matching plus traversal scoping. `capPool`'s degree ranking is deliberately left unchanged (D6) rather than expanded to a new, unrequested dimension. |
| VI. TDD | PASS (process gate for implementation) | `internal/core/filter_test.go`'s table-driven rewrite, the CLI byte-identical-output regression tests, and the new predicate-scoping tests in `subgraph_test.go`/`context_test.go`/`serve_test.go` MUST all be written first, per Principle VI (see Testing section of the original technical-approach input, reproduced in tasks.md). |
| VII. External Integration & Adapter Consistency | PASS | No new adapter, no new external system. `Filter`/`Statement`/`Matcher` touch no filesystem/network I/O; existing `internal/adapter/fsys` usage in `enumerateNodes`/`walkNodeFiles` is unchanged. |
| VIII. E2E Acceptance Testing & Spec Traceability | PASS (process gate for implementation) | Every acceptance scenario across spec.md's three user stories MUST get a colocated E2E test: User Story 1 in `cmd/arc/graph/subgraph_test.go` (new `--predicate` scenarios) and `cmd/arc/graph/serve_test.go` (`subgraph_get`/`context_retrieve` predicate-scoping scenarios); User Story 2 in `cmd/arc/graph/grep_test.go`/`serve_test.go` (edge-fact matching scenarios); User Story 3 as the byte-identical regression suite across `grep_test.go`/`subgraph_test.go`. |
| IX. Command & Flag Design (CLIG) | PASS | `--predicate` follows the exact naming/repeatability convention `--type` already established (long-form only, no shorthand, `StringArrayVar`, OR-within — Principle IX's flag-naming consistency rule). No new subcommand, no new confirmation/destructive-action concern (still strictly read-only). |
| ADR 003 (MCP Server as second primary-adapter family) | PASS | Rule 1 (thin wrapper) verified above. Rule 3 (loopback-by-default bind) is untouched — this feature does not touch `--http`/`resolveHTTPAddr`. |

No violations requiring Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/027-triple-filter-model/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── cli-contract.md  # --type/--tag/--attr unchanged + new --predicate (arc subgraph only)
│   └── mcp-contract.md  # triple-shaped `filter` argument for node_grep/subgraph_get/context_retrieve
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
└── core/
    ├── filter.go            # REWRITTEN: Filter{Statements []Statement}, Statement{Source,Predicate,Target Matcher},
    │                        #   Matcher{Values,Patterns}, Filter.Match/Traversal/Narrowing (research.md D1-D3)
    ├── filter_test.go       # REWRITTEN: table-driven Filter.Match/Traversal/Narrowing tests (unit, fogfish/it/v2)
    └── ast.go               # UNCHANGED — Node/Link/Predicate shapes already carry everything Filter needs

internal/app/graph/
├── service/
│   ├── subgraph.go          # nodeTargets → admittedEdges; buildReverseIndex → reverseIndex[string][]backlinkEdge
│   │                        #   plus admittedBacklinks; Subgraph's neighbor closures apply filter.Traversal()
│   │                        #   (research.md D5); addCandidate's Match call switches to filter.Narrowing()
│   ├── subgraph_test.go     # + predicate-scoping regression/new-capability tests (User Story 1)
│   ├── context.go           # same Traversal()/Narrowing() split applied to its own neighbor pooling +
│   │                        #   addCandidate (research.md D3)
│   └── context_test.go      # + predicate-scoping tests mirroring subgraph_test.go's additions

cmd/arc/graph/
├── grep.go                  # optsFilter gains --predicate registration helper (used by subgraph.go only,
│                             #   research.md D10); opts.build() final step changes to emit []core.Statement
│                             #   (accumulation logic itself unchanged, research.md D4)
├── grep_test.go             # + edge-fact matching E2E scenarios (User Story 2); existing scenarios unchanged
├── subgraph.go               # NewSubgraphCmd registers --predicate; Long/Example text updated
├── subgraph_test.go          # + --predicate scenarios (User Story 1); existing scenarios unchanged
├── serve.go                  # mcpFilter redesigned around `statements` (research.md D7); subgraphGetArgs
│                             #   gains Filter field (D8); schemaAdvertisement + three Tool.Description
│                             #   strings gain a predicate-scoping clause (D9)
└── serve_test.go             # node_grep/subgraph_get/context_retrieve filter-shape tests replaced (delete
                              #   old type/tags/attrs/attrPatterns JSON assertions, per spec); + predicate-
                              #   scoping E2E scenarios (User Story 1)

specs/VISION.md               # Filtering section rewritten: triple statement model replaces the
                              #   Type/Tag/Attribute-filter-as-separate-axes prose and the old MCP filter
                              #   JSON example (research.md D9)

ARCHITECTURE.md               # "Filter" glossary row corrected; "Statement"/"Matcher" rows added
                              #   (Principle I/II, tracked as an implementation-phase task)
```

**Structure Decision**: This feature touches exactly one domain type family (`internal/core.Filter`/`Statement`/`Matcher`), one service package's traversal internals (`internal/app/graph/service`), and three existing `cmd/arc/graph` command files. No new package, no new adapter, no new Cobra command — the entire change is a reimplementation-behind-a-stable-CLI-surface plus one deliberately-broken MCP wire shape, matching the plan's own "Constraints" section.

## Complexity Tracking

*No entries — Constitution Check has no violations to justify.*
