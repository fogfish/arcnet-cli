# Contract: `arc init` command surface

**Feature**: `031-init-existing-git-repo` | **Plan**: [plan.md](../plan.md)

The user-facing contract. Constitution X treats `--json` as the stable
scriptable surface; the human line is a free-to-change convenience.

---

## C1 — Flag

```
arc init [<dir>] [--skip-git-init]
```

| Property | Value | Source |
|---|---|---|
| Long form | `--skip-git-init` | FR-008 |
| Shorthand | none | Constitution IX — shorthands reserved for frequent flags; this is advanced usage |
| Default | `false` | FR-004 |
| Type | boolean | — |
| Help text | `Create the graph inside the existing repository instead of starting a new one (advanced)` | FR-030 |

Interacts with no other flag. `--json`, `--quiet`, `--verbose` behave as they do
today.

---

## C2 — Mode selection

| Target inside a repository? | `--skip-git-init` | Outcome |
|---|---|---|
| No | absent | Create repository, commit. Today's behavior (FR-007). |
| No | present | **Refuse** — `ErrNoParentRepository` (FR-012). |
| Yes | absent | **Refuse** — `ErrInsideRepository` (FR-004). |
| Yes | present | Use parent repository, commit into it (FR-009). |

---

## C3 — Machine-readable output

Success, both modes:

```json
{
  "path": "/abs/path/to/graph",
  "commit": "a1b2c3d",
  "foldersCreated": ["Source", "Entity", "Resource", "Reference",
                     "timeline/yearly", "timeline/monthly",
                     "_schema/Class", "_schema/Property"],
  "repository": "/abs/path/to/repository/root"
}
```

| Guarantee | Requirement |
|---|---|
| `path`, `commit`, `foldersCreated` keep their names, types and meanings | FR-028 |
| `repository` is present in **both** modes and is never empty or null | FR-027 |
| `repository == path` ⟺ the graph root is the repository root | FR-027 |
| No mode boolean is emitted; mode is derived by comparison | FR-027, clarification 4 |

A standalone init reports `repository` equal to `path` — the repository it just
created is at the graph root. Consumers therefore never branch on output shape.

---

## C4 — Human-readable output

Unchanged single concise line (spec 002 FR-016, BUG-001) in the default case:

```
✔ Initialized empty knowledge graph at /path (commit a1b2c3d)
```

With `--skip-git-init`, the line additionally states that an existing repository
was used rather than a new one (FR-026):

```
✔ Initialized empty knowledge graph at /path (commit a1b2c3d) in existing repository /repo/root
```

Constraints carried over from BUG-001 and unchanged: exactly one `\n`, no
embedded newline passed to a single lipgloss `Render()` call, plain text rather
than `StatusOK` green, icon carries the visual confirmation. The existing
`PostRunE` hint is unchanged.

---

## C5 — Failure contract

All exit non-zero (FR-029) and render through `cmd/arc/main.go`'s single
`humanize` path — never a raw Go error value (Constitution IX). The codebase
exits `1` uniformly for every failure; this feature introduces no new exit code.

| Condition | Message shape | Requirement |
|---|---|---|
| Inside a repository, no flag | names the repository root **and** names `--skip-git-init` | FR-006 |
| Flag set, no enclosing repository | states there is no parent project to add the graph to | FR-012 |
| Layout collision (file or folder) | names the conflicting path **and** states the subfolder recovery | FR-015, FR-031 |
| Target ignored by parent rules | states the target is excluded from version control | FR-020 |
| Already initialized | today's message, unchanged | FR-011 |
| Target not empty (default mode only) | today's message, unchanged | FR-007 |
| Target is a file | today's message, unchanged | spec Edge Cases |
| Git unavailable | today's message, unchanged | spec Edge Cases |

Every one of these leaves the filesystem and the parent repository's history
exactly as found (FR-005, FR-018, SC-006).

---

## C6 — On-disk contract

| Path | Standalone | `--skip-git-init` | Requirement |
|---|---|---|---|
| Eight canonical folders, each with `.gitkeep` | created | created | unchanged |
| `_schema/` seed files | created | created | unchanged |
| `.arc/.gitignore` containing `*` | created | created | FR-016 |
| `<graph root>/.gitignore` | **no longer written** | never written | FR-016, FR-017 |
| Host project's existing `.gitignore` | n/a | **never read, created or modified** | FR-017 |

The change to standalone init is intentional and uniform (clarification 1). The
observable outcome SC-008 protects is unchanged: local state stays out of
version control, and the working tree is clean after init.

---

## C7 — Port contract addition

`internal/app/ctrl/port.VCS` gains exactly one method:

```go
// StagePaths stages exactly the given graph-relative paths, and nothing
// else. Unlike StageAll it never sweeps in unrelated changes, which is
// required when the graph root and the repository root coincide.
StagePaths(ctx context.Context, dir string, paths []string) error
```

…and replaces its use of `Commit` with a pathspec-carrying `CommitPaths`, so
the commit is confined to the footprint regardless of what else the user had
staged (research D3):

```go
// CommitPaths commits exactly the given paths, ignoring whatever else the
// index holds. Distinct from Commit, which commits the whole index.
CommitPaths(ctx context.Context, dir, message string, paths []string) (hash string, err error)
```

`CommitPaths` is a **new** method rather than a signature change to `Commit`.
The one concrete `git.VCS` type satisfies both `ctrl` and `graph` port
interfaces structurally (ADR 001), and `internal/app/graph/port.VCS` declares
`Commit` without a pathspec — a single method cannot satisfy both shapes. Adding
a new method leaves `internal/app/graph` entirely unmodified, which the
verification-first decision (research D7) requires.

`IsAvailable` and `Init` are unchanged. `StageAll` and `Commit` are retained on
the git adapter for `internal/app/graph`, but are dropped from the **ctrl**
port, which no longer calls either.

Detection (`rev-parse --show-toplevel`, `check-ignore`) is deliberately **not**
on this port: it is resolved in `cmd` through the git adapter and passed in as
`kernel.InitOpts` values (research D6).
