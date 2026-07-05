# Phase 0 Research: `arc subgraph`

## D1: Package structure — extends `internal/app/graph`/`cmd/arc/graph`, not a new domain

**Decision**: `arc subgraph` is a new primary port method on the *existing* `graph` (graph I/O) use-case: `internal/app/graph/kernel/subgraph.go`, `internal/app/graph/service/subgraph.go`, `component.go` gains a `Subgraph(...)` delegator alongside its existing `Apply(...)`/`Grep(...)`, and `cmd/arc/graph/subgraph.go` sits next to `apply.go`/`grep.go`. No new `internal/app/<domain>` package, no new `cmd/arc/<domain>` package.

**Rationale**: Direct implementation of the user's explicit planning instruction (worded as "implement grap grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`" — read in context as applying the same placement instruction given for 006's `arc grep` to this feature, `arc subgraph`, since spec.md/the active feature branch is exclusively about `arc subgraph`). `arc subgraph` is, like `arc grep`, a read of the same graph I/O surface `apply` already reads (mount, walk, parse front-matter) — it has no port of its own (D8), so folding it into `graph` avoids a fourth package that would own nothing `graph` doesn't already own.

**Alternatives considered**: A new `internal/app/context`/`internal/app/query` domain, matching how `arc lint` got its own package — rejected: `arc lint`'s domain got a dedicated package because it is a distinct validation concern with its own port; `arc subgraph` shares `arc grep`'s exact same "mount, enumerate, parse every node" shape, differing only in what it does with the parsed graph (traverse instead of scan-content), so it belongs beside `Grep`, not in a new package.

## D2: `internal/core.RenderPatch` — the structural inverse of `ParsePatch`

**Decision**: A new exported function, `RenderPatch(p Patch) ([]byte, error)`, added to `internal/core/markdown.go` beside the existing `ParsePatch`. It renders the document-level manifest (`kind: patch`, `document`, `published`, `title`, `stats`) as a `---`-delimited YAML front-matter block, then, for each distinct `Kind` present in `p.Nodes` (sorted alphabetically by `Kind` — research.md D9), a `# <Kind>` H1 heading, followed by each of that kind's nodes (sorted alphabetically by `ID` within the kind) as a `## <ID>` H2 heading, a fenced ` ```yaml ` front-matter block (attributes only — `kind` is deliberately excluded here, since `ParsePatch`'s own `parsePatchBody` derives a node's kind from its enclosing H1, never from the per-node YAML fence), and the node's body (Text/Edges/Links/Notes) rendered by the same logic `RenderNode` already uses for the on-disk single-node format.

**Rationale**: This is the exact serialization the spec (and the original VISION.md line) names as "CORE §12.2" — grouped by kind under `# <Kind>`, each node under `## <basename>`, fenced YAML front-matter, verbatim body — and it must be the precise structural mirror of what `parsePatchBody` already parses, so that `ParsePatch(RenderPatch(p))` round-trips (spec FR-008, SC-005). Per the user's explicit instruction ("The graph serialization to patch format is part of the core `internal/core`"), this lives in `internal/core`, not in `internal/app/graph` — consistent with `ParsePatch`/`RenderNode` already living there, and consistent with ADR 001's phase-1 tier (a core type's own reusable serialization, no use-case vocabulary).

**Alternatives considered**: Reusing `RenderNode` directly, wrapping its output in a synthesized `## <ID>` heading after the fact — rejected: `RenderNode` unconditionally writes `kind` into the front-matter and a `# <ID>` H1 (the on-disk single-node file shape), neither of which matches the patch-exchange body shape (`parsePatchBody` explicitly reads a node's YAML fence via `decodeYAMLBlock`, distinct from `RenderNode`'s `---`-delimited block, and derives kind from the enclosing H1, never expecting it inside the per-node YAML). Post-processing `RenderNode`'s bytes with string surgery to strip `kind:` and change the heading level was rejected as fragile, string-shaped logic standing in for what should be a second, small, structurally-correct rendering path sharing the same body-rendering primitives (D9).

## D3: Traversal — two independent, full-graph BFS passes (direct + backlink)

**Decision**: `service.Subgraph` runs two separate breadth-first traversals from the seed, each bounded by `<n>` hops, over the same in-memory node/edge index built by one enumeration pass (D7):
- **Direct**: follows only a node's own outgoing structural connections (`core.Node.Edges` plus every `core.Node.Links` block's `Seq`) — "what this node points to, and what those point to," hop by hop.
- **Backlink**: follows only the reverse-edge index (D4) — "what points to this node, and what points to those," hop by hop.

Each pass runs to completion over the whole reachable graph within `<n>` hops (not stopped early by its cap — D5) before either cap is evaluated. A node reachable by both passes appears in both pools; the final subgraph node set is the seed plus the union of both (possibly-capped) pools, deduplicated by ID.

**Rationale**: Directly operationalizes spec FR-003 (bidirectional reachability, resolved in `/speckit-clarify`) and FR-014/015/016 (two *independently* configured and *independently* capped pools — "the direct cap is large 4096, the back link cap is 1024," each ranked "by quantity of edges"). A single combined/undirected BFS cannot recover, after the fact, which direction reached a given node — information both caps need to evaluate their own pool membership independently.

**Alternatives considered**: One combined BFS that tags each discovered node with the direction of the edge that first reached it — rejected: a node reachable by both a forward and a backward path from the seed would need an arbitrary, order-dependent tie-break to decide which single pool "owns" it, before either cap is even evaluated; that tie-break would make cap/truncation behavior depend on traversal/iteration order, undermining SC-007's "always exactly the highest-degree candidates" determinism guarantee. Two independent passes make each pool's membership, ranking, and cap evaluation independently correct regardless of iteration order, at the cost of some nodes being visited by both passes — cheap, bounded by SC-004's several-thousand-node/10s budget (Complexity Tracking, plan.md).

## D4: Reverse-edge index and degree computation

**Decision**: During the single enumeration pass (D7), alongside the `id → core.Node` index, build `reverse map[string][]string`: for every node's `Edges` and every `Links` block's `Seq`, append that node's own ID under the key `Link.Target`. A candidate node's **degree** (used for cap-truncation ranking, spec FR-015) is `len(node.Edges) + Σ len(block.Seq) for each of node.Links` (its own out-degree) `+ len(reverse[node.ID])` (its in-degree) — its total structural connectivity across the whole graph, not just within the currently-reached subgraph.

**Rationale**: "The back links are ranked by quantity of edges. The most connected nodes are selected" (the clarify-session answer) reads naturally as a global connectivity/popularity signal, not a subgraph-local recount — a node's degree does not change depending on which BFS pass or hop discovered it. The reverse index is the only way to answer "who points at this node" in less than an O(V) rescan per node; building it once, during the same pass that already parses every node (mirroring `arc grep`'s existing enumeration cost), keeps the whole traversal within SC-004's budget.

**Alternatives considered**: Recomputing in-degree on demand per candidate via a full rescan — rejected: quadratic in the worst case (a hop's frontier rescanning the whole graph per node), unnecessary when one linear pass already builds the reverse index for free alongside the existing `id → core.Node` index `arc grep`'s own `service.Grep` already builds the analogous forward index for.

## D5: Cap evaluation is a post-traversal, deterministic truncation step

**Decision**: Both BFS passes (D3) run to full completion (bounded only by `<n>` hops, never by the cap) before either cap is applied. If a pool's discovered-node count exceeds its cap (`DirectCap` default `4096`, `BacklinkCap` default `1024` — D6), the candidates in that pool are sorted by degree (D4) descending, ties broken by `ID` ascending for determinism, and only the top `cap` are retained; the rest are dropped from that pool (a node dropped from one pool may still survive via the other, if also reachable there and within its own cap). Neither cap ever causes the command to refuse to run (spec FR-015 — "soft").

**Rationale**: Directly implements SC-007 ("the retained nodes are always exactly the highest-degree candidates... 100% accuracy") — a cap applied *during* traversal (stopping BFS early once a pool hits its cap) would make the retained set depend on traversal/enumeration order rather than on global degree, which is exactly the non-determinism SC-007 rules out. Running both passes to completion first is affordable: SC-004 already budgets 10 seconds for a several-thousand-node graph, and both passes together are one bounded, linear-in-edges traversal over an already-in-memory index.

**Alternatives considered**: A hard cap that refuses to run once exceeded — rejected explicitly by the clarify-session answer ("both caps are soft"). An early-exit BFS that stops expanding a pool once its cap is hit — rejected: produces a traversal-order-dependent, not degree-ranked, result, failing SC-007.

## D6: `.arc/config.yml` — `Subgraph` gains the config struct's second real field

**Decision**: `internal/app/config/kernel.Config` (which already carries `Grep GrepConfig` per 006 research.md D10) gains a second field:

```go
type Config struct {
    Grep     GrepConfig     `yaml:"grep,omitempty"`
    Subgraph SubgraphConfig `yaml:"subgraph,omitempty"`
}

type SubgraphConfig struct {
    DirectCap   int `yaml:"directCap,omitempty"`
    BacklinkCap int `yaml:"backlinkCap,omitempty"`
}
```

A zero/absent `DirectCap`/`BacklinkCap` (including an absent `.arc/config.yml` entirely) resolves to the built-in defaults (`4096`/`1024`) at the `cmd/arc/graph/subgraph.go` wiring layer, exactly mirroring how `GrepConfig.Workers`/`MaxLineWidth` already resolve (006 research.md D10) — `internal/app/config.Load`/`Save` stays a pure YAML round-trip, no defaulting logic inside that package.

**Rationale**: The clarify-session answer states the caps are "defined via config," and this codebase already has exactly one configuration file/mechanism for per-graph tunables (`.arc/config.yml`, `kernel.Config`) — reusing it is both the user's stated preference and consistent with constitution Principle XI (no second configuration mechanism for one command).

**Alternatives considered**: Flag-only configuration (`--direct-cap`, `--backlink-cap`) — rejected: the clarify answer names config explicitly, and a graph-wide, rarely-changed tunable belongs in the graph's own persistent config, not a flag repeated on every invocation (consistent with how `arc grep`'s `Workers`/`MaxLineWidth` were placed, not left flag-only).

## D7: Shared file-walk helper — reused, not re-derived, from `arc grep`

**Decision**: `internal/app/graph/service`'s existing `walkGrepNodeFiles` (recursive, `.md`-only, excluding `.arc/`/`_schema/`, sorted — 006 research.md D9) is renamed to a direction-neutral `walkNodeFiles` and reused, unmodified in behavior, by both `Grep` and the new `Subgraph`.

**Rationale**: Constitution Principle V ("no duplicate, divergent implementations of the same capability") applies within a package exactly as it does across packages — `Subgraph` needs the identical "every node file in the graph, `.md` only, `.arc`/`_schema` excluded, deterministic order" enumeration `Grep` already implements; a second, copy-pasted walker in the same `service` package would be exactly the drift Principle V exists to prevent, this time one file over rather than one package over.

**Alternatives considered**: A private, subgraph-only copy — rejected outright given the identical existing function sits in the same package, not even a different one; renaming for direction-neutrality is the only change needed.

## D8: No new port

**Decision**: `internal/app/graph/service.Subgraph` takes only `fsys.Mounter` (plus `core.Filter`, the seed basename, depth, and resolved `kernel.SubgraphConfig`) — no `port.VCS`, no `port.SchemaRegistry`.

**Rationale**: `arc subgraph` never touches git history and never registers a kind or predicate — it is a pure read of already-parsed node content and structural links, mirroring `arc grep`'s own precedent (006 research.md D13) and ADR 001 port isolation rule 2 ("as narrow as the use-case's actual need").

## D9: Deterministic node ordering owned by the renderer, not the caller

**Decision**: `RenderPatch` itself sorts its input `Patch.Nodes` for output purposes — grouping by `Kind` (kinds ordered alphabetically), then by `ID` alphabetically within each kind — rather than requiring `service.Subgraph` to pre-sort before constructing the `Patch` value it passes in.

**Rationale**: Matches this codebase's existing precedent that a renderer owns its own deterministic ordering (`renderFrontMatter` already sorts attribute keys internally, rather than trusting a caller to hand them in sorted). Placing this in `RenderPatch` also means any *future* caller of the same function (VISION.md's `arc export patch`, which the spec text explicitly notes shares this serialization) gets deterministic output for free, without re-deriving the sort.

**Alternatives considered**: Sorting in `service.Subgraph` before constructing the `Patch` — rejected: would leave a second, future caller of `RenderPatch` (e.g. `arc export patch`) responsible for re-deriving the same ordering rule itself, or risking non-deterministic output if it forgot to.

## D10: No color — a deliberate, user-directed deviation from `arc grep`'s presentation pattern

**Decision**: `cmd/arc/graph/subgraph.go` never references `bios.SCHEMA`'s styled fields. It still resolves output mode through the shared `bios.Registry[kernel.SubgraphResult]{Human: ...}` / `bios.ResolveMode()` machinery (ADR 002 DS-04), exactly as every other command does, so `--json` is answered automatically and for free — but its `Human` renderer writes `core.RenderPatch`'s bytes unstyled, verbatim, to stdout. A truncation notice (D5, spec FR-015's "the output MUST indicate that truncation occurred") is carried two ways: (1) inside the patch document's own manifest `stats` map (so a consumer parsing the patch sees it was truncated, e.g. `stats: {directTruncated: true, directIncluded: 4096, directReachable: 5000, ...}`), and (2) as one plain, unstyled diagnostic line on stderr (no `SCHEMA.Hint`/color — just `fmt.Fprintf(os.Stderr, ...)`), so an interactive user notices it without it polluting the stdout document.

**Rationale**: Directly implements the user's explicit planning instruction ("Do not use the color system for output"). This is a legitimate, narrow deviation from `arc grep`'s own D11 pattern (006), not a constitution violation: Principle X requires color be *automatically disabled* under certain conditions and never be the *sole* carrier of information — it does not require every command to use color, and a command whose entire stdout contract is a machine/LLM-round-trippable document is exactly the case where styling would corrupt the contract (an ANSI escape sequence embedded in a document handed to `arc apply` or pasted into an LLM prompt is actively harmful, not merely unnecessary).

**Alternatives considered**: Coloring the patch document's headings/front-matter fences for an interactive terminal, stripped when piped (mirroring `arc grep`'s TTY-gated highlighting) — rejected outright per the user's explicit instruction; also would have required `arc apply` (or an LLM) to tolerate embedded ANSI codes if a user copy-pasted colored terminal output instead of piping raw bytes, a fragility `arc grep`'s own match-highlighting doesn't carry (that command's output is consumed as plain text fields, never re-ingested as a structured document).

## D11: Exit-code convention — success is unconditional; no `bios.ErrSilent` path

**Decision**: `cmd/arc/graph/subgraph.go`'s `RunE` prints its result and returns `nil` whenever the command actually ran (seed found, depth valid, target is a graph) — there is no "ran successfully but found nothing" case to signal via `bios.ErrSilent`, unlike `arc grep`/`arc lint`. A genuine refusal (seed not found, invalid `--depth`, not an initialized graph) returns a real `error`, formatted at the single `Execute()` site (DS-07), exactly like every other command's refusal path.

**Rationale**: Spec FR-002 guarantees the seed is always included in the output — there is no analogue to `arc grep`'s "zero matches" or `arc lint`'s "zero violations" outcome; a `--depth 0` run with a filter matching nothing still produces a valid one-node patch document (spec Edge Cases), which is a successful, non-empty result, not an empty one. Introducing a silent/zero-result exit path with no actual zero-result case to signal would be an unused, speculative branch (Principle V/YAGNI).

**Alternatives considered**: Signaling "no reachable nodes beyond the seed" via `bios.ErrSilent`, treating an all-filtered-out or depth-0 extraction like `arc grep`'s "zero matches" — rejected: that outcome is not a failure or an empty result the way zero grep matches is; the command still produced exactly what was asked for (a valid, non-empty subgraph document), so signaling it as a "nothing found" condition would be actively misleading to a script checking exit status.
