# Feature Specification: Machine-Readable Predicate & Type Schema

**Feature Branch**: `011-machine-readable-schema`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "Turn arc's graph schema mechanism from a name-existence registry into a fully machine-readable schema, matching ARCNET-CORE v0.7 §9. Today, arc init seeds `_schema/nodes/<kind>.md` (kind name + merge behavior only) and `_schema/predicates/<name>.md` (existence only — the file's body is never read back). Both are populated from hardcoded values built into arc itself. The new spec requires every predicate in use across the graph to be registered as a real node at `_schema/predicates/<name>.md` declaring, in machine-readable form: its serialization role (one of meta/text/href/edge/link), its merge behavior, an optional display label, and an optional standard-vocabulary alignment — plus a human-readable description in the body. Every `@type` in use must be registered as a real node at `_schema/types/<name>.md` (renamed from today's `_schema/nodes/`) declaring, via a `## Requires` and a `## Optional` section, which predicates a conforming instance of that type must or may carry. arc must be able to load every predicate/type node in a graph once and build an in-memory index other commands consult — this replaces today's hardcoded Go constants as the source of truth, while still seeding a graph with CORE's own baseline vocabulary on `arc init` so a freshly initialized graph is self-describing from the start. This schema index must be usable by arc apply, arc lint, and any future consumer. Backward compatibility: existing graphs whose `_schema/nodes/` folder is not required — the absence of a valid schema causes failing; `arc init` supports only the new schema. Out of scope: per-predicate merge algebra changes to arc apply's actual merge logic; lint rule changes."

## Clarifications

### Session 2026-07-07

- Q: `_schema/types/<name>.md` (CORE §9.2) declares only `## Requires`/`## Optional` — merge behavior moved entirely onto predicates (§9.1/§9.3). But `arc apply`'s actual whole-node merge dispatch (out of scope to change) is currently keyed by one merge behavior per type (`source`→none, `entity`→union, `resource`→union-first-writer, `timeline`→append), previously read from `_schema/nodes/<kind>.md`'s `merge` field. Once that field disappears from the renamed `_schema/types/` node, where should `arc apply`'s existing whole-node dispatch keep reading each type's merge behavior from? → A: Keep a `merge` field on `_schema/types/<name>.md` in addition to CORE's `## Requires`/`## Optional` — an arcnet-cli-specific superset of CORE's documented shape — so `arc apply`'s existing whole-node merge dispatch keeps working exactly as today, with zero regression, until a future feature wires true per-predicate merge dispatch.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A freshly initialized graph fully describes its own vocabulary (Priority: P1)

A user initializes a new knowledge graph. Instead of stub files that merely assert "this predicate/kind exists," every predicate and type the core specification defines is registered as a real, machine-readable document: a predicate states its serialization role, its merge behavior, and (where applicable) a display label and standard-vocabulary alignment; a type states exactly which predicates a conforming instance must carry and which it may carry — all readable without running any tool and without consulting the tool's own source code.

**Why this priority**: Every other capability in this feature — recognition during `arc apply`, future conformance checking, any other consumer — depends on the schema documents themselves actually carrying the declarations they claim to. This is the foundational deliverable; nothing else has anything correct to read until this exists.

**Independent Test**: Can be fully tested by initializing a new graph and inspecting `_schema/predicates/` and `_schema/types/` — delivers a graph whose full vocabulary is genuinely machine-readable with no other command required.

**Acceptance Scenarios**:

1. **Given** a directory with no existing graph, **When** the user initializes a new graph, **Then** every predicate the core specification defines is registered at `_schema/predicates/<name>.md`, declaring its role, merge behavior, and (where the core specification provides one) label and standard-vocabulary alignment, plus a descriptive body.
2. **Given** a freshly initialized graph, **When** the user inspects any document under `_schema/types/`, **Then** it declares, via a `## Requires` section and a `## Optional` section, exactly the predicates the core specification requires or permits for that type, plus a descriptive body.
3. **Given** a freshly initialized graph, **When** the user inspects the graph's file tree, **Then** no `_schema/nodes/` folder exists — `_schema/types/` is the only place type declarations live.
4. **Given** the canonical core specification cannot be retrieved at initialization time (e.g., no network access), **When** the user initializes a new graph, **Then** initialization still succeeds, seeding `_schema/predicates/` and `_schema/types/` from the tool's built-in copy of that vocabulary instead.

---

### User Story 2 - Applying content keeps recognizing and now fully registers new vocabulary (Priority: P2)

A user applies a patch that introduces a predicate or `@type` not yet present in the graph's schema. As today, the contribution is not rejected — but the schema document created for it is no longer a bare existence stub: it carries the same machine-readable declaration (role/merge for a predicate; empty but present `## Requires`/`## Optional` for a type) that every other schema document carries, using safe defaults, so it is genuinely readable and ready for a person to refine later.

**Why this priority**: The graph format is designed to grow with domain-specific vocabulary. This story keeps that self-extension working under the new, richer schema shape — without it, `arc apply` would either have to reject unknown vocabulary outright or keep producing schema documents that violate the new shape.

**Independent Test**: Can be fully tested by applying a patch that introduces a previously unseen predicate or `@type`, then confirming a fully machine-readable schema document exists for it immediately afterward.

**Acceptance Scenarios**:

1. **Given** a patch introduces a predicate with no existing schema document, **When** the patch is applied, **Then** the tool creates `_schema/predicates/<name>.md` declaring a role and merge behavior using safe defaults, and the patch's content is still applied successfully.
2. **Given** a patch introduces an `@type` with no existing schema document, **When** the patch is applied, **Then** the tool creates `_schema/types/<name>.md` with empty `## Requires`/`## Optional` sections, ready for later curation.
3. **Given** a patch introduces a predicate or `@type` that already has a schema document, **When** the patch is applied, **Then** the existing schema document is left unchanged.
4. **Given** a patch application discovers and registers new predicates or types, **When** the application finishes, **Then** the new schema documents are recorded in the same commit as the rest of that patch's changes.
5. **Given** a graph whose `_schema/` folder is absent, or contains a predicate/type document that does not conform to the mandatory machine-readable shape, **When** any command that depends on the schema runs, **Then** the command fails with a clear error identifying the missing or non-conforming document, before making any other change.

---

### User Story 3 - The schema becomes a reusable index, not tool-internal knowledge (Priority: P3)

Anyone building on top of `arc` — today's `arc apply`, today's `arc lint`, or a future command — needs to know a predicate's role and merge behavior, or a type's required and optional predicates. Instead of guessing from a document's Markdown shape or duplicating the tool's own hardcoded vocabulary, every consumer loads the graph's schema once into an in-memory index and asks it directly.

**Why this priority**: This is the structural payoff of making the schema machine-readable — a single, reliable, in-graph source of truth every consumer shares — but it depends on Stories 1 and 2 already producing well-formed schema documents to load.

**Independent Test**: Can be fully tested by loading the schema index for a graph that has been initialized and had at least one patch applied, and confirming it reports the correct role/merge for every registered predicate and the correct required/optional sets for every registered type, matching what a person reading the same documents by eye would conclude.

**Acceptance Scenarios**:

1. **Given** a graph with a mix of core and discovered predicates, **When** a command loads the schema index, **Then** it reports, for every registered predicate, its role and merge behavior, and its label/alignment when the document declares one.
2. **Given** a graph with a mix of core and discovered types, **When** a command loads the schema index, **Then** it reports, for every registered type, its full required and optional predicate sets.
3. **Given** `arc apply` and `arc lint` both need to recognize the same graph's vocabulary, **When** each runs, **Then** both consult the same schema index built the same way — neither falls back to a separate, hardcoded copy of the vocabulary.
4. **Given** a user edits a predicate's declared role, merge behavior, or a type's `## Requires`/`## Optional` list, **When** a command next loads the schema index, **Then** the edited declaration is the one reported.

---

### Edge Cases

- What happens when a predicate or type schema document is missing a mandatory field (role/merge for a predicate; `@type: Class` for a type) or carries an invalid value (e.g., a role outside meta/text/href/edge/link)? Loading the schema index fails outright, naming the offending document and field — the index is never built from partially-invalid data (a deliberate reversal of the previous "fall back to safe default" edge-case behavior, per the user's explicit direction that an invalid schema now causes failure).
- What happens when a graph has no `_schema/` folder at all? Any command that depends on the schema index fails, the same as an invalid one — a valid schema is a hard precondition for operating on a graph, not an optional enhancement.
- What happens when the same previously unseen predicate or `@type` appears more than once within a single patch application? Exactly one schema document is created for it, not one per occurrence.
- What happens when a discovered predicate is observed in more than one structural position within the same patch (e.g., once as an edge, once as a plain attribute)? The first-observed position's role is the one recorded; later occurrences in the same application do not change an already-registered predicate's declared role.
- What happens when initializing a graph that already has a `_schema/` folder (already initialized)? The tool refuses, consistent with existing already-initialized-graph protection — no schema content is lost or overwritten.
- What happens to a graph created before this feature, whose `_schema/nodes/` folder still exists in the old existence-only shape? No automatic migration is performed; the old folder is not recognized by the new schema index, so any command depending on the schema fails until the graph is re-initialized or its schema documents are hand-migrated to the new shape.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST represent every predicate in use across a graph as a machine-readable node at `_schema/predicates/<name>.md`, whose front matter declares a mandatory serialization role (one of `meta`/`text`/`href`/`edge`/`link`) and a mandatory merge behavior, and MAY declare a display label and a standard-vocabulary alignment, with a mandatory human-readable description in the document body.
- **FR-002**: The tool MUST represent every `@type` value in use across a graph as a machine-readable node at `_schema/types/<name>.md` — replacing today's `_schema/nodes/<kind>.md` — whose document body declares, via a `## Requires` section and a `## Optional` section, exactly which predicates a conforming instance of that type must and may carry, alongside a mandatory human-readable description.
- **FR-003**: The tool MUST load every predicate and type node under a graph's `_schema/` folder once per command invocation and build a single in-memory schema index from them, exposing each predicate's role, merge behavior, optional label, and optional alignment, and each type's full required and optional predicate sets.
- **FR-004**: This schema index MUST be the runtime source of truth every schema-aware command consults to recognize known vs. unknown predicates and types — replacing the tool's previously hardcoded Go constants (the fixed four kinds, the fixed thirteen predicates) as that source of truth.
- **FR-005**: `arc apply` MUST consult the schema index built this way to recognize whether an incoming patch's predicates and `@type` values are already registered, exactly as it recognizes them today, but against the richer, machine-readable declarations rather than bare existence.
- **FR-006**: `arc lint` MUST consult the same schema index — built the same way, from the same `_schema/` documents — as `arc apply`, rather than maintaining or falling back to a separate copy of the graph's vocabulary.
- **FR-007**: `arc init` MUST seed a freshly initialized graph's `_schema/predicates/` and `_schema/types/` folders with the complete, fully machine-readable vocabulary the core specification itself defines — every core predicate's role/merge/label/alignment/description, and every core type's `## Requires`/`## Optional`/description — so the graph is self-describing from its first commit. "Complete" means every predicate the core specification documents as its own vocabulary, not only the narrower subset the tool hardcodes today: identity, content, metadata/control, and structural predicates; semantic predicates; citation predicates; type-specific predicates (e.g., those used only by `source`/`entity`/`resource`/`timeline`); and the schema mechanism's own predicates (`role`, `merge`, `label`, `aligned`, `description`, `required`, `optional`). It likewise means every type the core specification documents, including the schema mechanism's own `Property`/`Class` markers, since a predicate/type node's own `"@type"` value is itself a type in use.
- **FR-008**: `arc init` MUST continue to succeed when the canonical core specification cannot be retrieved over the network, seeding `_schema/` from the tool's built-in copy of that vocabulary instead, with no regression to the existing offline-first guarantee.
- **FR-009**: `arc init` MUST only ever produce a graph with this new machine-readable schema shape; it MUST NOT offer, nor fall back to, the previous existence-only schema format.
- **FR-010**: When `arc apply` encounters a predicate with no existing schema document, it MUST auto-register `_schema/predicates/<name>.md` for it, assigning the safe-default merge behavior and a role inferred from the structural position in which the predicate was first observed — never left with a missing mandatory field.
- **FR-011**: When `arc apply` encounters an `@type` with no existing schema document, it MUST auto-register `_schema/types/<name>.md` for it, with empty `## Requires`/`## Optional` sections, ready for later curation.
- **FR-012**: `arc apply` MUST NOT create a duplicate or overwrite an already-registered predicate's or type's schema document.
- **FR-013**: Schema documents auto-registered while applying a patch MUST be included in that same patch application's single commit, not committed separately.
- **FR-014**: Every command that depends on the schema index MUST fail — before making any other change — when a graph's `_schema/` folder is absent, or when an existing predicate or type document under it does not conform to the mandatory machine-readable shape (missing or invalid role/merge on a predicate; missing/invalid `@type` marker on a type), naming the offending document and the missing or invalid field.
- **FR-015**: ~~`_schema/types/<name>.md` MUST additionally declare that type's whole-node merge behavior (an arcnet-cli-specific field beyond the core specification's own documented shape), so `arc apply`'s existing whole-node merge dispatch continues to read each type's merge behavior exactly as it does today, with no change to the dispatch logic itself.~~ Superseded by spec 012 FR-015 (which retired the whole-node dispatch this field bridged) and spec 012 FR-020 (Bugfix 018/BUG-001, 2026-07-19): `_schema/types/<name>.md` MAY declare a whole-node `merge` field for continuity with the built-in/auto-registered shape, but schema-index loading MUST NOT require it to be present or valid — its presence is no longer load-bearing and MUST NOT block loading or importing an otherwise-conformant `Class` document.
- **FR-016**: Migrating an already-existing graph's old-format `_schema/nodes/` documents into the new `_schema/types/` shape is explicitly out of scope; no automatic migration path is provided.
- **FR-017**: FR-010's "role inferred from the structural position in which the predicate was first observed" MUST recognize a text-shaped body-block occurrence (a `**Label**`-headed block whose content isn't wikilink-shaped), not only an edge-position occurrence — auto-registering such a predicate as `role: text, merge: append` rather than defaulting every unrecognized predicate to `role: edge, merge: union`. *(Bugfix 010/BUG-002, 2026-07-20)*

### Key Entities

- **Schema Index**: The in-memory structure built once per command invocation from a graph's `_schema/` documents, exposing every registered predicate's role/merge/label/alignment and every registered type's required/optional predicate sets to whichever command loaded it — a graph-specific realization of the "Schema Index" the core specification family already documents as the standard read-time convenience derived from a graph's `Property`/`Class` nodes.
- **Predicate Schema Node** (`_schema/predicates/<name>.md`): The versioned, machine-readable declaration of one predicate — its serialization role, merge behavior, optional label and standard-vocabulary alignment, and descriptive prose.
- **Type Schema Node** (`_schema/types/<name>.md`, replacing `_schema/nodes/<kind>.md`): The versioned, machine-readable declaration of one `@type` value — the predicates a conforming instance must and may carry, its whole-node merge behavior, and descriptive prose.
- **Discovered Predicate / Type**: A predicate or `@type` value encountered mid-`arc apply` with no existing schema document, auto-registered with safe defaults into the appropriate `_schema/` subfolder.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of the core specification's documented predicates and types are represented as fully machine-readable schema nodes (role/merge/label/alignment for predicates; required/optional predicate sets for types) immediately after `arc init`.
- **SC-002**: A person or a future tool can determine any registered predicate's role and merge behavior, or any registered type's required and optional predicates, by consulting the schema index once — with no need to guess from a document's Markdown shape or read the tool's own source code.
- **SC-003**: 100% of previously unseen predicates or types encountered while applying a patch have a corresponding, fully machine-readable schema document in the graph immediately after that application completes.
- **SC-004**: Every command that depends on the schema fails clearly, before making any other change, on 100% of the cases where a graph's schema is missing or does not conform to the mandatory shape — zero silent fallbacks.
- **SC-005**: `arc apply` and `arc lint` recognize identical sets of known predicates and types for the same graph, since both consult the same schema index built the same way.

## Assumptions

- Auto-registered predicates are assigned `merge: union` as their safe default (mirroring the existing safe-default precedent already established for auto-discovered vocabulary), and a role inferred from the structural position the predicate was first observed in. ~~today's auto-discovery path only ever observes edge-position predicates, so this in practice yields `role: edge` until a future feature broadens auto-discovery to other positions.~~ That future feature is FR-017 (Bugfix 010/BUG-002, 2026-07-20): auto-discovery now also observes text-position occurrences (a labeled block whose content isn't wikilink-shaped), yielding `role: text, merge: append` for those instead.
- Auto-registered types are given empty `## Requires`/`## Optional` sections (maximally permissive) since the discovery context provides no information about which predicates a conforming instance must or may carry; a person curates this afterward by editing the type's schema node directly.
- Reconciling the core specification's seven-value merge-behavior vocabulary (`immutable`/`union`/`firstWriteWin`/`fillIfEmpty`/`lastWriteWin`/`append`/`validatedOverwrite`) with the tool's existing internal merge-dispatch vocabulary is not addressed by this feature — a predicate's `merge` field is recorded and exposed by the schema index as data; wiring it into `arc apply`'s actual per-node merge algorithm is separate-feature scope, per the stated exclusion.
- FR-015's type-level `merge` field is a deliberate, temporary deviation from the core specification family's own documented Schema Index shape (which derives a type's information solely from its `## Requires`/`## Optional` edges, with no per-type merge value) — kept only so `arc apply`'s existing whole-node merge dispatch has zero regression during this feature, and expected to be retired once a future feature moves `arc apply` to genuine per-predicate merge dispatch sourced from the schema index's predicate-level `merge` values instead. That retirement happened in two steps: spec 012 retired the field's *function* (whole-node dispatch no longer runs); spec 012 FR-020 (Bugfix 018/BUG-001, 2026-07-19) retired its *mandatory presence* in validation, since a real, CORE-conformant `Class` document has no reason to carry it.

**Bugfix**: 2026-07-19 — 018/BUG-001 struck FR-015's mandatory-presence requirement (superseded by spec 012 FR-020) and annotated this assumption: the type-level `merge` field, already functionally retired by spec 012, is no longer required to be present either.

**Bugfix**: 2026-07-20 — 010/BUG-002 added FR-017 and closed the auto-discovery-only-observes-edges assumption this spec itself flagged as pending "a future feature": `arc apply` was silently dropping patch body content shaped as prose/text rather than wikilinks, in part because auto-registration never recognized a text-shaped occurrence as a reason to register `role: text` instead of always defaulting to `role: edge`. The actual parsing fix (resolving a labeled block's predicate/role at parse time) lives in spec 010's tasks.md Phase 6, since it touches `internal/core`'s shared parser, not this feature's own schema-index-building code.
- Whether a predicate or `@type` actually used somewhere in a graph's content has a corresponding registered schema document (as opposed to the schema documents that do exist being individually well-formed) is a conformance question left to `arc lint`'s separate, future rule-change feature; this feature's failure behavior (FR-014) concerns the well-formedness of the schema documents that exist, not completeness of registration against content.
- This feature governs newly initialized graphs and the ongoing operation of any graph already carrying the new schema shape; it does not provide a path for migrating an existing graph's old-format `_schema/nodes/` folder forward (FR-016).
- Two sibling hardcoded vocabularies discovered alongside today's kind/predicate registry are left untouched by this feature, since neither is what the user's request targets: the citation-predicate set `arc lint` checks a citation edge's predicate against (a fixed list, independent of `_schema/predicates/`) keeps working exactly as it does today, even though citation predicates are now also present in the schema index as part of the complete core vocabulary (FR-007) — reconciling the two is a lint-rule change, out of scope; and the kind-to-folder-name mapping `arc apply` uses to decide each node's on-disk directory (e.g., `entity` → `entities/`) is a presentation/layout concern independent of predicate/type recognition and is not affected by this feature.
