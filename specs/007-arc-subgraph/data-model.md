# Phase 1 Data Model: `arc subgraph`

Value types are immutable (constitution Principle IV) and carry no Cobra, `os/exec`, raw `os.*` filesystem, or goldmark AST types.

## Shared core (`internal/core`)

### Patch, Node (existing, reused unmodified)

`arc subgraph` reuses `core.Patch`/`core.Node` exactly as `ParsePatch`/`RenderNode` already define them (`internal/core/ast.go`) — no new fields, no new type. `service.Subgraph` builds a `core.Patch` value (seed + reachable nodes as `Patch.Nodes`, plus a synthesized manifest — research.md D2) and hands it to the new `RenderPatch`.

### RenderPatch (new, research.md D2)

```go
func RenderPatch(p Patch) ([]byte, error)
```

The structural inverse of the existing `ParsePatch`: renders `p`'s manifest as a `---`-delimited YAML block (`kind: patch`, `document`, `published`, `title`, `stats`), then `p.Nodes` grouped by `Kind` (kinds sorted alphabetically, nodes sorted alphabetically by `ID` within each kind — research.md D9) under `# <Kind>` H1 headings, each node under a `## <ID>` H2 heading with a fenced ` ```yaml ` front-matter block (attributes only, `kind` excluded — implied by the enclosing H1) followed by the node's body (Text/Edges/Links/Notes), reusing the same body-rendering logic `RenderNode` already applies for the on-disk single-node shape. `error` is non-nil only for a YAML-encoding failure on a malformed attribute value (mirrors `RenderNode`'s own error contract).

**Round-trip property** (the central correctness guarantee, spec FR-008/SC-005): for any `Patch` value `p` with a valid manifest and well-formed nodes, `ParsePatch(bytes.NewReader(RenderPatch(p)))` returns a `Patch` whose `Nodes` set is equal to `p.Nodes` (order may differ from `p.Nodes`' original order, since `RenderPatch` imposes its own deterministic order — research.md D9 — but node content is preserved exactly).

## Application values (`internal/app/graph/kernel`)

### SubgraphResult (new)

The domain value `component.go`'s `Subgraph` returns to `cmd/arc/graph`, rendered by `bios.Registry[SubgraphResult]`.

| Field | Type | Notes |
|---|---|---|
| `Root` | `string` | The graph root that was extracted from |
| `Seed` | `string` | The seed node's basename (`<basename>` as given) |
| `Depth` | `int` | The `<n>` hops requested (post-default-resolution) |
| `Patch` | `core.Patch` | The seed + reachable nodes, plus the synthesized manifest (research.md D2) — passed directly to `core.RenderPatch` by the `Human`/`--json` renderers |
| `DirectReachable` | `int` | Count of nodes discovered by the "direct" (outgoing) BFS pass before capping (research.md D3/D5) |
| `DirectIncluded` | `int` | Count of "direct" nodes actually retained after capping (`= min(DirectReachable, DirectCap)`) |
| `DirectTruncated` | `bool` | `true` when `DirectReachable > DirectIncluded` |
| `BacklinkReachable` | `int` | Count of nodes discovered by the "backlink" (incoming) BFS pass before capping |
| `BacklinkIncluded` | `int` | Count of "backlink" nodes actually retained after capping |
| `BacklinkTruncated` | `bool` | `true` when `BacklinkReachable > BacklinkIncluded` |

`cmd/arc/graph/subgraph.go` derives its optional stderr truncation notice (research.md D10) from `DirectTruncated`/`BacklinkTruncated`; the same counts are additionally carried inside `Patch.Stats` for a `--json`/re-ingesting consumer (research.md D10).

## Configuration (`internal/app/config/kernel`)

### Config (extended — second real field, research.md D6)

| Field | Type | Notes |
|---|---|---|
| `Grep` | `GrepConfig` | unchanged (006) |
| `Subgraph` | `SubgraphConfig` | `yaml:"subgraph,omitempty"` |

### SubgraphConfig (new)

| Field | Type | Notes |
|---|---|---|
| `DirectCap` | `int` | `yaml:"directCap,omitempty"`; `<= 0` (including absent) resolves to the built-in default `4096` |
| `BacklinkCap` | `int` | `yaml:"backlinkCap,omitempty"`; `<= 0` (including absent) resolves to the built-in default `1024` |

Resolution (zero → default) happens once, in `cmd/arc/graph/subgraph.go`, immediately after `internal/app/config.Load` — mirroring `GrepConfig`'s existing resolution point exactly (006 data-model.md).

## Internal traversal state (`internal/app/graph/service`, unexported)

Not part of any public contract, but load-bearing for the design (research.md D3/D4):

| Type | Shape | Purpose |
|---|---|---|
| `nodeIndex` | `map[string]core.Node` | Every parsed node, keyed by `ID`, built during the single enumeration pass (research.md D7) |
| `reverse` | `map[string][]string` | For each target `ID`, the list of node `ID`s with a structural connection (`Edges`/`Links`) to it — the backlink adjacency (research.md D4) |
| `degree(id string) int` | function | `len(node.Edges) + Σ len(block.Seq) + len(reverse[id])` — total structural connectivity, used to rank cap-truncation candidates (research.md D4/D5) |

## Ports

None. `internal/app/graph/service.Subgraph` depends only on `fsys.Mounter` (research.md D8) — no `port.VCS`, no `port.SchemaRegistry`.

## Filesystem I/O

All reads go through `fsys.Store` (`internal/adapter/fsys`, unchanged). `arc subgraph` mounts the graph root the same way `arc apply`/`arc lint`/`arc grep` do and uses the identical `Store.Stat(".arc")` guard (`guardIsGraph`). `arc subgraph` never calls `Store.Create`, `Store.Remove`, or any `File` write method — strictly read-only, like `arc lint`/`arc grep`.

## Validation rules (from spec Functional Requirements)

| Rule | Source | Enforced in |
|---|---|---|
| `<basename>` required; `--depth` optional (default 1); filter optional | FR-001 | `cobra.ExactArgs(1)` in `cmd/arc/graph/subgraph.go`; `--depth` `IntVar` default `1` |
| Seed always included, never filtered | FR-002 | `service.Subgraph` adds the seed to `Patch.Nodes` unconditionally, before the filter is ever consulted |
| Reachability traverses both directions, Edges/Links only (not inline prose) | FR-003 | The two independent BFS passes (research.md D3), built from `core.Node.Edges`/`.Links` and the reverse index (research.md D4) — `HRefs` never consulted |
| Shortest-hop, dedup, cycle-safe | FR-004 | Standard visited-set BFS per pass (research.md D3) |
| Filter restricts non-seed inclusion only | FR-005 | `core.Filter.Match` applied only to the non-seed candidates after both BFS passes complete, before `Patch.Nodes` is finalized |
| Dangling link target excluded, not a hard failure | FR-006 | BFS only enqueues targets present in `nodeIndex`; an absent target is silently skipped |
| Output is the patch-exchange format, kind-grouped | FR-007 | `core.RenderPatch` (research.md D2/D9) |
| Output carries a synthesized manifest, applies via `arc apply` | FR-008 | `service.Subgraph` constructs `Patch.Document`/`Patch.Published` (research.md D2); round-trip proven by `core.RenderPatch`/`ParsePatch` unit tests |
| Read-only, no graph/git mutation | FR-009 | Structural — `service.Subgraph` never receives a write-capable dependency |
| Refuse when target is not an initialized graph | FR-010 | `service.Subgraph`'s `guardIsGraph`, before enumeration begins |
| Seed not found ⇒ clear error, no output | FR-011 | `ErrSeedNotFound`, checked immediately after the enumeration pass, before any BFS runs |
| Invalid `--depth` ⇒ clear usage error, no output | FR-012 | Cobra's own `IntVar` parse error (non-integer) + `opts.build()`'s explicit negative-value check (`ErrInvalidDepth`) |
| `--depth 0` ⇒ seed only | FR-013 | BFS with `n=0` naturally produces an empty reachable set for both passes |
| Two independent, configurable soft caps (direct/backlink) | FR-014 | `kernel.SubgraphConfig{DirectCap,BacklinkCap}` (research.md D6), defaults `4096`/`1024` |
| Cap exceeded ⇒ keep highest-degree candidates, never refuse | FR-015 | Post-traversal degree-sort + truncate per pool (research.md D4/D5); `SubgraphResult.Direct/BacklinkTruncated` |
| Caps independently configurable, defaulting when absent | FR-016 | `SubgraphConfig` zero-value resolution at the `cmd/` wiring layer, mirroring `GrepConfig`'s existing pattern |
