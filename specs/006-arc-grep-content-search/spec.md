# Feature Specification: Search Graph Content by Pattern (`arc grep`)

**Feature Branch**: `006-arc-grep-content-search`

**Created**: 2026-07-04

**Status**: Draft

**Input**: User description: "`arc grep [<filter>] <pattern>` — scan nodes matching the filter (see Filtering) for lines matching the regexp `<pattern>`; print `<kind>  <id>  <line-number>  <matched line>`, one match per output line; without a filter, scans every node file; suitable for piping to standard tools."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find every occurrence of a term across the whole graph (Priority: P1)

A user wants to know everywhere a specific word, phrase, or code pattern appears across the entire graph — for example, checking whether a term is used anywhere before renaming it, or locating every node that mentions a particular protocol. They run the search with no filter and expect every node file in the graph to be scanned, with each matching line reported alongside enough identifying information (which node, which line) to jump straight to it.

**Why this priority**: This is the core value of the command — a single, fast way to answer "where does X appear in this graph?" without writing a custom script or manually opening files. Every other capability (filtering, piping) builds on this base scan.

**Independent Test**: Can be fully tested by creating a graph with a known term appearing in a known set of nodes and lines, running the search with no filter, and confirming every occurrence is reported with the correct node identity and line number, and no occurrence is missed or duplicated.

**Acceptance Scenarios**:

1. **Given** a graph where a pattern appears on specific lines in several different node files, **When** the user runs the search with no filter, **Then** every matching line across every node file is reported, each as its own output line naming the node's kind, id, line number, and the matched line's text.
2. **Given** a graph where the pattern does not appear anywhere, **When** the user runs the search, **Then** the tool produces no matching output and indicates via its exit status that nothing was found.
3. **Given** a single node file where the pattern appears on more than one line, **When** the user runs the search, **Then** each matching line is reported as a separate output line with its own correct line number — the node is not reported only once.

---

### User Story 2 - Narrow the search to a subset of nodes (Priority: P2)

A user only cares about occurrences within a specific slice of the graph — for example, only within `source` nodes, or only within nodes carrying a particular tag or attribute. They apply a filter alongside the pattern so the scan is restricted to that subset, using the same filter syntax already used by other graph-wide commands.

**Why this priority**: Narrowing scope is a common refinement once the base search works, and reusing the graph's existing, familiar filter syntax keeps the command consistent with the rest of the CLI. It is not needed for the command to deliver its core value, hence P2.

**Independent Test**: Can be fully tested by creating a graph with the same pattern present in nodes both inside and outside a chosen filter (e.g. a specific kind), running the search with that filter applied, and confirming only matches from nodes inside the filtered subset are reported.

**Acceptance Scenarios**:

1. **Given** a graph where the pattern appears in nodes of multiple kinds, **When** the user runs the search with a kind filter, **Then** only matches from nodes of the specified kind(s) are reported.
2. **Given** a graph where the pattern appears in nodes both with and without a specific tag, **When** the user runs the search with a tag filter, **Then** only matches from nodes carrying that tag are reported.
3. **Given** a filter that combines a kind condition and an attribute condition, **When** the user runs the search, **Then** only nodes satisfying all the combined conditions are scanned, consistent with the graph's general filter composition rules (see Filtering).
4. **Given** a filter that matches zero nodes, **When** the user runs the search, **Then** the tool produces no matching output and indicates via its exit status that nothing was found, without error.

---

### User Story 3 - Pipe search results into other command-line tools (Priority: P3)

A user or script wants to feed the search results into standard Unix tools — `wc -l` to count matches, `cut`/`awk` to extract a column, `sort`/`uniq` to deduplicate, or `xargs` to act on the matched nodes. They expect a stable, whitespace-delimited, one-match-per-line output with no decoration that would need special parsing.

**Why this priority**: This is what makes the command composable with the rest of a user's toolchain rather than a dead end, but it depends on the output format already established by P1 and P2 — it is a consumption pattern of the same output, not new scanning behavior.

**Independent Test**: Can be fully tested by running the search against a graph with a known number of matches and piping the output through `wc -l`, confirming the count equals the known number of matching lines, and through `cut`/`awk` on the fixed field positions, confirming each field extracts cleanly.

**Acceptance Scenarios**:

1. **Given** a graph with a known, fixed number of matching lines, **When** the user pipes the search output through a line-counting tool, **Then** the count equals exactly the number of matching lines, with no extra header, footer, or summary lines mixed into the output.
2. **Given** search output, **When** a standard field-extraction tool splits each line on whitespace, **Then** the first field is the node's kind, the second is the node's id, the third is the line number, and the remainder of the line is the matched line's text.
3. **Given** a matched line in a node file that itself contains a literal newline is not possible (a single line cannot contain a newline), **When** the tool reports that match, **Then** the output row remains exactly one line, preserving the one-match-per-output-line guarantee.

---

### Edge Cases

- What happens when `<pattern>` is not a valid regular expression? The tool must report a clear error identifying the pattern as invalid and exit with a failing status, without scanning any nodes.
- What happens when the pattern matches multiple times within the same line? The tool must report that line once, not once per match within the line.
- What happens when the filter matches zero nodes, or the pattern matches nothing across all scanned nodes? The tool must exit cleanly with no matches reported and an exit status distinguishing "ran successfully but found nothing" from an error.
- What happens when a node file cannot be read (e.g. permission error) or contains malformed front-matter? The tool must still attempt to scan the file's content for line matches and must not abort the entire run because one file could not be fully parsed; if the file cannot be read at all, the tool reports that specific file as unreadable and continues with the rest of the graph.
- What happens when the target directory is not an initialized graph? The tool must refuse to run and report this clearly, consistent with other graph commands, rather than attempting a scan.
- What happens when the same node's `<id>` cannot be determined for a candidate file (e.g. the file is not a recognized node)? That file is excluded from the scan, consistent with how filtering already restricts scanning to recognized nodes.
- What happens when the graph is very large (many thousands of nodes)? The tool must still complete and report every match in a single run; performance at that scale is addressed by SC-004.
- What happens when no `<pattern>` argument is supplied? The tool must report a clear usage error and exit with a failing status rather than scanning with an empty or implicit pattern.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST accept a required `<pattern>` argument, interpreted as a regular expression, and MUST accept an optional filter argument preceding it.
- **FR-002**: When no filter is given, the tool MUST scan every node file in the graph for lines matching `<pattern>`.
- **FR-003**: When a filter is given, the tool MUST restrict scanning to only the nodes matching that filter, using the same filter syntax and composition rules (kind, tag, attribute, AND/OR semantics) defined for other graph-wide commands (see Filtering).
- **FR-004**: For each line within a scanned node's file that matches `<pattern>`, the tool MUST print exactly one output line containing, in order: the node's `kind`, the node's `id`, the 1-based line number within the file, and the full text of the matching line.
- **FR-005**: The tool MUST report every matching line across every scanned node in a single run — it MUST NOT stop at the first match found, and MUST NOT report the same matching line more than once even if `<pattern>` matches multiple times within that line.
- **FR-006**: The tool's output MUST be one match per output line, with fields ordered and delimited consistently so it can be parsed by standard field-extraction tools without special-casing.
- **FR-007**: The tool MUST NOT print any header, footer, summary, or decorative output mixed into the matching lines — output consists solely of match rows (and, if applicable, distinctly-formatted error/warning lines directed to a separate error stream).
- **FR-008**: When `<pattern>` is not a valid regular expression, the tool MUST report a clear error and exit with a failing status without scanning any nodes.
- **FR-009**: When zero matches are found (whether because the filter matched no nodes or because `<pattern>` matched nothing), the tool MUST exit successfully with no matches reported, distinguishing this outcome from an error condition via its exit status.
- **FR-010**: The tool MUST make no changes to the graph's files or git history — searching is a read-only, non-mutating operation.
- **FR-011**: The tool MUST refuse to run, and MUST report this clearly instead of scanning, when the target directory is not an initialized graph.
- **FR-012**: When an individual node file cannot be read, the tool MUST report that file as unreadable and continue scanning the remaining nodes rather than aborting the whole run.

### Key Entities

- **Match**: One reported occurrence of `<pattern>` on a single line within a single node's file; carries the node's kind, the node's id, the 1-based line number, and the full text of the matched line.
- **Filter**: The optional, composable node-selection criteria (kind, tag, attribute) that restricts which nodes are scanned, shared with other graph-wide commands (see Filtering).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can find every occurrence of a term across an entire graph, with exact node and line identification, from a single command run — no manual file-by-file review or custom scripting required.
- **SC-002**: 100% of known, deliberately placed pattern occurrences in a test graph are reported, with no occurrence missed and no occurrence reported more than once.
- **SC-003**: Applying a filter reduces the reported matches to exactly those within the filtered subset, with 100% accuracy against a graph containing matches both inside and outside the filter.
- **SC-004**: A search across a graph of several thousand nodes completes in under 10 seconds.
- **SC-005**: Output piped into standard line-counting and field-extraction tools produces correct counts and correctly split fields in 100% of tested cases, with no parsing workaround needed.
- **SC-006**: Running the search command never modifies any graph file or git history — verified by the graph's state being byte-for-byte identical before and after any run.

## Assumptions

- The pattern is matched against each line's raw text, including both front-matter lines and body lines of a node's file — a node's file is treated as a flat sequence of lines for search purposes, consistent with the command being "suitable for piping to standard tools" like a conventional line-oriented grep.
- The regular expression dialect follows the same convention already used elsewhere in the CLI for pattern matching (e.g. the `--attr <name>~=<pattern>` attribute filter), so users do not need to learn a second regex flavor.
- Matching is case-sensitive by default, consistent with standard `grep` behavior; case-insensitive matching, if ever needed, would be expressed within the pattern itself.
- Output field separation follows the same convention as other listing commands in the CLI (e.g. `arc list`, `arc lint`) — whitespace-delimited columns ending in a free-text final field — so existing muscle memory and scripts around those commands carry over.
- Exit status follows this codebase's established convention (`arc lint`, `arc apply`): a run that completes and prints its full result — whether or not any match was found — exits via the same path the human/JSON output was already written through, and only a refusal to run at all (invalid pattern, uninitialized graph, i/o failure before any output was produced) is reported as a distinct error with its own message. A script distinguishes "ran, found nothing" from "refused to run" by the presence of an error message, not by a third exit-code value, consistent with how every other command in this CLI signals a finding versus a refusal.
