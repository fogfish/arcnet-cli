# Feature Specification: Node Provenance Timestamps (`published`/`indexed`/`updated`)

**Feature Branch**: `009-node-timestamp-attrs`

**Created**: 2026-07-05

**Status**: Draft

**Input**: User description: "encode timestamp attribute for graph nodes. The patch document carries on the timestamp `published`. This timestamp has to propogate to each newly created node (except stub on) in the graph. Then, it adds a new attribute for each newly created node `indexed` with ISO8601 timestamp at seconds resolution. The `indexed` timestamp is identical for all nodes in the patch. In node has been merged then `updated` with ISO8601 timestamp at seconds resolution. Both `indexed` and `updated` carries on identical timestamp for the single patch document. All node in the graph carries on `published` and `indexed`. All node been merged carries on `updated`. The `published` attribute is exported out."

## Clarifications

### Session 2026-07-05

- Q: Should the `_schema/nodes`/`_schema/predicates` documents that a patch application auto-creates when it discovers a previously-unseen node kind or predicate receive `published`/`indexed` like ordinary content nodes? → A: Exempt, like stub nodes — a schema document never receives `published`, `indexed`, or `updated`; it stays pure tool bookkeeping.
- Q: Does a "none"-merge-behavior kind's already-established no-op (a second contribution to an existing node of that kind leaves the file byte-for-byte unchanged) count as "merged" for the purpose of stamping `updated`? → A: No — the existing no-op guarantee wins; `updated` is never added when nothing about the file actually changes.
- Q: What does "the `published` attribute is exported out" mean for this feature's scope? → A: `published` is guaranteed to survive, unchanged, whenever a node's attributes are serialized elsewhere (e.g. `arc subgraph` extraction) — in contrast to `indexed`/`updated`, which are this feature's own apply-time bookkeeping and carry no such guarantee.
- Q: For a merge behavior other than "none" (union, union-first-writer, or a registered "append"/"validated-overwrite" kind), if the merge runs but nets out identical to the existing file — e.g. a union merge where every incoming relation or value was already present — does the node still get stamped with `updated`? → A: No — `updated` is stamped only when the resulting node file content actually differs, byte-for-byte, from what it was before the merge; a merge that changes nothing is left untouched exactly like a "none"-behavior no-op.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Know when a node's source was published and when it entered the graph (Priority: P1)

A user applies a patch that contributes new content to the graph. Every ordinary content node the patch creates — the source record itself, and every entity or resource the document introduces for the first time — now carries the document's own publication date and the moment it was written into the graph, without the user having to cross-reference the patch file or the commit history to answer either question.

**Why this priority**: Provenance is the whole point of the graph's growth mechanism (`arc apply`, spec 003). Without a per-node record of "when was this published" and "when did this enter the graph," answering either question requires digging through git history or the original patch file — information the graph itself should carry once a node exists.

**Independent Test**: Can be fully tested by applying a patch for a document with a known `published` date into a graph, then inspecting every newly created node file and confirming each carries a `published` value matching the patch and an `indexed` value at the moment of that application.

**Acceptance Scenarios**:

1. **Given** a patch whose manifest declares a `published` date, **When** the patch creates a new node for an ordinary content kind (source, entity, resource, or a registered domain/extension kind), **Then** that node's front matter carries a `published` attribute equal to the patch's declared date.
2. **Given** a successful patch application that creates one or more new nodes, **When** the user inspects those nodes afterward, **Then** every one of them carries an `indexed` attribute — an ISO 8601 timestamp at second resolution — and every node created by that same application carries the identical `indexed` value.
3. **Given** a patch section that is a minimal stub (kind and id only, no other content — the placeholder shape an `arc subgraph --stubs` extraction produces), **When** that section causes a new node file to be created, **Then** the resulting node carries neither `published` nor `indexed`.
4. **Given** a patch application that auto-registers a previously-unseen node kind or predicate as a `_schema/` document, **When** that schema document is created, **Then** it carries neither `published` nor `indexed`, the same as a stub node.

---

### User Story 2 - Know when a node was last touched by an incoming contribution (Priority: P2)

A user applies a second patch that mentions a subject already present in the graph. The existing node's file is updated in place with the union of both contributions, and now also records when that update happened, so a reader can tell the node isn't frozen at its original creation but has continued to absorb new contributions over time.

**Why this priority**: Merging (spec 003, User Story 2) is how the graph accumulates a connected body of knowledge instead of duplicating content. Recording when a merge last touched a node completes the provenance picture User Story 1 establishes for creation — without it, an updated node looks indistinguishable from one that has sat untouched since it was first written.

**Independent Test**: Can be fully tested by applying a patch that merges into an already-existing node, then confirming that node's file carries an `updated` attribute at the moment of that application, using the same timestamp value as every other node the same application touched.

**Acceptance Scenarios**:

1. **Given** a graph already containing a node, **When** a patch is applied whose contribution actually changes that node's file content (per its kind's declared merge behavior), **Then** the node's front matter carries an `updated` attribute — an ISO 8601 timestamp at second resolution — and it is identical to the `indexed` value given to any node the same application newly created.
2. **Given** a node kind whose merge behavior is "none," **When** a patch re-contributes to an already-existing node of that kind, **Then** the existing no-op guarantee holds exactly as before this feature: the file is left completely unchanged, and no `updated` attribute is added.
3. **Given** a node that was originally created as a stub (kind and id only, no `published` set), **When** a later patch merges real content into it for the first time, **Then** the merge fills in `published` the same way it fills in any other previously-absent attribute under that kind's merge rules, and the node also receives `updated` for this application — but it still never receives an `indexed` value, since that attribute is only ever assigned at a node's initial (non-stub, non-schema) creation, and this node's creation was a stub, not that.
4. **Given** a node kind whose merge behavior is "union" (or another non-"none" behavior), **When** a patch re-contributes to an already-existing node of that kind with relations and values that are all already present, **Then** the resulting file content is unchanged from before the merge, and no `updated` attribute is added — the same treatment as a "none"-behavior no-op, even though the kind's declared merge behavior is not itself "none."

---

### User Story 3 - Read a node's provenance directly from the file, with no separate command (Priority: P3)

A person browsing the graph's content opens any node file and can immediately see, in its own front matter, when the underlying document was published, when the node itself entered the graph, and — if it has ever been merged into — when it was last touched, all without running a separate inspection command or cross-referencing git history.

**Why this priority**: This is the payoff of Stories 1 and 2 — a self-describing graph where provenance lives in the content itself. It depends on those two stories already producing well-formed timestamp attributes to read.

**Independent Test**: Can be fully tested by browsing any node created or merged by a completed patch application and confirming its provenance timestamps are present, correctly valued, and readable as plain text.

**Acceptance Scenarios**:

1. **Given** a node created by a patch application, **When** a person opens that node's file, **Then** they can read its `published` and `indexed` values directly from the front matter with no other tool.
2. **Given** a node that has since been merged into by a later patch, **When** a person opens that node's file, **Then** they can additionally read its `updated` value, distinguishing it from a node that has never been merged.
3. **Given** a node carrying a `published` attribute, **When** that node's content is extracted or serialized by another command (e.g. `arc subgraph`), **Then** the exported node still carries the same `published` value, unchanged.

---

### Edge Cases

- What happens to a node's `published` value when two different patches, applied at different times, both contribute to the same node identity? The first patch to populate `published` on that node is authoritative; a later patch's own `published` date does not overwrite it, consistent with the existing first-writer-wins rule already governing other scalar attributes on a merged node.
- What happens when a stub node (created with neither `published` nor `indexed`) is re-contributed with a kind whose merge behavior is "none"? The existing no-op guarantee applies — the file remains unchanged, and the node continues to carry neither `published` nor `indexed` nor `updated` until a merge behavior that actually writes to it eventually does.
- What happens to a node's `indexed` value across later merges into that same node? `indexed` is set exactly once, at creation, and is never modified by any later merge — only `updated` reflects the timestamp of a later touch.
- What happens when a single patch application both creates some nodes and merges into others? Every node it creates gets the same `indexed` value, and every node whose file content is actually changed by a merge (per the byte-for-byte rule above) gets the same `updated` value — and both values are identical to each other, since both describe the same single application.
- What happens when a "union" (or other non-"none") merge runs for a node but contributes nothing the node doesn't already have — e.g. a patch re-mentions an entity with relations that are all already present? The resulting file is byte-for-byte identical to before the merge, so no `updated` attribute is added, the same as a "none"-behavior no-op; only a merge that actually changes the file's content is stamped.
- What happens to the format's own timeline period files (`timeline/yearly/`, `timeline/monthly/`)? They are not reconstructed through the ordinary node create/merge path this feature governs (spec 003's FR-025/D8b) and are unaffected by this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When applying a patch creates a new node file for an ordinary content kind (source, entity, resource, or a registered domain/extension kind), the tool MUST set that node's `published` attribute to the value of the patch manifest's own `published` date.
- **FR-002**: The tool MUST NOT set `published`, `indexed`, or `updated` on a node created from a stub patch section (kind and id only, no other attributes, per the minimal placeholder shape an `arc subgraph --stubs` extraction produces).
- **FR-003**: The tool MUST NOT set `published`, `indexed`, or `updated` on a `_schema/nodes` or `_schema/predicates` document created or extended by patch application's kind/predicate auto-registration.
- **FR-004**: For every node newly created during a single patch application (excluding FR-002/FR-003's exemptions), the tool MUST set an `indexed` attribute to an ISO 8601 timestamp at second resolution.
- **FR-005**: The `indexed` value MUST be identical across every node newly created by one patch application — a single point in time captured once for that application, not computed separately per node.
- **FR-006**: Once set at a node's creation, `indexed` MUST NOT be modified by any later merge into that node.
- **FR-007**: When a patch's contribution to an already-existing node results in that node's file content actually differing, byte-for-byte, from what it was before the merge, the tool MUST set that node's `updated` attribute to an ISO 8601 timestamp at second resolution.
- **FR-008**: The tool MUST NOT set `updated` on a node whose merge produces no byte-for-byte change to the file — whether because the kind's merge behavior is "none" (spec 003 FR-007's existing no-op guarantee) or because a non-"none" merge (e.g. union) nets out identical to the file's prior content.
- **FR-009**: The `updated` value MUST be identical to the `indexed` value used within that same patch application — the two attributes share one timestamp captured once per application, whichever nodes they are applied to.
- **FR-010**: When a merge fills in a previously-empty `published` attribute on an existing node (e.g. a node originally created as a stub), the tool MUST do so using that node kind's ordinary merge rules for a previously-empty scalar attribute — the same rules already governing every other such attribute — rather than a special case unique to `published`.
- **FR-011**: A node's `published` attribute MUST be preserved, unchanged, whenever that node's content is exported or serialized by another capability (e.g. `arc subgraph` extraction).

### Key Entities

- **Provenance Timestamp Attributes**: The three front-matter attributes this feature introduces or governs — `published` (the source document's declared publication date, propagated to the nodes it produces), `indexed` (the moment a node first entered the graph), and `updated` (the moment a node was last changed by a merge).
- **Newly Created Node**: An ordinary content node (source, entity, resource, or registered domain/extension kind) written to the graph for the first time during a patch application; excludes stub nodes and `_schema/` documents.
- **Merged Node**: An already-existing node whose file content actually differs, byte-for-byte, from before a patch's contribution was combined into it per that kind's declared merge behavior; excludes any merge — "none"-behavior or otherwise — that nets out to no change.
- **Application Timestamp**: The single point-in-time value captured once per patch application, shared by every newly created node's `indexed` attribute and every merged node's `updated` attribute in that same application.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of nodes newly created by a successful patch application (excluding stub nodes and `_schema/` documents) carry both `published` and `indexed` immediately afterward.
- **SC-002**: 100% of nodes whose file content is actually changed, byte-for-byte, by a merge during a patch application carry `updated`, using the exact same timestamp value as every other node — created or merged — touched by that same application.
- **SC-003**: 100% of stub nodes and `_schema/` documents created during a patch application carry none of `published`, `indexed`, or `updated`.
- **SC-004**: 100% of nodes re-contributed by a later patch with no new content (whether the kind's merge behavior is "none" or the merge simply nets out identical) remain byte-for-byte unchanged, with zero instances of an `updated` attribute being added.
- **SC-005**: A person can determine a node's publication date, graph-entry time, and (if applicable) last-updated time by reading that single node file, with zero additional commands needed.
- **SC-006**: 100% of nodes carrying a `published` attribute retain that exact value when their content is exported or serialized by another capability.

## Assumptions

- This feature only governs node kinds already reconstructed through the ordinary patch create/merge path (spec 003); the format's own timeline period files, which use their own specialized derivation and rendering (spec 003 FR-025), are out of scope.
- `published` keeps the same date format the patch manifest already declares it in (spec 003's `document`-level `published` field); only `indexed` and `updated` are newly required to be full ISO 8601 timestamps at second resolution.
- `indexed` and `updated` are this feature's own apply-time bookkeeping attributes; no requirement of this feature depends on whether other commands surface or omit them — only `published`'s durability across exports (FR-011) is a requirement here.
- A "registered domain/extension kind" follows the same rules as the format's built-in source/entity/resource kinds for the purposes of this feature; no kind-specific exception beyond the stub/schema exemptions already stated is introduced.
