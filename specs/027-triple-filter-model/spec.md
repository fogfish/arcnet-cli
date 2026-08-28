# Feature Specification: Triple Filter for Node Attributes and Graph Edges

**Feature Branch**: `027-triple-filter-model`

**Created**: 2026-08-27

**Status**: Draft

**Input**: User description: "Replace `core.Filter`'s node-matching model with a single-hop conjunctive triple filter, so both node attributes and graph edges can be matched — and, critically, so neighbor expansion in `arc subgraph`/`arc serve`'s `subgraph_get`/`context_retrieve` can be scoped by edge predicate instead of walking every outgoing/incoming edge regardless of what relation it represents."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scope neighbor expansion to a specific relation (Priority: P1)

A user — often an LLM agent acting through the knowledge-service tools — wants to know what a node relates to through one specific kind of connection (for example, "what does this source cite"), not everything one hop away regardless of relation. Today, expanding a node's neighborhood always follows every connection it carries, so the caller gets back citations, mentions, and every other relation mixed together and must sort out which is which after the fact. This capability lets the caller name the relation up front, so expansion only follows connections of that kind.

**Why this priority**: This is the entire reason the feature exists. Without it, retrieval calls are forced to be maximally wide even when the caller knows exactly what relation they care about — costing more to transmit, more for a downstream model to sift through, and losing precision. Every other capability in this feature exists to support this one cleanly.

**Independent Test**: Can be fully tested by building a graph where a seed node carries connections of at least two different relations to different targets, requesting neighborhood expansion scoped to one relation, and confirming only targets reached through that relation appear — targets reachable only through the other relation are absent even though they are within the requested hop distance.

**Acceptance Scenarios**:

1. **Given** a seed node with connections of relation "cites" to node A and relation "mentions" to node B, **When** a caller expands the seed's neighborhood scoped to relation "cites", **Then** the result includes node A and excludes node B.
2. **Given** the same seed, **When** a caller expands its neighborhood with no relation scope specified, **Then** the result includes both node A and node B, unchanged from today's behavior.
3. **Given** a seed node with no connection of the requested relation, **When** a caller expands its neighborhood scoped to that relation, **Then** the result contains no expanded neighbors and no error is reported.
4. **Given** an expansion requested beyond one hop, **When** a relation scope is specified, **Then** the scope is applied at every hop of the expansion, so a node reached only through an intermediate connection of a different relation is excluded even if a later hop from it would have used the scoped relation.
5. **Given** a relation-scoped expansion, **When** the results are additionally narrowed by other matching criteria (see User Story 2), **Then** both the scoping and the narrowing apply together — scoping controls what gets reached, narrowing controls what survives into the final result.

---

### User Story 2 - Match nodes by either their attributes or their relations (Priority: P2)

A user wants to select nodes using a single, uniform kind of criterion regardless of whether the matching fact lives in the node's own descriptive attributes (its type, tags, or other fields) or in one of its relations to other nodes (for example, "give me every node that cites node X"). Today, relations are invisible to search and selection entirely — only attributes can be matched. This capability lets a selection criterion match a node through either kind of fact.

**Why this priority**: This is the same job today's node selection already does for `arc grep` and for the final narrowing of what's returned by the neighborhood-retrieval tools, extended to also see relations. It's foundational to User Story 1's scoping (which reuses the identical matching rules) but delivers value on its own even without touching traversal.

**Independent Test**: Can be fully tested by building a graph where one node's own attribute matches a given criterion and a different node's relation (not its attributes) matches an equivalent criterion, applying that criterion as a selection filter, and confirming both nodes are selected.

**Acceptance Scenarios**:

1. **Given** a node whose attribute matches a selection criterion and a separate node whose relation (not its attributes) matches the same kind of criterion, **When** the criterion is applied as a filter, **Then** both nodes are selected.
2. **Given** a filter combining more than one criterion, **When** it is applied to a candidate node, **Then** the node is selected only if every criterion is satisfied — each independently, by any one attribute or relation the node carries, not necessarily the same one.
3. **Given** a filter with a criterion that no node's attributes or relations satisfy, **When** the filter is applied, **Then** no node is selected and no error is reported.
4. **Given** the neighborhood-retrieval tools' final result narrowing, **When** a filter is supplied, **Then** the seed node used to start the retrieval is still always present in the result regardless of whether it matches the filter, consistent with today's behavior — the filter narrows only the surrounding nodes.

---

### User Story 3 - Existing command-line filtering keeps working unchanged (Priority: P3)

A user who already filters nodes from the command line by type, by tag, or by attribute value or pattern continues to get exactly the same results, spelled exactly the same way, after this feature ships. Nothing about their existing scripts, muscle memory, or documentation needs to change.

**Why this priority**: This is a non-regression guarantee rather than new value — it protects existing workflows while the underlying matching model is replaced beneath them. It is lower priority than User Stories 1 and 2 only because it delivers no new capability, but it is a hard requirement, not an optional nicety.

**Independent Test**: Can be fully tested by re-running the existing command-line filtering test scenarios (type, tag, attribute equality, attribute pattern, and combinations of these) unchanged and confirming identical observable output to before this feature.

**Acceptance Scenarios**:

1. **Given** a command-line invocation restricting results to one or more types, **When** it is run, **Then** results include a node if its type equals any one of the given types, exactly as before.
2. **Given** a command-line invocation requiring one or more tags, **When** it is run, **Then** results include only nodes carrying every one of the given tags, exactly as before.
3. **Given** a command-line invocation requiring an attribute to equal a value, or to match a pattern, **When** it is run, **Then** results include only nodes whose named attribute satisfies that condition, exactly as before, including when several such conditions are combined in one invocation.
4. **Given** any existing automated test covering command-line filtering behavior, **When** it is re-run after this feature ships, **Then** it passes without modification.

---

### Edge Cases

- What happens when a selection criterion constrains only the "what it connects to" side, with no relation named? It matches any attribute value or relation target equal to the given value(s), regardless of which attribute name or relation it came from.
- What happens when two criteria in the same filter could each be satisfied by a node's attributes or relations in more than one way? Each criterion only needs one satisfying attribute or relation on the node; different criteria may be satisfied by different attributes or relations on the same node.
- What happens when relation-scoped expansion excludes every relation the seed carries? The expansion produces no neighbors; the seed itself is still returned, consistent with today's behavior for an unmatched or fully-exhausted neighborhood.
- What happens when a node happens to carry both an attribute and a relation that share the same name (for example, an attribute called "cites" alongside a relation also named "cites")? A criterion naming that shared name is satisfied by either the attribute or the relation — the caller does not need to know which one it is.
- What happens when a caller supplies conflicting or nonsensical combinations of criteria (for example, a relation-scope combined with narrowing criteria that no reachable node could ever satisfy)? The retrieval still completes normally and simply returns nothing beyond the seed — this is not treated as an error.
- What happens to a caller still sending the previous relation-blind selection request shape to the knowledge-service tools? It is no longer accepted — those tools require the new shape once this feature ships, with no transitional dual acceptance.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST represent node-selection criteria as a filter made of one or more statements, each combined with the others by requiring all to be satisfied (conjunction).
- **FR-002**: Each statement MUST independently constrain, or leave open ("wildcard"), three positions: which node the fact is about, which relation or attribute name the fact uses, and what value or target the fact points to.
- **FR-003**: When a statement constrains a position, that position MUST accept a set of one or more acceptable values, satisfied if the fact matches any one of them (mirroring today's type-filter OR-of-values convention).
- **FR-004**: The system MUST treat both a node's own descriptive attributes and its outgoing relations to other nodes as equally matchable facts for filtering purposes — a fact contributed by an attribute and a fact contributed by a relation MUST be selectable through the same statement shape.
- **FR-005**: When testing whether a candidate node satisfies a filter for final selection (as used by content search and by the final narrowing step of neighborhood retrieval), the system MUST select the node only if every statement in the filter is satisfied by at least one of that node's own attributes or relations — different statements MAY be satisfied by different attributes or relations on the same node.
- **FR-006**: During one-hop neighborhood expansion (both the command-line neighborhood-extraction tool and the equivalent knowledge-service retrieval tools), the system MUST support restricting which relations are followed to reach new nodes, in addition to (not instead of) the existing final-selection narrowing.
- **FR-007**: When no relation restriction is specified for a neighborhood expansion, the system MUST follow every relation the expanding node carries, exactly as before this feature.
- **FR-008**: When a relation restriction is specified for a multi-hop neighborhood expansion, the system MUST apply that same restriction at every hop of the expansion.
- **FR-009**: The neighborhood-extraction command's seed node MUST always be present in its output regardless of whether it satisfies the filter, consistent with today's behavior; the filter and any relation restriction apply only to what is reached from the seed and to which of the reached nodes survive into the result.
- **FR-010**: The command-line `--type <type>` option MUST continue to accept repeated values with OR semantics, restricting results to nodes whose type equals any one of the given values, with no observable change from today's behavior.
- **FR-011**: The command-line `--tag <tag>` option MUST continue to accept repeated values with AND semantics, restricting results to nodes carrying every one of the given tags, with no observable change from today's behavior.
- **FR-012**: The command-line `--attr <name>=<value>` option MUST continue to restrict results to nodes whose named attribute equals the given value (case-insensitive), and `--attr <name>~=<pattern>` MUST continue to restrict results to nodes whose named attribute matches the given pattern; both MUST remain repeatable with AND semantics across occurrences, with no observable change from today's behavior.
- **FR-013**: Combining `--type`, `--tag`, and `--attr` options in a single command-line invocation MUST continue to require all of them to be satisfied together, with no observable change from today's behavior.
- **FR-014**: Every existing automated test that exercises command-line filtering behavior (type, tag, attribute equality, attribute pattern, and their combinations) MUST continue to pass without modification after this feature ships.
- **FR-015**: The knowledge-service tools that accept a selection filter (content search, and both neighborhood-retrieval tools) MUST accept it in a new shape capable of expressing the statement-based filter described above, including a relation-restriction for the neighborhood-retrieval tools' expansion step; the previous relation-blind filter shape MUST be removed with no dual acceptance and no transitional alias.
- **FR-016**: The new knowledge-service filter shape's field names MUST be understandable by a caller with no prior background in formal graph/logic terminology (for example, describing "what it's about", "what kind of fact", and "what it points to" in plain, self-explanatory terms) rather than reusing specialist jargon.
- **FR-017**: The system MUST NOT support chaining or joining multiple statements into a multi-hop pattern (where one statement's target position feeds into another statement's node-about position as a shared unknown) — every statement in a filter is evaluated independently against a single node or a single relation, never across a path.

### Key Entities

- **Filter**: The complete node-selection criteria for one operation — a set of statements, all of which must be satisfied (conjunction). A filter with no statements matches every node.
- **Statement**: One clause of a filter, independently constraining (or leaving open) which node a fact is about, which relation or attribute name it uses, and what value or target it points to; a constrained position is satisfied by any one of a set of acceptable values.
- **Attribute Fact**: A fact contributed by one of a node's own descriptive attributes — the node, the attribute's name, and its literal value.
- **Relation Fact**: A fact contributed by one of a node's outgoing relations to another node — the node, the relation's name, and the related node.
- **Relation Restriction**: The part of an active filter, applied only during neighborhood expansion, that determines which relations are followed to discover new nodes — distinct from, and applied in addition to, the filter's role in deciding which discovered nodes survive into the final result.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A caller who wants "everything connected to a node through one specific relation" gets exactly that in a single retrieval call, with zero nodes reached through any other relation, verified against a test graph where the seed carries connections of at least two distinct relations.
- **SC-002**: 100% of existing command-line filtering test scenarios (type, tag, attribute equality, attribute pattern, and their combinations) produce identical output after this feature ships compared to before.
- **SC-003**: A selection criterion matches a node whether the matching fact comes from an attribute or from a relation, with 100% accuracy verified against a test graph containing both cases.
- **SC-004**: Relation-scoped neighborhood retrieval reduces the number of nodes returned, compared to the equivalent unscoped retrieval over the same seed and depth, whenever the seed carries relations of more than one kind — demonstrating the intended narrowing effect.
- **SC-005**: Knowledge-service callers can express both "which relations to follow" and "which resulting nodes to keep" in one filter argument, without needing a second, separate parameter.

## Assumptions

- "Single-hop" scope means a filter's statements each describe one node's own facts (its attributes or its direct relations); chaining statements into a multi-hop path query is out of scope for this feature and remains a separate, not-yet-designed capability (see the project's roadmap notes on query-style filtering).
- The command-line neighborhood-extraction tool gains the ability to scope its expansion by relation as part of this feature, consistent with the feature's stated purpose; the exact command-line syntax for expressing that scope is a planning-level detail, not a behavioral change to the flags this spec calls out for compatibility (`--type`/`--tag`/`--attr`).
- Compatibility is guaranteed only for the exact command-line flags named in User Story 3; the knowledge-service (tool-calling) interface for filtering is intentionally allowed to change shape, since no external caller depends on its previous wire format being preserved.
- A filter's relation restriction and its final-selection role are expressed through the same filter value passed once per retrieval call, not through two independent parameters, consistent with how today's single filter argument already serves final-selection alone.
- Listing a node's own relations directly (as opposed to using them to filter or scope expansion) is out of scope for this feature; that remains a separate, not-yet-built capability.
- Default behavior when no filter or relation restriction is supplied is unchanged from today: every relation is followed during expansion, and every node is eligible for final selection.
