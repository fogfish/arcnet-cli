# Feature Specification: MCP `node_links` and `node_backlinks` Tools

**Feature Branch**: `030-mcp-node-links-backlinks`

**Created**: 2026-08-29

**Status**: Draft

**Input**: User description: "Add two new tools to MCP server: `node_links(id)` return outgoing hrefs/edges/links as `[{predicate, target}]` from the target node. `node_backlinks(id)` return incoming hrefs/edges/links as `[{source, predicate}]` from the backlink index."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See what a node points to without fetching its full content (Priority: P1)

An agent connected to the knowledge graph knows a node's id and wants to enumerate every outgoing relation from that node — both its structural relation list and any references embedded inline in its prose text — along with the predicate connecting to each, without fetching the node's full content (front-matter attributes, the prose text itself, everything else `node_get` returns). Today the agent must call `node_get` and manually pick the relations out of the response, discarding the rest. `node_links` gives the agent this combined outgoing relation list directly.

**Why this priority**: This is the primary requested capability and the foundation for graph-traversal questions such as "what does this node reference?"

**Independent Test**: Can be fully tested by calling `node_links` with the id of a node that has a known set of outgoing relations, and confirming the response lists exactly one `{predicate, target}` entry per relation and nothing else.

**Acceptance Scenarios**:

1. **Given** a node with several outgoing relations to different targets under different predicates, **When** an agent calls `node_links` with that node's id, **Then** the result lists one entry per relation, each shaped `{predicate, target}`, matching the node's own relations exactly.
2. **Given** a node with zero outgoing relations, **When** an agent calls `node_links` with that node's id, **Then** the result is an empty list, not an error.

---

### User Story 2 - Discover what references a known node (Priority: P1)

An agent knows a node's id and needs to find every other node that references it — whether through that other node's structural relation list or a reference embedded inline in its prose text — for example, to gauge how central a node is, find everything that cites it, or check whether a node it is about to remove is still referenced elsewhere. No existing tool answers this today: `node_get` only exposes a node's own outgoing relations, and finding references to it otherwise requires inspecting every node in the graph by hand. `node_backlinks` gives the agent this reverse lookup directly.

**Why this priority**: This delivers the feature's most novel value — information not obtainable today through any other exposed tool short of enumerating the whole graph — and directly answers "what references this node?"

**Independent Test**: Can be fully tested by calling `node_backlinks` with the id of a node that several other nodes are known to reference, and confirming the response lists exactly one `{source, predicate}` entry per referencing relation.

**Acceptance Scenarios**:

1. **Given** several nodes in the graph hold relations targeting a common node, **When** an agent calls `node_backlinks` with that common node's id, **Then** the result lists one entry per incoming relation, each shaped `{source, predicate}`.
2. **Given** a node that no other node in the graph references, **When** an agent calls `node_backlinks` with that node's id, **Then** the result is an empty list, not an error.

---

### User Story 3 - Audit a node's full relation footprint before changing it (Priority: P2)

Before deleting, renaming, or substantially editing a node, an agent (or the person directing it) wants to see both directions of its relations at once — what it points to and what points to it — to understand the impact of the change. With both tools available, the agent builds this picture with two direct calls instead of scanning the graph.

**Why this priority**: This value only exists once both P1 tools are available; it adds no capability of its own beyond calling both.

**Independent Test**: Can be fully tested by calling `node_links` and `node_backlinks` for the same node id and confirming that, together, they account for every relation touching that node — nothing is missing from either list and nothing appears in the wrong one.

**Acceptance Scenarios**:

1. **Given** a node with both outgoing and incoming relations, **When** an agent calls `node_links` and then `node_backlinks` for its id, **Then** every relation where the node is the source appears only in the `node_links` result and every relation where the node is the target appears only in the `node_backlinks` result.

---

### Edge Cases

- What happens when `id` does not match any node in the graph? Both tools report a clear not-found error, consistent with how `node_get` already handles an unknown id — they do not silently return an empty list for a nonexistent node.
- What happens when a valid node has zero relations in the requested direction? The tool returns an empty list, not an error — this is different from an unknown id.
- What happens when a node holds a relation whose target is itself (a self-reference)? It is reported like any other relation: it appears in that node's own `node_links` result (as `{predicate, target: id}`) and in that node's own `node_backlinks` result (as `{source: id, predicate}`).
- What happens when two nodes are connected by more than one relation (the same predicate repeated, or several different predicates between the same pair)? Each occurrence is reported as its own entry; entries are not deduplicated or merged.
- What happens with a reference embedded inline in a node's prose text, as opposed to a relation in its structural relation list? Both are reported the same way, through the same `{predicate, target}` / `{source, predicate}` shape — the tools do not distinguish where a relation came from (see Assumptions).
- What happens when an inline prose reference has no explicit predicate (a bare reference with no relation type stated)? It is still reported as one entry, with `predicate` empty, rather than being dropped.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The MCP server MUST expose a `node_links` tool that accepts one required `id` argument identifying an existing node by basename, using the same id convention already used by `node_get`.
- **FR-002**: For a valid `id`, `node_links` MUST return one entry per outgoing relation on that node — including both its structural relation list and any reference embedded inline in its prose text — shaped `{predicate, target}`.
- **FR-003**: The MCP server MUST expose a `node_backlinks` tool that accepts one required `id` argument identifying an existing node by basename, using the same id convention.
- **FR-004**: For a valid `id`, `node_backlinks` MUST return one entry per relation elsewhere in the graph whose target is that node — including both structural relations and inline prose references on the referencing node — shaped `{source, predicate}`.
- **FR-005**: Both tools MUST reject a call whose `id` matches no node in the graph with a clear not-found error, consistent with `node_get`'s existing behavior for the same condition.
- **FR-006**: Both tools MUST return an empty list, not an error, when `id` is valid but the node has zero relations in the requested direction.
- **FR-007**: Both tools MUST report every matching relation occurrence exactly once per occurrence — including repeated predicate/target pairs and self-referencing relations — without deduplicating or merging occurrences.
- **FR-008**: `node_links` and `node_backlinks` MUST report relations from both sources — a node's structural relation list and references embedded inline in its prose text — through the same entry shape, without indicating which source a given entry came from; every entry `node_links` reports for a node MUST have a matching entry in that target's `node_backlinks` result, and vice versa, regardless of source.
- **FR-009**: Neither tool's response MUST include the node's other content — its type, attributes, or prose text — restricting each response to the relation list only.

### Key Entities

- **Outgoing relation entry**: One reported relation from the queried node to another — from its structural relation list or from a reference embedded inline in its prose text — shaped `{predicate, target}`; returned by `node_links`.
- **Incoming relation entry**: One reported relation from another node onto the queried node — from that node's structural relation list or an inline prose reference — shaped `{source, predicate}`; returned by `node_backlinks`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can list every outgoing relation from a node, or every relation referencing it, in a single tool call, without fetching that node's or any other node's full content.
- **SC-002**: An agent can always distinguish "no relations exist in this direction" from "the request failed": a valid id with zero matching relations returns an empty list every time, and an id naming no node returns an explicit error every time.
- **SC-003**: For any relation in the graph, the source node's `node_links` result and the target node's `node_backlinks` result agree with each other — the relation appears in exactly one entry of each, with the same predicate.
- **SC-004**: Determining what references a given node no longer requires inspecting every node in the graph — one `node_backlinks` call replaces what previously required checking every candidate node's own relations by hand.

## Assumptions

- Both tools are read-only and included in `arc serve`'s fixed tool set alongside `node_get`, `node_match`, and `node_grep`; `arc serve` exposes only read-only tools today, so no separate read-only mode or flag is involved.
- "Outgoing hrefs/edges/links" in the feature request is taken literally: both a node's structural relation list and references embedded inline in its prose text count as relations for `node_links` and `node_backlinks`, reported through the same entry shape without distinguishing their origin. This is a deliberate widening of scope beyond how the graph treats these two kinds of reference everywhere else today (where inline prose references are informational only and are not treated as navigable, indexed relations) — introduced specifically so `node_links` and `node_backlinks` give a complete picture of what a node references and is referenced by, including references only ever written inline.
- Because inline prose references are not part of the graph's existing incoming-relation index today, `node_backlinks` requires that index (or its equivalent) to be extended to also cover inline references, not just the structural relation list it covers today. This is a dependency of this feature, not something it can assume already exists.
- No result-count limit is imposed by this feature, consistent with `node_grep`, `node_match`, and the graph's other unlimited MCP listing tools.
- Result ordering is not user-visible behavior that requires specification here; a stable, deterministic order is expected but not mandated by this feature.
