# Contract: Core Type Vocabulary (CORE §11, v0.11)

**Feature**: `022-reference-type-folders` · **Status**: normative for this feature

## C1. Five core content types

`Source`, `Entity`, `Resource`, `Timeline`, `Reference`. Each declares `Node` as its base type
via `subClassOf`. `Property` and `Class` remain the schema-mechanism types and are unchanged.

## C2. `Resource`

```yaml
"@id": Resource
"@type": Class
Requires: [text, tags, mentionedIn]
Optional: [notes]
subClassOf: [Node]
```

Meaning: an anonymous, tag-classified fragment of an **ingested** document's content that does
not warrant its own dedicated type — the staging ground from which a recurring pattern is later
promoted into a proper domain type.

`ref`, `relevance`, `url`, `authors`, `year`, `doi`, `status`, `isCitedBy` MUST NOT appear in
either list.

## C3. `Reference`

```yaml
"@id": Reference
"@type": Class
Requires: [title, ref, relevance]
Optional: [url, authors, year, doi, status, isCitedBy, notes]
subClassOf: [Node]
```

Meaning: an external work the graph points to but has **not** ingested, or a topic/area tracked
for reading or research. Records only enough to identify, locate, and justify keeping the
pointer.

## C4. Leading-prose predicate

A node's opening body prose is stored under, and read back from, exactly one predicate per type:

| `@type` | Leading | Trailing |
|---|---|---|
| `Source` | `abstract` | `notes` |
| `Entity` | `definition` | `notes` |
| `Resource` | `text` | `notes` |
| `Reference` | `relevance` | `notes` |
| *any other* | `text` | `notes` |

The write side and the read side MUST derive this from a single shared definition, so they
cannot disagree. Any duplicated copy of this table MUST agree with the canonical one for all
five core types, enforced by test.

## C5. Predicate registration

Every predicate named in C2 and C3 MUST already be registered in the seeded schema. This
feature introduces **no new predicate**. A freshly initialized graph MUST lint clean.

## C6. Seeding is replacement, never merge

The corrected definitions reach a graph only through `arc init`'s seed, which renders each type
definition and writes it to a fresh path. It MUST NOT be routed through the merge path.

`required` and `optional` carry `merge: union`, and union has no inverse. Merging a corrected
`Resource` onto a retired one yields a type requiring `ref + relevance + text + tags +
mentionedIn` — neither definition, and conformant to no revision of the specification. This is
why re-seeding cannot serve as a migration, and why the break is clean rather than incremental.

## C7. Merge behaviour

`Reference` carries `merge: firstWriteWin`, inherited from `Resource`'s former definition — the
op belonged to the external-work semantics, which moved wholesale. `Resource`'s own type-level
merge op is unchanged.
