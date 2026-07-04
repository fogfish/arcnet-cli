# Feature Specification: Graph Schema as a First-Class Citizen (`_schema/`)

**Feature Branch**: `005-graph-schema-first-class`

**Created**: 2026-07-04

**Status**: Draft

**Input**: User description: "Make a schema as a first class citizen of the graph. Instead of `_meta` and `.arc/config` a new folder `_schema` is defined. The folder contains subfolders: (a) `nodes/` contains a document per node kind (e.g. entity.md) and `predicates/` contains a documents per predicate (e.g. related.md). Each of them has `id` equal to file base name (equal to name of this entity) and `kind: schema`. The nodes document also contains a `merge` attribute. It substitude `.arc/config` behaviour. The schema is created by `arc init` for core specification (see https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md). The schema is extended by `arc apply` when new node kind or predicate is discovered in the graph."

## Clarifications

### Session 2026-07-04

- Q: Are `_schema/nodes/*.md` and `_schema/predicates/*.md` documents subject to `arc lint`'s ordinary per-kind content validation (e.g. required fields for that kind, source-citation-back rule), or are they exempt as schema/tooling metadata? → A: Exempt from ordinary lint — validated only against schema-document well-formedness (id/kind: schema, and merge for node-kind documents), not against the content rules that apply to ordinary graph nodes.
- Q: Can a patch itself carry an explicit merge-behavior instruction for a newly discovered node kind, or does auto-discovery always assign the safe default, with customization only via a later direct edit of the schema document? → A: Always the safe default — a patch never carries a merge-behavior instruction; customizing a discovered kind's merge behavior is done only by directly editing its `_schema/nodes/<kind>.md` document afterward.
- Q: Do `_schema/nodes/` and `_schema/predicates/` documents participate in `arc lint`'s existing whole-graph basename-uniqueness check (CORE §3.2), or do they occupy a separate namespace from ordinary content nodes? → A: Separate namespace — basenames are unique only within each `_schema/` subfolder; a schema document's basename coinciding with an ordinary content node's id elsewhere in the graph is not a conflict.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bootstrap a graph with a first-class, versioned schema (Priority: P1)

A user initializes a new knowledge graph. Instead of the schema living in a hidden, tool-only configuration file and a pair of loosely-structured stub files, every node kind and predicate the core specification defines is represented as its own readable document, sitting alongside the rest of the graph's content and tracked in the same version history.

**Why this priority**: The schema (what kinds of nodes exist, what predicates connect them, and how each kind's content is combined) is foundational to every other graph operation. Establishing it as ordinary, versioned graph content — rather than tool-internal state — is the core premise of this feature and must exist before extension (User Story 2) or inspection (User Story 3) have anything to act on.

**Independent Test**: Can be fully tested by initializing a new graph and inspecting the resulting file tree — delivers a graph whose schema is fully readable and versioned with no other command required.

**Acceptance Scenarios**:

1. **Given** a directory with no existing graph, **When** the user initializes a new graph, **Then** the tool creates a `_schema/` folder containing `nodes/` and `predicates/` subfolders, populated with one document per node kind and one document per predicate defined by the core specification.
2. **Given** a freshly initialized graph, **When** the user inspects any document under `_schema/nodes/`, **Then** its identity matches its file's base name, it is marked as a schema document, and it states the merge behavior for that node kind.
3. **Given** a freshly initialized graph, **When** the user inspects any document under `_schema/predicates/`, **Then** its identity matches its file's base name and it is marked as a schema document.
4. **Given** a freshly initialized graph, **When** the user inspects the graph's file tree, **Then** no `_meta/` folder and no merge-rule configuration exist — the schema folder is the only place this information lives.
5. **Given** the canonical core specification cannot be retrieved at initialization time (e.g., no network access), **When** the user initializes a new graph, **Then** initialization still succeeds, seeding the schema folder from the tool's built-in defaults instead.

---

### User Story 2 - Schema grows automatically as new content is ingested (Priority: P2)

A user applies a patch that introduces a node kind or predicate not yet present in the graph's schema. Rather than being rejected or silently ignored, the new kind or predicate is recognized and added to the schema so future contributions of the same kind or predicate are consistently recognized too.

**Why this priority**: The graph format is designed to be extended with domain-specific kinds and predicates. Without this, every extension would require a separate, manual registration step before its first contribution could be ingested — this story keeps the schema self-maintaining as the graph itself grows.

**Independent Test**: Can be fully tested by applying a patch that introduces a previously-unseen node kind or predicate, then confirming a corresponding schema document exists in the graph immediately afterward.

**Acceptance Scenarios**:

1. **Given** a patch introduces a node kind with no existing schema document, **When** the patch is applied, **Then** the tool creates a new document for that kind under `_schema/nodes/`, using the safe default merge behavior, and the patch's content is still applied successfully.
2. **Given** a patch introduces a predicate with no existing schema document, **When** the patch is applied, **Then** the tool creates a new document for that predicate under `_schema/predicates/`.
3. **Given** a patch introduces a node kind or predicate that already has a schema document, **When** the patch is applied, **Then** the existing schema document is left unchanged — no duplicate or overwritten schema document is created.
4. **Given** a patch application discovers and registers new node kinds or predicates, **When** the application finishes, **Then** the new schema documents are recorded in the same commit as the rest of that patch's changes.

---

### User Story 3 - Understand and curate the graph's schema directly (Priority: P3)

A person maintaining a graph wants to know what node kinds and predicates exist and how each kind's content is merged, without running a special inspection command or reading tool-internal configuration. They open the schema folder like any other part of the graph, and — since it is ordinary versioned content — can propose a change to it (e.g., adjusting a kind's merge behavior) the same way they would propose any other graph change.

**Why this priority**: This is the qualitative payoff of making schema "first class" — a human-readable, diffable, reviewable schema — but it depends on Stories 1 and 2 already producing well-formed schema documents to look at and edit.

**Independent Test**: Can be fully tested by browsing `_schema/nodes/` and `_schema/predicates/` in a graph that has been initialized and had at least one patch applied, and confirming every node kind and predicate actually used in the graph is represented and its merge behavior is stated in plain text.

**Acceptance Scenarios**:

1. **Given** a graph with a mix of core and discovered node kinds, **When** the user lists `_schema/nodes/`, **Then** every node kind present anywhere in the graph has exactly one corresponding document there.
2. **Given** a graph with a mix of core and discovered predicates, **When** the user lists `_schema/predicates/`, **Then** every predicate used anywhere in the graph has exactly one corresponding document there.
3. **Given** a user edits a node kind's schema document to change its declared merge behavior, **When** a later patch contributes to that kind, **Then** the edited behavior is the one applied.

---

### Edge Cases

- What happens when a schema document is malformed or missing its required fields? The tool treats the kind or predicate as unrecognized and falls back to the same safe default behavior used for a never-before-seen kind, rather than failing the whole operation.
- What happens when initializing a graph that already has a `_schema/` folder (already initialized)? The tool refuses, consistent with existing already-initialized-graph protection — no schema content is lost or overwritten.
- What happens when a patch reuses a node-kind name and a predicate name that are identical? Node kinds and predicates are tracked separately (`_schema/nodes/` vs. `_schema/predicates/`), so the same name may appear as both without conflict.
- What happens when the same previously-unseen node kind or predicate appears more than once within a single patch application? Exactly one schema document is created for it, not one per occurrence.
- What happens when a graph's content is validated (e.g., by `arc lint`)? A `_schema/` document is checked only against schema-document well-formedness (identity, schema marker, and — for node-kind documents — a stated merge behavior); the ordinary content rules that apply to source/entity/resource/domain-kind nodes (e.g., the source-citation-back requirement) do not apply to it.
- What happens when a `_schema/` document's basename coincides with an ordinary content node's id elsewhere in the graph? This is not a conflict — schema documents occupy their own namespace, unique only within their own `_schema/nodes/` or `_schema/predicates/` subfolder, separate from the graph-wide basename-uniqueness check applied to ordinary content nodes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST represent a graph's schema — its node kinds and predicates — as a `_schema/` folder containing versioned documents, replacing the previous `_meta/` folder and the merge-rule portion of `.arc/config` as the source of this information.
- **FR-002**: The tool MUST organize `_schema/` into two subfolders: `nodes/`, holding one document per node kind, and `predicates/`, holding one document per predicate.
- **FR-003**: Every schema document's identity MUST equal its own file's base name, and that base name MUST equal the name of the node kind or predicate it describes. This identity is unique only within its own `_schema/nodes/` or `_schema/predicates/` subfolder — schema documents occupy a separate namespace from ordinary content nodes and do not participate in the graph-wide basename-uniqueness check applied to those nodes.
- **FR-004**: Every schema document MUST declare itself as a schema-kind document, distinguishing it from ordinary graph content nodes.
- **FR-005**: Every node-kind schema document MUST additionally declare that kind's merge behavior.
- **FR-006**: The tool MUST create and populate `_schema/` when initializing a new graph, seeding it with a document for every node kind and predicate defined by the core specification.
- **FR-007**: When the core specification cannot be retrieved at initialization time, the tool MUST still succeed, seeding `_schema/` from built-in defaults instead, consistent with initialization's existing offline-first guarantee.
- **FR-008**: The tool MUST NOT create a `_meta/` folder or a merge-rule section in `.arc/config` when initializing a new graph.
- **FR-009**: When applying a patch, the tool MUST inspect every node kind and predicate the patch contributes and, for any one with no existing schema document, MUST create a new schema document for it in the appropriate subfolder.
- **FR-010**: A node-kind schema document created this way MUST always be assigned the safe default merge behavior — a patch MUST NOT be able to specify a different merge behavior for a kind it introduces; customizing a discovered kind's merge behavior is only ever done afterward, by directly editing its schema document (FR-014), consistent with the tool's existing unrecognized-kind handling.
- **FR-011**: The tool MUST NOT create a duplicate or overwrite an already-existing schema document for a node kind or predicate that is already registered.
- **FR-012**: Schema documents created or extended while applying a patch MUST be included in that same patch application's single commit, not committed separately.
- **FR-013**: Every capability that previously relied on `_meta/` or `.arc/config` to recognize node kinds, resolve merge behavior, or validate predicates MUST instead read that information from `_schema/`, with no loss of existing recognition, merge, or validation behavior.
- **FR-014**: A node kind's merge behavior used by patch application MUST be read from that kind's `_schema/nodes/` document, so that editing the document changes future behavior for that kind.
- **FR-015**: Content-validation checks that apply to ordinary graph nodes (e.g., a per-kind required-field rule or a source-citation-back requirement) MUST NOT apply to `_schema/` documents themselves; a schema document is validated only against schema-document well-formedness (identity, schema marker, and — for node-kind documents — a stated merge behavior).

### Key Entities

- **Schema Folder (`_schema/`)**: The versioned location holding the complete description of a graph's node kinds and predicates, replacing `_meta/` and `.arc/config`'s merge-rule content.
- **Node-Kind Schema Document**: One document per node kind (e.g., `_schema/nodes/entity.md`), identified by its base name, marked as a schema document, and stating that kind's merge behavior.
- **Predicate Schema Document**: One document per predicate (e.g., `_schema/predicates/related.md`), identified by its base name and marked as a schema document.
- **Discovered Kind / Predicate**: A node kind or predicate encountered while applying a patch that has no existing schema document yet, and is registered automatically as a result.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of node kinds and predicates defined by the core specification are represented as individual schema documents immediately after a graph is initialized.
- **SC-002**: A person can determine any node kind's merge behavior by reading one plain-text document, with no need to consult a separate configuration file or run a special command.
- **SC-003**: 100% of previously-unseen node kinds or predicates encountered while applying a patch have a corresponding schema document in the graph immediately after that application completes.
- **SC-004**: Existing kind-recognition, merge, and predicate-validation behavior operates with zero regressions after the schema relocation — every case that worked against `_meta/`/`.arc/config` continues to work against `_schema/`.
- **SC-005**: A newly initialized graph is fully self-describing from its own committed content alone — no external or tool-internal state is required to know what node kinds, predicates, or merge behaviors it supports.

## Assumptions

- This feature governs newly initialized graphs going forward; migrating an already-existing graph's `_meta/`/`.arc/config` content into `_schema/` is out of scope, since the project has no production graphs predating this change.
- The `_meta/aliases.md` stub, which currently has no reader anywhere in the tool, has no replacement in `_schema/`; its alias-registry intent is dropped rather than carried forward, since nothing in the tool ever consumed it.
- Beyond `id`, the schema-document marker, and (for node kinds) `merge`, no additional required fields are introduced by this feature; a predicate document needs no attributes beyond its identity and schema marker.
- The safe default merge behavior applied to a newly discovered node kind is the same "union" default already established for unregistered kinds.
