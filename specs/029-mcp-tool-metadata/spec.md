# Feature Specification: MCP Tool Metadata Colocation

**Feature Branch**: `029-mcp-tool-metadata`

**Created**: 2026-08-29

**Status**: Draft

**Input**: User description: "Improve metadata management for MCP Server by coallocating the each tool and its params specification in single block at the code base. Improve the specification of arguments with possibility to add descriptions and examples. As part of this feature, the tool maintainer gets possibilities to 1. Specify the server and main workflows; 2. Specify tool/function description, input and output; 3. Specify the complex syntax of filters with examples; 4. Guidance on which tool to prefer when they overlap"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One block per tool (Priority: P1)

A maintainer adding or changing an MCP tool wants the tool's name, description, every input parameter (with its own description), and the shape/meaning of its output to live in one colocated place in the codebase, so that reviewing or editing a tool's complete contract does not require jumping between a separate parameter-definition block and a separate registration block.

**Why this priority**: This is the core problem statement — today a tool's description lives at the point it is registered, while its parameters (each with at most a one-line description) are declared in a separate struct earlier in the file, and its output is documented only in code comments, not exposed as part of the tool's own metadata. Without this, every other improvement in this feature has nowhere consistent to attach.

**Independent Test**: Can be fully tested by picking any one existing MCP tool (e.g. `node_grep`), locating its complete specification (name, description, each parameter's name/description, output description) within a single contiguous block, and confirming no part of that contract is defined elsewhere.

**Acceptance Scenarios**:

1. **Given** an existing MCP tool, **When** a maintainer opens its specification, **Then** the tool's name, description, full parameter list (each with a description), and output description are all present in one colocated block, with no part of that contract declared in a separate location.
2. **Given** a maintainer is adding a brand-new MCP tool, **When** they follow the established pattern, **Then** they can express name, description, parameters, and output in the same single block without needing to touch a second, disconnected block elsewhere.
3. **Given** a maintainer changes a parameter (renames it, adds one, or changes whether it's required), **When** they save that change, **Then** the parameter's description lives right alongside the change, so there is no separate list left stale.

---

### User Story 2 - Parameter descriptions and examples (Priority: P1)

A maintainer wants to give every tool parameter a human-readable description plus one or more concrete example values, so an AI client calling the tool can infer correct usage from the metadata alone rather than guessing or failing calls first.

**Why this priority**: Parameters today carry only a single short description string and no example values at all. This is functionally equal in priority to Story 1 — colocation without richer per-parameter content wouldn't change what a client actually sees.

**Independent Test**: Can be fully tested by inspecting any one parameter of any existing tool and confirming it exposes both a non-empty description and at least one example value that is itself valid input for that parameter.

**Acceptance Scenarios**:

1. **Given** a tool parameter that takes a simple value (e.g. a node id or a search pattern), **When** a maintainer specifies it, **Then** they can attach a description and at least one example value that a client can read before making a call.
2. **Given** a parameter that is optional and has a default behavior when omitted, **When** a maintainer documents it, **Then** the default behavior is stated as part of that same parameter's specification.
3. **Given** a client requests the tool's metadata (e.g. during connection/tool-listing), **When** the server responds, **Then** every parameter's description and example values are included in that response, not only visible to someone reading the source code.

---

### User Story 3 - Complex filter syntax with examples (Priority: P2)

A maintainer wants to document the structured filter argument accepted by several tools (source/predicate/target constraints, pattern matching, single-value-or-list forms) with multiple realistic example values covering distinct usage patterns, so a client can construct a valid, useful filter without trial and error.

**Why this priority**: The filter argument is the most complex, highest-value-to-document input in the server, and today its syntax is explained only in free-text prose (in the tool description and in a separate whole-server instructions string) with no example payloads attached to the argument itself. It builds directly on Story 2 (examples in general) but deserves separate priority because its syntax is materially harder to guess correctly than a scalar parameter.

**Independent Test**: Can be fully tested by inspecting the filter argument on any tool that accepts one, and confirming it carries two or more example values that each demonstrate a distinct, valid usage pattern (e.g. a single exact-match constraint, and a pattern-based constraint restricted to one relation).

**Acceptance Scenarios**:

1. **Given** a tool that accepts the structured filter argument, **When** a maintainer documents it, **Then** at least two examples are attached showing different valid usage patterns of that argument.
2. **Given** a client reads a filter example, **When** it submits that example unmodified as a call argument, **Then** the server accepts it as valid input.
3. **Given** the filter argument's accepted syntax changes in the future (a new constraint type is added), **When** a maintainer updates the syntax, **Then** the examples for that argument live close enough to the syntax definition that a reviewer is likely to notice they now need updating too.

---

### User Story 4 - Workflow and tool-preference guidance (Priority: P3)

A maintainer wants to record (a) the server's overall purpose and recommended multi-tool workflows, and (b) explicit guidance on which tool to prefer when two or more tools' capabilities overlap, in a form that is structured and attributable to specific tools rather than a single freeform paragraph that has to be remembered and manually kept in sync.

**Why this priority**: This guidance already exists today, but only as one hand-maintained paragraph disconnected from the tools it describes, making it easy to forget to update when a tool is added, removed, or changed. It is lower priority than Stories 1-3 because the guidance is currently present in some form (just not structured or colocated), whereas parameter examples and colocation are missing outright.

**Independent Test**: Can be fully tested by (a) confirming a client receives a description of the server's overall purpose and recommended starting workflow when it connects, and (b) picking any two tools whose capabilities overlap and confirming documented guidance exists stating which one to prefer and why.

**Acceptance Scenarios**:

1. **Given** a client connects to the server, **When** it reads the server-level guidance, **Then** it finds a description of the server's purpose and the recommended order of tool calls for a typical session.
2. **Given** two or more tools that can both be used to accomplish a similar goal, **When** a maintainer documents them, **Then** each such tool's specification (or the server-level guidance) states when to prefer it over the other(s) and why.
3. **Given** a new tool is added whose capability overlaps an existing tool, **When** a maintainer completes its specification, **Then** the preference guidance for that overlap is recorded as part of adding the tool, not deferred to a later, separate edit of an unrelated block.

---

### Edge Cases

- What happens when a parameter's valid values are better expressed as a fixed set of choices rather than free-form examples (e.g. a mode flag)? The specification approach must accommodate both example values and enumerated choices.
- How does the system handle a tool whose output is not simple text (e.g. structured data plus a rendered summary) — can both be described without ambiguity about which is authoritative?
- What happens when a maintainer adds a new tool but does not supply a description, a parameter description, or any examples — is the omission visible/flagged, or does the tool silently ship under-documented?
- How is overlap guidance kept correct when a tool that was previously the "preferred" choice is removed or renamed — is there a way to catch guidance that now refers to a tool that no longer exists?
- What happens when a single example value would be misleading in isolation (e.g. a filter example that only makes sense combined with a specific node id) — can an example be paired with a short explanation of what it demonstrates, not just the raw value?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a maintainer to define, for each MCP tool, its name, its description, its full list of input parameters, and a description of its output within a single colocated specification unit in the codebase.
- **FR-002**: The system MUST allow each input parameter to carry a human-readable description distinct from its name and type.
- **FR-003**: The system MUST allow a maintainer to attach one or more concrete example values to any input parameter.
- **FR-004**: The system MUST allow a maintainer to attach two or more example values to the structured filter argument, each demonstrating a distinct valid usage pattern.
- **FR-005**: The system MUST allow a maintainer to record, for an optional parameter, what behavior applies when it is omitted, as part of that parameter's own specification.
- **FR-006**: The system MUST allow a maintainer to describe a tool's output — its shape and meaning — as part of the tool's specification, not only in surrounding code comments.
- **FR-007**: The system MUST allow a maintainer to record the server's overall purpose and its recommended multi-tool workflow(s) in a form that is part of the tool metadata the server exposes, not only a comment visible to someone reading the source.
- **FR-008**: The system MUST allow a maintainer to record explicit guidance on which tool to prefer, and why, for any set of tools whose capabilities overlap.
- **FR-009**: A connecting client MUST receive, as part of the server's exposed metadata, every parameter's description and example values, the tool's output description, the server's overall purpose/workflow guidance, and any recorded tool-preference guidance — not only a maintainer reading the source code.
- **FR-010**: The system MUST apply this specification format to every MCP tool the server currently exposes, so the improvement is not limited to newly added tools.
- **FR-011**: Adding a new tool or changing an existing tool's parameters MUST NOT require editing more than one colocated specification unit to keep the tool's name, description, parameters, and output description consistent with each other.

### Key Entities

- **Tool Specification**: The complete, colocated definition of one MCP tool — its name, description, ordered list of parameters, and output description.
- **Parameter Specification**: One input parameter's name, whether it is required, its description, its default behavior (if optional), and its example value(s).
- **Filter Usage Example**: A concrete, valid example of the structured filter argument, paired with a short note on what usage pattern it demonstrates.
- **Server Workflow Guidance**: The server's stated overall purpose and its recommended order/combination of tool calls for a typical session.
- **Tool Preference Guidance**: A statement, attached to one or more overlapping tools, of which tool to use in which situation and why.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For 100% of MCP tools the server exposes, a maintainer can find that tool's complete contract (name, description, every parameter's description, output description) inside a single contiguous specification block.
- **SC-002**: For 100% of MCP tool parameters, the metadata a connecting client receives includes both a non-empty description and at least one example value.
- **SC-003**: For every tool that accepts the structured filter argument, the metadata a connecting client receives includes at least two example filter values demonstrating distinct usage patterns.
- **SC-004**: A connecting client receives, without making any tool call, a statement of the server's purpose and its recommended starting workflow.
- **SC-005**: For every identified pair/group of overlapping tools, documented guidance exists stating which tool to prefer and why, discoverable from the metadata the server exposes.
- **SC-006**: A maintainer adding a new tool can supply its name, description, parameters (with descriptions and examples), and output description by editing a single specification unit, with zero additional edits required elsewhere to keep the tool internally consistent.

## Assumptions

- "Tool maintainer" refers to the developer(s) working in this codebase who add or change MCP tools; the feature's direct beneficiary is that maintainer's workflow, and its indirect beneficiary is any AI client that connects to the server and relies on the resulting metadata to choose and call tools correctly.
- This feature retrofits every MCP tool the server exposes today (not only tools added going forward), since the goal is to fix metadata management as a whole rather than to establish a pattern only new tools follow.
- The set of currently overlapping tools that need explicit preference guidance is derived from the server's existing tools whose capabilities already visibly overlap (e.g. multiple tools that can each narrow results by the same structured filter, or multiple tools that can each be used to gather context about a node); as tools are added or removed in the future, maintainers are expected to keep this guidance current using the same colocated mechanism.
- Example values attached to parameters are illustrative and are not a substitute for input validation; a client is not guaranteed a call will succeed merely because it used a documented example verbatim against a different graph than the one an example was written against (e.g. a node id in an example may not exist in every graph).
- Output description content documents the shape/meaning of a tool's response for a human or AI reader; it is not intended to change what the server actually returns for existing tools as part of this feature.
