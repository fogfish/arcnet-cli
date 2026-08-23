# Contract C2 — Seeded Schema Tree

**Authority**: ARCNET-CORE v0.11 §9, §10, §11 · **Requirements**: FR-003, FR-005, FR-007–FR-009,
FR-011, FR-013, FR-014, SC-002, SC-007

## C2.1 Golden snapshot

`schema/service.Seed()` output is fixed byte-for-byte by `testdata/golden/schema/`, one file per
map entry, at the same relative paths `Seed()` keys.

- Asserted by `internal/app/schema/service/seed_golden_test.go`.
- Refreshed by `go test ./internal/app/schema/service -run TestSeedGolden -update`.
- A diff in this tree is a **reviewable change to what every new graph contains** and MUST be
  read as such in review, never regenerated reflexively.

This snapshot is the regression net `CORE-FIX.md` §7 identified as the highest-value artifact of
the whole v0.11 conformance program: findings B3 and B4 survived four spec revisions because the
seeded vocabulary was only ever reviewed as diffs of Go map literals.

## C2.2 Invariants over the seeded tree

Asserted directly, independent of the golden files, so a careless `-update` cannot mask a
regression:

| # | Invariant | Requirement |
| --- | --- | --- |
| C2.2a | Every `_schema/Property/*.md` declares a `merge` drawn from C1.1 | FR-001, SC-002 |
| C2.2b | No `_schema/Class/*.md` carries a `merge` front-matter key | FR-005, SC-002 |
| C2.2c | Every predicate named in any `Class` document's `required::`/`optional::`/`subClassOf::` bullets has its own `_schema/Property/` document | SC-003 |
| C2.2d | `_schema/Property/` contains documents for `author`, `about`, `genre` | FR-011 |
| C2.2e | `abstract`, `description`, `definition`, `relevance` each declare `firstWriteWin` | FR-013 |
| C2.2f | `cites` declares `union` | FR-014 |
| C2.2g | `Node.md` carries no `required::` bullet | FR-007 |
| C2.2h | `Timeline.md`'s only `required::` bullet is `cites` | FR-009 |
| C2.2i | `Source.md`'s `required::` bullets are exactly `title`, `published`, `abstract`, `mentions` | FR-008 |
| C2.2j | Every seeded document round-trips `ParseNode`→`RenderNode` byte-identically | — |

## C2.3 Self-consistency

A graph produced by `arc init` MUST lint clean (SC-003). C2.2c is the invariant that makes this
hold: the seeded tree is closed under predicate reference.

Note that `Class.md` gaining `subClassOf` in its Optional list is required for C2.2c's
converse — the tool writes `subClassOf::` bullets onto five seeded `Class` documents while
`Class` itself never declared the predicate, which is a `typeOptional` violation the seeded tree
inflicts on itself today.
