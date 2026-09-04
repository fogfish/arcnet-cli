# Phase 0 Research: Graph Initialization Inside an Existing Git Repository

**Feature**: `031-init-existing-git-repo` | **Date**: 2026-09-04 | **Plan**: [plan.md](plan.md)

All findings below were verified empirically against real `git` before being
committed to, not taken from memory. Each decision names the probe that
established it.

---

## D1 — Repository-context detection primitive

**Decision**: `git -C <dir> rev-parse --show-toplevel`. Exit 0 with the
repository root on stdout means the directory is inside a work tree; a non-zero
exit (`fatal: not a git repository (or any of the parent directories): .git`)
means it is not.

**Rationale**: This is the same upward search git itself performs, so `arc`'s
judgment about "am I inside a project" agrees by construction with what every
other tool in that project sees (spec Assumptions, "Detection is by upward
search"). It answers FR-001, FR-002 and FR-003 in one call: git stops at the
first `.git` it finds walking up, which is exactly FR-003's "innermost
enclosing repository" rule — no manual ancestor loop, no submodule/worktree
special-casing.

**Probe**: from a nested `sub/` inside a repo, returned the repo root; from a
directory outside any repo, exited non-zero with the `fatal:` message.

**Alternatives considered**:
- `git rev-parse --is-inside-work-tree` — answers the boolean but not *which*
  repository, and FR-027 needs the root path itself for the machine-readable
  `repository` field. Rejected: would require a second call.
- Manually walking parents looking for a `.git` entry — reimplements git's own
  resolution and gets `.git`-as-a-file (worktrees, submodules) wrong. Rejected
  as a reimplementation of a solved problem.

---

## D2 — Detection for a target that does not yet exist (FR-002)

**Decision**: the caller walks up from the requested target to the nearest
**existing** ancestor directory, then runs D1's probe there. `git -C` requires
its argument to exist, so probing a not-yet-created target directly fails for
the wrong reason.

**Rationale**: `arc init ./new-folder` inside a project must be refused
(User Story 1, scenario 3) *before* `ResolveLocalRoot` creates the folder —
otherwise the refusal leaves the folder behind, violating FR-005. Resolving the
nearest existing ancestor first is what makes the check possible pre-creation.

**Probe**: `git -C <existing ancestor> rev-parse --show-toplevel` returns the
correct root; the ancestor walk is plain path arithmetic above it.

**Consequence for ordering**: detection MUST run *before* `resolveLocalRoot`.
This inverts one step of the current `service.Init` sequence and is the reason
the parent-repo value is resolved in `cmd` and passed in (see plan's Structure
Decision) rather than probed from inside the service after the root exists.

---

## D3 — Confining the initial commit to the initialization's footprint (FR-013, FR-014)

**Decision**: two changes, both required together.

1. **Stage by explicit pathspec**: `git add -- <path> <path> …` listing exactly
   the files the layout wrote, replacing `git add -A -- .` for this path.
2. **Commit by explicit pathspec**: `git commit -m <msg> -- <path> <path> …`.

**Rationale**: each closes a different hole, and neither alone is sufficient.

- `git add -A -- .` is correctly scoped when the graph is a *subfolder* (the
  existing BUG-002 fix), but when the graph root **is** the repository root
  `-- .` covers the entire project — it would stage every unrelated modified and
  untracked file the user had lying around. FR-014 forbids exactly this.
- Even with staging fixed, a plain `git commit -m` commits **the whole index**,
  including anything the user had staged themselves before running `arc init`.
  A pathspec commit ignores the rest of the index entirely.

**Probe**: in a repo with a user-staged `staged.txt` and a modified `README.md`,
`git add -- Entity/.gitkeep` followed by `git commit -m … -- Entity/.gitkeep`
produced a commit containing only `Entity/.gitkeep`; `staged.txt` remained
staged and uncommitted, `README.md` remained modified. This is FR-014 and User
Story 2 scenario 3 satisfied exactly.

**Alternatives considered**:
- `git stash` around the operation — mutates the user's working state, and a
  failure mid-way could strand their changes in the stash. Rejected as far more
  invasive than the problem.
- `git commit --only <paths>` — `--only` is the long form of the pathspec
  behavior already obtained above; no advantage, less familiar.

---

## D4 — Local-state exclusion via `.arc/.gitignore` (FR-016, FR-017)

**Decision**: write `.arc/.gitignore` containing a single `*` line. Do not write
or modify any `.gitignore` at the graph root, in either mode.

**Rationale**: resolved in the spec's Clarifications. A `*` rule inside `.arc/`
excludes that directory's entire contents, including the rule file itself, so
nothing under `.arc/` is ever tracked and no file `arc` does not own is created
or touched (Constitution XI). It behaves identically whether the graph is at a
repository root or nested, which is why it replaces the root `.gitignore` write
in **both** modes rather than only the new one.

**Probe**: in a host repo containing `sub/.arc/.gitignore` (`*`) and
`sub/Entity/.gitkeep`, `git status --porcelain` after staging showed
`A sub/Entity/.gitkeep` and no mention of `sub/.arc` at all — the directory is
invisible to git without any entry in the host's own `.gitignore`.

**Nuance that matters for D5**: the `*` rule ignores the *contents* of `.arc/`,
not the directory entry itself. `git check-ignore -q sub/.arc` therefore exits
non-zero ("not ignored") even though nothing inside it will ever be tracked.
This is correct git semantics and is not a defect — but it means
`check-ignore` must be pointed at the right path when used for FR-020.

**Migration note**: the existing `cmd/arc/ctrl/init_test.go` suite asserts on a
root `.gitignore` containing `.arc/`. Those assertions change. SC-008 governs
what must stay true: local state excluded from tracking, working tree clean
after init — the observable outcome, not the file location.

---

## D5 — Detecting a target excluded by the host project's ignore rules (FR-020)

**Decision**: `git check-ignore -q <graph dir>` — exit 0 means the path is
ignored by the parent project, and initialization is refused.

**Rationale**: without this check the failure still happens, but late and
incomprehensibly: the layout is written, `git add` silently matches nothing
because everything is ignored, and `git commit` fails with "nothing to commit."
FR-020 converts that into an upfront, explanatory refusal.

**Probe**: with `vendor/` in `.gitignore`, `git check-ignore -q vendor/g` exited
0; a non-ignored path exited non-zero.

**Ordering**: this check runs only when a parent repository was found, and only
before any write.

---

## D6 — Where the new logic lives (answers the `/speckit-plan` constraint)

**Decision**: repository-root resolution and ignore-status probing happen in
`cmd/arc/ctrl` via the git adapter, and are passed into `service.Init` as plain
values on a new `kernel.InitOpts`. `port.VCS` gains only `StagePaths`; the
service's own port surface does not grow to accommodate detection.

**Rationale**: this was put to the user as an explicit choice during planning,
because the instruction "limit changes at `cmd` and adapters level; ask if a
service requires modification" cannot be satisfied literally — 11 of the 31
requirements live inside `service.Init`'s guard sequence, `writeLayout` and
`rollback`, all unexported with no seam, and Constitution Principle III forbids
relocating them to `cmd`. The user selected the **minimal service surface**
option.

What this buys, concretely:
- `port.VCS` grows by one method (`StagePaths`) instead of three
  (`RepoRoot`, `IsIgnored`, `StagePaths`).
- `internal/app/ctrl/adapter/mock.VCS` needs one new method, not three.
- Detection is an *environment probe*, not a business rule; the service still
  owns every *decision* made from the probe's result, which is where Principle
  III actually draws the line.
- D2's ordering constraint (detect before the directory exists) is satisfied
  naturally, because `cmd` probes before calling into the service at all.

**Alternatives considered**: full hexagonal (service probes through a widened
port) — more idiomatic in the abstract, but wider port, larger service diff,
every existing fake updated, and it fights D2's ordering. Rejected by the user.
cmd-only with a frozen service — rejected as unimplementable: it cannot satisfy
FR-016 or FR-018/019, which live inside unexported service functions.

---

## D7 — Scope of FR-021–FR-025 (cross-command parent-repo correctness)

**Decision**: verification-first. Add E2E coverage exercising `arc apply`,
`arc revert` and `arc lint` against a graph nested inside a larger repository,
then fix only what actually fails.

**Rationale**: the git adapter already carries targeted nested-repo fixes from
BUG-002 / spec 016 FR-021 — `-- .` on `StageAll`, `--relative` on
`ChangedPaths`, `./` on `ShowFile`, `-- .` on `CommitsMatching`. The spec's own
Assumptions say these are believed-correct but unproven, and that this feature's
obligation is to establish them as *verified*, not to assume them. Auditing and
"hardening" working code without a failing test is speculative churn that
expands the diff and risks regressions in paths this feature never needed to
touch.

**Expected outcome**: most scenarios pass on the existing adapter. Any that fail
become their own narrowly-scoped fix task.

---

## Resolved unknowns

No `NEEDS CLARIFICATION` markers remain. The four spec-level ambiguities were
settled in the `/speckit-clarify` session (see spec.md § Clarifications), and
the two plan-level ones — service-modification scope and FR-021–025 scope — were
put to the user during this planning session and answered (D6, D7).
