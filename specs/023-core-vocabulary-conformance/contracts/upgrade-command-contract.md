# Contract C3 — `arc upgrade`

**Requirements**: FR-017–FR-024 · **Design**: research.md D10, D11, D12

## C3.1 Synopsis

```
arc upgrade [--dry-run] [--json] [--verbose] [--quiet]
```

> Bring an existing graph's built-in schema up to date with this release of `arc`.

Registered at `cmd/arc/ctrl/upgrade.go`, a top-level sibling of `init` (research D11).

## C3.2 Execution order — normative

The order is the contract, not an implementation detail: it is what lets the command run on a
graph whose schema no longer validates (FR-022).

| Step | Action | Reads schema? |
| --- | --- | --- |
| 1 | Resolve graph root; fail `ErrNotInitialized` if `.arc/` absent | no |
| 2 | Compute `schema.Seed()` — pure, no I/O | no |
| 3 | Byte-compare each seeded path against its on-disk file | **no — no decode** |
| 4 | Write replacements/additions; delete built-in documents this release no longer seeds | no |
| 5 | `schema.Resolve()` the now-corrected tree | yes |
| 6 | Scan content nodes for prose-drift candidates using the Index from step 5 | yes |
| 7 | Stage and commit once, message `graph(migrate): …` | no |

Steps 1–4 never call `decodePredicateDef`, so `merge: validatedOverwrite` on disk cannot block
the command that removes it.

## C3.3 Scope of writes

| Path | Treatment |
| --- | --- |
| `_schema/Property/<name>.md` where `<name>` ∈ `CorePredicateDefs` | replaced outright |
| `_schema/Class/<name>.md` where `<name>` ∈ `CoreTypeDefs` | replaced outright |
| Any other file under `_schema/` | **untouched** (FR-020) |
| Any file outside `_schema/` | **untouched** (FR-019) |

"Replaced outright" (FR-018), not merged: `required`/`optional` are `union` and `role`/`merge`
are `immutable`, so the merge path can express none of this feature's changes — every one is a
retraction or an overwrite.

A hand-edited built-in document **is** replaced. The built-in vocabulary is not a starting point
to be customized in place; an author extending the vocabulary adds their own documents, which
C3.3 row 3 preserves.

## C3.4 Idempotency and the empty case

Step 3 finding no differences ⇒ no write, no commit, `CommitHash == ""`, exit 0 (FR-021, FR-024).
Running `arc upgrade` twice is indistinguishable from running it once.

Human output for the empty case mirrors `arc apply schema`'s existing phrasing:

```
✓ graph schema is already up to date — nothing to commit
```

## C3.5 `--dry-run`

Reports exactly what steps 3 and 6 found; performs no write, creates no commit, exits 0.
`DryRun: true` in the JSON result.

## C3.6 Commit

Exactly one commit (FR-021), shaped per CORE §13.3:

```
graph(migrate): adopt ARCNET-CORE v0.11 built-in schema

Replaced 8 type and 57 predicate schema documents.
```

## C3.7 Prose-drift report

Step 6 lists every content node where a `firstWriteWin`-declared text predicate holds more than
one paragraph — the shape only the previous `append` behaviour could have produced (research
D12). Reported, never repaired (FR-023); does not affect exit code.

```
⚠ 3 nodes may carry prose accumulated by the previous merge behaviour — review manually:
    Source/rescorla-2026-tls13.md    (abstract: 2 paragraphs)
    Entity/Transport Layer Security.md (definition: 3 paragraphs)
    Reference/RFC 8446.md            (relevance: 2 paragraphs)
```

## C3.8 Failure and rollback

Any failure in steps 4–7 leaves no partial state: `service.Init`'s existing `rollback` discipline
applies (FR-013 of spec 002). A failure in step 5 after step 4 wrote is a programming error —
`Seed()` output is always resolvable — and surfaces as such rather than as a user-facing
schema error.

## C3.9 Out of scope for this command

- Re-typing, re-filing, or rewriting content nodes (that was spec 022's declined migration).
- Repairing accumulated prose (C3.7).
- Upgrading a graph's folder layout to §6's type-named folders.
