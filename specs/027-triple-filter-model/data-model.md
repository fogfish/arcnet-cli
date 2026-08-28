# Phase 1 Data Model: Triple Filter for Node Attributes and Graph Edges

All types below live in `internal/core` (the domain package every port/adapter already depends on — no `cmd/`, no `mcp`, no `cobra` imports, per Principle III). `internal/core.Filter` is a pure value type: no I/O, no mutation of its receiver or its argument, matching the existing contract (`internal/core/filter.go`'s current GoDoc: "Match mutates neither f nor node").

## Matcher

```go
// Matcher constrains one position (Source, Predicate, or Target) of a
// Statement. A zero-value Matcher is a wildcard: it matches every string.
// Otherwise it matches s iff s case-insensitively equals any Values entry,
// or s matches any Patterns entry — the two slices are OR'd together, and
// either or both may be populated (research.md D1).
type Matcher struct {
    Values   []string
    Patterns []*regexp.Regexp
}

// IsWildcard reports whether m constrains nothing (both slices empty).
func (m Matcher) IsWildcard() bool

// Match reports whether s satisfies m.
func (m Matcher) Match(s string) bool
```

**Validation rules**: none at construction time — an empty `Matcher{}` is a valid wildcard, not an error. Regexp compilation failures are caught by the CLI/MCP translation layer (`cmd/arc/graph/grep.go`'s `opts.build()`, `cmd/arc/graph/serve.go`'s `toCoreFilter()`) before a `Matcher` is ever constructed, exactly as today (`service.ErrInvalidAttrFlag`/`service.ErrInvalidFilterPattern`).

## Statement

```go
// Statement is one clause of a Filter: independently-wildcardable
// constraints on the (Source, Predicate, Target) triple of a fact.
type Statement struct {
    Source    Matcher
    Predicate Matcher
    Target    Matcher
}

// IsTraversalConstraint reports whether s constrains only Predicate
// (Source and Target both wildcard) — research.md D3.
func (s Statement) IsTraversalConstraint() bool
```

**Relationships**: A `Statement` is evaluated against one **Fact** at a time (see below). A zero-value `Statement{}` (all three positions wildcard) matches every fact — never produced by CLI/MCP translation, but not rejected either.

## Fact (not a persisted type — an evaluation-time triple)

Two kinds of `Node`-derived data are treated as facts, matching spec.md's Key Entities "Attribute Fact"/"Relation Fact":

| Fact source | Source | Predicate | Target |
|---|---|---|---|
| Synthesized type fact (research.md D2) | `node.ID` | `"type"` | `node.Type` |
| One `node.Attrs[name]` entry's stringified value (skips `Value == nil`, i.e. reference-valued predicates — mirrors today's `attrStrings`) | `node.ID` | `name` | stringified `Predicate.Value` |
| One `node.Edges` entry | `node.ID` | `edge.Predicate` | `edge.Target` |

No `Fact` Go type is introduced — `Filter.Match` iterates `node.Attrs`/`node.Edges` (plus the one synthesized type fact) directly and tests each `Statement` inline, exactly as today's `matchTypes`/`matchTags`/`matchAttrs`/`matchAttrPatterns` do per-axis. Introducing a materialized `[]Fact` slice was considered and rejected as unnecessary allocation for a value never inspected as a collection — `Filter.Match` only ever asks "does *some* fact satisfy *this* statement," never "list every fact."

## Filter

```go
// Filter is the optional, composable node-selection criteria shared by
// every VISION.md Filtering-section command and by arc serve's MCP tools.
// A zero-value Filter{} matches every node (unchanged contract).
type Filter struct {
    Statements []Statement
}

// Match reports whether node satisfies every statement in f (AND across
// statements; each statement independently satisfied by any one of the
// node's own attribute or edge facts — spec FR-005). Match mutates neither
// f nor node.
func (f Filter) Match(node Node) bool

// Traversal returns the subset of f.Statements that are traversal
// constraints (research.md D3) — used to gate BFS edge admission.
func (f Filter) Traversal() Filter

// Narrowing returns the subset of f.Statements that are NOT traversal
// constraints — used for flat-inclusion narrowing of already-reached,
// non-seed candidates. Traversal() and Narrowing() partition f.Statements;
// their statement counts sum to len(f.Statements).
func (f Filter) Narrowing() Filter
```

**State/lifecycle**: `Filter` has no lifecycle — it is constructed once per command/tool invocation (CLI flag parsing or MCP argument decode) and consumed read-only for the remainder of that call. No persistence, no serialization to disk (the MCP wire shape in `mcpFilter` is a distinct, adapter-local type — see Contracts).

**Replaces**: `Types []string`, `Tags []string`, `Attrs map[string]string`, `AttrPatterns map[string]*regexp.Regexp` — deleted outright, per spec's "replaces `core.Filter` 100%."

## CLI → Filter lowering (research.md D4)

| Flag | Occurrences | Statement(s) produced |
|---|---|---|
| `--type <v>` (repeatable) | all repeats | one `Statement{Predicate: {Values:["type"]}, Target: {Values: [v1, v2, ...]}}` |
| `--tag <v>` (repeatable) | each repeat | one `Statement{Predicate: {Values:["tags"]}, Target: {Values: [v]}}` per repeat |
| `--attr <name>=<value>` (repeatable, dedup by name — last write wins, unchanged) | each surviving map entry | one `Statement{Predicate: {Values:[name]}, Target: {Values: [value]}}` |
| `--attr <name>~=<pattern>` (repeatable, dedup by name) | each surviving map entry | one `Statement{Predicate: {Values:[name]}, Target: {Patterns: [pattern]}}` |
| `--predicate <name>` (repeatable, `arc subgraph` only — research.md D10, new) | all repeats | one `Statement{Predicate: {Values: [n1, n2, ...]}}` (`Source`/`Target` wildcard → a traversal constraint) |

## Traversal-aware subgraph service types (`internal/app/graph/service/subgraph.go`)

```go
// backlinkEdge is one incoming structural connection recorded by
// buildReverseIndex: the id of the node that carries the edge, and that
// edge's own predicate (research.md D5).
type backlinkEdge struct {
    Source    string
    Predicate string
}

// reverseIndex maps a target id to every backlinkEdge pointing at it.
type reverseIndex map[string][]backlinkEdge
```

`nodeTargets` is replaced by two small, pure helpers:

```go
// admittedEdges returns n's own outgoing edges whose (n.ID, edge.Predicate,
// edge.Target) triple satisfies scope (empty scope admits every edge —
// today's unscoped default, research.md D5).
func admittedEdges(n Node, scope Filter) []Link

// admittedBacklinks mirrors admittedEdges for the reverse direction, over
// one target id's recorded backlinkEdge list.
func admittedBacklinks(id string, rev reverseIndex, scope Filter) []backlinkEdge
```

`bfs`'s own signature (`neighbors func(id string) []string`) is unchanged; `Subgraph`/`ContextRetrieve` build their two direction-specific `neighbors` closures on top of `admittedEdges`/`admittedBacklinks`, extracting just the `Target`/`Source` id strings `bfs` expects.

## Key entity cross-reference (spec.md alignment)

| spec.md Key Entity | Data model realization |
|---|---|
| Filter | `core.Filter{Statements []Statement}` |
| Statement | `core.Statement{Source, Predicate, Target Matcher}` |
| Attribute Fact | one `node.Attrs[name]` entry (plus the synthesized type fact, D2), evaluated inline by `Match` |
| Relation Fact | one `node.Edges` entry (`core.Link`), evaluated inline by `Match` and by `admittedEdges`/`admittedBacklinks` |
| Relation Restriction | `Filter.Traversal()`'s result, consumed by `admittedEdges`/`admittedBacklinks` |
