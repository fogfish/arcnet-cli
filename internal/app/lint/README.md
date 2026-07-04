# internal/app/lint

The graph conformance validation domain use-case (fourth `internal/app/<domain>`
use-case, after `ctrl`, `config`, `graph`), following ADR 001's `componentX`
layout:

- `kernel/` — `Rule`, `Violation`, `NodeStatus`, `LintResult`, and the CORE
  §9.2.1 Sowa category decode tables.
- `port/` — `VCS`, the narrowest of this codebase's three `port.VCS`
  interfaces (`CommitsMatching` only); lint never initializes, stages, or
  commits.
- `adapter/mock/` — an in-memory fake `VCS` for `service` unit tests.
- `service/` — the `Lint` use-case: enumerates every node file, runs every
  CORE §14 checklist rule, and aggregates every violation found, never
  stopping at the first one.
- `component.go` — the primary port, `Lint(ctx, mounter, vcs, reporter,
  rules, dir) (kernel.LintResult, error)`.

Lint is strictly read-only: it never calls `fsys.Store.Create`/`Remove`, and
its one external dependency (`port.VCS.CommitsMatching`) only reads git
history, never writes it. See `specs/004-arc-lint/` for the full
spec/plan/research/data-model.
