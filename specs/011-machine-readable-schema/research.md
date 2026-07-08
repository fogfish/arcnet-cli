# Research: Machine-Readable Predicate & Type Schema

## D1: `Index`/`PredicateDef`/`TypeDef` live in `internal/core`, not `internal/app/schema/kernel`

**Decision**: Add `Index`, `PredicateDef`, and `TypeDef` to `internal/core` (extending `rules.go`, alongside the `MergeOp`/`MergeRuleSet` types they retire), not to `internal/app/schema/kernel` as a first draft might suggest.

**Rationale**: `internal/app/graph` and `internal/app/lint` both need to consume the value `Resolve` returns. Today they do this via `core.MergeRuleSet`/`map[string]bool` — both either a shared `internal/core` type or a plain built-in type, never a type from `internal/app/schema/kernel`. This is not an accident: ADR 001's port-isolation rule 1 ("a use-case's `port/` package is private to that use-case; other use-cases MUST NOT import it directly") and the project's own domain-tiering model (`internal/core` for the cross-cutting shared domain, `internal/app/<use-case>/kernel` for that use-case's private value types) both push a cross-use-case value type down into `internal/core`. Putting `Index` in `schema/kernel` would force `internal/app/graph` and `internal/app/lint` to import another use-case's kernel package directly — a new inter-use-case coupling this codebase has specifically avoided until now (confirmed by grep: `MergeRuleSet` today is used only inside `internal/core`, `internal/app/schema`, `internal/app/graph`, and `internal/app/lint`, each already depending on `internal/core`, none on each other's kernels).

**Alternatives considered**:
- `Index` in `internal/app/schema/kernel`, imported by `graph`/`lint` directly — rejected: violates the existing "use-cases have no reference to another use-case's fine-grained types" discipline (ADR 001 Application Services section), and would be the first such cross-import in the codebase.
- `Index` in `internal/app/schema/kernel`, with `graph`/`lint` depending on a narrow interface instead of the concrete type — rejected as needless indirection: `Index` is a plain, immutable-after-construction data value (two maps), not a capability with multiple implementations: an interface adds ceremony with no Dependency-Inversion benefit here, since there is exactly one producer (`schema.Resolve`) and the constitution's own ISP guidance is about behavior, not passive data.

## D2: `Resolve`'s fail-fast contract distinguishes "not a graph" from "schema missing/invalid"

**Decision**: `Resolve(store fsys.Store) (core.Index, error)` first checks for `.arc/` (the same graph-root marker `guardIsGraph` already checks in `internal/app/graph/service` and `internal/app/lint/service`), returning the existing "not an initialized graph, run `arc init`" error family if absent — distinct from a new `ErrSchemaMissing`/`ErrSchemaInvalid` family used when `.arc/` is present but `_schema/predicates/`/`_schema/types/` is absent or contains a malformed document.

**Rationale**: Today, `cmd/arc/graph/apply.go` and `cmd/arc/lint/lint.go` both call `appschema.Resolve(store)` *before* calling into `appgraph.Apply`/`applint.Lint`, which is where the existing `guardIsGraph` check actually lives. Under spec FR-014's new fail-fast rule, if `Resolve` treated a merely-absent `_schema/` folder as an immediate hard error without first checking `.arc/`, running any schema-aware command inside a plain, uninitialized directory would surface a confusing "schema missing" error instead of the correct, existing "not a graph, run `arc init` first" guidance — a real UX regression the constitution's Principle XII (rewrite errors into actionable guidance) would flag. Checking `.arc/` first inside `Resolve` itself (rather than relying on caller ordering) keeps the correct error surfaced regardless of call order, and costs one extra `Stat` call.

**Alternatives considered**:
- Reorder `cmd/` wiring so `Apply`'s/`Lint`'s own `guardIsGraph` runs before `Resolve` is ever called — rejected: `Resolve` would still need its own defensive check to be robust against any future caller that doesn't reorder correctly, so the check has to live in `Resolve` regardless; reordering the callers on top of that would be duplicate, non-load-bearing work.
- Let a missing `_schema/` folder and a missing `.arc/` folder both produce the same generic error — rejected: conflates two genuinely different, differently-actionable conditions ("this isn't a graph at all" vs. "this graph's schema is broken"), which Principle XII's "rewrite errors into human-readable guidance" explicitly disfavors.

## D3: Property/Class nodes map onto the existing `core.Node` shape with no `core.Node` changes

**Decision**: A predicate schema node (`_schema/predicates/<name>.md`) is an ordinary `core.Node{ID: name, Type: "Property", Attrs: {"role": [...], "merge": [...], "label"?: [...], "aligned"?: [...]}, Texts: {"description": ...}}`. A type schema node (`_schema/types/<name>.md`) is `core.Node{ID: name, Type: "Class", Attrs: {"merge": [...]} (FR-015 bridge), Texts: {"description": ...}, Edges: [{Predicate: "required", Target: <predicateName>}, ..., {Predicate: "optional", Target: <predicateName>}, ...]}`.

**Rationale**: `internal/core.Node`'s post-spec-010 shape (`Attrs map[string][]Predicate`, `Texts map[string]string`, `Edges []Link`) already represents everything a `Property`/`Class` node needs with zero changes to `internal/core`'s AST types — exactly the "no new AST shape" property ARCNET-AST §8 itself calls out ("CORE v0.5 makes predicates and types ordinary graph nodes ... so they need no dedicated AST shape"). Reusing `core.ParseNode`/`RenderNode` verbatim (as spec 005 already established for the existence-only shape) means this feature adds zero parser/renderer code, only richer `Attrs`/`Texts`/`Edges` construction and decoding inside `internal/app/schema/service`.

**Alternatives considered**: A dedicated `PropertyNode`/`ClassNode` Go struct distinct from `core.Node` — rejected: duplicates the AST's own explicit "no dedicated shape" guidance and would need its own parser/renderer pair, contradicting Principle V (YAGNI) for no added capability.

## D4: Rendering gap — flat bulleted list instead of "## Requires"/"## Optional" headings

**Decision**: Accept, and explicitly document, that `internal/core.RenderNode` still renders every `Edges` entry as one flat bulleted list (`- required:: [[title]]`, `- optional:: [[url]]`) rather than grouping `required`/`optional` under their own `## Requires`/`## Optional` headings, per spec 010's own already-accepted Complexity Tracking deferral of heading-grouped, role-driven rendering to a later feature.

**Rationale**: `internal/core.ParseNode`'s `walkNodeBody` already recognizes an inline `predicate:: [[Target]]` bullet regardless of whether it sits under a heading, a bold label, or in the plain, ungrouped bare-list position — so a flat list round-trips a type node's `required`/`optional` predicates with zero data loss. Implementing heading-grouped rendering narrowly, only for `_schema/types/` documents, ahead of the general role-driven rendering feature already scoped for later, would be a second, throwaway heuristic — exactly the outcome spec 010's own Complexity Tracking entry warned against when it deferred this same capability.

**Alternatives considered**: Implement a narrow, `_schema/types/`-only heading-grouping special case in `RenderNode` now — rejected per the reasoning above; also rejected because CORE's own `role: link` (heading-grouped) vs. `role: edge` (flat) distinction is exactly the general mechanism the deferred feature will introduce, and `required`/`optional` are themselves ordinary predicates with their own `role: link` declarations (CORE §10.8) — special-casing them here would need un-teaching once the general feature lands.

## D5: Auto-registration defaults for a newly discovered predicate/type

**Decision**: `RegisterPredicate` assigns `role: edge`, `merge: union` to a newly discovered predicate. `RegisterType` assigns `merge: union` (the FR-015 bridge field) and empty `Required`/`Optional` lists to a newly discovered type.

**Rationale**: `internal/app/graph/service.Apply`'s `distinctPredicates` helper only ever observes a predicate from `Node.Edges` (never from `Attrs`/`Texts`/`HRefs`) when deciding to auto-register it — so `role: edge` is not a guess, it is the one structural position auto-discovery can actually observe today. `merge: union` mirrors the already-established safe-default precedent for auto-discovered node kinds (spec 005 FR-010, "always the safe default"). Empty `Required`/`Optional` for a newly discovered type is the maximally permissive choice: the discovery context provides no signal about what a conforming instance of that type must or may carry, so declaring nothing is the only choice that does not risk asserting something false about the type.

**Alternatives considered**: Inferring a type's `Required`/`Optional` from the one instance that triggered its discovery (its actual `Attrs`/`Edges` keys) — rejected: one instance's *actual* predicates are not necessarily every predicate a *conforming* instance of that type must or may carry (that is exactly the distinction CORE §9.2 draws), so this would risk asserting an over-fitted, possibly-wrong constraint a human did not review.

## D6: Complete CORE vocabulary seeded by `arc init`

**Decision**: `Seed()` renders one `_schema/predicates/<name>.md` for every predicate CORE §10 documents — identity (`@id`, `@type`), content (`tags`, `text`), metadata/control (`published`, `created`, `updated`), structural (`mentions`, `mentionedIn`), semantic (`broader`, `narrower`, `isPartOf`, `hasPart`, `requires`, `replaces`, `isReplacedBy`, `conformsTo`, `related`), citation (`cites`, `citesAsEvidence`, `citesAsAuthority`, `supports`, `confirms`, `extends`, `critiques`, `disputes`, `refutes`, `isCitedBy`), type-specific (`title`, `abstract`, `authors`, `url`, `doi`, `category`, `aliases`, `definition`, `notes`, `ref`, `year`, `status`, `relevance`, `granularity`, `entries`, `heading`), and the schema mechanism's own (`role`, `merge`, `label`, `aligned`, `description`, `required`, `optional`) — plus one `_schema/types/<name>.md` for each of `source`/`entity`/`resource`/`timeline` (CORE §11) and for `Property`/`Class` themselves (since a schema node's own `"@type"` value is itself a type in use, per spec FR-007).

**Rationale**: Spec FR-007 defines "complete" this broadly, and CORE §10.8 explicitly settles the self-reference question: "Predicates used by exactly `Property`/`Class` nodes ... the schema mechanism's own vocabulary, registered like any other predicate rather than left as unregistered structure" — i.e., CORE itself already decided `role`/`merge`/`label`/`aligned`/`description`/`required`/`optional` and `Property`/`Class` get registered, so no separate judgment call is needed here.

**Alternatives considered**: Seeding only the narrower ~13-predicate/4-kind subset the tool hardcodes today — rejected: contradicts spec FR-007's explicit "not only the narrower subset" language and CORE §10.8's own self-registration instruction.

## D7: `RegisterKind` renamed `RegisterType`; `RegisterPredicate` keeps its name

**Decision**: Rename the `port.SchemaRegistry` interface method and its `internal/app/schema` implementation from `RegisterKind` to `RegisterType`, mechanically, wherever it appears (`internal/app/graph/port/schema.go`, `internal/app/schema/service/schema.go`, `internal/app/schema/component.go`, `internal/app/graph/service/apply.go`'s call site).

**Rationale**: Constitution Principle II (DDD & Glossary) requires ubiquitous language consistency — spec.md's own vocabulary is "Type Schema Node"/"`@type`", never "kind," and the folder itself is renamed `_schema/nodes/`→`_schema/types/`. Keeping a method literally named `RegisterKind` after this rename would be exactly the "flag named `--name` and a domain field named `Identifier` for the same concept" glossary violation Principle II calls out by name. `node.Type` (the Go field, already renamed in spec 010) is unaffected — only the schema-registration verb changes.

**Alternatives considered**: Leave `RegisterKind` named as-is since its behavior doesn't change, only its target folder — rejected per the Principle II rationale above; the rename is purely mechanical (a `grep`-and-replace, no logic change) so the cost is negligible against the glossary-consistency benefit.

## D8: `Apply`/`Lint` accept `core.Index` directly, replacing the `(rules, predicates)` tuple

**Decision**: `internal/app/graph/service.Apply` and `internal/app/lint/service.Lint` each replace their `rules core.MergeRuleSet, predicates map[string]bool` parameter pair with a single `index core.Index` parameter. Internal call sites read `index.Types[node.Type]` (presence + `.Merge`) where they previously called `rules.Lookup(node.Type)`, and `index.Predicates[name]` (presence) where they previously checked `predicates[name]`.

**Rationale**: This is the direct, mechanical consequence of D1 (the shared type lives in `internal/core`) and D6/spec FR-004 (the Index is the one runtime source of truth both consumers share) — collapsing two loosely-related parameters into the one cohesive value they always traveled together as, with no behavioral change to either consumer's own logic beyond the lookup syntax.

**Alternatives considered**: Keep `rules`/`predicates` as two separate parameters, both now sourced from `Index.Types`/`Index.Predicates` at the `cmd/` call site — rejected: this would still require `Apply`/`Lint`'s signatures to name two parameters that are always constructed and passed together from the same `Resolve` call, adding no flexibility, only two names where one would do.

## D9: `core.MergeRuleSet` is retired, not kept alongside `core.Index`

**Decision**: Delete `core.MergeRuleSet` and its `.Lookup`/`.Union` methods once every caller is migrated to `core.Index`; do not keep both types alive in parallel.

**Rationale**: A repo-wide grep confirms `MergeRuleSet` is used only inside `internal/core` itself and its three consumers being migrated in this same feature (`internal/app/schema`, `internal/app/graph`, `internal/app/lint`) — nothing external depends on it surviving. Keeping both types after `Index` subsumes `MergeRuleSet`'s entire purpose (`Index.Types[name].Merge` is `MergeRuleSet.Lookup(name)` plus `Required`/`Optional`) would be exactly the "maintaining two parallel representations of the same data indefinitely" pattern Principle V (YAGNI) already ruled out for a comparable choice in spec 010's Complexity Tracking.

**Alternatives considered**: Keep `MergeRuleSet` as a deprecated type alias or thin wrapper for compatibility — rejected: pre-1.0, zero external consumers, no compatibility obligation exists to preserve.
