# Feature Specification: Explicit Serve Context & Shareable Graph State

**Feature Branch**: `034-serve-dir-public-state`

**Created**: 2026-09-11

**Status**: Draft

**Input**: User description: "The cli suffers from usability especially when `arc serve` is used from clones of repository. An improvement for following aspects is required: * `arc serve` take an optional parameter `arc serve <dir>` to switch the context from current to specified dir. This approach allows explicit MCP configuration with agents. * `.arc` is git ignored but `arc` cli requires it. The purpose of `.gitignore` was a preparation for internal state management. The structure of `.arc` has to be split into the public `.arc` and `.arc/cache` where `.arc/cache` has gitignore file with `*`."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Point the MCP server at a graph by path (Priority: P1)

A knowledge worker wires `arc` into an AI agent's MCP configuration. The agent host decides the working directory the server process is launched in, and that directory is rarely the graph. Today the only way to make this work is a wrapper script that changes directory first, which is fragile and cannot be expressed in a plain MCP server entry. The user wants to name the graph directly in the agent configuration — `arc serve /path/to/graph` — and have the server expose exactly that graph.

**Why this priority**: This is the acute failure. Without it, `arc serve` cannot be configured declaratively in any agent host, which is the primary way the tool is consumed. It is also independently valuable even if nothing about graph state changes.

**Independent Test**: Launch the MCP server with a directory argument from a working directory that is not a graph, then call a tool and confirm it returns content from the named graph.

**Acceptance Scenarios**:

1. **Given** a graph at a known path and a shell whose working directory is somewhere else entirely, **When** the user starts the MCP server naming that path, **Then** the server starts and every tool answers from that graph.
2. **Given** the same graph, **When** the user starts the MCP server from inside the graph with no path argument, **Then** behaviour is exactly as it is today — the current directory is the graph.
3. **Given** an agent configuration that names the graph by absolute path, **When** the agent host launches the server from an arbitrary working directory, **Then** the session succeeds with no wrapper script and no directory change.
4. **Given** a path that does not exist, or exists but is not a graph, **When** the user starts the MCP server naming it, **Then** the command refuses before any transport is opened, names the offending path, states the corrective action, and exits non-zero.
5. **Given** a graph named by a path, **When** the server is started over network transport rather than the default transport, **Then** the directory argument applies identically.

---

### User Story 2 - Clone a graph repository and use it immediately (Priority: P1)

A collaborator clones a colleague's graph repository and runs an `arc` command. Today every command refuses, because the marker directory that identifies a graph is excluded from version control and therefore absent from the clone. The user expects a clone of a graph to *be* a graph, with no initialization step and no local repair.

**Why this priority**: This blocks every multi-person and multi-machine use of the tool, including the agent scenario in Story 1 — an agent pointed at a fresh clone fails for this reason alone. It is the second half of the usability problem and is independently testable.

**Independent Test**: Initialize a graph, push or copy the repository, clone it into a fresh location, and run a read command inside the clone without any setup — it must succeed.

**Acceptance Scenarios**:

1. **Given** a newly initialized graph, **When** the user inspects what version control tracks, **Then** the graph's state directory and its published contents are tracked.
2. **Given** a graph repository, **When** a second user clones it, **Then** the clone is recognized as an initialized graph by every command with no additional step.
3. **Given** a graph whose owner has tuned the graph's configuration, **When** a collaborator clones the repository, **Then** the collaborator's commands run under the same configuration with no manual copying.
4. **Given** a freshly initialized graph, **When** the user checks the working tree immediately afterwards, **Then** the tree is clean — nothing is left unstaged or untracked.

---

### User Story 3 - Machine-local state never leaves the machine (Priority: P2)

The graph's state directory carries two kinds of content that must not be confused: what the team shares (configuration, the marker that says "this is a graph") and what belongs only to one checkout (derived indexes, caches, scratch state). The user wants a single, clearly reserved location for the second kind, excluded from version control in a way that cannot accidentally leak.

**Why this priority**: It is what makes Story 2 safe to keep. Without a designated ignored area, the pressure to exclude the whole state directory returns the moment any derived state is introduced. It is lower priority than P1 only because no derived state exists yet — this reserves the space correctly.

**Independent Test**: Initialize a graph, place an arbitrary file in the reserved cache location, and confirm version control reports the working tree as clean and the file as excluded.

**Acceptance Scenarios**:

1. **Given** a newly initialized graph, **When** the user inspects the state directory, **Then** a dedicated cache sub-location exists and carries its own exclusion rule covering everything inside it, including the rule file itself.
2. **Given** an initialized graph, **When** any file is written into the cache location, **Then** version control ignores it and the working tree stays clean.
3. **Given** an initialized graph, **When** the initial commit is inspected, **Then** it contains the shared state directory contents and nothing from the cache location.
4. **Given** a graph in a project that already has its own version-control exclusion rules, **When** the graph is initialized, **Then** `arc` has neither read, created, nor modified any exclusion rule outside the graph's own state directory.

---

### User Story 4 - Bring an existing graph to the new layout (Priority: P3)

A user who already has a graph created by an earlier version of the tool upgrades. Their state directory is entirely excluded from version control, so their existing clones and any new clone are still broken. They need a clear, low-risk path from the old layout to the new one.

**Why this priority**: It affects only graphs created before this change, and can be delivered after the two P1 stories are working. It matters for adoption, not for the feature's core value. It is deliberately documentation-only — the tool stays silent about the old layout rather than nagging or rewriting a user's repository.

**Independent Test**: Take a graph created under the old layout, follow the documented migration verbatim, and confirm a fresh clone of it is usable — and that the same graph worked before migrating too.

**Acceptance Scenarios**:

1. **Given** a graph in the old layout, **When** the user runs any `arc` command in it, **Then** the command still works — the graph is not rejected for being in the old layout.
2. **Given** a graph in the old layout, **When** the user follows the documented migration, **Then** a fresh clone of the repository is a usable graph.
3. **Given** a graph in the old layout, **When** the user consults the project documentation, **Then** they find the migration described as an explicit, ordered, reversible sequence of steps, with the consequence of not migrating (clones stay unusable) stated plainly.

---

### Edge Cases

- The directory argument names a path that exists but is a file, not a directory → refuse with a message naming the path; the server never starts.
- The directory argument names a path the user cannot read → refuse with a permission-specific message rather than "not a graph".
- The directory argument is relative (`./notes`, `..`) → resolved against the current working directory, then treated exactly as an absolute path.
- The directory argument is given *and* the current working directory is also a graph → the named directory wins; the current directory is never consulted.
- More than one directory argument is supplied → usage error; the command does not guess.
- The named graph is deleted or moved while the server is running → tool calls fail with a clear error; the server does not silently fall back to another directory.
- A graph is initialized inside a repository whose existing exclusion rules already ignore the graph's state directory → the graph's own rules cannot un-ignore it; the user is told rather than left with a silently broken clone.
- A graph is initialized under a path that itself matches a broad exclusion rule of the host repository → same as above.
- Cache location exists but has lost its exclusion rule → the rule is a required part of the layout; its absence must be detectable rather than silently leaking cache contents into commits.

## Requirements *(mandatory)*

### Functional Requirements

**Explicit serve context**

- **FR-001**: `arc serve` MUST accept an optional positional directory argument naming the graph to serve.
- **FR-002**: When the argument is omitted, `arc serve` MUST behave exactly as it does today — the current working directory is the graph. No existing invocation changes meaning.
- **FR-003**: A relative directory argument MUST be resolved against the current working directory before use; an absolute argument MUST be used as given.
- **FR-004**: When the argument names a path that does not exist, is not a directory, is not readable, or is not an initialized graph, `arc serve` MUST refuse before opening any transport, with a message that names the path and states the corrective action, and MUST exit non-zero.
- **FR-005**: Every tool exposed by the server MUST answer from the named graph for the whole lifetime of the session, and MUST NOT expose content from outside it.
- **FR-006**: The directory argument MUST apply identically to the default transport and to the network transport.
- **FR-007**: Supplying more than one positional argument MUST be a usage error.
- **FR-008**: The command's help text and examples MUST document the argument, including an example suitable for pasting into an agent's server configuration.

**Shareable graph state**

- **FR-009**: `arc init` MUST produce a graph whose state directory is tracked by version control — the state directory MUST NOT carry a blanket exclusion rule.
- **FR-010**: `arc init` MUST create a dedicated cache location inside the state directory, carrying its own exclusion rule that excludes that location's entire contents including the rule file itself.
- **FR-011**: The single commit `arc init` produces MUST include the tracked contents of the state directory and MUST NOT include anything from the cache location.
- **FR-012**: After `arc init` the working tree MUST be clean.
- **FR-013**: A clone of a graph repository MUST be recognized as an initialized graph by every `arc` command, with no initialization or repair step.
- **FR-014**: A graph's configuration, when present, MUST be tracked by version control so that it travels with a clone and applies to every collaborator.
- **FR-015**: The cache location MUST be reserved for machine-local, reproducible state only. No command may place data there that another user needs in order to operate the graph.
- **FR-016**: `arc` MUST NOT read, create, or modify any version-control exclusion rule outside the graph's own state directory — the behaviour that lets a graph live inside a project with pre-existing rules is preserved unchanged.
- **FR-017**: When the graph's state directory cannot be tracked because the surrounding repository excludes it, `arc init` MUST tell the user, naming the consequence (clones will not work), rather than completing silently.

**Compatibility and migration**

- **FR-018**: A graph created by an earlier version, whose state directory is entirely excluded from version control, MUST continue to be recognized and fully usable by every `arc` command.
- **FR-019**: The migration from the old layout to the new one MUST be documented as a small, explicit, reversible sequence of steps.
- **FR-020**: The tool MUST NOT detect, warn about, or convert the old layout. A graph in the old layout is accepted silently and behaves as it does today; migration is a documented, user-driven action only.
- **FR-021**: The project documentation MUST state, alongside the migration steps, what continues to fail until a user migrates — namely that clones of an unmigrated graph are not recognized as graphs.

### Key Entities

- **Graph directory**: The root a command operates on. Today always the current working directory; after this feature, explicitly nameable for the MCP server.
- **Graph state directory**: The marker directory that identifies a directory as a graph and holds graph-scoped settings. Becomes shared, version-controlled content.
- **Local cache location**: A reserved sub-location of the graph state directory holding machine-local derived state. Always excluded from version control; empty of content in this feature.
- **Graph configuration**: Graph-scoped tuning that lives in the state directory. Becomes shared with every clone as a consequence of the split.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can start an MCP server against any graph on disk with a single command, from any working directory, with no wrapper script and no directory change.
- **SC-002**: An agent configuration that names a graph by absolute path succeeds on 100% of launches, independent of the working directory the agent host chooses.
- **SC-003**: A collaborator who clones a graph repository can run their first `arc` command successfully with zero setup steps, down from a mandatory local repair step today.
- **SC-004**: Every existing `arc serve` invocation in use today produces identical behaviour after the change — zero regressions in the no-argument path.
- **SC-005**: Immediately after initializing a graph, version control reports a clean working tree, and no file placed in the cache location ever appears in any commit.
- **SC-006**: Graph configuration authored by one user is in effect for every collaborator after a clone, with no manual file copying.
- **SC-007**: Serving a path that is missing or is not a graph fails within the first second, names the path, states the corrective action, and never opens a transport.
- **SC-008**: A graph created by a previous version keeps working after upgrade — 100% of commands that succeeded before still succeed.

## Assumptions

- Only `arc serve` gains a directory argument in this feature. Every other command keeps its current-working-directory semantics; extending them is deliberately out of scope (see below).
- The cache location is created empty apart from its exclusion rule. No command produces cache content in this feature — the location is reserved so that future derived state has a correct home from day one.
- The graph-recognition rule itself is unchanged: a directory is a graph when the state directory is present. The change is which parts of that directory version control tracks, not what marks a graph.
- `arc init`'s existing guards (refusing an already-initialized graph, repository-context checks, rollback on failure) stay as they are; only the layout written and the set of paths committed change.
- No new configuration keys are introduced.
- Shell-level path expansion (`~`, globs, environment variables) is the shell's job; the tool receives an already-expanded path.
- A "clone" means any copy obtained through version control; the same reasoning covers checkouts on a second machine and CI checkouts.
- Graphs created before this change are rare enough, and the migration short enough, that documenting it is sufficient; no in-tool detection is warranted.

## Out of Scope

- A directory argument, global flag, or environment variable for commands other than `arc serve`.
- Any write-side behaviour for the MCP server; it stays strictly read-only.
- Serving more than one graph from a single server process.
- Producing, populating, or invalidating any cache content.
- Changing the format or schema of the graph configuration.
- Automated rewriting of a host repository's own exclusion rules.
- Any in-tool detection of, warning about, or automatic conversion of the legacy state-directory layout. Migration is documentation-driven and user-initiated (FR-020).
