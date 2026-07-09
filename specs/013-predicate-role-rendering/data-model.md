# Data Model: Schema-Driven Link Rendering

This feature changes behavior, not shape: no field is added to any existing type (`Node`, `Link`, `PredicateDef`,
`Index`), and no new predicate/schema attribute is introduced (spec.md Assumptions: "no new schema field is
introduced"). Documented here for traceability against spec.md's Key Entities, and to fix the exact algorithm
`renderNodeBody` implements.

## Inputs already in scope, unchanged shape

- **`Node.Edges []Link`** (`internal/core/ast.go`) — every outgoing structural link, in document order,
  regardless of source shape (spec 010, unchanged by this feature: parsing still produces exactly this one
  flat, ungrouped slice — research.md D1/D2).
- **`PredicateDef.Role string`** (`internal/core/rules.go`) — one of `meta`/`text`/`href`/`edge`/`link`; this
  feature reads exactly two of the five values (`edge`, `link`) to drive rendering. `meta`/`text`/`href`
  predicates never appear in `Edges` (they live in `Attrs`/`Texts`/`HRefs` respectively) and are unaffected.
- **`PredicateDef.Label string`** (`internal/core/rules.go`) — optional display heading for a `link`-role
  predicate; already documented as defaulting to the predicate name, capitalized.
- **`Index.Predicates map[string]PredicateDef`** (`internal/core/rules.go`) — the lookup this feature
  consults; a predicate absent from this map has no declared `Role` (research.md D3).

## New: the render-time partition (in-memory only, never persisted)

A pure function of `(n.Edges, index)`, recomputed on every render, never stored on `Node`:

| Group | Membership test | Rendered shape |
|---|---|---|
| **Flat group** | every `Link` in `n.Edges` whose `resolveRenderRole(index, l.Predicate) == "edge"` (includes any predicate absent from `index.Predicates`, research.md D3) | one bare bulleted list, original relative order, `renderLinkBullet` per line (unchanged format) |
| **Link groups** | every `Link` in `n.Edges` whose `resolveRenderRole(index, l.Predicate) == "link"`, bucketed by `l.Predicate` | one `"## " + label + "\n"` block per distinct predicate name present, `renderLinkBullet` per occurrence in original relative order within the group; blocks ordered by resolved label, ascending |
| **Single-group omission** (FR-006/FR-007) | Flat group is empty **and** exactly one distinct predicate name appears across all Link groups | that one group's heading is omitted; its occurrences render as a bare list instead (same shape as the Flat group's, landing in the same parser slot) |

`label` for a given predicate name resolves as: `index.Predicates[name].Label` if non-empty, else
`titleCaseType(name)` (existing helper — research.md D4).

## Physical layout invariant (unchanged from today, now populated by two kinds of content instead of one)

`renderNodeBody`'s existing, load-bearing physical ordering (leading text → edges → trailing text, documented
at `internal/core/markdown.go`'s existing `renderNodeBody` comment) is preserved exactly; only what fills the
"edges" slot changes:

```
[leading text, if any]

[flat group's bare bulleted list, if non-empty]

[## <Label 1>
 <link group 1's bulleted list>]

[## <Label 2>
 <link group 2's bulleted list>]
...

[trailing text, if any]
```

When the single-group omission applies, there is exactly one bare bulleted list in the "edges" position (no
flat group and no heading both being present simultaneously in that case — mutually exclusive by
construction, since the omission's own precondition is "flat group is empty").

## Call-site `Index` sourcing (research.md D6/D7 — summarized)

| Caller | `Index` value |
|---|---|
| `internal/app/schema/service/schema.go` `Seed()` | `core.Index{Predicates: kernel.CorePredicateDefs, Types: kernel.CoreTypeDefs}` (built-in, static) |
| `internal/app/schema/service/schema.go` `registerIfAbsent` | `core.Index{}` (safe: rendered node never carries `Edges`) |
| `internal/app/graph/service/apply.go` | `Apply`'s own existing `index core.Index` parameter (spec 012), threaded through |
| `cmd/arc/graph/subgraph.go`, `cmd/arc/graph/serve.go` | `resolveIndexOrDefault(store)` — real index if `.arc`/`_schema` resolves, else `core.Index{}` (research.md D7) |

## No new Key Entity

spec.md names two entities (**Predicate Schema**, **Node Body**) purely descriptively — both already exist as
`core.PredicateDef`/`core.Index` and the already-rendered Markdown body respectively. Neither gains a field.
