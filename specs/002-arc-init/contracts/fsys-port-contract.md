# Adapter Contract: `internal/adapter/fsys`

Shared, cross-use-case adapter (ADR 001's application-level adapter tier) — not private to `ctrl`. Built exclusively on stdlib `io/fs`, `io.Reader`, and `io.Writer` (constitution v1.5.0, Mandatory Libraries & Tooling: "Filesystem Abstraction") — no third-party filesystem or object-storage library.

**`internal/adapter/fsys` is the only package in the entire codebase permitted to call `os`'s file/directory functions** (`os.Open`, `os.Create`, `os.Stat`, `os.ReadDir`, `os.MkdirAll`, `os.Remove`, `os.RemoveAll`). Every other package — including `internal/app/ctrl/service` — depends only on the `io/fs`-shaped `Store`/`Mounter` interfaces below.

Two concerns, deliberately kept apart (research.md D3): **resolving** a root location and **mounting and performing I/O against** an already-resolved root.

```go
package fsys

import (
    "io"
    "io/fs"
)

// Resolution — the only functions in this package that touch a location
// before it is known to be a valid, existing directory.

func ResolveLocalRoot(root string) (created bool, err error)
func RemoveLocalRoot(root string) error

// Mounting and I/O — expressed entirely in stdlib io/fs / io.Writer terms.

// File is the writable counterpart to fs.File (io/fs itself is read-only:
// fs.FS only defines Open). Discard is borrowed from the same idea as
// stream.Canceler: undo an in-flight write instead of leaving a
// half-written file behind.
type File interface {
    io.Writer
    io.Closer
    Stat() (fs.FileInfo, error)
    Discard() error
}

type Store interface {
    fs.FS
    fs.StatFS
    fs.ReadDirFS
    Create(name string) (File, error)
    Remove(name string) error
}

type Mounter interface {
    Mount(root string) (Store, error)
}
```

`Store` is the narrow subset of `fs.FS`/`fs.StatFS`/`fs.ReadDirFS` that `ctrl.Init` (and, going forward, every other graph-root-mounting use-case) actually calls, plus the `Create`/`Remove` write-side extension `io/fs` itself omits by design — not a copy of every possible filesystem operation (`Glob`, symlink handling, permissions management are all omitted; a future use-case that needs them declares its own wider interface, still structurally satisfied by the same concrete `Local` type).

## Path convention (stdlib `io/fs`, `fs.ValidPath`)

- Every path passed to `Store` methods is relative, slash-separated, and MUST NOT start or end with `/`. `"."` denotes the root itself.
- There is no `Mkdir`/`MkdirAll` operation on `Store` — a directory becomes observable (via `ReadDir`/`Stat` on its clean relative name) only once at least one file exists under it. This is why every `ArcNetCoreLayout.Folders` entry gets a `.gitkeep` file (data-model.md, spec FR-005): git itself does not track empty directories, so a placeholder file is required regardless of what mounts the filesystem underneath — this is not a `Store` limitation, it mirrors a real git limitation.

## Function/method contracts

All errors are wrapped with a `github.com/fogfish/faults` constant declared in `internal/adapter/fsys/errors.go` (data-model.md "Error sentinels"), consistent with the project-wide error-annotation mandate.

### `ResolveLocalRoot(root) (created bool, err error)`

- If `root` does not exist: creates it (`os.MkdirAll(root, 0o755)`) and returns `created=true`. On a create failure, returns `(false, ErrRootCreate.With(err))`.
- If `root` exists and is not a directory: returns `(false, ErrRootNotDirectory.With(...))` — spec FR-010.
- If `root` exists and is a directory: returns `(false, nil)`.

### `RemoveLocalRoot(root) error`

- `os.RemoveAll(root)` — undoes exactly what a prior `ResolveLocalRoot` call's `created=true` created.
- MUST only ever be called by `service.Init` when the immediately-preceding `ResolveLocalRoot` call for the same `root` returned `created=true` — calling it otherwise would delete a pre-existing directory this run does not own.

### `Mounter.Mount(root) (Store, error)`

- Mounts an **already-resolved** `root`. `Local.Mount(root)` wraps `os.DirFS(root)` for the read side (`Open`/`Stat`/`ReadDir`/`Glob` — `os.DirFS`'s returned value already satisfies `fs.StatFS`/`fs.ReadDirFS`/`fs.GlobFS` directly in the Go version this project pins) and stores `root` itself for the `Create`/`Remove` write-side methods, which join a given relative `name` against `root` and call `os.MkdirAll`/`os.Create`/`os.Remove` directly.
- Requires `ResolveLocalRoot(root)` to have already succeeded — performs no existence checks or creation of its own.

### `Store.Stat(name) (fs.FileInfo, error)` / `Store.ReadDir(name) ([]fs.DirEntry, error)`

- Standard `io/fs` semantics, delegated to `os.DirFS(root)`. Not-found is `fs.ErrNotExist`; guard code checks `errors.Is(err, fs.ErrNotExist)` to distinguish "absent" from a real I/O failure.

### `Store.Create(name) (File, error)`

- Joins `name` against the mounted root, `os.MkdirAll`s the parent directory, then `os.Create`s the file. Returns `(nil, ErrCreate.With(err))` on failure.
- The returned `File` wraps the resulting `*os.File`, which already satisfies `Write`/`Close`/`Stat` natively; the wrapper adds only `Discard()`.
- `service.Init` MUST call `Close()` on a successful write and `Discard()` (not `Close()`) if a write fails partway, to avoid leaving a corrupt or partially-written file behind. `Discard()` closes the underlying `*os.File` then `os.Remove`s it.

### `Store.Remove(name) error`

- Joins `name` against the mounted root and calls `os.Remove`. Returns `ErrRemove.With(err)` on failure. Used both by `service.Init`'s FR-013 rollback (removing specific `ArcNetCoreLayout` paths written by a prior `Store.Create`+`Close`) and, in later features, by any use-case that needs to delete a single graph node file.

## Implementation

### `fsys.Local` (real, production; wraps `os.DirFS` + `os.Create`/`os.MkdirAll`/`os.Remove`)

- The only concrete `Mounter`/`Store` implementation. No S3 or other remote backend exists — see research.md D3, Correction 3, for why that was dropped rather than deferred as a stub.

### Test doubles

- `internal/app/ctrl/service` unit tests (constitution Principle VI) use an in-memory fake satisfying `fsys.Store`/`fsys.Mounter`, plus a fake/stubbed `ResolveLocalRoot`/`RemoveLocalRoot` pair — no real disk access needed to test guard logic (FR-010/011/014/015) or rollback (FR-013).
- `internal/adapter/fsys`'s own tests for `ResolveLocalRoot`/`RemoveLocalRoot`/`Local` exercise the real local filesystem against `t.TempDir()` (constitution Principle VI: real file I/O against a temp directory is sanctioned).
