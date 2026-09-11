# Phase 1 Data Model: Explicit Serve Context & Shareable Graph State

**Feature**: `034-serve-dir-public-state` | **Date**: 2026-09-11

This feature introduces **no new domain type**. It changes the value of one layout
constant set, adds one boolean to an existing options struct, and adds four error
sentinels. Everything below is an edit to something that already exists.

---

## 1. Graph state layout (`internal/app/ctrl/service/init.go` constants)

### Before

```go
arcStateDir      = ".arc"
arcStateMarker   = ".arc/.gitkeep"
arcIgnorePath    = ".arc/.gitignore"
arcIgnoreContent = "*\n"
```

### After

```go
arcStateDir        = ".arc"
arcStateMarker     = ".arc/.gitkeep"
arcIgnorePath      = ".arc/.gitignore"
arcIgnoreContent   = "cache/\n"
arcCacheDir        = ".arc/cache"
arcCacheIgnorePath = ".arc/cache/.gitignore"
arcCacheIgnoreContent = "*\n"
```

`arcStateDir` keeps its role as the graph marker (`guardIsGraph`,
`guardNotAlreadyInitialized`) — unchanged.

### On-disk result of `arc init`

```
my-graph/
├── Source/ Entity/ Resource/ Reference/     # unchanged
├── timeline/{yearly,monthly}/               # unchanged
├── _schema/{Class,Property}/                # unchanged
├── .arc/
│   ├── .gitkeep                             # tracked   (marker, unchanged path)
│   ├── .gitignore                           # tracked   — content "cache/" (was "*")
│   ├── config.yml                           # tracked when present (newly shareable)
│   └── cache/
│       └── .gitignore                       # untracked — content "*", ignores itself
└── .git/
```

### Invariants

| # | Invariant | Enforced by |
|---|-----------|-------------|
| I1 | `.arc/` exists ⇔ the directory is an initialized graph | `guardIsGraph` (unchanged) |
| I2 | `.arc/.gitkeep` and `.arc/.gitignore` are tracked | `footprint.Tracked()` + init's commit pathspec |
| I3 | Nothing under `.arc/cache/` is ever tracked | two independent rules — `cache/` in the tracked parent, `*` in the child |
| I4 | The working tree is clean immediately after `arc init` | I2 ∧ I3 |
| I5 | A clone carries I1, I2 and I3 | `.arc/.gitignore` is tracked and is not self-ignoring (research D3) |
| I6 | `.arc/cache/` does **not** exist in a fresh clone | git cannot materialise a directory whose only file ignores itself |

I6 is a deliberate, documented consequence, not a defect: cache content is
machine-local by definition, and I5 guarantees the rule protecting it is present
before any future writer creates the directory.

---

## 2. Write-order / footprint (`layoutPaths`, `writeLayout`)

`layoutPaths` appends, in order, after the folder stubs and seed files:

```
.arc/.gitkeep          →  ""
.arc/.gitignore        →  arcIgnoreContent       ("cache/\n")
.arc/cache/.gitignore  →  arcCacheIgnoreContent  ("*\n")
```

One path is added. `contentFor` gains one branch. `absentDirs` picks up
`.arc/cache` for free and sorts it deepest-first, so rollback removes
`.arc/cache` before `.arc` with no change.

### `footprint.Tracked()` — predicate change

| Path | Before | After |
|------|--------|-------|
| `Source/.gitkeep`, `_schema/…` | tracked | tracked |
| `.arc/.gitkeep` | **excluded** | **tracked** |
| `.arc/.gitignore` | **excluded** | **tracked** |
| `.arc/cache/.gitignore` | *(did not exist)* | **excluded** |

Predicate: `path == arcCacheDir || strings.HasPrefix(path, arcCacheDir+"/")`
(was the same test against `arcStateDir`).

---

## 3. `kernel.InitOpts` — one added field

```go
type InitOpts struct {
	ParentRepo    string
	SkipGitInit   bool
	TargetIgnored bool
	StateIgnored  bool   // NEW
}
```

**StateIgnored**: whether the parent repository's ignore rules exclude the graph's
`.arc/` state directory. Meaningful only when `ParentRepo != ""`. Resolved in
`cmd/arc/ctrl.resolveRepoContext` via
`probe.IsIgnored(ctx, repo, <rel>/.arc/.gitkeep)` — the same probe method already
used for `TargetIgnored`, called a second time.

The probe names the **marker file**, not the directory. A directory-only pattern —
`.arc/`, the rule a user is most likely to have written — matches only a path git
knows to be a directory, and nothing exists yet at probe time (FR-005), so asking
about a bare `<rel>/.arc` answers "not ignored" and the guard misses the case it
exists for. Asking about the file `arc init` must stage is correct for that pattern
and strictly more sensitive: a rule excluding the directory excludes its contents
too. (Corrected while validating quickstart V6; see research D5.)

### Guard rules in `guardRepositoryContext` (evaluation order)

| Rule | Condition | Error | Status |
|------|-----------|-------|--------|
| R1 | `!SkipGitInit && ParentRepo != ""` | `ErrInsideRepository` | unchanged |
| R2 | `SkipGitInit && ParentRepo == ""` | `ErrNoParentRepository` | unchanged |
| R3 | `SkipGitInit && TargetIgnored` | `ErrTargetIgnored` | unchanged |
| **R4** | `SkipGitInit && StateIgnored` | `ErrStateIgnored` | **new** (FR-017) |

R4 sits last: a wholly-ignored target (R3) is the more fundamental complaint and
keeps priority. Like R1-R3, R4 reads no filesystem and refuses before any write.

---

## 4. Error sentinels

### `internal/app/ctrl/service/errors.go` — one added

```go
ErrStateIgnored = faults.Safe1[string](
	"%s/.arc is excluded by the repository's ignore rules; a clone would not be a usable graph")
```

### `internal/app/graph/service/errors.go` — three added (research D2)

```go
ErrGraphDirNotFound     = faults.Safe1[string]("%s does not exist")
ErrGraphDirNotDirectory = faults.Safe1[string]("%s is not a directory")
ErrGraphDirUnreadable   = faults.Safe1[string]("%s cannot be read")
```

`ErrNotAGraph` is unchanged and remains the answer for a directory that exists,
is readable, and simply has no `.arc/`.

---

## 5. `arc serve` argument (`cmd/arc/graph/serve.go`)

| Property | Before | After |
|----------|--------|-------|
| `Use` | `serve` | `serve [<dir>]` |
| `Args` | `cobra.NoArgs` | `cobra.MaximumNArgs(1)` |
| dir source | `filepath.Abs(".")` | `resolveGraphDir(args)` |

```go
func resolveGraphDir(args []string) (string, error) {
	if len(args) == 1 {
		return filepath.Abs(args[0])
	}
	return filepath.Abs(".")
}
```

`filepath.Abs` resolves a relative argument against the process working directory
(FR-003) and returns an absolute argument unchanged. `cobra.MaximumNArgs(1)`
produces the usage error required by FR-007 for a second argument.

No state is added to the command: `buildServer(ctx, dir)` and all eight tool
handlers already take `dir` as a parameter. Confinement (FR-005) is already
provided by `os.DirFS(root)` inside `fsys.Local` and needs no new code.

---

## 6. Entity mapping — spec → code

| Spec entity | Existing code | Change |
|-------------|---------------|--------|
| Graph directory | `dir string` threaded through `buildServer` and every handler | resolution site only |
| Graph state directory | `arcStateDir` / `guardIsGraph` | ignore-rule content only |
| Local cache location | *(none)* | `arcCacheDir` + its ignore file |
| Graph configuration | `configkernel.ConfigPath` = `.arc/config.yml` | no code change; becomes tracked as a consequence of I2 |
