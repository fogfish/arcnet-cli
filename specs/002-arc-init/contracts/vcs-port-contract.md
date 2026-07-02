# Port Contract: `internal/app/ctrl/port.VCS`

Private to the `ctrl` use-case (ADR 001 port isolation rule 1) — no other domain package imports this interface.

```go
package port

import "context"

type VCS interface {
    IsAvailable(ctx context.Context) error
    Init(ctx context.Context, dir string) error
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

## Method contracts

All errors returned across this interface are wrapped with a `github.com/fogfish/faults` constant declared in `internal/app/ctrl/adapter/git` (see data-model.md "Error sentinels"), never a raw `exec.ErrNotFound`/`*exec.ExitError` or an ad hoc `fmt.Errorf` string (constitution Principle XII, Mandatory Libraries & Tooling). Callers match specific failures with `errors.Is(err, git.ErrGitInit)` etc.

### `IsAvailable(ctx) error`

- Returns `nil` if the `git` binary is resolvable and runnable (equivalent to `git --version` succeeding).
- Returns `git.ErrGitNotFound.With(execErr)` otherwise — never a raw `exec.ErrNotFound` or `*exec.ExitError`.

### `Init(ctx, dir) error`

- Runs the equivalent of `git init` with working directory `dir`.
- `dir` MUST already exist and be writable; this method does not create it.
- Idempotent at the git level (matches real `git init` behavior), but the calling service never invokes it on an existing graph — the FR-014 guard runs first.
- On failure, returns `git.ErrGitInit.With(execErr)`.

### `StageAll(ctx, dir) error`

- Runs the equivalent of `git add -A` with working directory `dir`.
- Stages every file created by the layout step, and nothing else (the directory is guaranteed to contain only files this command just wrote).
- On failure, returns `git.ErrGitStage.With(execErr)`.

### `Commit(ctx, dir, message) (hash string, err error)`

- Runs the equivalent of `git commit -F <tmpfile-containing-message>` with working directory `dir`.
- `message` is the exact CORE §11.3-shaped subject the caller assembled: subject line `graph(init): empty knowledge graph`, `Nodes:`/`Source-Id:`-equivalent trailers are not applicable to an empty-graph commit (no source ingested yet) — the init commit message is exactly the mandatory subject line, per spec FR-007.
- On success, returns the resulting commit's **short** hash (equivalent to `git rev-parse --short HEAD` immediately after committing, not the full 40-character SHA — research.md D2 Bugfix, BUG-001).
- On failure, returns `("", git.ErrGitCommit.With(execErr))`.

## Implementations

### `internal/app/ctrl/adapter/git` (real, production)

- Backs every method with `os/exec.CommandContext(ctx, "git", ...)`, working directory set via `exec.Cmd.Dir`.
- Wraps each call with the `bios.Reporter` Start/Done/Error sequence (data-model.md "Reporter events") — the adapter, not the service, owns the exact git subprocess invocation, so it is also the natural place to own progress reporting around that invocation.
- Captures combined stdout+stderr from the subprocess only for inclusion in the wrapped error message on failure; discards it on success (git's own success output is not meaningful to the end user beyond "it worked," which the Reporter's `Done` already conveys).
- Never logs or echoes anything through this adapter that could contain credentials (constitution Principle VII) — not applicable here since no remote/auth operations are performed by `init`.

### `internal/app/ctrl/adapter/mock` (test double)

- In-memory fake implementing `port.VCS` with configurable return values/errors per method and a call log, for `internal/app/ctrl/service/init_test.go` unit tests (constitution Principle VI: unit tests run with no live git process).

**Bugfix**: 2026-07-02 — BUG-001: `Commit` now returns a short hash (`--short`), not the full SHA. Reporter Start/Done/Error progress is `--verbose`-gated and styled faint/gray, not shown by default in `SCHEMA.StatusOK` green (see `cli-contract.md` and research.md D2 Bugfix).
