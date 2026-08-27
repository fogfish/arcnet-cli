# Feature Specification: Graph Ontology Schema Tool

**Feature Branch**: `026-mcp-schema-tool`

**Created**: 2026-08-27

**Status**: Draft

**Input**: User description: "Add `schema()` function to MCP server. Its purpose to explain a full ontology of the graph. It return all existing class/predicates and its description. The MCP server should advertise usage of the function as the first operation in the session."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discover the graph's ontology in one call (Priority: P1)

An AI agent connects to the graph server to work with a knowledge graph it has never seen before. Before it can create, query, or link nodes correctly, it needs to know what classes (node types) and predicates (relationships/attributes) exist and what each one means.

**Why this priority**: Without this, an agent has to guess class and predicate names, or discover them only after a failed operation. This is the foundational capability the rest of the feature depends on.

**Independent Test**: Can be fully tested by connecting to the server and invoking the schema operation with no arguments, and verifying the response lists every class and predicate currently defined in the graph, each with a description.

**Acceptance Scenarios**:

1. **Given** a newly initialized graph with only the built-in vocabulary, **When** the agent invokes the schema operation, **Then** it receives every built-in class and predicate, each with a name and a human-readable description.
2. **Given** a graph where project-specific classes or predicates have been added on top of the built-in vocabulary, **When** the agent invokes the schema operation, **Then** the response includes both the built-in and the project-specific definitions.
3. **Given** a class definition in the response, **When** the agent inspects it, **Then** it can tell which attributes are required and which are optional for that class.
4. **Given** a predicate definition in the response, **When** the agent inspects it, **Then** it can tell what role the predicate plays and how conflicting values are resolved when merged.

---

### User Story 2 - Learn about the schema operation without prior documentation (Priority: P2)

An agent connects to the graph server for the first time in a session, with no external documentation about which operations exist or in what order to use them. It needs to learn, from the server itself, that a schema discovery operation exists and that using it first will make every subsequent operation more accurate.

**Why this priority**: Discoverability is what makes User Story 1 actually get used. A schema operation nobody knows to call delivers no value.

**Independent Test**: Can be fully tested by connecting a fresh client to the server and confirming that, before any other operation is invoked, the client is told that a schema discovery operation exists and that it is the recommended starting point.

**Acceptance Scenarios**:

1. **Given** a client that has just connected to the server, **When** the session starts, **Then** the client is presented with a description of the schema operation and guidance that it should be used first.
2. **Given** a client that already knows about the schema operation from a previous session, **When** it connects again, **Then** the same guidance is presented, so behavior is consistent regardless of what the client remembers.

---

### User Story 3 - Use ontology descriptions to form correct requests (Priority: P3)

Having retrieved the ontology, an agent uses the descriptions to decide which class to use when creating a new node, and which predicate to use when linking two nodes, without needing to consult a human or trial-and-error against the graph.

**Why this priority**: This is the payoff of User Stories 1 and 2 — accurate downstream operations — but it depends on both being in place first, and is validated indirectly through them.

**Independent Test**: Can be fully tested by giving an agent only the schema operation's output (no other documentation) and confirming it selects the correct class and predicate names for a set of representative graph-editing tasks.

**Acceptance Scenarios**:

1. **Given** the ontology response, **When** the agent needs to represent a new kind of information already covered by an existing class, **Then** it selects that class's name rather than inventing a new one.
2. **Given** the ontology response, **When** the agent needs to relate two nodes, **Then** it selects a predicate whose description matches the intended relationship.

---

### Edge Cases

- What does the schema operation return when the graph has just been initialized and no project-specific classes or predicates have been added yet? (Must still return the full built-in vocabulary, not an empty result.)
- What happens if the schema operation is invoked more than once in the same session? (Must succeed every time and return the current definitions, not fail as a duplicate or redundant call.)
- What happens if project-specific classes or predicates are added to the graph after the session's initial advertisement was shown? (A later call to the schema operation must reflect the new definitions.)
- What happens if a class or predicate definition is missing a description in the underlying graph data? (The operation must still return the name and other known fields rather than omitting the entry or failing.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The graph server MUST provide an operation that, when invoked, returns the complete ontology of the graph: every currently defined class and every currently defined predicate.
- **FR-002**: For each class, the operation MUST return the class's name, its required attributes, its optional attributes, and a human-readable description of what the class represents.
- **FR-003**: For each predicate, the operation MUST return the predicate's name, its role (what kind of relationship or attribute it represents), its merge behavior (how conflicting values are resolved), and a human-readable description.
- **FR-004**: The operation's response MUST include both the built-in vocabulary and any project-specific classes or predicates that have been defined in the graph.
- **FR-005**: The operation MUST reflect the graph's current schema at the time it is invoked, including any project-specific classes or predicates added earlier in the same session.
- **FR-006**: The operation MUST NOT require any input parameters to return the full ontology.
- **FR-007**: The operation MUST be read-only: invoking it MUST NOT create, modify, or delete any class, predicate, or node in the graph.
- **FR-008**: When a client connects to the graph server, the server MUST present a description of the schema operation and guidance recommending it be invoked first, before the client needs to have any prior knowledge of the operation's existence.
- **FR-009**: The session-start guidance MUST be presented consistently on every new connection, regardless of whether the connecting client has interacted with the server in a prior session.

### Key Entities

- **Class**: A node type defined in the graph. Represented by a name, a description of what it represents, and the sets of attributes that are required versus optional for nodes of that type.
- **Predicate**: A relationship or attribute definition usable between or on nodes in the graph. Represented by a name, a description of its meaning, the role it plays, and how conflicting values for it are merged.
- **Ontology**: The complete collection of classes and predicates currently defined in a given graph, combining built-in vocabulary with any project-specific additions.
- **Session**: A single connected-client interaction with the graph server, over which the schema operation's session-start guidance is presented once at the start.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A client connecting to the graph server for the first time can retrieve the complete ontology — every class and predicate with its description — in a single operation call, with no prior documentation.
- **SC-002**: 100% of new client sessions receive the schema operation's guidance before any graph-editing operation is invoked.
- **SC-003**: Given only the ontology response, an agent selects the correct class name for a new node in at least 95% of representative graph-editing tasks, without trial-and-error against the graph.
- **SC-004**: After a project-specific class or predicate is added to the graph, a subsequent call to the schema operation within the same session reflects that addition 100% of the time.

## Assumptions

- "The graph" refers to the single graph store that the connected MCP session is already scoped to; the schema operation does not need to select among multiple graphs.
- The full ontology (built-in vocabulary plus project-specific definitions) is always returned; the operation does not support filtering to a subset of classes or predicates in this iteration.
- "Advertise ... as the first operation" is interpreted as a strong, consistent recommendation communicated to the client at the start of every session (via whatever session-level introduction mechanism the server already uses) rather than the server outright blocking or rejecting other operations until the schema operation has been called. Enforcing a hard ordering requirement would need session-state tracking with its own failure-handling behavior, which is a larger scope not indicated by the request.
- Descriptions returned by the operation are the same descriptions already authored for classes and predicates elsewhere in the system (e.g., when they were defined), not newly authored text specific to this operation.
- The operation is exposed through the same graph server that already exposes other read operations to MCP clients, not a separate service.
