# Feature Specification: Extract a Self-Contained Subgraph (`arc subgraph`)

**Feature Branch**: `007-arc-subgraph`

**Created**: 2026-07-04

**Status**: Draft

**Input**: User description: "`arc subgraph <basename> [--depth <n>] [<filter>]` — extract a self-contained subgraph: the seed node plus all nodes reachable within N hops (default 1), optionally filtered by kind or attributes on the reached nodes; the filter applies to the expanded nodes, not the seed; output uses the patch exchange format (CORE §12.2) as the serialization: nodes are grouped by kind under `# <Kind>` headings, each node under `## <basename>`, front-matter in a fenced YAML block, body verbatim below — human-readable, LLM-friendly, and round-trippable back into `arc apply`."

## Clarifications

### Session 2026-07-04

- Q: With bidirectional traversal, a heavily-referenced node could pull in a very large "self-contained" subgraph even at depth 1 — should there be a size safeguard? → A: Two independent, configurable soft caps: a direct (outgoing-reachable) cap defaulting to 4096, and a backlink (incoming-reachable) cap defaulting to 1024. Neither refuses the run; when a cap is exceeded, the most-connected candidates (by edge count) are kept up to the cap and the rest are dropped.

## Bugfix Log

- **Bugfix**: 2026-07-05 — BUG-001 Added FR-017/SC-008/the Stub Node key entity and a new User Story 1 acceptance scenario + edge case for an opt-in `--stubs` flag. Root cause: an included node's structural links to a target excluded from the extraction boundary (by `--depth`, a truncation cap, or the filter) were rendered verbatim with no corresponding node section for that target, so applying the extracted document into a graph that does not already contain the target (e.g. an empty graph) produced a dangling reference. `--stubs` emits a minimal (kind + id only) placeholder node section for every such boundary target, never itself traversed further.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pull a node and its immediate context into a portable document (Priority: P1)

A user (or an LLM agent acting on their behalf) wants to hand a single node's full context to a language model, or to a teammate, without manually opening and copy-pasting a handful of related files. They name the node by its basename and get back one self-contained document: the node itself plus everything directly connected to it, grouped and formatted so it reads naturally and can be fed straight back into the graph later.

**Why this priority**: This is the entire reason the command exists — turning "everything relevant to node X" from a manual, error-prone file hunt into a single command. Every other capability (depth, filtering) refines this base extraction.

**Independent Test**: Can be fully tested by creating a graph where a seed node links to a known, fixed set of other nodes, running the extraction with no extra flags, and confirming the output contains exactly the seed plus its directly connected nodes, correctly grouped by kind, with valid re-parsable front-matter and unmodified body content.

**Acceptance Scenarios**:

1. **Given** a graph where a seed node has direct connections to several other nodes of different kinds, **When** the user extracts the subgraph for that seed with default settings, **Then** the output contains the seed node and every directly connected node once each, grouped under a heading per kind, each node's front-matter and body preserved verbatim.
2. **Given** a seed node with no connections to any other node, **When** the user extracts its subgraph, **Then** the output contains only the seed node, formatted the same way, with no error.
3. **Given** the extracted output, **When** it is fed back into the graph's own ingestion command, **Then** it is accepted as a valid document and produces no unexpected structural errors.
4. **Given** a basename that does not correspond to any node in the graph, **When** the user requests its subgraph, **Then** the tool reports a clear error and produces no output.
5. **Given** `--stubs` is passed and an included node references a target that exists in the source graph but falls outside the extraction boundary (excluded by `--depth`, a truncation cap, or the filter), **When** the user extracts the subgraph, **Then** the output includes a minimal stub section for that target — kind and id only, no other attributes, empty body — so the document contains no dangling reference even when applied into a graph that does not already have that target. *(added by BUG-001)*

---

### User Story 2 - Widen or narrow the reach of the extraction (Priority: P2)

A user wants more or less surrounding context than the immediate neighborhood — for example, two or three hops out to capture an entity's broader neighborhood, or just the seed itself with no expansion at all. They control this with a depth flag.

**Why this priority**: Depth control is a natural refinement once the base one-hop extraction works — it adjusts how much context is pulled in, but the extraction, grouping, and output format are unchanged from User Story 1.

**Independent Test**: Can be fully tested by creating a graph with a chain of connected nodes several hops deep, running the extraction at different depth values, and confirming the set of included nodes grows to match exactly what is reachable within that many hops, no more and no less.

**Acceptance Scenarios**:

1. **Given** a chain of nodes each connected to the next, **When** the user extracts the subgraph at a depth of 2, **Then** the output includes every node reachable within 2 hops of the seed, and excludes nodes only reachable at a greater distance.
2. **Given** the same chain, **When** the user extracts the subgraph at a depth of 0, **Then** the output contains only the seed node.
3. **Given** no depth flag is supplied, **When** the user extracts the subgraph, **Then** the tool behaves as if depth 1 was requested.
4. **Given** a node reachable from the seed by more than one path of different lengths, **When** the user extracts the subgraph, **Then** that node appears exactly once in the output, included as soon as it falls within the requested depth.

---

### User Story 3 - Keep the extraction focused with a filter (Priority: P3)

A user only wants certain kinds of surrounding context — for example, only the `source` nodes that back up an entity, or only nodes carrying a particular tag — without the seed itself being excluded just because it doesn't match that criterion.

**Why this priority**: Filtering trims an already-correct extraction down to what's useful for a specific purpose; it depends on Users Stories 1 and 2 already working and reuses filter syntax already established elsewhere in the CLI.

**Independent Test**: Can be fully tested by creating a graph where the seed's reachable nodes span multiple kinds, applying a kind filter, and confirming the output always includes the seed regardless of its own kind, plus only the reachable nodes matching the filter.

**Acceptance Scenarios**:

1. **Given** a seed node whose own kind does not match a requested `--kind` filter, **When** the user extracts the subgraph with that filter, **Then** the seed node is still included in the output.
2. **Given** reachable nodes of several kinds, **When** the user applies a kind filter, **Then** only reachable nodes of the matching kind(s) appear in the output alongside the seed.
3. **Given** a filter that matches none of the reachable nodes, **When** the user extracts the subgraph, **Then** the output contains only the seed node, with no error.
4. **Given** a filter combining kind, tag, and attribute conditions, **When** the user extracts the subgraph, **Then** only reachable nodes satisfying all combined conditions are included, consistent with the graph's general filter composition rules (see Filtering).

---

### Edge Cases

- What happens when `<basename>` does not exist in the graph? The tool must report a clear error and produce no output.
- What happens when `--depth` is given a negative value or a non-integer? The tool must report a clear usage error and produce no output.
- What happens when the seed node participates in a cycle (e.g. A links to B, B links back to A)? Each node must appear exactly once in the output regardless of how many distinct paths lead back to it.
- What happens when a reachable node has a structural link whose target file does not exist (a dangling link)? That target is excluded from the subgraph — it cannot be serialized without a node to read.
- What happens when the filter excludes every reachable node? The output still contains the seed node alone; this is not an error.
- What happens when the graph is very large and/or the requested depth is high, producing a large fan-out? The tool must still complete and include every node within the requested depth in a single run.
- What happens when the target directory is not an initialized graph? The tool must refuse to run and report this clearly, consistent with other graph commands.
- What happens when two nodes of different kinds would need to be serialized? Each kind gets its own heading, and a node is listed under its own kind's heading exactly once.
- What happens when the number of nodes reachable via outgoing structural connections, or via incoming ones, exceeds its configured cap? The tool does not fail: it keeps the most-connected candidates (ranked by each candidate's own structural edge count, highest first) up to that direction's cap, drops the remainder, and indicates in its output that the set was truncated.
- *(added by BUG-001)* What happens when an included node references a target that exists in the source graph but was excluded from this particular extraction (by `--depth`, a truncation cap, or the filter)? Without `--stubs`, behavior is unchanged: the reference is rendered verbatim as part of the including node's own content and the target itself is not serialized. With `--stubs`, the tool additionally emits a minimal stub node (kind and id only, no other attributes, empty body) for that target, so every link in the output resolves to a real node section even when the extracted document is applied into a graph that does not already contain it; a stub is never itself expanded — its own structural connections are not traversed, regardless of `--depth`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a required `<basename>` argument identifying the seed node, an optional `--depth <n>` flag (default `1`), and an optional filter argument (see Filtering).
- **FR-002**: The tool MUST always include the seed node in the output, regardless of whether the seed matches the given filter.
- **FR-003**: The tool MUST compute the set of nodes reachable from the seed within `<n>` hops by traversing structural connections in both directions — a node's own outgoing edges/links, and any other node's outgoing edge/link that targets it — so the neighborhood captures both what the seed (or an intermediate node) points to and what points back to it; inline prose references are excluded from this traversal, consistent with the AST's existing distinction between structural edges and non-navigable inline references.
- **FR-004**: The tool MUST treat hop distance as the shortest number of edge traversals from the seed to a given node, MUST include each reachable node exactly once even when multiple paths of different lengths reach it, and MUST NOT loop indefinitely when the reachable set contains a cycle.
- **FR-005**: When a filter is given, the tool MUST restrict the reachable (non-seed) nodes included in the output to those matching the filter, using the same filter syntax and composition rules (kind, tag, attribute, AND/OR semantics) defined for other graph-wide commands (see Filtering); the filter MUST NOT be applied to the seed node.
- **FR-006**: When a structural link target does not correspond to an existing node, the tool MUST exclude that target from the subgraph rather than failing the whole extraction.
- **FR-007**: The tool MUST serialize its output in the patch exchange format: nodes grouped under a heading per kind, each node under its own heading naming its basename, with the node's front-matter in a fenced YAML block followed by its body content unmodified.
- **FR-008**: The tool MUST emit a document-level manifest alongside the per-node sections — a synthetic document identifier derived from the seed's basename and the extraction's timestamp as the published date — so the output is structurally complete and applies via the graph's own patch-ingestion command without a structural parsing failure, unmodified or after editing.
- **FR-009**: The tool MUST make no changes to the graph's files or git history — extraction is a read-only, non-mutating operation.
- **FR-010**: The tool MUST refuse to run, and MUST report this clearly instead of extracting, when the target directory is not an initialized graph.
- **FR-011**: When `<basename>` does not identify any existing node, the tool MUST report a clear error and produce no output.
- **FR-012**: When `--depth` is given a value that is not a non-negative integer, the tool MUST report a clear usage error and produce no output.
- **FR-013**: A `--depth` of `0` MUST produce output containing only the seed node.
- **FR-014**: The tool MUST apply two independent, configurable soft caps to the reachable set, evaluated before the filter: a "direct" cap bounding the count of nodes reached via outgoing structural connections (default `4096`), and a "backlink" cap bounding the count of nodes reached via incoming structural connections (default `1024`).
- **FR-015**: When the direct-reachable or backlink-reachable node count exceeds its respective cap, the tool MUST retain the most-connected candidates — ranked by each candidate node's own total structural edge count, highest first — up to that cap, MUST discard the remaining candidates for that direction, and MUST NOT fail or refuse to run; the output MUST indicate that truncation occurred.
- **FR-016**: Both caps MUST be independently configurable and MUST fall back to their stated defaults when not overridden.
- **FR-017** *(added by BUG-001)*: The tool MUST support an opt-in `--stubs` flag, off by default. When enabled, for every structural link target that exists in the source graph's index but was not itself selected for inclusion in the extraction, the tool MUST emit a minimal node section for it — kind and id only, no other attributes, empty body (no text, notes, edges, or links) — so that every structural link present in the output resolves to a real node section. A stub node's own structural connections MUST NOT be traversed (no recursive stub expansion), regardless of `--depth`. This is distinct from FR-006: FR-006's target does not exist anywhere in the source graph (nothing to serialize); FR-017's target exists but was excluded by this extraction's own boundary (`--depth`, a cap, or the filter). Without `--stubs`, output is unchanged from FR-006/FR-007/FR-014/FR-015's existing behavior.

### Key Entities

- **Seed Node**: The node named by `<basename>`, always present in the output, never excluded by the filter.
- **Reachable Node**: Any node other than the seed found within `<n>` hops of the seed by following structural edges/links, in either direction; subject to the optional filter and to its direction's traversal cap.
- **Subgraph**: The seed node plus the set of reachable nodes selected for a given extraction, serialized as one patch-exchange document grouped by kind.
- **Filter**: The optional, composable node-selection criteria (kind, tag, attribute) that restricts which reachable nodes are included, shared with other graph-wide commands (see Filtering).
- **Traversal Cap**: A configurable ceiling on how many nodes are retained per traversal direction before filtering — direct (outgoing, default 4096) and backlink (incoming, default 1024) — each independent of the other; when exceeded, the highest-degree candidates are kept.
- **Stub Node** *(added by BUG-001, opt-in via `--stubs`)*: A minimal placeholder node section — kind and id only — emitted for a structural link target that exists in the source graph but was excluded from this extraction's boundary; carries no attributes, text, notes, or connections of its own, and is never itself traversed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can produce a ready-to-share or ready-to-inject document covering a node and its context in a single command run, with no manual file collection.
- **SC-002**: 100% of nodes actually within the requested hop distance are included in a test graph's extraction, with no reachable node missed and no node duplicated.
- **SC-003**: Applying a filter reduces the reachable nodes in the output to exactly the filtered subset, with the seed always present, verified with 100% accuracy against a graph containing both matching and non-matching reachable nodes.
- **SC-004**: A subgraph extraction against a graph of several thousand nodes completes in under 10 seconds.
- **SC-005**: Extracted output is accepted without structural error when passed to the graph's own ingestion command, verified across a range of test graphs with varying shapes.
- **SC-006**: Running the extraction command never modifies any graph file or git history — verified by the graph's state being byte-for-byte identical before and after any run.
- **SC-007**: When a direction's reachable set exceeds its configured cap, the retained nodes are always exactly the highest-degree candidates up to that cap, verified with 100% accuracy against a test graph with a known degree distribution.
- **SC-008** *(added by BUG-001)*: With `--stubs` enabled, output applied into a freshly initialized, otherwise empty graph produces zero unresolved-link violations when the target graph is subsequently checked by the graph's own conformance validation, verified across a range of test graphs with varying shapes.

## Assumptions

- The filter reuses the same kind/tag/attribute syntax and AND/OR composition rules already established for other graph-wide commands (see Filtering), consistent with `arc grep`.
- Structural connections for hop-counting are a node's Edges and Link blocks only, traversed in both directions (a node's own outgoing connections, and any other node's connection that targets it); inline prose references are not navigable connections, consistent with the existing AST invariant that inline references never constitute graph edges.
- Nodes within the output are ordered deterministically (by kind, then alphabetically by basename within each kind) so repeated runs against an unchanged graph produce byte-identical output.
- A node's front-matter and body are serialized exactly as they already are for a single on-disk node (front-matter fields, then body content), matching the same per-node rendering used elsewhere in the codebase, not re-derived or reformatted.
- Exit status follows this codebase's established convention: a run that completes and prints its result exits successfully, and only a refusal to run at all (missing seed, invalid depth, uninitialized graph) is reported as a distinct error before any output is produced.
- The direct and backlink traversal caps are configurable per graph rather than hardcoded, consistent with how this CLI already exposes other per-graph tunables; a graph that does not override them uses the stated defaults (4096 direct / 1024 backlink).
- *(added by BUG-001)* Referential integrity across the extraction boundary is opt-in (`--stubs`), not the default: the default output continues to render boundary-crossing links verbatim with no corresponding node section, matching FR-006's existing treatment of the narrower already-absent-anywhere case. A user who intends to apply extracted output into a different or empty graph is expected to pass `--stubs`.
