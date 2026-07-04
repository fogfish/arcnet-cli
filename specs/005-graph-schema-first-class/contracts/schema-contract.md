# Schema Contract: `internal/app/schema`

## Primary port (`component.go`)

```go
func Seed() map[string][]byte
func Resolve(store fsys.Store) (core.MergeRuleSet, map[string]bool, error)
func RegisterKind(store fsys.Store, kind core.Kind) (created bool, err error)
func RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
```

- `Seed()` — pure, no I/O, no `context.Context` (no network call, research.md D5). Returns one entry per built-in kind (`_schema/nodes/<kind>.md`) and built-in predicate (`_schema/predicates/<predicate>.md`), rendered via `core.RenderNode` from `kernel.CoreMergeRules`/`kernel.CorePredicates` (research.md D7). Called once by `cmd/arc/ctrl/init.go`, replacing the retired `appconfig.Default`.
- `Resolve(store)` — walks `_schema/nodes/` and `_schema/predicates/`, `core.ParseNode`-ing each file. A file that fails to parse (malformed front-matter, missing `id`/`kind`) is **skipped, not an error** — that kind/predicate is simply absent from the returned set, exactly the "malformed schema doc → treated as unrecognized" behavior spec.md's Edge Cases require. An absent `_schema/` folder entirely (e.g., invoked directly against pre-feature graph state — out of scope, spec.md Assumptions) resolves to two empty results, not an error, mirroring `config.Resolve`'s existing "absent file is not an error" precedent. Called by `cmd/arc/graph/apply.go` and `cmd/arc/lint/lint.go`, replacing the retired `appconfig.Resolve`.
- `RegisterKind`/`RegisterPredicate` — idempotent create-if-absent. `created=false` and no write when a file already exists at that path (spec FR-011 — never overwrite). When creating a kind's document, `merge` is always `union` (spec FR-010, clarified — never any other value, since a patch cannot specify one). Both funcs use `core.RenderNode` for content, `store.Create` for the write — the same two calls every other `fsys.Store` writer in this codebase already uses.

No `port`/`adapter` subpackage: `schema` has no use-case-private external dependency (its only I/O is the already-shared `fsys.Store`, consumed directly — the same precedent `internal/app/ctrl/service.Init` already sets by taking `fsys.Mounter` as a plain parameter with no private port wrapping it).

## Secondary port consumed by `internal/app/graph` (`internal/app/graph/port`, graph-private)

```go
type SchemaRegistry interface {
    RegisterKind(store fsys.Store, kind core.Kind) (created bool, err error)
    RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
}
```

Satisfied structurally by `internal/app/schema`'s concrete component (ADR 001 port isolation rule 1 — the same technique already used for `internal/adapter/git.Git` satisfying three separate `port.VCS` interfaces). `graph.Apply`'s signature grows two parameters: `schema port.SchemaRegistry` and `predicates map[string]bool` (research.md D2/D3):

```go
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, predicates map[string]bool, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error)
```

## `internal/app/lint` signature change

```go
func Lint(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, predicates map[string]bool, dir string) (kernel.LintResult, error)
```

`predicates` replaces the deleted `parsePredicateRegistry(store)` call inside `Lint` (research.md D6); `rules` is unchanged in type and meaning, only its origin changes (schema, not config).

## On-disk shape (`_schema/`)

```text
_schema/
├── nodes/
│   ├── source.md         # merge: none
│   ├── entity.md         # merge: union
│   ├── resource.md       # merge: union-first-writer
│   ├── timeline.md       # merge: append
│   └── <discovered>.md   # merge: union (always, when auto-registered)
└── predicates/
    ├── mentions.md
    ├── mentionedIn.md
    ├── cites.md
    ├── isCitedBy.md
    ├── broader.md
    ├── narrower.md
    ├── isPartOf.md
    ├── hasPart.md
    ├── requires.md
    ├── replaces.md
    ├── isReplacedBy.md
    ├── conformsTo.md
    ├── related.md
    └── <discovered>.md
```

Every file: `kind: schema` front-matter, `id` equal to its own basename (guaranteed by `core.RenderNode`'s existing fallback); node-kind files additionally carry `merge`.

## Seeding at `arc init` (touches `internal/app/ctrl`, research.md D9)

`cmd/arc/ctrl/init.go` calls `appschema.Seed()` (no `ctx`, no fetcher — pure), producing `schemaSeed map[string][]byte`, passed as `appctrl.Init(ctx, mounter, vcs, dir, schemaSeed)`'s renamed final parameter (was `configSeed []byte`). `internal/app/ctrl/service.Init` merges `schemaSeed` into a per-call copy of `kernel.DefaultLayout.SeedFiles` (renamed from `MetaStubs`) before `writeLayout` — mechanically identical to how the retired `configSeed []byte` was merged in at one path; now it is many paths. `kernel.DefaultLayout.Folders` includes `_schema/nodes` and `_schema/predicates` in place of `_meta` (research.md D9); neither needs a `.gitkeep` since `schemaSeed` guarantees both are non-empty at write time.

## Auto-discovery at `arc apply` time (research.md D3/D4)

Inside `graph.Apply`'s existing per-node loop, immediately after the existing `op, ok := rules.Lookup(node.Kind)`:

```go
if !ok {
    op = core.MergeUnion
    result.Warnings = append(result.Warnings, ...) // unchanged, existing behavior
    if _, err := schema.RegisterKind(store, node.Kind); err != nil {
        // treated the same as any other write failure on this path: rollback + return
    }
}
```

and, after `merged` is computed, for every distinct predicate name in `merged.Links`' keys and `merged.Edges`' non-empty `Predicate` fields not already present in `predicates`:

```go
if _, err := schema.RegisterPredicate(store, name); err != nil {
    // same failure handling
}
```

Both writes land in `store` before the existing `vcs.StageAll`/`vcs.Commit` call, landing in the same commit as the rest of the patch's changes (spec FR-012) with no change to the commit-boundary logic itself.

## `arc lint` exemption and namespace separation (research.md D6)

`internal/app/lint/service/lint.go`'s `walkNodeFiles` skips the entire `_schema` directory (one new `if full == "_schema" { continue }` check, alongside the existing `.arc` skip) — schema documents are never parsed as content nodes, never enter the basename-uniqueness index, and are never checked by any per-kind content rule (spec FR-015, Clarifications Q1/Q3).
