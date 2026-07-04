# schema

The graph schema domain use-case: it isolates ARCNET-CORE's declared
vocabulary of node kinds, merge behaviors, and predicates as
`_schema/nodes/*.md` and `_schema/predicates/*.md` — one versioned,
human-readable document per node kind and predicate.

## Layout

Unlike every other `internal/app/<domain>` use-case, `schema` has no `port`/
`adapter` subdirectory: it has no use-case-private external dependency
beyond the already-shared `internal/adapter/fsys`, consumed directly by
`service.Resolve`/`RegisterKind`/`RegisterPredicate` (mirroring
`internal/app/ctrl/service.Init`'s existing precedent of taking
`fsys.Mounter`/`fsys.Store` as a plain parameter). It also has no `cmd/`
package of its own — it is consumed only by `arc init` and `arc apply`
(and referenced, read-only, by `arc lint`'s wiring), never invoked directly.

- `kernel/` — `CoreMergeRules`, `CorePredicates` (ARCNET-CORE §9/§7.4's
  built-in vocabulary), `SchemaKind`, `NodesDir`/`PredicatesDir` path
  constants.
- `service/` — `Seed`, `Resolve`, `RegisterKind`, `RegisterPredicate`.
- `component.go` — primary port: thin delegators into `service`.

## Consumed by

- `cmd/arc/ctrl/init.go` calls `Seed()` to populate a new graph's
  `_schema/` folder — pure, no network access.
- `cmd/arc/graph/apply.go` calls `Resolve(store)` to get the merge-rule set
  and predicate set `graph.Apply` needs, and wires this package's component
  as `graph.port.SchemaRegistry` for auto-discovery mid-apply.
- `cmd/arc/lint/lint.go` calls `Resolve(store)` for the predicate set
  `lint.Lint` checks node-declared predicates against.
