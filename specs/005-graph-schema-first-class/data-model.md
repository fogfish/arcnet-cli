# Data Model: Graph Schema as a First-Class Citizen

## Schema Document (on-disk shape)

A schema document is an ordinary `internal/core.Node`, parsed/rendered by the existing, unmodified `core.ParseNode`/`core.RenderNode` — no new AST or codec is introduced.

| Field (`core.Node`) | Node-Kind Schema Document | Predicate Schema Document |
|---|---|---|
| `ID` | the node kind's name, e.g. `"entity"` | the predicate's name, e.g. `"related"` |
| `Kind` | always `"schema"` | always `"schema"` |
| `Attrs["merge"]` | required — one of `none`/`union`/`union-first-writer`/`append`/`validated-overwrite` | absent |
| `Text` | optional short prose description (informational only, never parsed back structurally) | optional short prose description |
| `Notes`, `HRefs`, `Edges`, `Links` | always empty for a schema document | always empty |

On disk: `_schema/nodes/<ID>.md` / `_schema/predicates/<ID>.md`. `ID` equals the file's own basename by construction (`core.RenderNode`'s existing front-matter fallback already guarantees `id` round-trips even when not independently repeated in Attrs).

Example (`_schema/nodes/entity.md`):

```markdown
---
kind: schema
merge: union
---
# entity

A concept or subject mentioned across sources, mergeable across contributions.
```

Example (`_schema/predicates/related.md`):

```markdown
---
kind: schema
---
# related

A general association between two nodes with no more specific predicate (skos:related).
```

## Schema (runtime, resolved)

Not a new named type — `schema.Resolve` returns the two plain values every consumer already knows how to use:

- `core.MergeRuleSet` (`map[core.Kind]core.MergeOp`, unchanged type from `internal/core`) — one entry per file under `_schema/nodes/`.
- `map[string]bool` — the set of predicate names with a file under `_schema/predicates/`.

## Kind/Predicate Seed (built-in, `internal/app/schema/kernel`)

- `CoreMergeRules map[core.Kind]core.MergeOp` — the 4 fixed kinds (research.md D7).
- `CorePredicates []string` — the 13 fixed predicate names (research.md D7), each paired with a one-line description used only to render `Seed()`'s `Text` field.

## State / Lifecycle

A node kind or predicate has exactly two states, both persistent (never reverted automatically):

1. **Unknown** — no file exists under its `_schema/` subfolder. `graph.Apply` treats it as unrecognized (safe default merge / no registration check), same externally-visible behavior as before this feature for a first encounter.
2. **Registered** — a file exists. Created either by `arc init` (the 17 core kinds/predicates) or by `arc apply`'s auto-discovery (any other kind/predicate, the first time a patch contributes one). Once registered, a file is never deleted or overwritten automatically (spec FR-011) — only a human, editing the file directly (User Story 3), changes its content.

There is no "deprecated" or "removed" state in this feature's scope — pruning a no-longer-used kind/predicate's schema document is not addressed (not requested by spec.md, consistent with the schema growing monotonically as new content is discovered).

## Relationships

- A **Node-Kind Schema Document**'s `merge` value is the authoritative input to `internal/core.Merge`'s `op` parameter for every ordinary content node of that kind (`graph.Apply`'s existing `rules.Lookup(node.Kind)` call site, now fed by `schema.Resolve` instead of the retired `config.Resolve`).
- A **Predicate Schema Document**'s mere existence is what `arc lint`'s `checkPredicateRegistered` checks an ordinary content node's declared predicates against (replaces `_meta/predicates.md`'s bullet-list registry).
- Schema documents are **not** ordinary graph content nodes: they are excluded from `arc lint`'s basename-uniqueness index and every per-kind content rule (source-citation-back, entity Sowa category, etc.) — see research.md D6.
