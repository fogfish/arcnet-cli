# Feature Specification: Context Retrieve Tool (`context_retrieve`)

**Feature Branch**: `025-context-retrieve-tool`

**Created**: 2026-08-24

**Status**: Draft

**Input**: User description: "`context_retrieve(query, filter?, limit?)` → the primary RAG tool: runs the same three-pass retrieval as `arc context` — grep match, attribute match, neighbor expansion — and returns the top-N node objects with full content (attrs + text + edges + links); designed to let an agent build its working context in a single tool call without iterating through grep results and fetching each node separately; `limit` defaults to 10"

## Clarifications

### Session 2026-08-24

- Q: Should `context_retrieve` require VISION.md's (entirely unbuilt) Phase 4 index as a prerequisite, or scan the graph directly on every call like the already-shipped `node_grep`/`subgraph_get` tools do? → A: Direct on-disk scan on every call, no index dependency — same posture as `node_grep`/`subgraph_get`.
- Q: How should the top-`limit` result be ranked when content match, attribute match, and neighbor expansion surface overlapping or competing candidates? → A: Direct matches (content or attribute) rank above neighbor-only matches; within a tier, rank by hop-distance and connectivity.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Assemble working context from a topic in one call (Priority: P1)

An LLM agent connected to the graph as an MCP client has a topic or question in mind and wants the full content of every node relevant to it — not just a list of ids or matching lines, but the complete node objects it can reason over immediately — without first calling a search tool and then fetching each candidate one at a time.

**Why this priority**: This is the entire reason the tool exists — turning "search, then fetch each result" (multiple round trips) into "search and fetch" (one round trip). Every other capability in this feature (filtering, limiting) refines this base retrieval; without it, an agent gets no more value than it already has from `node_grep` plus repeated `node_get` calls.

**Independent Test**: Can be fully tested by seeding a graph with nodes whose content or attributes reference a known topic, plus at least one node connected to a match only through an edge, calling `context_retrieve` with a query naming that topic and no `filter`/`limit`, and confirming the returned array contains full node objects (attrs, text, edges, links) for the directly matching nodes and for the edge-connected neighbor, with no node appearing more than once.

**Acceptance Scenarios**:

1. **Given** a graph where some nodes' content or front-matter attributes reference a topic, **When** an agent calls `context_retrieve` with a query naming that topic and no `filter` or `limit`, **Then** the tool returns an array of full node objects (attrs, text, edges, links) for the nodes relevant to the query, up to the default limit of 10.
2. **Given** a node that does not itself match the query but is directly connected (by edge or link) to a node that does, **When** an agent calls `context_retrieve` with that query, **Then** the connected node is included in the result alongside the direct match, as a full node object.
3. **Given** the same query and no change to the graph between calls, **When** an agent calls `context_retrieve` twice in a row, **Then** both calls return the same set of nodes in the same order.
4. **Given** a query that matches no node's content, no node's attributes, and reaches no neighbor through expansion, **When** an agent calls `context_retrieve`, **Then** the tool returns an empty array, not an error.

---

### User Story 2 - Narrow retrieval to a relevant slice of the graph (Priority: P2)

An agent already knows it only cares about a certain kind of node, or nodes carrying a certain tag or attribute — for example, only `source` nodes, or only nodes tagged `cryptography` — and wants the same one-call retrieval restricted to that slice, instead of getting relevant-but-out-of-scope nodes mixed into the result.

**Why this priority**: Filtering makes the tool usable on graphs where an unrestricted query would surface too broad a mix of node kinds; it depends on the base retrieval in Story 1 but is independently testable and independently valuable once that base exists.

**Independent Test**: Can be fully tested by seeding a graph where a query matches nodes of more than one kind, calling `context_retrieve` with that query plus a filter restricting to one kind, and confirming only nodes of that kind appear in the result — including nodes that would otherwise have been pulled in only through neighbor expansion.

**Acceptance Scenarios**:

1. **Given** a graph where a query matches nodes across multiple kinds/tags, **When** an agent calls `context_retrieve` with that query and a `filter` object (kind/tags/attrs/attrPatterns), **Then** only nodes satisfying the filter appear in the result.
2. **Given** a node reachable only through neighbor expansion from a direct match, **When** that node does not satisfy the supplied `filter`, **Then** it is excluded from the result entirely, not just excluded as a "direct match".
3. **Given** a `filter` that excludes every node the query would otherwise have surfaced, **When** an agent calls `context_retrieve`, **Then** the tool returns an empty array, not an error.

---

### User Story 3 - Control how much context comes back (Priority: P3)

An agent needs to bound how many full node objects it pulls into its context window at once — sometimes a handful is enough, sometimes it wants more — so it sets an explicit result count instead of relying on the default.

**Why this priority**: This refines the volume of an already-working retrieval; it matters once Story 1 already returns useful results, and lets an agent tune the tool to its own context-window budget.

**Independent Test**: Can be fully tested by seeding a graph where a query matches more nodes than the default limit, calling `context_retrieve` with different explicit `limit` values, and confirming the returned array size never exceeds the requested limit while still containing the most relevant matches.

**Acceptance Scenarios**:

1. **Given** a query matching more candidate nodes than the default, **When** an agent calls `context_retrieve` with no `limit`, **Then** the tool returns at most 10 node objects.
2. **Given** the same query, **When** an agent calls `context_retrieve` with an explicit `limit` of N, **Then** the tool returns at most N node objects.
3. **Given** a `limit` larger than the number of candidate nodes found, **When** an agent calls `context_retrieve`, **Then** the tool returns every candidate found, with no padding and no error.
4. **Given** a `limit` that is zero, negative, or not an integer, **When** an agent calls `context_retrieve`, **Then** the tool reports a clear error and returns no results.

---

### Edge Cases

- What happens when the query string contains characters that would be invalid as a regular expression (parentheses, brackets, unescaped special characters typical of natural-language questions)? The tool must still run successfully, treating the query as literal text rather than failing with a pattern-syntax error.
- What happens when the same node is reachable through more than one retrieval pass (e.g., it matches the query's content directly and is also a neighbor of another match)? It must appear exactly once in the result.
- What happens when the query, filter, and graph together yield zero candidates? The tool returns an empty array, not an error.
- What happens when a directly-matching node has a very large neighborhood (many edges)? Neighbor expansion for that node is bounded the same way `subgraph_get`/`arc subgraph` bounds it, so a single call always completes.
- What happens when `filter` is present but syntactically malformed? The tool reports a clear error and returns no results, consistent with the other filterable tools.
- What happens when the graph's files change on disk between two calls? The second call reflects the graph's current on-disk state; no restart is required.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `arc serve` MCP server MUST expose a `context_retrieve(query, filter?, limit?)` tool that returns an array of full node objects (attrs, text, edges, links) relevant to `query`.
- **FR-002**: `context_retrieve` MUST assemble its result from three retrieval passes over the candidate set: (a) a grep-style match of `query` against node content, (b) a match of `query` against front-matter attribute values, and (c) neighbor expansion — nodes directly connected by edge or link to any node found by pass (a) or (b).
- **FR-003**: `query` MUST be matched as literal, case-insensitive text in passes (a) and (b), not compiled as a caller-supplied regular expression, so a natural-language query never fails with a pattern-syntax error.
- **FR-004**: When `filter` is supplied, it MUST restrict every pass — content matching, attribute matching, and neighbor expansion targets — to nodes satisfying the filter (kind/tags/attrs/attrPatterns, see Filtering); a node excluded by the filter MUST NOT appear in the result even if reachable through neighbor expansion from an included node.
- **FR-005**: The result MUST be deduplicated so that each node id appears at most once, regardless of how many passes or paths surfaced it.
- **FR-006**: `limit` MUST default to 10 when omitted; the tool MUST return at most `limit` node objects, returning fewer without padding or error when fewer candidates exist.
- **FR-007**: `context_retrieve` MUST report a clear error and return no results when `limit` is supplied but is not a positive integer.
- **FR-008**: `context_retrieve` MUST return an empty array, not an error, when no candidate is found by any pass or when `filter` excludes every candidate.
- **FR-009**: Neighbor expansion MUST apply the same reachable-set size safeguards (direct/backlink caps) as `subgraph_get`/`arc subgraph`, so a single call always completes rather than growing unboundedly against a large or highly-connected graph.
- **FR-010**: `context_retrieve` MUST NOT ever modify the graph's files or git history — it is strictly read-only.
- **FR-011**: Every call MUST reflect the graph's current on-disk state at the time of the call; the server MUST NOT require a restart to observe changes made by other processes since it started.
- **FR-012**: A per-call failure (invalid `limit`, malformed `filter`) MUST NOT terminate the server process; the server MUST remain available to serve subsequent calls.
- **FR-013**: Every `context_retrieve` call MUST emit one line to stderr recording the tool name, its key arguments (query, whether a filter was supplied, limit), and whether it succeeded or failed, consistent with the logging already produced by the server's other tools.

### Key Entities

- **Retrieval Candidate**: A node surfaced by one or more of the three passes (content match, attribute match, neighbor expansion) before deduplication, ranking, and truncation to `limit`.
- **Node Object**: The unit returned per result entry — id (basename), kind, attributes, prose text, structural edges, and predicate-grouped links; identical in shape to the object returned by `node_get`.
- **Query**: The free-text string an agent supplies describing what it needs context on; matched literally against node content and attribute values.
- **Filter**: The optional, composable node-selection object (kind/tags/attrs/attrPatterns) narrowing the candidate universe for all three passes, matching the MCP filter object schema shared across this codebase's tools (see Filtering).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can obtain the full content of every node relevant to a topic in a single tool call, with no follow-up search-then-fetch round trips required to reach the same working context.
- **SC-002**: A call with no `limit` never returns more than 10 node objects.
- **SC-003**: Retrieval against a graph of several thousand nodes completes in under 10 seconds — matching `subgraph_get`'s own performance bar (FR-002's neighbor-expansion pass is the most expensive of the three and reuses that tool's traversal), since no index accelerates this feature in this increment.
- **SC-004**: Running any number of `context_retrieve` calls never modifies any graph file or git history, verified by the graph's state being byte-for-byte identical before and after a session.
- **SC-005**: Across a session of mixed valid and invalid calls (bad `limit`, malformed `filter`), the server never crashes and never requires a restart to keep serving valid calls.
- **SC-006**: Repeated calls with the same query, filter, and limit against an unchanged graph return the same set of nodes in the same order every time.

## Assumptions

- `context_retrieve` is added to the existing `arc serve` MCP server (VISION.md Phase 5) as a new tool alongside `node_get`, `node_grep`, and `subgraph_get`; this feature does not introduce a new server, transport, or standalone `arc context` CLI command.
- No caching or index layer (VISION.md Phase 4, entirely unbuilt) backs this tool in this increment: each call reads the graph's current on-disk state directly, exactly like `node_grep` and `subgraph_get` already do — this keeps the feature shippable independently of Phase 4 and keeps results always fresh, at the cost of tying performance to direct-scan bounds (see SC-003).
- The attribute-match pass compares `query` against all scalar and array front-matter attribute values (e.g. title, tags, category), using substring containment — the same fields reachable through `--attr~=` filtering elsewhere in this codebase.
- Neighbor expansion traverses one hop (depth 1) outward from every node found by the content-match or attribute-match pass, following the same bidirectional edge/link traversal rules as `subgraph_get`/`arc subgraph`.
- Within the top-`limit` result, nodes are ranked in two tiers: direct matches (found by the content-match or attribute-match pass, hop-distance 0) rank ahead of nodes present only through neighbor expansion (hop-distance 1). Within each tier, nodes are ordered by connectivity — total edge/link count (degree), descending — since a more-connected node is more likely to anchor useful surrounding context; any remaining tie is broken alphabetically by id, guaranteeing the deterministic order SC-006 requires across repeated calls against unchanged graph state.
- Server-level behaviors this feature does not change — transport selection (stdio default, `--http` opt-in), concurrency handling, and the one-line-per-call stderr logging format — remain exactly as already established for the server's other tools.
