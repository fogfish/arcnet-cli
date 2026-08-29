# Phase 1 Data Model: MCP `node_links` and `node_backlinks` Tools

## `core.Link` (existing, reused unchanged — `internal/core/ast.go`)

```go
type Link struct {
	Predicate string `json:"predicate,omitempty"`
	Target    string `json:"target"`
	Alias     string `json:"alias,omitempty"`
}
```

- Reused as-is for `node_links`' output entries. `Alias` is populated for some `HRefs` occurrences (`[[Target|alias]]` markup) but is not part of `node_links`' rendered shape — the MCP table renders only `predicate`/`target`.
- `Predicate` empty means a bare inline reference with no explicit relation type stated (spec 030 Edge Cases) — rendered as an empty table cell, never dropped.

## `kernel.BacklinkEntry` (new, `internal/app/graph/kernel/links.go`)

```go
// BacklinkEntry is one reported relation from another node onto the
// queried node — the row rendered as "| source | predicate |" by
// node_backlinks (specs/030-mcp-node-links-backlinks).
type BacklinkEntry struct {
	Source    string `json:"source"`
	Predicate string `json:"predicate,omitempty"`
}
```

- Mirrors `service/subgraph.go`'s already-existing private `backlinkEdge{Source, Predicate}` shape exactly; `kernel.BacklinkEntry` is the exported, `cmd/`-facing equivalent, distinct from `backlinkEdge` (which stays private to `Subgraph`'s own reverse-index bookkeeping).
- No `Target` field: the target is always the node the caller queried (`id`), carried implicitly by the request, not repeated per entry — matching spec 030's `{source, predicate}` shape exactly.

## `service.NodeLinks` (new, `internal/app/graph/service/links.go`)

```go
// NodeLinks mounts dir, enumerates and indexes every node (reusing
// enumerateNodes/guardIsGraph, unchanged from NodeGet's own precedent),
// looks up id, returning ErrSeedNotFound on a miss, and reports every
// outgoing relation on that node — its own Edges followed by its own
// HRefs (nodeRelations; spec 030 FR-002/FR-008) — as []core.Link.
func NodeLinks(ctx context.Context, mounter fsys.Mounter, dir, id string) ([]core.Link, error)
```

## `service.NodeBacklinks` (new, `internal/app/graph/service/links.go`)

```go
// NodeBacklinks mounts dir, enumerates and indexes every node, looks up
// id (ErrSeedNotFound on a miss — an unknown id is distinct from a valid
// id with zero backlinks, spec 030 FR-005/FR-006), builds the Edges+HRefs
// reverse index (buildRelationReverseIndex — NOT Subgraph's Edges-only
// buildReverseIndex), and reports every relation elsewhere in the graph
// whose target is id, as []kernel.BacklinkEntry.
func NodeBacklinks(ctx context.Context, mounter fsys.Mounter, dir, id string) ([]kernel.BacklinkEntry, error)
```

## `nodeRelations` (new, package-private, `internal/app/graph/service/links.go`)

```go
// nodeRelations returns every outgoing relation n carries — its own Edges
// followed by its own HRefs (spec 030 Assumptions: node_links/
// node_backlinks deliberately treat inline prose references as relations,
// unlike nodeTargets, which stays Edges-only for Subgraph's traversal).
func nodeRelations(n core.Node) []core.Link {
	return append(append([]core.Link(nil), n.Edges...), n.HRefs...)
}
```

## `buildRelationReverseIndex` (new, package-private, `internal/app/graph/service/links.go`)

```go
// buildRelationReverseIndex mirrors buildReverseIndex (subgraph.go), but
// iterates nodeRelations(n) instead of n.Edges alone, so node_backlinks
// reports HRefs-sourced relations too (spec 030 FR-004/FR-008). Reuses
// the existing reverseIndex/backlinkEdge types unchanged; Subgraph's own
// buildReverseIndex call is untouched.
func buildRelationReverseIndex(index nodeIndex) reverseIndex {
	rev := reverseIndex{}
	for id, n := range index {
		for _, e := range nodeRelations(n) {
			rev[e.Target] = append(rev[e.Target], backlinkEdge{Source: id, Predicate: e.Predicate})
		}
	}
	return rev
}
```

## MCP wire shape (new, `cmd/arc/graph/serve_tool_node_links.go` / `serve_tool_node_backlinks.go`)

```go
// nodeLinksArgs / nodeBacklinksArgs — identical single-field shape to
// nodeGetArgs (serve_tool_node_get.go).
type nodeLinksArgs struct {
	ID string `json:"id" jsonschema:"the node's basename, e.g. the value returned as \"@id\" by node_grep/node_match"`
}
type nodeBacklinksArgs struct {
	ID string `json:"id" jsonschema:"the node's basename, e.g. the value returned as \"@id\" by node_grep/node_match"`
}
```

- Both mirror `nodeGetArgs` exactly — one required `id` string, no filter/scope parameter (spec Assumptions, plan Constraints).

## Relationships

```
core.Node.Edges, core.Node.HRefs (existing)
    │ nodeRelations(n)  ────────────────────────────► []core.Link   (new, node_links' own node)
    │
    │ buildRelationReverseIndex(index)  ─────────────► reverseIndex  (new — Edges+HRefs, node_backlinks only)
    ▼                                                    │
service.NodeLinks(id)   ──► []core.Link                  │ rev[id]
service.NodeBacklinks(id) ─────────────────────────────► []kernel.BacklinkEntry{Source, Predicate}
    │                                                         │
    │ renderLinkTable (cmd/arc/graph/serve_tool_node_links.go)│ renderBacklinkTable (serve_tool_node_backlinks.go)
    ▼                                                         ▼
MCP node_links reply: "| predicate | target |"    MCP node_backlinks reply: "| source | predicate |"
```

`Subgraph`'s own `nodeTargets`/`admittedEdges`/`buildReverseIndex`/`admittedBacklinks` (subgraph.go) are unmodified and remain Edges-only — no relationship to the above.
