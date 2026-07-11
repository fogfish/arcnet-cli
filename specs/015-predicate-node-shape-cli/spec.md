# Feature Specification: CLI/MCP "Type" Terminology Consistency

**Feature Branch**: `015-predicate-node-shape-cli`

**Created**: 2026-07-11

**Status**: Draft

**Input**: User description: "Update every arc command whose output or behavior depends on a graph node's shape (arc apply, arc grep, arc subgraph, arc serve's MCP tools) so they correctly produce and consume the new predicate-first node representation (@id/@type, open texts, array-valued attrs, unified edges) instead of the old kind/id/text-notes/attrs/edges-links shape, and update the corresponding `--json` output schemas. This includes: arc apply's per-node create/merge reporting (which today references Kind in its human-readable and --json output); arc grep and arc subgraph's node filtering (which today filters on a flat Kinds list and flat Attrs map — the filter semantics need to keep working sensibly against array-valued attrs); arc subgraph's exported patch shape (kernel.SubgraphResult.Patch.Nodes, an already-documented --json contract that external tools may depend on); and arc serve's MCP tool responses, if they expose node shape directly. Out of scope: adding new CLI flags or commands; changing arc's human-readable (non-JSON) terminal output beyond what's mechanically required by the field renames (e.g. \"kind:\" labels in reporter output becoming \"type:\")."

**Scoping note**: Investigation ahead of writing this specification found that the underlying node-shape migration this input describes (`@id`/`@type`, array-valued attributes, unified edges, and the corresponding `--json` schema changes for `apply`, `grep`, `subgraph`, and `serve`) was already delivered by prior features (predicate-first node model, predicate-based merge policies, predicate-based rendering). What remains, confirmed against the current codebase, is a narrower terminology gap: a handful of command flags, help text, a warning message, and the MCP filter's wire field/table column still say "kind" even though the underlying data and every other surface already say "type". This specification covers closing that terminology gap, per explicit user direction after this finding was surfaced.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Filter by type using consistent CLI vocabulary (Priority: P1)

A graph maintainer runs `arc grep` or `arc subgraph` to narrow results to nodes of a particular type. They expect the flag they use for this to be named consistently with the vocabulary arc already uses everywhere else (the node's `@type`, the `--json` output's `type` field, the plain-text output's type column) — not a leftover synonym that suggests it might mean something different.

**Why this priority**: This is the most frequently used surface (every filtered `grep`/`subgraph` invocation touches it) and the most visible inconsistency — a maintainer who has internalized "type" from reading node files and JSON output stumbles when the flag itself still says "kind".

**Independent Test**: Run `arc grep --type source <pattern>` and `arc subgraph <node> --type source` against a fixture graph and confirm both commands accept the flag and filter identically to how the old `--kind` flag filtered before the rename.

**Acceptance Scenarios**:

1. **Given** a graph with nodes of several types, **When** a maintainer runs `arc grep --type source <pattern>`, **Then** only matches from nodes whose type is `source` are returned, with the same OR-across-repeated-flag semantics the filter had before the rename.
2. **Given** the same graph, **When** a maintainer runs `arc subgraph <node> --type source`, **Then** the extracted subgraph is narrowed to nodes of type `source`, identically to how the equivalent `--kind` invocation behaved before the rename.
3. **Given** a maintainer runs `arc grep --help` or `arc subgraph --help`, **When** they read the flag description and usage examples, **Then** every reference to this filter — the flag name, its description, and any example invocation — says "type", not "kind".

---

### User Story 2 - Consistent terminology in apply's reporting (Priority: P2)

A graph maintainer applies a patch and reads `arc apply`'s output, including any warning about an unrecognized node type. They expect the wording to match the vocabulary the rest of the toolchain uses.

**Why this priority**: Lower traffic than filtering (the warning only fires for unrecognized types), but a maintainer debugging a schema mismatch specifically needs this message to be unambiguous, and "kind" here could be misread as referring to a different concept than the type system they already know about.

**Independent Test**: Apply a patch introducing a node of a type not yet registered in the graph's schema and confirm the resulting warning text says "type", not "kind".

**Acceptance Scenarios**:

1. **Given** a patch introduces a node whose type is not yet present in the graph's resolved schema index, **When** `arc apply` runs, **Then** the warning it reports for that node reads "... is not a recognized node type for this graph ..." (not "kind").

---

### User Story 3 - Consistent vocabulary across the MCP interface (Priority: P2)

A tool integrator connects an MCP client to `arc serve` and uses the `node_grep` tool to search node content, optionally narrowing by type. They expect the filter's wire field name and the result table's column header to use the same "type" vocabulary as everything else the server and the rest of arc expose.

**Why this priority**: MCP clients are external tools coded against arc's wire contract; a field still named `kind` next to a `type`-only data model is a durable point of confusion for anyone integrating fresh, and it is the last exposed surface where the old term survives.

**Independent Test**: Call `node_grep` with a type-restriction filter using the field name `type` and confirm it narrows results; inspect the returned match table and confirm its column header reads "type".

**Acceptance Scenarios**:

1. **Given** an MCP client calls `node_grep` with a filter narrowing to a specific node type, **When** the filter is expressed under the field name `type`, **Then** the tool restricts matches to nodes of that type.
2. **Given** any `node_grep` call that returns matches, **When** the result table is rendered, **Then** its header row labels the type column "type", not "kind".

### Edge Cases

- A maintainer or script still invokes the old `--kind` flag on `arc grep` or `arc subgraph` after this change ships: the command MUST fail with a standard "unknown flag" error, the same way any other unrecognized flag would — no silent alias and no deprecation grace period, consistent with how this project has already treated other breaking, pre-1.0 CLI/JSON surface changes.
- An MCP client still sends its `node_grep` filter under the old field name `kind` after this change ships: the call MUST fail with a clear tool-error identifying `kind` as an unrecognized filter property — the MCP server validates tool arguments against a generated JSON Schema that disallows unlisted properties (`additionalProperties: false`), the same strict-by-default posture every `node_grep`/`node_get`/`subgraph_get` argument object already has, so a stale `kind` key is rejected rather than silently dropped — no alias, mirroring the CLI's own no-alias treatment (see the `--kind` edge case above) — this must be called out in the change's release notes so integrators know to update.
- A warning or help string references the type concept in a sentence structure where a mechanical word swap would read awkwardly (e.g., pluralization, article agreement): the surrounding sentence MUST be adjusted for grammatical correctness, not just have "kind" substituted verbatim.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `arc grep` MUST expose its type-restriction filter as a flag named `--type` (replacing `--kind`), preserving the existing repeatable/OR-combined matching behavior unchanged.
- **FR-002**: `arc subgraph` MUST expose its type-restriction filter as a flag named `--type` (replacing `--kind`), preserving the existing repeatable/OR-combined matching behavior unchanged.
- **FR-003**: `arc grep`'s and `arc subgraph`'s help text (flag description, long description, and usage examples) MUST refer to this filter and the value it matches as "type" everywhere "kind" previously appeared.
- **FR-004**: `arc subgraph`'s `--stubs` flag help text MUST describe the identity fields a placeholder stub node carries as "type and id" (not "kind and id").
- **FR-005**: `arc apply`'s warning message for a node whose type is not yet recognized by the graph's schema MUST say "... is not a recognized node type for this graph ..." (not "kind").
- **FR-006**: `arc serve`'s `node_grep` MCP tool MUST accept its optional type-restriction filter criterion under the field name `type` (replacing `kind`), preserving existing matching behavior unchanged.
- **FR-007**: `arc serve`'s `node_grep` MCP tool's result table MUST label its type column "type" (replacing "kind").
- **FR-008**: For every command covered by FR-001 through FR-007, the rename MUST be applied consistently everywhere the concept appears for that command — no single command may mix "kind" and "type" vocabulary for what is the same underlying node-type concept.
- **FR-009**: None of the renames in FR-001 through FR-007 MUST alter which nodes match a filter, what gets reported, or any other observable filtering/reporting behavior — only the name used to invoke or describe that behavior changes.
- **FR-010**: After the flag renames in FR-001/FR-002 ship, invoking `arc grep` or `arc subgraph` with the old `--kind` flag MUST fail with the same error a user would get for any other unrecognized flag — no alias, no warning-then-continue behavior.
- **FR-011**: Documentation of this change (e.g. release/changelog notes) MUST explicitly call out the `--kind` → `--type` flag rename and the MCP `node_grep` filter's `kind` → `type` field rename as breaking, so integrators relying on either old name know to update.

### Key Entities

- **Type filter criterion**: The type-restriction condition already present in `arc grep`, `arc subgraph`, and `arc serve`'s `node_grep` filtering — unchanged in behavior by this feature, renamed in every surface where it is invoked, described, or displayed so that "type" is the single term used across the whole toolchain.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `arc grep --help`, `arc subgraph --help`, and inspecting `arc apply`'s unrecognized-type warning text yields zero occurrences of the word "kind" describing the node-type concept; every occurrence reads "type".
- **SC-002**: Inspecting `arc serve`'s `node_grep` tool's filter field names and its rendered result table yields zero occurrences of "kind"; both read "type".
- **SC-003**: Every filtering and reporting scenario that passed before this change (same inputs, same matched nodes, same counts) continues to pass identically after the rename, confirming the change altered vocabulary only, not behavior.
- **SC-004**: A user who has read arc's node file format, its `--json` output, and any one of `grep`/`subgraph`/`serve` no longer needs to learn a second term ("kind") to use the remaining commands — one term covers the concept everywhere.

## Assumptions

- This is a breaking, pre-1.0 rename: `--kind` and the MCP `kind` filter field are renamed in place, not kept as backward-compatible aliases alongside the new names — consistent with how this project has already treated other breaking CLI/JSON surface changes (e.g. the earlier node-shape `--json` schema change).
- The underlying graph file format's own `@id`/`@type` front-matter vocabulary, and the `--json` output field names produced by `apply`, `grep`, and `subgraph`, are already correct (they already say "type") and are unaffected by this feature — only command flags, help/usage text, a warning message, and the MCP filter's wire field name and table header change.
- No new CLI flags, MCP fields, or commands are introduced; existing "kind"-named surfaces are renamed to "type" in place, per this feature's explicit out-of-scope boundary.
- Internal, non-user-facing implementation details (Go identifier names, source code comments, internal struct field names not exposed through a flag, JSON field, or printed label) are not covered by this specification's requirements — only surfaces a CLI user or MCP client actually sees or invokes are in scope.
