# schema

The graph schema domain use-case: it isolates ARCNET-CORE's declared
vocabulary of predicates and types as machine-readable
`_schema/predicates/*.md` (`Property` nodes) and `_schema/types/*.md`
(`Class` nodes) — one versioned, human-readable document per predicate and
type, each declaring its own role/merge/label/aligned (predicates) or
merge/required/optional/subClassOf (types), per CORE §9.1/§9.2.

A type may declare zero or more `subClassOf` edges (`Role: "edge"`,
`Aligned: "rdfs:subClassOf"`) naming other registered types it inherits
from — multiple inheritance and transitive chains both resolve correctly.
Every type except the built-in `Node`, `Property`, and `Class` also
implicitly inherits `Node`'s own contract (`published`/`created` required;
`tags`/`text`/`updated`/`scoreZ`/`scoreC` optional), whether or not it
declares the edge explicitly — factoring the cross-cutting contract every
content type previously redeclared directly out into one place (spec 017).
`service.Resolve`/`Seed` flatten this hierarchy once, at schema-indexing
time, into each type's fully inherited `core.TypeDef.Required`/`.Optional`
— no downstream consumer (`arc lint` foremost) has any notion of hierarchy.
A cycle or a reference to an unregistered base type fails schema loading
(`ErrSchemaCycle`/`ErrSchemaUnresolvedBase`) before any other
schema-dependent work proceeds.

## Layout

Unlike every other `internal/app/<domain>` use-case, `schema` has no `port`/
`adapter` subdirectory: it has no use-case-private external dependency
beyond the already-shared `internal/adapter/fsys`, consumed directly by
`service.Resolve`/`RegisterType`/`RegisterPredicate` (mirroring
`internal/app/ctrl/service.Init`'s existing precedent of taking
`fsys.Mounter`/`fsys.Store` as a plain parameter). It also has no `cmd/`
package of its own — it is consumed only by `arc init` and `arc apply`
(and referenced, read-only, by `arc lint`'s wiring), never invoked directly.

- `kernel/` — `CorePredicateDefs`, `CoreTypeDefs` (ARCNET-CORE §10/§11's
  full built-in vocabulary, plus the `Node` universal-base type),
  `CoreTypeBases` (each seeded content type's explicit `subClassOf` base),
  `TypesDir`/`PredicatesDir` path constants.
- `service/` — `Seed`, `Resolve` (returns `core.Index`, fail-fast on a
  missing/malformed document or a broken `subClassOf` hierarchy),
  `RegisterType`, `RegisterPredicate`. The `subClassOf` resolution pass
  (decode into a private `rawType` record, then memoized recursive
  flattening) is internal to this package — `core.TypeDef` never carries
  raw hierarchy, only the resolved result.
- `component.go` — primary port: thin delegators into `service`.

## Consumed by

- `cmd/arc/ctrl/init.go` calls `Seed()` to populate a new graph's
  `_schema/` folder — pure, no network access.
- `cmd/arc/graph/apply.go` calls `Resolve(store)` to get the `core.Index`
  `graph.Apply` needs, and wires this package's component as
  `graph.port.SchemaRegistry` for auto-discovery mid-apply.
- `cmd/arc/lint/lint.go` calls `Resolve(store)` for the same `core.Index`
  `lint.Lint` checks node-declared predicates/types against.
