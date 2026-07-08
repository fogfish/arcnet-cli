# Research: Per-Predicate Merge Reconciliation for arc apply

## D1: `Merge`'s new signature needs no layering violation

**Decision**: `core.Merge` drops its `op MergeOp` parameter and instead takes the graph's schema index directly: `Merge(existing, incoming Node, index Index, sourceID string) (Node, []string, error)`.

**Rationale**: The user's own brief worried this might require a resolver interface to avoid `internal/core` depending on `internal/app/schema` (ADR 001 §Domain: `internal/core` "allowed dependencies on themselves or on open-source modules only"). That worry doesn't apply here: `Index`, `PredicateDef`, and `TypeDef` are already declared *inside* `internal/core` (`internal/core/rules.go`), not in `internal/app/schema`. `internal/app/schema/service.Resolve` is simply the use-case that *builds* a `core.Index` value from disk; the type itself is core-domain. `internal/app/graph/service.Apply` already receives a `core.Index` as a parameter (`apply.go:152`) and already reads `index.Types[node.Type]` — passing that same value one level deeper into `core.Merge` instead of pre-extracting one `MergeOp` is a strictly smaller change than introducing a new interface, and keeps `Merge` pure (no I/O, no port dependency).

**Alternatives considered**:
- A `func(predicate string) MergeOp` resolver closure — rejected: adds an abstraction with no caller that would ever supply a different implementation than "look up `core.Index.Predicates`"; passing the concrete `Index` value is simpler and just as testable (unit tests build a literal `core.Index{Predicates: ...}`).
- A `port.SchemaLookup` interface in `internal/core` — rejected for the same reason: `Index` is already a plain, comparable-by-construction data value, not a capability that needs swapping per adapter.

## D2: Widening the `MergeOp` vocabulary from 5 to 7 named values

**Decision**: Replace `internal/core/ast.go`'s five constants with the seven CORE §9.3 names, as literal string values (so `_schema/predicates/*.md`'s `merge:` front matter reads naturally):

```go
const (
    MergeImmutable          MergeOp = "immutable"
    MergeUnion              MergeOp = "union"
    MergeFirstWriteWin      MergeOp = "firstWriteWin"
    MergeFillIfEmpty        MergeOp = "fillIfEmpty"
    MergeLastWriteWin       MergeOp = "lastWriteWin"
    MergeAppend             MergeOp = "append"
    MergeValidatedOverwrite MergeOp = "validatedOverwrite"
)
```

**Rationale**: `internal/app/schema/kernel/schema.go` already assigns each of CORE's seven concepts to every predicate today (`mergeImmutable`, `mergeFirstWriteWin`, `mergeFillIfEmpty`, `mergeLastWriteWin`, `mergeUnion`, `mergeAppend` — `schema.go:34-39`), but those names are aliases collapsed onto the *old* 5-value enum: `mergeFirstWriteWin` and `mergeFillIfEmpty` both resolve to `core.MergeUnionFirstWriter` (indistinguishable at runtime), and `mergeLastWriteWin` resolves to `core.MergeValidatedOverwrite` (a name that also means something functionally different — see D5). Genuinely distinguishing all seven, as this feature's own purpose requires (spec FR-002), is not optional plumbing; today's enum cannot express the target design. This is groundwork the spec's Assumptions section already commits to. `MergeNone` and the old `"union-first-writer"`/`"none"`/`"validated-overwrite"` spellings are retired outright — nothing in the new per-predicate model needs a "whole node is frozen" op (see D4).

**Migration**: `kernel/schema.go`'s six local aliases (lines 33-39) are deleted; `CorePredicateDefs`/`CoreTypeDefs` reference the new constants directly. `internal/app/schema/service/schema.go`'s `validMergeOps` map is updated to the new seven values (governs both `_schema/predicates/*.md` and `_schema/types/*.md` validation — the latter's `merge` field stays required per spec's Assumptions, just no longer consulted by dispatch, see D4). Three test files reference retired constants as inert fixture values only (`internal/core/ast_test.go`, `internal/app/lint/service/lint_test.go`, `internal/app/lint/service/rules_frontmatter_test.go`) and need a mechanical rename (`MergeNone`→`MergeImmutable`, `MergeUnionFirstWriter`→`MergeFirstWriteWin`) — grep confirms lint's own logic never reads `.Merge`, only carries it as fixture data for `Index`, so this is a pure rename with zero behavior risk.

## D3: `Node.Published` stays a typed field, routed through the same generic dispatch

**Decision**: Keep `Node.Published time.Time` as a dedicated field (not flattened into `Attrs["published"]`) — `core.ParseNode`/`RenderNode` already special-case it (`markdown.go:extractPublished`) for good reason: AST §7 documents this as a sanctioned typed convenience accessor, and every timeline-derivation call site (`applyTimeline`, `core.TimelinePeriods`) depends on it being a `time.Time`, not a stringly-typed Attrs entry. But retire the bespoke `mergePublished` helper's hardcoded "fill once, freeze forever" logic in favor of routing it through the same generic scalar-dispatch primitive as every other predicate, parameterized by `index.Predicates["published"].Merge` (which resolves to `immutable` under the corrected seed data — reproducing today's exact behavior, now for the right, uniform reason instead of a hardcoded special case).

**Rationale**: Directly satisfies the review brief's ask ("deriving its merge purely from the schema-declared rule so the logic isn't duplicated"). Go 1.26 (this repo's `go.mod` version) has full generics support, so the scalar primitive is written once as `mergeScalar[T comparable](existing, incoming T, zero T, op MergeOp) (merged T, diverges bool)` and instantiated for both `time.Time` (Published) and `string` (every Attrs/Texts scalar, which already compare via `fmt.Sprint` today). This is a genuine simplification, not added complexity: one generic function replaces `mergeScalarString` and `mergeScalarPredicate` and now also `mergePublished`.

**Alternative considered**: Leave `mergePublished` exactly as-is, hardcoded, and only generify the rest. Rejected because it would leave `published` as the one predicate whose behavior isn't actually driven by the schema index, contradicting FR-013 ("arc MUST determine each present predicate's merge behavior by looking it up in the... index") for that one field, and perpetuating exactly the duplicated-logic smell the brief flagged.

## D4: Whole-node `TypeDef.Merge` dispatch is deleted from `apply.go`, not generalized

**Decision**: `internal/app/graph/service/apply.go:220-231`'s `op := typeDef.Merge` / `op = core.MergeUnion` fallback and its single `core.Merge(existing, node, op, patch.Document)` call are replaced by `core.Merge(existing, node, index, patch.Document)` — no per-node op is computed at all. The `typeDef, ok := index.Types[node.Type]` lookup is kept (still needed for the "unrecognized kind" warning and `RegisterType` call), but `.Merge` is never read from it again.

**Rationale**: This is precisely what the feature is for. `TypeDef.Merge` remains a required, validated field on `_schema/types/*.md` (spec's Assumptions: removing it from the schema-document *shape* is out of scope, spec 011 territory) — it simply becomes inert data from `arc apply`'s point of view, exactly as spec 011's own Assumptions section already predicted ("expected to be retired once a future feature moves arc apply to genuine per-predicate merge dispatch").

## D5: A truth table replaces the `(fillEmpty, flagConflicts, unionText)` triple

**Decision**: Collapse the seven named ops into three underlying scalar-dispatch behaviors, since several are runtime-identical by the spec's own definition:

| Behavior class | Ops | Scalar rule |
|---|---|---|
| **freeze** | `immutable`, `validatedOverwrite` | Existing value, once non-empty, is permanent; empty existing accepts first incoming; never flags. (Identical today until a future "designated validation pass" feature gives `validatedOverwrite` its own overwrite path — out of scope here, spec Assumptions.) |
| **flagOnDiverge** | `firstWriteWin`, `fillIfEmpty` | Same accept-first/freeze rule as *freeze*, except a later, genuinely diverging non-empty incoming value is wrapped in the existing conflict marker instead of silently dropped. (Spec FR-006: fillIfEmpty is explicitly defined as behaviorally identical to firstWriteWin once set — one code path serves both, matching CORE's own definition rather than inventing a distinction the spec doesn't ask for.) |
| **alwaysOverwrite** | `lastWriteWin` | Incoming, whenever non-empty, always replaces existing; never flags. Order-sensitive by design (spec FR-007) — this is arc's own last-applied-wins rule, not a timestamp comparison (see D5a). |

List-shaped predicates (`union`, `append`, and every Edge/HRef regardless of declared op — links are structurally always a list) go through the existing `unionPredicates`/`unionLinks` dedup-by-key merge unconditionally; `union` and `append` are operationally identical for a list (both "combine, dedup, preserve existing-then-new order") — CORE's own definitions distinguish them only for scalars ("first writer wins" has no meaning for a set) and for prose (D5b), not for reference lists.

Which underlying dispatch a key uses is now chosen by **the key's own declared `MergeOp`**, not by how many values happen to be present in this particular merge (see D5c for why that's a deliberate, documented behavior change).

### D5a: `lastWriteWin` is application-order-sensitive, not timestamp-sensitive

**Decision**: `lastWriteWin` resolves by which contribution `arc apply` processed most recently — plain "incoming always wins when non-empty" — not by comparing a timestamp declared inside each patch.

**Rationale**: Genuine timestamp-based, order-*independent* resolution (the spec's first draft of FR-007) requires knowing, at merge time, when the value currently in `existing` was written. Each `arc apply` invocation is a separate OS process that only ever sees the *previous* merge's plain output — a bare YAML scalar with no timestamp attached — so that provenance would have to be persisted somewhere in the node file itself, specifically for `lastWriteWin`-governed predicates, to survive between separate invocations. That is a visible on-disk shape change (e.g. `status: {value: read, asOf: 2026-07-01}` instead of a bare scalar), which is a materially bigger change than "wire already-declared per-predicate rules into reconciliation" and edges into spec 010/011's AST-shape territory the feature explicitly excludes. Asked directly, the user chose this smaller, no-AST-change interpretation: `lastWriteWin` adopts the same convention git itself already applies to any tracked file (the most recent commit wins), which is intuitive for a tool whose whole persistence model is "git is the authoritative history" (CORE §4.9/§13.2). `spec.md` FR-007/FR-010/SC-003/User Story 3/Acceptance Scenario 3 and one Edge Case were revised accordingly (see `checklists/requirements.md` note dated 2026-07-08) — `lastWriteWin` is now the sole documented, deliberate exception to the otherwise-universal order-commutativity guarantee (FR-010).

### D5b: `append` on a Texts (prose) key reuses the existing paragraph-dedup logic verbatim

**Decision**: `mergeText`/`mergeParagraphs`/`shingles`/`jaccardSimilarity` (`merge.go:163-261`) are kept exactly as-is, just re-gated: today they run whenever the *whole node's* op is `MergeUnion` and the key isn't literally `"notes"` (`unionText && k != "notes"`); after this feature, they run whenever *that specific key's* declared op is `append`. Every seeded Texts-role predicate that should paragraph-append (`text`, generic prose) already declares `merge: append`; every one that should scalar-freeze-or-flag (`abstract`, `definition`, `notes`, `relevance`, `description`) already declares `merge: firstWriteWin` — so the old hardcoded `k != "notes"` carve-out disappears entirely. `"notes"` needs no special case anymore: it naturally goes through the scalar `flagOnDiverge` path because *its own* predicate declares `firstWriteWin`, not because the code singles out its name. This is a genuine simplification the redesign produces for free.

### D5c: List-vs-scalar dispatch is chosen by declared op, not by observed arity — a documented, intentional behavior change

**Decision**: Today, `mergeAttrs` (`merge.go:314-359`) decides list-union vs. scalar-compare purely by counting: "a key present on both sides that is multi-valued on either side" gets list-union, "a key that is single-valued on both sides" gets scalar dispatch — *regardless* of the whole node's chosen op. After this feature, a key declared `union` or `append` **always** list-unions, even when both sides happen to carry exactly one value right now (e.g. a resource with a single author on both sides, contributing a second, different author) — today that case would fall into the scalar branch and, depending on the node's whole-node op, either get silently dropped or conflict-flagged, which is arguably already a latent bug given `authors` is declared `merge: union`. After this feature it correctly becomes a two-author list. This is called out here explicitly, per the review brief's instruction, as an intentional, documented outcome of driving dispatch by the predicate's own declared shape/op rather than incidental cardinality — not a silent regression.

### D5d: Link/edge-role predicates never use the five scalar ops in practice

Every seeded link/edge-role predicate declares `union` or `append` (`entries` is the one `append` case; everything else is `union`) — CORE reserves "first writer wins"/"latest write wins" for scalars, not reference lists. If a graph-registered custom predicate ever declared a link-role predicate with a scalar-natured op, dispatch falls back to `union`'s list-merge (a link has no single "slot" to freeze or overwrite) — a defensive fallback, not a scenario this feature adds dedicated tests for, since no seeded predicate exercises it.

## D6: A predicate touched during merge with no schema document falls back to `union`, mirroring the existing type-fallback precedent

**Decision**: Today, `distinctPredicates` (`apply.go:65-80`) only scans `Edges` for auto-registration — an Attrs or Texts key's name is never auto-discovered (spec 011's own Assumptions already document this: "today's auto-discovery path only ever observes edge-position predicates"). That scope is **not** widened by this feature (it is schema-registration/discovery behavior, spec 011 territory, explicitly out of scope). Instead, purely for merge dispatch, a predicate absent from `index.Predicates` (which in practice only happens for a hand-authored custom Attrs/Texts predicate on a graph-registered custom type, since every built-in predicate is seeded by `arc init`) is merged as `union`, mirroring the exact fallback-with-warning pattern `apply.go` already uses for an unrecognized node type (`op = core.MergeUnion` + a `result.Warnings` entry) — reused verbatim rather than inventing a second fallback convention.

**Rationale**: A hard failure mid-merge for one obscure, never-conflicting predicate would be a worse outcome than a safe, already-precedented default; this keeps the blast radius of "missing schema doc" consistent across both the type-level and predicate-level fallback paths.

## D7: `internal/core/rules.go`'s `Index`/`PredicateDef`/`TypeDef` shapes are unchanged

**Decision**: No field is added to `PredicateDef`, `TypeDef`, or `Index`. This feature only changes what `internal/core/merge.go` and `internal/app/graph/service/apply.go` *do* with the `Merge MergeOp` field already on `PredicateDef` — confirming the spec's own scope boundary ("this feature only consumes [spec 011's] output").

## Summary of blast radius

| File | Change |
|---|---|
| `internal/core/ast.go` | `MergeOp` constants: 5 → 7, renamed/retired (D2) |
| `internal/core/merge.go` | `Merge` signature (D1); per-key dispatch loop replaces whole-node `mergeCore` (D5); `mergePublished` folded into generic scalar dispatch (D3) |
| `internal/core/merge_test.go` | Rewritten table-driven per op × shape (Attrs/Texts/Edges), plus explicit idempotency/commutativity cases |
| `internal/core/ast_test.go` | Rename retired constants in the roundtrip test |
| `internal/app/schema/kernel/schema.go` | Delete the 6 local `mergeXxx` aliases; reference new constants directly (D2) |
| `internal/app/schema/service/schema.go` | `validMergeOps` updated to the 7 new values (D2) |
| `internal/app/graph/service/apply.go` | Delete whole-node `op` computation; pass `index` straight through (D4) |
| `internal/app/graph/service/apply_test.go` | Update any test asserting the old whole-node outcome (e.g. `TestApplyNoneKindMergeAddsNoUpdated`) to the new per-predicate outcome; add cases per spec's new acceptance scenarios |
| `cmd/arc/graph/apply_test.go` | Add/adjust E2E cases mapping 1:1 to spec.md's acceptance scenarios (Principle VIII) |
| `internal/app/lint/service/lint_test.go`, `rules_frontmatter_test.go` | Mechanical constant rename (fixture data only, D2) |
| `ARCHITECTURE.md` | Glossary entries for "Merge Behavior", "Predicate Schema Node", "Source Node"/"Entity/Resource Node" updated to describe per-predicate dispatch and the 7-value vocabulary (Principle I/II) |

No `cmd/` package is added; no new Cobra flag, subcommand, or `--json` schema changes — this is purely a domain/service-layer behavior correction.
