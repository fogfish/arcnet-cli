# Port Contracts: `internal/app/schema/port.VCS` and `port.Fetcher`

## `port.VCS`

```go
type VCS interface {
    StageAll(ctx context.Context, dir string) error
    Commit(ctx context.Context, dir, message string) (hash string, err error)
}
```

- **Precondition**: `dir` is already a git working tree (`arc init` already ran) — this port never checks or establishes that; unlike `internal/app/ctrl/port.VCS`, there is no `IsAvailable`/`Init`.
- **`StageAll`**: stages every change under `dir` (mirrors `graph.port.VCS.StageAll`'s contract exactly — same method signature, same semantics, reused verbatim by the shared `internal/adapter/git.VCS` concrete implementation).
- **`Commit`**: commits currently staged changes with `message` as the commit subject/body, returns the resulting **short** hash (matching `arc init`/`arc apply`'s established convention). Calling `Commit` when nothing is staged is never done by this feature's caller (research.md D7 — the caller checks for a no-op before staging/committing at all).
- **Test double**: `internal/app/schema/adapter/mock.VCS` — configurable `StageAllErr`, `CommitHash`/`CommitErr`, and a `Calls []string` log, mirroring `internal/app/ctrl/adapter/mock.VCS`'s existing shape exactly.

## `port.Fetcher`

```go
type Fetcher interface {
    Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}
```

- **Precondition**: `url` has scheme `http` or `https` (the caller only invokes `Fetch` after `research.md D1`'s `url.Parse` classification).
- **Postcondition on success**: returns a readable, caller-closed body containing the fetched patch document's raw bytes; the concrete adapter has already verified a 2xx status before returning (a non-2xx response is `ErrFetchStatus`, not a successful `Fetch` call).
- **Postcondition on failure**: returns a nil reader and a non-nil error — `ErrFetch` (network/timeout) or `ErrFetchStatus` (non-2xx), both from `internal/adapter/http`, both `faults`-typed per Principle XII.
- **Context**: `ctx` carries the effective deadline (`--timeout`, default 30s, research.md D2); the adapter MUST use `http.NewRequestWithContext` so cancellation/timeout is honored mid-fetch, not just at connection time.
- **Test double**: `internal/app/schema/adapter/mock.Fetcher` — configurable `Body []byte`/`Err error` and a `Calls []string` log (records the requested URL), returning `io.NopCloser(bytes.NewReader(Body))` on success.

## Concrete adapter: `internal/adapter/http.Client`

```go
type Client struct {
    HTTPClient *http.Client
}

func New(timeout time.Duration) Client
func (c Client) Fetch(ctx context.Context, url string) (io.ReadCloser, error)
```

- `New` constructs an `http.Client` with `Timeout: timeout` (defaulting to 30s when the CLI's `--timeout` flag is unset).
- `Fetch` issues a single `GET`, no redir:follow limit beyond `net/http`'s own default (10), no request body, no custom headers beyond what `net/http` sets by default.
- Never logs the URL's query string or any response header at any verbosity level beyond what `bios.Reporter`'s existing progress labels already show (no secret material expected in a public schema-patch URL, but Principle VII's "never log secret material" discipline is followed defensively regardless).
