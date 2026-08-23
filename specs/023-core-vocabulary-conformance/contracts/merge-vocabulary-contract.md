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
C1.1, `schema/service.Resolve` MUST fail with `ErrSchemaInvalid`.

The rendered message MUST name, in this order:

1. the offending document's path,
2. the offending value,
3. the six legal values,
4. `arc upgrade` as the remedy.

Item 4 is mandatory: a graph seeded by a previous release carries `validatedOverwrite` on
`scoreZ.md`/`scoreC.md`, so this error is the *first thing* an existing user sees after
upgrading the binary. Without the remedy the failure is a dead end.

```
schema document invalid: _schema/Property/scoreZ.md declares merge "validatedOverwrite";
must be one of immutable, union, firstWriteWin, fillIfEmpty, lastWriteWin, append.
Run `arc upgrade` to bring this graph's built-in schema up to date.
```

## C1.3 Tolerance on `Class` nodes

Given a `_schema/Class/<name>.md` carrying **any** `merge` front-matter value — legal, illegal,
or malformed — `Resolve` MUST succeed and MUST ignore the field.

No lint rule reports it. Rationale: an un-upgraded graph carries the attribute on all eight
seeded type documents, where it is inert; reporting it would produce eight actionable-looking
violations with no user-visible consequence.

## C1.4 Dispatch behaviour after removal

`core.Merge`'s scalar dispatch classes (`internal/core/merge.go` `mergeScalar`) lose
`validatedOverwrite` from the **freeze** class. No other class changes.

Consequence for `scoreZ`/`scoreC`: they move from *freeze* (a value, once written, was
permanent) to *alwaysOverwrite*. This is a behaviour change and a bug fix — their own registered
descriptions claim they are "recomputed by a validation/ingest pass", which the freeze class
made impossible.
