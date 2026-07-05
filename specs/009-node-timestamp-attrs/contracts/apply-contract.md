# Apply Contract Delta: `internal/app/graph/service.Apply` / `internal/app/graph.Apply`

No change to the exported signature of either function:

```go
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, predicates map[string]bool, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error)
```

No change to `kernel.ApplyResult`'s fields, to `--json`/`--plain`/human output, or to the commit message format (`buildCommitMessage`) — this feature's effect is entirely inside the node files `Apply` writes, not in the result it reports back to the caller.

## New behavior, per invocation

1. **One Application Timestamp** is captured (`time.Now().UTC()`) once, near the top of `Apply`, and formatted once (`time.RFC3339`) to a `stamp` string reused for every node this invocation touches.
2. **On creating a new node** (`!existed`, per the existing per-node loop):
   - If the node is **not** a stub (`isStub`, data-model.md) — i.e. it carries any `Attrs`/`Text`/`Notes`/`HRefs`/`Edges`/`Links` beyond `ID`/`Kind` — its `Published` is set to `patch.Published` unless the node's own patch section already carried a non-zero `Published` of its own (research.md D11), and `Attrs["indexed"] = stamp`.
   - If the node **is** a stub, neither `Published` nor `indexed` is set — it is created exactly as it is today, before this feature.
3. **On merging into an existing node** (`existed`):
   - `core.Merge` runs exactly as before (this feature does not change when/how `Merge` is invoked, only what it now additionally does to `Published` internally, per the AST contract delta).
   - After `Merge` returns, `Apply` compares the rendered bytes of `existing` and `merged` (`nodeContentChanged`, research.md D6). If they differ, `merged.Attrs["updated"] = stamp`. If they are identical (a `"none"`-kind no-op, or any other op's re-contribution that adds nothing new), no `updated` is added and the file, once written, is byte-identical to what was already on disk.
4. **`_schema/nodes`/`_schema/predicates` documents** — created via the pre-existing, unmodified `schema.RegisterKind`/`RegisterPredicate` calls in this same loop — are entirely unaffected; that code path never constructs or writes a `core.Node` through the create/merge logic above (research.md D8).
5. **`ApplyResult.Created`/`.Merged` counters and each node's `outcome` string** (`reporter.Step`) are unchanged from spec 003 — incremented/set exactly as before, independent of whether `nodeContentChanged` found a real difference (research.md D9). A `"merged"`-reported node with no `updated` stamp is an expected, valid outcome (a no-op re-contribution), not a contradiction.

## Non-goals (explicitly out of scope, per spec.md)

- No new CLI flag, no new `--json` field, no new Reporter phase.
- No change to `internal/app/schema`'s `RegisterKind`/`RegisterPredicate`/`Seed`/`Resolve`.
- No change to timeline period file handling (`applyTimeline`/`upsertTimelinePeriod`) — timeline period files are not reconstructed through this create/merge path (spec 003 FR-025) and this feature does not touch them.
