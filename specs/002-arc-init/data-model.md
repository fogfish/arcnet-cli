# Phase 1 Data Model: `arc init`

Value types are immutable (constitution Principle IV) and carry no Cobra, `os/exec`, or raw `os.*` filesystem types (constitution Principle III, VII) — filesystem-shaped values are stdlib `io/fs`/`io.Reader`/`io.Writer` types only, with `os` itself confined entirely to `internal/adapter/fsys` (research.md D3).

## Domain entities (`internal/app/ctrl/kernel`)

### GraphRoot

Represents the resolved location of a graph, before or after initialization.

| Field | Type | Notes |
|---|---|---|
| `Root` | `string` | The resolved local directory path passed to `fsys.Mounter.Mount` (research.md D3) |

**Note**: There is no separate `StateDir` field. `.arc`'s location is always the fixed, `Store`-relative path `".arc"` (relative, no leading or trailing slash, per stdlib `io/fs`'s own path convention — `fs.ValidPath`) — it is never derived by joining host-filesystem path segments with `filepath.Join`, since a `Store`-relative path stays meaningful independent of how `Store` is implemented.

### ArcNetCoreLayout

A static, pure description of what an empty graph must contain — used both to drive creation and (in a later feature) to drive `arc lint`'s structural check. Not user-configurable in this feature.

| Field | Type | Notes |
|---|---|---|
| `Folders` | `[]string` | `"sources"`, `"entities"`, `"resources"`, `"timeline/yearly"`, `"timeline/monthly"`, `"_meta"` — no leading/trailing slash, per stdlib `io/fs`'s path convention; each becomes real only once a placeholder file is `Store.Create`'d under it (research.md D3: there is no `Mkdir` step, and git itself does not track empty directories regardless) |
| `MetaStubs` | `map[string]string` | `"_meta/predicates.md"` → stub content, `"_meta/aliases.md"` → stub content — keys are `Store`-relative file paths, values are the literal file content `Init` writes via `Store.Create` |

**Rule**: `ArcNetCoreLayout` is a package-level constant value (`var DefaultLayout = ArcNetCoreLayout{...}`), not read from configuration — matches spec Assumption "only the CORE canonical layout is created by this command." Every `Folders` entry additionally gets a `.gitkeep` placeholder file (`<folder>.gitkeep`) so git tracks the otherwise-empty directory (spec FR-005).

### InitResult

The domain value `component.go`'s `Init` returns to `cmd/arc/ctrl`, rendered by the `bios.Registry[InitResult]`.

| Field | Type | Notes |
|---|---|---|
| `Root` | `GraphRoot` | Where the graph was created |
| `CommitHash` | `string` | The short or full hash of the single initial commit |
| `FoldersCreated` | `[]string` | Relative paths, for human/JSON output |

## Filesystem: resolve, then mount (`internal/adapter/fsys`, research.md D3)

Unlike `VCS` below, these interfaces/functions are declared in the shared `internal/adapter/fsys` package, not a `ctrl`-private `port` package — filesystem mounting is a cross-use-case concern by explicit design, reused by every future VISION.md command that operates on a graph root, not just `init`. Built exclusively on stdlib `io/fs`/`io.Reader`/`io.Writer` — no third-party filesystem library (constitution v1.5.0). See research.md D3 for the full listing and the rationale for this one deliberate exception to ADR 001's port-isolation rule 1.

Two separate steps, and `os`'s file/directory functions confined to this package alone:

1. **`fsys.ResolveLocalRoot(root string) (created bool, err error)`** — ensures `root` exists as a local directory (creating it if missing) before anything is mounted; also owns the FR-010 "not a directory" check.
2. **`fsys.Mounter.Mount(root string) (Store, error)`** — called after step 1 has already guaranteed `root` is valid; wraps `os.DirFS(root)` for reads and adds `Create`/`Remove` for writes. `internal/app/ctrl/service.Init` takes an `fsys.Mounter` as a constructor parameter (injected from `cmd/arc/ctrl/init.go`, always `fsys.Local{}` — `arc init` is local-only, see research.md D3).

## Ports (`internal/app/ctrl/port`)

### VCS

Narrow, capability-scoped per Interface Segregation (constitution Principle V, VII) — exactly the three operations `Init` needs, nothing mirroring the full `git` CLI surface.

```go
type VCS interface {
    IsAvailable(ctx context.Context) error
    Init(ctx context.Context, dir string) error
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

- `IsAvailable` — checks `git --version` succeeds; returns a human-readable error if `git` is missing (spec FR-011)
- `Init` — `git init` in `dir`
- `StageAll` — `git add -A` in `dir`
- `Commit` — `git commit -F <tmpfile>` in `dir` with the exact CORE §11.3 message; returns the resulting commit hash (`git rev-parse HEAD`)

No vendor/subprocess types (`exec.Cmd`, `exec.ExitError`) appear in this signature — only `context.Context`, `string`, and `error` (constitution Principle VII: "vendor SDK types MUST NOT leak through port interfaces").

## Reporter events emitted during `Init` (`internal/bios.Reporter`, ADR 002 DS-06)

| Label | Emitted around |
|---|---|
| `"Resolving graph root"` | `fsys.ResolveLocalRoot` + `Mounter.Mount` together, as one reported step (research.md D3) |
| `"Checking git availability"` | `VCS.IsAvailable` |
| `"Creating graph layout"` | Canonical folder/file creation via `Store.Create` (research.md D3) |
| `"Initializing git repository"` | `VCS.Init` |
| `"Staging graph files"` | `VCS.StageAll` |
| `"Committing initial graph"` | `VCS.Commit` |

Each label gets one `Start`/`Done` (or `Error`) pair; no sub-step granularity needed at this scale (ADR 002 DS-08: flat `Reporter` is sufficient for single-phase-per-call commands).

## State transitions

`Init` is a single forward-only transition with no intermediate persisted state: `absent-or-empty directory` → (guards pass) → `fully-initialized graph, one commit`. Failure at any point during the transition rolls back to `absent-or-empty directory` (research.md D4) — there is no partially-initialized state a later command could observe.

## Error sentinels (`github.com/fogfish/faults`, research.md D7)

Declared once as package-level constants; wrapped via `.With()` at the failure site, matched via `errors.Is()`.

### `internal/adapter/fsys` (`errors.go`)

| Constant | Kind | Message | Produced by |
|---|---|---|---|
| `ErrRootNotDirectory` | `faults.Safe1[string]` | `"%s is not a directory"` | `ResolveLocalRoot` (FR-010) |
| `ErrRootCreate` | `faults.Safe1[string]` | `"failed to create graph root at %s"` | `ResolveLocalRoot` (`os.MkdirAll` failure) |
| `ErrCreate` | `faults.Safe1[string]` | `"failed to create %s"` | `Store.Create` (`os.MkdirAll`/`os.Create` failure) |
| `ErrRemove` | `faults.Safe1[string]` | `"failed to remove %s"` | `Store.Remove` (`os.Remove` failure) |

`ResolveLocalRoot` owns the FR-010 "target is a file, not a directory" check directly (it is the first thing to inspect the raw path before any `Store` exists), so this is where `ErrRootNotDirectory` is produced and returned — `service.Init` does not re-declare a duplicate sentinel for the same condition, it just propagates what `ResolveLocalRoot` returned.

### `internal/app/ctrl/service` (`errors.go`)

| Constant | Kind | Message |
|---|---|---|
| `ErrGitUnavailable` | `faults.Type` | `"git is required but was not found on PATH"` |
| `ErrAlreadyInitialized` | `faults.Safe1[string]` | `"%s is already an initialized graph"` |
| `ErrTargetNotEmpty` | `faults.Safe1[string]` | `"%s is not empty; arc init requires an empty or non-existent directory"` |
| `ErrLayoutWrite` | `faults.Safe1[string]` | `"failed to write graph layout at %s"` |

### `internal/app/ctrl/adapter/git` (`git.go`)

| Constant | Kind | Message |
|---|---|---|
| `ErrGitNotFound` | `faults.Type` | `"git binary not found on PATH"` |
| `ErrGitInit` | `faults.Type` | `"git init failed"` |
| `ErrGitStage` | `faults.Type` | `"git add failed"` |
| `ErrGitCommit` | `faults.Type` | `"git commit failed"` |

`service.ErrGitUnavailable` is what `service.Init` returns to the caller when the guard fails; internally it is produced by wrapping whatever `adapter/git` returned (typically `git.ErrGitNotFound`) — i.e. `service.ErrGitUnavailable.With(gitErr)`, so both the domain-level guard reason and the underlying adapter-level cause remain inspectable via `errors.Is` at their respective layers.

## Validation rules (from spec Functional Requirements)

| Rule | Source | Enforced in |
|---|---|---|
| Target path exists as non-directory → refuse, no writes | FR-010 | `fsys.ResolveLocalRoot`, before any `Store` exists |
| `git` unavailable → refuse, no writes | FR-011 | `service.Init` guard (`VCS.IsAvailable`), before any write |
| Target directory already contains `.arc` → refuse, no writes | FR-014 | `service.Init` guard (`Store.Stat(".arc")`), before any write |
| Target directory exists, non-empty, no `.arc/` → refuse, no writes | FR-015 | `service.Init` guard (`Store.ReadDir(".")`), before any write |
| Any failure after guards pass → clean up fully (no partial state) | FR-013 | `service.Init`, via `fsys.RemoveLocalRoot` (if `ResolveLocalRoot` created the root) or per-path `Store.Remove` against the known `ArcNetCoreLayout` list (research.md D4) |
