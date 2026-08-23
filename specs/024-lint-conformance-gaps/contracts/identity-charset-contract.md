# Contract C2 — Node Identity Charset

**Authority**: ARCNET-CORE §7.1 · **Requirements**: FR-001 through FR-006, FR-010

## C2.1 The closed set of ten forbidden characters

```
/  \  :  *  ?  "  <  >  |  .
```

One shared primitive in `internal/core` ([research.md D6](../research.md), [data-model.md
§3](../data-model.md)) is the sole definition of this set; `arc lint` and `arc apply` both consume
it, never redeclare it.

**Verified by**: a unit test iterating each of the ten characters individually plus one legal
control identity, asserting each forbidden character alone is caught and the control is not.

## C2.2 `arc lint` — detection, every node, content and schema

Given any node — content (walked from the graph root) or schema (`_schema/Class/`,
`_schema/Property/`) — whose identity contains one or more characters from C2.1, `arc lint` MUST
report one `RuleIdentityCharset` violation naming, for **every** offending character found (not
only the first): the character itself and its 1-indexed rune position within the identity value
([research.md D5](../research.md)).

The identity value checked is the `@id`/map-key string itself, never the derived `<id>.md`
filename — an identity with no `.` is legal even though every node file ends in `.md`
(spec.md Edge Cases).

`arc lint` MUST NOT modify any file while performing this check (spec.md FR-006).

## C2.3 `arc apply` — rejection, whole-operation, before any write

Given a patch that would introduce or modify:

- a content node whose own identity contains a C2.1 character, **or**
- a not-yet-registered type whose name (`node.Type`) would become a new `Class` document's
  identity and contains a C2.1 character, **or**
- a not-yet-registered predicate whose observed name would become a new `Property` document's
  identity and contains a C2.1 character,

`arc apply` MUST reject the entire operation before any file from this patch is written, and MUST
leave the graph byte-for-byte unchanged — including any other node in the same patch that would
otherwise have been valid (spec.md FR-001, FR-003; [research.md D7](../research.md)).

The rejection message names every offending character and its 1-indexed rune position, worded
identically to C2.2's violation message for the same identity ([research.md D8](../research.md)).

**Out of scope**: `arc apply-schema` is a distinct command and patch format. This contract does
not extend to it (research.md D7).

## C2.4 Message wording parity

C2.2 and C2.3 render the same detail fragment for the same offending identity — precedent already
established by `ErrIdentityQuoting`/`RuleIdentityQuoting` sharing wording in this codebase
(`internal/core/errors.go:55`).
