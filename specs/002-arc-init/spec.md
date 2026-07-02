# Feature Specification: Initialize a New Knowledge Graph (`arc init`)

**Feature Branch**: `002-arc-init`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "`arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root from VISION.md); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11 https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bootstrap a new graph in the current directory (Priority: P1)

A user starting a brand-new knowledge graph project runs the init command in an empty directory. The command sets up the complete folder structure, registry stub files, local tool state, and version-control history needed before any other graph operation (apply, list, query) can be used.

**Why this priority**: Every other capability of the tool (ingest, query, lint, serve) depends on a graph root existing first. Without this, the tool has nothing to operate on — it is the mandatory entry point.

**Independent Test**: Can be fully tested by running the init command in an empty directory and inspecting the resulting file tree and git history — delivers a ready-to-use empty graph with no other command required.

**Acceptance Scenarios**:

1. **Given** an empty directory with no existing graph, **When** the user runs the init command with no arguments, **Then** the tool creates the `sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, and `_meta/` folders, the `_meta/predicates.md` and `_meta/aliases.md` stub files, a `.arc/` state directory, and a `.gitignore` that excludes `.arc/`.
2. **Given** the graph was just initialized, **When** the user inspects the git history, **Then** exactly one commit exists whose subject line is `graph(init): empty knowledge graph`.
3. **Given** the graph was just initialized, **When** the user checks the working tree status, **Then** the tree is clean (nothing untracked, nothing staged-but-uncommitted) and `.arc/` does not appear in tracked files.
4. **Given** the graph was just initialized, **When** the user inspects any of the canonical folders, **Then** each one is present in git history (via a placeholder) even though it has no real content yet.

---

### User Story 2 - Bootstrap a graph in a named target directory (Priority: P2)

A user wants the new graph created in a specific directory rather than the current one, optionally one that does not exist yet.

**Why this priority**: Common secondary usage — scripting, tooling, and multi-graph workflows need to target an explicit path without first `cd`-ing into it.

**Independent Test**: Can be fully tested by running the init command with a directory argument pointing at a path that does not yet exist, and confirming the graph is created there while the current directory is left untouched.

**Acceptance Scenarios**:

1. **Given** a target directory that does not exist, **When** the user runs the init command with that directory as an argument, **Then** the tool creates the directory and the full canonical layout inside it, and the current working directory is unaffected.
2. **Given** initialization into a named directory succeeds, **When** the command finishes, **Then** the tool reports the resolved path of the newly created graph to the user.

---

### User Story 3 - Protected against accidentally destroying an existing graph (Priority: P3)

A user runs the init command again, by mistake or out of habit, against a directory that is already a graph.

**Why this priority**: Lower priority than the two creation paths, but essential as a safety net — an ingested graph represents accumulated, hard-to-reconstruct work, and an initializer command is the most likely command to be re-run absent-mindedly.

**Independent Test**: Can be fully tested by running the init command twice against the same directory and confirming the second run does not remove or alter any content produced after the first run.

**Acceptance Scenarios**:

1. **Given** a target directory that already contains a `.arc/` state directory (an already-initialized graph), **When** the user runs the init command against it again, **Then** the tool refuses with a clear error, makes no filesystem or git changes, and no existing graph file, commit, or state is lost or overwritten.

---

### Edge Cases

- What happens when the target path already exists but is a regular file rather than a directory? The tool must refuse and make no changes.
- What happens when the target directory exists, is not already a graph, but already contains unrelated files? The tool must refuse and require an empty directory (FR-015).
- What happens when git is not installed or not available on the system? The tool must refuse with a clear explanation, since version control is mandatory for every graph.
- What happens when the user lacks write permission on the target directory or its parent? The tool must fail with a clear explanation and leave no partially-created state behind.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST create the canonical top-level folders `sources/`, `entities/`, `resources/`, `timeline/yearly/`, and `timeline/monthly/` inside the target graph directory.
- **FR-002**: The tool MUST create a `_meta/` folder containing stub registry files `_meta/predicates.md` and `_meta/aliases.md`.
- **FR-003**: The tool MUST create a `.arc/` directory in the target graph directory to hold all arc-managed local state, separate from the versioned graph content.
- **FR-004**: The tool MUST initialize the target graph directory as a git repository.
- **FR-005**: The tool MUST place a placeholder file in every canonical folder that would otherwise be empty, so that git tracks the folder's existence.
- **FR-006**: The tool MUST write a `.gitignore` file that excludes `.arc/` from version control.
- **FR-007**: The tool MUST stage every file created during initialization and produce exactly one initial commit whose subject line is exactly `graph(init): empty knowledge graph`.
- **FR-008**: The tool MUST default to the current working directory as the target graph directory when no directory argument is supplied.
- **FR-009**: The tool MUST accept an optional target directory argument and create that directory if it does not already exist.
- **FR-010**: The tool MUST refuse to initialize, and MUST make no filesystem or git changes, when the resolved target path exists and is a regular file rather than a directory.
- **FR-011**: The tool MUST refuse to initialize, and MUST make no filesystem or git changes, when git is not available on the system, and MUST explain that version control is required.
- **FR-012**: The tool MUST report the resolved path of the newly created graph to the user on success.
- **FR-013**: The tool MUST leave no partially-created graph state on the filesystem when initialization fails partway through (e.g., due to a permission error).
- **FR-014**: When the target directory already contains a `.arc/` state directory, the tool MUST refuse to initialize, MUST make no filesystem or git changes, and MUST print a clear error explaining that a graph already exists at that location.
- **FR-015**: When the target directory exists but is not already a graph (no `.arc/` present) and already contains any files or subdirectories, the tool MUST refuse to initialize, MUST make no filesystem or git changes, and MUST print a clear error explaining that the target directory must be empty.
- **FR-016**: By default, the tool MUST report success as a single, concise line (resolved path and a short, unambiguous commit reference per FR-012) and MUST NOT print the intermediate steps it took internally; a user who wants to see those intermediate steps MUST be able to request them explicitly via a verbosity option, without changing the default's conciseness. *(Added 2026-07-02 — BUG-001: default output was over-reporting every internal step.)*
- **FR-017**: The tool MUST seed the new graph with a default per-kind merge-rule configuration usable by later graph-mutating commands, preferring the canonical published defaults when they can be fetched, and MUST always succeed using a built-in fallback default when they cannot be fetched (no network access, or any fetch failure) — initialization MUST NOT fail, and MUST NOT block on network access, on this basis alone. *(Added 2026-07-02 — cross-referenced from `specs/003-apply-patch`, which introduces the first consumer of this configuration.)*

### Key Entities

- **Knowledge Graph (Graph Root)**: The directory tree representing one graph instance; identified by the presence of a `.arc/` directory at its top level.
- **Canonical Folder**: One of the fixed top-level directories that every graph must contain (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`).
- **Metadata Stub**: A registry placeholder file (`_meta/predicates.md`, `_meta/aliases.md`) that later commands append controlled-vocabulary entries to.
- **Arc State Directory**: The `.arc/` directory holding tool-managed local state that is never versioned alongside the graph content.
- **Initial Commit**: The single git commit that records the graph's creation, with the mandatory subject line `graph(init): empty knowledge graph`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can go from "no graph exists" to "a ready-to-use empty graph with committed history" in a single command invocation, completing in under 5 seconds on a typical local filesystem.
- **SC-002**: 100% of freshly initialized graphs contain exactly one commit in their history, with no untracked or uncommitted files remaining afterward.
- **SC-003**: 100% of freshly initialized graphs contain every canonical folder and both metadata stub files, with zero structural violations when the graph is later validated against project conformance rules.
- **SC-004**: A user can confirm initialization succeeded from the command's own output alone, without needing to run any follow-up inspection command.
- **SC-005**: Re-running initialization against an already-initialized graph never results in loss of previously accumulated graph content, across 100% of observed cases.

## Assumptions

- Git is installed and available on the user's system; the tool depends on it and does not bundle or substitute for it.
- The user has write permission to the target directory and, when the directory does not yet exist, to its parent directory.
- Initialization always succeeds fully local and offline; network access (FR-017's best-effort canonical-config fetch) is an optional enhancement the tool attempts but never requires, and its absence or failure is never visible to the user as an initialization error. *(Revised 2026-07-02 — see FR-017 cross-reference.)*
- The user's git identity (`user.name` / `user.email`) is already configured through the user's normal git setup; the tool does not configure git identity itself.
- Only the CORE canonical layout is created by this command. Domain-profile-specific folders or files are opt-in additions made by later commands, not part of initialization, consistent with the tool's extension-agnostic design.

## Notes

**Bugfix**: 2026-07-02 — BUG-001: Added FR-016 (default output MUST be a single concise line; step-by-step detail is opt-in via a verbosity option). The rest of BUG-001's fixes (progress-line styling/alignment, short commit hash, `PostRunE` hint text) are presentation-layer decisions that stay in `plan.md`/`research.md`/`contracts/` per this spec's technology-agnostic scope.

**Cross-feature update**: 2026-07-02 — `specs/003-apply-patch` introduces the first command (`arc apply`) that consumes per-kind merge-rule configuration, and requires `arc init` to seed it (FR-017, added above). The fetch source, URL, fallback content, and adapter design are implementation detail owned by `specs/003-apply-patch/plan.md`/`research.md` (D5), not this spec, per this document's technology-agnostic scope — this spec only records the resulting behavioral change to `arc init` itself (always-local-success, network as best-effort enhancement).
