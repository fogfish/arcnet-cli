# Phase 1 Data Model: Graph Initialization Inside an Existing Git Repository

**Feature**: `031-init-existing-git-repo` | **Date**: 2026-09-04 | **Plan**: [plan.md](plan.md)

This feature adds one value type and extends one existing one, both in
`internal/app/ctrl/kernel`. No `internal/core` domain type changes — nothing
here is graph content.

---

## `kernel.InitOpts` (new)

The environment facts `cmd` resolves before calling the service, plus the user's
flag. Carrying them as a struct rather than positional parameters keeps
`service.Init`'s signature readable as it grows, and lets later options land
without another signature break.

| Field | Type | Meaning |
|---|---|---|
| `ParentRepo` | `string` | Root of the innermost repository enclosing the target, resolved by `cmd` per research D1/D2. Empty string means no enclosing repository was found. |
| `SkipGitInit` | `bool` | The `--skip-git-init` flag as given by the user. |
| `TargetIgnored` | `bool` | Whether the parent repository's ignore rules exclude the target (research D5). Meaningful only when `ParentRepo` is non-empty. |

**Validation rules** (enforced by `service.Init`, all before any write):

| Rule | Condition | Error | Requirement |
|---|---|---|---|
| R1 | `!SkipGitInit && ParentRepo != ""` | `ErrInsideRepository` | FR-004, FR-005, FR-006 |
| R2 | `SkipGitInit && ParentRepo == ""` | `ErrNoParentRepository` | FR-012 |
| R3 | `SkipGitInit && TargetIgnored` | `ErrTargetIgnored` | FR-020 |

`ParentRepo` is a resolved absolute path, comparable to `dir` by string
equality — `cmd` resolves both through the same `filepath.Abs` path, so the
"graph root is the repository root" case is a plain equality test.

---

## `kernel.InitResult` (extended)

| Field | JSON | Status | Meaning |
|---|---|---|---|
| `Root` | `path` | unchanged | Graph root. |
| `CommitHash` | `commit` | unchanged | Short hash of the initial commit. |
| `FoldersCreated` | `foldersCreated` | unchanged | Canonical folders created. |
| `Repository` | `repository` | **new** | Root of the repository the commit landed in (FR-027). |

`Repository` is populated in **both** modes and is never empty: standalone init
sets it to the graph root it just created a repository at, so it equals `path`;
`--skip-git-init` sets it to `InitOpts.ParentRepo`, an ancestor path. FR-027's
"consumers distinguish the modes by comparing this value to the graph location"
follows directly, with no mode boolean (FR-028 keeps the three existing fields'
names and meanings frozen).

---

## `Footprint` (new, unexported — `internal/app/ctrl/service`)

Not a kernel type; an implementation value threaded through the service.

The ordered list of graph-relative paths this run wrote. Built by `writeLayout`
as it writes, and consumed by exactly two callers:

- **staging and committing** — the explicit pathspec of research D3, which is
  what makes FR-013/FR-014 true;
- **rollback** — the precise set to remove on failure, replacing today's
  "remove every path the static layout *could* have written" approach.

Replacing the static removal list with a recorded footprint is what lets FR-018
("remove every file it created") and FR-019 ("never delete a pre-existing
directory") both hold in a directory the tool does not exclusively own. Today's
rollback is safe only because `guardTargetEmpty` guarantees the directory was
empty; FR-010 removes that guarantee for `--skip-git-init`, so the footprint
becomes load-bearing rather than a refinement.

---

## Layout collision surface (FR-015)

The guard checks two disjoint classes before any write:

| Class | Checked against | Fails when |
|---|---|---|
| Files | Every path `writeLayout` would write — the eight `<folder>/.gitkeep` markers, every `_schema/` seed file, and `.arc/.gitignore` | The path exists |
| Folders | The eight canonical folder names of `kernel.DefaultLayout.Folders` | The folder exists, empty or not (clarification 2) |

Both use `fsys.Store.Stat`, whose `fs.FileInfo.IsDir()` distinguishes the two —
no `fsys` interface change is needed. The folder rule is deliberately stricter
than the file rule: an existing folder is refused regardless of contents, and
the failure names the recovery path (FR-031).

---

## State transition — `service.Init`

Ordering is a correctness property, not a style choice. Every guard precedes
every write, so FR-005 ("filesystem left exactly as found") holds on refusal
without relying on rollback at all.

```
                    ┌─ R1 violated ──────────────► ErrInsideRepository      (no writes)
                    ├─ R2 violated ──────────────► ErrNoParentRepository    (no writes)
  [cmd probes]      ├─ R3 violated ──────────────► ErrTargetIgnored         (no writes)
  ParentRepo   ──►  ├─ .arc/ exists ─────────────► ErrAlreadyInitialized    (no writes)
  TargetIgnored     ├─ collision (file/folder) ──► ErrLayoutCollision       (no writes)
  SkipGitInit       ├─ !SkipGitInit && !empty ───► ErrTargetNotEmpty        (no writes)
                    └─ git unavailable ──────────► ErrGitUnavailable        (no writes)
                                │
                                ▼
                       resolveLocalRoot
                                │
                                ▼
                    writeLayout → Footprint
                                │
                    ┌───────────┴───────────┐
              SkipGitInit               otherwise
                    │                       │
                    │                  vcs.Init(dir)
                    └───────────┬───────────┘
                                ▼
                  vcs.StagePaths(repo, Footprint)
                                ▼
                   vcs.Commit(repo, msg, Footprint)
                                ▼
                          InitResult
```

Any failure at or below `writeLayout` rolls back the recorded footprint and,
when `resolveLocalRoot` created the directory, removes the directory itself
(FR-018, FR-019).

Two ordering notes:

- **Guards move ahead of `resolveLocalRoot`.** Today the directory is created
  first and rolled back on refusal. FR-005 wants a refusal that never touched
  the disk, and research D2 requires detection before the directory exists.
  Both point the same way.
- **`vcs.IsAvailable` stays a guard, not a step.** It is unchanged and applies
  in both modes.
