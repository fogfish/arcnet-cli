# Contract C1 — Merge Vocabulary

**Authority**: ARCNET-CORE v0.11 §9.3 · **Requirements**: FR-001, FR-002, FR-004

## C1.1 The closed set

`core.MergeOp` has exactly six inhabitants. Any seventh is a contract violation.

```
immutable · union · firstWriteWin · fillIfEmpty · lastWriteWin · append
```

**Verified by**: `internal/core/ast_test.go` asserts the six string values *and* asserts, by
exhaustive comparison against a literal set, that no seventh constant exists.

## C1.2 Rejection of an out-of-menu value

Given a `_schema/Property/<name>.md` whose `merge` front-matter field holds a value outside
C1.1, `schema/service.Resolve` MUST fail with `ErrSchemaInvalid`, naming the offending
document's path.

> **Simplified 2026-08-23, post-implementation.** An earlier revision of this contract required
> the message to also name the offending value, the six legal values, and `arc upgrade` as a
> remedy — supporting a dedicated `arc upgrade` migration command for graphs seeded by a
> previous release. That command was implemented, then removed: `arc` is pre-1.0/experimental,
> and the compatibility/migration machinery it required was judged unnecessary tech debt. A
> graph carrying `validatedOverwrite` (e.g. on `scoreZ.md`/`scoreC.md`) now simply fails to load,
> with no remedy beyond re-initializing.

## C1.3 Tolerance on `Class` nodes

Given a `_schema/Class/<name>.md` carrying **any** `merge` front-matter value — legal, illegal,
or malformed — `Resolve` MUST succeed and MUST ignore the field.

No lint rule reports it. This falls out of the data model itself (`core.TypeDef` carries no
`Merge` field to populate), not from dedicated compatibility code.

## C1.4 Dispatch behaviour after removal

`core.Merge`'s scalar dispatch classes (`internal/core/merge.go` `mergeScalar`) lose
`validatedOverwrite` from the **freeze** class. No other class changes.

Consequence for `scoreZ`/`scoreC`: they move from *freeze* (a value, once written, was
permanent) to *alwaysOverwrite*. This is a behaviour change and a bug fix — their own registered
descriptions claim they are "recomputed by a validation/ingest pass", which the freeze class
made impossible.
