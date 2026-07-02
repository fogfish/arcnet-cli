# ctrl

The `ctrl` use-case (graph management / control plane) owns the lifecycle
operations that create and validate a knowledge graph's local state,
starting with `Init` — bootstrapping a brand new, empty graph.

## Layout (ADR 001 `componentX` convention)

- `kernel/` — domain value types: `GraphRoot`, `ArcNetCoreLayout`, `InitResult`
- `port/` — the `VCS` secondary port, private to this use-case
- `adapter/git/` — the real `os/exec`-backed `VCS` implementation
- `adapter/mock/` — an in-memory fake `VCS` for service unit tests
- `service/` — the `Init` use-case logic: guards, layout creation, git orchestration, rollback
- `component.go` — the primary port `Init(ctx, mounter, vcs, dir)`, called by `cmd/arc/ctrl`

## Filesystem access

Unlike `VCS`, filesystem mounting is not a `ctrl`-private port: `service.Init`
depends directly on the shared `internal/adapter/fsys.Mounter`/`fsys.Store`,
since every future graph-root-mounting use-case reuses the same mount
contract (see `specs/002-arc-init/research.md` D3).
