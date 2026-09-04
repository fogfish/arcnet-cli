# Implementation Notes: 031-init-existing-git-repo

Recorded during `/speckit-implement`. Holds the evidence tasks T001, T002 and
T048 ask for, so the claims they make are checkable rather than asserted.

---

## T001 — Green baseline

`go test ./...` and `go vet ./...` both pass on `031-init-existing-git-repo`
before any edit of this feature (commit `cadead2`).

### Assertions depending on a root `.gitignore` (research D4 predicts these change)

| Location | Assertion | Disposition |
|---|---|---|
| `cmd/arc/ctrl/init_test.go:151-154` (`TestInitCurrentDirectoryCreatesLayout`) | reads `<root>/.gitignore`, asserts it contains `.arc/` | **Changes** (T047) — becomes `.arc/.gitignore` containing `*` |
| `internal/app/ctrl/service/init_test.go:182` (`TestInitSuccessWritesLayoutAndCommits`) | `store.written[".gitignore"]` | **Changes** (T040/T046) — becomes `.arc/.gitignore` |
| `internal/app/ctrl/service/init_test.go:199` (`TestInitRollsBackOnCommitFailureWithoutCreatedRoot`) | rollback removed `.gitignore` | **Changes** (T044/T046) — footprint-scoped rollback removes `.arc/.gitignore` |

### SC-008's observable outcomes — must stay true, and their assertions unchanged

| Location | Outcome pinned |
|---|---|
| `cmd/arc/ctrl/init_test.go` `TestInitCurrentDirectoryCleanWorkingTree` | `git status --short` empty; `git ls-files` excludes `.arc/` |
| `cmd/arc/ctrl/init_test.go` `TestInitCurrentDirectorySingleCommit` | exactly one commit, canonical subject |
| `cmd/arc/ctrl/init_test.go` `TestInitCreatesExactlyTheTypeNamedFolders` | folder set unchanged (walk skips `.arc`) |

Not affected: `cmd/arc/lint/lint_test.go:97` and `cmd/arc/graph/apply_test.go:131`
write their own `.gitignore` **fixtures**; they do not observe `arc init`'s
output and need no change.

## T002 — staticcheck baseline

`staticcheck ./...` **cannot run** in this environment: the installed binary
predates the Go 1.27 toolchain and fails to decode standard-library export data
(`export data version 4 is greater than maximum supported version 2`) for every
package, emitting only `(compile)` errors and no analysis findings. There is
therefore no pre-edit staticcheck baseline to attribute new findings against;
`go vet ./...` is the working static gate for this feature and is clean before
and after (T057).

## T048 — User Story 4 triage (verification-first, research.md D7)

The five FR-021..FR-025 scenarios in `cmd/arc/graph/nested_repo_test.go` were
written first and run against the **unmodified** git adapter. Result:

| Scenario | Requirement | Outcome on the existing adapter |
|---|---|---|
| `arc apply` commits only the graph subtree, dirty sibling untouched | FR-022 | **PASS** (`StageAll`'s `-A -- .`) |
| `arc apply` parity, nested vs standalone | FR-025 | **PASS** |
| `arc revert` touches only the graph subtree | FR-023, FR-024 | **PASS** (`ChangedPaths --relative`, `ShowFile ./`) |
| `arc revert` parity, nested vs standalone | FR-025 | **PASS** |
| `arc lint` history check ignores a look-alike `Source-Id` commit elsewhere in the host repo | FR-023 | **PASS** (`CommitsMatching`'s `-- .`) |

### Round 2 — the root-sharing axis (T069, Bugfix BUG-001, 2026-09-04)

The table above is accurate and remains closed on its evidence. It is also
**incomplete**, and the way it is incomplete is the lesson worth recording:
every scenario in it nests the graph in a subfolder (`<repo>/kb`) and puts the
host's content in a *sibling* directory (`<repo>/other`). Nothing arc did not
write ever sat inside the tree a command walks, so the whole filesystem-walk
layer was unreachable by construction. Parity has two axes, and this feature
verified one of them:

| Axis | Meaning | Verified by | Outcome |
|---|---|---|---|
| Repository nesting | graph root **below** the repository root | `initHostGraph`, T048 | **PASS** on the existing adapter |
| Root sharing | graph root **shared** with files arc did not write | `initRootModeGraph`, T058-T061 | **FAILED** — three defects, fixed in T062-T066 |

Root-sharing triage, run against the unmodified walk layer:

| Scenario | Requirement | Outcome before the fix |
|---|---|---|
| `arc apply` beside `README.md`/`CONTRIBUTING.md`/`docs/` | FR-032, FR-033 | **FAIL** — `guardNoOldFormatNodes` parsed the host's markdown and aborted every apply |
| `arc apply` error names the operation it attempted | FR-034 | **FAIL** — read-path rejection reported as `failed to write` |
| `arc apply` read cost independent of host tree size | FR-033, SC-009 | **FAIL** — the walk read every `*.md` under the root |
| `arc lint` on a shared root | FR-035, SC-010 | **FAIL** — host markdown counted and reported as failing nodes |
| `arc grep` on a shared root | FR-035 | **PASS** — already indexed unparseable files as `Unreadable` |

`arc grep` passing alone is the tell: all three commands walk the same tree,
and only the one that already had a foreign-file index survived contact with a
shared root. That index is now lint's too (T065/T066), and apply no longer
walks at all (T062).

**One interpretation call, recorded because it deviates from FR-032's letter.**
FR-032 defines a foreign file as one that "does not declare both `"@id"` and
`"@type"`". Implemented literally, that silently retires two of lint's own
checks: `Entity/Broken.md` holding unresolved merge-conflict markers, or
garbage where its front matter belongs, declares neither key either — and
would have been skipped as host content. Two existing tests caught this
(`TestLintUnresolvedMergeConflictReportedOnce`,
`TestLintRecordsFrontMatterViolationWithoutAbortingWalk`), which is precisely
what they are for. `service.isForeignFile` therefore asks **location first**: a
file directly inside a folder named for a registered node type is arc's by
construction — arc created that folder, writes every node as
`<Type>/<id>.md`, and FR-015 refuses to initialize where such a folder already
exists — so anything there is a node, malformed if it will not parse, never
host content. Only elsewhere (the graph root itself, a `docs/` tree) does the
identity test decide. Same outcome for every file FR-032 was written about;
lint's coverage intact.

---

**T049, T050 and T051 are therefore closed as no-change-needed**, with the
passing tests as the evidence. BUG-002 / spec 016's nested-repository fixes are
now verified rather than assumed, and `internal/app/graph`,
`internal/app/lint` and `internal/adapter/fsys` are unmodified by this feature
exactly as [plan.md](plan.md) requires.

The lint scenario asserts **parity with the standalone run** rather than "0
failing": its patch fixture leaves an unrelated conformance violation, and what
FR-025 promises is that nesting changes nothing — an unscoped history query
would add a violation to the nested run alone.

---

## Deviations from the task list, and why

Three places where implementation diverged from a literal reading of the design
artifacts. Each is a refinement the artifacts' own contracts point at.

### 1. Staging and committing run at the graph root, not at `ParentRepo`

[data-model.md](data-model.md)'s state diagram writes `vcs.StagePaths(repo, …)`
and T035 says "direct staging and committing at `InitOpts.ParentRepo`". The
implementation passes the **graph root** in both modes.

[contracts/vcs-adapter-contract.md](contracts/vcs-adapter-contract.md) A3/A4 is
the precise statement and it says `git -C <dir>` with **graph-relative** paths —
which is only coherent when `dir` is the graph root. git resolves a pathspec
against the directory it runs in and walks up to the enclosing repository by
itself, so one call shape is correct whether the graph root is the repository
root or a subfolder of one. Passing `ParentRepo` would have required
repository-relative paths and a path-prefix computation that is fragile across
symlinked paths (see 3). Observable behavior is identical; verified by
`TestVCSCommitPathsFromNestedGraphDirectory`.

### 2. The commit's pathspec excludes `.arc/`

Not anticipated by any artifact, and forced by git. `git add -A -- .` skips
ignored paths silently, but `git add -- <path>` **naming** an ignored path fails
outright ("The following paths are ignored by one of your .gitignore files").
Since `.arc/.gitignore` excludes `.arc/`'s own contents (research D4), the
footprint's `.arc/` entries cannot appear in the pathspec. `footprint.Tracked()`
draws that line: the full footprint is still what rollback removes, only the
commit is narrowed. The observable outcome is unchanged and is what SC-008
pins — `.arc/` was never tracked before either.

### 3. `RepoRoot` reports the root in the caller's own path spelling

[contracts/vcs-adapter-contract.md](contracts/vcs-adapter-contract.md) A1
specifies `rev-parse --show-toplevel`. That reports a fully **symlink-resolved**
path — `/private/var/…` where the caller said `/var/…`, the everyday case on
macOS, where `TMPDIR` lives under a symlinked `/var`. A different string for the
same directory breaks two things [data-model.md](data-model.md) relies on:
`ParentRepo` being "comparable to `dir` by string equality", and FR-027's
`repository == path ⟺ the graph root is the repository root`.

`RepoRoot` therefore reads `--show-toplevel` **and** `--show-cdup` from a single
invocation, and returns `dir` joined with the relative walk upward — the same
directory, in the caller's spelling. `--show-toplevel` is still what
distinguishes being inside a work tree. Verified by
`TestVCSRepoRootKeepsCallerPathSpelling`, and end to end by
`arc init --json --skip-git-init <repo-root>` reporting `repository == path`.

### Guard ordering: "already initialized" precedes the repository guards

[data-model.md](data-model.md)'s diagram places R1/R2/R3 above the `.arc/`
check. Implemented the other way round, because re-running `arc init` on an
existing graph — which is itself a repository — would otherwise answer "this is
inside an existing git repository" instead of "this is already a graph". FR-011
and [contracts/cli-contract.md](contracts/cli-contract.md) C5 both require
today's message for that case, unchanged. The ordering property that is actually
load-bearing — **every** guard precedes every write (FR-005) — holds either way,
and is asserted directly by `TestInitGuardsRunBeforeRootIsResolved`.

---

## T057 — final gate

`go build ./...`, `go vet ./...` and `go test ./...` all pass (18 packages ok, 0
failures). `staticcheck` remains unrunnable for the reason recorded under T002,
unchanged from the pre-edit baseline.

## TN15 — release/versioning impact

No major bump. The `--json` surface change is purely additive: `path`, `commit`
and `foldersCreated` keep their names, types and meanings, and `repository` is a
new key existing consumers can ignore (FR-028, verified by
`TestInitJSONReportsRepositoryInBothModes` and by `arc init --json` emitting
exactly `["commit","foldersCreated","path","repository"]`).

The one behavior change on an existing path is where the local-state exclusion
is written — `.arc/.gitignore` instead of `<graph root>/.gitignore`. SC-008's
observable outcomes are unchanged and still asserted: local state untracked, a
clean working tree after init, exactly one commit. A graph initialized by an
earlier release keeps its root `.gitignore` and keeps working; nothing reads
that file, so no migration is needed.
