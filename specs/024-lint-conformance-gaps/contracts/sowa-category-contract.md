# Contract C1 — Sowa Category Combination

**Authority**: ARCNET-CORE §10.2 · **Requirements**: FR-007, FR-008, FR-009, FR-010

## C1.1 The closed set of twelve

`sowaCategories` (`internal/app/lint/kernel/lint.go`, [data-model.md §1](../data-model.md)) has
exactly twelve rows. Any tuple not present, byte-for-byte, as one of the twelve rows is a
violation — including a tuple where each individual word belongs to the right word-group but the
four words are not one of the twelve rows together.

**Verified by**: an exhaustive unit test over all 144 positional combinations (3 × 2 × 2 × 12),
asserting exactly 12 pass and 132 fail.

## C1.2 Rejection of a structurally-valid-but-illegal tuple

Given an `Entity` node whose `category` is a well-formed 4-word list not present in C1.1,
`arc lint` MUST report a `RuleEntityCategory`-tagged violation naming:

1. the rejected tuple, verbatim, and
2. at least one legal tuple sharing the longest matching leading-word prefix with it (first match
   in table order on a tie — [research.md D2](../research.md)).

Example: `[independent, physical, continuant, purpose]` → suggests `[independent, physical,
continuant, object]` (3-word shared prefix; no other row shares more).

## C1.3 Wrong-length tuple keeps its own message

A `category` with a word count other than four MUST continue to produce the existing wrong-length
message, unaffected by C1.1/C1.2. This is `ValidSowaCategory`'s existing first check
(`len(words) != 4`), unchanged by this feature.

## C1.4 Detection only

Neither C1.2 nor C1.3 may rewrite, reformat, or otherwise modify the node file carrying the
violation. `arc lint` never has file-write access to a node it is checking.
