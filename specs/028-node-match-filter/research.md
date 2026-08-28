# Phase 0 Research: MCP `node_match` Filter Tool

## D1 — Materializing matched facts: reversing 027's rejection

**Problem**: `specs/027-triple-filter-model/data-model.md` explicitly rejected a materialized `[]Fact` slice: "a value never inspected as a collection — `Filter.Match` only ever asks 'does *some* fact satisfy *this* statement,' never 'list every fact.'" `node_match`'s entire purpose (spec FR-003/FR-004) is exactly that second question.

**Decision**: Add `core.Fact{Property, Value string}` and `func (f Filter) MatchingFacts(node Node) []Fact` to `internal/core/filter.go`, alongside the unchanged `Filter.Match`. `MatchingFacts` walks the same three fact sources `statementSatisfiedBy` already visits per statement — the synthesized `(id, "type", Type)` fact, each attribute fact, each edge fact — but instead of returning on the first satisfying fact, it collects every `(property, value)` pair that satisfies *any* statement in `f`, deduplicated.

**Rationale**: 027's rejection was correct for its own scope (`Filter.Match`'s only consumers needed a boolean) but was explicitly scoped to "a value never inspected as a collection" — that premise no longer holds once a feature (`node_match`) exists whose entire contract is that collection. Reusing the existing per-statement fact walk keeps the two functions' semantics provably consistent (a node's `MatchingFacts` is non-empty iff `Match` is true for at least one statement) without touching `Match`'s existing contract or any of its callers.

**Alternatives considered**: Changing `Filter.Match`'s signature to also return facts was rejected — it would force every existing call site (`node_grep`, `Subgraph`, `ContextRetrieve`'s `addCandidate`) to handle a return value they don't need, violating Interface Segregation (constitution Principle V) for no benefit to those callers.

## D2 — Empty/missing filter: reject, don't enumerate everything

**Problem**: Every other filter-accepting MCP tool treats an absent/empty filter as "match everything" (`core.Filter{}`'s zero value vacuously satisfies `Match` for every node). Applying that same default to `node_match` would mean a call with no filter enumerates every fact of every node in the graph — a huge, informationless dump the tool's own reason for existing (evidence of *why* something matched) cannot supply for an empty criterion.

**Decision**: `service.Match` MUST reject a filter with zero statements up front, before any node file is opened, with a new sentinel `ErrEmptyFilter` (mirroring `Grep`'s existing "validate before enumeration" precedent for `ErrInvalidPattern`).

**Rationale**: Spec FR-005 and Edge Cases already settle this at the specification level; this decision is the mechanical implementation of that requirement, placed at the same point in the request pipeline `service.Grep` already validates its own required input (`pattern`).

**Alternatives considered**: Returning an empty result for an empty filter (silently) was rejected — spec SC-002 explicitly requires distinguishing "no matches" from "invalid request," and a silent empty result would collapse that distinction for the one input that is actually invalid, not merely unmatching.

## D3 — Fact deduplication and ordering

**Problem**: A node can satisfy the same statement via multiple facts (e.g., two elements of an array-valued `tags` attribute both matching one statement's target), and two different statements can independently be satisfied by the exact same fact. Spec Edge Cases requires the former to produce one entry per distinct matching element, and the latter to collapse to one entry.

**Decision**: `MatchingFacts` deduplicates by the `(Property, Value)` pair only — not by which statement(s) produced it — using a `map[Fact]bool` seen-set exactly as `internal/app/graph/service/apply.go`'s existing `combineLabels`/`unionFirstSeen` helpers already dedupe by value. Results are sorted by `(Property, Value)` for deterministic output, matching `walkNodeFiles`'s existing sorted-enumeration precedent.

**Rationale**: A `(property, value)` pair is the atomic unit of "evidence" the spec's Key Entities section defines a match entry around; which specific statement(s) happened to reference it is not part of that entity's shape (`{id, property, value}`, no statement index).

**Alternatives considered**: Reporting one entry per (statement, fact) pair — allowing the same fact to appear twice if it satisfies two statements — was rejected as directly contradicting spec Edge Cases' explicit example.

## D4 — `node_match`'s node-level scan reuses `service.Grep`'s enumeration, not `service.Subgraph`'s

**Problem**: Two existing enumeration shapes exist in `internal/app/graph/service`: `Grep`'s flat `walkNodeFiles` + `readGrepNode` scan (no traversal), and `Subgraph`'s seeded BFS (`enumerateNodes` + `bfs` + `Traversal()`/`Narrowing()`). `node_match` has no seed and does not traverse — spec Assumptions already states every statement narrows (FR-006).

**Decision**: `service.Match` is built on the same flat scan `service.Grep` already uses (`walkNodeFiles`, `readGrepNode`, `guardIsGraph`), calling `filter.Match(node)` (the full, un-split filter — never `Traversal()`/`Narrowing()`) exactly as `Grep` already does.

**Rationale**: This is the same conclusion 027's own research.md D3 already drew for `Grep`: "`arc grep`/`node_grep` call `Match` on the full, original filter, exactly as today — they never traverse, so there is nothing to split." `node_match` is architecturally identical to `node_grep` minus the regexp content scan, so it reuses the same enumeration and the same undivided `Match` call.

**Alternatives considered**: Building `Match` on `Subgraph`'s `enumerateNodes`/`bfs` machinery was rejected — that machinery exists specifically to support seeded traversal and BFS-order capping, neither of which `node_match` has any use for; reusing it would pull in unused complexity (Principle V).

## D5 — Unreadable node files: report, don't fail

**Problem**: `service.Grep` records unparseable node files in `GrepResult.Unreadable` and continues rather than failing the whole scan. Spec's Edge Cases do not mention malformed files explicitly (out of scope for the spec's business framing) but the underlying scan has the same failure mode `Grep` already handles.

**Decision**: `service.Match`/`kernel.MatchResult` carry the identical `Unreadable []string` field and behavior as `kernel.GrepResult`, for the same reason.

**Rationale**: Reuse, not reinvention — this is Principle V's "single, shared enumeration behavior" applied consistently rather than `node_match` inventing a second, divergent policy for a condition `Grep` already solved correctly.

**Alternatives considered**: Failing the whole `node_match` call on the first unreadable file was rejected — it would make one malformed node file in an otherwise-healthy graph break every `node_match` query, a regression relative to `node_grep`'s existing tolerance.

## D6 — No Cobra command in this release

**Problem**: Every prior filter-accepting capability (`arc grep`/`arc subgraph`) has both a Cobra command and an MCP tool. `node_match` has no CLI-side counterpart in VISION.md's roadmap (`arc list [<filter>]` is a separate, not-yet-implemented roadmap item with different output shape — node listing, not fact listing).

**Decision**: This feature adds `node_match` to `arc serve` only. No new Cobra subcommand is introduced.

**Rationale**: Explicit user instruction for this feature. This also matches existing precedent: `context_retrieve` and `schema` are both already MCP-only, with `NewServeCmd`'s own `Long` text stating so ("`context_retrieve` ... and `schema` ... are MCP-only, with no Cobra command of their own").

**Alternatives considered**: Adding `arc match <filter-flags>` in the same PR was rejected per explicit user instruction reserving it for "a future release."
