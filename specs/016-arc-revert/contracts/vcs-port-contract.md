# Contract: `internal/app/graph/port.VCS` additions

`arc revert` widens the graph domain's own narrow `port.VCS` (`internal/app/graph/port/vcs.go`) — it does not introduce a new port. The concrete `internal/adapter/git.VCS` type (already shared across the `ctrl`/`graph`/`lint` ports per its own package doc, ADR 001 port isolation rule 1) gains the implementations; no second git client is introduced (constitution Principle VII).

## New methods

```go
type VCS interface {
	// existing, unchanged
	IsTracked(ctx context.Context, dir, path string) (bool, error)
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)

	// new (research.md D1/D3/D4/D7/D8)
	CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
	ChangedPaths(ctx context.Context, dir, hash string) ([]string, error)
	CommitsTouching(ctx context.Context, dir, path string) ([]string, error)
	RevertCommit(ctx context.Context, dir, hash string) (newHash string, err error)
	Blame(ctx context.Context, dir, path string) ([]BlameLine, error)
	ShowFile(ctx context.Context, dir, hash, path string) ([]byte, error)
}
```

## Git command mapping

| Method | Git command | Notes |
|---|---|---|
| `CommitsMatching(dir, needle)` | `git log --all --fixed-strings --grep=<needle> --format=%H` | Already implemented (`internal/adapter/git/git.go:154`), reused verbatim — only the `graph` port interface is new, not the adapter body. |
| `ChangedPaths(dir, hash)` | `git diff-tree --no-commit-id --name-only -r <hash>` | Root-commit-safe: `git diff-tree` diffs a root commit against the empty tree automatically, unlike a plain two-dot `git diff`. |
| `CommitsTouching(dir, path)` | `git log --follow --format=%H -- <path>` | Newest-first, matching `CommitsMatching`'s own ordering convention. `--follow` matters for a node whose file was ever renamed (out of this feature's normal flow, but git does not distinguish). |
| `RevertCommit(dir, hash)` | `git revert --no-edit <hash>` then `git rev-parse --short HEAD` | Two subprocess calls, mirroring `Commit`'s own existing `commit` + `rev-parse --short HEAD` pattern (`internal/adapter/git/git.go:104-119`). A non-zero exit (merge conflict during the revert — should not occur given D3's eligibility precondition, but git does not guarantee it structurally) surfaces as `ErrGitRevert`. |
| `Blame(dir, path)` | `git blame --line-porcelain HEAD -- <path>` | Parsed line-by-line: a `--line-porcelain` block starts with `<sha1> <orig-line> <final-line> [<num-lines>]`; only the leading commit hash and the running final-line counter are kept, into `[]BlameLine{Number, Commit}`. |
| `ShowFile(dir, hash, path)` | `git show <hash>:<path>` | Returns the raw bytes of `path` as it existed at `hash`; a `path` that did not yet exist at `hash` is a normal, expected non-error case the caller (D8b) must handle by treating that commit as not having set the predicate yet, not by treating the git failure as fatal — implemented by checking `git show`'s specific "does not exist" exit condition the same way `IsTracked` already distinguishes an expected "not tracked" exit from a genuine failure (`internal/adapter/git/git.go:128-144`). |

## New error sentinels (`internal/adapter/git/git.go`)

```go
const (
	ErrGitDiffTree = faults.Type("git diff-tree failed")
	ErrGitRevert   = faults.Type("git revert failed")
	ErrGitBlame    = faults.Type("git blame failed")
	ErrGitShow     = faults.Type("git show failed")
)
```

Mirrors the existing `ErrGitLog`/`ErrGitLsFiles`/`ErrGitCommit` pattern (`internal/adapter/git/git.go:29-34`) exactly — one `faults.Type` per subprocess failure mode, wrapped via `.With(err)` at the call site, never an ad hoc `fmt.Errorf` (constitution Mandatory Libraries & Tooling, Error Handling & Annotation).

## Behavior contract: what a caller can rely on

1. **`CommitsTouching` ordering is the single source of truth for both eligibility questions** (research.md D3/D5): whole-operation eligibility and per-node exclusivity are both expressed purely in terms of this one method's output — no second "is this HEAD" or "is this the first commit" primitive exists or should be added.
2. **`ChangedPaths`/`CommitsTouching`/`Blame` never mutate the working tree or the repository** — read-only `git` subcommands only. Only `RevertCommit` and the existing `StageAll`/`Commit` (per-node path) touch working-tree/history state, and each produces at most one commit (FR-015).
3. **`ShowFile`'s "path absent at this commit" case is not an error** — see the table above; a caller that treats it as fatal breaks D8(b)'s historical walk on the (normal) case of a predicate that was empty before some later commit first set it.
4. **Test doubles**: `internal/app/graph/adapter/mock.VCS` (research.md D11) exposes one configurable return/error field per new method plus the existing `Calls` log, so `internal/app/graph/service/revert_test.go` can assert exactly which git primitives a given scenario invoked, the same way existing `apply_test.go` cases already assert on `Calls`.
