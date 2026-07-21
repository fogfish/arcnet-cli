# Feature Specification: Predicate-First Graph Node Model

**Feature Branch**: `010-predicate-node-model`

**Created**: 2026-07-07

**Status**: Draft

**Input**: User description: "Rewrite arc's internal graph node representation to match ARCNET-CORE v0.7 / ARCNET-AST v0.6's predicate-first data model, replacing the current pre-0.5 shape."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Author a node with open, predicate-keyed identity and content (Priority: P1)

A graph maintainer creates or edits a node file. They declare the node's identity and type explicitly (`@id` matching the filename, `@type` naming its kind), give it as many named prose sections as the node needs (an `abstract`, a `definition`, a `relevance`, a `claim`, or any other predicate name — not limited to a fixed "text" and "notes"), and list its attributes and outgoing links. When arc reads the file back and writes it out again unchanged, the file's content and every declared connection survive exactly.

**Why this priority**: This is the foundational capability — every other consumer (lint, grep, subgraph, serve, apply) depends on the node file being read and written correctly in the new shape. Without this, nothing else in the feature has a foundation.

**Independent Test**: Can be fully tested by writing a node file in the new shape, parsing it, re-rendering it, and diffing the result against the original — delivers value on its own by proving the new format is representable and stable.

**Acceptance Scenarios**:

1. **Given** a node file whose front matter declares `"@id"` equal to the file's basename and `"@type"` naming its kind, **When** arc reads the file, **Then** arc recognizes the file as valid and establishes the node's identity and type from those two declarations alone (no fallback to any other field).
2. **Given** a node file with several independently named prose sections (e.g. `abstract`, `relevance`), **When** arc reads and then re-writes the file, **Then** every prose section is preserved individually, under its own name, with its original content intact.
3. **Given** a node file whose front matter has a single-valued attribute (e.g. one author) and a multi-valued attribute (e.g. several tags), **When** arc reads the file, **Then** both attributes are treated uniformly as ordered lists of values internally, regardless of how many values each has.
4. **Given** a node file whose links to other nodes were originally written as a flat bulleted list, and another node file whose links were originally written grouped under a heading, **When** arc reads both files, **Then** both nodes' outgoing links land in one single ordered list per node, with no distinction in how they were stored.
5. **Given** a node produced by arc (round-tripped at least once), **When** it is read and re-written again, **Then** the result is byte-for-byte stable (a second round-trip introduces no further change).

---

### User Story 2 - Operate the full toolchain against a predicate-first graph (Priority: P2)

A graph maintainer runs their everyday arc commands — applying a patch, linting the graph, searching it, extracting a subgraph, browsing it through the serve interface, and exporting `arc subgraph --json` for another tool to consume — against a graph stored in the new predicate-first shape. Every command works correctly: patches merge into the right node identity, lint checks see the real attributes and links, search finds content in any named prose section, and subgraph export reproduces the graph faithfully in both Markdown and JSON.

**Why this priority**: The rewrite only delivers value once every existing command understands the new shape end to end; a graph maintainer must be able to keep using the whole toolchain, not just read a single file in isolation.

**Independent Test**: Can be tested independently of User Story 3 by exercising each command (apply, lint, grep, subgraph, serve, subgraph --json) against a small predicate-first fixture graph and confirming each produces correct output.

**Acceptance Scenarios**:

1. **Given** a predicate-first graph and a patch contributing to an existing node, **When** the patch is applied, **Then** the contribution merges into the node identified by its `@id`, using its declared attributes, prose predicates, and links.
2. **Given** a predicate-first graph, **When** lint checks are run, **Then** lint evaluates the graph's real attributes and links rather than a stale two-slot prose shape.
3. **Given** a predicate-first graph with content in a non-default named prose predicate (e.g. `definition`), **When** a search is run against that content, **Then** matching nodes are found.
4. **Given** a predicate-first graph, **When** a subgraph is extracted (both as Markdown and as `--json`), **Then** every node's identity, type, attributes (as lists), prose predicates, and links are represented completely and consistently between the two output forms.
5. **Given** a predicate-first graph, **When** it is browsed through the serve interface, **Then** node identity, prose, attributes, and links display correctly.

---

### User Story 3 - Fail safely on a pre-0.5 graph instead of misreading it (Priority: P3)

A graph maintainer who still has an old-format graph (using `kind`/`id`-or-fallback identity, two fixed prose slots, and split link containers) runs any arc command against it. Instead of silently misinterpreting the file's structure or producing corrupted output, arc detects that the file does not match the new shape and stops with a clear error.

**Why this priority**: Lower priority than actually delivering the new format, but essential as a safety net — without it, a maintainer could silently lose or corrupt content by running the new arc against an old graph.

**Independent Test**: Can be tested independently by pointing arc at a fixture graph still written in the pre-0.5 shape and confirming every command exits with an error rather than partial or corrupted output.

**Acceptance Scenarios**:

1. **Given** a node file using the old `kind` front-matter field with no `"@id"`/`"@type"`, **When** any arc command reads it, **Then** arc exits with a clear error identifying the file as an unsupported format, and makes no write.
2. **Given** a node file that has `"@type"` but is missing `"@id"`, **When** arc reads it, **Then** arc exits with an error rather than falling back to a title or period field.
3. **Given** a node file whose `"@id"` does not equal the file's basename, **When** arc reads it, **Then** arc exits with an error rather than accepting the mismatch.
4. **Given** an old-format graph, **When** `arc apply` is run with a patch, **Then** no node file in the graph is modified as a result of the failed read.

---

### Edge Cases

- A node file has `"@type"` and a correct `"@id"`, but zero attributes, zero prose predicates, and zero links (a stub node) — arc must still read and re-render it without error.
- A node declares the same prose predicate name twice (e.g. two separate `abstract` sections) — behavior must be deterministic (this feature defines *representation*, not merge behavior across duplicates within one file; see Out of Scope).
- An attribute list contains a mix of scalar types (string, number, boolean) — each element is preserved as authored.
- A link's target does not correspond to any existing node in the graph — the link is still represented (arc lint, not this feature, is responsible for flagging broken targets).
- A node's front matter has an unrecognized attribute key the current schema has never seen — the key and its value(s) must be preserved verbatim, not dropped.
- A patch or node body has a `**Label**`-headed block (list or prose) whose content does not structurally look like a list of links, and/or whose label does not (yet) match any predicate the schema recognizes — the block's content must still be captured verbatim as prose (auto-registering a new `role: text` predicate from the label when nothing matches), never silently dropped just because it isn't wikilink-shaped (FR-019, Bugfix BUG-002, 2026-07-20).
- A `**Label**`-resolved `text`-role block's content is a Markdown list (not a single prose paragraph) — each list item's own literal markup (wikilink brackets, inline `predicate::` tags) must survive a read/write cycle exactly as authored, not merely as extracted words; the same free-prose inline-link extraction/reconstruction heuristic used for paragraph text is not sufficient for list items, whose boundaries and literal syntax are structurally significant (FR-020, Bugfix BUG-003, 2026-07-21).
- A wikilink inside `text`-role content is immediately followed by a non-whitespace, non-punctuation character (an inflectional suffix, e.g. `[[LLM]]s`) — reinsertion of the link's markup on write must not silently fail just because the boundary heuristic designed for prose doesn't recognize this pattern (FR-020, Bugfix BUG-003, 2026-07-21).
- A predicate is auto-registered from a `**Label**`-headed block whose label carries multi-word/spaced text (e.g. `"Related Aporias"`) — the auto-registered predicate must be able to recover that exact label on a later write, not a `titleCaseType`-mangled single-word approximation of its derived id (FR-021, Bugfix BUG-003, 2026-07-21).
- A `**Label**`-headed block's content resolves to edges (wikilink occurrences) and the block had its own label distinct from any other block in the same node body (e.g. `**Assumes**`, `**Derived From**`, `**Related Aporias**` each in the same Hypothesis node) — each must remain its own distinctly labeled, grouped block on write, never collapsed together with another such block's occurrences into one undifferentiated flat list (FR-022, Bugfix BUG-003, 2026-07-21).
- A patch document (the format `arc apply` consumes) references a node contribution — its identity declaration must follow the same `@id`-equals-basename rule as a standalone node file, with no fallback to `title`/`period`/legacy `id`. Unlike a standalone file, a patch section's `"## <ID>"` heading itself satisfies this declaration (mirroring the pre-existing CORE §12.2 convention where the heading conveys identity); an explicit `"@id"` key duplicated inside the node's own yaml fence is optional, and if present MUST agree with the heading or the contribution is rejected as inconsistent (BUG-001).
- Two node files in the same graph declare conflicting `@type` for what would be the same `@id` — this is a normal merge/conflict scenario, unaffected by this feature's representation change.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Arc MUST require every node file's front matter to declare `"@id"` and `"@type"` as explicit, quoted keys, and MUST reject any node file where either is absent.
- **FR-002**: Arc MUST establish a node's identity solely from its `"@id"` declaration, and MUST reject the file if `"@id"` does not equal the file's basename (without extension) — no fallback to `title`, `period`, or any other field is permitted.
- **FR-003**: Arc MUST establish a node's type solely from its `"@type"` declaration, replacing the old `kind` field entirely.
- **FR-004**: Arc MUST represent every front-matter attribute (other than `"@id"`/`"@type"`) internally as an ordered list of values, including attributes that carry only one value.
- **FR-005**: Arc MUST support an open, unbounded set of named prose fields per node (e.g. `abstract`, `definition`, `relevance`, `claim`, and any other predicate name), rather than a fixed pair of slots.
- **FR-006**: Arc MUST preserve each prose field's content verbatim under its own name across a read/write cycle, without merging, renaming, or reordering distinctly named prose fields relative to each other.
- **FR-007**: Arc MUST represent every outgoing link from a node (regardless of whether the source document wrote it as a flat bulleted list or grouped under a heading/label) as membership in one single ordered list per node.
- **FR-008**: Arc MUST decide, at the point content is written back to Markdown, whether a given link renders as a flat bullet or grouped under a heading — this decision MUST be derived from the link's own predicate, never fixed by how the source document happened to write it.
- **FR-009**: Arc MUST preserve unrecognized attribute keys and unrecognized predicates verbatim (value and position), whether or not the current graph's schema has previously seen them.
- **FR-010**: `arc apply`, `arc lint`, `arc grep`, `arc subgraph` (including `--json` output), and `arc serve` MUST all read and write graph files in the predicate-first shape defined by this feature.
- **FR-011**: A node contribution inside a patch document (the exchange format `arc apply` consumes) MUST declare its identity under the same `"@id"`-equals-basename rule as a standalone node file, with no fallback to `title`/`period`/legacy `id`. For a patch specifically, the node's own `"## <ID>"` section heading satisfies this declaration by itself (mirroring the pre-existing CORE §12.2 convention); an explicit `"@id"` key duplicated inside the node's own yaml fence is optional, not mandatory, and when present MUST agree with the heading or the contribution MUST be rejected as inconsistent. *(Bugfix: 2026-07-07 — BUG-001 clarified that the heading itself, not a redundant yaml-fence key, is what satisfies this requirement for a patch contribution.)*
- **FR-018**: A node contribution inside a patch document MUST establish its type from the enclosing `"# <Type>"` section heading when no explicit `"@type"` key is present inside the node's own yaml fence; when both are present, they MUST agree, or the contribution MUST be rejected as inconsistent. This mirrors FR-011's identity carve-out and the pre-existing CORE §12.2 convention, and does not apply to a standalone node file, which has no section heading to derive from and MUST declare `"@type"` explicitly in its own front matter (FR-003). *(Bugfix: 2026-07-07 — BUG-001 added this requirement; a patch contribution that declares neither an explicit `"@type"` key nor has a `"# <Type>"` heading — impossible under CORE §12.2's own H1/H2 structure, but noted for completeness — is rejected under FR-012.)*
- **FR-012**: Arc MUST detect when a graph file does not match the predicate-first shape (e.g. it uses the old `kind` field, or is missing `"@id"`/`"@type"`, or `"@id"` does not match the basename) and MUST exit with a clear, non-zero-status error identifying the offending file, without writing any output.
- **FR-013**: Arc MUST NOT silently reinterpret an old-format graph file under the new shape's rules (no partial read, no best-effort guess at identity or content).
- **FR-014**: Reading a node file into arc's internal representation and immediately writing it back out MUST reproduce the original file's content and connectivity without loss (round-trip fidelity), except where cosmetic normalization of link-grouping layout is explicitly permitted.
- **FR-015**: A node written by arc, when read back a second time and written again, MUST produce byte-for-byte identical output to the first write (idempotent round-trip / stable serialization).
- **FR-016**: Arc MUST preserve the item order within an attribute's value list, within each prose field, and within the node's overall link list.
- **FR-017**: `arc subgraph --json` MUST expose every node's attributes as lists, its prose fields by name, and its links as one unified ordered collection, consistent with what a Markdown round-trip of the same node would produce.
- **FR-019**: When parsing a patch document or standalone node file, a `**Label**`-prefixed body block (list or paragraph) MUST have its predicate identity resolved against the graph's schema index, by matching the label text to a registered predicate's resolved display label — the inverse of the label resolution `RenderNode`/`RenderPatch` already perform. The resolved predicate's declared role, not the block's own structural shape, MUST determine whether its content is captured as a link (`edge`/`link` role) or as prose under that predicate's own name (`text` role) — realizing FR-005/FR-006/FR-009 for this shape of content, which this feature's own plan.md explicitly deferred pending schema role knowledge (spec 011/013). A block whose label resolves to no registered predicate, or that carries no label at all, MUST still be captured as prose (auto-registering a new `role: text` predicate from the label when one exists) rather than discarded. *(Bugfix BUG-002, 2026-07-20 — "captured as prose" here means content is present under the right key; it does not by itself guarantee the captured content's own source markup, or the block's label/grouping, survive a subsequent write — see FR-020/FR-021/FR-022, Bugfix BUG-003, 2026-07-21.)*
- **FR-020**: A `**Label**`-resolved `text`-role block's content, when captured under `Texts[predicateID]` (FR-019), MUST preserve each list item's own literal inline markup (wikilink brackets, inline `predicate::` tags) verbatim, rather than passing list-item content through the same inline-link extraction/reconstruction heuristic used for free-flowing paragraph prose — a heuristic whose boundary conditions (e.g. a wikilink immediately followed by a non-whitespace, non-punctuation suffix) are not reliable for the short, self-contained, structurally-delimited nature of list items.
- **FR-021**: A predicate auto-registered from a `**Label**`-headed block (FR-019's "auto-registering a new `role: text` predicate from the label") MUST have its schema document's `label` attribute set to the block's original literal label text, so the predicate's display label (as already defined by spec 013 FR-004) recovers the exact original heading/bold-label text — including spacing and casing — on a later write, rather than falling back to a derived-id-based approximation.
- **FR-022**: When arc auto-registers a predicate discovered from a `**Label**`-headed block whose content resolves to edges (wikilink occurrences), it MUST register that predicate with `role: link` (grouped/headed rendering per spec 013 FR-001/FR-004/FR-014), not `role: edge` (flat), so the block's original distinct grouping and label survive a round-trip. Edges observed in a bare, label-less list (no enclosing `**Label**`/`## Label` block) are unaffected and continue to default to `role: edge`.
  *(FR-020/FR-021/FR-022 added — Bugfix BUG-003, 2026-07-21: BUG-002's fix satisfied FR-019's content-presence requirement but a `**Label**`-resolved block's list-item markup, label recoverability, and per-block grouping were not preserved on write — a narrower but still-severe violation of this spec's own pre-existing FR-006/FR-015 verbatim/byte-stable guarantees and spec 013's label/grouping guarantees.)*

### Key Entities

- **Node**: The graph's addressable unit. Identified by `@id` (equal to its filename) and typed by `@type`. Carries an open set of attributes (each an ordered list of values), an open set of named prose fields, and one unified ordered list of outgoing links.
- **Attribute**: A named, front-matter-declared predicate whose value is always represented as an ordered list, whether the node declares one value or several (e.g. `tags`, `authors`, `category`).
- **Prose Field**: A named, body-declared predicate whose value is a block of Markdown prose (e.g. `abstract`, `definition`, `relevance`, `claim`). Any number of distinctly named prose fields may appear on one node.
- **Link**: A single outgoing reference from one node to another, optionally carrying the predicate name that describes the relationship and a display alias. Every node's links live in one ordered collection regardless of how they render.
- **Patch**: The exchange document `arc apply` consumes, carrying one or more node contributions; each contribution's identity follows the same rules as a standalone node file.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of node files across a representative fixture set (covering every existing node type and every combination of attributes/prose fields/link styles) round-trip (read then write) with no loss of content or connectivity.
- **SC-002**: A second round-trip of any already-round-tripped node file produces zero further changes (stable output).
- **SC-003**: Every one of `arc apply`, `arc lint`, `arc grep`, `arc subgraph`, `arc subgraph --json`, and `arc serve` operates correctly (produces correct, non-corrupted results) against a predicate-first graph, verified across a scenario per command.
- **SC-004**: 100% of old-format graph files presented to any arc command result in a clear error and zero file writes — never a silent misread or corrupted output.
- **SC-005**: A node maintainer can add a new, previously unused named prose field to any node without arc rejecting the file or discarding the field.

## Assumptions

- No conversion/migration tool from the old shape to the new shape is included in this feature (out of scope: CLI-visible flag/command changes); a maintainer who needs to migrate an old graph does so by hand or with a separate tool.
- Deciding whether a link renders flat or grouped at write time (FR-008) uses a minimal, built-in default for well-known predicates sufficient to satisfy round-trip conformance; a fully configurable, schema-declared role/description system for predicates is explicitly out of scope (separate feature, per the user's exclusion) and may later refine this decision without changing the underlying single-list representation.
- A front-matter attribute with exactly one value continues to render on disk as a plain scalar (matching existing graph files' ergonomics), even though it is held internally as a one-element list; an attribute with multiple values renders as a list. This preserves round-trip fidelity against files a maintainer already wrote by hand.
- `"@id"` and `"@type"` remain dedicated, top-level identity declarations and are never duplicated inside the general attribute set.
- Per-predicate merge behavior (how multiple contributions to the same attribute or prose field combine) is unchanged by this feature; only the internal representation and Markdown shape change.
- The set of recognized node types (`source`, `entity`, `resource`, `timeline`, plus graph-registered custom types) is unchanged by this feature.

**Bugfix**: 2026-07-07 — BUG-001 clarified FR-011 and the Edge Cases entry on patch-document identity, and added FR-018: a patch-document node contribution's `"@id"`/`"@type"` are satisfied by its own `"## <ID>"`/`"# <Type>"` section headings (mirroring the pre-existing CORE §12.2 convention that every patch fixture in this repository, and at least one real external tool, already produces), not by requiring both to be redundantly duplicated as explicit yaml-fence keys. This does not weaken FR-002's no-fallback rule for a *standalone* node file, which has no section heading to derive from.

**Bugfix**: 2026-07-20 — BUG-002 added FR-019 and an Edge Cases entry: a `**Label**`-headed body block whose content isn't wikilink-shaped, or whose label isn't yet registered, was being silently dropped by `arc apply` instead of preserved as prose — violating FR-005/FR-006/FR-009's existing verbatim-preservation guarantees. This feature's own plan.md Complexity Tracking table had already named and deferred this exact gap ("exactly the Schema Index question spec 011 owns"); spec 011 built the Schema Index but never wired it into parsing, and spec 013 (predicate-role-rendering) explicitly scoped its own schema-role dispatch to rendering only. FR-019 closes the loop those two specs each deferred.

**Bugfix**: 2026-07-21 — BUG-003 added FR-020/FR-021/FR-022 and four Edge Cases entries: BUG-002's fix stopped dropping a `**Label**`-headed block's content but did not preserve its *formatting* — a text-role block's list items lost their literal wikilink brackets and list shape (running through a paragraph-prose reconstruction heuristic not suited to list items), the block's own `**Label**` never reappeared on write (no `label` attribute was ever auto-registered, and no heading was ever rendered for a `role: text` predicate at all), and distinct labeled edge blocks (e.g. `**Assumes**`, `**Derived From**`, `**Related Aporias**`) collapsed into one undifferentiated flat list (auto-discovery had no signal to register them `role: link` instead of `role: edge`). FR-020/FR-021/FR-022 close these three narrower gaps, completing what FR-019 started.
