# Implementation Plan: Type Inheritance via `rdfs:subClassOf`

**Branch**: `017-subclass-of-predicate` | **Date**: 2026-07-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/017-subclass-of-predicate/spec.md`

## Summary

Types (`_schema/types/<name>.md` Class nodes) gain a new `rdfs:subClassOf` edge to declare one or more base types. `internal/app/schema/service.Resolve` and `Seed` recursively flatten each type's inherited `## Requires`/`## Optional` predicates into its `core.TypeDef.Required`/`.Optional` at schema-indexing time, so every existing consumer of `core.Index.Types` (`arc lint`'s conformance checks foremost) sees an already-complete, inherited contract without any code change of its own. A new built-in type, `Node`, becomes the implicit universal base of the four content types (`source`, `entity`, `resource`, `timeline`) — every content node's contract now always includes Node's `published`/`created` (required) and `tags`/`text`/`updated`/`scoreZ`/`scoreC` (optional), whether or not a type explicitly declares the relationship. `Property`/`Class` (the schema meta-types) are excluded from this implicit base. Cycles and unresolved base-type references fail schema loading with a new, clearly-named error, consistent with existing `ErrSchemaInvalid`/`ErrSchemaMissing` handling. The change is confined to `internal/app/schema` (`kernel` + `service`); `internal/core` (node/edge model, Markdown rendering/parsing) and `internal/app/lint` are untouched in behavior, since `rdfs:subClassOf` is carried entirely as an ordinary registered edge predicate the existing generic Edges/Link machinery already supports.

## Technical Context

**Language/Version**: Go 1.26.5 (match `go.mod`)

**Primary Dependencies**: `github.com/spf13/cobra` (unaffected — no CLI surface change), `github.com/fogfish/faults` (two new schema-load error types, matching `ErrSchemaInvalid`'s existing pattern)

**Storage**: Local graph files under `_schema/types/` and `_schema/predicates/`, accessed via the existing `internal/adapter/fsys` port (no new storage mechanism)

**Testing**: `go test ./...` with `github.com/fogfish/it/v2`; new unit tests in `internal/app/schema/kernel` and `internal/app/schema/service` (multi-level chains, multiple bases, diamond inheritance, cycle detection, unresolved-reference detection); existing `internal/app/lint` E2E fixtures/tests updated where the reshaped `CoreTypeDefs` (Node factored out) changes what a seeded graph's lint output looks like

**Target Platform**: linux/darwin/windows amd64+arm64 (per `.goreleaser.yaml`) — unaffected, no platform-specific code

**Project Type**: Single Cobra CLI binary — this feature adds no new command or flag; it changes `internal/app/schema`'s data and resolution logic only

**Performance Goals**: Schema indexing (`arc init`, and the schema load every schema-consuming command performs) must remain effectively instant for realistic graphs — SC-003 requires correct resolution at ≥4 inheritance levels and ≥3 declared bases per type; the resolution algorithm (memoized recursive flattening, one pass over the type set) is linear in total (type × inherited-predicate) pairs, no measurable regression expected

**Constraints**: Zero behavior change to `internal/core` (node parsing/rendering) or `internal/app/lint`'s own code — `arc lint`'s existing Requires/Optional checks (`internal/app/lint/service/rules_type_conformance.go`) must gain inheritance-awareness purely as a side effect of `core.Index.Types` already carrying flattened contracts, not through any change to that file

**Scale/Scope**: Two files touched primarily (`internal/app/schema/kernel/schema.go`, `internal/app/schema/service/schema.go`) plus their tests; no new package, no new `cmd/` command

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle III (Hexagonal Architecture)**: PASS. All new logic lives in `internal/app/schema/service` (domain/use-case layer) and `internal/app/schema/kernel` (value types). No `cmd/` package is touched — there is no new flag or command. No Cobra import enters domain code.
- **Principle IV/V (Functional style, SOLID/YAGNI)**: PASS. Resolution is a pure function over the decoded type map (name → raw required/optional/subClassOf); no new abstraction beyond what multi-pass recursive flattening requires. No speculative generality — the resolver only ever needs to answer "what is this type's effective contract," nothing more.
- **Principle VI (Assertion library)**: PASS. New unit tests use `github.com/fogfish/it/v2`, matching `internal/app/schema/kernel/schema_test.go` and `internal/app/schema/service/schema_test.go`'s existing convention.
- **Principle VII (External Integration & Adapter Consistency)**: PASS. No new external system integration; the existing `fsys.Store` port is reused unchanged for reading `_schema/types/*.md`.
- **Principle VIII (E2E Acceptance Testing)**: Applies indirectly — this feature has no new `cmd/` command of its own, so no new `cmd/<cmd>_test.go` is added for *this* feature directly. However, `arc lint`'s existing E2E tests (`cmd/lint/lint_test.go` or equivalent) exercise the reshaped `CoreTypeDefs` output end-to-end and MUST be re-verified/updated as part of implementation, since Node's factoring-out changes what a freshly-`arc init`'d graph's lint baseline looks like.
- **Principle I (ADRs)**: No accepted ADR in `adrs/` currently documents the predicate/type schema mechanism itself (ADR 001 covers general system architecture, not this domain's file format specifically) — no conflict found. If this feature's design (the `rdfs:subClassOf` edge convention, the Node universal-base convention) should become a durable, binding pattern, a follow-up ADR is recommended but not required to ship this feature.

No violations requiring Complexity Tracking.

**Post-Phase-1 re-check**: research.md/data-model.md/contracts confirm the design stays entirely within `internal/app/schema` (`kernel` + `service`) — no `core.TypeDef` shape change, no `internal/core` change, no `internal/app/lint` code change, no new `cmd/` command. All gates above still PASS unchanged.

## Project Structure

### Documentation (this feature)

```text
specs/017-subclass-of-predicate/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
└── app/
    └── schema/
        ├── kernel/
        │   ├── schema.go        # + "rdfs:subClassOf" predicate def, "Node" type def,
        │   │                     #   updated source/entity/resource/timeline defs
        │   └── schema_test.go   # + coverage for the reshaped seed data
        └── service/
            ├── schema.go        # + rdfs:subClassOf edge decode/encode, recursive
            │                     #   flattening pass in resolveTypes/Seed, cycle and
            │                     #   unresolved-base-type detection
            ├── errors.go        # + ErrSchemaCycle, ErrSchemaUnresolvedBase
            └── schema_test.go   # + multi-level/multi-base/diamond/cycle/unresolved cases

internal/app/lint/
└── service/
    └── (no code change — existing Requires/Optional checks consume the already-
        flattened core.Index.Types transparently)

testdata/                 # fixture graphs used by lint's E2E tests, updated if they
                           # assert against the old (pre-Node) CoreTypeDefs shape
```

**Structure Decision**: This feature touches only the `internal/app/schema` domain package (its two subpackages `kernel` and `service`) plus test fixtures in `internal/app/lint` that assert against seeded schema content. No new command, no new domain package, no adapter changes.

## Complexity Tracking

*No Constitution Check violations — table intentionally omitted.*
