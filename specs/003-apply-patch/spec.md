# Feature Specification: Apply a Document Patch to the Graph (`arc apply`)

**Feature Branch**: `003-apply-patch`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "`arc apply <patch.md>` — apply a patch file to the graph (CORE §12.3): parse the patch manifest (`kind: patch`, `document`, `published`, `stats`); check idempotency and skip with a clear message if `sources/<id>.md` is already tracked (CORE §11.2); for each H1/H2 node section reconstruct the node object (ARCNET-AST §4); **create** new node files when the basename does not exist; **merge** into existing files per the kind's declared merge operation — `none` for `source`, `union` for `entity`, `union first-writer` for `resource`, and per-profile operation for domain/extension kinds (CORE §10); derive and append timeline entries from the source's `published` date (CORE §9.4); produce exactly one git commit with the mandatory subject, stats, and `Source-Id:` trailer (CORE §11.3); update the local index cache (Phase 4) atomically within the same filesystem transaction"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ingest a brand-new document into the graph (Priority: P1)

A user has a patch file — the full extracted contribution of one document (its source record, the entities and resources it mentions, everything the extraction tool derived) — and wants it added to their graph as a single, traceable unit of history.

**Why this priority**: This is the graph's primary growth mechanism. Every other capability of the tool (query, lint, serve) exists to make sense of content that first entered the graph this way. Without a working, trustworthy ingest path, the graph never grows past its empty starting state.

**Independent Test**: Can be fully tested by applying a patch for a document that shares nothing with the current graph, then inspecting the resulting files and git history — delivers a graph that has grown by exactly one document's worth of content with no other command required.

**Acceptance Scenarios**:

1. **Given** an initialized, empty graph and a well-formed patch for a document not yet in the graph, **When** the user applies the patch, **Then** the tool creates a new file for every node the patch carries (the source and each entity/resource it introduces), each written in the canonical node format.
2. **Given** a patch whose source has a `published` date, **When** the patch is applied, **Then** the tool creates or extends the timeline entry for that date's period (yearly and monthly) with a reference to the newly ingested source, in chronological order among existing entries.
3. **Given** a successful patch application, **When** the user inspects the git history afterward, **Then** exactly one new commit exists, its subject names the ingested document, and it records how many nodes of each kind were added.
4. **Given** a successful patch application, **When** the command finishes, **Then** the tool reports what was created (counts by kind) so the user can confirm the outcome without a separate inspection step.

---

### User Story 2 - Merge a patch's contribution into overlapping graph content (Priority: P2)

A user applies a patch for a second document that discusses subjects (entities, cited works) already present in the graph from a previously ingested document.

**Why this priority**: Documents overlap constantly in a real corpus — the same standard, technology, or person is mentioned across many sources. If every mention created a duplicate node, the graph would fragment into disconnected copies instead of accumulating a connected body of knowledge. Correct merging is what makes repeated ingestion valuable rather than harmful.

**Independent Test**: Can be fully tested by applying two patches that both reference an entity or resource of the same name, and confirming only one node file exists for that subject afterward, carrying the union of both contributions.

**Acceptance Scenarios**:

1. **Given** a graph already containing an entity node, **When** a patch is applied that reintroduces the same entity (same basename) with additional or different relations, **Then** the existing node file is updated in place — its relations become the union of what was already there and what the patch contributed — and no duplicate file is created.
2. **Given** a graph already containing a resource node with an optional field left blank, **When** a patch is applied that supplies a value for that same field on the same resource, **Then** the existing node is updated to carry the newly supplied value, since it was previously empty.
3. **Given** a graph already containing a resource node with a field already set, **When** a patch is applied that would set that same field to a different value, **Then** the existing value is preserved unchanged, since the first value written to that field wins.
4. **Given** a merge into existing content, **When** the application finishes, **Then** the resulting commit's stats reflect that some nodes were merged rather than newly created, so the user can distinguish "grew the graph" from "connected to existing content."

---

### User Story 3 - Apply a patch that introduces domain-specific node kinds (Priority: P3)

A user's graph has opted into a domain profile (a set of node kinds beyond the format's built-in source/entity/resource, each with its own merge rule declared by that profile) and applies a patch whose extraction tool produced nodes of one of those domain-specific kinds.

**Why this priority**: The graph format is explicitly designed to be extended by domain profiles, and a real deployment is expected to use them. Without this, `arc apply` would only ever work for the three built-in kinds, permanently blocking any domain-specific use of the tool.

**Independent Test**: Can be fully tested by registering a domain kind and its merge behavior for a graph, applying a patch that contains a node of that kind, and confirming it is created or merged correctly using the registered behavior.

**Acceptance Scenarios**:

1. **Given** a graph that has registered a domain-specific node kind together with its merge behavior, **When** a patch containing a node of that kind is applied, **Then** the tool creates or merges that node using the registered behavior, exactly as it would for a built-in kind.
2. **Given** a graph with no domain kinds registered, **When** a patch containing a node of an unregistered kind is applied, **Then** the tool refuses the entire application (per FR-018) and reports which kind was not recognized.
3. **Given** a user wants a graph to accept a domain-specific kind, **When** they register that kind together with its merge behavior, **Then** subsequent patch applications recognize and correctly process nodes of that kind.

---

### User Story 4 - Re-running an already-applied patch is a safe no-op (Priority: P4)

A user accidentally applies the same patch file twice — by mistake, because a script re-ran, or because they were unsure whether the first attempt succeeded.

**Why this priority**: Lower priority than the two content-producing paths, but essential as a safety net. Ingestion is the tool's most frequently repeated operation and the one most likely to be re-run absent-mindedly or by an automated pipeline; without idempotency, a careless re-run would corrupt graph history with duplicate commits and inflated node contributions.

**Independent Test**: Can be fully tested by applying the same patch file twice in a row and confirming the second run makes no changes and produces no new commit.

**Acceptance Scenarios**:

1. **Given** a document already tracked in the graph (its source node exists), **When** the user applies a patch for that same document again, **Then** the tool makes no filesystem changes, creates no commit, and reports clearly that the document is already tracked.

---

### Edge Cases

- What happens when the patch file's manifest is missing a mandatory field (`kind: patch`, `document`, or `published`)? The tool must refuse with a clear explanation and make no changes.
- What happens when a patch is not valid for this tool at all (wrong `kind` value, unparsable front-matter, or a body that doesn't follow the H1-kind/H2-node section structure)? The tool must refuse with a clear explanation and make no changes.
- What happens when the target directory is not an initialized graph? The tool must refuse and make no changes, the same way other graph-mutating commands do.
- What happens when a patch node section names a kind the tool has no merge rule for? The tool must refuse that node (and, per the all-or-nothing contribution model, the whole patch) rather than silently guessing a merge strategy or dropping the node.
- What happens when merging a patch's node into an existing node produces conflicting values for a field that is supposed to have one authoritative value? The contribution is not silently discarded and not silently overwritten; see FR-013 for the resolution behavior.
- What happens when applying is interrupted partway through (process killed, disk full, permission error)? The graph and its git history must be left exactly as they were before the attempt — no partially written node files, no dangling commit.
- What happens when the patch references a node (via a link) that the patch itself does not define and that doesn't already exist in the graph? The link is recorded as written; resolving it is not this command's responsibility (consistent with the graph format allowing forward/dangling references).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a single patch file path as input and parse its manifest, validating that the mandatory fields (`kind: patch`, `document`, `published`) are present and well-formed before making any change.
- **FR-002**: The tool MUST refuse to apply, and MUST make no filesystem or git changes, when the patch manifest is missing a mandatory field or the patch body does not follow the expected node-section structure.
- **FR-003**: Before making any change, the tool MUST check whether the patch's document is already tracked in the graph and, if so, MUST skip the entire operation, make no filesystem or git changes, and report clearly that the document is already tracked.
- **FR-004**: For every node section the patch carries, the tool MUST reconstruct the full node — its kind, identity, scalar attributes, and body content — from the patch's declared structure.
- **FR-005**: When a reconstructed node's identity does not already exist in the graph, the tool MUST create a new node file for it, written in the graph's canonical node format.
- **FR-006**: When a reconstructed node's identity already exists in the graph, the tool MUST merge the contribution into the existing node file using the merge behavior declared for that node's kind, rather than creating a duplicate or overwriting the file wholesale.
- **FR-007**: For a node kind whose merge behavior is "none," a second contribution to an already-existing node of that kind MUST be treated as a no-op for that node — its file is left unchanged.
- **FR-008**: For a node kind whose merge behavior is "union," the tool MUST combine the existing node's relations and multi-valued fields with the patch's contribution (duplicates collapsed, nothing dropped from either side).
- **FR-009**: For a node kind whose merge behavior is "union, first-writer," the tool MUST additionally allow the patch to fill in a previously empty optional field on the existing node, while leaving any already-populated field at its existing value.
- **FR-010**: The tool MUST derive the timeline period(s) — yearly and monthly — implied by the patch document's `published` date and MUST append an entry for the newly ingested document to each, in chronological order among that period's existing entries, creating the period's timeline file if it does not yet exist.
- **FR-011**: On success, the tool MUST produce exactly one git commit containing every file the application touched — new node files, merged node files, and updated timeline entries — leaving nothing uncommitted and nothing committed separately.
- **FR-012**: The commit MUST use the mandatory subject line format identifying the operation and the ingested document, MUST include a stats line reporting node counts by kind, and MUST include a trailer recording the document's identity.
- **FR-013**: When merging produces conflicting values for a field that is supposed to carry one authoritative value, the tool MUST preserve the first-written value as authoritative, MUST record the conflicting later value alongside it in a form that keeps the file human-readable and clearly marks the value as needing review, and MUST still complete the commit rather than aborting the whole application.
- **FR-014**: The tool MUST refuse to apply, and MUST make no filesystem or git changes, when the target directory is not an initialized graph.
- **FR-015**: The tool MUST leave no partially-created or partially-modified graph state, and no dangling commit, when application fails or is interrupted at any point before the commit completes.
- **FR-016**: On success, the tool MUST report to the user what happened — counts of nodes created versus merged, by kind, and a reference to the resulting commit — without requiring a separate inspection command to confirm the outcome.
- **FR-017**: The tool MUST preserve every attribute and relation on an existing node that the patch's contribution does not itself concern, whether or not the tool recognizes that attribute or relation.
- **FR-018**: The tool MUST refuse to apply, and MUST make no filesystem or git changes, when a node section in the patch declares a kind that is neither one of the format's built-in kinds (source/entity/resource) nor a domain/extension kind registered for that graph (FR-019); the refusal MUST name the unrecognized kind.
- **FR-019**: The tool MUST provide a way for a graph to register a domain/extension node kind together with its declared merge behavior (one of the format's fixed merge behaviors — none / union / union-first-writer / other declared equivalent), so that later patch applications recognize and correctly process nodes of that kind. Registration MUST itself be refused, leaving no partial state, if it would register the same kind twice with conflicting merge behavior.
- **FR-020**: When applying a patch node whose kind is a registered domain/extension kind, the tool MUST create or merge it using that kind's registered merge behavior, with the same creation/merge/conflict-flagging guarantees (FR-005 through FR-013) that apply to the format's built-in kinds.

### Out of Scope

- **Local index maintenance**: This feature does not build or update any local navigation index as a side effect of applying a patch. The committed graph content (files + git history) is the sole source of truth this feature produces or depends on; keeping a local index consistent with that content is separate, later work.
- **Conflict resolution**: This feature's responsibility ends at flagging an unresolved merge conflict in a human-readable form (FR-013). Listing previously flagged conflicts across the graph, or resolving one by choosing a side or accepting a hand-edited value, is separate, later work.

### Key Entities

- **Patch**: A single Markdown file that serializes one document's entire contribution to the graph — a manifest (document identity, publication date) plus one section per node it carries. Never itself stored in the graph; consumed once and discarded.
- **Node Contribution**: One patch section representing one node's worth of content — its kind, identity, attributes, and body — as reconstructed by the tool before it is created or merged.
- **Source Node**: The record of the ingested document itself; the provenance root that other nodes in the same patch derive from.
- **Entity / Resource Node**: A subject or external work mentioned or cited by the document; may already exist in the graph from a prior ingestion, in which case this patch's mention is merged into it rather than duplicating it.
- **Timeline Entry**: A chronological reference to the newly ingested source, filed under the yearly and monthly period implied by its publication date.
- **Merge Behavior**: The per-kind rule (none / union / union-first-writer / other) governing how a node contribution reconciles with an already-existing node of the same identity.
- **Ingest Commit**: The single git commit produced by a successful application, carrying every file the operation touched plus the mandatory subject, stats, and document-identity trailer.
- **Kind Registration**: A graph-level declaration that a domain/extension node kind is active for this graph, together with the merge behavior patches carrying that kind must be processed with.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can go from "have a patch file" to "content committed to the graph, with a clear summary of what changed" in a single command invocation, completing in under 5 seconds for a typical single-document patch.
- **SC-002**: 100% of successful applications result in exactly one new commit, with no untracked or uncommitted files remaining afterward.
- **SC-003**: 100% of the time a patch mentions a subject already present in the graph under the same identity, the graph ends up with exactly one node for that subject, not a duplicate.
- **SC-004**: 100% of attempts to re-apply an already-tracked document result in zero graph or history changes.
- **SC-005**: A user can confirm what an application did (created vs. merged, by kind) from the command's own output alone, without needing to run a follow-up inspection command.
- **SC-006**: 100% of failed or interrupted applications leave the graph and its git history exactly as they were beforehand — zero observed cases of partial writes or dangling commits.
- **SC-007**: A patch that introduces a registered domain-specific node kind is applied using its declared merge behavior with the same correctness guarantees (SC-002–SC-004) as the format's built-in kinds.

## Assumptions

- The graph the patch is applied to already exists and was created by the graph's initialization command; this feature does not initialize a graph.
- The patch file itself is trusted input produced by a compatible extraction tool; this feature validates its required structure (FR-001, FR-002) but does not attempt to authenticate its origin.
- Applying a patch is fully local and offline — no network access is required or attempted.
- This feature covers applying exactly one patch file per invocation; batch application of a directory of patches, retraction, and reapplication are separate, later capabilities and are out of scope here.
- A graph may register any number of domain/extension kinds; each registration declares exactly one of the format's fixed merge behaviors for that kind (no new merge-behavior kinds beyond none / union / union-first-writer / other declared equivalent).
