# `--json` Contract: `arc stats`

**Feature**: 032-arc-stats | **Version**: v1 (new schema — additive, not breaking)

Per Principle XIV, `--json` output is a **stable, scriptable contract**.
Breaking a field name or its semantics requires a major version bump. This
document is the v1 baseline.

## Shape — default (`arc stats --json`)

```json
{
  "root": "/path/to/graph",
  "nodes": 1284,
  "edges": 5310,
  "byType": [
    { "type": "Entity", "count": 902 },
    { "type": "Source", "count": 311 },
    { "type": "Timeline", "count": 48 },
    { "type": "Property", "count": 19 },
    { "type": "Class", "count": 4 }
  ],
  "brokenLinks": 7,
  "ingestion": [
    { "period": "2024", "count": 96 },
    { "period": "2025", "count": 158 },
    { "period": "2026", "count": 57 }
  ],
  "unreadable": [],
  "foreign": ["README.md"]
}
```

**`detail` is absent**, not `null` (FR-020a). A consumer distinguishes
"not requested" from "requested and empty" by key presence alone.

## Shape — verbose (`arc stats --json --verbose`)

Identical, plus:

```json
{
  "detail": {
    "byPredicate": [ { "predicate": "cites", "count": 2140, "declared": true } ],
    "ingestionMonthly": [ { "period": "2026-01", "count": 12 } ],
    "orphans": 3,
    "stubs": 21,
    "unresolvedTargets": [ { "target": "missing-node", "refs": 4 } ],
    "avgOutDegree": 4.13,
    "medianOutDegree": 3,
    "topReferenced": [ { "id": "some-node", "refs": 88 } ],
    "schema": {
      "classes": 4, "classesUsed": 4,
      "properties": 19, "propertiesUsed": 15,
      "undeclaredPredicates": ["mentionsInPassing"]
    },
    "content": {
      "inlineRefs": 611,
      "attributeValues": 4022,
      "nodesWithoutPublished": 69
    }
  }
}
```

`byType[].declared` is present only in verbose output, for the same reason
`detail` is: it is a verbose-only figure (data-model.md).

## Contract invariants

A conforming implementation MUST satisfy all of the following. Each is a
required test.

| ID | Invariant | Source |
|---|---|---|
| **C1** | `Σ byType[].count == nodes` | FR-004 |
| **C2** | `Σ detail.byPredicate[].count == edges` | FR-012 |
| **C3** | For each year *y* in `ingestion`, the `detail.ingestionMonthly` entries whose period begins with *y* sum to *y*'s count | FR-013 |
| **C4** | `Σ detail.unresolvedTargets[].refs == brokenLinks` | FR-014 |
| **C5** | `brokenLinks` equals the number of `linkResolves` violations `arc lint` reports for the same graph | FR-006, **SC-003** |
| **C6** | `detail` key is absent when `--verbose` is not passed | FR-018, FR-020a |
| **C7** | Two runs over an unchanged graph emit byte-identical output | FR-021, SC-005 |
| **C8** | An empty graph emits every key with zero/empty values and exits 0 | FR-010 |
| **C9** | No key is derived from a node file's path; moving a node between folders changes no value | FR-004a |

**C5 is the load-bearing one.** It is enforced by a test in `cmd/arc/` that runs
both `Lint` and `Stats` over one fixture and compares the two numbers, so the
assertion fails when *either* command drifts ([research D6](../research.md)).

## Ordering

Array ordering is part of the contract (C7 depends on it):

- `byType`, `detail.byPredicate` — count descending, then name ascending
- `ingestion`, `detail.ingestionMonthly` — period ascending
- `detail.topReferenced` — refs descending, then id ascending, max 10 entries
- all remaining arrays — ascending

## Compatibility notes

- Adding a new key to `detail` or a new array element field is **additive**
  (minor bump).
- Renaming a key, changing a count's meaning, or moving a field between the
  root and `detail` is **breaking** (major bump, preceded by a deprecation
  warning on stderr per Principle XIV).
- `nodes` counts the census population (schema documents included) while
  `edges` and `brokenLinks` are computed over the content population
  ([research D1](../research.md)). This asymmetry is deliberate and is part of
  the contract — a consumer MUST NOT assume one population.
