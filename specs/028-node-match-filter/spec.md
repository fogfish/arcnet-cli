# Feature Specification: MCP `node_match` Filter Tool

**Feature Branch**: `028-node-match-filter`

**Created**: 2026-08-28

**Status**: Draft

**Input**: User description: "Add `node_match(filter)` function to MCP server. It list of `{id, property, value}` for nodes matching the filter object (see Filtering — MCP filter object). Apparently the result is list of "statements" matching the filter pattern."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find nodes by structured criteria without paying for full content (Priority: P1)

An agent connected to the knowledge graph wants to find every node that satisfies a structured criterion — e.g. "type is `Source` and tag is `cryptography`" — but does not need each matching node's full text, front matter, and edge list. Today the agent must call `node_grep` (which requires a text pattern) or fetch each candidate node individually via `node_get` just to confirm it matches. The agent needs a tool that takes only the structured filter already used elsewhere in the graph's Filtering model and returns a compact list of the facts that justify each match.

**Why this priority**: This is the feature's entire reason for existing — a lightweight, filter-only lookup that complements `node_grep` (text pattern required) and `node_get` (single node, full content) with a third option: many nodes, minimal content, structured criteria only.

**Independent Test**: Can be fully tested by calling `node_match` with a filter containing one attribute statement (e.g. type equals `Source`) against a graph with a known mix of matching and non-matching nodes, and confirming the response lists exactly the matching nodes' ids together with the property and value that satisfied the statement.

**Acceptance Scenarios**:

1. **Given** a graph containing nodes of several types, **When** an agent calls `node_match` with a filter statement constraining `type` to `Source`, **Then** the result lists one entry per `Source` node, each entry shaped `{id, property, value}` with `property` equal to `type` and `value` equal to `Source`.
2. **Given** a graph where no node satisfies the supplied filter, **When** an agent calls `node_match`, **Then** the result is an empty list, not an error.

---

### User Story 2 - See exactly which fact caused a multi-condition match (Priority: P2)

An agent has built a filter combining several statements (for example, a type constraint and a tag constraint) and wants to understand, for each node that passed, which specific facts on that node satisfied which conditions — useful when refining a filter or explaining a result to a user.

**Why this priority**: Delivers the tool's diagnostic value beyond simple existence-checking; it is what distinguishes `node_match` from a plain boolean "does this filter match" check, but the feature is still useful (User Story 1) without it.

**Independent Test**: Can be fully tested by calling `node_match` with a filter combining two statements against a node known to satisfy both via two different facts (e.g. its type and one of its tags), and confirming the result includes a separate entry for each satisfying fact rather than collapsing them into one.

**Acceptance Scenarios**:

1. **Given** a node whose type satisfies one filter statement and whose tag array satisfies a second filter statement, **When** an agent calls `node_match` with a filter combining both statements, **Then** the result includes one entry for the type fact and one entry for the tag fact, both carrying the same node id.
2. **Given** a node whose array-valued attribute (e.g. `tags`) contains two elements that each independently satisfy the same filter statement, **When** an agent calls `node_match`, **Then** the result includes a separate entry for each matching element.

---

### User Story 3 - Discover nodes by relation without knowing the source (Priority: P3)

An agent wants to find every node that participates in a named relation as either the source or the target — for example, every fact recorded with predicate `cites` — without already knowing which nodes to inspect via `node_edges`/`node_backlinks`.

**Why this priority**: A useful, additive convenience once the core lookup (User Story 1) exists, exercising the same filter object against edge facts instead of attribute facts; not required for the feature to deliver its primary value.

**Independent Test**: Can be fully tested by calling `node_match` with a filter statement constraining only `predicate` to a known relation name, and confirming the result lists an entry for every node carrying an outgoing edge with that predicate, each entry naming the relation and its target.

**Acceptance Scenarios**:

1. **Given** a graph with several `cites` edges among its nodes, **When** an agent calls `node_match` with a filter statement naming `predicate: cites` only, **Then** the result includes one entry per citing node, with `property` equal to `cites` and `value` equal to the cited node's id.

---

### Edge Cases

- What happens when the supplied filter has no statements (an empty or absent filter)? Since `node_match`'s output is only meaningful as evidence for *why* a node matched, and every node vacuously "matches" an empty filter, the tool treats a missing or empty statement list as invalid input and reports a validation error rather than enumerating every fact of every node in the graph.
- How does the tool handle a filter statement whose predicate/target combination matches zero facts anywhere in the graph? It returns an empty result — this is not an error.
- How does the tool handle a node that satisfies every statement in the filter but where two different statements are independently satisfied by the exact same fact (same property and value)? That fact is reported once, not once per satisfied statement — the result lists distinct facts, not distinct statement satisfactions.
- How does the tool represent a match against a node's own type (the synthesized type fact) versus a match against a front-matter attribute versus a match against an outgoing relation? All three are reported through the same `{id, property, value}` shape, with `property` equal to `type`, the attribute name, or the relation name respectively.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The MCP server MUST expose a `node_match` tool that accepts one required `filter` argument, shaped identically to the filter object already accepted by `node_grep`, `subgraph_get`, and `context_retrieve` (a list of source/predicate/target statements).
- **FR-002**: The system MUST evaluate the supplied filter against every node's own facts — its type, its front-matter attributes, and its outgoing relations — using the same all-statements-must-be-satisfied rule already defined for graph filtering.
- **FR-003**: For every node that satisfies the filter, the system MUST include in the result one entry per distinct fact on that node that satisfied at least one filter statement; the tool MUST NOT include facts that did not satisfy any statement.
- **FR-004**: Each result entry MUST be shaped `{id, property, value}`, where `id` identifies the matching node, `property` identifies the matched attribute name, relation name, or the literal `type`, and `value` identifies the matched attribute value or relation target.
- **FR-005**: The system MUST reject a call whose filter is missing or contains no statements with a clear validation error, rather than returning results for every node in the graph.
- **FR-006**: A filter statement in `node_match` MUST always narrow which nodes and facts are reported — `node_match` performs no relation traversal, so a statement naming only a relation (predicate) behaves the same as any other narrowing statement rather than scoping a neighbor expansion.
- **FR-007**: When a filter matches zero nodes, the system MUST return an empty result rather than an error.
- **FR-008**: The system MUST NOT include a matching node's unrelated facts (attributes or relations that did not themselves satisfy a statement) in the result — `node_match` reports evidence of the match, not the full node.

### Key Entities

- **Filter**: The existing structured selection criteria (a list of source/predicate/target statements) already used by `node_grep`, `subgraph_get`, and `context_retrieve`; `node_match` reuses this object as its sole input rather than defining a new filter shape.
- **Match entry**: One reported fact justifying a node's inclusion in the result, carrying the matching node's id, the fact's property (attribute name, relation name, or `type`), and the fact's value (attribute value or relation target).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can determine, for any node reported by `node_match`, which specific property or relation caused it to match — without making any further tool call.
- **SC-002**: An agent can distinguish "no nodes matched" from "the request failed" for every call: a filter with zero matches always returns an empty list, and an invalid (empty/missing) filter always returns a validation error, never a silent empty list.
- **SC-003**: `node_match` returns results for a filter without requiring the agent to first fetch each candidate node individually — one call replaces what previously took one call per candidate node.
- **SC-004**: A `node_match` response for a filter matching N nodes with a single satisfying fact each contains exactly N entries — its size scales with the number of matching facts, not with the total size of the matching nodes' content.

## Assumptions

- `node_match` is read-only and included in `arc serve --readonly`'s tool set alongside `node_get`, `node_grep`, `subgraph_get`, and `context_retrieve`.
- The filter object's syntax and matching semantics (statements, wildcards, OR-of-values, regexp patterns, AND across statements) are fully defined by the existing Filtering model and are not redefined by this feature.
- Because `node_match` scans nodes directly rather than traversing from a seed, it has no notion of traversal-vs-narrowing statements — every statement narrows the flat result set, matching how `node_grep` already treats its filter today.
- No result-count limit is imposed by this feature, consistent with the other unlimited MCP listing tools (`node_grep`, `node_edges`, `node_backlinks`); this may be revisited if graph size makes it necessary.
- Result ordering is not user-visible behavior that requires specification here; a stable, deterministic order (e.g. by node id) is expected but not a functional requirement of this feature.
