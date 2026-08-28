# Phase 1 Data Model: MCP `node_match` Filter Tool

## `core.Fact` (new, `internal/core/filter.go`)

```go
// Fact is one (property, value) pair carried by a node that satisfied a
// Filter statement (specs/028-node-match-filter) — node_match's unit of
// evidence for why a node matched.
type Fact struct {
	Property string
	Value    string
}
```

- `Property` is `"type"`, an attribute name, or an edge's predicate name — the same vocabulary `Statement.match`'s `predicate` position already matches against.
- `Value` is the node's own type value, the attribute's value, or the edge's target — the same vocabulary `Statement.match`'s `target` position already matches against.
- No `Source`/node-id field: `Fact` describes *what* matched on a node, not *which* node — the owning node's id is carried separately by `kernel.MatchEntry` (below), keeping `Fact` reusable as a plain per-node value, independent of any particular node's identity.

## `Filter.MatchingFacts` (new method, `internal/core/filter.go`)

```go
// MatchingFacts returns every distinct fact on node — its synthesized type
// fact, an attribute fact, or an edge fact — that satisfies at least one
// statement in f, deduplicated by (Property, Value) and sorted for
// deterministic output (research.md D1/D3). Match mutates neither f nor
// node; MatchingFacts does not itself require f.Match(node) — callers that
// want facts only for nodes satisfying every statement call Match first
// (research.md D4).
func (f Filter) MatchingFacts(node Node) []Fact
```

- Pure function, no I/O, same contract shape as the existing `Filter.Match`/`Traversal`/`Narrowing`.
- Empty `f.Statements` (or a `node` satisfying no statement) yields `nil`.

## `kernel.MatchEntry` / `kernel.MatchResult` (new, `internal/app/graph/kernel/match.go`)

```go
// MatchEntry is one reported fact justifying a node's inclusion in
// node_match's result — the row rendered as "| id | property | value |".
type MatchEntry struct {
	ID       string `json:"id"`
	Property string `json:"property"`
	Value    string `json:"value"`
}

// MatchResult is the domain value component.go's Match returns to
// cmd/arc/graph, rendered by the MCP handler as a markdown table.
type MatchResult struct {
	Root       string       `json:"root"`
	Matches    []MatchEntry `json:"matches"`
	Unreadable []string     `json:"unreadable"`
}
```

- Mirrors `kernel.GrepResult`/`kernel.Match`'s existing shape and naming convention exactly (`Root`, `Matches`, `Unreadable`).
- `MatchEntry` is `kernel`'s own DTO, distinct from `core.Fact` — `kernel` types are what `cmd/` renders, `core` types are the pure domain value; `service.Match` is the seam that maps one `node.ID` + `[]core.Fact` into `[]kernel.MatchEntry`, following the same pattern `service.Grep` uses to map `grep.Search`'s internal `Match` type into `kernel.Match`.

## `service.Match` (new, `internal/app/graph/service/match.go`)

```go
// Match enumerates every node file in the graph rooted at dir, keeps only
// nodes fully satisfying filter (core.Filter.Match), and reports every
// distinct fact on each kept node that satisfied at least one filter
// statement (core.Filter.MatchingFacts). filter MUST carry at least one
// statement — an empty/absent filter is rejected via ErrEmptyFilter before
// any node file is opened, since a vacuously-matching filter would carry
// no evidence for MatchingFacts to report (research.md D2, spec FR-005).
// A node file that cannot be opened or parsed is recorded in the result's
// Unreadable list and excluded from the scan (research.md D5).
func Match(ctx context.Context, mounter fsys.Mounter, filter core.Filter, dir string) (kernel.MatchResult, error)
```

Reuses, unchanged: `guardIsGraph`, `walkNodeFiles`, `readGrepNode` (all already defined in `apply.go`/`grep.go`, same package).

## `service.ErrEmptyFilter` (new sentinel, `internal/app/graph/service/errors.go`)

```go
// ErrEmptyFilter is node_match's own sentinel (specs/028-node-match-filter,
// research.md D2): a missing or zero-statement filter is rejected outright
// rather than silently matching (and reporting facts for) every node.
ErrEmptyFilter = faults.Type("filter must contain at least one statement")
```

## MCP wire shape (new, `cmd/arc/graph/serve.go`)

```go
// nodeMatchArgs is node_match's input schema. Unlike node_grep/subgraph_get/
// context_retrieve, Filter is REQUIRED (non-pointer, no "omitempty") — spec
// FR-001/FR-005.
type nodeMatchArgs struct {
	Filter mcpFilter `json:"filter" jsonschema:"required filter; at least one statement"`
}
```

- Reuses `mcpFilter`/`mcpStatement`/`stringOrArray`/`toCoreFilter`/`inputSchemaFor` from `specs/027-triple-filter-model` verbatim — no new wire type for statements.
- `toCoreFilter` is called on `&args.Filter` (a concrete, always-non-nil value here, unlike the other tools' `*mcpFilter`); the empty-statements check happens in `service.Match`, not in the handler, so the validation lives in one place regardless of transport.

## Relationships

```
core.Filter (027) ──Match(node)──────────► bool
             └─────MatchingFacts(node)───► []core.Fact  (new)

service.Match(filter, dir)
    │ walkNodeFiles + readGrepNode (reused from grep.go)
    │ for each node: filter.Match(node) ⇒ keep; filter.MatchingFacts(node) ⇒ facts
    ▼
kernel.MatchResult{ Matches: []kernel.MatchEntry{ID, Property, Value} }
    │ rendered by renderFactTable (cmd/arc/graph/serve.go, mirrors renderMatchTable)
    ▼
MCP node_match reply: "| id | property | value |" markdown table
```
