# Config Contract: `internal/app/config`

**Revised 2026-07-02** (post-plan) — seed content now comes from a live fetch of `github.com/fogfish/arcnet-spec`'s canonical config, with a built-in fallback; see research.md D5 (revised).

## Primary port (`component.go`)

```go
func Resolve(store fsys.Store) (core.MergeRuleSet, error)
func Save(store fsys.Store, cfg kernel.Config) error
func Default(ctx context.Context, fetcher port.Fetcher) (cfg kernel.Config, usedFallback bool)
```

- `Resolve` — reads `.arc/config.yml` (`core.ConfigPath`) via `store.Open`. File absent → returns `core.CoreMergeRules` alone (no error; this is "no domain kinds registered", spec User Story 3 Acceptance Scenario 2). File present but not valid YAML → `ErrConfigMalformed`. File present and valid → `core.CoreMergeRules.Union(loaded)` (the format's three built-in kinds always win if a file somehow tries to redeclare one differently — `Union` is first-writer over the two rule sets being combined, with `CoreMergeRules` as the first/authoritative side). Called by `internal/app/graph/service.Apply` (via `cmd/arc/graph/apply.go`) on every `arc apply` invocation.
- `Default` — the seed-content resolver, called once by `cmd/arc/ctrl/init.go` per `arc init` invocation. Attempts one `fetcher.Fetch(ctx, DefaultSourceURL)`; unmarshals the result into `kernel.Config` on success. **Any failure at all** — network error, non-2xx status, timeout, or a response body that isn't valid YAML — is treated identically: `Default` returns `kernel.Config{MergeRules: core.CoreMergeRules}` with `usedFallback = true`, and **has no `error` return at all**, so a caller cannot mistakenly propagate a fetch failure as an `arc init` failure. `DefaultSourceURL = "https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/config.yml"`.
- `Save` — writes `cfg` back as YAML via `store.Create`. In this feature's scope, `Save`'s only caller is `internal/app/config`'s own unit tests (round-tripping `Resolve`); no CLI command calls `Save` directly (no `arc config` mutation command ships in this iteration). `internal/app/ctrl`, notably, does **not** call `Save` — it marshals a `kernel.Config` value (received from `cmd/arc/ctrl/init.go`'s call to `appconfig.Default`) to bytes itself and writes them via its own layout mechanism, since `internal/app/ctrl` never imports `internal/app/config` (ADR 001 use-case decoupling, research.md D5).

## `port.Fetcher` (new, `config`-private)

```go
type Fetcher interface {
    Fetch(ctx context.Context, url string) ([]byte, error)
}
```

- Real adapter (`internal/app/config/adapter/http`) — stdlib `net/http`, `http.Client{Timeout: 3 * time.Second}`, no retries. Any non-2xx response is treated as `Fetch` returning an error (so `Default`'s fallback triggers), never returned as a "successful" empty/error-page body.
- Mock adapter (`internal/app/config/adapter/mock`) — used only by `internal/app/config`'s own unit tests; `go test ./...` makes no real network call anywhere in this feature (constitution Principle VI).
- **Known compliance gap, flagged not silently resolved**: constitution Principle VII requires network calls to have a timeout "overridable by a flag or config value." No override exists in this iteration — see research.md D5 (revised) for why, and the suggested follow-up (an `ARC_CONFIG_TIMEOUT` environment variable).

## On-disk shape (`.arc/config.yml`)

```yaml
mergeRules:
  source: none
  entity: union
  resource: union-first-writer
```

A user opts a domain kind in by hand-adding a line, e.g. `hypothesis: validated-overwrite` (copied from `core.KnownProfileMergeRules`, or an arbitrary value if defining an entirely new, project-local kind not documented by any known profile — `Resolve` does not restrict `mergeRules` keys/values to `KnownProfileMergeRules`, only to the fixed `MergeOp` vocabulary itself).

## Seeding at `arc init` (touches `internal/app/ctrl`, research.md D5 revised)

`cmd/arc/ctrl/init.go` constructs the real `Fetcher`, calls `appconfig.Default(ctx, fetcher)`, marshals the resulting `kernel.Config` to YAML bytes, and passes those bytes as a new `configSeed []byte` parameter to `appctrl.Init(ctx, mounter, vcs, dir, configSeed)`. `internal/app/ctrl/service.Init` writes `configSeed` to `core.ConfigPath` via a per-call copy of `kernel.DefaultLayout` with that one `MetaStubs` entry added (the package-level `DefaultLayout` itself stays static and config-free, since the actual seed content is no longer a compile-time constant) — `writeLayout` itself is unchanged. `rollback` additionally removes `core.ConfigPath` on a mid-run failure, alongside the existing static paths.

Under `--verbose`, `arc init` reports one more step, `"Fetching default configuration"`, and — only when `usedFallback` is true — a follow-up note that the built-in core-only default was used instead (offline or unreachable). This never affects `arc init`'s exit code or success message (`specs/002-arc-init/spec.md` FR-017: initialization MUST NOT fail on this basis alone).

## Unrecognized-kind fallback at `arc apply` time (research.md D5 revised — supersedes the original "hard refuse" design)

`core.MergeRuleSet.Lookup(kind)` returns `ok=false` when a patch node's kind is absent from the graph's resolved rule set (built-in ∪ `.arc/config.yml`). `internal/app/graph/service.Apply` does **not** refuse the patch in that case: it applies the node using `core.MergeUnion` and appends a warning sentence to `kernel.ApplyResult.Warnings`. See `contracts/cli-contract.md` for the resulting stderr/`--json` presentation.

## Known limitation (research.md D5)

`.arc/` is `.gitignore`d — `.arc/config.yml` is local to one clone, not synced via git. Two collaborators on the same graph repository can have different locally-registered domain kinds; both `arc apply` runs still succeed (one using the registered behavior, the other using the `union` default plus a warning), so this is a lower-severity tension than in the pre-revision design, but still flagged, not silently resolved — see research.md D5 for the full note.
