# Feature Specification: MCP Server (`arc serve`)

**Feature Branch**: `008-arc-serve-mcp`

**Created**: 2026-07-05

**Status**: Draft

**Input**: User description: "`arc serve` — start an MCP server (stdio transport by default; `--http <port>` for SSE) exposing these tools: `node_get(id)` → full node object (ARCNET-AST §4): attrs, text, edges, links; `node_grep(pattern, filter?)` → list of `{id, kind, line, snippet}` for nodes whose content matches a regexp pattern, optionally pre-filtered by the filter object; `subgraph_get(id, depth?)` → return the fully-resolved subgraph rooted at `id` to `depth` hops (default 1): a flat array of complete node objects for the seed and every reachable neighbor; covers the same operation as `arc subgraph` for agent context expansion mid-conversation."

## Clarifications

### Session 2026-07-05

- Q: Should `arc serve --http <port>` bind only to loopback by default, bind to all interfaces, or expose a configurable bind address? → A: The `--http` value is a full `[host]:port` address spec, not a bare port number: omitting the host (e.g. `:8080` or a bare `8080`) binds loopback-only (`127.0.0.1`) by default; an explicit host binds there instead — `0.0.0.0:8080` for all interfaces, `192.168.1.10:8080` for one specific NIC.
- Q: Does `arc serve` need to log anything while running, or is silent operation (output only on fatal startup errors) acceptable? → A: One line per tool call to stderr — method name, key arguments (id/pattern/depth), and outcome (success or error) — no full structured/metrics/tracing layer in this increment.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fetch a node's full context by id (Priority: P1)

An LLM agent connected to the graph as an MCP client wants the complete content of one specific node — its attributes, prose, and connections — to answer a question or continue a conversation, without the user manually opening the file and pasting it in.

**Why this priority**: This is the most basic unit of graph access an agent needs — a reliable way to turn "I know this node's id" into "I have its full content." Every other tool in this feature either helps an agent find an id first (`node_grep`) or expands outward from one (`subgraph_get`); both are only useful once this baseline retrieval works.

**Independent Test**: Can be fully tested by starting the server over its default (stdio) transport, connecting a standard MCP client, calling `node_get` with a known node's id, and confirming the returned object contains exactly that node's attributes, text, edges, and links, matching the on-disk file.

**Acceptance Scenarios**:

1. **Given** a running graph with a known node, **When** an agent calls `node_get` with that node's id, **Then** the tool returns the node's full object — attributes, text, edges, and links — matching the node's on-disk content.
2. **Given** an id that does not correspond to any node in the graph, **When** an agent calls `node_get` with that id, **Then** the tool reports a clear error and returns no node object.
3. **Given** the server has just started with its default transport, **When** an MCP client connects, **Then** it can discover and successfully invoke `node_get` with no additional configuration beyond starting the server.

---

### User Story 2 - Search node content to find relevant nodes (Priority: P2)

An agent doesn't yet know which node it needs — it has a topic or keyword instead. It searches node content for a pattern, optionally narrowed by kind/tag/attribute, to get a short list of candidate nodes before fetching any of them in full.

**Why this priority**: Search is how an agent typically arrives at an id in the first place; it depends on nothing beyond the server running, and makes User Story 1 reachable without the user first telling the agent exactly which node to open.

**Independent Test**: Can be fully tested by seeding a graph with nodes of known content, calling `node_grep` with a pattern that matches a known subset, and confirming the returned list contains exactly one entry per matching line, each with the matched node's id, kind, line number, and a snippet of the matching text.

**Acceptance Scenarios**:

1. **Given** a graph where some nodes' content matches a regexp pattern and others don't, **When** an agent calls `node_grep` with that pattern and no filter, **Then** the tool returns one entry per matching line across all nodes, each with `{id, kind, line, snippet}`.
2. **Given** the same graph, **When** an agent calls `node_grep` with the pattern plus a filter object (kind/tags/attrs/attrPatterns), **Then** only matches within nodes satisfying the filter are returned.
3. **Given** a pattern that matches no node's content, **When** an agent calls `node_grep`, **Then** the tool returns an empty list, not an error.
4. **Given** a syntactically invalid regexp pattern, **When** an agent calls `node_grep` with it, **Then** the tool reports a clear error and returns no results.

---

### User Story 3 - Expand a node into its neighborhood in one call (Priority: P3)

Mid-conversation, an agent already has one node's id but needs its surrounding context — everything reachable within a few hops — without issuing a separate `node_get` for every neighbor it discovers.

**Why this priority**: This is the same context-expansion value `arc subgraph` already delivers at the CLI, now reachable as a single tool call inside an agent's own reasoning loop; it builds on `node_get`'s node-object shape but is otherwise independent of Stories 1 and 2 for testing purposes.

**Independent Test**: Can be fully tested by seeding a graph where a seed node has a known, fixed set of directly reachable neighbors, calling `subgraph_get` with that seed's id and default depth, and confirming the returned array contains exactly the seed plus its direct neighbors, each as a complete node object, with no duplicates.

**Acceptance Scenarios**:

1. **Given** a seed node with several directly connected neighbors of different kinds, **When** an agent calls `subgraph_get` with the seed's id and no depth argument, **Then** the tool returns a flat array containing the seed plus every directly connected neighbor (depth 1), each as a complete node object.
2. **Given** the same graph, **When** an agent calls `subgraph_get` with an explicit `depth` of 2, **Then** the returned array includes every node reachable within 2 hops, and excludes nodes only reachable at a greater distance.
3. **Given** a node reachable from the seed by more than one path, **When** an agent calls `subgraph_get`, **Then** that node appears exactly once in the returned array.
4. **Given** an id that does not correspond to any node in the graph, **When** an agent calls `subgraph_get` with that id, **Then** the tool reports a clear error and returns no data.

---

### User Story 4 - Reach the server over a network connection (Priority: P4)

A team wants an MCP client that isn't a local child process of the server — for example, a shared client on another machine, or a browser-based tool — to reach the same three tools. They start the server with an HTTP port instead of the default stdio transport.

**Why this priority**: Most single-user, single-agent setups need nothing beyond the default stdio transport; network reachability only matters once a client can't be launched as the server's own subprocess. It changes how a client connects, not what any of the three tools do or return.

**Independent Test**: Can be fully tested by starting the server with `--http <port>`, connecting an SSE-capable MCP client to that port, and confirming the same `node_get`/`node_grep`/`subgraph_get` calls succeed and return results identical to the stdio transport for the same graph and arguments.

**Acceptance Scenarios**:

1. **Given** the server is started with `--http <port>`, **When** an MCP client connects over SSE to that port, **Then** it can discover and invoke all three tools, receiving the same results it would over the default stdio transport.
2. **Given** no `--http` flag is supplied, **When** the server starts, **Then** it serves exclusively over stdio, opening no network port.
3. **Given** `--http` is given a bare port or a `:port` address with no host, **When** the server starts, **Then** it binds loopback-only (`127.0.0.1`), unreachable from any other host on the network.
4. **Given** `--http` is given an explicit host in its address (e.g. `0.0.0.0:8080` or a specific NIC's address), **When** the server starts, **Then** it binds exactly that host, reaching whatever network that interface is attached to.
5. **Given** `--http` is given an address whose port is already in use, or that is not a syntactically valid `[host]:port` value, **When** the server attempts to start, **Then** it reports a clear error and does not silently fall back to stdio or to a different address.

---

### Edge Cases

- What happens when the target directory is not an initialized graph? The server must refuse to start and report this clearly, consistent with other graph commands.
- What happens when `node_get` or `subgraph_get` is called with an id that doesn't exist? A clear tool-level error is returned; the server itself keeps running and stays usable for subsequent calls.
- What happens when `subgraph_get`'s `depth` is negative or not an integer? The tool reports a clear usage error and returns no data.
- What happens when `subgraph_get`'s reachable set is very large? It is bounded the same way `arc subgraph` bounds it, so a single call always completes rather than growing unboundedly.
- What happens when `node_grep`'s filter object matches no nodes at all? The tool returns an empty list, not an error.
- What happens when a client disconnects mid-call (stdio EOF, or an SSE connection dropping)? The in-flight call is abandoned without corrupting the graph or crashing the server; the server remains able to accept new connections.
- What happens when multiple clients call tools concurrently over `--http`? Each call is served independently and correctly against the graph's current on-disk state; one client's call never blocks or corrupts another's result.
- What happens when the graph's files change on disk (e.g. another `arc` command applies a patch) while the server is running? Subsequent tool calls reflect the graph's current on-disk state; the server does not need to be restarted to see the change.
- What happens when a node's content changes mid-read (a concurrent write touches the exact file being read for a given call)? This is a known limitation shared with the graph's other read commands, not solved newly by this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST provide an `arc serve` command that starts an MCP server exposing exactly three tools: `node_get`, `node_grep`, and `subgraph_get`.
- **FR-002**: The server MUST default to the stdio transport when no transport flag is given, requiring no network port.
- **FR-003**: The server MUST support an `--http <addr>` flag that serves the same three tools over SSE instead of stdio, where `<addr>` is a `[host]:port` address spec: a bare port or a `:port` form (no host) MUST bind loopback-only (`127.0.0.1`), while an explicit host (e.g. `0.0.0.0` for all interfaces, or a specific NIC's address) MUST bind exactly that host.
- **FR-004**: The tool MUST refuse to start, and report this clearly, when the target directory is not an initialized graph.
- **FR-005**: The tool MUST refuse to start, and report a clear error, when `--http`'s address has a port already in use, or is not a syntactically valid `[host]:port` value.
- **FR-006**: `node_get(id)` MUST return the full node object for `id` — its attributes, prose text, edges, and predicate-grouped links — matching that node's current on-disk content.
- **FR-007**: `node_get` MUST report a clear error and return no node object when `id` does not correspond to any existing node.
- **FR-008**: `node_grep(pattern, filter?)` MUST return one entry per node-content line matching the regexp `pattern`, each entry carrying `{id, kind, line, snippet}`, optionally restricted beforehand to nodes matching an MCP filter object (kind/tags/attrs/attrPatterns, see Filtering), using the same matching semantics as `arc grep`.
- **FR-009**: `node_grep` MUST return an empty list, not an error, when `pattern` matches no node's content or when `filter` excludes every node.
- **FR-010**: `node_grep` MUST report a clear error when `pattern` is not a syntactically valid regexp.
- **FR-011**: `subgraph_get(id, depth?)` MUST return a flat array of complete node objects (same shape as `node_get`'s result) for the seed node named by `id` plus every node reachable within `depth` hops (default `1`), traversing structural edges/links in both directions, consistent with `arc subgraph`'s own traversal rules.
- **FR-012**: `subgraph_get` MUST include each reachable node exactly once regardless of how many distinct paths reach it, and MUST NOT loop indefinitely when the reachable set contains a cycle.
- **FR-013**: `subgraph_get` MUST report a clear error and return no data when `id` does not correspond to any existing node, or when `depth` is not a non-negative integer.
- **FR-014**: `subgraph_get` MUST apply the same reachable-set size safeguards (direct/backlink caps) as `arc subgraph`, so a single call always completes rather than growing unboundedly against a large or highly connected graph.
- **FR-015**: None of the three tools MUST ever modify the graph's files or git history — all three are strictly read-only.
- **FR-016**: Every tool call MUST reflect the graph's current on-disk state at the time of the call; the server MUST NOT require a restart to observe changes made by other processes since it started.
- **FR-017**: The server MUST remain available to serve subsequent tool calls after a single call fails (unknown id, invalid pattern, invalid depth) — a per-call error MUST NOT terminate the server process.
- **FR-018**: When serving over `--http`, the server MUST handle multiple concurrent client calls without one call's failure or slowness corrupting or blocking another's result.
- **FR-019**: For every tool call, the server MUST emit one line to stderr recording the tool name, its key arguments (id, pattern, or depth as applicable), and whether it succeeded or failed — sufficient for an operator to reconstruct what an agent asked for and what it got back, without a separate metrics or tracing system.

### Key Entities

- **Node Object**: The unit returned by `node_get` and, per-entry, by `subgraph_get` — id (basename), kind, attributes, prose text, structural edges, and predicate-grouped links (ARCNET-AST §4); no attribute stripped or reformatted from its on-disk representation.
- **Match**: One entry in `node_grep`'s result — the id and kind of the node containing the match, the matched line number, and a snippet of the matching text.
- **Filter**: The optional, composable node-selection object (`kind`, `tags`, `attrs`, `attrPatterns`) narrowing which nodes `node_grep` searches, matching the MCP filter object schema shared across this codebase's tools (see Filtering).
- **Transport**: The connection mode a client uses to reach the server — stdio (default, no network exposure) or HTTP/SSE (opt-in via `--http <port>`); both expose the identical three tools with identical behavior.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can retrieve any existing node's complete content by id in a single tool call, with zero manual file-opening or copy-pasting by the user.
- **SC-002**: An MCP-compatible client can discover and successfully invoke all three tools immediately after connecting over either transport, with no configuration beyond starting the server.
- **SC-003**: Search results from `node_grep` against a graph of several thousand nodes are returned in under 2 seconds.
- **SC-004**: `subgraph_get` against a graph of several thousand nodes completes in under 10 seconds, matching `arc subgraph`'s own performance bar.
- **SC-005**: Across a full test session of mixed valid and invalid calls (unknown ids, bad patterns, bad depths), the server never crashes and never requires a restart to keep serving valid calls.
- **SC-006**: Running any of the three tools any number of times never modifies any graph file or git history, verified by the graph's state being byte-for-byte identical before and after a session.
- **SC-007**: The same request against the same graph state returns identical results whether made over stdio or over `--http`, verified across all three tools.
- **SC-008**: Given the server's stderr output from a session, an operator can identify which tool was called, its key arguments, and whether it succeeded, for 100% of calls made in that session, without any additional tooling.

## Assumptions

- `id` in all three tools is the node's basename, consistent with `arc subgraph <basename>`'s existing addressing scheme.
- `node_grep`'s matching semantics (regexp scope, per-line matching, snippet extraction) and `subgraph_get`'s traversal semantics (hop-counting, direction, reachable-set caps and their defaults) are identical to the already-implemented `arc grep` and `arc subgraph` commands — this feature exposes those existing behaviors as MCP tools rather than redefining them.
- No caching or index layer backs these tools in this increment: each call reads the graph's current on-disk state directly, exactly like the equivalent CLI commands do today. This keeps results always fresh but ties this feature's performance to the same bounds as the CLI commands it mirrors.
- All three tools are read-only by construction (no tool in this feature accepts or applies a mutation), so no `--readonly` flag is needed for this increment; that flag becomes meaningful once a future increment adds write-capable tools.
- `--http` includes no built-in authentication in this increment; loopback-only-by-default binding (see Clarifications) limits accidental exposure, but an operator who explicitly binds a non-loopback host is opting into a trusted-network-only posture and is expected to place it behind their own access control — consistent with this being the first, minimal slice of VISION.md's broader MCP Server phase.
- Concurrent external writes to the graph while the server is running (e.g. another process applying a patch mid-read) are not specially locked against in this increment, matching the existing, documented open concurrency question for this codebase; a call may occasionally observe a torn read of a file being written at that exact moment, same as the CLI commands it mirrors.
- The server is a single long-running process per invocation, stopped by standard process termination (not by a dedicated shutdown tool).
