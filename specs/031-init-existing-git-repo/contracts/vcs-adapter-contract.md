# Contract: `internal/adapter/git` additions

**Feature**: `031-init-existing-git-repo` | **Plan**: [plan.md](../plan.md)

Every behavior below was verified against real `git` during Phase 0
(see [research.md](../research.md) D1–D5). Exit-code semantics are part of the
contract, not incidental.

---

## A1 — `RepoRoot(ctx, dir) (string, error)`

**Invocation**: `git -C <dir> rev-parse --show-toplevel`

| Outcome | Adapter returns | Meaning |
|---|---|---|
| Exit 0 | `(<trimmed stdout>, nil)` | `dir` is inside a work tree; value is the innermost repository root |
| Exit non-zero, output contains `not a git repository` | `("", nil)` | Not inside a repository — an **expected outcome, not an error** |
| Any other failure | `("", ErrGitRevParse.With(err))` | Genuine failure (git missing, permission) |

The `("", nil)` shape follows the precedent already set by `IsTracked` and
`ShowFile` in this adapter: distinguish an expected negative answer from a real
failure rather than making the caller parse an error string.

`dir` **must exist** — `git -C` fails otherwise. Resolving a not-yet-existing
target to its nearest existing ancestor is the caller's job (research D2), not
the adapter's; the adapter stays a thin git wrapper.

Handles submodules, linked worktrees and `.git`-as-a-file for free, because git
performs the resolution. FR-003's "innermost wins" is git's own behavior.

---

## A2 — `IsIgnored(ctx, dir, path) (bool, error)`

**Invocation**: `git -C <dir> check-ignore -q <path>`

| Outcome | Adapter returns | Meaning |
|---|---|---|
| Exit 0 | `(true, nil)` | Path is excluded by some ignore rule |
| Exit 1 | `(false, nil)` | Not excluded — **expected, not an error** |
| Exit ≥2 | `(false, ErrGitCheckIgnore.With(err))` | Genuine failure |

Exit 1 is `check-ignore`'s documented "no path matched" status and must not be
treated as a failure — the same `exec.ExitError` code-1 discrimination
`IsTracked` already performs.

**Semantic caveat that matters** (research D4): a `*` rule inside `.arc/.gitignore`
excludes the *contents* of `.arc/`, not the directory entry. `check-ignore` on
`.arc` itself therefore reports "not ignored" while nothing inside it is ever
tracked. Correct git semantics, not a defect. This method is used only for
FR-020 — asking whether the *host project* excludes the graph directory — and
must not be repurposed to verify D4's exclusion mechanism.

---

## A3 — `StagePaths(ctx, dir, paths) error`

**Invocation**: `git -C <dir> add -- <path>…`

Stages exactly the listed graph-relative paths. Distinct from the existing
`StageAll`, which uses `-A -- .`: correctly scoped for a graph in a subfolder,
but equivalent to "everything in the project" when the graph root and the
repository root coincide.

| Property | Behavior |
|---|---|
| Empty `paths` | Return nil without invoking git — never degrade to staging everything |
| Path separator | Forward slashes, graph-relative (git's own convention, matching `fs.ValidPath`) |
| Failure | `ErrGitStage.With(err)` — unchanged sentinel |

`StageAll` is **not** removed from the adapter: `internal/app/graph/port.VCS`
still declares and uses it, and that port is untouched by this feature.

---

## A4 — `Commit(ctx, dir, message, paths) (string, error)`

**Invocation**: `git -C <dir> commit -m <message> -- <path>…`, then
`git -C <dir> rev-parse --short HEAD`.

The trailing pathspec is the load-bearing change. A plain `git commit -m`
commits **the entire index**, including files the user staged themselves before
running `arc init`. A pathspec commit ignores the rest of the index.

**Verified** (research D3): with a user-staged `staged.txt` and a modified
`README.md` present, a pathspec commit of `Entity/.gitkeep` produced a commit
containing only `Entity/.gitkeep`; `staged.txt` stayed staged and uncommitted,
`README.md` stayed modified. This is FR-014 and User Story 2 scenario 3.

The existing commit-then-`rev-parse` pattern, the `ErrGitCommit` sentinel, and
the reporter `Done`/`Error` calls are unchanged.

**Signature change**: this is `ctrl`'s `Commit`. The concrete `git.VCS` type
satisfies both `ctrl` and `graph` port interfaces structurally (ADR 001), and
`graph.port.VCS` declares `Commit` **without** the pathspec — so one Go method
cannot satisfy both. Resolution: add `CommitPaths` as a new method for ctrl and
leave `Commit` untouched for graph. This keeps `internal/app/graph` completely
unmodified, which the verification-first decision (research D7) requires.

---

## A5 — Reporter behavior

Unchanged from BUG-001's consolidation: three reported steps — availability,
preparing the repository, committing. `StagePaths` reports nothing of its own,
exactly as `StageAll` does not.

One consequence of `--skip-git-init`: the "Preparing git repository" step does
not run, because `vcs.Init` is not called. Under `--verbose` the user sees two
steps rather than three, which correctly reflects that no repository was
created.

---

## A6 — New error sentinels

Added to `internal/adapter/git`, following the existing `faults.Type` pattern:

```go
ErrGitRevParse    = faults.Type("git rev-parse failed")
ErrGitCheckIgnore = faults.Type("git check-ignore failed")
```

Added to `internal/app/ctrl/service`, following the existing `faults.Safe1`
pattern (`errNoCause` for guards with no underlying Go error):

```go
ErrInsideRepository   = faults.Safe1[string]("%s is inside an existing git repository; use --skip-git-init to add a graph to it")
ErrNoParentRepository = faults.Safe1[string]("%s is not inside a git repository; --skip-git-init requires an existing repository to add the graph to")
ErrLayoutCollision    = faults.Safe1[string]("%s already exists; initialize the graph into a subfolder instead")
ErrTargetIgnored      = faults.Safe1[string]("%s is excluded by the repository's ignore rules; the graph could never be committed")
```

Message wording satisfies FR-006 (names the repository root and the flag) and
FR-031 (names the conflicting path and the subfolder recovery).
