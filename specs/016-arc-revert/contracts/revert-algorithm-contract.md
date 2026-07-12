# Contract: `internal/app/graph/service.Revert` reconciliation algorithm

This is the internal contract `service.Revert` guarantees to its caller (`component.Revert` → `cmd/arc/graph/revert.go`) and to its own tests, mirroring `specs/012-predicate-merge-policies/contracts/merge-behavior-contract.md`'s shape for this feature's own crux logic.

## Signature

```go
// Revert locates sourceID's ingest commit (D1), and — unless the source
// node no longer exists (D2, FR-003) — retracts that patch's contribution
// from the graph via whichever of the two reconciliation approaches
// (D3/D4 whole-commit, or D5-D9 per-node) the ingest commit's current
// eligibility calls for, producing exactly one commit. Any failure before
// that commit leaves no removed node file and no rewritten node content
// behind (FR-016).
func Revert(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, index core.Index, dir, sourceID string) (kernel.RevertResult, error)
```

## Top-level decision (research.md D1-D5)

```
hashes := vcs.CommitsMatching(dir, "Source-Id: " + sourceID)
if len(hashes) == 0 { refuse FR-002 }
if len(hashes) > 1  { refuse, integrity anomaly }
ingestHash := hashes[0]

tracked := vcs.IsTracked(dir, "sources/" + sourceID + ".md")
if !tracked { return Skipped result, FR-003 }

paths := vcs.ChangedPaths(dir, ingestHash)
eligible := true
for _, p := range paths {
    touching := vcs.CommitsTouching(dir, p)
    if touching[0] != ingestHash { eligible = false }
}

if eligible {
    // D4 — whole-commit path
    newHash := vcs.RevertCommit(dir, ingestHash)
    return RevertResult{Approach: "whole-commit", CommitHash: newHash, ...}
}

// D5-D9 — per-node path, one pass over `paths`
for _, p := range paths {
    touching := vcs.CommitsTouching(dir, p)
    if len(touching) == 1 {
        removeNode(p)       // D6, always includes the source node itself
    } else {
        reconcileShared(p, ingestHash)  // D7/D8/D9
    }
}
vcs.StageAll(dir); vcs.Commit(dir, perNodeCommitMessage(...))
```

**Invariant**: the source node (`sources/<sourceID>.md`) is always a member of `paths` (`arc apply` always writes it, spec 003 FR-004) and is always exclusively owned (research.md D2's assumption) — so every revert, whole-commit or per-node, always removes it. A revert can therefore never leave `sources/<sourceID>.md` behind, which is exactly what D2's idempotency check for the *next* invocation relies on.

## `removeNode(path)` (research.md D6)

1. `store.Remove(path)`.
2. `rev := buildReverseIndex(enumerateNodes(store))` (reused from `subgraph.go`, built once per `Revert` call, not once per removed node — a removal earlier in the same pass must not resurrect a stale reverse-index entry for a node removed later in the same pass; the index is rebuilt from the *pre-revert* snapshot taken once at the start, and every referrer touched by more than one removal in the same pass is filtered against the full removed-set in one rewrite, not once per removed target).
3. For every id in `rev[removedID]`: read the referrer, drop every `Link` in its `Edges` whose `Target == removedID`, and rewrite — via `core.RenderNode` for an ordinary node, via `removeTimelineEntry` (D6) for a `@type: timeline` referrer.

## `reconcileShared(path, ingestHash)` (research.md D7-D9)

1. Parse the node at `path` into `core.Node`.
2. For each Texts key in `renderNodeBody`'s own physical order (leading key, other keys alphabetically, trailing key — `markdown.go:768-808`):
   a. If the current value matches `conflictMarker`'s shape (D8): resolve via D8(a)/(b)/neither.
   b. Else: intersect `Blame(path)`'s `ingestHash`-attributed line numbers against this key's paragraph byte range (D7); drop any matched paragraph.
3. If no key changed (D9): `NodeOutcome.Kind = "unchanged"`, no write.
4. Else: rewrite via `core.RenderNode`; `NodeOutcome.Kind = "reconciled"`.

**Never touched in this function**: `Attrs`, `Edges`, `HRefs` — FR-011's explicit scope guard. A shared node's predicates and links are read but never written by `reconcileShared`.

## Behavior contract: what a caller can rely on

1. **Exactly one commit** (FR-015): whichever path is taken, `Revert` either produces exactly one commit and a non-empty `CommitHash`, or returns an error with the graph and history untouched — never a partial state.
2. **A later patch's contribution is never lost** (FR-011, spec SC-003): `reconcileShared` never writes to `Attrs`/`Edges`/`HRefs`, and the Texts-key paragraph removal in step 2 only ever drops paragraphs whose blamed commit is `ingestHash` — a paragraph blamed to any other commit is left in place, by construction of the intersection in step 2b.
3. **Idempotent on re-invocation for the same `sourceID`** (FR-003/SC-008, D2): the source node's removal in `removeNode` (guaranteed to run for every revert, per the invariant above) makes the *next* call's `IsTracked` check `false`, short-circuiting before `CommitsMatching`/`ChangedPaths` are even consulted.
4. **A conflict marker is never left malformed** (FR-013): D8(a)/(b) each replace a full marker with one side's plain text in one rewrite; there is no code path that strips only `<<<<<<< existing` or only `>>>>>>> <sourceID>` without also resolving the other delimiter.
5. **No distinctly-attributable content is ever an error** (FR-014, D9): `reconcileShared` returning `"unchanged"` is a normal, successful outcome, not a partial failure — `Revert`'s own error return is reserved for a git/filesystem operation failing, never for "nothing to remove."

## Reporter events (verbose report, FR-019, ADR 002 DS-06)

One `reporter.Step` per node touched by the per-node path (mirrors `apply.go:299-302`'s existing per-node/per-predicate report shape):

```
<path>: removed (<n> links swept)
<path>: reconciled (<n> paragraph(s) stripped)
<path>: unchanged (no attributable content)
```

Silent by default; revealed under `--verbose`, exactly like `apply`'s own per-node/per-predicate detail (spec 003 FR-021, spec 012 FR-017).
