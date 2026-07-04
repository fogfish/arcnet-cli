# Research: Graph Schema as a First-Class Citizen

## D1: Package boundary — what moves into `internal/app/schema`, what stays in `internal/core`

**Decision**: `internal/core` keeps the graph-format-wide, use-case-independent primitives every use-case already depends on and that have nothing to do with which specific kinds/predicates a graph recognizes: the `Kind`/`MergeOp` *type* definitions and the five `MergeOp` constants (`ast.go:20-31`), the `Node`/`Patch`/`Link`/`LinkBlock` AST types, the `Merge` algebra (`merge.go`), the Markdown codec (`ParseNode`/`RenderNode`/`ParsePatch`, `markdown.go`), and timeline derivation (`timeline.go`). These stay because `graph.Apply` calls `core.Merge`/`core.RenderNode`/`core.ParseNode` directly and always will, regardless of where schema data lives.

`internal/core/rules.go` in its entirety — `ConfigPath`, `MergeRuleSet.MarshalYAML`/`UnmarshalYAML`/`Union`, `CoreMergeRules`, `KnownProfileMergeRules` — is **deleted from `internal/core`** and its content redistributed:
- `CoreMergeRules`' four entries (`source→none, entity→union, resource→union-first-writer, timeline→append`, ARCNET-CORE §9.1-9.4) move into `internal/app/schema/kernel` as the seed data only `arc init` needs.
- `KnownProfileMergeRules` (the `hypothesis`/`aporia`/`thought` example table) is **dropped**, not moved: it was a copy-paste reference for hand-editing `.arc/config.yml`, which no longer exists for this purpose now that a graph's actual registered kinds are individually visible, readable, and editable as `_schema/nodes/*.md` files (User Story 3) — the reference table's job is now done by the files themselves (YAGNI, constitution Principle V).
- `MergeRuleSet`'s type definition (`map[Kind]MergeOp`) stays in `internal/core` (it is the plain, shared value type `graph.Apply`/`lint.Lint` already accept as a parameter — moving it would force those packages to import `internal/app/schema`, see D2). Its YAML marshal/unmarshal methods, needed only by `.arc/config.yml`'s old on-disk shape, are deleted; `Union`/`Lookup` are kept (`Lookup` is still how `graph.Apply`/`lint` test "is this kind recognized").
- `ConfigPath` (`.arc/config.yml`) is relocated to `internal/app/config/kernel` (config's own, now-sole concern) since no other package needs to know that path once `arc init` no longer seeds it (D5).

**Rationale**: The user's instruction — "Isolate ALL ARCNET-CORE abstractions, definitions, const and invariants within `schema` domain" — is satisfied for everything that is genuinely *ARCNET-CORE's declared vocabulary of defaults* (which kinds exist, which merge op each gets, which predicates are canonical). It is **not** extended to the AST/merge-algebra machinery, because ADR 001 ("Application Services") states use-cases "MUST NOT" reference "any fine-grained code unit from another use-case," and `internal/core` is explicitly the project's tier-1, non-use-case-owned evolution stage for exactly this kind of cross-cutting primitive (`adrs/001-system-architecture.md` "Domain" subsection). Moving `Node`/`Merge`/the codec into a `schema` *use-case* package would force `graph`/`lint` to import a sibling use-case directly — an actual architecture violation the constitution requires be raised, not silently made (Principle I: "Resolving cross-cutting design tension by ignoring an ADR is FORBIDDEN"). Splitting the CORE-spec-defaults data (schema's business) from the AST/algebra (core's business) resolves the tension without violating either instruction.

**Alternatives considered**: Moving `Node`/`Merge`/codec wholesale into `internal/app/schema` and having `graph`/`lint` import it — rejected, direct ADR 001 violation. Leaving `CoreMergeRules` in `internal/core` and only adding new code in `schema` — rejected, contradicts the explicit "isolate ALL... within schema domain" instruction and leaves ARCNET-CORE-specific defaults smeared across two packages, the exact problem the instruction is naming.

## D2: How `graph`/`lint`/`ctrl` consume schema data without importing `internal/app/schema`

**Decision**: `internal/app/schema`'s primary port (`component.go`) exposes plain, already-shared types only:
- `Seed() map[string]string` — pure; relative path → rendered Markdown for every core kind + core predicate. Consumed once, by `cmd/arc/ctrl/init.go`, replacing `appconfig.Default`.
- `Resolve(store fsys.Store) (core.MergeRuleSet, map[string]bool, error)` — reads `_schema/nodes/*.md` and `_schema/predicates/*.md` back into a `core.MergeRuleSet` (unchanged type) and a plain predicate-name set. Consumed by `cmd/arc/graph/apply.go` and `cmd/arc/lint/lint.go`, replacing `appconfig.Resolve`. **No signature change to `graph.Apply` or `lint.Lint`'s existing `rules core.MergeRuleSet` parameter** — only the cmd/-layer call site changes which package resolves it, exactly mirroring how `config.Resolve`'s result already crosses into `graph`/`lint` today as a plain value, never as an imported `config` type (confirmed: `internal/app/graph/service/apply.go` and `internal/app/lint/service/lint.go` both take `rules core.MergeRuleSet` as a parameter and never import `internal/app/config`).
- `RegisterKind(store fsys.Store, kind core.Kind) (created bool, err error)` / `RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)` — the two write operations `arc apply`'s discovery loop needs mid-transaction (D3).

`lint.Lint`'s signature gains one new plain parameter, `predicates map[string]bool` (replacing its internal `parsePredicateRegistry` call, D6) — still no import of `internal/app/schema` from `internal/app/lint`.

**Rationale**: This is the exact pattern already established for `configSeed []byte` (`ctrl.Init`) and `rules core.MergeRuleSet` (`graph.Apply`/`lint.Lint`) — a plain value resolved by one package and handed to another by `cmd/`, which is the one layer ADR 001 permits to know about multiple use-cases ("Distribution as Code": "All use-cases... are wired together... composed once"). No new named domain type crosses a use-case boundary; `map[string]bool`/`core.MergeRuleSet` are structurally plain.

**Alternatives considered**: A shared `PredicateSet` named type in `internal/core` alongside `MergeRuleSet` — rejected as unnecessary: a bare `map[string]bool` carries no less information and avoids growing `internal/core`'s public surface for a type with exactly one use.

## D3: Wiring `arc apply`'s schema auto-discovery into its existing single-commit transaction

**Decision**: `internal/app/graph/port` (graph's own, private port package) gains a second interface:

```go
type SchemaRegistry interface {
    RegisterKind(store fsys.Store, kind core.Kind) (created bool, err error)
    RegisterPredicate(store fsys.Store, predicate string) (created bool, err error)
}
```

satisfied structurally by `internal/app/schema`'s concrete component (no explicit `implements` needed — the same structural-satisfaction technique already used for `internal/adapter/git.Git` satisfying `ctrl.port.VCS`/`graph.port.VCS`/`lint.port.VCS`, three separate interfaces, one concrete type). `graph.Apply` gains a `schema port.SchemaRegistry` parameter. Inside the existing per-node loop (`internal/app/graph/service/apply.go:126-194`), right after the existing `op, ok := rules.Lookup(node.Kind)` (line 144): when `!ok`, additionally call `schema.RegisterKind(store, node.Kind)` (in addition to the existing union-default-plus-warning behavior, unchanged). After computing `merged` (which carries `.Links`/`.Edges`), collect every distinct predicate name from `merged.Links`'s keys and every non-empty `Link.Predicate` in `merged.Edges`, and for each not present in the `predicates` set resolved at the top of `Apply` (a new parameter, mirroring `rules`), call `schema.RegisterPredicate(store, name)`. Both writes land in `store` before `vcs.StageAll`/`vcs.Commit` (`apply.go:208-219`) run, so they are part of the same commit by construction — no new transaction boundary needed.

**Rationale**: This is precisely ADR 001 port isolation rule 1's prescribed remedy: "If use-case B needs a behaviour that use-case A's infrastructure provides, use-case B defines its own narrow port interface... and the wiring layer (`cmd/`) connects the two." `graph`'s existing single-commit design (`rollback`/`createdPaths` already track partial-failure cleanup) needs no restructuring — schema-file writes are ordinary `fsys.Store` writes indistinguishable from node-file writes as far as the existing rollback/commit flow is concerned.

**Alternatives considered**: Performing schema registration in `cmd/arc/graph/apply.go` *after* `graph.Apply` returns, then committing separately — rejected, violates FR-012 (schema docs MUST be in the *same* commit as the triggering patch) and would require a second commit or restructuring `graph.Apply` to stage-but-not-commit, a much larger change for no benefit.

## D4: Predicate discovery source (why `Links`/`Edges`, not a separate document scan)

**Decision**: A node's distinct predicates are exactly the union of `core.Node.Links`' map keys (predicate-grouped blocks, `linkBlockKey`) and every non-empty `Link.Predicate` in `core.Node.Edges` (bare list items can still carry an inline `predicate::` prefix per `parseListItemLink`'s regex). `HRefs` are excluded — those are inline citation-style predicates already governed by `internal/app/lint/service/rules_predicates.go`'s separate `citoPredicates` citation-type vocabulary (CORE §8), a different, pre-existing axis this feature does not touch (D8).

**Rationale**: `core.Node` already carries exactly this information post-parse; no new parsing capability is needed in `internal/core`.

## D5: Dropping the config-seed network fetch entirely (no HTTP fetcher in `schema`)

**Decision**: `internal/app/schema` has **no** `port`/`adapter` subpackage and performs **no** network access. `Seed()` is a pure function returning built-in Go constants only (ARCNET-CORE §9's four kinds, §7.4's thirteen predicates — D7). The `https://raw.githubusercontent.com/fogfish/arcnet-spec/.../ARCNET-CORE.md` URL in the spec is the human-readable specification these built-in constants must stay faithful to at each `arcnet-spec` release, not a runtime fetch target — unlike the retired `config.Default`'s target, `.../config.yml`, which was a small, already-machine-readable file `config.Default` fetched and parsed directly. `ARCNET-CORE.md` is prose; parsing it at runtime into structured merge rules would be a substantially larger, more fragile undertaking than this feature asks for, and nothing in spec.md requests it.

**Rationale**: The plan input is explicit — "remove the github downloader, it is not relevant anymore" — read most simply as retiring config's network-fetch responsibility outright, not relocating it. This satisfies spec.md FR-007 ("when the core specification cannot be retrieved... seed from built-in defaults") **trivially and by construction**: there is no fetch attempt, so the fallback path is the only path, and `arc init` is unconditionally offline-safe (strengthens, never weakens, the existing guarantee from `specs/002-arc-init` FR-017).

**Alternatives considered**: Re-implementing an HTTP fetch of a new, still-to-be-defined machine-readable schema-seed source in `arcnet-spec` — rejected as speculative (YAGNI, Principle V): no such machine-readable source exists yet at the referenced URL, and inventing one is a cross-project change outside this feature's control or scope.

## D6: `internal/app/lint` changes

**Decision**: Three changes, all additive/subtractive, no new lint rule added:
1. `walkNodeFiles` (`internal/app/lint/service/lint.go:190-225`): add `if full == "_schema" { continue }` alongside the existing `if full == ".arc" { continue }` (line 204) — `_schema/` (and everything under it) is never walked as content. This single change satisfies both spec FR-015 (schema docs exempt from ordinary content rules — they are simply never visited by any `check*` function) and the basename-uniqueness namespace separation (spec Clarifications Q3 — never entering `basenameIndex` at all). `excludedMetaFiles` (lines 36-39) and its two entries are deleted.
2. `parsePredicateRegistry` (`rules_predicates.go:36-60`) and `predicatesPath` (line 28) are deleted; `Lint`'s signature gains a `predicates map[string]bool` parameter (D2), used directly at the current `registry` call site (`lint.go:124`).
3. `checkPredicateRegistered`'s violation message (`rules_predicates.go:144`, `"...not registered in %s", occ.predicate, predicatesPath`) is reworded to name `_schema/predicates/` instead of the retired `_meta/predicates.md` path.

`citoPredicates` (citation-TYPE vocabulary, CORE §8) is untouched — confirmed to be a distinct concern from the structural/semantic predicate vocabulary (CORE §7.4) this feature seeds (D8).

## D7: Concrete seed content (ARCNET-CORE.md §9 / §7.4, fetched and confirmed)

Fixed node kinds (`_schema/nodes/`, `arc init`-seeded, §9.1-9.4):

| id | merge |
|---|---|
| `source` | `none` |
| `entity` | `union` |
| `resource` | `union-first-writer` |
| `timeline` | `append` |

Core predicate vocabulary (`_schema/predicates/`, `arc init`-seeded, §7.4 — structural + semantic, 13 total): `mentions`, `mentionedIn`, `cites`, `isCitedBy`, `broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`.

**Rationale/scoping note**: CORE §8's separate CITO citation-*type* list (`citesAsEvidence`, `citesAsAuthority`, `supports`, `confirms`, `extends`, `critiques`, `disputes`, `refutes`, plus `cites`/`isCitedBy` already counted above) qualifies a citation *edge*, not a registrable predicate name in the §7.3/§7.4 sense — it is already implemented separately as lint's own `citoPredicates` (D6) and is explicitly out of scope for `_schema/predicates/` seeding.

## D8: `internal/app/config` — what "keep the infrastructure alive" means concretely

**Decision**: Delete `port/fetcher.go`, `adapter/http/`, `adapter/mock/`, `service.Default`, `service.Resolve`, and `component.go`'s `Default`/`Resolve` exports (the merge-rule-specific, network-fetching half). Keep `kernel.Config` (its `MergeRules core.MergeRuleSet` field removed, leaving an intentionally empty struct), `service.Load`/`Save`, and `component.go`'s corresponding exports, unchanged in behavior. `ConfigPath` moves into `internal/app/config/kernel` (D1). After this feature, `.arc/config.yml` is written and read by nothing in `cmd/` — it becomes dormant infrastructure, kept only because the user explicitly asked for it to remain available for an unspecified future configuration need, not because any current call site needs it.

**Rationale**: Directly follows the plan input ("keep the config infrastructure alive, just remove the github downloader"). Flagged explicitly here (and in plan.md's Complexity Tracking) as a deliberate exception to YAGNI, made on direct instruction rather than discovered independently — the kind of thing Principle V would otherwise flag as dead code.

## D9: `internal/app/ctrl` layout changes

**Decision**: `kernel.ArcNetCoreLayout.Folders` (`internal/app/ctrl/kernel/graph.go:32-39`) drops `_meta`, adds `_schema/nodes` and `_schema/predicates`. `MetaStubs map[string]string` is renamed `SeedFiles map[string]string` (same shape, same `writeLayout`/`hasStub`/`rollback` mechanics, zero behavioral change to those functions — `hasStub`'s prefix-match already generalizes to the two new folders being non-empty once seeded). `DefaultLayout.SeedFiles` becomes empty (no more static `_meta` stub content); all seed content now arrives at runtime as `schemaSeed map[string]string`, produced by `appschema.Seed()` in `cmd/arc/ctrl/init.go` and merged into `layout.SeedFiles` exactly where `configSeed []byte` was previously merged in at `core.ConfigPath` (`internal/app/ctrl/service/init.go:76-81`). `Init`'s signature parameter is renamed `configSeed []byte` → `schemaSeed map[string]string`. `rollback` (`init.go:189-204`)'s cleanup loop iterates the same merged `layout.SeedFiles` instead of the two now-deleted statics plus one hardcoded `core.ConfigPath` removal.

**Rationale**: `_schema/nodes/` and `_schema/predicates/` are never actually empty once seeded (17 files between them), so no `.gitkeep` placeholder is needed for either — consistent with `hasStub`'s existing logic, not a special case.

## D10: `ARCHITECTURE.md` and Glossary obligations (Principle I)

**Decision**: Add `internal/app/schema` (fifth `internal/app/<domain>` use-case) to the Directory Structure section, following the `kernel/service/component.go` shape (no `port`/`adapter` subdirectory — D5/D2 note why). Update the Glossary: rewrite **Metadata Stub** and **Kind Registration** entries (both currently describe `_meta/`/`.arc/config.yml`, both retired) into a new **Schema Document** / **Node-Kind Schema Document** / **Predicate Schema Document** entry set describing `_schema/`; update **Canonical Folder**'s folder list (drop `_meta`, add `_schema/nodes`, `_schema/predicates`); note in **Extension Profile Checklist**'s entry that its previously-flagged gap ("no mechanism for a profile to declare schema exists yet") is unaffected by this feature — this feature adds kind/merge *recognition* storage, not field-level schema declaration, so `arc lint`'s FR-011 scope note in `specs/004-arc-lint/plan.md` Complexity Tracking remains accurate and undisturbed.

**Rationale**: Constitution Principle I ("ARCHITECTURE.md MUST be updated in the same PR", "domain concepts... MUST be added to the Glossary").
