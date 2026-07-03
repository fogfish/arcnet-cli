# graph

The `graph` use-case is the graph-mutation / graph I/O domain: `Apply`
ingests a CORE §12 document patch into an already-initialized graph. Future
VISION.md commands (`retract`, `reapply`, batch apply) will live in this
same package.

- `kernel/` — `ApplyResult`, the domain value returned to `cmd/arc/graph`.
- `port/` — `VCS`, a `graph`-private secondary port narrower than
  `internal/app/ctrl/port.VCS` (`apply` never bootstraps a repository).
- `adapter/mock/` — an in-memory fake `VCS` for service unit tests.
- `service/` — `Apply`'s use-case logic: guards, per-node create/merge via
  `internal/core.Merge`, timeline update, commit.
- `component.go` — the primary port: `Apply`.

The one promoted, shared `internal/adapter/git.Git` type satisfies this
package's `port.VCS` structurally, exactly as it satisfies
`internal/app/ctrl/port.VCS` (ADR 001 port isolation rule 1,
research.md D4 in `specs/003-apply-patch/`).
