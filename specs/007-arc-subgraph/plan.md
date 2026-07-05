# Implementation Plan: Extract a Self-Contained Subgraph (`arc subgraph`)

**Branch**: `007-arc-subgraph` | **Date**: 2026-07-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/007-arc-subgraph/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

**Bugfix**: 2026-07-05 — BUG-001 Added the opt-in `--stubs` flag (spec FR-017): a boundary target referenced by an included node but excluded from the extraction now gets a minimal (kind + id only) stub node section when `--stubs` is passed, so applying extracted output into a graph that lacks the excluded targets (e.g. an empty graph) never produces a dangling reference. See "Bugfix (BUG-001)" notes below for the affected sections.

## Summary

Implement `arc subgraph <basename> [--depth <n>] [<filter>]`: extract the seed node plus every node reachable from it within `<n>` hops (default `1`), traversing structural connections in **both** directions (a node's own outgoing Edges/Links, and any other node's connection that targets it), optionally narrowing the non-seed nodes with the existing `--kind`/`--tag`/`--attr` filter, and serialize the result as one patch-exchange document (CORE §12.2): a synthesized document manifest followed by nodes grouped under `# <Kind>` headings, each under `## <basename>` with a fenced YAML front-matter block and verbatim body — ready to paste into an LLM prompt or feed straight back into `arc apply`. Per the user's explicit instruction, `arc subgraph` extends the **existing** `internal/app/graph`/`cmd/arc/graph` domain (a third primary-port method, `Subgraph`, alongside `Apply` and `Grep`) — no new domain package. The graph-to-patch-exchange serializer, `core.RenderPatch` (the structural inverse of the existing `core.ParsePatch`), is new core-domain logic in `internal/core`, per the user's explicit instruction that "graph serialization to patch format is part of the core `internal/core`". The two reachability directions named in spec.md's Clarifications — "direct" (outgoing) and "backlink" (incoming) — are computed as two independent, full-graph BFS passes bounded by `<n>` hops, each subject to its own independently configurable soft cap (`4096`/`1024` defaults) that, when exceeded, retains only the most-connected candidates (ranked by total structural edge count) rather than refusing to run (spec FR-014/015/016). Per the user's explicit UX instruction, this command's own rendering makes **no use of `bios.SCHEMA`'s color/style system** — its stdout output is a structured document meant for machine/LLM consumption, not a colorized human table — while still resolving output mode (`--json`/human) through ADR 002 DS-04's `bios.Registry`/`ResolveMode()` exactly like every other command. **Bugfix (BUG-001)**: an opt-in `--stubs` flag additionally emits a minimal (kind + id only) node section for every structural link target that exists in the source graph but falls outside this extraction's own boundary (depth, cap, or filter), so the document's own links never dangle once applied into a graph that does not already contain those targets (spec FR-017) — off by default, so today's boundary rendering is unchanged unless requested.

## Technical Context

**Language/Version**: Go 1.26, matching `go.mod`

**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/fogfish/faults`, `gopkg.in/yaml.v3` (already a transitive dependency of `internal/core`'s existing YAML front-matter codec) — no new third-party dependency added by this feature. `github.com/charmbracelet/lipgloss` is **not** used by this command's own rendering (per the user's explicit "do not use the color system for output" instruction) — it remains a dependency of the binary as a whole (used by other commands), just not referenced from `cmd/arc/graph/subgraph.go`

**Storage**: The mounted graph root, read exclusively through the existing `internal/adapter/fsys` `Store`/`Mounter` (no changes to that package's own contract) — `arc subgraph` performs **zero writes**, the third strictly read-only command in the codebase after `arc lint` and `arc grep` (spec FR-009)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; E2E tests colocated at `cmd/arc/graph/subgraph_test.go` via the existing `sut()`/fixture-graph helpers `cmd/arc/graph/apply_test.go`/`grep_test.go` already establish, one per spec.md acceptance scenario; unit tests for `internal/app/graph/service.Subgraph` against a fake `fsys.Mounter`/`fsys.Store` (no VCS/schema port needed, research.md D8); unit tests for `internal/core.RenderPatch` proving the round-trip property (`ParsePatch(RenderPatch(p))` reproduces `p`'s node set) against table-driven `core.Patch` fixtures, including the kind-grouping/alphabetical-ordering contract (research.md D2/D9). **Bugfix (BUG-001)**: an E2E regression test extracts with `--stubs`, applies the result into a freshly initialized, otherwise empty graph, and confirms `arc lint` reports no resolvable-links violation (spec SC-008) — the specific untested flow (apply into a graph that does not already contain the excluded targets) that let the original gap ship.

**Target Platform**: linux, darwin, windows (amd64 + arm64, `windows/arm64` excluded) — unchanged from `.goreleaser.yaml`

**Project Type**: Single Cobra CLI binary, module `github.com/fogfish/arcnet-cli`, binary name `arc` — extends the existing `internal/app/graph` use-case with its third primary-port method (`Subgraph`, alongside `Apply` and `Grep`)

**Performance Goals**: Spec SC-004 — a subgraph extraction against a graph of several thousand nodes completes in under 10 seconds; achievable because both BFS passes (research.md D3) run over an already-fully-parsed, in-memory node/edge index (one sequential pass to build, mirroring `arc lint`/`arc grep`'s existing enumeration cost) rather than re-reading files per hop

**Constraints**: Target directory MUST already be an initialized graph (spec FR-010, same `guardIsGraph` guard `arc apply`/`arc lint`/`arc grep` already use); `<basename>` MUST identify an existing node or the command refuses before any traversal (spec FR-011); `--depth` MUST be a non-negative integer (spec FR-012/FR-013); the seed is never excluded by the filter (spec FR-002/FR-005); a dangling link target is silently excluded, never a hard failure (spec FR-006); `arc subgraph` MUST make no filesystem or git-history changes under any circumstance (spec FR-009); output MUST carry a synthesized document-level manifest so it is structurally acceptable to `arc apply` unmodified (spec FR-008); this command's renderer MUST NOT call `bios.SCHEMA`'s styled fields (user's explicit UX instruction). **Bugfix (BUG-001)**: with `--stubs`, a boundary target's stub node MUST carry no attributes beyond kind/id and MUST NOT be expanded (its own Edges/Links are never traversed), or the `--depth` bound becomes meaningless (spec FR-017)

**Scale/Scope**: One new bare-verb command (`arc subgraph`), one new method on the existing `internal/app/graph` use-case (`Subgraph`, alongside `Apply`/`Grep`), one new core-domain function (`internal/core.RenderPatch`, the inverse of the existing `ParsePatch`), one new field on the existing `internal/app/config/kernel.Config` (`Subgraph SubgraphConfig{DirectCap,BacklinkCap}`) — no changes to `internal/core.Node`/`ParseNode`/`RenderNode`/`ParsePatch`'s existing public contracts, no changes to `internal/adapter/fsys`'s public contract, no changes to `internal/bios.Schema`, no new port anywhere

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applies? | Status |
|---|---|---|
| I — Architecture Documentation & ADRs | Yes | PASS, with obligation — [ARCHITECTURE.md](../../ARCHITECTURE.md)'s Directory Structure and Glossary sections MUST be updated in the same PR to add `internal/app/graph`'s new `Subgraph`/`subgraph.go` members, `internal/core.RenderPatch`, and new glossary terms (Seed Node, Reachable Node, Traversal Cap, Subgraph — from spec.md Key Entities). `tasks.md` MUST include this task |
| II — DDD & Glossary | Yes | PASS — new glossary terms (`Seed Node`, `Reachable Node`, `Subgraph`, `Traversal Cap`) copied into ARCHITECTURE.md per the Principle I obligation above; no new type duplicates an existing one — `core.Patch`/`core.Node` (already defined for `ParsePatch`) are reused as-is, not re-modeled |
| III — Hexagonal Architecture | Yes | PASS — `cmd/arc/graph/subgraph.go` is Cobra wiring only (flag parsing, mount, render); `internal/app/graph/{kernel,service}` holds the BFS/capping/filter logic (no new `port/` needed, research.md D8); `internal/core.RenderPatch` has no dependency on Cobra or any `internal/app/*` package, matching `ParsePatch`'s own existing placement exactly |
| IV — Functional Programming Style | Yes | PASS — BFS traversal, degree ranking, and `RenderPatch` are pure functions over `core.Node`/`core.Patch` values (I/O confined to the enumeration pass, mirroring `arc grep`'s existing shape); no inline comments |
| V — Code Quality & Simplicity (SOLID/YAGNI) | Yes | PASS — the graph's node/edge-file-walk helper is shared with `arc grep` rather than re-derived a second time in this package (research.md D7); the existing `optsFilter` struct from `cmd/arc/graph/grep.go` is reused verbatim, not redeclared (research.md D6); `RenderPatch` is added once, in `internal/core`, not duplicated per caller |
| VI — TDD | Yes | PASS — E2E, service, and `core.RenderPatch` unit tests written first; the round-trip property (`ParsePatch(RenderPatch(p))`) is the central correctness test for the new serializer |
| VII — External Integration & Adapter Consistency | Yes | PASS — no new external integration; all filesystem I/O continues through the existing `internal/adapter/fsys` `Store`/`Mounter`, exactly as `arc apply`/`arc lint`/`arc grep` already do |
| VIII — E2E Acceptance Testing | Yes | PASS — spec.md's 3 user stories / 12 acceptance scenarios map 1:1 to E2E tests in `cmd/arc/graph/subgraph_test.go` |
| IX — CLIG/Cobra (ADR 002) | Yes | PASS — DS-01 bare-verb grammar (`arc subgraph`, continuing `arc init`/`arc apply`/`arc lint`/`arc grep`'s precedent); DS-02's existing `optsFilter` options struct is reused, plus one new local `--depth` int flag; DS-03's reserved shorthands untouched, no new shorthand claimed. **Bugfix (BUG-001)**: one additional local `--stubs` bool flag (default `false`), same DS-02 pattern, no shorthand claimed |
| X — Terminal Output, Color & Interactivity | Yes, narrowly | PASS, with a documented, deliberate deviation — the user's explicit instruction is that this command's own output MUST NOT use the color system. This does not contradict Principle X: Principle X requires color to be *automatically disabled* under certain conditions and never be the *sole* carrier of information; it does not mandate that every command use color. `arc subgraph`'s stdout is a structured, machine-consumable patch document (round-trip target: `arc apply`), for which styling would be actively harmful (ANSI codes embedded in a document meant to be re-ingested or pasted into an LLM prompt). `bios.Registry`/`ResolveMode()`/`ErrSilent` (the mode-resolution machinery, not `SCHEMA`'s styles) are still used, exactly as every other command uses them, for the `--json` contract (DS-04) |
| XI — Configuration, Env & Secrets | Yes | PASS — extends the existing `.arc/config.yml` `Config` struct with a second real field (`subgraph.directCap`, `subgraph.backlinkCap`, research.md D5) via the unmodified `internal/app/config.Load`/`Save` round-trip; no new configuration file, no secrets involved |
| XII — Documentation & Help System | Yes | PASS — `Short`/`Long`/`Example` populated per DS-11; every expected failure (seed not found, invalid `--depth`, not-a-graph) declared as a `faults.Type`/`faults.SafeN` constant in `internal/app/graph/service/errors.go` (extending the existing file), wrapped via `.With()` |
| XIII — Distribution & Release Engineering | No | N/A — no changes to the release pipeline |
| XIV — Versioning/Security | Yes | PASS — adds a new, additive `--json` schema (`kernel.SubgraphResult`); no breaking change to any existing `--json` contract (`kernel.GrepResult`/`kernel.ApplyResult` are untouched); zero new third-party dependencies means no new supply-chain surface for `govulncheck` to track. **Bugfix (BUG-001)**: `kernel.SubgraphResult` gains one additional field (`Stubs int`) and `Patch.Stats` one additional key (`stubs`) — both additive, no existing field removed or retyped |

**ADR 001 port isolation rule 2** (explicit check, since a port is conspicuously *absent* here): satisfied — `internal/app/graph/service.Subgraph` needs neither `port.VCS` nor `port.SchemaRegistry` (research.md D8), exactly like `arc grep`'s own precedent (006 research.md D13); this is the rule working as intended, not an oversight.

**ADR 001 domain-evolution phases**: `core.RenderPatch` matches phase 1 exactly (`internal/core`: "a collection of core types in the context of the application's problem domain... allowed dependencies on themselves or on open-source modules only") — it operates purely on `core.Patch`/`core.Node`, has no use-case-specific vocabulary, and is the direct structural inverse of the already-phase-1 `core.ParsePatch`. It is deliberately **not** placed in `internal/app/graph` (serialization of the patch-exchange format is graph-shape-general, already how `ParsePatch`/`RenderNode` are placed) and **not** a new `internal/pkg/<lib>` (unlike 006's `internal/pkg/grep`, it has a direct, non-generic dependency on `core.Node`/`core.Kind`, so phase 1 — not phase 2 — is the correct tier, per the user's own explicit placement instruction).

No unresolved Constitution Check conflicts. No entries required in Complexity Tracking below beyond the one documented, non-speculative trade already called out there.

## Project Structure

### Documentation (this feature)

```text
specs/007-arc-subgraph/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/            # Phase 1 output
│   └── cli-contract.md
└── tasks.md              # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── arc/
    ├── root.go               # + registers graph.NewSubgraphCmd(); no new persistent flags
    └── graph/                 # existing — gains one new command file
        ├── apply.go             # unchanged
        ├── grep.go              # unchanged; its unexported optsFilter is reused (research.md D6)
        ├── subgraph.go          # NEW — package graph: NewSubgraphCmd() *cobra.Command; local
        │                        #   --depth int flag + the existing optsFilter{kind,tag,attr};
        │                        #   mounts the graph, loads internal/app/config, calls
        │                        #   internal/app/graph.Subgraph, writes core.RenderPatch's bytes
        │                        #   to stdout via bios.Registry (NO bios.SCHEMA use — research.md
        │                        #   D10); a truncation notice (if any) goes to stderr, plain text;
        │                        #   BUGFIX BUG-001: + local --stubs bool flag (default false),
        │                        #   threaded into the appgraph.Subgraph call
        └── subgraph_test.go     # NEW — E2E tests, one per spec.md acceptance scenario, via sut();
                                  #   BUGFIX BUG-001: + a regression test extracting with --stubs,
                                  #   applying into a freshly initialized empty graph, then
                                  #   confirming arc lint reports no resolvable-links violation

internal/
├── core/
│   ├── markdown.go            # + RenderPatch(Patch) ([]byte, error) — inverse of ParsePatch
│   │                           #   (research.md D2); factors shared node-body/front-matter
│   │                           #   rendering out of the existing RenderNode so both share one
│   │                           #   body-rendering implementation (research.md D9)
│   └── markdown_test.go        # + round-trip tests: ParsePatch(RenderPatch(p)) reproduces p
│
└── app/
    ├── config/
    │   └── kernel/
    │       └── config.go        # + Config.Subgraph SubgraphConfig{DirectCap,BacklinkCap} (D5)
    │
    └── graph/                  # existing — gains Subgraph alongside Apply/Grep
        ├── component.go          # + Subgraph(ctx, mounter, filter, basename, depth, cfg, dir)
        │                         #   (kernel.SubgraphResult, error); BUGFIX BUG-001: signature
        │                         #   gains a trailing `stubs bool` parameter
        ├── kernel/
        │   └── subgraph.go        # NEW — SubgraphResult, and the core.Patch it wraps;
        │                          #   BUGFIX BUG-001: + Stubs int field (count of stub node
        │                          #   sections emitted, mirroring Direct/BacklinkIncluded)
        └── service/
            ├── subgraph.go        # NEW — Subgraph use-case: enumerate+parse+index every node
            │                       #   (reusing walkNodeFiles, research.md D7), build the
            │                       #   reverse-edge index, run the two independent BFS passes
            │                       #   (research.md D3), apply the two independent caps by
            │                       #   degree (research.md D4), build+return the core.Patch
            │                       #   (research.md D5's synthesized manifest); BUGFIX BUG-001:
            │                       #   when stubs is true, compute the boundary set (targets of
            │                       #   any included node's Edges/Links present in the node index
            │                       #   but not selected for inclusion) and append one minimal
            │                       #   core.Node{ID,Kind} per boundary target — never expanded
            ├── subgraph_test.go    # unit tests against fake fsys.Mounter/Store; BUGFIX BUG-001:
            │                       #   + stub-emission cases (only when requested, no attributes
            │                       #   beyond kind/id, never itself traversed)
            ├── grep.go             # unchanged; its walkGrepNodeFiles is renamed/reused
            │                       #   (research.md D7) as this package's shared file-walk helper
            └── errors.go           # existing — + ErrSeedNotFound, ErrInvalidDepth

ARCHITECTURE.md               # + Directory Structure/Glossary updated (Principle I obligation above)
```

**Structure Decision**: This feature extends the project's existing `internal/app/graph` use-case with a third primary-port method (`Subgraph`, alongside `Apply` and `Grep`), per the user's explicit instruction ("implement grap grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`" — read here as applying to this feature's own command, consistent with how `arc grep` was placed in 006). No new `internal/app/<domain>` or `cmd/arc/<domain>` package is created. Per the user's second explicit instruction, the graph-to-patch-exchange serializer lives in `internal/core` (`RenderPatch`, alongside the existing `ParsePatch`), not inside `internal/app/graph` and not as a new `internal/pkg/<lib>` — it is graph-shape-general core logic, the direct structural inverse of an already-phase-1 function. `internal/app/config/kernel.Config` gains its second real field, following the same extension pattern D10 of 006 established for `Grep`. No existing package's public contract changes: `internal/core.Node`/`ParseNode`/`RenderNode`/`ParsePatch`, `internal/app/config.Load`/`Save`, `internal/adapter/fsys`, `internal/bios.Schema`, and `internal/app/graph.Apply`/`Grep` are all consumed/extended without modification to their current behavior.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Two independent, full-graph BFS passes (direct + backlink) instead of one combined, direction-agnostic BFS (research.md D3) | The finalized spec's two caps (FR-014/015/016) are independently configured and independently ranked per direction — "the direct cap is large 4096, the back link cap is 1024," each evaluated against "the most connected nodes" within *that* direction's own candidate pool. A single undirected BFS cannot recover which direction reached a node once merged, so it cannot honor two independent caps | A single BFS tagging each discovered edge with its direction was considered, but a node reachable by both a forward and a backward path would need an arbitrary tie-break to decide which cap/pool "owns" it for capping purposes before the caps are even evaluated — two independent passes make each pool's membership, ranking, and cap evaluation independently correct and independently testable, at the cost of visiting some nodes twice (cheap: bounded by SC-004's several-thousand-node/10s budget, the same budget `arc grep`'s own full-graph enumeration already fits inside) |
