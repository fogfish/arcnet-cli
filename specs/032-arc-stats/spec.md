# Feature Specification: Graph Statistics (`arc stats`)

**Feature Branch**: `032-arc-stats`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "`arc stats` summary about the graph: total nodes and edges, node count by class, broken link count, source ingestion by year. Usage of verbose mode adds detailed statistic with total edges by predicate, source ingestion rate by month and other graph statistic."

## Clarifications

### Session 2026-09-05

- Q: What counts toward "total nodes" — every node file, content nodes only, or content + timeline with schema separate? → A: Every node file (content + timeline + schema), so the per-type breakdown always sums to the total. Additionally: the folder layout is NOT fixed — a user may add folders and move nodes between them — so no statistic may be derived from a node's location.
- Q: Does `--json` emit detail figures regardless of `--verbose`, or mirror the requested verbosity? → A: Mirror verbosity — machine-readable output carries summary figures only by default and adds the detail figures when verbose is requested. Verbosity therefore gates what is computed, not merely what is rendered; detail fields are absent from the structure entirely when not requested.
- Q: Should the command exit non-zero when the graph has broken links, so it can gate CI? → A: No — always exit 0 on a completed scan. Reported problems are results, not failures; gating on graph health remains `arc lint`'s role. Only a genuine operational error exits non-zero.
- Q: Does the broken-link count cover inline prose references as well as structural edges, and how are duplicates counted? → A: Match the conformance check exactly — it spans both structural edges and inline prose references, counted as distinct unresolved targets per node. This is deliberately asymmetric with the edge total (occurrences, structural only); the report states both counting rules.
- Q: Is the node breakdown labelled by "class" or by "type"? → A: By **type** — it groups nodes by the `@type` each node declares. "Class" is reserved for its established meaning, the schema node that registers a type's vocabulary, and appears only in the schema-coverage figures. The two terms are kept distinct throughout the spec.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the shape and health of a graph at a glance (Priority: P1)

A user has a knowledge graph that has grown over time through repeated ingestion runs. They open the graph — perhaps for the first time, perhaps after months away — and want one command that answers "how big is this, what is in it, is it healthy, and how far back does it reach?" without reading a single node file or running a full conformance check.

**Why this priority**: This is the whole point of the feature. A single summary — total nodes and edges, the type breakdown, the broken-link count, and the per-year ingestion coverage — is what turns an opaque folder of Markdown into something a person can reason about. Every other part of this feature is additional detail layered on top of this one screen.

**Independent Test**: Can be fully tested by running the command against a graph with a known, fixed set of nodes, edges, types, unresolved links, and timeline years, and confirming every reported figure matches the known expected value exactly.

**Acceptance Scenarios**:

1. **Given** an initialized graph containing nodes of several types, **When** the user runs the statistics command, **Then** the tool reports the total node count, the total edge count, a per-type node count breakdown, the broken-link count, and the number of sources ingested in each year.
2. **Given** a graph in which the per-type breakdown is reported, **When** the user reads the breakdown, **Then** the individual type counts sum exactly to the reported total node count, so no node is unaccounted for.
3. **Given** a graph containing links whose targets do not exist, **When** the user runs the statistics command, **Then** the reported broken-link count equals the number of unresolved-link violations the graph's conformance check reports for the same graph.
4. **Given** a freshly initialized, empty graph, **When** the user runs the statistics command, **Then** the tool reports zero counts across every category and succeeds, rather than failing or reporting nothing.
5. **Given** a directory that is not an initialized graph, **When** the user runs the statistics command, **Then** the tool refuses with a clear message, consistent with how the graph's other commands refuse, rather than reporting an empty graph.

---

### User Story 2 - Investigate the graph in depth (Priority: P2)

Having seen the summary, a user wants to understand the graph's internals: which relationship types actually carry its structure, how ingestion is distributed month by month, whether parts of the graph are disconnected, where the graph concentrates, and whether its vocabulary is fully declared. They re-run the same command in verbose mode.

**Why this priority**: Real value, but only once the summary exists and the user has a reason to dig further. The summary alone is a usable deliverable; the deep dive is not usable without it.

**Independent Test**: Can be fully tested by running the command in verbose mode against a graph with a known distribution of predicates, monthly ingestion periods, orphan nodes, reference counts, declared schema, and inline references, and confirming each detailed figure matches the known expected value.

**Acceptance Scenarios**:

1. **Given** a graph whose edges use several distinct predicates, **When** the user runs the statistics command in verbose mode, **Then** the tool additionally reports an edge count for each predicate in use, and those counts sum exactly to the total edge count reported in the summary.
2. **Given** a graph with sources ingested across several months, **When** the user runs the statistics command in verbose mode, **Then** the tool additionally reports the ingestion count for each month, and the monthly counts belonging to a given year sum exactly to that year's count in the summary.
3. **Given** a graph containing nodes with no incoming and no outgoing links, **When** the user runs the statistics command in verbose mode, **Then** the tool reports how many such disconnected nodes exist.
4. **Given** a graph whose broken-link count is non-zero, **When** the user runs the statistics command in verbose mode, **Then** the tool additionally names each distinct unresolved link target behind that count.
5. **Given** a graph in which some nodes are referenced far more than others, **When** the user runs the statistics command in verbose mode, **Then** the tool reports the average and median number of outgoing edges per node, and lists the most-referenced nodes with their incoming reference counts.
6. **Given** a graph declaring Class and Property schema nodes, some unused, and whose nodes use predicates that are not declared, **When** the user runs the statistics command in verbose mode, **Then** the tool reports how many Class and Property schema nodes the graph declares, how many of each are actually used, and which predicates appear in nodes without being declared.
7. **Given** a graph whose nodes carry inline references and attributes, **When** the user runs the statistics command in verbose mode, **Then** the tool reports the inline reference count separately from the structural edge count, the total attribute count, and how many nodes carry no publication date.
8. **Given** any graph, **When** the user runs the statistics command without verbose mode, **Then** none of the verbose-only figures appear in the output.

---

### User Story 3 - Track graph growth from a script or pipeline (Priority: P3)

A user wants to record the graph's statistics over time — in continuous integration, in a dashboard, or in a scheduled job — and needs the figures in a form a program can consume without parsing human-readable text.

**Why this priority**: Extends the same computation to an automated consumer. Valuable, but the interactive summary must exist and be correct first, and a human reading the terminal output is the primary case.

**Independent Test**: Can be fully tested by running the command in machine-readable mode against a known graph and confirming the emitted structure parses successfully and carries exactly the same figures as the human-readable output for that graph.

**Acceptance Scenarios**:

1. **Given** any graph, **When** the user runs the statistics command in machine-readable mode, **Then** the tool emits a single well-formed structured document carrying every summary figure, and nothing else is written to standard output.
2. **Given** any graph, **When** the user runs the statistics command in machine-readable and verbose modes together, **Then** the emitted structure additionally carries every verbose figure.
3. **Given** two runs against an unchanged graph, **When** the user compares the two emitted structures, **Then** they are byte-for-byte identical, so the output can be diffed across time.

---

### Edge Cases

- What happens when the graph is freshly initialized and completely empty? Every count is reported as zero and the command succeeds; it does not fail, and it does not print an empty report.
- What happens when the target directory is not an initialized graph? The command refuses with a clear message, consistent with the graph's other commands, rather than reporting a graph full of zeros.
- What happens when a node file's front-matter is missing, malformed, or unparseable? The command does not crash and does not abort the run; it counts the file as unreadable, reports the unreadable count distinctly from the node total, and completes the rest of the statistics.
- What happens when a node file sits in a folder the graph's default layout does not define, or has been moved from one folder to another? Every figure is unchanged: node discovery walks the whole graph and every node is grouped by the type it itself declares, so folder layout never influences a statistic.
- What happens when a node declares a type for which the graph has no Class schema node? The type still appears in the per-type breakdown under the name the node declares, so the breakdown always reflects what is actually in the graph; verbose mode identifies it as undeclared.
- What happens when the graph has no ingestion periods recorded at all? The per-year breakdown is reported as empty rather than omitted, so the reader can tell "no ingestion recorded" apart from "this figure was not computed."
- What happens when an ingestion period record contains an entry the tool cannot interpret? That entry is skipped and the count of skipped entries is reported, rather than being silently absorbed into or excluded from the period's count.
- What happens when a link points at a stub node — one carrying no content beyond its identity and type? The target exists, so the link is not broken; the stub is counted as a node and reported in the stub count in verbose mode.
- What happens when a link points at the node containing it? It is counted as an edge like any other and is not treated as broken; the node is not counted as disconnected.
- What happens when two nodes share the same identity? Both files are counted, and verbose mode reports the colliding identity, since counts derived from identity would otherwise be silently wrong.
- What happens when several nodes tie for the most-referenced position? Ordering among tied entries is deterministic and stable across runs, so repeated runs of the same graph produce identical output.
- What happens when the graph is very large (many thousands of nodes)? The command completes in a single pass and reports every figure; performance at that scale is addressed by SC-004.
- What happens when a program consumes the machine-readable output and the run was not verbose? The detailed figures are absent from the structure rather than present-and-empty, so the consumer can tell "not requested" from "requested and genuinely zero"; a consumer that needs the detail must request verbose output.
- What happens when the user asks for quiet output? Progress reporting is suppressed while the statistics report itself is still emitted, since the report is the command's result, not progress.

## Requirements *(mandatory)*

### Functional Requirements

#### Summary statistics (default output)

- **FR-001**: The tool MUST provide a statistics command that reads the graph in the current working directory and reports a summary of it, without modifying the graph in any way. It MUST discover node files wherever they reside in the graph, including folders the user has added, and MUST NOT assume any fixed folder layout.
- **FR-002**: The tool MUST report the total number of nodes in the graph, counting every node file the graph contains regardless of which folder holds it — content nodes, the derived timeline index, and schema documents alike.
- **FR-003**: The tool MUST report the total number of structural edges in the graph, counted as link occurrences, so a node linking twice to the same target contributes two edges.
- **FR-004**: The tool MUST report a per-type node count covering every type actually present in the graph, and these counts MUST sum exactly to the reported total node count (FR-002).
- **FR-004a**: The tool MUST derive every node's type from the type that node itself declares, never from the folder the node file resides in, so that moving a node between folders — or introducing a new folder — changes no reported figure.
- **FR-005**: The tool MUST order the per-type breakdown by descending count, with a deterministic, stable tie-break so repeated runs against an unchanged graph produce identical output.
- **FR-006**: The tool MUST report the number of broken links in the graph, where a broken link is a reference whose target does not exist as a node. The count MUST span both structural edges and inline prose references, and MUST count each distinct unresolved target once per node that references it — identical to how the graph's conformance check counts unresolved-link violations, so the two commands never disagree on the same graph.
- **FR-006a**: The tool MUST make its two counting rules legible in the report itself, since they deliberately differ: edges are counted as occurrences over structural links only (FR-003), while broken links are counted as distinct targets per node over structural links and inline references alike (FR-006). A reader MUST NOT have to infer why a graph can report few edges and many broken links.
- **FR-007**: The tool MUST report, for each year the graph has ingestion recorded for, how many sources were ingested in that year, deriving these figures from the graph's own yearly ingestion period records rather than by re-deriving them from individual source nodes.
- **FR-008**: The tool MUST report years in chronological order and MUST NOT omit a year that has a recorded period, even when its count is zero.
- **FR-009**: The tool MUST report a count of node files it could not read or interpret, distinctly from the total node count, and MUST complete the remaining statistics rather than aborting on the first such file.
- **FR-010**: The tool MUST succeed and report zero counts for an initialized but empty graph.
- **FR-011**: The tool MUST refuse to run, with a clear message consistent with the graph's other commands, when the target directory is not an initialized graph.

#### Detailed statistics (verbose output)

- **FR-012**: When verbose output is requested, the tool MUST additionally report an edge count for each distinct predicate in use, and those counts MUST sum exactly to the total edge count (FR-003).
- **FR-013**: When verbose output is requested, the tool MUST additionally report, for each month the graph has ingestion recorded for, how many sources were ingested in that month, derived from the graph's own monthly ingestion period records, and the months belonging to a year MUST sum to that year's summary figure (FR-007).
- **FR-014**: When verbose output is requested, the tool MUST additionally report connectivity health: the number of disconnected nodes (no incoming and no outgoing links), the number of stub nodes (nodes carrying no content beyond their identity and type), and each unresolved link target behind the broken-link count (FR-006), listed once per distinct target graph-wide with the number of nodes referencing it — so the per-target figures sum exactly to FR-006's total, which counts a target once per referencing node.
- **FR-015**: When verbose output is requested, the tool MUST additionally report the average and median number of outgoing edges per node — computed over every node counted by FR-002, including nodes with no outgoing edges — and a ranked list of the ten most-referenced nodes with their incoming reference counts, listing fewer only when the graph holds fewer nodes.
- **FR-016**: When verbose output is requested, the tool MUST additionally report schema coverage: how many Class and how many Property schema nodes the graph declares, how many of each are actually used — a Class by at least one node declaring that type, a Property by at least one edge or attribute using it — and every predicate used by a node without being declared.
- **FR-017**: When verbose output is requested, the tool MUST additionally report content volume: the number of inline references (reported separately from structural edges, which they are not), the total number of attribute values, and the number of nodes carrying no publication date.
- **FR-018**: The tool MUST NOT include any detailed figure in its default output, in any form — neither rendered in human-readable output nor present as an empty or null placeholder in machine-readable output. Verbosity gates what the tool computes, not only what it renders, so the default run does no work to produce figures it will not report.
- **FR-019**: Every ranked or grouped list the tool reports MUST have a deterministic, stable ordering, including among tied entries, so repeated runs against an unchanged graph produce identical output.

#### Output and behavior

- **FR-020**: The tool MUST support machine-readable structured output carrying exactly the same figures as the human-readable output for the same graph and the same verbosity, and MUST write nothing else to standard output in that mode.
- **FR-020a**: Machine-readable output MUST mirror the requested verbosity: summary figures only by default, summary plus detailed figures when verbose output is requested. The detailed figures MUST be omitted from the structure altogether when not requested, so a consumer can distinguish "not requested" from "requested and empty" without ambiguity.
- **FR-021**: The tool MUST produce identical machine-readable output across repeated runs against an unchanged graph, so the output can be compared over time.
- **FR-022**: The tool MUST suppress progress reporting when quiet output is requested, while still emitting the statistics report itself.
- **FR-023**: The tool MUST be strictly read-only: it MUST NOT create, modify, or delete any file in the graph, and MUST NOT alter version-control state.
- **FR-024**: The tool MUST exit successfully whenever it completed its scan, including when the graph it describes has broken links, unreadable node files, or undeclared predicates, since reporting a problem is a successful outcome for this command and gating on graph health remains the conformance check's job. The tool MUST exit non-zero only when it could not produce a report at all — the target is not a graph, or the graph could not be read.

### Key Entities

- **Graph Statistics Report**: The complete result of one run — the summary figures always present, plus the detailed figures present only in verbose mode. This is the value rendered to the user in human-readable or machine-readable form.
- **Type Breakdown Entry**: One type name paired with the number of nodes declaring it, plus, in verbose mode, whether a Class schema node declares that type. Distinct from the schema-coverage figures (FR-016), which count Class and Property schema nodes themselves.
- **Predicate Breakdown Entry**: One predicate name paired with the number of edges using it, plus whether that predicate is declared in the graph's schema.
- **Ingestion Period Entry**: One period — a year in the summary, a month in verbose mode — paired with the number of sources recorded as ingested in it, read from the graph's own ingestion period records.
- **Reference Rank Entry**: One node identity paired with the number of incoming references to it, used for the most-referenced ranking.
- **Unresolved Target**: One link target that no node in the graph provides, paired with the number of nodes referencing it. Underlies the broken-link count and is listed individually in verbose mode.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user unfamiliar with a given graph can state its size, its composition by type, whether it has broken links, and the years it covers, within 30 seconds of running one command, without opening any file in the graph.
- **SC-002**: Every reported figure is exactly correct for a graph with a known composition: 100% of the summary and verbose figures match the independently known expected values, with no approximation or sampling.
- **SC-003**: The broken-link count agrees with the graph's conformance check on 100% of graphs, so a user never has to reconcile two different answers to the same question.
- **SC-004**: The command completes in under 5 seconds on a graph of 10,000 nodes, and its running time grows no faster than proportionally with graph size.
- **SC-005**: Repeated runs against an unchanged graph produce identical output 100% of the time, in both human-readable and machine-readable form, so the output is usable as a tracked artifact.
- **SC-006**: A graph containing unreadable or malformed node files still produces a complete report: the command reports every figure it can compute plus the count of files it could not read, and never fails to produce a report because of such a file.
- **SC-007**: The command never modifies the graph: after any run, the graph's files and version-control state are byte-for-byte unchanged.

## Assumptions

- Ingestion figures are read from the graph's own yearly and monthly ingestion period records — the derived chronological index the graph already maintains — rather than by re-deriving them from each source node's publication date. This was the user's explicit direction, and it keeps `arc stats` and the graph's own timeline index in agreement by construction; a discrepancy between the index and the underlying nodes is the conformance check's concern, not this command's.
- "Total nodes" counts every node file in the graph, and the per-type breakdown enumerates every type observed — including the graph's reserved index type and its schema types. This keeps the breakdown summing to the total with no hidden buckets; the breakdown itself shows the content/index/schema split, so nothing is obscured by the choice. *(Clarified 2026-09-05.)*
- The graph's folder layout is a convention, not a constraint: users add folders and move nodes between them. Every statistic is therefore derived from node content — the type a node declares, the links it carries — and never from a node file's path. Node discovery walks the whole graph rather than a fixed list of folders. *(Clarified 2026-09-05.)*
- "Edges" means the graph's structural, navigable links. Inline references embedded in prose are a different kind of reference and are counted separately, reported only in verbose mode, and never folded into the edge total.
- Broken links are counted per distinct unresolved target per node and span inline prose references as well as structural edges, matching the existing conformance check's counting rule exactly, so the two commands report the same number for the same graph. This is knowingly asymmetric with the edge total, which counts occurrences of structural links only; FR-006a requires the report to make both rules explicit rather than leaving the difference to be inferred. *(Clarified 2026-09-05.)*
- The command operates on the graph in the current working directory and takes no positional arguments, consistent with the graph's other read-only commands.
- Verbosity, quiet mode, and machine-readable output are requested through the tool's existing global output controls rather than through options specific to this command. Verbosity is passed through to the statistics computation itself, so the detailed figures are neither computed nor emitted unless asked for, and the default run stays cheap. *(Clarified 2026-09-05.)*
- The most-referenced ranking is truncated to ten entries for readability; the full distribution is out of scope for this feature. Ten is a readability default, not a user-derived requirement — it is a safe thing to revisit during planning if the rendered report argues for another figure.
- The command reports the graph's current state only. Trend analysis, comparison against a previous run, and history-derived figures such as ingestion throughput over commit history are out of scope.
- A node's type is taken as the node declares it, whether or not a Class schema node declares that type, so the breakdown always describes what is actually present. "Type" and "Class" are kept distinct throughout: a *type* is what a node declares about itself and is what the per-type breakdown groups by; a *Class* is the schema node that registers a type's vocabulary and is counted only in the schema-coverage figures. *(Clarified 2026-09-05.)*
