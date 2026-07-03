# Feature Specification: Validate Graph Conformance (`arc lint`)

**Feature Branch**: `004-arc-lint`

**Created**: 2026-07-03

**Status**: Draft

**Input**: User description: "`arc lint` — run the full CORE §14 checklist across every node and report violations with file path and line number for each: valid YAML front-matter and `kind` field; unique basenames (CORE §3.2); every `[[link]]` resolves to an existing basename; `source` citekey `id` equals its basename (CORE §6.2); `entity` four-word decoded Sowa `category` (CORE §9.2.1); derived nodes link back to at least one `source` (CORE §3.4); predicates are camelCase and registered in `_meta/predicates.md` (CORE §7.3); citations use a registered `cito:`-aligned predicate (CORE §8); each document is exactly one `graph(ingest):` commit (CORE §11.1); extension kind conformance per the kind's profile checklist and graph nodes does not have any active merge conflicts."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confirm a graph is conformant before trusting it (Priority: P1)

A user maintains a graph that has been growing through repeated `arc apply` runs from multiple extraction pipelines over time. Before relying on the graph for downstream work (querying, publishing, serving), they want a single command that checks every node against the format's own conformance rules and tells them, with exact file and line references, anywhere the graph has drifted from spec.

**Why this priority**: This is the only capability that lets a user trust an arbitrarily-grown graph without manually re-reading every file. Every other planned capability (query, serve) implicitly assumes the graph it operates on is well-formed; without a lint command, that assumption is never actually checked.

**Independent Test**: Can be fully tested by running the command against a graph containing a deliberately introduced violation of each checked rule, and confirming the command's output names every violation with its file path and line number, and against a graph with no violations, confirming a clean pass is reported.

**Acceptance Scenarios**:

1. **Given** a graph where every node conforms to the CORE §14 checklist, **When** the user runs the lint command, **Then** the tool reports a clean pass and exits successfully, with no violations listed.
2. **Given** a graph containing one or more nodes that violate one or more checklist rules, **When** the user runs the lint command, **Then** the tool lists every violation found, each naming the specific rule violated, the file path of the offending node, and the line number within that file where the violation is located, and exits with a non-zero, failing status.
3. **Given** a graph with violations spanning multiple distinct rules across multiple files, **When** the user runs the lint command, **Then** every violation is reported in the same run — the tool does not stop at the first violation found.

---

### User Story 2 - Catch a broken link introduced by a hand edit or a bad patch (Priority: P2)

A user (or an `arc apply` run) has introduced a `[[wiki-link]]` that points to a basename no longer present in the graph — the target node was renamed, removed, or never existed. The user wants this caught before it silently degrades navigation.

**Why this priority**: Broken links are the most common and most consequential drift in a graph that's edited by hand or extended by multiple tools, since a node's identity is its basename and links are the graph's only structural connective tissue (CORE §3.2). Catching them is close to the core value of linting but is one specific rule among the full checklist, hence P2.

**Independent Test**: Can be fully tested by introducing a `[[link]]` to a nonexistent basename into an otherwise-conformant graph, running the lint command, and confirming exactly that link is reported with its file and line, and no other violation is reported.

**Acceptance Scenarios**:

1. **Given** a node whose body contains a `[[link]]` to a basename that does not exist anywhere in the graph, **When** the user runs the lint command, **Then** the tool reports that specific link as a violation, with the file and line where the link appears.
2. **Given** a node whose body contains a `[[link]]` to a basename that does exist in the graph, **When** the user runs the lint command, **Then** that link is not reported as a violation.
3. **Given** two different nodes that were independently created (e.g. by two separate `arc apply` runs) and happen to share the same basename, **When** the user runs the lint command, **Then** the tool reports the basename collision as a violation, naming both file paths involved.

---

### User Story 3 - Spot a graph left mid-merge-conflict (Priority: P3)

A user's graph was checked out or merged in a way that left unresolved git merge-conflict markers inside one or more node files (e.g. a manual `git merge` that wasn't fully resolved before the files were saved). The user wants this caught, since a node file containing conflict markers is not valid YAML front-matter or valid node content and would otherwise corrupt any tool that reads it.

**Why this priority**: Rarer than a broken link or malformed front-matter in normal single-operator use, since `arc apply`'s own commit discipline (CORE §11.1) is designed to prevent it, but a real risk whenever a graph's files are also touched by ordinary git workflows outside this tool. Important enough to be part of the checklist but affects fewer users day-to-day than P1/P2.

**Independent Test**: Can be fully tested by writing a node file that contains an unresolved git conflict marker block, running the lint command, and confirming that file is reported as containing an active merge conflict, with the line number of the conflict marker.

**Acceptance Scenarios**:

1. **Given** a node file containing unresolved git conflict markers (e.g. `<<<<<<<`, `=======`, `>>>>>>>`), **When** the user runs the lint command, **Then** the tool reports that file as having an active merge conflict, with the line number of the first conflict marker found.
2. **Given** a graph with no files containing conflict markers, **When** the user runs the lint command, **Then** no merge-conflict violations are reported.

---

### Edge Cases

- What happens when a node file's front-matter is missing entirely, is not valid YAML, or is missing the mandatory `kind` field? The tool must report this as a violation naming the file and, where a line number is meaningful (e.g. a YAML parse error), the offending line; it must not crash or abort the rest of the run.
- What happens when a `source` node's `id` field does not equal its own basename (CORE §6.2)? The tool must report this as a violation, since it breaks the deterministic citekey guarantee the graph depends on.
- What happens when an `entity` node's `category` is missing, has other than four words, or decodes to a combination not in CORE §9.2.1's fixed vocabulary? The tool must report this as a violation with the file and the line the `category` field appears on.
- What happens when a derived node (any node distilled from a document, per CORE §3.4) has no link back to any `source` node at all? The tool must report this as a violation, since it breaks the graph's provenance chain.
- What happens when a predicate used in a node's body is not camelCase, or is camelCase but not registered in `_meta/predicates.md`? The tool must report both cases as violations, distinguishing "not camelCase" from "not registered" in the reported message.
- What happens when `_meta/predicates.md` itself does not exist? The tool must treat every predicate encountered as unregistered and report accordingly, rather than crashing.
- What happens when a citation in a node's body uses a predicate that is not one of the registered `cito:`-aligned citation predicates? The tool must report this as a violation distinct from a generic unregistered-predicate violation, since citations have their own required predicate family (CORE §8).
- What happens when the git history shows more than one commit — or zero commits — contributing a single document's ingestion (CORE §11.1's "one document, one `graph(ingest):` commit" rule)? The tool must report this as a violation identifying the document and the commit(s) involved.
- What happens when a node's `kind` is an extension/domain kind the graph has registered, and that kind's profile defines its own additional checklist items (e.g. required fields beyond CORE's base rules)? The tool must run that kind's profile-specific checks in addition to the base CORE §14 checks, and report any profile-specific violation the same way — file, line, rule violated.
- What happens when a node's `kind` is not one of CORE's built-in kinds and not a registered extension kind? The tool must report this as a violation (unrecognized kind), consistent with how `arc apply` treats an unregistered kind as needing a visible warning rather than being silently accepted as conformant.
- What happens when the target directory is not an initialized graph? The tool must refuse to run and report that clearly, the same way other graph commands do, rather than reporting a graph full of violations.
- What happens when the graph is very large (many thousands of nodes)? The tool must still complete and report every violation in a single run; performance at that scale is addressed by SC-004.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST scan every node file in the graph and, for each, validate the presence of well-formed YAML front-matter and a mandatory `kind` field, reporting a violation with file path and line number for any node that fails this check.
- **FR-002**: The tool MUST verify that every node's basename is unique across the whole graph, reporting a violation naming all file paths that share a colliding basename when a collision is found.
- **FR-003**: The tool MUST verify that every `[[link]]` found in any node's body resolves to a basename that exists elsewhere in the graph, reporting a violation with the file path and line number of any link that does not resolve.
- **FR-004**: For every node of kind `source`, the tool MUST verify that its citekey `id` field equals its own basename, reporting a violation with file path and line number when they differ.
- **FR-005**: For every node of kind `entity`, the tool MUST verify that its `category` field decodes to exactly a four-word Sowa category drawn from the fixed CORE §9.2.1 vocabulary, reporting a violation with file path and line number when the field is missing, malformed, or does not decode to a valid four-word combination.
- **FR-006**: For every derived node (any node distilled from a document, i.e. every node other than a `source` node itself), the tool MUST verify it links to at least one `source` node, reporting a violation with file path when no such link is found.
- **FR-007**: The tool MUST verify that every predicate used in a node's body is camelCase, reporting a violation with file path and line number for any predicate that is not.
- **FR-008**: The tool MUST verify that every predicate used in a node's body is registered in `_meta/predicates.md`, reporting a violation with file path and line number for any predicate that is not registered, distinguishing this from the camelCase violation (FR-007) in the reported message.
- **FR-009**: The tool MUST verify that every citation recorded in a node's body uses a registered predicate drawn from the `cito:`-aligned citation-type vocabulary, reporting a violation with file path and line number for any citation that does not.
- **FR-010**: The tool MUST verify, from the graph's git history, that each ingested document corresponds to exactly one `graph(ingest):` commit, reporting a violation identifying the document and the commit(s) found when a document maps to zero or more than one such commit.
- **FR-011**: For any node whose `kind` is a registered extension/domain kind, the tool MUST additionally run that kind's own profile-defined conformance checklist and report any violation the same way as a base CORE check — with file path and line number.
- **FR-012**: The tool MUST verify that no node file contains unresolved git merge-conflict markers, reporting a violation with file path and the line number of the first marker found in any file that does.
- **FR-013**: The tool MUST NOT stop at the first violation found — it MUST continue scanning and report every violation across every node in a single run.
- **FR-014**: The tool MUST make no changes to the graph's files or git history — linting is a read-only, non-mutating operation.
- **FR-015**: The tool MUST report, for every violation, at minimum: which checklist rule was violated, the file path of the offending node, and a line number within that file (or a clear indication that a line number is not applicable, e.g. for a basename-collision violation spanning two files).
- **FR-016**: When the graph contains zero violations, the tool MUST report a clean pass and exit with a success status; when the graph contains one or more violations, the tool MUST exit with a distinct, non-zero failing status.
- **FR-017**: The tool MUST refuse to run, and MUST report this clearly instead of scanning, when the target directory is not an initialized graph.
- **FR-018**: When a node's `kind` is neither a CORE built-in kind nor a registered extension kind, the tool MUST report this as a violation (unrecognized kind) rather than silently treating the node as conformant.

### Key Entities

- **Violation**: One reported instance of a node (or pair of nodes, or commit) failing a specific CORE §14 checklist rule; carries the rule violated, the file path(s) involved, and a line number where applicable.
- **Lint Run**: One invocation of the command against a graph; produces zero or more Violations and an overall pass/fail outcome.
- **Checklist Rule**: One of the fixed CORE §14 checks (front-matter/kind validity, basename uniqueness, link resolution, source citekey identity, entity category shape, derived-node provenance, predicate naming/registration, citation predicate registration, one-commit-per-document, extension profile conformance, no active merge conflicts) that every Lint Run evaluates against every applicable node.
- **Predicate Registry**: The contents of `_meta/predicates.md`, consulted to determine whether a predicate encountered in a node body is registered.
- **Extension Profile Checklist**: The additional, kind-specific conformance checks a registered domain/extension kind declares beyond CORE's base rules, evaluated only against nodes of that kind.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can determine whether their entire graph conforms to the format's own rules, and get an exact file-and-line pointer to every violation, from a single command run — no manual file-by-file review required.
- **SC-002**: 100% of graphs with zero introduced violations produce a clean-pass report from the tool.
- **SC-003**: 100% of a graph's known violation types (one deliberately introduced per checklist rule) are each individually detected and reported, with no rule silently skipped.
- **SC-004**: A graph of several thousand nodes completes a full lint run in under 30 seconds.
- **SC-005**: A user can distinguish a passing run from a failing one purely from the command's exit status, without needing to parse its output text.
- **SC-006**: Running the lint command never modifies any graph file or git history — verified by the graph's state being byte-for-byte identical before and after any run.

## Assumptions

- The graph being linted already exists and was created by the graph's initialization command; this feature does not initialize or repair a graph, only reports on its conformance.
- Linting operates entirely on local files and local git history — no network access is required or attempted.
- Extension/domain kind profile checklists (CORE §10/§14's "per-kind profile checklist") are themselves discoverable from the graph's own registered extension configuration (the same registration mechanism `arc apply` uses to recognize domain kinds); this feature does not define new syntax for declaring a profile's checklist, only consumes what's already registered.
- Fixing a reported violation (auto-correcting a node file, re-registering a predicate, resolving a conflict marker) is out of scope for this feature — `arc lint` reports; it does not repair. A future command may add auto-fix behavior.
- A "line number" for a violation refers to the line within the offending node's own Markdown file (front-matter or body), using 1-based line counting consistent with common editor and diff conventions.
