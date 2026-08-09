# `--json` Output Contract: `kernel.BatchResult`

**Feature**: [../spec.md](../spec.md) | **Data model**: [../data-model.md](../data-model.md)

Satisfies FR-018. This is a **stable, scriptable contract** under constitution
Principle XIV — a breaking change to any field name, type, or enum value
requires a major version bump.

Emitted on stdout by `arc apply batch --json`, rendered by
`bios.Registry.Resolve(bios.ModeJSON)`'s generic `json.MarshalIndent` printer,
so the shape follows the Go struct tags exactly.

---

## Schema

```json
{
  "directory": "string",
  "patches": [
    {
      "path": "string",
      "document": "string",
      "published": "RFC3339 timestamp",
      "outcome": "applied | skipped | failed | unprocessed",
      "reason": "string (present only when outcome is failed)",
      "commit": "string (present only when outcome is applied)",
      "created": { "<NodeType>": 0 },
      "merged":  { "<NodeType>": 0 }
    }
  ],
  "applied": 0,
  "skipped": 0,
  "failed": 0,
  "unprocessed": 0,
  "not_a_patch": 0,
  "conflicts": ["string"],
  "warnings": ["string"]
}
```

### Field notes

| Field | Guarantee |
|---|---|
| `directory` | The patch directory, resolved to an absolute path (the examples below abbreviate it for readability). |
| `patches` | Always an array, never `null`. Entries are in **application order** — publication date ascending, ties broken by `path` (FR-004, FR-005). |
| `patches[].path` | Slash-separated and relative to `directory` on every platform, so output is comparable across machines. |
| `patches[].published` | RFC3339. A date-only manifest serialises at `T00:00:00Z`. |
| `patches[].outcome` | One of exactly four values; the set is closed and additions are a breaking change. |
| `patches[].reason` | Omitted unless `outcome` is `failed`. |
| `patches[].commit` | Omitted unless `outcome` is `applied`. |
| `patches[].created` / `merged` | Omitted unless `outcome` is `applied`. Keys are node type names as they appear in the graph schema. |
| `conflicts`, `warnings` | Always arrays, never `null`; empty when there are none. |

### Counter invariants

- `applied + skipped + failed + unprocessed == len(patches)`
- `not_a_patch` counts Markdown files that never entered `patches` (FR-003) and
  is deliberately **not** part of that sum
- `unprocessed > 0` is possible only under `--fail-fast` (FR-011)
- `failed > 0` ⟺ process exit code is `1` (FR-014)

---

## Example — mixed run, default (continue-on-error) mode

```json
{
  "directory": "./patches",
  "patches": [
    {
      "path": "2024/rescorla-tls13.patch.md",
      "document": "rescorla-2024-tls13",
      "published": "2024-03-11T00:00:00Z",
      "outcome": "applied",
      "commit": "a1b2c3d",
      "created": { "Source": 1, "Entity": 4 },
      "merged": {}
    },
    {
      "path": "2026/pqkex.patch.md",
      "document": "chen-2026-pqkex",
      "published": "2026-01-20T00:00:00Z",
      "outcome": "applied",
      "commit": "e4f5a6b",
      "created": { "Source": 1 },
      "merged": { "Entity": 2 }
    },
    {
      "path": "2026/karpathy.patch.md",
      "document": "karpathy-2026-notes",
      "published": "2026-04-02T00:00:00Z",
      "outcome": "skipped"
    },
    {
      "path": "2026/truncated.patch.md",
      "document": "",
      "published": "0001-01-01T00:00:00Z",
      "outcome": "failed",
      "reason": "patch manifest is invalid"
    }
  ],
  "applied": 2,
  "skipped": 1,
  "failed": 1,
  "unprocessed": 0,
  "not_a_patch": 1,
  "conflicts": ["entities/tls.md"],
  "warnings": ["Hypothesis is not a recognized node type for this graph — auto-registered with a default schema document"]
}
```

Exit code `1` (`failed` is non-zero).

## Example — nothing to apply (FR-022)

```json
{
  "directory": "./notes",
  "patches": [],
  "applied": 0,
  "skipped": 0,
  "failed": 0,
  "unprocessed": 0,
  "not_a_patch": 3,
  "conflicts": [],
  "warnings": []
}
```

Exit code `0`.

## Example — `--fail-fast` halt (FR-011)

A patch that classifies cleanly but fails during application sits at its real
position in the publication order, so the run halts mid-plan and everything
after it is `unprocessed`.

```json
{
  "directory": "./patches",
  "patches": [
    { "path": "a.patch.md", "document": "doc-a", "published": "2025-01-01T00:00:00Z", "outcome": "applied", "commit": "aaa1111", "created": { "Source": 1 }, "merged": {} },
    { "path": "b.patch.md", "document": "doc-b", "published": "2025-03-14T00:00:00Z", "outcome": "failed", "reason": "node id does not match its filename" },
    { "path": "c.patch.md", "document": "doc-c", "published": "2025-06-01T00:00:00Z", "outcome": "unprocessed" }
  ],
  "applied": 1,
  "skipped": 0,
  "failed": 1,
  "unprocessed": 1,
  "not_a_patch": 0,
  "conflicts": [],
  "warnings": []
}
```

Exit code `1`.

## Ordering of classification failures

A patch that fails to parse loses its manifest date along with everything
else, so it carries the zero timestamp `0001-01-01T00:00:00Z` and is ordered
**after** every successfully classified patch, among its peers by `path`
(research.md D5b). The mixed-run example above shows this: `truncated.patch.md`
is last despite its filename sorting earlier.
