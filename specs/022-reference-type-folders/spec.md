# Feature Specification: ARCNET-CORE v0.11 — `Reference` Type and Type-Named Folders

**Feature Branch**: `022-reference-type-folders`

**Created**: 2026-08-23

**Status**: Draft

**Input**: User description: "ARCNET-CORE v0.11 corrects a role mix-up between two node types and folders notation. (i) It has defined a \"functional\" folders that contains a heterogenous classes. It is explicitly defined with the role and class (data folder) that contains homogenous classes - each folder holds exactly one type. A type's folder name MUST be character-for-character identical to the type name, same case, no pluralization, no abbreviation, no synonym. Two functional folders are exempt, because neither is a type folder: `_schema/` — a namespace prefix, not a type; its children are type folders and MUST follow the rule: `Property/` for predicate nodes (§9.1), `Class/` for type nodes (§9.2). `timeline/` — an index whose `Timeline` nodes (§11.5) are bucketed by granularity into `yearly/` and `monthly/` subfolders rather than filed flat. (ii) New type introduced. `Resource` had been defined as an external work the graph points to but has not ingested; that is now the role of a new fifth core type, `Reference`. `Resource` is redefined as its original implementation meaning: an anonymous, tag-classified fragment of *ingested* content that does not yet warrant its own dedicated type. The tool still carries the pre-0.10 definition and has never heard of `Reference`. First, `Resource` must be redefined to require `text`, `tags` and `mentionedIn`, losing its previous `ref`/`relevance` requirement and its `url`/`authors`/`year`/`doi`/`status`/`isCitedBy` options. Second, a `Reference` type must be added, owning exactly those displaced predicates, filed in its own folder, and registered in the seeded schema like any other core type. Third, a `Resource` node's leading prose is currently keyed as `relevance`; under v0.10 it is `text`, and a `Reference` node's leading prose is `relevance` — so the body-prose predicate derivation must change for both types. The folder derivation must also be corrected: the generic fallback appends an `s` to the type name. Because the `required`/`optional` predicates merge by union and union cannot retract, re-seeding cannot perform this migration; an explicit migration path is required that replaces the affected schema nodes outright and re-files and re-types existing `Resource` nodes that carry the old external-work shape. Validate against https://github.com/fogfish/arcnet-spec/blob/main/ARCNET-CORE.md"

## Clarifications

### Session 2026-08-23

- **Q: Which folder should `Reference` nodes be filed in?** → **A: `Reference/`.** The feature description names a lowercase `references/` in part (ii), but part (i) of the same description mandates that a type folder's name be character-for-character identical to the type name, and upstream §6 explicitly lists `references/` → `Reference/` among the v0.11 renames. Part (ii)'s wording is pre-v0.11 text carried forward. The naming rule is universal: `sources/`, `entities/`, `resources/`, `_schema/predicates/`, and `_schema/types/` are renamed on the same grounds.
- **Q: Which predicate lists govern `Reference`? Upstream §11.6's normative `Class` block and its own 0.9→0.10 revision note disagree.** → **A: The revision note's lists.** `Reference` requires `title`, `ref`, `relevance` and offers `url`, `authors`, `year`, `doi`, `status`, `isCitedBy`, `notes`. This matches the description's "owning exactly those displaced predicates" and is the only reading under which §11.6's own worked example — which carries `ref`, `status`, and leading prose — conforms to the type it illustrates.
- **Q: How should existing graphs be migrated?** → **A: They are not.** This is an accepted breaking change. No migration path is built, and graphs created before this feature are out of scope. See Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A new graph carries the corrected core type vocabulary (Priority: P1)

An author initializes a fresh graph. The seeded schema declares five core content types — `Source`, `Entity`, `Resource`, `Timeline`, and `Reference` — with `Resource` meaning *an anonymous, tag-classified fragment of ingested content* and `Reference` meaning *an external work the graph points to but has not ingested*. Neither type claims the other's predicates.

Today the tool seeds only four content types. Its `Resource` still carries the retired external-work definition (`ref` + `relevance` required; `url`, `authors`, `year`, `doi`, `status`, `isCitedBy` optional), and `Reference` does not exist at all. Every graph the tool creates is therefore non-conformant from the first commit, and an author who wants to record an un-ingested external work has no correct type to record it as.

**Why this priority**: This is the substance of the role correction. Until the vocabulary is right, no other part of the feature has a correct target to converge on — both the prose derivation and type-conformance checking read from this vocabulary.

**Independent Test**: Initialize a fresh graph, read back the seeded schema, and confirm the two type declarations match ARCNET-CORE §11.4 and §11.6 — `Resource` requiring `text`/`tags`/`mentionedIn` and offering `notes`, `Reference` owning the displaced external-work predicates — without touching folder layout.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the author initializes a graph, **Then** the seeded schema contains a `Resource` type node whose required predicates are exactly `text`, `tags`, `mentionedIn`.
2. **Given** the same freshly initialized graph, **When** the author reads the seeded schema, **Then** `Resource` no longer offers `ref`, `url`, `authors`, `year`, `doi`, `status`, or `isCitedBy` as required or optional predicates.
3. **Given** the same freshly initialized graph, **When** the author reads the seeded schema, **Then** a `Reference` type node exists, requires `title`/`ref`/`relevance`, offers `url`/`authors`/`year`/`doi`/`status`/`isCitedBy`/`notes`, and declares `Node` as its base type exactly as the other four content types do.
4. **Given** a patch that carries a node typed `Reference`, **When** the author applies it to a freshly initialized graph, **Then** the node is accepted, written, and reported without any "unknown type" diagnostic.
5. **Given** a freshly initialized graph, **When** the author lints it, **Then** the graph reports clean — the seeded schema is self-consistent and every seeded type's predicates are themselves registered.
6. **Given** a `Reference` node carrying `title`, `ref`, and leading prose, **When** the author lints the graph, **Then** no diagnostic reports a missing required predicate or an undeclared one.

---

### User Story 2 - Ingested prose round-trips under the correct predicate (Priority: P2)

An author applies a patch containing a `Resource` node. The node's leading prose is stored under `text`, matching the type's own required predicate. An author applying a `Reference` node gets its leading prose stored under `relevance`. Reverting either node reconstructs the same prose from the same key.

Today a `Resource` node's leading prose is keyed `relevance` — a predicate the corrected `Resource` no longer declares at all. Every ingested `Resource` would silently store its body under a predicate its own type forbids: the file still looks right to a human reader, but type-conformance checking flags a required `text` that is missing and an undeclared `relevance` that is present, and any consumer reading `Resource.text` finds nothing. Because the write is silent and the file renders normally, the damage accumulates unnoticed across every ingest.

**Why this priority**: This is a silent data-correctness defect rather than a missing capability — it corrupts content quietly on every ingest, so it ranks above the visible filing change. It depends on Story 1 only for the definition of what "correct" is.

**Independent Test**: Apply a patch containing one `Resource` node and one `Reference` node, inspect the two written files, and confirm the leading prose is keyed `text` and `relevance` respectively; then revert the ingest and confirm both bodies are reconstructed from those same keys.

**Acceptance Scenarios**:

1. **Given** a patch carrying a `Resource` node with leading body prose, **When** the author applies it, **Then** the written node stores that prose under `text` and stores nothing under `relevance`.
2. **Given** a patch carrying a `Reference` node with leading body prose, **When** the author applies it, **Then** the written node stores that prose under `relevance`.
3. **Given** an applied `Resource` node and an applied `Reference` node, **When** the author reverts the ingesting source, **Then** each node's prose is recognized under the same key it was written with, and the revert leaves no orphaned prose behind.
4. **Given** an applied `Resource` node, **When** the author lints the graph, **Then** no diagnostic reports a missing required `text` or an undeclared `relevance` on that node.
5. **Given** a patch carrying a node of a type with no type-specific prose predicate, **When** the author applies it, **Then** its leading prose is still stored under `text` and its trailing prose under `notes`, unchanged by this feature.
6. **Given** a patch carrying `Source` and `Entity` nodes, **When** the author applies it, **Then** their leading prose is still stored under `abstract` and `definition` respectively, unchanged by this feature.

---

### User Story 3 - Nodes are filed in folders named for their type (Priority: P3)

An author looks at a graph on disk and sees `Source/`, `Entity/`, `Resource/`, `Reference/`, `timeline/`, and `_schema/`. Every type folder's name is the type name, character for character. Reading a folder name gives the `@type` value by string equality — no pluralization table, no case folding, no synonym lookup.

Today the tool files nodes under lowercase plural folders (`sources/`, `entities/`, `resources/`) and holds schema nodes under `_schema/predicates/` and `_schema/types/`, none of which is its type's name. For any type it does not have a hardcoded entry for, it appends an `s` to the type name — which would file `Reference` nodes under `References/` and a domain profile's `Thought` nodes under `Thoughts/`, both wrong under the new rule.

**Why this priority**: The filing change is highly visible and touches every command that reads or writes a node path, but it breaks no content — links resolve by basename, so no edge is affected by where a node sits. It is correctness of layout, delivered after correctness of content.

**Independent Test**: Initialize a fresh graph and confirm the created folders are exactly the type-named set; apply a patch carrying `Source`, `Entity`, `Resource`, `Reference`, and an unrecognized domain type, and confirm each node lands in a folder whose name equals its `@type` verbatim.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the author initializes a graph, **Then** the created folders are `Source/`, `Entity/`, `Resource/`, `Reference/`, `timeline/yearly/`, `timeline/monthly/`, `_schema/Property/`, and `_schema/Class/`, and no lowercase plural content folder is created.
2. **Given** a freshly initialized graph, **When** the author applies a patch carrying one node of each core content type, **Then** each node is written to `<TypeName>/<id>.md`.
3. **Given** a freshly initialized graph, **When** the author applies a patch carrying a node whose type is a domain-profile type unknown to the tool, **Then** the node is written to a folder named exactly that type, with no pluralizing suffix and no case change.
4. **Given** a graph containing `Timeline` nodes, **When** the author applies a patch that updates the timeline, **Then** those nodes remain bucketed under `timeline/yearly/` and `timeline/monthly/` and are never filed in a flat `Timeline/` folder.
5. **Given** a schema patch declaring a new predicate and a new type, **When** the author applies it, **Then** the predicate node is written under `_schema/Property/` and the type node under `_schema/Class/`.
6. **Given** an applied patch, **When** the author reverts it, **Then** every node it created is located and removed from its type-named folder, and every backlink referrer is rewritten in place.
7. **Given** a graph filed under type-named folders, **When** the author runs a subgraph export, a content search, or a lint, **Then** all three traverse the new layout and continue to exclude `.arc/` and `_schema/` from content traversal exactly as before.
8. **Given** a graph filed under type-named folders, **When** the author exports a subgraph and applies it to a second freshly initialized graph, **Then** every node lands at the same path it occupied in the first — the round trip is path-stable.

---

### Edge Cases

- **A domain type whose name is already lowercase or already plural.** A type literally named `thoughts` files under `thoughts/`, and a type named `Series` files under `Series/` — the folder is the type name verbatim, with no attempt to normalize either direction.
- **A `Timeline` node arriving as an ordinary node.** A patch carries `@type: Timeline` outside the timeline-index path. Existing handling is unchanged: it must never be filed flat in a `Timeline/` folder.
- **A `Resource` node with no body prose.** Nothing is stored under `text`; the type's required-predicate check reports the omission the same way it reports any other missing required predicate. The prose derivation introduces no empty-value write.
- **A `Reference` node whose type is not registered in the target graph.** A patch carries a `Reference` node against a graph whose schema predates this feature. The existing unknown-type behaviour applies unchanged; this feature adds no new special case.
- **A node re-filed by an author by hand.** An author moves a node out of its type folder. `@type` is read from the node, never from the folder, so every command continues to treat it correctly — the folder is a mirror, not a source of truth.
- **A graph created before this feature.** Out of scope by the breaking-change decision. Behaviour is undefined and no detection, refusal, or repair is built. See Assumptions.

## Requirements *(mandatory)*

### Functional Requirements

**Type vocabulary**

- **FR-001**: The seeded schema MUST declare `Resource` with required predicates `text`, `tags`, `mentionedIn` and optional predicate `notes`, and MUST NOT declare `ref`, `relevance`, `url`, `authors`, `year`, `doi`, `status`, or `isCitedBy` on `Resource` in either list.
- **FR-002**: `Resource`'s description MUST state its corrected meaning: a fragment of an ingested document's content, relevant to the graph but not warranting its own dedicated type, tag-classified so a recurring pattern can later be promoted into a proper domain type.
- **FR-003**: The seeded schema MUST declare a fifth core content type `Reference`, described as an external work the graph points to but has not ingested, or a topic/area tracked for reading or research.
- **FR-004**: `Reference` MUST require `title`, `ref`, and `relevance`, and MUST offer `url`, `authors`, `year`, `doi`, `status`, `isCitedBy`, and `notes` — exactly the predicates displaced from `Resource`, plus `title`.
- **FR-005**: `Reference` MUST declare `Node` as its base type, exactly as `Source`, `Entity`, `Resource`, and `Timeline` do, and MUST be registered through the same seeding mechanism as any other core type — no special-casing.
- **FR-006**: Every predicate `Reference` declares MUST itself be registered as a predicate in the seeded schema, so a freshly initialized graph lints clean.
- **FR-007**: Predicate descriptions that name `Resource` while describing external-work semantics — at minimum `ref`, `status`, and `relevance` — MUST be reworded to name `Reference`.
- **FR-008**: The `Node` base type's description MUST name all five content types that inherit from it.
- **FR-009**: Applying a patch that carries a `Reference` node to a graph initialized after this feature MUST succeed without any unknown-type diagnostic.

**Body prose derivation**

- **FR-010**: A node's leading body prose MUST be stored under `text` when its type is `Resource`, and under `relevance` when its type is `Reference`.
- **FR-011**: Leading-prose derivation for `Source` (`abstract`), `Entity` (`definition`), and every other type (`text`) MUST be unchanged, as MUST trailing-prose derivation (`notes`) for every type.
- **FR-012**: Reverting an ingest MUST reconstruct a node's prose from the same predicate the ingest wrote it to, for `Resource` and `Reference` as for every other type — the write-side and read-side derivations MUST agree for all types at all times.

**Folder layout**

- **FR-013**: A type folder's name MUST be character-for-character identical to the type name — same case, no pluralization, no abbreviation, no synonym. This MUST hold for core types and for types the tool does not recognize alike.
- **FR-014**: The folder derivation MUST NOT append a pluralizing suffix, lowercase, or otherwise transform the type name.
- **FR-015**: Initializing a graph MUST create exactly `Source/`, `Entity/`, `Resource/`, `Reference/`, `timeline/yearly/`, `timeline/monthly/`, `_schema/Property/`, and `_schema/Class/`.
- **FR-016**: Predicate schema nodes MUST be filed under `_schema/Property/` and type schema nodes under `_schema/Class/`, for seeding and for applied schema patches alike.
- **FR-017**: `timeline/` MUST remain exempt: `Timeline` nodes stay bucketed under `timeline/yearly/` and `timeline/monthly/` and are never filed flat.
- **FR-018**: `_schema/` MUST remain a namespace prefix rather than a type folder, and MUST continue to be excluded from content traversal by search and lint exactly as today, alongside `.arc/`.
- **FR-019**: Every operation that resolves a node's path — apply, batch apply, revert, backlink sweep, subgraph export, search, lint — MUST use the corrected derivation, so no two operations disagree about where a node lives.
- **FR-020**: A node's `@type` MUST continue to be read from the node itself and never inferred from its folder; the naming rule makes the folder a reliable mirror of the type, not a substitute for it.

**Breaking change**

- **FR-021**: The corrected `Resource` and the new `Reference` MUST be seeded as complete definitions into a new graph, not merged into any pre-existing definition. `required`/`optional` merge by union and union cannot retract a predicate, so merging the corrected `Resource` onto the retired one would yield the sum of both — neither definition, and conformant to no revision of the specification.
- **FR-022**: No migration, detection, or repair of a pre-existing graph is in scope. The tool MUST NOT attempt to reconcile a graph created before this feature.

### Key Entities

- **`Resource`**: A core content type. An anonymous, tag-classified fragment of an ingested document's content that does not warrant its own dedicated type — the staging ground from which a recurring pattern is later promoted to a proper domain type. Requires prose (`text`), `tags`, and a backlink to the document it was drawn from (`mentionedIn`). Files under `Resource/`.
- **`Reference`**: A new, fifth core content type. An external work the graph points to but has not ingested, or a topic/area tracked for reading or research. Records only enough to identify (`title`, `ref`), locate (`url`, `doi`, `year`, `authors`), and justify keeping the pointer (`relevance`). Owns the external-work predicates displaced from `Resource`. Files under `Reference/`.
- **Type folder**: A folder holding nodes of exactly one type, named character-for-character for that type. The mirror of `@type` on disk, never its substitute.
- **Functional folder**: A folder that is not a type folder and is exempt from the naming rule — `_schema/`, a namespace prefix whose children are themselves type folders (`Property/`, `Class/`), and `timeline/`, an index that buckets its `Timeline` nodes by granularity rather than filing them flat.
- **Leading-prose predicate**: The per-type predicate a node's opening body prose is stored under — `abstract` for `Source`, `definition` for `Entity`, `text` for `Resource`, `relevance` for `Reference`, `text` otherwise. Trailing prose is `notes` for every type.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A graph created after this feature conforms to ARCNET-CORE v0.11 on first commit: all five core content types are declared with the specified predicates, and every folder created is either type-named or one of the two named exemptions. Zero conformance diagnostics.
- **SC-002**: An author can record an external, un-ingested work as a first-class node — a capability that does not exist today at all.
- **SC-003**: Ingested `Resource` and `Reference` prose round-trips through apply and revert with zero loss and zero key drift: the predicate read back is the predicate written, for 100% of nodes of both types.
- **SC-004**: Determining a node's folder from its type requires no lookup table, no pluralization, and no case folding — the folder name equals the type name for 100% of typed nodes outside the `timeline/` exemption.
- **SC-005**: Ingesting a document that produces `Resource` and `Reference` nodes, then linting the result, yields zero type-conformance diagnostics — where the same ingest today yields one missing-required and one undeclared-predicate diagnostic per `Resource` node.
- **SC-006**: A subgraph exported from one graph and applied to another places every node at the same path in both — zero path divergence across the round trip.
- **SC-007**: Every command that resolves a node path agrees on that path: apply, revert, subgraph, search, and lint locate the same node at the same location in 100% of cases, with no operation left reading the retired layout.

## Assumptions

- **This is a deliberate breaking change; existing graphs are not migrated.** Confirmed in clarification. Graphs created before this feature keep the retired folder layout and the retired `Resource` definition, and this feature does nothing about them — no migration command, no detection, no refusal, no repair. An author holding such a graph re-initializes and re-ingests. This is consistent with the precedent set when the patch manifest moved from `kind: patch` to `@type: patch`, where compatibility was likewise explicitly not required.
  - *Consequence, verified rather than assumed*: the break **fails closed, not silently**. Schema resolution reads `_schema/Property/` and `_schema/Class/` and treats a missing schema folder as a hard load failure, never a skip. A pre-feature graph has neither folder, so every command that resolves the schema — apply, batch, revert, subgraph, grep, lint, serve — refuses immediately and names the missing folder. There is no window in which a pre-feature graph is half-written into the new layout. The residual gap is only the wording of that refusal: it reports a missing schema folder rather than "this graph predates the folder rename". Improving that message is in scope as ordinary error-message quality; detecting, repairing, or migrating the graph is not.
- **Re-seeding cannot substitute for a migration.** A type's `required`/`optional` predicates merge by union, and union has no retraction. Seeding the corrected `Resource` over the retired one would produce a type that still requires `ref` and still offers `url`, `year`, `doi`, `status`, `isCitedBy` alongside the new predicates. This is why the break is clean rather than incremental.
- **`Reference`'s predicate lists follow the upstream revision note, not §11.6's `Class` block.** Confirmed in clarification. It is the only reading under which §11.6's own worked example conforms to the type it illustrates.
- **`Reference` inherits `Resource`'s former type-level merge behaviour.** `Resource` carried a first-write-wins type merge while it meant "external work"; that behaviour belongs with the external-work semantics, so `Reference` takes it. `Resource`'s own merge behaviour under its corrected meaning is left as-is, since neither the feature description nor the upstream specification states one.
- **`Resource`'s inherited and structural predicates are retained.** Beyond the four predicates §11.4 names, `Resource` continues to receive whatever the universal `Node` base type contributes, on the same terms as every other content type.
- **No new predicates are introduced.** Every predicate `Reference` declares — `title`, `ref`, `relevance`, `url`, `authors`, `year`, `doi`, `status`, `isCitedBy`, `notes` — is already registered. Only their descriptions and their owning type change.
- **Domain-profile folders follow the same rule.** A node of a type the tool does not recognize files under its type name verbatim, so a domain profile's nodes need no per-profile configuration.
- **Out of scope: other v0.11 drift.** The upstream revision also moves `published` into `Source`'s required list, adds `tags` as optional to `Source` and `Entity`, and trims `Timeline`'s required list to `cites` alone. None is part of the role correction or the folder rule this feature addresses; each is a separate change and is deliberately excluded.
- **Out of scope: a folder-conformance lint rule.** Making the folder ↔ type rule *checkable* by lint is a natural follow-up to making it *true*, but the feature description asks only for correct filing. No new lint rule is added here.
- **Out of scope: upstream domain profiles and the example graph.** The upstream revision note records that the domain profiles and the reference example graph still use the retired layout and need their own follow-up pass. That work belongs upstream, not to this tool.
