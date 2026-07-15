# Feature Specification: Import Schema Definitions via `arc apply schema`

**Feature Branch**: `018-apply-schema-patch`

**Created**: 2026-07-15

**Status**: Draft

**Input**: User description: "`arc apply schema <patch.md> | <url>` sub-command applies a patch document into the schema. Essentially, the patch document has same format as the graph based patch document but carries only definition for the schema. The command is defined to import schemas specification for arcnet extensions. It only accepts `Property` and `Class` types from the patch document. If the patch contains other types, entire process fails."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Import a published extension's schema in one step (Priority: P1)

A schema maintainer wants to adopt an arcnet extension published by a third party (or by themselves in another graph) — a bundle of predicate (`Property`) and type (`Class`) definitions that describe the extension's vocabulary. Instead of hand-copying each definition into the local graph's schema directory, the maintainer points `arc apply schema` at the extension's patch document — a local file, a full URL, or a short `arcnet:`-prefixed reference to the official arcnet extensions catalog — and every `Property`/`Class` definition it carries is created in the local schema in one step.

**Why this priority**: This is the feature's entire reason to exist — a schema maintainer who cannot bulk-import a published vocabulary is forced to reconstruct it by hand, one predicate or type at a time, which is exactly the friction this command removes.

**Independent Test**: Can be fully tested by pointing the command at a patch document containing only `Property` and `Class` node sections and confirming the local schema gains a matching predicate/type definition for each one.

**Acceptance Scenarios**:

1. **Given** a well-formed patch document containing only `Property` node sections, **When** the maintainer runs `arc apply schema` against that document, **Then** a schema predicate definition is created for each `Property` node it carries.
2. **Given** a well-formed patch document containing only `Class` node sections, **When** the maintainer runs `arc apply schema` against that document, **Then** a schema type definition is created for each `Class` node it carries.
3. **Given** a patch document containing a mix of `Property` and `Class` node sections, **When** the command is applied, **Then** every `Property` becomes a predicate definition and every `Class` becomes a type definition in the same run.
4. **Given** the command finishes applying a patch, **When** the maintainer reviews the output, **Then** it reports how many predicate and type definitions were created (and how many were merged into existing ones, if any).
5. **Given** the maintainer wants a published arcnet extension without knowing or typing its full download URL, **When** they run `arc apply schema` with an `arcnet:`-prefixed reference (e.g. `arcnet:media.schema.md`), **Then** the command resolves it to the corresponding location in the official arcnet extensions catalog and imports it exactly as if the maintainer had supplied that full URL directly.

---

### User Story 2 - Reject a patch that isn't schema-only (Priority: P1)

A schema maintainer accidentally points `arc apply schema` at an ordinary content patch (one carrying `source`, `entity`, or `resource` nodes, or a mix of schema and content nodes) instead of a schema-only patch. The command must refuse to apply any part of it and explain why, rather than silently importing the schema-shaped nodes and dropping the rest.

**Why this priority**: This is the safety guarantee the command is built around — without it, a misapplied patch could partially and silently corrupt the schema/content boundary, and the maintainer would have no clear signal that something went wrong.

**Independent Test**: Can be fully tested by applying a patch containing at least one non-`Property`/`Class` node section and confirming the command fails, names the offending node, and leaves the schema completely unchanged.

**Acceptance Scenarios**:

1. **Given** a patch document containing a `source`, `entity`, or `resource` node section, **When** the maintainer runs `arc apply schema` against it, **Then** the command fails and identifies the node's id and type as the reason.
2. **Given** a patch document containing both valid `Property`/`Class` sections and one disallowed node section, **When** the command is applied, **Then** none of the patch's definitions are written to the schema — not even the otherwise-valid ones.
3. **Given** a patch document containing a node kind reserved for graph structure (e.g. `timeline`), **When** the command is applied, **Then** it is treated the same as any other disallowed kind and the whole patch is rejected.

---

### User Story 3 - Refresh an already-imported extension's schema (Priority: P2)

An extension the maintainer previously imported publishes an update — a revised description, an added optional predicate on an existing type, or a new predicate altogether. The maintainer re-applies the extension's updated patch document, and the local schema picks up the changes for definitions it already knows about, without needing to be told which specific definitions changed.

**Why this priority**: Extensions evolve after their first import; without a working re-apply path, every update would require the maintainer to manually diff and hand-edit schema files instead of trusting the same command they used the first time.

**Independent Test**: Can be fully tested by importing a patch, changing one field in one of its `Property`/`Class` sections, re-applying it, and confirming only that field changed in the local schema while everything else was left intact.

**Acceptance Scenarios**:

1. **Given** a predicate definition already present in the schema from a prior import, **When** a patch re-declaring that predicate (with an added or changed field) is applied, **Then** the existing definition is updated according to its declared merge behavior rather than duplicated.
2. **Given** a type definition already present in the schema, **When** a patch re-declaring that type with no actual changes is applied, **Then** the command completes without reporting any created or merged changes for it.

---

### Edge Cases

- What happens when the given path does not resolve to an existing local file and is not a fetchable URL?
- What happens when a URL is unreachable, times out, or returns a non-success response?
- What happens when an `arcnet:` reference has nothing after the prefix?
- What happens when an `arcnet:`-resolved location is unreachable, times out, or returns a non-success response (same as any other URL)?
- What happens when the patch document is not well-formed (fails to parse as a patch document at all)?
- What happens when the patch document is well-formed but contains zero node sections?
- What happens when a `Property` or `Class` node section is itself malformed (e.g. missing its mandatory description, or naming an invalid role/merge value)?
- What happens when the command is run outside an initialized graph (no schema to import into)?
- What happens when a `Class` node's re-import would remove a predicate the local schema's own type extended it with?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The command MUST accept a single input that is a path to a local patch document, a URL referencing one, or an `arcnet:`-prefixed reference to one.
- **FR-002**: When the input is a URL, the command MUST fetch the patch document's contents before parsing it.
- **FR-002a**: When the input begins with the literal prefix `arcnet:`, the command MUST treat everything after that prefix as a path within the official arcnet extensions catalog, resolve it to that catalog's fixed base location, and fetch the patch document from the resolved location exactly as it would for a directly supplied URL.
- **FR-003**: The command MUST parse the input using the same patch document format used by the graph's own patch-apply command.
- **FR-004**: The command MUST inspect every node section in the patch document and classify it as `Property`, `Class`, or disallowed.
- **FR-005**: If the patch document contains any node section whose type is not `Property` or `Class`, the command MUST fail the entire operation and MUST NOT write any of the patch's definitions to the schema, including otherwise-valid `Property`/`Class` sections in the same document.
- **FR-006**: When a failure occurs per FR-005, the command MUST report the id and type of at least one disallowed node so the maintainer can identify the offending section.
- **FR-007**: For each valid `Property` node section, the command MUST create a new predicate definition in the schema if none exists for that name, or merge into the existing one if it does, following that definition's declared merge behavior.
- **FR-008**: For each valid `Class` node section, the command MUST create a new type definition in the schema if none exists for that name, or merge into the existing one if it does, following that definition's declared merge behavior.
- **FR-009**: The command MUST report a summary of how many predicate and type definitions were created versus merged by the run.
- **FR-010**: The command MUST require an initialized graph to run against and MUST fail with a clear message if none is present.
- **FR-011**: Re-applying an identical, previously-imported patch document MUST leave the schema unchanged and MUST be reported as having made no changes.
- **FR-012**: The command MUST leave the schema fully unchanged if any step of applying the patch fails partway through.

### Key Entities

- **Schema Patch Document**: A document in the standard patch format restricted, for this command, to carrying only `Property` and `Class` node sections — the unit of import this command accepts.
- **Predicate Definition**: A named schema entry (role, merge behavior, description, and related attributes) describing one relation a node may carry; created or updated from a patch's `Property` sections.
- **Type Definition**: A named schema entry (required/optional predicates, base types, merge behavior, description) describing one node kind's contract; created or updated from a patch's `Class` sections.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A schema maintainer can import a published extension's full set of predicate and type definitions with a single command invocation, regardless of how many definitions the extension declares.
- **SC-002**: 100% of patch documents containing any non-`Property`/`Class` node section are rejected with zero definitions written to the schema.
- **SC-003**: Re-applying an unchanged, previously-imported schema patch reports zero created or merged definitions.
- **SC-004**: After a failed import, the schema directory's contents are byte-for-byte identical to their state before the command ran.
- **SC-005**: A maintainer can import any officially cataloged arcnet extension using only its short name, with identical results to supplying that extension's full download URL by hand.

## Assumptions

- The patch document format for schema-only patches is the same manifest/node-section format used by the existing graph patch-apply command, simply restricted in which node types are present.
- A URL input is fetched as a plain, unauthenticated HTTP(S) request; no credential handling is in scope for this feature.
- The `arcnet:` prefix resolves to a fixed, built-in base location — `https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/` — with the remainder of the input appended verbatim as the path suffix (e.g. `arcnet:media.schema.md` → `https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/media.schema.md`). This base is not user-configurable in this feature's scope.
- "Entire process fails" means the operation is all-or-nothing: if any disallowed node section is present anywhere in the patch, none of the patch's `Property`/`Class` definitions are written, even the valid ones.
- Merge behavior for re-imported `Property`/`Class` definitions follows each definition's own declared merge policy, consistent with how the graph's patch-apply command merges other node kinds.
- This command targets the local schema only; it does not create or modify any graph content nodes (sources, entities, resources).
