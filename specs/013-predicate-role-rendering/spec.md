# Feature Specification: Schema-Driven Link Rendering

**Feature Branch**: `013-predicate-role-rendering`

**Created**: 2026-07-08

**Status**: Draft

**Input**: User description: "Change how arc decides whether a node's outgoing links are written to Markdown as a flat bulleted list or grouped under a \"## Heading\" block, so that decision comes from each predicate's own declared schema (its role: edge = flat, link = grouped) rather than from whatever shape happened to be used in whichever file arc last read. ... A round-trip test (read a node, write it back unchanged) must still produce byte-stable output; cosmetic reordering of which heading-group appears before another, or normalization of an inconsistently-shaped input into the canonical schema-driven shape, is acceptable and expected... Out of scope: changing parse-time behavior (already unified into one Edges list by spec 010); changing merge behavior (spec 012)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consistent shape for a schema-declared predicate everywhere it appears (Priority: P1)

A graph maintainer has declared a predicate's schema (e.g. `broader`) with role `edge`, and another predicate (e.g. `mentions`) with role `link`. When arc writes any node in the graph that carries either predicate, `broader` always renders as a flat bullet and `mentions` always renders grouped under its own heading — regardless of which node, which file, or how that particular occurrence happened to be written previously.

**Why this priority**: This is the core behavior change requested: rendering shape becomes a property of the predicate's schema, not an artifact of file history. Without this, the feature delivers nothing.

**Independent Test**: Author two nodes by hand — one where a `link`-role predicate is written as a flat bullet list, one where an `edge`-role predicate is written grouped under a heading. Run arc's write/normalize path on both and confirm each now renders in its schema-declared shape (grouped for the `link`-role predicate, flat for the `edge`-role predicate), regardless of the original shape.

**Acceptance Scenarios**:

1. **Given** a predicate schema declares role `link` for `mentions`, **When** arc writes a node ~~whose Edges include one or more `mentions` occurrences~~ **to a standalone graph node file (`RenderNode`) whose Edges include one or more `mentions` occurrences**, **Then** those occurrences render under a `## Mentions`-style heading block, not as flat bullets mixed with other predicates. *(Scope clarified — BUG-001: this scenario governs `RenderNode`'s graph-node-file output only; see Acceptance Scenario 4 below for the distinct patch-document rule.)*
2. **Given** a predicate schema declares role `edge` for `broader`, **When** arc writes a node whose Edges include one or more `broader` occurrences, **Then** those occurrences render as flat bullets, never grouped under a heading. This holds identically for both `RenderNode` and `RenderPatch` — `edge`-role rendering does not differ by format.
3. **Given** a node was previously written with a `link`-role predicate as a flat bullet (e.g. hand-edited or produced by an older arc version), **When** arc reads and re-writes that node **to a standalone graph node file**, **Then** the output groups that predicate's occurrences under its heading — the on-disk shape is corrected to match the schema, not preserved.
4. **Given** the same `mentions` predicate occurrences, **When** arc writes them as part of a patch-exchange document (`RenderPatch`), **Then** those occurrences render under a bold-label paragraph (`**Mentions**`), never a Markdown heading — ARCNET-CORE §14.2 reserves `##` exclusively for a patch's `@type`/`@id` structure ("H1 = node's `@type`", "H2 = node's `@id`"), so a node's own body content, including `link`-role predicate groups, MUST NOT introduce a competing heading at any level. *(Added — BUG-001.)*

**Bugfix**: 2026-07-09 — BUG-001 Clarified that Acceptance Scenarios 1 and 3 govern `RenderNode` (standalone
graph node files, ARCNET-CORE §5) only; added Acceptance Scenario 4 for `RenderPatch` (patch-exchange
documents, ARCNET-CORE §14.2), which reserves Markdown headings for type/identity structure and requires
`link`-role predicate groups to render under a bold label instead. See amended FR-001/FR-004 and new FR-014
below.

---

### User Story 2 - Heading omitted when a type's body is a single link-role predicate (Priority: P2)

A graph maintainer works with a `timeline` node, whose entire body is just the `entries` predicate (role `link`). When arc writes this node, the `entries` list appears as the node's body content directly — no redundant `## Entries` heading is added, since the heading would name the only content the reader already knows is there.

**Why this priority**: Named explicitly in the request as a required behavior; affects an existing node type (`timeline`) that graph maintainers already interact with, so a naïve implementation of User Story 1 alone would visibly clutter every timeline node with a heading that adds no information.

**Independent Test**: Write a timeline node whose only body content is one or more `entries` occurrences. Confirm arc's output has no heading before the entries list. Then write a node whose body has that same link-role predicate plus at least one other predicate occupying the body (a second link-role predicate, or any edge-role predicate), and confirm the heading reappears — the omission applies only when exactly one link-role predicate constitutes the entire body.

**Acceptance Scenarios**:

1. **Given** a node's entire body consists of exactly one link-role predicate's occurrences (all sharing that one predicate), **When** arc writes the node, **Then** no heading precedes the occurrence list — the list itself is the whole body.
2. **Given** a node's body contains that same link-role predicate's occurrences alongside at least one other predicate's occurrences (edge-role bullets or a second link-role group), **When** arc writes the node, **Then** the single-predicate heading is no longer omitted — every predicate group is properly headed/labeled per its role.

---

### User Story 3 - Round-trip stability for already-canonical documents (Priority: P1)

A graph maintainer runs arc's read-then-write path on a node that already matches the schema-driven canonical shape (correct grouping, correct heading labels, correct flat bullets). The output is byte-identical to the input — arc does not churn a document that is already correct, which matters for diffs, git history, and trust in the tool.

**Why this priority**: Without this guarantee, every `arc` invocation that touches a node produces spurious diffs across an entire graph, making the schema-driven rendering change itself indistinguishable from unrelated noise in version control. This is the acceptance bar the request calls out explicitly.

**Independent Test**: Take a node already written in canonical schema-driven shape, run it through arc's read/write path, and diff the output against the original — the diff must be empty.

**Acceptance Scenarios**:

1. **Given** a node file already in canonical schema-driven shape (correct flat/grouped rendering per each predicate's role, correct heading labels, single-predicate heading omitted where applicable), **When** arc reads and writes that node back unchanged in content, **Then** the resulting bytes are identical to the original file.
2. **Given** a node file whose predicate occurrences are already grouped correctly but the relative order of two heading-groups (or the order of edge bullets versus link groups) differs from a canonical ordering rule, **When** arc reads and writes the node, **Then** the reordering it introduces is limited to the ordering permitted for edges/link-groups and does not otherwise alter content, labels, or grouping decisions.

### Edge Cases

- What happens when a node carries occurrences of a predicate that has no registered schema (no role declared)? The predicate has no declared role to derive a shape from; arc MUST fall back to a documented default shape (see Assumptions) rather than failing to render the node.
- What happens when a `link`-role predicate's schema declares no explicit heading label? The heading text falls back to the predicate's own name in a human-readable form (this already exists as the predicate schema's `label` field with a documented default), consistent with today's schema model.
- What happens when a node's body has zero occurrences of a given predicate? No heading or bullet is rendered for that predicate at all — an empty group is never emitted.
- What happens when a node's body has exactly one link-role predicate group, but that predicate also happens to be the only predicate the node's type permits at all (not just the only one present)? Still governed by the "exactly one link-role predicate present" rule at render time — a type permitting more predicates that simply weren't used in this instance is treated the same as a type that only ever allows the one predicate.
- What happens when two or more distinct link-role predicates each have exactly one occurrence, alongside no edge-role predicates? The single-predicate-body omission does not apply — there are two distinct predicate groups, so each keeps its own heading.
- What happens when the same node is written as part of a patch-exchange document instead of a standalone graph node file? The single-predicate-body omission (FR-006/FR-007) and the flat-vs-grouped role decision (FR-001) still apply identically — but a patch document's `##` heading level is reserved for its own `@type`/`@id` structure (ARCNET-CORE §14.2), so `link`-role predicate groups render under a bold-label paragraph there instead of a heading (FR-014), never introducing a second, competing use of `##` inside a node's body. *(Added — BUG-001.)*

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When writing a node's outgoing structural links ~~to Markdown~~ **to a standalone graph node file (`RenderNode`, ARCNET-CORE §5)**, the system MUST determine, for each predicate present, whether its occurrences render as a flat bulleted list or as a heading-grouped block, based solely on that predicate's own declared schema role (`edge` → flat, `link` → grouped). *(Scope clarified — BUG-001: this requirement governs `RenderNode` only; `RenderPatch` follows FR-014 instead.)*
- **FR-002**: The rendering decision for a given predicate MUST be identical across every node in the graph that carries an occurrence of that predicate — the same predicate never renders flat in one node and grouped in another because of how either node's source file happened to be shaped previously. This applies within each format independently: a predicate's `RenderNode` shape is consistent across every graph node file, and its `RenderPatch` shape (FR-014) is consistent across every patch document, even though the two formats' shapes for a `link`-role predicate differ from each other by design (heading vs. bold label).
- **FR-003**: The system MUST NOT derive the flat-vs-grouped rendering decision from the shape found when the node was last read (no shape-preservation from parse time); rendering is a pure function of the predicate's current schema role at write time.
- **FR-004**: For a `link`-role predicate rendered as a grouped block **in a standalone graph node file (`RenderNode`)**, the system MUST render a heading using that predicate's declared display label (falling back to the predicate's own name, capitalized, when no label is declared), followed by that predicate's occurrences. *(Scope clarified — BUG-001: patch-document rendering is governed by FR-014, which uses a bold label, not a heading, for the identical label text.)*
- **FR-005**: For an `edge`-role predicate, the system MUST render each occurrence as a flat bullet, with no enclosing heading, interleaved with other edge-role predicates' bullets per existing flat-list conventions.
- **FR-006**: When a node's entire body content consists of the occurrences of exactly one `link`-role predicate (no other predicate's occurrences present in the body), the system MUST omit that predicate's heading and render its occurrences directly as the body content.
- **FR-007**: When a node's body contains occurrences of more than one predicate (any mix of edge-role bullets and/or link-role groups), the system MUST render every link-role predicate's heading — the omission in FR-006 applies only to the single-predicate-body case.
- **FR-008**: Reading a node already in canonical schema-driven shape and immediately writing it back MUST produce byte-identical output to the original.
- **FR-009**: Reading a node whose predicate occurrences are shaped inconsistently with their predicates' declared schema roles, and writing it back, MUST normalize that node into the canonical schema-driven shape (correcting flat-vs-grouped rendering, heading presence/label, and single-predicate omission) rather than preserving the original inconsistent shape.
- **FR-010**: The system MAY reorder the relative position of heading-grouped blocks, and MAY reorder edge-role bullets relative to link-role groups, when normalizing a node to canonical shape, as long as no predicate occurrence's content, predicate association, or target is altered.
- **FR-011**: This change MUST NOT alter how a node is parsed from Markdown into its in-memory Edges representation — the existing unified parse-time behavior (spec 010) is unaffected; only the write/render path changes. *(Scoped to this feature's own diff — still accurate as written. It does NOT mean parsing stays permanently schema-blind: spec 010's own plan.md explicitly deferred giving parsing the same schema-role awareness this feature gave rendering, and spec 010/BUG-002 (2026-07-20) closed that deferred gap in a later feature, not as part of this one. See spec 010 FR-019.)*
- **FR-012**: This change MUST NOT alter merge-policy behavior (spec 012) — how contributions to a predicate combine during a graph update is independent of how the resulting occurrences are later rendered.
- **FR-013**: When a predicate present in a node's body has no schema registered for it (no declared role), the system MUST render its occurrences using the documented default shape (see Assumptions) rather than failing or dropping the occurrences.
- **FR-014**: When writing a node's outgoing structural links as part of a patch-exchange document (`RenderPatch`, ARCNET-CORE §14.2), the system MUST NOT introduce any Markdown heading inside a node's body — headings are reserved exclusively for the patch's own `@type`/`@id` structure (H1/H2). `edge`-role predicates render as flat bullets exactly as under FR-005; `link`-role predicates render grouped, using the identical role/label resolution as FR-001/FR-004 (including the FR-006/FR-007 single-predicate-body omission), but under a bold-label paragraph (`**Label**`) instead of a `## Label` heading. *(Added — BUG-001.)*

### Key Entities

- **Predicate Schema**: The existing, already-persisted definition of a predicate (role, merge policy, optional label, optional aligned vocabulary term, description). This feature reads its `role` (`edge`/`link`) and `label` fields to drive rendering; it does not add new fields to this entity.
- **Node Body**: The rendered Markdown content of a node other than its front matter — the ordered arrangement of prose, flat edge bullets, and link-role predicate groups that this feature's rendering decision governs. The group's own presentation differs by serialization target (BUG-001): a heading-grouped block (`## Label`) in a standalone graph node file (`RenderNode`, FR-001/FR-004), a bold-label paragraph (`**Label**`) in a patch-exchange document (`RenderPatch`, FR-014) — the underlying partition/label/omission decisions (FR-001/FR-004/FR-006/FR-007) are otherwise identical between the two.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For 100% of predicates carrying a declared schema role, every rendered occurrence of that predicate across the entire graph uses the same flat-or-grouped shape, with zero exceptions attributable to source-file history.
- **SC-002**: Re-running arc's write path on a graph already in canonical schema-driven shape produces zero byte-level differences across 100% of node files.
- **SC-003**: A graph maintainer can change a predicate's declared role once, in its schema, and see every existing occurrence of that predicate across the whole graph adopt the new rendering shape the next time each affected node is written — with no per-node manual edits required.
- **SC-004**: Timeline-style nodes (single link-role predicate body) render with zero superfluous headings, verified across all existing timeline nodes in a representative graph.

## Assumptions

- A predicate with no registered schema (no declared role) is rare in a well-formed graph (schema registration is already established practice per specs 010/011); this feature's default-shape fallback (FR-013) treats such a predicate as `edge`-role (flat bullet) — the currently-implemented default — since that is the least surprising choice when no grouping intent has been declared.
- "Canonical schema-driven shape" refers only to flat-vs-grouped rendering, heading presence, and heading label text; it does not impose a specific ordering requirement among edge-role bullets or among link-role groups beyond what FR-010 already permits as acceptable normalization.
- The existing predicate schema's `label` field (already documented as "the human-readable title shown as a link-role predicate's heading, defaulting to the predicate name, capitalized") is the sole source of heading text for this feature; no new schema field is introduced.
- This feature applies uniformly to every node type (`source`, `entity`, `resource`, `timeline`, and any graph-registered custom type) — there is no per-type opt-out from schema-driven rendering.
- "Body" for the purposes of FR-006/FR-007's single-predicate-omission rule means the rendered link/edge content area of the node, independent of any surrounding prose text predicates, which render separately and do not count toward the "exactly one predicate" check.

**Bugfix**: 2026-07-20 — 010/BUG-002 annotated FR-011: `arc apply` was silently dropping a `**Label**`-headed body block's content whenever it wasn't wikilink-shaped, since parsing never consulted a predicate's declared role the way this feature's own `RenderNode`/`RenderPatch` already do. FR-011 remains accurate about this feature's own diff (it didn't touch parsing), but the underlying render/parse asymmetry it left in place is now closed by spec 010's own FR-019, in spec 010's tasks.md Phase 6 — not by reopening this feature.

**Bugfix**: 2026-07-21 — 010/BUG-003 cross-reference (no FR text change needed here — this feature's FR-001/FR-004/FR-014 label/grouping rendering machinery is correct and unaffected): spec 010's parse-time auto-discovery path (BUG-002, spec 010 FR-019) was found to never populate a discovered predicate's `label` attribute (Assumptions line 110 above), and to always register auto-discovered edge-shaped content as `edge`-role even when it occurred under its own `**Label**` block — so this feature's own FR-001/FR-004/FR-014 rendering never got the inputs (a populated `label`, a `link` role) it needs to reproduce a `**Label**`-derived block's original heading and grouping. Closed by spec 010's new FR-021/FR-022, in spec 010's tasks.md Phase 7 — not by reopening this feature.
