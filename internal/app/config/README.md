# config

The `config` use-case owns `.arc/config.yml` — a graph's local, hand-edited
registry of domain-specific node kinds and their merge behaviors, unioned
with the graph format's own fixed kinds (`internal/core.CoreMergeRules`) at
resolve time.

- `kernel/` — `Config`, the on-disk shape.
- `port/` — `Fetcher`, the config-private port for the one-shot config-seed
  HTTP fetch `arc init` performs.
- `adapter/http/` — the real, stdlib `net/http`-backed `Fetcher`.
- `adapter/mock/` — a fake `Fetcher` for `service.Default`'s unit tests.
- `service/` — `Load`/`Save`/`Resolve`/`Default`.
- `component.go` — the primary port: `Resolve`, `Save`, `Default`.

`internal/app/ctrl` never imports this package (ADR 001 use-case
decoupling) — `cmd/arc/ctrl/init.go` is the wiring layer that composes both,
per research.md D5 (revised) in `specs/003-apply-patch/`.
