# VCS Port Contract: `internal/adapter/git` (promoted, research.md D4)

`internal/adapter/git.Git` is the one concrete implementation satisfying two separate, narrow port interfaces (ADR 001 port isolation rule 1) — no vendor/subprocess types (`exec.Cmd`, `exec.ExitError`) leak through either.

## `internal/app/ctrl/port.VCS` (unchanged)

```go
type VCS interface {
    IsAvailable(ctx context.Context) error
    Init(ctx context.Context, dir string) error
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

## `internal/app/graph/port.VCS` (new)

```go
type VCS interface {
    IsTracked(ctx context.Context, dir, path string) (bool, error)
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

- `IsTracked` — `git ls-files --error-unmatch <path>` in `dir` (CORE §11.2's documented idempotency check). Exit `0` → `(true, nil)`. Exit `1` (git's own "not tracked" status for `--error-unmatch`) → `(false, nil)` — this is an expected outcome, not an error. Any other failure (git missing, not a repository) → `(false, err)`.
- `StageAll` — `git add -A` in `dir`.
- `Commit` — `git commit -F <tmpfile>` in `dir` with the CORE §11.3 message (research.md D9's Reporter step `"Committing"`); returns the short hash (`git rev-parse --short HEAD`), matching `arc init`'s established convention.

## Migration note

`internal/app/ctrl/adapter/git` is deleted; `cmd/arc/ctrl/init.go` and the new `cmd/arc/graph/apply.go` both import `internal/adapter/git`. `internal/app/ctrl/adapter/mock` (the existing fake `VCS` for `ctrl`'s unit tests) stays put — it is `ctrl`-private and unaffected by the promotion. `internal/app/graph` gets its own fake (`internal/app/graph/adapter/mock`) satisfying `graph.port.VCS`, following the same pattern.
