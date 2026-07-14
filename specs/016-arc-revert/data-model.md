# Data Model: `arc revert`

## `kernel.RevertResult`

New file `internal/app/graph/kernel/revert.go`, sibling to `kernel/apply.go`, rendered by `bios.Registry[RevertResult]` exactly as `ApplyResult` is today.

```go
// RevertResult is the domain value component.go's Revert returns to
// cmd/arc/graph, rendered by bios.Registry[RevertResult].
type RevertResult struct {
	// Document is the retracted patch's source id.
	Document string `json:"document"`
	// Skipped is true when the source node no longer exists (Clarifications
	// Session 2026-07-12, FR-003); every other field is zero-valued then.
	Skipped bool `json:"skipped"`
	// Approach is "whole-commit" (D3/D4) or "per-node" (D5-D9) — FR-018.
	Approach string `json:"approach"`
	// Removed holds node counts by kind, deleted outright (FR-009).
	Removed map[string]int `json:"removed"`
	// Reconciled holds node counts by kind that had only the reverted
	// patch's own text content stripped, kept otherwise intact (FR-012).
	Reconciled map[string]int `json:"reconciled"`
	// LinksRemoved is the count of Edges dropped across every referrer
	// node touched by a removed node's backlink sweep (FR-010).
	LinksRemoved int `json:"linksRemoved"`
	// Nodes holds one NodeOutcome per node the revert touched, in
	// deterministic (path-sorted) order — populated always, surfaced only
	// under --verbose (FR-019, mirrors apply's per-predicate report
	// precedent, spec 012 FR-017).
	Nodes []NodeOutcome `json:"nodes"`
	// CommitHash is the short hash of the single resulting commit; empty
	// when Skipped.
	CommitHash string `json:"commit"`
}

// NodeOutcome records one node's fate within a revert (FR-019).
type NodeOutcome struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"` // "removed" | "reconciled" | "unchanged"
	Detail string `json:"detail,omitempty"` // e.g. "3 links removed", "1 paragraph stripped"
}
```

`Approach` values: `"whole-commit"` (D4's `git revert`) or `"per-node"` (D5-D9's node-by-node reconciliation) — FR-018's "which approach was used" requirement. A revert that lands on the whole-commit path never populates `Removed`/`Reconciled`/`Nodes` individually — see Reconciliation Approach below.

## `port.BlameLine`

New type in `internal/app/graph/port/vcs.go`, alongside the widened `VCS` interface:

```go
// BlameLine is one current line of a node file's git-blame attribution
// (D7) — content is never needed, only which commit last touched it.
type BlameLine struct {
	Number int
	Commit string
}
```

## `port.VCS` additions

`internal/app/graph/port/vcs.go`'s `VCS` interface gains six methods (research.md D1/D3/D4/D7/D8; full git-command mapping in `contracts/vcs-port-contract.md`):

```go
type VCS interface {
	IsTracked(ctx context.Context, dir, path string) (bool, error)
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)

	CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
	ChangedPaths(ctx context.Context, dir, hash string) ([]string, error)
	CommitsTouching(ctx context.Context, dir, path string) ([]string, error)
	RevertCommit(ctx context.Context, dir, hash string) (newHash string, err error)
	Blame(ctx context.Context, dir, path string) ([]BlameLine, error)
	ShowFile(ctx context.Context, dir, hash, path string) ([]byte, error)
}
```

## Domain entities (from spec.md, grounded in code)

- **Ingest Commit**: a git commit reachable from `CommitsMatching(dir, "Source-Id: "+id)`. Exactly one is expected; see D1's anomaly handling.
- **Exclusively-Owned Node**: a node file path `p` where `len(CommitsTouching(p)) == 1` (D5).
- **Shared Node**: a node file path `p` where `len(CommitsTouching(p)) > 1` and `ingestHash ∈ CommitsTouching(p)`.
- **Node Contribution**: the set of Texts-key paragraphs (D7) or conflict-marker sides (D8) attributable to `ingestHash` on a shared node — never Attrs/Edges/HRefs content (FR-011's scope guard).
- **Reconciliation Approach**: `"whole-commit"` when every path in `ChangedPaths(ingestHash)` passes D3's eligibility test; `"per-node"` otherwise. Computed once per revert, reported as `RevertResult.Approach`.
- **Link**: `core.Link{Predicate, Target, Alias}` (`internal/core/ast.go:34`) — unchanged type, reused as-is. A removed node's backlink sweep filters `Edges` by `Target == removedID`.
- **Revert Commit**: the single commit `vcs.RevertCommit` (whole-commit path) or `vcs.Commit` (per-node path, new message format below) produces.

## Per-node reconciliation decision table

| Node state (relative to `ingestHash`) | Action | `NodeOutcome.Kind` |
|---|---|---|
| Not present in `ChangedPaths(ingestHash)` | not touched | (absent from `Nodes`) |
| Exclusively owned (D5) | `store.Remove(path)`; sweep backlinks (D6) | `"removed"` |
| Shared, attributable text found (D7/D8) | strip only that text; rewrite via `core.RenderNode` (or `removeTimelineEntry` for a `timeline` referrer touched by the backlink sweep) | `"reconciled"` |
| Shared, no attributable text found (D9) | no-op | `"unchanged"` |

## Per-node commit message (per-node reconciliation path only)

Mirrors `apply.go`'s `buildCommitMessage` shape (`internal/app/graph/service/apply.go:473`) — same overall structure, new subject and a deliberately distinct trailer key:

```
graph(revert): <source-id> — per-node reconciliation

Removed: <n> nodes (<kind>: <n>, ...)
Reconciled: <n> nodes (<kind>: <n>, ...)
Links removed: <n>
Reverted-Document: <source-id>
```

The trailer is **`Reverted-Document:`, not `Source-Id:`**, and this is load-bearing, not cosmetic: `CommitsMatching`'s caller in `internal/app/lint/service/rules_history.go` and this feature's own D1 both match via `--fixed-strings --grep="Source-Id: <id>"`, a literal substring test. A trailer like `Reverted-Source-Id:` would still contain `Source-Id: <id>` as a substring and silently satisfy that grep — corrupting spec 003 FR-012's "exactly one ingest commit per document" invariant the moment a retracted document is later re-applied (out of this feature's scope, but a real future command) and re-triggering `arc lint`'s `RuleIngestCommit` as a false violation. `Reverted-Document:` shares no substring with `Source-Id:` and is used purely for human/`git log` traceability — never read back by D1 or D2, both of which use `sources/<id>.md` existence (D2) or the true ingest commit's own untouched `Source-Id:` trailer (D1).

The whole-commit path (D4) does not use this template; `git revert --no-edit` generates its own message.
