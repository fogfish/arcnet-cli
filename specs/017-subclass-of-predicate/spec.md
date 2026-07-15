# Feature Specification: Type Inheritance via `rdfs:subClassOf`

**Feature Branch**: `017-subclass-of-predicate`

**Created**: 2026-07-14

**Status**: Draft

**Input**: User description: "define `rdfs:subClassOf` predicate for types (classes). The purpose of predicate to specify `C1 rdfs:subClassOf C2` implications. The class C1 inherits all properties and predicates from the class C2. The predicate is used to simplify schema definitions. It allows schema to define a multiple base types to be in-use by other types. The `rdfs:subClassOf` important for linting and all schema management operations."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reuse a shared base type instead of redeclaring its contract (Priority: P1)

A schema maintainer manages several related types that all share a common set of required and optional predicates (for example, several "creative work" subtypes that all require a `title` and permit an `abstract`). Instead of repeating that shared contract in every type's declaration, the maintainer declares one base type once, then declares each related type as `rdfs:subClassOf` that base type. Every subtype now behaves as if it directly declared every predicate the base type requires or permits, without the maintainer writing it out again.

**Why this priority**: This is the entire point of the feature — a schema maintainer who cannot reuse a shared contract across types is exactly the pain point this predicate exists to remove. Nothing else in this feature has value if a subtype doesn't actually inherit its base type's predicates.

**Independent Test**: Declare a base type with a required and an optional predicate, declare a second type as `rdfs:subClassOf` the base type with no predicates of its own, then confirm a node of the subtype is held to the same required/optional contract as a node of the base type.

**Acceptance Scenarios**:

1. **Given** a type `C2` whose schema requires predicate `p` and permits predicate `q`, **When** a type `C1` is declared `rdfs:subClassOf` `C2` with no predicates of its own, **Then** `C1`'s effective contract requires `p` and permits `q`, identical to `C2`'s.
2. **Given** a type `C1` that is `rdfs:subClassOf` `C2`, **When** a node declared with `@type` `C1` carries every predicate `C2` requires, **Then** that node satisfies `C1`'s required-predicate check exactly as it would if `C1` had declared those predicates directly.
3. **Given** a type `C1` that is `rdfs:subClassOf` `C2`, **When** a node declared with `@type` `C1` is missing a predicate `C2` requires, **Then** the missing-required-predicate check reports the violation against `C1`'s node, naming the inherited predicate.
4. **Given** a type `C1` that both declares its own required predicate `r` and is `rdfs:subClassOf` `C2` (which requires `p`), **When** the effective contract for `C1` is computed, **Then** it requires both `r` and `p`.

---

### User Story 2 - Combine several base types into one composed type (Priority: P1)

A schema maintainer defines a type that needs the combined contract of more than one existing base type — for example, a type that is both a citable work and a timestamped record. The maintainer declares multiple `rdfs:subClassOf` relationships on the same type, one per base type, and the resulting type's effective contract is the union of everything every base type contributes, plus anything the type declares itself.

**Why this priority**: Explicitly named in the request ("allows schema to define a multiple base types to be in-use by other types") — without multiple inheritance, a maintainer facing overlapping needs is forced back into full redeclaration, defeating User Story 1's purpose whenever a type needs more than one shared contract.

**Independent Test**: Declare two independent base types, each requiring a different predicate, declare a third type as `rdfs:subClassOf` both, and confirm the third type's effective contract requires predicates from both bases.

**Acceptance Scenarios**:

1. **Given** type `C2` requires predicate `p` and type `C3` requires predicate `s`, **When** type `C1` is declared `rdfs:subClassOf` both `C2` and `C3`, **Then** `C1`'s effective contract requires both `p` and `s`.
2. **Given** type `C2` and type `C3` both require the same predicate `p`, **When** type `C1` is declared `rdfs:subClassOf` both, **Then** `p` appears exactly once in `C1`'s effective required set (no duplicate reporting, no double-counting).
3. **Given** type `C1` is `rdfs:subClassOf` both `C2` and `C3`, **When** a node of type `C1` carries every predicate required by `C2` and by `C3`, **Then** the node passes the required-predicate check with no violations attributable to either base type.

---

### User Story 3 - Inheritance chains resolve correctly across multiple levels (Priority: P2)

A schema maintainer builds a hierarchy more than one level deep — a type is `rdfs:subClassOf` a base type that is itself `rdfs:subClassOf` another base type. The maintainer expects the contract to flow all the way down the chain: a type at the bottom of the hierarchy inherits every predicate declared anywhere above it, not just from its immediate parent.

**Why this priority**: A schema that only supported a single level of inheritance would still force redeclaration once a hierarchy grows past two tiers, undermining the simplification this feature promises for any but the shallowest schemas. This depends on User Stories 1 and 2 already inheriting correctly at one level.

**Independent Test**: Declare a three-level chain (`C1 rdfs:subClassOf C2`, `C2 rdfs:subClassOf C3`) where each level requires a distinct predicate, and confirm `C1`'s effective contract requires all three predicates.

**Acceptance Scenarios**:

1. **Given** `C1 rdfs:subClassOf C2` and `C2 rdfs:subClassOf C3`, where `C3` requires predicate `t`, **When** `C1`'s effective contract is computed, **Then** it requires `t` even though `C1` has no direct `rdfs:subClassOf` relationship to `C3`.
2. **Given** a hierarchy where two different branches both eventually lead to the same common ancestor type (diamond-shaped inheritance), **When** a type at the bottom of the hierarchy has its effective contract computed, **Then** the common ancestor's predicates appear exactly once, regardless of how many paths reach it.

---

### User Story 4 - Malformed base-type declarations are caught, not silently ignored (Priority: P2)

A schema maintainer makes a mistake while wiring up a type hierarchy — declaring `rdfs:subClassOf` a type name that doesn't exist in the schema, or accidentally creating a cycle where a type is, directly or transitively, its own ancestor. The maintainer wants this caught with a clear report rather than the tool silently ignoring the relationship or hanging while trying to resolve it.

**Why this priority**: Schema hierarchies are hand-authored and mistakes are expected; without this safeguard, a broken hierarchy would either produce silently incomplete conformance checks (a maintainer trusts a contract that isn't actually being enforced) or a tool that never terminates while walking a cycle. Lower priority than the core inheritance mechanism itself, but necessary before the feature can be trusted in a real schema.

**Independent Test**: Declare a type `rdfs:subClassOf` a base type name that has no schema document, and separately declare two types that are each `rdfs:subClassOf` the other, then confirm both situations are reported clearly rather than crashing, hanging, or passing silently.

**Acceptance Scenarios**:

1. **Given** a type declares `rdfs:subClassOf` a type name with no corresponding registered type in the schema, **When** the schema is loaded or checked, **Then** the tool reports a clear violation naming the type and the unresolved base-type reference, and does not treat the missing base type as contributing any predicates.
2. **Given** two types are declared `rdfs:subClassOf` each other (directly or through a longer chain back to themselves), **When** the schema is loaded or checked, **Then** the tool reports a clear violation naming the cycle, and does not hang or crash while computing either type's effective contract.

---

### Edge Cases

- What happens when a type declares no `rdfs:subClassOf` relationship at all? Its effective contract is exactly its own directly declared predicates, unchanged from today's behavior.
- What happens when a subtype directly declares a predicate that an ancestor already requires? The predicate is required either way — no conflict, no duplicate violation.
- What happens when an ancestor requires a predicate but a subtype's own declaration would otherwise have left it optional? The stricter outcome wins: once any ancestor requires a predicate, it is required for the subtype too.
- What happens when a type is declared `rdfs:subClassOf` itself directly? This is the smallest case of the cycle scenario in User Story 4 and is reported the same way.
- What happens when the same base type is named more than once in a single type's `rdfs:subClassOf` declarations? Treated as a single relationship; no duplicate contribution.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Schema maintainers MUST be able to declare that a type (`C1`) is `rdfs:subClassOf` another registered type (`C2`).
- **FR-002**: Schema maintainers MUST be able to declare more than one `rdfs:subClassOf` relationship on the same type, naming multiple distinct base types.
- **FR-003**: A type's effective set of required predicates MUST be the union of its own directly declared required predicates and every required predicate declared by its base type(s).
- **FR-004**: A type's effective set of optional predicates MUST be the union of its own directly declared optional predicates and every optional predicate declared by its base type(s), excluding any predicate that is required in the effective set.
- **FR-005**: Inheritance MUST resolve transitively: a type inherits from its base types' own base types, to any depth, not only from its immediate base type(s).
- **FR-006**: When two or more base types (directly or transitively) contribute the same predicate to a type's effective contract, that predicate MUST be represented exactly once, with no duplicate reporting.
- **FR-007**: When a predicate is required by at least one ancestor in a type's inheritance hierarchy, it MUST remain required in the type's effective contract even if another part of the hierarchy or the type itself would otherwise treat it as merely optional.
- **FR-008**: `arc lint`'s existing checks for a node missing a predicate its type requires, and for a node carrying a predicate its type does not permit, MUST evaluate against each type's effective (inherited) contract, not only the predicates declared directly on that node's own type.
- **FR-009**: Every schema management operation that reads a type's required/optional predicate contract MUST consult the effective (inherited) contract, consistently with the lint checks in FR-008.
- **FR-010**: System MUST detect a type whose `rdfs:subClassOf` declarations form a cycle (a type that is, directly or transitively, its own base type) and report it as a violation, without hanging or crashing while computing any type's effective contract.
- **FR-011**: System MUST detect a type's `rdfs:subClassOf` declaration naming a base type that has no corresponding registered type in the schema, and report it as a violation, treating the unresolved reference as contributing no predicates.
- **FR-012**: A type's declared merge behavior (its own whole-node reconciliation rule) is unaffected by `rdfs:subClassOf` — inheritance applies to required/optional predicate contracts only.

### Key Entities *(include if feature involves data)*

- **Type (Class)**: A registered schema entry naming a node type; carries its own directly declared required and optional predicates, plus zero or more `rdfs:subClassOf` relationships to other registered types.
- **`rdfs:subClassOf` relationship**: A directed relationship from one type to another, declaring that the first inherits the second's predicate contract; a type may hold more than one such relationship (multiple inheritance) and relationships may chain across several types.
- **Effective (inherited) contract**: The computed, de-duplicated union of required and optional predicates a type carries once every `rdfs:subClassOf` ancestor, at any depth, has contributed its own required and optional predicates.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A schema maintainer defining a family of related types that share a common contract writes that shared contract exactly once, regardless of how many subtypes reuse it.
- **SC-002**: 100% of nodes that satisfy a type's full inherited contract pass conformance checks, and 100% of nodes missing any inherited-required predicate are flagged — matching the accuracy already guaranteed for directly declared requirements.
- **SC-003**: Inheritance hierarchies at least four levels deep, and types with at least three declared base types, resolve to the correct effective contract with no missed or duplicated predicates.
- **SC-004**: Every malformed `rdfs:subClassOf` declaration (unresolved base type, or a cycle of any length) is reported clearly, with zero cases of the tool hanging, crashing, or silently passing a broken hierarchy as valid.

## Assumptions

- `rdfs:subClassOf` inheritance is purely additive: a subtype can broaden a base type's contract by adding its own predicates, but nothing in the hierarchy can narrow or remove a predicate a base type contributes.
- Inheritance applies to the required/optional predicate contract only; a type's whole-node merge behavior remains an independent, directly declared property of that type and is not inherited.
- Reporting of unresolved base types and inheritance cycles follows the same violation-reporting channel and format schema/lint validity issues already use elsewhere, so a maintainer sees these problems the same way they see other schema problems.
- Existing types that declare no `rdfs:subClassOf` relationship are completely unaffected — their effective contract remains exactly their own directly declared predicates.
