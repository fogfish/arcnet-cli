# `legacy-graph` — the built-in vocabulary as a pre-023 release seeded it

A byte-exact, hand-preserved snapshot of `schema.Seed()`'s output **before**
`specs/023-core-vocabulary-conformance` corrected it, captured from the golden
baseline at the feature's first commit.

It is what `arc upgrade` exists to replace, and the only fixture that can prove it:

- `_schema/Property/scoreZ.md` and `scoreC.md` declare `merge: validatedOverwrite`,
  the retired seventh value — so a graph shaped like this **fails to load** once the
  merge menu is closed at six, which is precisely the deadlock `arc upgrade`'s
  write-before-resolve ordering (contract C3.2) resolves.
- All eight `_schema/Class/*.md` carry the retired type-level `merge` attribute.
- `Node.md` requires `published`/`created`; `Timeline.md` requires `granularity`,
  `cites`, `period`; `Reference.md` requires `title`, `ref`, `relevance`.
- `abstract`, `description`, `definition`, `relevance` declare `append`;
  `cites` declares `append`; `year` declares `fillIfEmpty`; `author`, `about`,
  and `genre` are absent entirely.

**Never regenerate this fixture.** It is a historical record, not a snapshot of
current behaviour — regenerating it against the corrected `Seed()` would make every
US5 test pass vacuously by comparing the current vocabulary against itself. The
snapshot that DOES track current behaviour is
`internal/app/schema/service/testdata/golden/schema/`.

The tree holds `_schema/` only; a consumer supplies folders, `.arc/`, git, and any
content nodes it needs.
