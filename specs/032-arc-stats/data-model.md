# Phase 1 Data Model: Graph Statistics (`arc stats`)

**Feature**: 032-arc-stats | **Spec**: [spec.md](./spec.md) | **Research**: [research.md](./research.md)

All types live in `internal/app/graph/kernel/stats.go`. They are plain domain
values: no cobra, no fsys, no behavior beyond deriving themselves.

---

## Populations

Every figure below is derived from one of two populations produced by a single
walk ([research D1](./research.md)):

| Population | Contents | Feeds |
|---|---|---|
| **Census** | Every file `core.LooksLikeNode` accepts, including `_schema/` | `Nodes`, `ByType`, `Unreadable`, `Foreign` |
| **Content** | Census minus nodes whose `@type` is `Class` or `Property` | `Edges`, `BrokenLinks`, and every `StatsDetail` figure except `Schema` |

Foreign files (markdown the graph does not own) belong to neither and are
counted separately ([research D8](./research.md)).

---

## `StatsResult` — always present

| Field | JSON | Type | Derivation | Requirement |
|---|---|---|---|---|
| `Root` | `root` | `string` | The graph root scanned | — |
| `Nodes` | `nodes` | `int` | `len(census)` | FR-002 |
| `Edges` | `edges` | `int` | Σ `len(n.Edges)` over content population — **occurrences**, so a repeated target counts twice | FR-003 |
| `ByType` | `byType` | `[]TypeCount` | Census grouped by `Node.Type`; Σ counts == `Nodes` | FR-004, FR-004a |
| `BrokenLinks` | `brokenLinks` | `int` | Σ over content population of distinct unresolved targets per node, across `Edges`+`HRefs` | FR-006 |
| `Ingestion` | `ingestion` | `[]PeriodCount` | Yearly timeline period nodes, chronological | FR-007, FR-008 |
| `Unreadable` | `unreadable` | `[]string` | Paths that failed to open or parse | FR-009 |
| `Foreign` | `foreign` | `[]string` | Markdown files `LooksLikeNode` rejected | FR-009, [D8](./research.md) |
| `Detail` | `detail,omitempty` | `*StatsDetail` | `nil` unless verbose requested | FR-018, FR-020a |

**Invariant A**: `Σ ByType[].Count == Nodes` (FR-004). Unit-tested directly.

**Invariant B**: `Detail == nil` ⟺ verbose not requested. A nil pointer with
`omitempty` marshals to an absent key, which is exactly FR-020a's
"absent, not null".

---

## `StatsDetail` — verbose only

| Field | JSON | Type | Derivation | Requirement |
|---|---|---|---|---|
| `ByPredicate` | `byPredicate` | `[]PredicateCount` | Content-population edges grouped by `Link.Predicate`; Σ == `Edges` | FR-012 |
| `IngestionMonthly` | `ingestionMonthly` | `[]PeriodCount` | Monthly timeline period nodes, chronological | FR-013 |
| `Orphans` | `orphans` | `int` | Content nodes with no outgoing `Edges`/`HRefs` **and** no incoming reference | FR-014 |
| `Stubs` | `stubs` | `int` | Content nodes with empty `Attrs`, `Texts`, `HRefs`, `Edges` — the existing `isStub` shape | FR-014 |
| `UnresolvedTargets` | `unresolvedTargets` | `[]TargetCount` | Distinct unresolved target + number of referencing nodes; Σ `Refs` == `BrokenLinks` | FR-014 |
| `AvgOutDegree` | `avgOutDegree` | `float64` | `Edges / len(content)`, zero-degree nodes included in the denominator | FR-015 |
| `MedianOutDegree` | `medianOutDegree` | `float64` | Median of per-node `len(Edges)`; even size → mean of the two central values | FR-015 |
| `TopReferenced` | `topReferenced` | `[]RefRank` | Top 10 by incoming `Edges`+`HRefs`, ties by ascending `ID` | FR-015, FR-019 |
| `Schema` | `schema` | `SchemaCoverage` | Class/Property declaration and usage | FR-016 |
| `Content` | `content` | `ContentVolume` | Inline refs, attribute values, nodes lacking `Published` | FR-017 |

**Invariant C**: `Σ ByPredicate[].Count == Edges` (FR-012).

**Invariant D**: for each year *y* in `Ingestion`,
`Σ {m.Count : m ∈ IngestionMonthly, m.Period starts with y}` `==` *y*`.Count`
(FR-013).

**Invariant E**: `Σ UnresolvedTargets[].Refs == BrokenLinks` (FR-014). This is
why `TargetCount` carries a reference count rather than being a bare list —
`BrokenLinks` counts a target once per referencing node, so a plain list would
be shorter than the number it explains.

---

## Entry types

```text
TypeCount       { Type string; Count int; Declared bool }
PredicateCount  { Predicate string; Count int; Declared bool }
PeriodCount     { Period string; Count int }
TargetCount     { Target string; Refs int }
RefRank         { ID string; Refs int }
SchemaCoverage  { Classes, ClassesUsed, Properties, PropertiesUsed int
                  UndeclaredPredicates []string }
ContentVolume   { InlineRefs, AttributeValues, NodesWithoutPublished int }
```

`TypeCount.Declared` answers "does a `Class` schema node declare this type?" —
distinct from `SchemaCoverage.Classes`, which counts the schema nodes
themselves. Populated only when verbose; `false` and unrendered otherwise.

---

## Ordering rules (FR-005, FR-019, FR-021, SC-005)

Every slice is sorted **in the service**, never in the printer, so `--json` is
deterministic too:

| Slice | Order |
|---|---|
| `ByType`, `ByPredicate` | Count descending, then name ascending |
| `Ingestion`, `IngestionMonthly` | Period ascending (lexicographic == chronological for `2006` / `2006-01`) |
| `TopReferenced` | Refs descending, then `ID` ascending, truncated to 10 |
| `UnresolvedTargets`, `UndeclaredPredicates`, `Unreadable`, `Foreign` | Ascending |

Go randomizes map iteration order per run, so an unsorted map-derived slice
would violate SC-005 intermittently — passing locally, flaking in CI.

---

## Period identity

A timeline node is a period node when its `@type` is `Timeline`. Its period is
its own `@id`: `2006` → yearly, `2006-01` → monthly. Anything else is counted
as a Timeline node in `ByType` but contributes to no period figure. Paths are
never consulted ([research D3, D5](./research.md)).

---

## Glossary additions

Staged here for `ARCHITECTURE.md` when it is created (Principle II; see
plan.md Complexity Tracking):

- **Type breakdown** — node counts grouped by the `@type` each node declares.
  Distinct from *Class*, the schema node registering a type's vocabulary.
- **Stub node** — a node carrying no content beyond its identity and type;
  the shape `service.isStub` already recognizes.
- **Ingestion period node** — a `Timeline` node whose `@id` is a year (`2006`)
  or month (`2006-01`) code, holding one entry per source ingested in it.
- **Census population / content population** — the two node universes a
  whole-graph statistic is computed over ([research D1](./research.md)).
