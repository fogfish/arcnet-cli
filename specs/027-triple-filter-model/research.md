# Phase 0 Research: Triple Filter for Node Attributes and Graph Edges

This feature has no external-dependency unknowns (no new third-party library, no new adapter, no new external system — ADR 003's rule 1 stays satisfied: MCP handlers stay thin wrappers). Every open question below is a **design decision internal to `internal/core`/`internal/app/graph`**, called out explicitly per the plan's instruction not to leave the triple model's ambiguous corners implicit.

## D1 — Statement shape and the `--attr ~=` pattern question

**Decision**: A `Statement` has three positions (`Source`, `Predicate`, `Target`), each an identical `Matcher` value:

```go
type Matcher struct {
    Values   []string          // OR-set, case-insensitive equality; empty = no equality constraint
    Patterns []*regexp.Regexp  // OR-set, regexp match; empty = no pattern constraint
}
```

A `Matcher` is wildcard (matches anything) iff both `Values` and `Patterns` are empty. It matches a candidate string `s` iff wildcard, or `s` case-insensitively equals any `Values` entry, or `s` matches any `Patterns` entry.

**Rationale**: This is "a single matcher abstraction," applied uniformly to all three positions, rather than a fourth parallel `AttrPatterns`-style axis living outside the triple shape. `--attr name=value` lowers to a `Matcher{Values: [value]}` on `Target`; `--attr name~=pattern` lowers to `Matcher{Patterns: [pattern]}` on `Target` — same position, same struct, the only difference is which of the two slices is populated. This directly generalizes today's `matchAttrValue`/`matchAttrPattern` pair (`internal/core/filter.go`) into one reusable type instead of two parallel code paths.

**Alternatives considered**: A parallel `PatternStatements []Statement` list alongside `Statements []Statement` was rejected — it reintroduces exactly the "four independent axes ANDed together" shape this feature exists to replace, just with fewer axes.

## D2 — Synthesizing `Node.Type` as a triple

**Problem**: `Node.Type` is a first-class Go field (`internal/core/ast.go`), not an `Attrs` entry, but `--type` must keep lowering to an ordinary predicate-position statement (per the spec's CLI-compatibility requirement) so it composes with everything else through the same conjunction machinery.

**Decision**: `Filter.Match` synthesizes exactly one additional attribute-triple per node before evaluating statements: `(node.ID, "type", node.Type)`. `--type <value>` lowers to a single `Statement{Predicate: Matcher{Values: ["type"]}, Target: Matcher{Values: <every --type value>}}` (one statement, not one per repeat — see D4).

**Called-out behavior delta**: today's `matchTypes` compares `node.Type == k` — exact, case-sensitive. The synthesized triple flows through the same case-insensitive `Matcher` every other attribute uses. This is a deliberate, minor relaxation (case-sensitive → case-insensitive for `--type`), explicitly called out per the plan's "any observable CLI behavior difference must be called out explicitly as intentional" requirement. It has no observable effect against any conforming graph: every `Type` value is already mandated CamelCase (`specs/019-camelcase-node-types`), so there is no legitimate differently-cased value for case-insensitivity to newly accept. No existing test asserts case-sensitivity of `--type` (verified: no test in `internal/core/filter_test.go`, `cmd/arc/graph/grep_test.go`, or `cmd/arc/graph/subgraph_test.go` exercises differently-cased `--type` values), so FR-014/SC-002's "byte-identical output" requirement is unaffected in practice.

**Alternatives considered**: Giving `Matcher` a case-sensitivity flag so the synthesized `type` triple could stay exactly case-sensitive was rejected as unwarranted complexity for a single field with no live case-variance to protect against.

## D3 — Traversal scoping vs. flat-inclusion: which statements gate which check

**Problem**: The spec (`spec.md` FR-006/FR-009, User Story 1 Acceptance Scenario 5) requires that scoping BFS expansion to one relation must not, by itself, cause the newly-reached nodes to be dropped again by flat-inclusion narrowing. A relation is generally one-directional in what it records (e.g. only the citing document carries the `cites` edge; the cited document carries no reciprocal `cites` fact of its own). If the *same* statement that gated traversal were also required to hold on the destination node's own attributes/edges, `subgraph_get`/`context_retrieve` scoped to `cites` would come back empty — the opposite of the feature's stated purpose (SC-001).

**Decision**: A statement is a **traversal constraint** iff it constrains only the `Predicate` position — `Source` and `Target` are both wildcard `Matcher{}` values. `internal/core.Filter` gains one small, pure, exported helper reused by the service layer:

```go
// Traversal reports the subset of f's statements that scope BFS edge
// admission (Source and Target both wildcard), and Narrowing reports every
// other statement — the subset used for flat inclusion. The two are
// disjoint and their union is f's full statement list.
func (f Filter) Traversal() Filter
func (f Filter) Narrowing() Filter
```

- `Filter.Match(node)` — the single, uniform, pure function everything calls — is unchanged in spirit: it ANDs every statement currently in the receiver's `Statements` slice (per spec FR-005, literally). `arc grep`/`node_grep` call `Match` on the full, original filter, exactly as today — they never traverse, so there is nothing to split.
- `Subgraph`/`ContextRetrieve` compute `scope := filter.Traversal()` once per call and use it (see D5) to gate which edges BFS follows. When calling `Match` to decide whether an already-*discovered, non-seed* candidate survives into the final result, they call `filter.Narrowing().Match(node)` — i.e. they intentionally re-apply only the non-traversal statements, so a bare "expand along `cites`" filter with no other statement narrows nothing further (`Narrowing()` returns an empty statement list, whose `Match` is vacuously true for every node — same posture as `core.Filter{}` today). A filter that combines a traversal constraint *and* an ordinary narrowing statement (e.g. "expand along `cites`, but keep only `Source`-typed results") still narrows via the ordinary statement, because that statement is not a traversal constraint and stays in `Narrowing()`.
- `ContextRetrieve`'s upfront universe-narrowing pass (`filterIncluded`/`pathIncluded`, run over *every* graph node before any traversal happens) keeps using the **full, unsplit filter** — it is not "a newly-discovered node," it is the same use `arc grep` already makes of `Match`, and a bare traversal-only filter should not accidentally widen that pass to every node in the graph. Its `directMatch`/`attrMatch` seeds are therefore still governed by the complete filter, exactly as today; only the neighbor-expansion candidates it later pools get the `Narrowing()` treatment described above.

**Rationale**: This keeps `Filter.Match`'s contract textually identical to the spec (FR-005: "select the node only if every statement in the filter is satisfied") while giving the two call sites (`arc grep`'s direct evaluation vs. `Subgraph`/`ContextRetrieve`'s post-BFS narrowing of *reached* nodes) the two different, individually-correct filter views the spec's own User Story 1 Acceptance Scenario 5 requires ("scoping controls what gets reached, narrowing controls what survives"). No second parameter is added anywhere — `Traversal()`/`Narrowing()` are pure derivations of the one `filter core.Filter` value already threaded through `Subgraph`/`ContextRetrieve`'s signatures, matching the plan's explicit "rather than adding a second parallel parameter" instruction.

**Consequence for CLI compatibility**: `--type`/`--tag`/`--attr` always constrain `Target` (D2/D4), so they can never be classified as a traversal constraint by this rule. `arc subgraph SEED --type Foo` therefore continues to traverse every edge exactly as before this feature (`Traversal()` returns no statements → default admit-everything, D5) and narrows the result exactly as before (`Narrowing()` returns the same statement the old `Types` axis represented). This is the mechanism that keeps FR-010–FR-014/SC-002 true.

**Alternatives considered**: Applying every statement identically at both points (the plan text's literal "a predicate-position statement scopes both... narrowing... and... BFS follows" read maximally) was tried first and rejected once traced through `addCandidate`: it silently breaks the feature's headline scenario (SC-001) for any one-directional relation, which is the common case for a citation-style edge. This research decision resolves that ambiguity in the plan text in favor of what `spec.md` — the authoritative behavioral contract for this feature — actually specifies.

## D4 — CLI flag → statement lowering (exact preservation)

**Decision**: `optsFilter.build()` (`cmd/arc/graph/grep.go`) keeps its existing intermediate accumulation untouched — the same `map[string]string` for `--attr name=value` (last-write-wins on a repeated identical name, exactly as today) and the same `map[string]*regexp.Regexp` for `--attr name~=pattern` — and only changes its *final* step, converting that already-deduplicated intermediate state into statements:

- `--type` (all repeats, if any) → **one** `Statement{Predicate: {Values: ["type"]}, Target: {Values: <all --type values>}}` (OR realized within one statement's `Target` set, matching `Types`' existing OR-of-values convention exactly).
- `--tag` (each repeat) → **one `Statement` per repeat**: `Statement{Predicate: {Values: ["tags"]}, Target: {Values: [<tag>]}}` — AND realized by having each repeat be its own, separately-ANDed statement, matching `Tags`' existing per-tag AND convention exactly.
- `--attr name=value` (post-dedup map) → one `Statement{Predicate: {Values: [name]}, Target: {Values: [value]}}` per map entry.
- `--attr name~=pattern` (post-dedup map) → one `Statement{Predicate: {Values: [name]}, Target: {Patterns: [pattern]}}` per map entry.

**Rationale**: Changing only the final conversion step — not the flag-accumulation logic itself — is what makes "flag parsing itself... unchanged, only what it constructs downstream changes" (plan text) literally true, including the pre-existing last-write-wins quirk for a repeated identical `--attr` name, which no existing test exercises but which this design does not silently change either way.

## D5 — Predicate-scoped BFS without changing `bfs`'s own signature

**Decision**: `bfs(index, neighbors, seed, depth)` (`internal/app/graph/service/subgraph.go`) keeps its exact existing signature — `neighbors func(id string) []string`. What changes is what the two `neighbors` closures `Subgraph`/`ContextRetrieve` construct do internally:

- `nodeTargets` changes from `func(core.Node) []string` to exposing the underlying `[]core.Link` directly (`n.Edges` already *is* `[]core.Link{Predicate, Target, Alias}` — no new type needed); a thin `admittedTargets(n core.Node, scope core.Filter) []string` helper filters `n.Edges` to those whose `(n.ID, e.Predicate, e.Target)` triple satisfies `scope` (D3's `Traversal()` result — an empty `scope` admits every edge, preserving today's default) and returns their `Target`s.
- `buildReverseIndex` changes from `map[string][]string` to `map[string][]core.Link`-shaped backlink entries carrying enough to re-derive `(sourceID, predicate)` per backlink — concretely `reverseIndex map[string][]backlinkEdge` where `backlinkEdge struct { Source, Predicate string }` — so the backlink-direction `neighbors` closure can apply the identical scope check for backlink (incoming) expansion.
- The two `neighbors` closures passed into `bfs` (one per direction, in both `Subgraph` and `ContextRetrieve`) apply the scope check *before* returning, so `bfs` itself stays entirely predicate-agnostic — it never needs to know a `Filter` exists.

**Rationale**: Confines every triple-aware change to the two closure-construction sites and the reverse-index shape; `bfs`'s own recursion, visited-tracking, and hop-counting logic (already covered by `specs/007-arc-subgraph`'s existing tests) is untouched, minimizing the diff's blast radius per Principle V (YAGNI/simplicity).

## D6 — `capPool`'s degree ranking stays on total edge count

**Decision**: `outDegree`/`degree` (truncation-cap ranking) continue to count *every* structural edge on a node, not just scope-admitted ones. Scoping affects which nodes are *reachable* at all (via D5); once a node is in the reachable pool, its rank for cap-truncation purposes is unchanged.

**Rationale**: The plan's technical-approach text does not mention changing cap ranking, and doing so would conflate two independent concerns (which nodes are reachable vs. which of several reachable nodes are most worth keeping under a cap) with no stated requirement driving the change. Out of scope; flagged so a reviewer does not read its absence as an oversight.

## D7 — MCP triple filter field naming

**Decision**: The new `mcpFilter` JSON shape is a list of triple statements:

```json
{
  "statements": [
    { "predicate": "cites" },
    { "predicate": "type", "target": "Source" }
  ]
}
```

Each of `source`/`predicate`/`target` is optional; when present it accepts either a single string or an array of strings (OR-of-values), plus an optional sibling `*Pattern`-suffixed field (`targetPattern`, etc.) for the regexp case, mirroring D1's `Matcher` split at the wire level. Field names (`source`/`predicate`/`target`) match `core.Link`'s own field names (`Predicate`/`Target`) plus `source` for the implicit subject position, satisfying spec FR-016 (no `s`/`p`/`o`, no `subject`/`object`).

**Rationale**: A flat list of statement objects, each shaped exactly like the Go `Statement` type modulo JSON casing, needs no translation cleverness in `toCoreFilter` beyond a 1:1 field mapping — keeping the MCP handler a "thin wrapper" per ADR 003 rule 1.

## D8 — `subgraph_get` gains a `filter` argument it does not have today

**Finding**: `subgraphGetArgs` (`cmd/arc/graph/serve.go`) currently has no `Filter` field at all — `subgraphGetHandler` calls `appgraph.Subgraph` with a hardcoded `core.Filter{}`. Both flat-inclusion narrowing and the new traversal-scoping capability are unreachable from `subgraph_get` today.

**Decision**: `subgraphGetArgs` gains an optional `Filter *mcpFilter` field, following the exact pattern `nodeGrepArgs`/`contextRetrieveArgs` already use, wired through `toCoreFilter()` into `appgraph.Subgraph`'s existing `filter core.Filter` parameter. This is additive to `subgraph_get`'s argument schema (no removal, no compatibility concern — spec explicitly scopes MCP wire compatibility out) and is required for User Story 1's `subgraph_get` scenarios to be reachable at all.

## D9 — Session-advertisement and documentation updates

**Decision**: `schemaAdvertisement` (`cmd/arc/graph/serve.go`) and each of `node_grep`/`subgraph_get`/`context_retrieve`'s `mcp.Tool.Description` gain one clause mentioning predicate-scoped filtering/traversal is available, so a connecting agent's first `schema()` call (already the advertised first call) surfaces the new capability. `specs/VISION.md`'s "Datalog query as a filter" TBD section is left untouched (still out of scope, per spec's explicit boundary), but VISION.md's existing "Filtering" section (Type/Tag/Attribute filter prose plus the old "MCP filter object" JSON example) is rewritten to describe the triple statement model as the current behavior, replacing the `type`/`tags`/`attrs`/`attrPatterns` JSON example with the new `statements` shape from D7.

## D10 — Giving `arc subgraph` (CLI) its own traversal-scoping flag

**Problem**: spec.md's User Story 1 is framed around "a user — often an LLM agent," and its Assumptions section explicitly defers "the exact command-line syntax for expressing that scope" to planning. Without a CLI-facing way to construct a traversal-constraint statement (D3), `arc subgraph`'s own BFS never gains predicate scoping — only the MCP `subgraph_get`/`context_retrieve` tools would, via D7's `statements` argument — leaving the spec's own opening sentence ("neighbor expansion in `arc subgraph`/`arc serve`'s...") half-satisfied.

**Decision**: `arc subgraph` gains one new, purely additive flag, `--predicate <name>` (repeatable, `StringArrayVar`, OR-within — mirrors `--type`'s existing convention exactly): each repeat contributes to a single traversal-constraint `Statement{Predicate: {Values: <every --predicate value>}}` (`Source`/`Target` left wildcard, so D3 classifies it as `Traversal()`, never `Narrowing()`). `optsFilter` (`cmd/arc/graph/grep.go`) gains this field and flag registration; `--predicate` is a no-op for `arc grep` at the Cobra level — it is simply not registered on `arc grep`'s command, since `arc grep` never traverses and D3 already makes a pure predicate-constraint statement vacuous for flat-only matching. `arc lint` registers no filter flags today (confirmed: no `--type`/`--tag`/`--attr` in `cmd/arc/lint/lint.go`) and gains none here either — it is unaffected by this feature.

**Rationale**: `--predicate` is new capability, not a compatibility surface — it does not appear in spec.md's mandatory-compatibility flag list (`--type`/`--tag`/`--attr`), so there is no byte-identical-output constraint to satisfy for it, only the new acceptance scenarios in User Story 1. Naming it `--predicate` (singular value-noun, OR-repeatable) matches `--type`'s existing naming convention (CLIG-consistent, Principle IX) rather than inventing a new flag-naming pattern.

**Alternatives considered**: Overloading `--attr <edge-predicate>=<target>` to also mean "follow this edge" was rejected — D3 deliberately keeps any `Target`-constrained statement out of `Traversal()` precisely to protect `--attr`'s existing flat-inclusion-only meaning; conflating the two would either break that protection or require `--attr` to behave differently depending on whether the named predicate happens to coincide with a real edge predicate, which is exactly the kind of implicit, context-sensitive behavior this feature's own design principle (D1/D3) exists to avoid.
