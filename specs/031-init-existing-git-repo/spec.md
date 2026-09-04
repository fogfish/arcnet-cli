# Feature Specification: Graph Initialization Inside an Existing Git Repository

**Feature Branch**: `031-init-existing-git-repo`

**Created**: 2026-09-04

**Status**: Draft

**Bugfix**: 2026-09-04 — [BUG-001](bugs/BUG-001.md) FR-010 made the graph root a directory `arc` shares with a host project, but no requirement said what that means for commands that walk the tree. `arc apply`'s whole-graph scan parsed the host's own `README.md` as a graph node and refused to run. FR-032–FR-035, SC-009/SC-010 and three edge cases are added to state the sharing contract: `apply` reads only what its patch targets, and `lint`/`grep` carry an internal index of foreign files instead of failing on them.

**Bugfix (round 2)**: 2026-09-04 — [BUG-001](bugs/BUG-001.md) verification found FR-034 had no plan.md counterpart and no quickstart scenario exercised a shared graph root. No requirement changed: [plan.md](plan.md)'s Constraints now carry FR-034's read-path sentinel rule, and [quickstart.md](quickstart.md) gains S10 — the shape S3 and S9 never built, where the graph root *is* the project root.

**Input**: User description: "`arc init` fails to create an empty graph in the root of existing git repository. Also it creates a graph in the subfolder of existing git repository but it initializes a repository within the repository. It does not suites use-cases where you'd like to add a graph into existing project. The `arc init` MUST fail if it is executing in the context of git project also blocking the possibility to create a repository within repository. However, `arc init` should support the advanced usage flag `--skip-git-init`. It allows to create a graph within existing repository. All commands that requires and uses git must use existing \"parent\" repo fo all commands."

## Clarifications

### Session 2026-09-04

- Q: Where should the graph's local-state exclusion live, given that appending to the host project's ignore-rules file modifies a file `arc` does not own (Constitution XI)? → A: Self-contained — write `.arc/.gitignore` containing `*`, never touch the host project's ignore-rules file, and apply this uniformly in both modes (replacing today's root ignore-file write).
- Q: What happens when a folder the canonical layout would create already exists at the target? → A: Fail, whether that folder is empty or not. The user's recovery path is to initialize into a subfolder instead (`arc init --skip-git-init newgraph`), which the failure message must name.
- Q: What should `--skip-git-init` do when no enclosing repository exists? → A: Fail with a usage error. No fallback to creating a repository, and no history-less graph mode.
- Q: How should machine-readable output reflect which repository the commit landed in? → A: Add a `repository` field carrying the repository root, present unconditionally in both modes — equal to the graph path for a standalone graph, an ancestor path when nested. No mode boolean; it is derivable by comparison.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Refuse to nest a repository inside a repository (Priority: P1)

A user is working inside an existing project that is already under version control. They run `arc init` (with or without a target directory) somewhere inside that project, intending to create a graph. Today, if the target is a fresh subfolder, the tool silently creates a second, independent version-control repository nested inside the project's own — a state that is confusing to reason about, easy to create by accident, and hard to recognize after the fact. The user needs the tool to refuse this outright and tell them what to do instead.

**Why this priority**: This is the harmful default. Every user who tries the natural thing today ends up with a nested repository they did not ask for and may not notice for a long time. Blocking it is the single highest-value change and is independently shippable — even with nothing else in this feature, the tool stops producing a broken state.

**Independent Test**: Inside any existing versioned project, run `arc init` targeting a new subfolder. Verify the command fails, explains that the location is already inside a versioned project, names that project's root, points at the advanced flag, and leaves the filesystem exactly as it was.

**Acceptance Scenarios**:

1. **Given** the current directory is the root of an existing versioned project, **When** the user runs `arc init`, **Then** the command fails with a non-zero exit status, reports that the location already belongs to a versioned project, names that project's root, and creates no graph content.
2. **Given** the current directory is a subfolder of an existing versioned project, **When** the user runs `arc init`, **Then** the command fails the same way and does not create a nested repository.
3. **Given** the current directory is inside an existing versioned project, **When** the user runs `arc init ./new-folder` for a folder that does not yet exist, **Then** the command fails and `./new-folder` is not left behind on disk.
4. **Given** the failure has occurred, **When** the user reads the error message, **Then** it names the advanced flag that permits adding a graph to the existing project.
5. **Given** the current directory is NOT inside any versioned project, **When** the user runs `arc init` in an empty directory, **Then** the command behaves exactly as it does today: it creates the graph, starts a new repository, and records one initial commit.

---

### User Story 2 - Add a graph to an existing project (Priority: P1)

A user wants their knowledge graph to live inside a project they already version — either at the project root alongside existing files, or in a dedicated subfolder of it. They run `arc init` with the advanced flag. The tool creates the graph layout in place, does not start a second repository, and records the graph's initial commit in the project's own history.

**Why this priority**: This is the capability the user actually asked for, and the escape hatch that makes Story 1's refusal acceptable rather than merely obstructive. It is independently testable and delivers the whole "add a graph to my project" use case on its own.

**Independent Test**: In an existing versioned project that already contains files, run `arc init` with the advanced flag. Verify the graph layout appears, no nested repository is created, the project's history gains exactly one commit containing only the graph's own files, and the user's unrelated files are untouched and uncommitted.

**Acceptance Scenarios**:

1. **Given** the current directory is the root of an existing versioned project containing unrelated files, **When** the user runs `arc init` with the advanced flag, **Then** the graph layout is created alongside those files, no nested repository is created, and the command reports success.
2. **Given** the same situation, **When** the command completes, **Then** the project's history contains exactly one new commit, and that commit contains only files the initialization itself created.
3. **Given** the project had unrelated modified or untracked files before the command ran, **When** the command completes, **Then** those files are neither staged nor committed and remain exactly as they were.
4. **Given** the current directory is inside an existing versioned project, **When** the user runs `arc init ./notes` with the advanced flag for a folder that does not yet exist, **Then** the folder is created, the graph layout is written into it, and the initial commit is recorded in the enclosing project's history.
5. **Given** the target already contains an initialized graph, **When** the user runs `arc init` with the advanced flag there, **Then** the command fails with the existing "already initialized" error and changes nothing.
6. **Given** the user is NOT inside any versioned project, **When** they run `arc init` with the advanced flag, **Then** the command fails with a message explaining there is no enclosing project to add the graph to, and creates nothing.

---

### User Story 3 - Preserve the host project's existing files (Priority: P1)

A user adds a graph to the root of an existing project that already has its own ignore rules and possibly its own files at paths the graph layout also wants to use. The tool must add what is missing without touching, rewriting, or destroying anything the user already owns — the graph's own local-state exclusion included.

**Why this priority**: A tool that clobbers a project's existing ignore rules or seed files while "adding a graph" is worse than one that refuses to run at all. This is what makes Story 2 safe to actually use on a real project, so it ships with it.

**Independent Test**: In an existing versioned project that already has ignore rules and unrelated content, run `arc init` with the advanced flag. Verify the pre-existing ignore-rules file is byte-for-byte untouched, the graph's local state is nonetheless excluded from version control, and no pre-existing file or folder is overwritten or adopted.

**Acceptance Scenarios**:

1. **Given** the project root already has an ignore-rules file with user content, **When** the user runs `arc init` with the advanced flag there, **Then** that file is byte-for-byte unchanged and is not part of the initialization's commit.
2. **Given** the command has completed, **When** version-control status is inspected, **Then** the graph's local-state directory and everything under it are excluded from tracking, achieved without any entry in the host project's ignore-rules file.
3. **Given** the target directory already contains a file at a path the graph layout would write, **When** the command runs, **Then** it fails before writing anything, names the conflicting path, and leaves that file untouched.
4. **Given** the target directory already contains a folder whose name the canonical layout would also create, **When** the command runs, **Then** it fails before writing anything, names the conflicting folder, and suggests initializing into a subfolder instead.
5. **Given** that same conflicting project root, **When** the user re-runs the command naming a subfolder that does not yet collide, **Then** the graph is created there successfully.
6. **Given** the command fails partway through for any reason, **When** the user inspects the target, **Then** every file the command created has been removed, every pre-existing file is intact, and the enclosing project's history is unchanged.

---

### User Story 4 - Every git-backed command works from the parent repository (Priority: P2)

A user who added a graph to an existing project now runs the graph's other commands — applying content, reverting, and linting. Each of these reads or writes version-control history. They must all operate against the enclosing project's repository, treat paths as graph-relative, and confine their view of history to the graph's own subtree, so a graph living inside a larger project behaves identically to a standalone one.

**Why this priority**: Without this, Stories 1-3 produce a graph that is created correctly but degrades on every subsequent operation. It is P2 only because it depends on a graph having been created inside a project first, and because the existing behavior is already partly correct.

**Independent Test**: Create a graph in a subfolder of an existing project, apply content, then run the revert and lint commands. Verify each behaves identically to the same sequence performed on a standalone graph.

**Acceptance Scenarios**:

1. **Given** a graph inside a larger project, **When** the user applies content to it, **Then** the resulting commit contains only that graph's files, and unrelated changes elsewhere in the project are neither staged nor committed.
2. **Given** a graph inside a larger project whose history also contains commits from elsewhere in that project, **When** a command inspects graph history, **Then** only commits touching the graph's own subtree are considered.
3. **Given** a graph inside a larger project, **When** a command resolves a file path against history, **Then** the path is interpreted relative to the graph root, not the project root.
4. **Given** a graph inside a larger project, **When** the user runs the lint command, **Then** its history-dependent checks produce the same results as they would for an equivalent standalone graph.
5. **Given** a graph inside a larger project, **When** the user reverts a graph operation, **Then** only that graph's files are affected.

---

### Edge Cases

- **Target inside a nested worktree boundary**: the target directory is inside a versioned project, but a directory between it and that project's root is itself a separate repository. The tool treats the innermost enclosing repository as the parent.
- **Enclosing project has no commits yet**: the parent repository exists but its history is empty. Adding a graph with the advanced flag succeeds and produces the graph's initial commit as the project's first commit.
- **Target is a file, not a directory**: the existing "target is not a directory" failure still applies, and takes precedence over any repository-context check.
- **Version control unavailable**: the existing "version control is required" failure still applies in both modes, including when the advanced flag is used.
- **Advanced flag used in a directory that is not empty and not inside a project**: fails, because there is no enclosing repository to attach the graph to (see User Story 2, scenario 6).
- **Default mode in a non-empty directory not inside any project**: unchanged from today — the existing "target is not empty" failure applies.
- **Target is inside a project but is currently ignored by that project's rules**: the graph's files would be created but never tracked, producing a graph with no history. The tool reports this rather than creating a graph that silently cannot commit.
- **Concurrent unrelated staged changes in the parent repository**: the user has already staged unrelated files before running the command. The graph's initial commit must not absorb them.
- **Project root already occupies a canonical layout name**: a project whose root already contains, say, `Source/` or `timeline/` cannot host a graph at its root. The command fails and directs the user to a subfolder, which is the supported way to add a graph to such a project.
- **Repeated runs**: running the command twice in the same location fails the second time with the existing "already initialized" error, in both modes.

*(Edge cases below added by Bugfix BUG-001, 2026-09-04.)*

- **Host project markdown under the graph root**: the graph shares its root with a project carrying `README.md`, `CONTRIBUTING.md`, `docs/**.md` and similar. These are foreign files, not graph nodes. No command may parse them as nodes, fail because of them, or count them as graph content.
- **A large foreign tree under the graph root**: the host project contains far more markdown than the graph does. A command's read cost must be governed by what the command actually needs — for `arc apply`, the patch's own node set — not by the size of the host project's tree.
- **A patch targeting a path occupied by a foreign file**: the patch names a node whose canonical path already holds a markdown file lacking `"@id"`/`"@type"`. The command fails, naming that path and the missing mandatory field, and writes nothing — this is the one case where a foreign file is legitimately fatal, because the patch demanded it.

## Requirements *(mandatory)*

### Functional Requirements

#### Repository-context detection

- **FR-001**: The system MUST determine, before creating anything, whether the resolved target location falls inside an existing version-controlled project, by searching upward from the target through its ancestor directories.
- **FR-002**: The detection in FR-001 MUST work for a target directory that does not yet exist, by searching upward from the nearest existing ancestor.
- **FR-003**: When several enclosing repositories exist, the system MUST treat the innermost one as the parent repository.

#### Default behavior (no advanced flag)

- **FR-004**: When the resolved target falls inside an existing version-controlled project, `arc init` without the advanced flag MUST fail and MUST NOT create a nested repository.
- **FR-005**: The failure in FR-004 MUST occur before any file or directory is written, so the filesystem is left exactly as it was found.
- **FR-006**: The failure message MUST state that the location already belongs to a versioned project, MUST name that project's root location, and MUST name the advanced flag that permits adding a graph to it.
- **FR-007**: When the resolved target does not fall inside any existing version-controlled project, `arc init` without the advanced flag MUST behave as it does today: create the canonical layout in an empty or non-existent directory, start a new repository, and record exactly one initial commit. The sole intended change to this path is the local-state exclusion mechanism of FR-016.

#### Advanced flag

- **FR-008**: `arc init` MUST accept an opt-in flag, `--skip-git-init`, documented as advanced usage, that permits creating a graph inside an existing version-controlled project.
- **FR-009**: With the flag set, the system MUST NOT create a new repository at the target, and MUST use the enclosing parent repository for all version-control operations.
- **FR-010**: With the flag set, the system MUST allow the target directory to already contain unrelated files, replacing the current "target must be empty" restriction for this mode only. *(Bugfix BUG-001, 2026-09-04: "unrelated files" includes unrelated **markdown** files, which every tree-walking command previously assumed could not exist under a graph root — see FR-032.)*
- **FR-011**: With the flag set, the system MUST still fail when the target already contains an initialized graph, with the existing "already initialized" error, and MUST change nothing.
- **FR-012**: With the flag set and no enclosing version-controlled project found, the system MUST fail with a message explaining there is no parent project to add the graph to, and MUST create nothing.
- **FR-013**: With the flag set, the system MUST record exactly one commit in the parent repository, containing only the files the initialization itself created or modified.
- **FR-014**: The commit in FR-013 MUST NOT include any file the initialization did not create or modify, including files the user had already staged or left modified before running the command.

#### Preserving the host project

- **FR-015**: The system MUST NOT overwrite or adopt anything that already exists at the target. It MUST fail before writing anything, naming the conflicting path, when either (a) a file the layout would write already exists, or (b) a folder the canonical layout would create already exists — whether that folder is empty or not.
- **FR-016**: The system MUST exclude the graph's local-state directory from version control by a self-contained exclusion placed inside that directory itself, which excludes the directory's entire contents. This mechanism MUST be used identically in both modes.
- **FR-017**: The system MUST NOT create, modify, or append to any ignore-rules file it does not itself own, in either mode. A pre-existing ignore-rules file at the target MUST be left byte-for-byte unchanged and MUST NOT appear in the initialization's commit.
- **FR-018**: On any failure after writing has begun, the system MUST remove every file and directory it created, MUST leave every pre-existing file intact, and MUST NOT alter the parent repository's history.
- **FR-019**: The system MUST NOT delete a target directory that already existed before the command ran, even when rolling back.
- **FR-020**: When the target location is excluded by the parent project's ignore rules, the system MUST fail with an explanatory message rather than creating a graph whose files can never be committed.

#### Parent-repository usage across all commands

- **FR-021**: Every command that reads or writes version-control history MUST operate against the repository enclosing the graph root, whether that repository's root is the graph root itself or an ancestor of it.
- **FR-022**: Every command that stages changes MUST confine staging to the graph's own subtree, so unrelated changes elsewhere in the parent repository are never swept into a graph commit.
- **FR-023**: Every command that queries history MUST confine that query to commits touching the graph's own subtree, so commits belonging to the rest of the parent project are never mistaken for graph history.
- **FR-024**: Every command that resolves a file path against history MUST interpret that path relative to the graph root, never relative to the parent repository's root.
- **FR-025**: For any given operation, a graph located inside a larger project MUST produce the same observable outcome as an equivalent standalone graph. *(Bugfix BUG-001, 2026-09-04: "inside a larger project" has two independent axes — the graph root nested **below** the repository root, and the graph root **shared** with host-project files. Only the first was verified; the second is FR-032–FR-035.)*

#### Reporting and documentation

- **FR-026**: On success with the advanced flag, the human-readable output MUST report what changed, including the graph location, the recorded commit, and the fact that an existing parent repository was used rather than a new one.
- **FR-027**: Machine-readable output MUST carry the root of the repository the commit landed in, as a field present unconditionally in both modes. For a standalone graph it equals the graph location; for a graph inside a larger project it is that project's root. Consumers MUST be able to distinguish the two modes by comparing this value to the graph location, without a separate mode indicator.
- **FR-028**: The existing machine-readable fields — graph location, commit, and created folders — MUST retain their current names and meanings, so existing consumers continue to work unchanged.
- **FR-029**: All failures introduced by this feature MUST exit with a non-zero status and MUST print an actionable message, never a raw internal error value or stack trace.
- **FR-030**: The command's built-in help and the project's user documentation MUST describe the advanced flag, when to use it, and the default refusal it overrides.
- **FR-031**: The layout-collision failure of FR-015 MUST name the conflicting path and MUST state the recovery path: initializing the graph into a subfolder of the project instead of at the colliding location.

#### Sharing the graph root with the host project *(Bugfix BUG-001, 2026-09-04)*

- **FR-032**: A markdown file under the graph root that does not declare both `"@id"` and `"@type"` in its yaml front matter is a **foreign file** — content belonging to the host project, not to the graph. No command MAY interpret a foreign file as a graph node, and no command MAY fail merely because one exists under the graph root.
- **FR-033**: `arc apply` MUST NOT walk or parse the graph tree. It MUST read and parse only the node files its patch actually targets, so the number of files it opens is proportional to the patch's own node count and independent of how many files the graph root contains. This is required for correctness under FR-032 and for the read cost the command's latency budget assumes.
- **FR-034**: When a path an applied patch targets already exists but is not a graph node, `arc apply` MUST fail, naming that path and the mandatory field it lacks, and MUST write nothing. The message MUST describe the operation that actually failed — reading the file — and MUST NOT report it as a failed write.
- **FR-035**: `arc lint` and `arc grep` walk the graph tree by design and MUST therefore each maintain an internal index of the foreign files that walk encounters. Indexed files MUST be excluded from node processing, MUST NOT be counted among the nodes checked or matched, and MUST NOT be reported as failing or invalid nodes. The index is the command's own record of what it skipped and why, not a defect list.

### Key Entities

- **Target location**: the directory the user asked to initialize a graph in. May exist or not; may be empty or not; may be inside a versioned project or not.
- **Parent repository**: the innermost existing version-controlled project enclosing the target location, if any. Its root may be the target itself or any ancestor of it.
- **Graph root**: the directory containing the initialized graph's canonical layout. Equal to the target location. Distinct from the parent repository root, which it may or may not coincide with.
- **Graph subtree**: the portion of the parent repository's tree occupied by the graph — the scope every staging and history operation must confine itself to.
- **Initialization footprint**: the exact set of files and directories a single initialization created or modified — the scope of both the initial commit and of rollback.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Attempting to initialize a graph anywhere inside an existing versioned project without the advanced flag never produces a nested repository — 0 occurrences across all supported target shapes (project root, existing subfolder, non-existent subfolder).
- **SC-002**: A user who hits the refusal can determine the correct next step from the error message alone, without consulting documentation or source code.
- **SC-003**: Adding a graph to an existing project succeeds in a single command invocation, and the project's history grows by exactly one commit.
- **SC-004**: After adding a graph to an existing project, 100% of the user's pre-existing files are byte-for-byte unchanged, and 0 of their unrelated pending changes are committed.
- **SC-005**: Every graph operation performed on a graph inside a larger project yields results identical to the same operation on an equivalent standalone graph, across the full command set that touches version control.
- **SC-006**: Every failure path introduced by this feature leaves the filesystem and the parent project's history in exactly the state they were in before the command ran — verified for each distinct failure condition.
- **SC-007**: A script consuming machine-readable initialization output can determine which repository absorbed the commit, in both modes, from a single field that is always present — with no branching on output shape.
- **SC-008**: Existing standalone initialization behavior is preserved: every outcome the command currently guarantees outside a versioned project — layout, seeded content, single initial commit, clean working tree, and each existing failure condition — still holds. The one intended difference is where the local-state exclusion is written; the observable result (local state excluded from tracking, working tree clean after init) is unchanged.

*(Success criteria below added by Bugfix BUG-001, 2026-09-04.)*

- **SC-009**: Applying a patch to a graph that shares its root with a host project succeeds regardless of how many foreign markdown files that project contains, and the number of graph files `arc apply` opens is proportional to the patch's node count — unchanged whether the host project contributes zero foreign markdown files or thousands.
- **SC-010**: On a graph sharing its root with a host project, `arc lint` reports zero violations attributable to foreign files and its "nodes checked" count equals the graph's own node count, and `arc grep` returns zero matches drawn from foreign files.

## Assumptions

- **Advanced flag name**: the flag is `--skip-git-init`, exactly as named in the request. It has a long form only, with no single-letter shorthand, since it is advanced and infrequently used.
- **The flag requires an enclosing repository** (resolved in Clarifications): `--skip-git-init` means "attach this graph to the repository that already exists here," so using it where no repository exists is a usage error — neither a silent fallback to creating a repository (which would do the very thing the flag forbids) nor a history-less graph mode. The initial commit is the graph's contract, and every history-dependent command reads that history.
- **Initialization still commits**: even with the flag set, initialization records the graph's initial commit in the parent repository rather than leaving changes staged or unstaged. The graph's contract is that its state is versioned from creation onward, and history-dependent commands rely on that initial commit existing.
- **Commit scope over convenience**: the initial commit is confined to the initialization's own footprint even when the graph root and the parent repository root coincide. Sweeping in whatever else happened to be in the working tree would be surprising and, at a project root, potentially large.
- **Local-state exclusion is self-contained** (resolved in Clarifications): the exclusion lives inside the graph's own local-state directory rather than in the host project's ignore-rules file, so no file `arc` does not own is ever created or modified. This satisfies Constitution XI without a confirmation prompt, behaves identically whether the graph sits at a repository root or in a subfolder, and needs no tracking of its own since local state is never meant to reach a clone.
- **Detection is by upward search**: repository context is determined by walking up from the target, which is how version control itself resolves context. This makes the tool's judgment agree with what every other tool in the project sees.
- **Existing failure conditions are retained**: "target is not a directory," "version control unavailable," and "already an initialized graph" all continue to apply in both modes and are unchanged by this feature.
- **Layout collisions are refused, not merged** (resolved in Clarifications): an existing folder sharing a canonical layout name is refused regardless of whether it is empty, rather than being adopted. Adopting a non-empty one would silently turn the user's unrelated files into graph content; refusing an empty one too keeps the rule simple to state, simple to test, and free of per-folder rollback bookkeeping. Initializing into a subfolder is the supported alternative, so the refusal never leaves the user stuck.
- **Machine-readable contract grows, never breaks** (resolved in Clarifications): the new repository-root field is additive and unconditional. Existing fields keep their names and meanings, so `--json` remains the stable scriptable contract Constitution X requires.
- **Foreign files are the host project's, and stay that way** (Bugfix BUG-001, 2026-09-04): a graph root shared with a host project contains markdown `arc` neither wrote nor understands. The distinguishing test is the one the node format already mandates — `"@id"` and `"@type"` in the yaml front matter — so no new marker file, manifest or allow-list is introduced. Commands differ only in how they meet a foreign file: `apply` never opens one, because it reads only what its patch names; `lint` and `grep` must walk, so they index and skip.
- **No migration path**: graphs that were already created as nested repositories by the current behavior are out of scope. This feature prevents new occurrences; it does not detect or repair existing ones.
- **Scope of the cross-command requirement**: FR-021 through FR-025 describe correctness properties every git-backed command must satisfy. Some are already met by existing behavior; this feature's obligation is to establish them as verified properties across the whole command set, not to assume they hold.
