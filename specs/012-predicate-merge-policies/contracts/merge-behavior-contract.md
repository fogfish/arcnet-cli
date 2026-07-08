# Contract: `internal/core.Merge` Per-Predicate Dispatch

This feature has no CLI-visible contract change (no new flag, no `--json` schema change, no new command). The contract that matters is the internal `core.Merge` function signature and behavior, since it's what `internal/app/graph/service.Apply` and its tests depend on, and what a future consumer (a second use-case wanting node-merge behavior) would call.

## Signature

```go
// Merge reconciles incoming into existing. Every predicate present in
// Attrs, Texts, Edges, or HRefs on either side is reconciled individually
// according to its own MergeOp, looked up in index.Predicates — never by
// one behavior applied to the whole node. A predicate absent from
// index.Predicates falls back to MergeUnion (research.md D6). Published is
// reconciled the same way, keyed by index.Predicates["published"].Merge.
// existing is the zero Node (ID == "") only when no node with incoming's
// identity exists yet — callers treat that case as a plain create, never
// calling Merge. conflicts lists every predicate name whose value was
// flagged (firstWriteWin/fillIfEmpty divergence only); empty when nothing
// diverged. Merge performs no I/O and is commutative and idempotent for
// every MergeOp except lastWriteWin, which is intentionally
// application-order-sensitive (research.md D5a).
func Merge(existing, incoming Node, index Index, sourceID string) (Node, []string, error)
```

Removed: the four-argument `Merge(existing, incoming Node, op MergeOp, sourceID string)` form and every `MergeOp`-branching `switch` inside it. `ErrUnknownMergeOp` is retired as a `Merge`-level error (there is no longer a single top-level op to be "unknown" — an individual predicate's unrecognized `MergeOp` value can only originate from a schema document that already failed `internal/app/schema/service.decodePredicateDef`'s validation before `Merge` is ever reached, per spec FR-013's boundary: dispatch consumes an already-valid index).

## Caller contract (`internal/app/graph/service.Apply`)

Before (today):

```go
typeDef, ok := index.Types[node.Type]
op := typeDef.Merge
if !ok {
    op = core.MergeUnion
    // ...warn, RegisterType...
}
// ...
merged, conflicts, err = core.Merge(existing, node, op, patch.Document)
```

After:

```go
typeDef, ok := index.Types[node.Type]
if !ok {
    // ...warn, RegisterType... (unchanged — still needed for the
    // unrecognized-kind warning; typeDef.Merge is simply never read)
}
// ...
merged, conflicts, err = core.Merge(existing, node, index, patch.Document)
```

`kernel.ApplyResult`'s shape (`Created`/`Merged`/`Conflicts`/`Warnings`/`Timeline`/`CommitHash`) is unchanged. `result.Conflicts` still means "this node path had at least one flagged predicate" — same meaning, now populated from per-predicate `flagOnDiverge` outcomes instead of a whole-node scalar comparison. The commit message format (`buildCommitMessage`) is unaffected.

## Behavior contract: what a caller can rely on

1. **Per-predicate independence** (spec FR-001): the outcome for predicate X on a merged node depends only on X's own declared `MergeOp` and its own existing/incoming values — never on the node's `@type`, and never on any other predicate's outcome in the same merge.
2. **Conflict marker scope** (spec FR-011/FR-012): a conflict marker (`<<<<<<< existing\n...\n=======\n...\n>>>>>>> <sourceID>`, unchanged format) appears if and only if the touched predicate's declared behavior is `firstWriteWin`, or `fillIfEmpty` after its first value is set, and the two sides' values genuinely differ (not merely non-identical formatting — same divergence definition as today's `mergeScalarString`/`mergeScalarPredicate`).
3. **Idempotency** (spec FR-010): `Merge(Merge(existing, incoming, index, id), incoming, index, id)` produces the same result as `Merge(existing, incoming, index, id)`, for every `MergeOp`.
4. **Commutativity, with one named exception** (spec FR-010/FR-007): for every `MergeOp` except `lastWriteWin`, `Merge(Merge(n, a, index, idA), b, index, idB)` on independent predicates equals `Merge(Merge(n, b, index, idB), a, index, idA)`. For `lastWriteWin`, the result is whichever of `a`/`b` was applied last — by design, not a defect.
5. **Purity**: `Merge` remains a pure function — no `context.Context`, no filesystem/VCS access, matching ADR 001's domain-layer constraint.
