# Data Model: Per-Predicate Merge Reconciliation for arc apply

This feature changes behavior, not shape: no field is added to any existing type. It's documented here for traceability against spec.md's Key Entities.

## MergeOp (`internal/core/ast.go`)

The fixed, seven-value menu a predicate's schema document declares itself against (spec FR-002). Widened from today's five values (research.md D2).

| Constant | String value | Dispatch class (research.md D5) |
|---|---|---|
| `MergeImmutable` | `"immutable"` | freeze |
| `MergeUnion` | `"union"` | list (unchanged name/value) |
| `MergeFirstWriteWin` | `"firstWriteWin"` | flagOnDiverge |
| `MergeFillIfEmpty` | `"fillIfEmpty"` | flagOnDiverge (identical code path to firstWriteWin, spec FR-006) |
| `MergeLastWriteWin` | `"lastWriteWin"` | alwaysOverwrite |
| `MergeAppend` | `"append"` | list (unchanged name/value) |
| `MergeValidatedOverwrite` | `"validatedOverwrite"` | freeze (identical code path to immutable, until a future validation-pass feature) |

Retired: `MergeNone`, `MergeUnionFirstWriter`, and the string spellings `"none"`, `"union-first-writer"`, `"validated-overwrite"` (renamed to `"validatedOverwrite"`).

## PredicateDef / TypeDef / Index (`internal/core/rules.go`)

**Unchanged shape.** `PredicateDef.Merge` and `TypeDef.Merge` remain `MergeOp`-typed fields; only the *set of legal values* changes (via `internal/app/schema/service.validMergeOps`), and only `PredicateDef.Merge` is read by dispatch after this feature — `TypeDef.Merge` becomes vestigial (still required/validated on `_schema/types/*.md`, per research.md D4, but never consulted by `core.Merge`).

## Node / Predicate / Link (`internal/core/ast.go`)

**Unchanged shape.** `Node.Published time.Time` stays a dedicated typed field (research.md D3), now merged via the same generic scalar-dispatch primitive as any other predicate, parameterized by `index.Predicates["published"].Merge` instead of a hardcoded rule. `Attrs`/`Texts`/`Edges`/`HRefs` are unchanged; what changes is which merge primitive processes each key.

## Per-shape × per-op reconciliation table

The authoritative behavior contract this feature implements (also the direct source for `internal/core/merge_test.go`'s table-driven cases):

| MergeOp | Attrs (front-matter scalar/list) | Texts (prose) | Edges / HRefs (links) |
|---|---|---|---|
| `immutable` | Existing (once non-empty) is permanent; empty existing accepts first incoming; never flagged | Same, string-valued | Not used by seeded vocabulary; falls back to `union`'s list-merge if ever declared (research.md D5d) |
| `union` | List: dedup-union of all `Predicate` values, existing-then-incoming order, never flagged | Not used by seeded vocabulary; falls back to `append`'s paragraph-merge if ever declared (research.md D5, D5d) | List: dedup-union by `(Predicate, Target)`, existing-then-incoming order, never flagged |
| `firstWriteWin` | Existing (once non-empty) persists; a later, genuinely diverging non-empty incoming value is wrapped in the conflict marker | Same, string-valued | Not used by seeded vocabulary |
| `fillIfEmpty` | Identical code path to `firstWriteWin` (spec FR-006) | Identical code path to `firstWriteWin` | Not used by seeded vocabulary |
| `lastWriteWin` | Incoming, whenever non-empty, always replaces existing; never flagged; order-sensitive by design (research.md D5a) | Same, string-valued | Not used by seeded vocabulary |
| `append` | List: dedup-union (operationally identical outcome to `union` for a list, research.md D5) | Paragraph-level dedup-append via existing `mergeText`/Jaccard-shingle logic (research.md D5b); never flagged | List: dedup-union by `(Predicate, Target)` (operationally identical to `union`) |
| `validatedOverwrite` | Identical code path to `immutable` until a future validation-pass feature (spec FR-009, Assumptions) | Identical code path to `immutable` | Not used by seeded vocabulary |

"Not used by seeded vocabulary" means no `CorePredicateDefs` entry declares that combination today — the fallback rule (research.md D5d) still defines what happens if a future custom predicate does.

## Corrected `CorePredicateDefs` seed values (`internal/app/schema/kernel/schema.go`)

Every predicate keeps its existing `Role`/`Label`/`Aligned`/`Description`; only `Merge` is repointed from the old collapsed aliases to the new distinct constants (spec FR-016). No predicate's *conceptual* assignment changes — `internal/app/schema/kernel/schema.go`'s existing per-predicate choices (`mergeImmutable`/`mergeFirstWriteWin`/`mergeFillIfEmpty`/`mergeLastWriteWin`/`mergeUnion`/`mergeAppend` at lines 50-105) already encode the *intended* seven-way assignment — this feature makes those intentions actually distinguishable at runtime for the first time. E.g.: `published`/`created`/`title`/`ref`/`granularity`/`role`/`merge` → `MergeImmutable`; `status`/`updated` → `MergeLastWriteWin`; ~~`abstract`/`category`/`definition`/`notes`/`relevance`/`heading`/`label`/`aligned`/`description` → `MergeFirstWriteWin`~~ `category`/`heading`/`label`/`aligned` (all `role: meta`) → `MergeFirstWriteWin`; `abstract`/`definition`/`notes`/`relevance`/`description` (all `role: text`) → `MergeAppend`, not `MergeFirstWriteWin` (**BUG-001**, spec FR-018: every `role: text` predicate is seeded `append`, not distinguished per-name); `url`/`doi`/`year` → `MergeFillIfEmpty`; `tags`/`authors`/`aliases`/`mentions`/`mentionedIn`/every `edge`-role predicate/`required`/`optional` → `MergeUnion`; `text`/`entries` → `MergeAppend`.

**Bugfix**: 2026-07-08 — BUG-001: `abstract`/`definition`/`notes`/`relevance`/`description` were reassigned from `MergeFirstWriteWin` to `MergeAppend`, since all five are `role: text` and FR-018 now requires every `text`-role predicate to default to `append`. This is an on-disk-behavior change for any graph already using this seed: a genuinely diverging re-contribution to one of these five predicates no longer produces a conflict marker — it appends as a new paragraph instead (mergeText's existing near-duplicate-paragraph dedup applies, same as the generic `text` predicate already gets).

## `CoreTypeDefs` seed values

Kept for continuity/documentation (still required, validated field per `_schema/types/*.md`'s shape, research.md D4) even though no longer consulted by dispatch: `source` → `MergeImmutable` (was `MergeNone`), `entity`/`Property`/`Class` → `MergeUnion` (unchanged), `resource` → `MergeFirstWriteWin` (was `MergeUnionFirstWriter`), `timeline` → `MergeAppend` (unchanged).
