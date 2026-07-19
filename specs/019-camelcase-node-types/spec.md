# Feature Specification: CamelCase Node Class Names

**Feature Branch**: `019-camelcase-node-types`

**Created**: 2026-07-19

**Status**: Draft

**Input**: User description: "The app uses mixture of upper and lower cases for node classes. In the patch parser the type names are converted into the lower case. The lower-case names for type contradicts with RDFS and other type systems. The app MUST always treat types as CamelCase. The build-in schema, `arc init`, `arc lint` MUST define and treat class names as Came Case. The `arc apply` MUST reject any document if the H1 does not follow CamelCase and H1 name starts with lower case."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - `arc apply` rejects non-CamelCase class headings (Priority: P1)

A user authors a Markdown patch document to add or update nodes in the graph. Each node's class is declared by an H1 heading in the document (e.g. `# Entity`). Today, `arc apply` silently lowercases that heading text to derive the class name, so `# Entity` and `# entity` are treated as the same class and both are stored as `entity`. This contradicts RDFS-style type-naming conventions, where class names are CamelCase (start with an uppercase letter). Going forward, `arc apply` must preserve the heading's casing exactly and must refuse to apply a document whose class heading starts with a lowercase letter, instead of quietly normalizing it.

**Why this priority**: This is the primary enforcement point named by the requester and is the only place new casing violations can currently enter the graph. Without it, every other change in this feature is cosmetic.

**Independent Test**: Run `arc apply` against a fixture document whose H1 is `# entity`; verify the command exits with a non-zero/error status, reports the CamelCase requirement, and writes no changes to the graph. Run `arc apply` against a fixture document whose H1 is `# Entity`; verify the command succeeds and the resulting node's class is stored as `Entity` (not lowercased).

**Acceptance Scenarios**:

1. **Given** a patch document whose H1 heading begins with a lowercase letter (e.g. `# entity`), **When** the user runs `arc apply` on that document, **Then** the command exits with a non-zero status, the error message names the offending heading and states that class names must be CamelCase (start with an uppercase letter), and the graph is left unmodified.
2. **Given** a patch document whose H1 heading begins with an uppercase letter (e.g. `# Entity`), **When** the user runs `arc apply`, **Then** the command succeeds and the node's class is recorded using the heading's exact casing, with no automatic lowercasing.
3. **Given** a patch document containing multiple H1 sections, **When** the user runs `arc apply` and at least one H1 begins with a lowercase letter, **Then** the entire document is rejected (no partial apply) and the error identifies every offending heading.

---

### User Story 2 - Built-in schema and `arc init` use CamelCase class names (Priority: P2)

A user initializes a new graph repository with `arc init`. The tool seeds a default/built-in schema that currently mixes casing: some classes are CamelCase (`Node`, `Property`, `Class`) and others are lowercase (`source`, `entity`, `resource`, `timeline`). Going forward, every built-in class name must be CamelCase, so a freshly initialized repository is consistent with the CamelCase convention from the start and its own seeded schema is immediately compatible with the `arc apply` rule from User Story 1.

**Why this priority**: This is a prerequisite for User Story 1 to be usable end-to-end — if the seeded built-in classes are still lowercase, a user cannot reference them from a compliant, CamelCase H1 heading without first renaming them by hand.

**Independent Test**: Run `arc init` in an empty directory and inspect the seeded schema; verify every built-in class name begins with an uppercase letter (e.g. `Entity`, `Source`, `Resource`, `Timeline`, `Node`, `Property`, `Class`) and that no lowercase-first-letter duplicate remains.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the user runs `arc init`, **Then** every class name written to the seeded schema begins with an uppercase letter.
2. **Given** the schema seeded by `arc init`, **When** the user inspects the class definitions, **Then** no two classes differ only by casing (e.g. no simultaneous `Entity` and `entity`).

---

### User Story 3 - `arc lint` flags non-CamelCase class names (Priority: P3)

A user runs `arc lint` to validate an existing graph repository. Class names can enter a schema or a node's type reference other ways than `arc apply` (e.g. hand-edited schema files, older content). `arc lint` must check every class name defined in the schema, and every class name referenced by a node, and report a violation for any that does not start with an uppercase letter — so casing problems are caught by validation even when they didn't originate from a patch document.

**Why this priority**: Closes the loop for schemas or graphs that predate this change, or where content is edited outside of `arc apply`. Lower priority than Stories 1-2 because it is a detection mechanism, not a point of entry.

**Independent Test**: Run `arc lint` against a fixture schema/graph containing a class named `entity` (or a node whose `@type` is lowercase); verify the lint report includes a violation naming that class/node and stating the CamelCase requirement. Run `arc lint` against a fixture where every class name is CamelCase; verify no such violation is reported.

**Acceptance Scenarios**:

1. **Given** a schema containing a class definition whose name begins with a lowercase letter, **When** the user runs `arc lint`, **Then** the report includes a violation for that class stating it must be CamelCase.
2. **Given** a graph where a node's class/type reference begins with a lowercase letter, **When** the user runs `arc lint`, **Then** the report includes a violation for that node.
3. **Given** a schema and graph where every class name begins with an uppercase letter, **When** the user runs `arc lint`, **Then** no CamelCase-related violation is reported.

---

### Edge Cases

- H1 heading whose first character is not a letter at all (e.g. a leading digit, underscore, or punctuation) is treated as not starting with an uppercase letter, and is rejected by `arc apply` under the same rule as a lowercase heading.
- A single-character H1 heading that is an uppercase letter (e.g. `# X`) is valid.
- H1 headings containing Unicode letters use a Unicode-aware uppercase check (e.g. an accented capital letter is valid; its lowercase counterpart is not).
- A patch document whose H1 is blank or contains no letters at all is rejected for being malformed, independent of this feature's casing rule.
- Predicate/relationship (edge) type names are not node classes and are out of scope for this feature; this feature only governs node class/type names.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST treat CamelCase (name begins with an uppercase letter) as the canonical convention for node class/type names, applied consistently across the built-in schema, `arc init`, `arc lint`, and `arc apply`.
- **FR-002**: The built-in/default schema MUST define every class name beginning with an uppercase letter; no built-in class name may begin with a lowercase letter, and no two built-in classes may differ only by casing.
- **FR-003**: `arc init` MUST seed a new repository's schema using the CamelCase built-in class names; it MUST NOT seed any lowercase-first-letter variant of a built-in class.
- **FR-004**: `arc apply` MUST derive a node's class/type from its H1 heading text using the heading's exact casing; it MUST NOT lowercase or otherwise alter the casing of the heading text when deriving the class name.
- **FR-005**: `arc apply` MUST reject a patch document if any H1 heading that defines a class does not begin with an uppercase letter. Rejection MUST produce a non-zero/error exit, MUST leave the graph unmodified, and MUST report an error that names the offending heading(s) and states the CamelCase requirement.
- **FR-006**: `arc lint` MUST validate that every class name defined in the schema begins with an uppercase letter, and MUST report a violation, identifying the class, for each one that does not.
- **FR-007**: `arc lint` MUST validate that every node's referenced class/type name begins with an uppercase letter, and MUST report a violation, identifying the node, for each one that does not.
- **FR-008**: When a node declares an explicit `@type` value in addition to its H1 heading, `arc apply` MUST apply the same CamelCase-start requirement to the explicit `@type` value as it does to the H1 heading, rejecting the document if the explicit `@type` begins with a lowercase letter, independent of what the H1 heading itself contains.
- **FR-009**: `arc lint`, when run against a repository that already contains schema entries or nodes with lowercase-first-letter class names created before this rule took effect, MUST report those as ordinary CamelCase violations (the same violation reported for any other non-CamelCase class name) rather than silently ignoring or special-casing pre-existing content; no automatic rename/migration of existing content is performed as part of this feature.

### Key Entities

- **Class (Node Type)**: A named category of node in the graph (e.g. `Entity`, `Source`), identified by a CamelCase name. Defined either in the built-in schema or by users, and referenced by nodes via their `@type`.
- **Patch Document**: A Markdown file with H1 sections (each declaring a class) and H2 subsections (each declaring an individual node of that class), used as input to `arc apply`.
- **Schema**: The collection of class and predicate definitions governing a graph, seeded by `arc init` and validated by `arc lint`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of built-in class names shipped with the tool begin with an uppercase letter, with zero lowercase-first-letter duplicates, after this change.
- **SC-002**: `arc apply` rejects 100% of patch documents whose class-defining H1 heading (or explicit `@type`) starts with a lowercase letter, with zero instances of silent lowercasing.
- **SC-003**: `arc lint` reports a violation for 100% of class/type names in a repository's schema and graph that do not start with an uppercase letter, with zero false negatives across the test fixture set.
- **SC-004**: A user can determine why a document was rejected from the `arc apply` error message alone — without consulting external documentation — because the message names the offending heading and states the CamelCase rule.

## Assumptions

- "CamelCase" for this feature means the class/type name's first character MUST be an uppercase letter (Unicode-aware), consistent with RDFS/OWL and common ontology convention (e.g. `Entity`, `SubClassOf` target names). It does not additionally constrain internal characters (digits, further case changes) beyond that first-letter rule unless a future feature specifies stricter identifier rules.
- Predicate/relationship (edge) type names are out of scope; this feature governs node class/type names only.
- No automatic migration or bulk-rename tool for pre-existing lowercase class names is in scope; `arc lint` surfaces them as violations for the user to address manually (FR-009).
- An explicit `@type` field, where present, is held to the same CamelCase-start requirement as the H1 heading (FR-008), since both name the same class.
